package game

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/protocol"
	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/puzzle"
	"github.com/google/uuid"
)

// Puzzle-assembly timers. All of them pause during a host disconnect.
const (
	timerAssemblyDeadline = "assembly.deadline"
	timerAssemblyPreview  = "assembly.preview"
	timerAssemblyTick     = "assembly.tick"
	timerRecPrefix        = "rec."
)

// enterPuzzleAssembly starts the puzzle timer (host signal already
// validated).
func (e *Engine) enterPuzzleAssembly() {
	e.phase = protocol.PhasePuzzleAssembly
	e.puzzle.timerRunning = true
	e.puzzle.startTime = time.Now()
	e.puzzle.playersAtPuzzleStart = len(e.players)
	e.puzzle.grid = map[string]*Fragment{}
	e.puzzle.enteredGrid = map[string]bool{}
	e.puzzle.highlights = map[string][]protocol.Position{}
	e.puzzle.recommendations = map[string]*Recommendation{}
	e.puzzle.pendingBySender = map[string]string{}

	// totalTime = (baseTime + chronosBonus) × difficulty time multiplier;
	// the totalTime value is authoritative for all clients.
	raw := e.cfg.PuzzleBaseTime.Sec() + e.puzzle.effects.ChronosTimeBonus
	totalSeconds := raw * e.cfg.TimeMultiplier()
	e.puzzle.totalTime = time.Duration(totalSeconds * float64(time.Second))

	e.timers.Schedule(timerAssemblyDeadline, e.puzzle.totalTime, true)
	if e.puzzle.effects.ClarityPreviewDuration > 0 {
		previewWindow := time.Duration(e.puzzle.effects.ClarityPreviewDuration * float64(time.Second))
		e.timers.Schedule(timerAssemblyPreview, previewWindow, true)
	}
	e.timers.Schedule(timerAssemblyTick, e.cfg.GridUpdateInterval.Duration(), true)

	// Players still disconnected at timer start have their segments
	// auto-solved into unassigned fragments right now.
	for _, p := range e.players {
		if !p.Connected {
			e.autoSolve(p)
		}
	}
	e.revealUnassigned()

	start := protocol.PuzzlePhaseStart{
		StartTimestamp:         protocol.Timestamp(e.puzzle.startTime),
		TotalTime:              totalSeconds,
		BaseTime:               e.cfg.PuzzleBaseTime.Sec(),
		ChronosBonus:           e.puzzle.effects.ChronosTimeBonus,
		ClarityPreviewActive:   e.puzzle.effects.ClarityPreviewDuration > 0,
		ClarityPreviewDuration: e.puzzle.effects.ClarityPreviewDuration,
		PlayerPhases:           e.playerPhases(),
	}
	e.broadcastPlayers(protocol.PuzzleToClientPhaseStart, start)
	e.sendHost(protocol.PuzzleToHostPhaseStart, e.buildHostPhaseStart(totalSeconds, e.puzzle.startTime))
	e.touchGrid()

	// A random auto-solve landing everything correctly can win immediately.
	e.checkVictory()
}

// autoSolve converts a player's individual puzzle into an unassigned
// fragment at a random open cell (disconnect handling). Counts toward k.
func (e *Engine) autoSolve(p *Player) *Fragment {
	segment := e.puzzle.assignments[p.ID]
	if segment == "" || e.puzzle.grid[segment] != nil {
		return e.puzzle.grid[segment]
	}
	e.puzzle.enteredGrid[p.ID] = true
	return e.placeFragment(segment, "")
}

// playerPhases partitions players into 2A (still solving individually) and
// 2B (fragment on the central grid).
func (e *Engine) playerPhases() protocol.PlayerPhases {
	phases := protocol.PlayerPhases{Phase2A: []string{}, Phase2B: []string{}}
	for _, id := range e.playerIDsInJoinOrder() {
		if e.puzzle.enteredGrid[id] {
			phases.Phase2B = append(phases.Phase2B, id)
		} else {
			phases.Phase2A = append(phases.Phase2A, id)
		}
	}
	return phases
}

func (e *Engine) buildHostPhaseStart(totalSeconds float64, start time.Time) protocol.HostPuzzlePhaseStart {
	phases := e.playerPhases()
	return protocol.HostPuzzlePhaseStart{
		TimerActive:      e.puzzle.timerRunning && !e.puzzle.paused,
		StartTimestamp:   protocol.Timestamp(start),
		TotalTime:        totalSeconds,
		BaseTime:         e.cfg.PuzzleBaseTime.Sec(),
		ChronosBonus:     e.puzzle.effects.ChronosTimeBonus,
		PlayersInPhase2A: len(phases.Phase2A),
		PlayersInPhase2B: len(phases.Phase2B),
	}
}

// handleAssemblyTimer dispatches this phase's timers.
func (e *Engine) handleAssemblyTimer(name string) {
	switch {
	case name == timerAssemblyPreview:
		e.puzzle.previewExpired = true
		e.broadcastPlayers(protocol.PuzzleToClientPreviewExpired, struct{}{})
	case name == timerAssemblyDeadline:
		e.handlePuzzleTimeout()
	case name == timerAssemblyTick:
		e.broadcastGridTick()
		e.timers.Schedule(timerAssemblyTick, e.cfg.GridUpdateInterval.Duration(), true)
	case strings.HasPrefix(name, timerRecPrefix):
		e.expireRecommendation(strings.TrimPrefix(name, timerRecPrefix), "timeout")
	}
}

// broadcastGridTick sends the periodic player grid state plus each 2B
// player's private personal state.
func (e *Engine) broadcastGridTick() {
	state := e.buildGridState()
	for _, p := range e.players {
		if !p.Connected {
			continue
		}
		p.send(protocol.PuzzleToClientGridState, state)
		if e.puzzle.enteredGrid[p.ID] {
			p.send(protocol.PuzzleToPlayerPersonalState, e.personalState(p))
		}
	}
}

// assemblyActive reports whether puzzle actions may execute right now.
func (e *Engine) assemblyActive() bool {
	return e.phase == protocol.PhasePuzzleAssembly && e.puzzle.timerRunning && !e.puzzle.paused
}

// ── Phase 2A: segment completion ───────────────────────────────────────────

func (e *Engine) handleSegmentCompleted(p *Player, payload protocol.SegmentCompleted) {
	if !e.assemblyActive() {
		e.sendPlayerError(p, protocol.ErrorTypeGameState, protocol.ErrForbiddenPhase,
			"segment completions are not accepted now",
			"the puzzle is not running (wrong phase or paused by a host disconnect)")
		return
	}
	assigned := e.puzzle.assignments[p.ID]
	if payload.SegmentID != assigned {
		e.sendPlayerError(p, protocol.ErrorTypeValidation, protocol.ErrForbiddenNotOwner,
			"segment is not assigned to this player",
			fmt.Sprintf("%s is not your assigned segment", payload.SegmentID))
		return
	}

	// Idempotent re-ack: a retry after a network blip changes no state.
	if f, ok := e.puzzle.grid[assigned]; ok {
		p.send(protocol.PuzzleToPlayerSegmentAcknowledged, protocol.SegmentAcknowledged{
			SegmentID: assigned,
			Position:  posOf(f.Pos),
		})
		return
	}

	p.Stats.CompletedIndividual = true
	p.Stats.IndividualSolveTime = math.Max(payload.SolveTime, 0)
	e.puzzle.enteredGrid[p.ID] = true
	f := e.placeFragment(assigned, p.ID)
	e.drawHighlights(p)
	// Reveal before reporting so completionStats reflects the fragments this
	// completion brought onto the grid.
	e.revealUnassigned()

	p.send(protocol.PuzzleToPlayerSegmentAcknowledged, protocol.SegmentAcknowledged{
		SegmentID: assigned,
		Position:  posOf(f.Pos),
	})

	// The first personal-state snapshot arrives immediately so guide
	// highlights show without waiting for the next tick.
	p.send(protocol.PuzzleToPlayerPersonalState, e.personalState(p))

	phases := e.playerPhases()
	e.sendHost(protocol.PuzzleToHostSegmentCompleted, protocol.HostSegmentCompleted{
		PlayerID:       p.ID,
		PlayerName:     p.Name,
		SegmentID:      assigned,
		CompletionTime: p.Stats.IndividualSolveTime,
		Position:       posOf(f.Pos),
		PhaseTransition: protocol.PhaseTransition{
			PlayersInPhase2A: len(phases.Phase2A),
			PlayersInPhase2B: len(phases.Phase2B),
		},
		CompletionStats: protocol.CompletionStats{
			TotalCompleted:      e.completedCount(),
			TotalRequired:       e.puzzle.playersAtPuzzleStart,
			UnassignedFragments: e.unassignedVisibleCount(),
		},
	})

	e.touchGrid()
	e.checkVictory()
}

func (e *Engine) unassignedVisibleCount() int {
	n := 0
	for _, f := range e.puzzle.grid {
		if f.OwnerID == "" {
			n++
		}
	}
	return n
}

// ── Phase 2B: moves and swaps ──────────────────────────────────────────────

func (e *Engine) handleFragmentMove(p *Player, payload protocol.FragmentMove) {
	moveID := uuid.NewString()
	p.Stats.FragmentMoves++

	reject := func(reason string, info *protocol.CooldownInfo) {
		p.send(protocol.PuzzleToPlayerMoveResult, protocol.MoveResult{
			MoveID:       moveID,
			Status:       "rejected",
			SegmentID:    payload.SegmentID,
			Reason:       reason,
			CooldownInfo: info,
		})
	}

	if !e.assemblyActive() || !e.puzzle.enteredGrid[p.ID] {
		reject(protocol.MoveRejectPhaseInvalid, nil)
		return
	}

	f, exists := e.puzzle.grid[payload.SegmentID]
	if !exists {
		reject(protocol.MoveRejectTargetInvalid, nil)
		return
	}
	if !e.controls(p, f) {
		reject(protocol.MoveRejectNotOwner, nil)
		return
	}

	target := puzzle.Pos{X: payload.TargetPosition.X, Y: payload.TargetPosition.Y}
	g := e.puzzle.gridSize
	if target.X < 0 || target.X >= g || target.Y < 0 || target.Y >= g {
		reject(protocol.MoveRejectTargetInvalid, nil)
		return
	}

	occupant := e.fragmentAt(target)

	if payload.SwapWithSegmentID != nil {
		// Swap mode: the declared partner must actually occupy the target.
		if occupant == nil || occupant.SegmentID != *payload.SwapWithSegmentID || occupant == f {
			reject(protocol.MoveRejectTargetInvalid, nil)
			return
		}
		if !e.controls(p, occupant) {
			// Displacing another player's owned fragment needs their consent:
			// that is what recommendations are for.
			reject(protocol.MoveRejectNotOwner, nil)
			return
		}
		if e.fragmentOnCooldown(f) {
			reject(protocol.MoveRejectCooldown, e.cooldownInfo(f))
			return
		}
		if e.fragmentOnCooldown(occupant) {
			reject(protocol.MoveRejectCooldown, e.cooldownInfo(occupant))
			return
		}

		f.Pos, occupant.Pos = occupant.Pos, f.Pos
		e.restartCooldown(f)
		e.restartCooldown(occupant)
		p.Stats.SuccessfulMoves++

		swappedID := occupant.SegmentID
		swappedPos := posOf(occupant.Pos)
		newPos := posOf(f.Pos)
		p.send(protocol.PuzzleToPlayerMoveResult, protocol.MoveResult{
			MoveID:                    moveID,
			Status:                    "success",
			SegmentID:                 f.SegmentID,
			NewPosition:               &newPos,
			SwappedSegmentID:          &swappedID,
			SwappedSegmentNewPosition: &swappedPos,
			CooldownInfo:              e.cooldownInfo(f),
		})
	} else {
		// Move-to-empty mode.
		if occupant != nil {
			reject(protocol.MoveRejectTargetInvalid, nil)
			return
		}
		if e.fragmentOnCooldown(f) {
			reject(protocol.MoveRejectCooldown, e.cooldownInfo(f))
			return
		}

		f.Pos = target
		e.restartCooldown(f)
		p.Stats.SuccessfulMoves++

		newPos := posOf(f.Pos)
		p.send(protocol.PuzzleToPlayerMoveResult, protocol.MoveResult{
			MoveID:       moveID,
			Status:       "success",
			SegmentID:    f.SegmentID,
			NewPosition:  &newPos,
			CooldownInfo: e.cooldownInfo(f),
		})
	}

	e.touchGrid()
	e.checkVictory()
}

// controls implements the movement permission rule: a player controls their
// own fragment and any unassigned fragment.
func (e *Engine) controls(p *Player, f *Fragment) bool {
	return f.OwnerID == "" || f.OwnerID == p.ID
}

// ── Completion ─────────────────────────────────────────────────────────────

// checkVictory runs after every grid mutation (moves, swaps, accepted
// recommendations, and every fragment placement).
func (e *Engine) checkVictory() {
	if !e.puzzle.timerRunning || e.puzzle.finished || !e.victoryMet() {
		return
	}
	e.puzzle.finished = true
	e.puzzle.timerRunning = false

	remaining := e.puzzleTimeRemaining()
	total := e.puzzle.totalTime.Seconds()
	completion := round2(total - remaining.Seconds())
	e.stopAssemblyTimers()

	g2 := e.puzzle.gridSize * e.puzzle.gridSize
	e.broadcastAll(protocol.PuzzleToClientCompletedSuccess, protocol.CompletedSuccess{
		Success:        true,
		CompletionTime: completion,
		TotalTime:      total,
		TimeRemaining:  round2(remaining.Seconds()),
		FinalGridState: protocol.FinalGridState{
			AllFragmentsCorrect: true,
			TotalFragments:      g2,
			CorrectFragments:    g2,
		},
	})
	e.finishPuzzle(true, completion)
}

// handlePuzzleTimeout ends the game as a loss when the deadline fires.
func (e *Engine) handlePuzzleTimeout() {
	if e.puzzle.finished {
		return
	}
	e.puzzle.finished = true
	e.puzzle.timerRunning = false
	e.stopAssemblyTimers()

	g2 := e.puzzle.gridSize * e.puzzle.gridSize
	correct := e.correctFragments()
	total := e.puzzle.totalTime.Seconds()
	e.broadcastAll(protocol.PuzzleToClientCompletedTimeout, protocol.CompletedTimeout{
		Success:     false,
		Reason:      "time_expired",
		TotalTime:   total,
		TimeExpired: true,
		FinalStats: protocol.TimeoutFinalStats{
			FragmentsPlaced:      len(e.puzzle.grid),
			TotalFragments:       g2,
			CorrectlyPlaced:      correct,
			CompletionPercentage: round1(float64(correct) / float64(g2) * 100),
		},
	})
	e.finishPuzzle(false, total)
}

// stopAssemblyTimers cancels every assembly-phase timer, including pending
// recommendation timeouts.
func (e *Engine) stopAssemblyTimers() {
	e.timers.Cancel(timerAssemblyDeadline)
	e.timers.Cancel(timerAssemblyPreview)
	e.timers.Cancel(timerAssemblyTick)
	for id := range e.puzzle.recommendations {
		e.timers.Cancel(timerRecPrefix + id)
	}
}

// finishPuzzle emits the host completion analytics and enters analytics.
func (e *Engine) finishPuzzle(success bool, completionSeconds float64) {
	e.sendHost(protocol.PuzzleToHostCompletionAnalytics, e.buildCompletionAnalytics(success, completionSeconds))
	e.enterAnalytics(success, completionSeconds)
}

func (e *Engine) buildCompletionAnalytics(success bool, completionSeconds float64) protocol.CompletionAnalytics {
	contributions := map[string]protocol.PlayerContribution{}
	totalMoves, successfulMoves := 0, 0
	totalRecs, acceptedRecs := 0, 0
	var respSum float64
	respCount := 0
	completed := 0
	var solveSum, fastest, slowest float64

	for id, p := range e.players {
		frag := e.puzzle.grid[e.puzzle.assignments[id]]
		correct := false
		if frag != nil {
			if want, err := puzzle.CorrectPos(frag.SegmentID, e.puzzle.gridSize); err == nil {
				correct = frag.Pos == want
			}
		}
		contributions[id] = protocol.PlayerContribution{
			IndividualSolveTime:     p.Stats.IndividualSolveTime,
			FragmentMoves:           p.Stats.FragmentMoves,
			SuccessfulMoves:         p.Stats.SuccessfulMoves,
			RecommendationsSent:     p.Stats.RecommendationsSent,
			RecommendationsReceived: p.Stats.RecommendationsReceived,
			RecommendationsAccepted: p.Stats.RecommendationsAccepted,
			FinalFragmentCorrect:    correct,
		}

		totalMoves += p.Stats.FragmentMoves
		successfulMoves += p.Stats.SuccessfulMoves
		totalRecs += p.Stats.RecommendationsSent
		acceptedRecs += p.Stats.RecommendationsAccepted
		respSum += p.Stats.RecResponseTimeSum
		respCount += p.Stats.RecResponses

		if p.Stats.CompletedIndividual {
			completed++
			solveSum += p.Stats.IndividualSolveTime
			if completed == 1 || p.Stats.IndividualSolveTime < fastest {
				fastest = p.Stats.IndividualSolveTime
			}
			if p.Stats.IndividualSolveTime > slowest {
				slowest = p.Stats.IndividualSolveTime
			}
		}
	}

	collab := protocol.CollaborationMetrics{
		TotalMoves:              totalMoves,
		SuccessfulMoves:         successfulMoves,
		TotalRecommendations:    totalRecs,
		AcceptedRecommendations: acceptedRecs,
	}
	if respCount > 0 {
		collab.AverageResponseTime = round2(respSum / float64(respCount))
	}

	transitions := protocol.IndividualPhaseTransitions{PlayersCompletedIndividual: completed}
	if completed > 0 {
		transitions.AverageIndividualTime = round2(solveSum / float64(completed))
		transitions.FastestIndividual = fastest
		transitions.SlowestIndividual = slowest
	}

	return protocol.CompletionAnalytics{
		PuzzleSuccess:        success,
		CompletionTime:       completionSeconds,
		TotalTime:            e.puzzle.totalTime.Seconds(),
		PlayerContributions:  contributions,
		CollaborationMetrics: collab,
		PhaseTransitions:     transitions,
	}
}

// ── Host-disconnect pause ──────────────────────────────────────────────────

// pauseAssembly freezes the entire phase: deadline, preview window,
// recommendation timeouts, grid ticks, and per-fragment cooldowns.
func (e *Engine) pauseAssembly() time.Time {
	e.puzzle.paused = true
	e.puzzle.pausedAt = time.Now()
	e.timers.PauseAll()
	e.freezeCooldowns()
	return e.puzzle.pausedAt
}

// resumeAssembly re-arms everything frozen by pauseAssembly, extending each
// deadline by the pause duration.
func (e *Engine) resumeAssembly() {
	e.puzzle.paused = false
	e.timers.ResumeAll()
	e.thawCooldowns()
}

// previewWindowOpen reports whether /api/preview/full may serve right now.
// The window is flag-based, not clock-based, so a host-disconnect pause
// (which freezes the expiry timer) freezes the window too.
func (e *Engine) previewWindowOpen() bool {
	return e.phase == protocol.PhasePuzzleAssembly &&
		e.puzzle.effects.ClarityPreviewDuration > 0 &&
		!e.puzzle.previewExpired
}

// replayHostAssemblyState restores a reconnecting host mid-assembly:
// PHASE_LOAD, GRID_STATE, then PHASE_START re-anchored to the resume
// (totalTime = seconds remaining; baseTime/chronosBonus keep original
// values, display-only).
func (e *Engine) replayHostAssemblyState() {
	e.sendHost(protocol.PuzzleToHostPhaseLoad, e.buildHostPhaseLoad())
	e.sendHost(protocol.PuzzleToHostGridState, e.buildHostGridState())
	e.sendHost(protocol.PuzzleToHostPhaseStart,
		e.buildHostPhaseStart(round2(e.puzzleTimeRemaining().Seconds()), time.Now()))
}

// round1 rounds to one decimal (completionPercentage precision).
func round1(x float64) float64 {
	return math.Round(x*10) / 10
}
