package game

import (
	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/protocol"
	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/puzzle"
)

// drawHighlights draws a player's guide-highlight set the moment their
// fragment enters the central grid. The set always contains the fragment's
// correct cell plus random decoys, and never changes once drawn
// (game-design.md § Guide Tokens).
func (e *Engine) drawHighlights(p *Player) {
	count := e.puzzle.effects.GuideHighlightCount
	if count == 0 {
		e.puzzle.highlights[p.ID] = []protocol.Position{}
		return
	}

	correct, err := puzzle.CorrectPos(e.puzzle.assignments[p.ID], e.puzzle.gridSize)
	if err != nil {
		e.puzzle.highlights[p.ID] = []protocol.Position{}
		return
	}

	g := e.puzzle.gridSize
	decoys := make([]puzzle.Pos, 0, g*g-1)
	for y := 0; y < g; y++ {
		for x := 0; x < g; x++ {
			if pos := (puzzle.Pos{X: x, Y: y}); pos != correct {
				decoys = append(decoys, pos)
			}
		}
	}
	e.rng.Shuffle(len(decoys), func(i, j int) { decoys[i], decoys[j] = decoys[j], decoys[i] })

	cells := append([]puzzle.Pos{correct}, decoys[:count-1]...)
	e.rng.Shuffle(len(cells), func(i, j int) { cells[i], cells[j] = cells[j], cells[i] })

	highlights := make([]protocol.Position, len(cells))
	for i, c := range cells {
		highlights[i] = posOf(c)
	}
	e.puzzle.highlights[p.ID] = highlights
}

// personalState builds PUZZLE_TO_PLAYER_PERSONAL_STATE for one 2B player.
func (e *Engine) personalState(p *Player) protocol.PersonalState {
	highlights := e.puzzle.highlights[p.ID]
	if highlights == nil {
		highlights = []protocol.Position{}
	}
	return protocol.PersonalState{GuideHighlights: highlights}
}
