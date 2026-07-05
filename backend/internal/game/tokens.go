package game

import (
	"math"

	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/protocol"
)

// Station / token type names (they share the same identifiers).
const (
	stationAnchor  = "anchor"
	stationChronos = "chronos"
	stationGuide   = "guide"
	stationClarity = "clarity"
	stationUnknown = "unknown"
)

var stationNames = []string{stationAnchor, stationChronos, stationGuide, stationClarity}

// addTokens credits a station's pool.
func (e *Engine) addTokens(station string, n int) {
	switch station {
	case stationAnchor:
		e.tokens.AnchorTokens += n
	case stationChronos:
		e.tokens.ChronosTokens += n
	case stationGuide:
		e.tokens.GuideTokens += n
	case stationClarity:
		e.tokens.ClarityTokens += n
	}
}

func tokensFor(t protocol.TeamTokens, station string) int {
	switch station {
	case stationAnchor:
		return t.AnchorTokens
	case stationChronos:
		return t.ChronosTokens
	case stationGuide:
		return t.GuideTokens
	case stationClarity:
		return t.ClarityTokens
	}
	return 0
}

// thresholdCount implements game-design.md § Resource Token System:
// min(maxThresholds, floor(tokens / (tokenThreshold × thresholdMultiplier))).
func (e *Engine) thresholdCount(station string) int {
	denom := float64(e.cfg.TokenThreshold(station)) * e.cfg.ThresholdMultiplier()
	if denom <= 0 {
		return 0
	}
	n := int(math.Floor(float64(tokensFor(e.tokens, station)) / denom))
	return min(n, e.cfg.MaxThresholds)
}

func (e *Engine) currentThresholds() protocol.ThresholdSet {
	return protocol.ThresholdSet{
		Anchor:  e.thresholdCount(stationAnchor),
		Chronos: e.thresholdCount(stationChronos),
		Guide:   e.thresholdCount(stationGuide),
		Clarity: e.thresholdCount(stationClarity),
	}
}

// awardTokens computes one correct answer's token award (game-design.md
// § Token Scoring): base × roleMultiplier (if at the role's bonus station)
// × specialtyMultiplier (if a specialty question), floored to an integer.
// The returned bonuses decompose the award for display.
func (e *Engine) awardTokens(p *Player, isSpecialty bool) (int, protocol.AnswerBonuses) {
	if p.Station == "" {
		// "unknown" station earns nothing, even on a correct answer.
		return 0, protocol.AnswerBonuses{}
	}

	base := float64(e.cfg.BaseTokensPerCorrectAnswer)
	roleFactor := 1.0
	roleMatch := false
	if role, ok := roleByType(p.Role); ok && role.BonusTokenType == p.Station {
		roleFactor = e.cfg.RoleResourceMultiplier
		roleMatch = true
	}
	specialtyFactor := 1.0
	if isSpecialty {
		specialtyFactor = e.cfg.SpecialtyPointMultiplier
	}

	earned := int(math.Floor(base * roleFactor * specialtyFactor))
	bonuses := protocol.AnswerBonuses{
		RoleBonus:      roleMatch,
		SpecialtyBonus: isSpecialty,
	}
	if roleMatch {
		bonuses.RoleBonusTokens = int(math.Floor(base * (e.cfg.RoleResourceMultiplier - 1)))
	}
	if isSpecialty {
		bonuses.SpecialtyBonusTokens = int(math.Floor(base * roleFactor * (e.cfg.SpecialtyPointMultiplier - 1)))
	}
	return earned, bonuses
}

// ── Threshold → puzzle-phase effect formulas (game-design.md § Resource
// Token System, items 1–4) ─────────────────────────────────────────────────

// anchorPreSolvedCount: perThreshold = ceil(maxPreSolved/maxThresholds)
// pieces per threshold, cumulative, capped at floor(pieces × 0.75).
func (e *Engine) anchorPreSolvedCount(thresholds int) int {
	maxPre := int(math.Floor(float64(e.cfg.IndividualPuzzlePieces) * 0.75))
	if maxPre == 0 || thresholds == 0 {
		return 0
	}
	perThreshold := int(math.Ceil(float64(maxPre) / float64(e.cfg.MaxThresholds)))
	return min(maxPre, perThreshold*thresholds)
}

// chronosBonusSeconds: +timeExtensionPerThreshold per threshold.
func (e *Engine) chronosBonusSeconds(thresholds int) float64 {
	return float64(thresholds) * e.cfg.TimeExtensionPerThreshold.Sec()
}

// guideHighlightCount: 0 without thresholds; otherwise
// max(1, ceil(gridSize² × (1 − N/maxThresholds))) — exactly 1 at full
// thresholds. Computed in integer arithmetic: the float form misrounds
// (e.g. 9 × (1 − 4/6) evaluates to 3.0000000000000004, ceiling to 4).
func (e *Engine) guideHighlightCount(thresholds, gridSize int) int {
	if thresholds == 0 {
		return 0
	}
	cells := gridSize * gridSize
	maxT := e.cfg.MaxThresholds
	n := (cells*(maxT-thresholds) + maxT - 1) / maxT // ceil(cells × (maxT−N) / maxT)
	return max(1, n)
}

// clarityPreviewSeconds: 0 without thresholds; otherwise base + N×perThreshold.
func (e *Engine) clarityPreviewSeconds(thresholds int) float64 {
	if thresholds == 0 {
		return 0
	}
	return e.cfg.ClarityBasePreviewTime.Sec() + float64(thresholds)*e.cfg.PreviewTimePerThreshold.Sec()
}

// bonusEffects assembles the RESOURCE_TO_CLIENT_PHASE_COMPLETE block for the
// given achieved thresholds and grid size.
func (e *Engine) bonusEffects(t protocol.ThresholdSet, gridSize int) protocol.BonusEffects {
	return protocol.BonusEffects{
		AnchorPreSolved:        e.anchorPreSolvedCount(t.Anchor),
		ChronosTimeBonus:       e.chronosBonusSeconds(t.Chronos),
		GuideHighlightCount:    e.guideHighlightCount(t.Guide, gridSize),
		ClarityPreviewDuration: e.clarityPreviewSeconds(t.Clarity),
	}
}
