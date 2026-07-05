package game

import (
	"time"

	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/protocol"
)

// Puzzle-assembly timers. Both pause during a host disconnect.
const (
	timerAssemblyDeadline = "assembly.deadline"
	timerAssemblyPreview  = "assembly.preview"
)

// enterPuzzleAssembly starts the puzzle timer (host signal already
// validated). Grid mechanics (2A/2B, moves, recommendations) are built on
// top of this in milestone M6.
func (e *Engine) enterPuzzleAssembly() {
	e.phase = protocol.PhasePuzzleAssembly
	e.puzzle.timerRunning = true
	e.puzzle.startTime = time.Now()

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

	e.setupAssemblyGrid()

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
}

// setupAssemblyGrid initializes the central grid and auto-solves segments of
// players still disconnected at timer start. Extended in M6.
func (e *Engine) setupAssemblyGrid() {}

// playerPhases partitions players into 2A (individual) and 2B (grid).
// Until M6 adds completion tracking, everyone starts in 2A.
func (e *Engine) playerPhases() protocol.PlayerPhases {
	phases := protocol.PlayerPhases{Phase2A: []string{}, Phase2B: []string{}}
	for _, id := range e.playerIDsInJoinOrder() {
		phases.Phase2A = append(phases.Phase2A, id)
	}
	return phases
}

func (e *Engine) buildHostPhaseStart(totalSeconds float64, start time.Time) protocol.HostPuzzlePhaseStart {
	phases := e.playerPhases()
	return protocol.HostPuzzlePhaseStart{
		TimerActive:      e.puzzle.timerRunning,
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
	switch name {
	case timerAssemblyPreview:
		e.puzzle.previewExpired = true
		e.broadcastPlayers(protocol.PuzzleToClientPreviewExpired, struct{}{})
	case timerAssemblyDeadline:
		e.handlePuzzleTimeout()
	}
}

// handlePuzzleTimeout ends the game as a loss. Full implementation in M6/M7.
func (e *Engine) handlePuzzleTimeout() {
	e.puzzle.timerRunning = false
	e.phase = protocol.PhaseAnalytics
}

// previewWindowOpen reports whether /api/preview/full may serve right now.
// The window is flag-based, not clock-based, so a host-disconnect pause
// (which freezes the expiry timer) freezes the window too.
func (e *Engine) previewWindowOpen() bool {
	return e.phase == protocol.PhasePuzzleAssembly &&
		e.puzzle.effects.ClarityPreviewDuration > 0 &&
		!e.puzzle.previewExpired
}
