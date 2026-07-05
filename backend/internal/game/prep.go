package game

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"time"

	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/protocol"
	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/puzzle"
)

// puzzleState is everything frozen at (or accumulated after) the end of
// resource gathering: grid geometry, threshold effects, segment
// assignments, and the in-memory tile cache.
type puzzleState struct {
	gridSize    int
	thresholds  protocol.ThresholdSet
	effects     protocol.BonusEffects
	assignments map[string]string // playerID → segmentID

	tiles      map[string][]byte
	preview    []byte
	generating bool
	tilesReady bool

	timerRunning   bool
	startTime      time.Time
	totalTime      time.Duration
	previewExpired bool

	// Assembly state.
	playersAtPuzzleStart int
	grid                 map[string]*Fragment // segmentID → visible fragment
	enteredGrid          map[string]bool      // playerID → fragment on grid (completion or auto-solve)
	highlights           map[string][]protocol.Position
	recommendations      map[string]*Recommendation // moveID → pending
	pendingBySender      map[string]string          // senderID → moveID
	paused               bool
	pausedAt             time.Time
	finished             bool
}

// startPuzzlePreparation begins tile generation. Called from
// enterPuzzlePreparation after the phase-complete events; e.puzzle's
// geometry fields are already frozen.
func (e *Engine) startPuzzlePreparation() {
	e.assignSegments()
	e.sendHost(protocol.PuzzleToHostPreparing, struct{}{})

	e.puzzle.generating = true
	dir, imageName := e.opts.PuzzleSourcesDir, e.cfg.PuzzleImage
	gridSize := e.puzzle.gridSize
	// Generation runs off the engine goroutine (image decode + PNG encode
	// of gridSize² tiles is real work); the result posts back as a command.
	go func() {
		tiles, preview, err := generateTiles(dir, imageName, gridSize)
		e.post(cmdTilesReady{tiles: tiles, preview: preview, err: err})
	}()
}

// generateTiles loads the source image and produces the per-segment tile
// cache plus the full-image clarity preview (the un-cropped source).
func generateTiles(dir, imageName string, gridSize int) (map[string][]byte, []byte, error) {
	f, err := os.Open(filepath.Join(dir, imageName))
	if err != nil {
		return nil, nil, fmt.Errorf("open puzzle image: %w", err)
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, nil, fmt.Errorf("decode puzzle image: %w", err)
	}

	tiles, err := puzzle.GenerateTiles(img, gridSize)
	if err != nil {
		return nil, nil, err
	}
	preview, err := puzzle.EncodePNG(img)
	if err != nil {
		return nil, nil, err
	}
	return tiles, preview, nil
}

// assignSegments maps players to segments row-major in join order
// (player 1 → segment_a1, player 2 → segment_a2, ...).
func (e *Engine) assignSegments() {
	ids := puzzle.AllSegmentIDs(e.puzzle.gridSize)
	e.puzzle.assignments = map[string]string{}
	for i, playerID := range e.playerIDsInJoinOrder() {
		e.puzzle.assignments[playerID] = ids[i]
	}
}

func (e *Engine) handleTilesReady(c cmdTilesReady) {
	e.puzzle.generating = false
	if c.err != nil {
		// Startup validation guarantees a decodable image, so this is an
		// operational failure (file removed at runtime). The game cannot
		// proceed; recovery is a deployment restart per game-design.md.
		e.log.Error("tile generation failed; game cannot proceed", "error", c.err)
		return
	}
	e.puzzle.tiles = c.tiles
	e.puzzle.preview = c.preview
	e.puzzle.tilesReady = true

	e.sendHost(protocol.PuzzleToHostReady, struct{}{})
	e.sendHost(protocol.PuzzleToHostPhaseLoad, e.buildHostPhaseLoad())
	for _, p := range e.players {
		if p.Connected {
			p.send(protocol.PuzzleToClientPhaseLoad, e.buildPhaseLoad(p))
		}
	}
}

func (e *Engine) buildPhaseLoad(p *Player) protocol.PuzzlePhaseLoad {
	return protocol.PuzzlePhaseLoad{
		Phase:                  e.phase,
		ImageID:                e.cfg.PuzzleImage,
		AssignedSegmentID:      e.puzzle.assignments[p.ID],
		IndividualPuzzleSize:   e.cfg.IndividualPuzzlePieces,
		AnchorPreSolvedPieces:  e.puzzle.effects.AnchorPreSolved,
		CentralGridSize:        e.puzzle.gridSize,
		TotalFragments:         e.puzzle.gridSize * e.puzzle.gridSize,
		ClarityPreviewDuration: e.puzzle.effects.ClarityPreviewDuration,
		GuideHighlightCount:    e.puzzle.effects.GuideHighlightCount,
	}
}

func (e *Engine) buildHostPhaseLoad() protocol.HostPuzzlePhaseLoad {
	assignments := make(map[string]string, len(e.puzzle.assignments))
	for k, v := range e.puzzle.assignments {
		assignments[k] = v
	}
	return protocol.HostPuzzlePhaseLoad{
		Phase:                    e.phase,
		ImageID:                  e.cfg.PuzzleImage,
		CentralGridSize:          e.puzzle.gridSize,
		TotalFragments:           e.puzzle.gridSize * e.puzzle.gridSize,
		PlayerCount:              len(e.players),
		PlayerSegmentAssignments: assignments,
		BonusEffects:             e.puzzle.effects,
	}
}

// handlePuzzlePhaseStart processes the host's PUZZLE_TO_SERVER_PHASE_START.
func (e *Engine) handlePuzzlePhaseStart() {
	if e.phase != protocol.PhasePuzzlePreparation || !e.puzzle.tilesReady {
		e.sendHostError(protocol.ErrorTypeGameState, protocol.ErrForbiddenPhase,
			"puzzle timer cannot start now",
			"PUZZLE_TO_SERVER_PHASE_START requires completed tile generation and is only valid once")
		return
	}
	e.enterPuzzleAssembly()
}

// replayHostPrepState restores host state after a mid-preparation reconnect:
// PREPARING while generation runs, otherwise READY + PHASE_LOAD.
func (e *Engine) replayHostPrepState() {
	if e.puzzle.tilesReady {
		e.sendHost(protocol.PuzzleToHostReady, struct{}{})
		e.sendHost(protocol.PuzzleToHostPhaseLoad, e.buildHostPhaseLoad())
	} else {
		e.sendHost(protocol.PuzzleToHostPreparing, struct{}{})
	}
}

// replayPlayerPrepState restores a player after a mid-preparation reconnect:
// PHASE_LOAD if generation already finished (otherwise the normal broadcast
// will reach them).
func (e *Engine) replayPlayerPrepState(p *Player) {
	if e.puzzle.tilesReady {
		p.send(protocol.PuzzleToClientPhaseLoad, e.buildPhaseLoad(p))
	}
}
