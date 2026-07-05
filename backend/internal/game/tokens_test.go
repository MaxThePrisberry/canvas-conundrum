package game

import (
	"testing"
	"time"

	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/config"
	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/protocol"
)

// specConfig mirrors the committed game-config.json values relevant to the
// bonus-effect formulas, so these tests double-check the spec's canonical
// example numbers (websocket-events.md § Canonical example game).
func specConfig() *config.Config {
	return &config.Config{
		BaseTokensPerCorrectAnswer: 20,
		RoleResourceMultiplier:     1.5,
		SpecialtyPointMultiplier:   2.0,
		AnchorTokenThreshold:       25,
		ChronosTokenThreshold:      20,
		GuideTokenThreshold:        15,
		ClarityTokenThreshold:      30,
		MaxThresholds:              6,
		IndividualPuzzlePieces:     16,
		DifficultyMode:             "medium",
		MediumThresholdMultiplier:  1.0,
		TimeExtensionPerThreshold:  config.Seconds(20 * time.Second),
		ClarityBasePreviewTime:     config.Seconds(3 * time.Second),
		PreviewTimePerThreshold:    config.Seconds(time.Second),
	}
}

// TestCanonicalExampleNumbers reproduces the spec's example game: tokens
// anchor 90 / chronos 70 / guide 50 / clarity 70 → thresholds 3/3/3/2,
// 6 pre-solved pieces, +60s, 5 highlights on a 3×3 grid, 5s preview.
func TestCanonicalExampleNumbers(t *testing.T) {
	e := &Engine{cfg: specConfig()}
	e.tokens = protocol.TeamTokens{AnchorTokens: 90, ChronosTokens: 70, GuideTokens: 50, ClarityTokens: 70}

	got := e.currentThresholds()
	want := protocol.ThresholdSet{Anchor: 3, Chronos: 3, Guide: 3, Clarity: 2}
	if got != want {
		t.Fatalf("thresholds = %+v, want %+v", got, want)
	}

	be := e.bonusEffects(got, 3)
	if be.AnchorPreSolved != 6 {
		t.Errorf("anchorPreSolved = %d, want 6", be.AnchorPreSolved)
	}
	if be.ChronosTimeBonus != 60 {
		t.Errorf("chronosTimeBonus = %v, want 60", be.ChronosTimeBonus)
	}
	if be.GuideHighlightCount != 5 {
		t.Errorf("guideHighlightCount = %d, want 5", be.GuideHighlightCount)
	}
	if be.ClarityPreviewDuration != 5 {
		t.Errorf("clarityPreviewDuration = %v, want 5", be.ClarityPreviewDuration)
	}
}

func TestAnchorPreSolveCap(t *testing.T) {
	e := &Engine{cfg: specConfig()}
	// 16 pieces → cap floor(16×0.75)=12, perThreshold ceil(12/6)=2.
	cases := map[int]int{0: 0, 1: 2, 3: 6, 6: 12}
	for n, want := range cases {
		if got := e.anchorPreSolvedCount(n); got != want {
			t.Errorf("anchorPreSolvedCount(%d) = %d, want %d", n, got, want)
		}
	}
}

// Full guide thresholds must highlight exactly the one correct cell, and
// integer arithmetic must not inflate exact divisions (the float form turns
// 9 × (1 − 4/6) into 3.0000000000000004 → 4).
func TestGuideHighlightBounds(t *testing.T) {
	e := &Engine{cfg: specConfig()}
	if got := e.guideHighlightCount(6, 3); got != 1 {
		t.Errorf("full thresholds → %d highlights, want exactly 1", got)
	}
	if got := e.guideHighlightCount(4, 3); got != 3 {
		t.Errorf("N=4 on 3×3 → %d, want 3", got)
	}
	if got := e.guideHighlightCount(0, 3); got != 0 {
		t.Errorf("N=0 → %d, want 0", got)
	}
}

func TestAwardTokensMatrix(t *testing.T) {
	e := &Engine{cfg: specConfig()}
	detective := &Player{Role: "detective"} // bonus station: guide

	// Unknown station: nothing, even when correct.
	if earned, _ := e.awardTokens(detective, true); earned != 0 {
		t.Errorf("unknown station earned %d, want 0", earned)
	}

	detective.Station = "guide"
	earned, bonuses := e.awardTokens(detective, false)
	if earned != 30 || bonuses.RoleBonusTokens != 10 || !bonuses.RoleBonus {
		t.Errorf("role match: earned=%d bonuses=%+v", earned, bonuses)
	}

	earned, bonuses = e.awardTokens(detective, true)
	if earned != 60 || bonuses.SpecialtyBonusTokens != 30 {
		t.Errorf("role+specialty: earned=%d bonuses=%+v", earned, bonuses)
	}

	detective.Station = "anchor" // not the detective's bonus station
	earned, bonuses = e.awardTokens(detective, true)
	if earned != 40 || bonuses.RoleBonus || bonuses.SpecialtyBonusTokens != 20 {
		t.Errorf("specialty only: earned=%d bonuses=%+v", earned, bonuses)
	}
}
