package game

import (
	"time"

	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/protocol"
	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/puzzle"
)

// Fragment is one visible fragment on the central grid. OwnerID "" means
// unassigned (never assigned, or the owner disconnected). Cooldowns are per
// fragment: ReadyAt is when it may next move; frozenCooldown holds the
// remaining cooldown across a host-disconnect pause.
type Fragment struct {
	SegmentID string
	OwnerID   string
	Pos       puzzle.Pos

	ReadyAt        time.Time
	frozenCooldown time.Duration
	LastMoved      time.Time
	MoveCount      int
}

func posOf(p puzzle.Pos) protocol.Position { return protocol.Position{X: p.X, Y: p.Y} }

// fragmentAt returns the fragment occupying pos, if any.
func (e *Engine) fragmentAt(pos puzzle.Pos) *Fragment {
	for _, f := range e.puzzle.grid {
		if f.Pos == pos {
			return f
		}
	}
	return nil
}

// randomOpenCell picks a uniformly random unoccupied cell. The grid always
// has room: fragments never exceed gridSize².
func (e *Engine) randomOpenCell() puzzle.Pos {
	g := e.puzzle.gridSize
	open := make([]puzzle.Pos, 0, g*g)
	for y := 0; y < g; y++ {
		for x := 0; x < g; x++ {
			pos := puzzle.Pos{X: x, Y: y}
			if e.fragmentAt(pos) == nil {
				open = append(open, pos)
			}
		}
	}
	return open[e.rng.IntN(len(open))]
}

// placeFragment creates a fragment for segmentID at a random open cell.
// Placement is a grid mutation: callers must run the victory check.
func (e *Engine) placeFragment(segmentID, ownerID string) *Fragment {
	f := &Fragment{
		SegmentID: segmentID,
		OwnerID:   ownerID,
		Pos:       e.randomOpenCell(),
		LastMoved: time.Now(),
	}
	e.puzzle.grid[segmentID] = f
	return f
}

// completedCount is k in the proportional-reveal formula: players whose
// fragment has entered the grid (real completions and auto-solves alike).
func (e *Engine) completedCount() int {
	return len(e.puzzle.enteredGrid)
}

// revealUnassigned tops the grid up to ceil((k/N) × gridSize²) visible
// fragments with randomly-selected unassigned segments at random open cells
// (game-design.md § Central Grid Mechanics). Integer arithmetic avoids
// float-ceil misrounding. Each placement is a grid mutation; the caller
// runs the victory check afterwards.
func (e *Engine) revealUnassigned() {
	g2 := e.puzzle.gridSize * e.puzzle.gridSize
	n := e.puzzle.playersAtPuzzleStart
	if n == 0 {
		return
	}
	target := (e.completedCount()*g2 + n - 1) / n

	pool := e.unrevealedSegments()
	for len(e.puzzle.grid) < target && len(pool) > 0 {
		idx := e.rng.IntN(len(pool))
		segment := pool[idx]
		pool = append(pool[:idx], pool[idx+1:]...)
		e.placeFragment(segment, "")
	}
}

// unrevealedSegments lists segments that are neither assigned to a player
// nor already visible on the grid.
func (e *Engine) unrevealedSegments() []string {
	assigned := map[string]bool{}
	for _, seg := range e.puzzle.assignments {
		assigned[seg] = true
	}
	var pool []string
	for _, seg := range puzzle.AllSegmentIDs(e.puzzle.gridSize) {
		if !assigned[seg] && e.puzzle.grid[seg] == nil {
			pool = append(pool, seg)
		}
	}
	return pool
}

// correctFragments counts visible fragments sitting on their correct cell.
func (e *Engine) correctFragments() int {
	n := 0
	for _, f := range e.puzzle.grid {
		if correct, err := puzzle.CorrectPos(f.SegmentID, e.puzzle.gridSize); err == nil && f.Pos == correct {
			n++
		}
	}
	return n
}

// victoryMet checks both conditions: every fragment present (all players
// completed ⇒ reveals reached gridSize²) and every fragment correct.
func (e *Engine) victoryMet() bool {
	g2 := e.puzzle.gridSize * e.puzzle.gridSize
	return len(e.puzzle.grid) == g2 && e.correctFragments() == g2
}

// ── Cooldowns ──────────────────────────────────────────────────────────────

func (e *Engine) fragmentOnCooldown(f *Fragment) bool {
	if e.puzzle.paused {
		return f.frozenCooldown > 0
	}
	return time.Now().Before(f.ReadyAt)
}

func (e *Engine) restartCooldown(f *Fragment) {
	f.ReadyAt = time.Now().Add(e.cfg.FragmentMoveCooldown.Duration())
	f.LastMoved = time.Now()
	f.MoveCount++
}

func (e *Engine) cooldownInfo(f *Fragment) *protocol.CooldownInfo {
	remaining := max(time.Until(f.ReadyAt), 0)
	if e.puzzle.paused {
		remaining = f.frozenCooldown
	}
	return &protocol.CooldownInfo{
		NextMoveAvailable: protocol.Timestamp(time.Now().Add(remaining)),
		CooldownRemaining: round2(remaining.Seconds()),
	}
}

// freezeCooldowns / thawCooldowns implement the host-disconnect pause for
// per-fragment cooldown clocks.
func (e *Engine) freezeCooldowns() {
	now := time.Now()
	for _, f := range e.puzzle.grid {
		f.frozenCooldown = max(f.ReadyAt.Sub(now), 0)
	}
}

func (e *Engine) thawCooldowns() {
	now := time.Now()
	for _, f := range e.puzzle.grid {
		f.ReadyAt = now.Add(f.frozenCooldown)
		f.frozenCooldown = 0
	}
}

// ── Grid state payloads ────────────────────────────────────────────────────

func (e *Engine) buildGridState() protocol.GridState {
	fragments := make([]protocol.GridFragment, 0, len(e.puzzle.grid))
	for _, seg := range puzzle.AllSegmentIDs(e.puzzle.gridSize) {
		f, ok := e.puzzle.grid[seg]
		if !ok {
			continue
		}
		gf := protocol.GridFragment{SegmentID: f.SegmentID, Position: posOf(f.Pos)}
		if owner, ok := e.players[f.OwnerID]; ok && f.OwnerID != "" {
			id, name := f.OwnerID, owner.Name
			gf.PlayerID, gf.PlayerName = &id, &name
		}
		fragments = append(fragments, gf)
	}
	return protocol.GridState{
		Fragments:     fragments,
		TimeRemaining: round2(e.puzzleTimeRemaining().Seconds()),
	}
}

func (e *Engine) buildHostGridState() protocol.HostGridState {
	fragments := make([]protocol.HostGridFragment, 0, len(e.puzzle.grid))
	for _, seg := range puzzle.AllSegmentIDs(e.puzzle.gridSize) {
		f, ok := e.puzzle.grid[seg]
		if !ok {
			continue
		}
		hf := protocol.HostGridFragment{
			SegmentID: f.SegmentID,
			Position:  posOf(f.Pos),
			LastMoved: protocol.Timestamp(f.LastMoved),
			MoveCount: f.MoveCount,
		}
		if owner, ok := e.players[f.OwnerID]; ok && f.OwnerID != "" {
			id, name := f.OwnerID, owner.Name
			hf.PlayerID, hf.PlayerName = &id, &name
		}
		fragments = append(fragments, hf)
	}

	metrics := map[string]protocol.HostPlayerMetric{}
	for id, p := range e.players {
		phase := "phase2a"
		owned := 0
		if e.puzzle.enteredGrid[id] {
			phase = "phase2b"
		}
		if f, ok := e.puzzle.grid[e.puzzle.assignments[id]]; ok && f.OwnerID == id {
			owned = 1
		}
		metrics[id] = protocol.HostPlayerMetric{
			Phase:            phase,
			FragmentsOwned:   owned,
			MovesContributed: p.Stats.FragmentMoves,
			SuccessfulMoves:  p.Stats.SuccessfulMoves,
			LastActivity:     protocol.Timestamp(p.LastActivity),
		}
	}

	return protocol.HostGridState{
		Fragments:             fragments,
		PlayerMetrics:         metrics,
		ActiveRecommendations: len(e.puzzle.recommendations),
		TimeRemaining:         round2(e.puzzleTimeRemaining().Seconds()),
	}
}

// touchGrid pushes the host's immediate grid update (players get theirs on
// the periodic tick).
func (e *Engine) touchGrid() {
	e.sendHost(protocol.PuzzleToHostGridState, e.buildHostGridState())
}

func (e *Engine) puzzleTimeRemaining() time.Duration {
	return e.timers.Remaining(timerAssemblyDeadline)
}
