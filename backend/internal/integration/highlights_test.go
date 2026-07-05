package integration

import (
	"testing"
	"time"

	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/protocol"
)

// Guide highlights: drawn once when the player's fragment enters the grid,
// always containing the fragment's correct cell, immutable across ticks,
// private per player, and delivered only to 2B players.
func TestGuideHighlights(t *testing.T) {
	h := Start(t, noSpecialty, nil)
	host, players := JoinConfigured(t, h, 2)
	a, b := players[0], players[1] // b: detective (guide bonus)
	host.StartGame()

	// b farms guide: 2 rounds × 30 = 60 → floor(60/15) = 4 thresholds →
	// max(1, ceil(9×(1−4/6))) = 3 highlight cells.
	complete := PlayResource(t, host, players, 2, map[int]string{1: "hash-guide"})
	if complete.BonusEffects.GuideHighlightCount != 3 {
		t.Fatalf("highlight count = %d, want 3", complete.BonusEffects.GuideHighlightCount)
	}
	host.Expect(protocol.PuzzleToHostReady)
	host.StartPuzzle()
	a.Expect(protocol.PuzzleToClientPhaseStart)
	b.Expect(protocol.PuzzleToClientPhaseStart)

	// 2A players receive no personal state.
	b.ExpectNone(protocol.PuzzleToPlayerPersonalState, 150*time.Millisecond)

	// Completion delivers the first snapshot immediately.
	b.CompleteSegment("segment_a2", 1)
	first := payloadAs[protocol.PersonalState](t, b.Expect(protocol.PuzzleToPlayerPersonalState))
	if len(first.GuideHighlights) != 3 {
		t.Fatalf("highlights = %v", first.GuideHighlights)
	}
	containsCorrect := false
	seen := map[protocol.Position]bool{}
	for _, pos := range first.GuideHighlights {
		if seen[pos] {
			t.Errorf("duplicate highlight cell %+v", pos)
		}
		seen[pos] = true
		if pos == (protocol.Position{X: 1, Y: 0}) { // segment_a2's correct cell
			containsCorrect = true
		}
	}
	if !containsCorrect {
		t.Error("highlight set must contain the fragment's correct cell")
	}

	// Immutable: the same set arrives on every subsequent tick.
	for i := 0; i < 3; i++ {
		next := payloadAs[protocol.PersonalState](t, b.Expect(protocol.PuzzleToPlayerPersonalState))
		if len(next.GuideHighlights) != len(first.GuideHighlights) {
			t.Fatalf("highlight set changed size: %v", next.GuideHighlights)
		}
		for j, pos := range next.GuideHighlights {
			if pos != first.GuideHighlights[j] {
				t.Fatalf("highlight set mutated: %v vs %v", next.GuideHighlights, first.GuideHighlights)
			}
		}
	}

	// A completes too: their private set contains segment_a1's correct cell.
	a.CompleteSegment("segment_a1", 1)
	aState := payloadAs[protocol.PersonalState](t, a.Expect(protocol.PuzzleToPlayerPersonalState))
	foundA := false
	for _, pos := range aState.GuideHighlights {
		if pos == (protocol.Position{X: 0, Y: 0}) {
			foundA = true
		}
	}
	if !foundA {
		t.Error("player A's highlights must contain their own correct cell")
	}
}

// With zero guide thresholds the personal state carries an empty array.
func TestGuideHighlightsEmptyWithoutThresholds(t *testing.T) {
	h := Start(t, noSpecialty, nil)
	_, players := ReachAssembly(t, h, 2, 2, nil)
	a := players[0]
	a.CompleteSegment("segment_a1", 1)

	state := payloadAs[protocol.PersonalState](t, a.Expect(protocol.PuzzleToPlayerPersonalState))
	if state.GuideHighlights == nil || len(state.GuideHighlights) != 0 {
		t.Errorf("highlights = %v, want empty array", state.GuideHighlights)
	}
}
