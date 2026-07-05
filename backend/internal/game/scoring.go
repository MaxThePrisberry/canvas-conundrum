package game

import (
	"sort"

	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/protocol"
)

// scoreBreakdown computes one player's score per game-design.md § Scoring
// Algorithm. The completion bonus applies to everyone when the team's
// puzzle succeeded; recommendationsAccepted counts recommendations the
// player received and accepted.
func (e *Engine) scoreBreakdown(p *Player, gameSuccess bool) protocol.ScoreBreakdown {
	b := protocol.ScoreBreakdown{
		TriviaPoints:    p.Stats.CorrectAnswers * e.cfg.PointsPerCorrectAnswer,
		SpecialtyPoints: p.Stats.SpecialtyCorrect * e.cfg.SpecialtyBonusPoints,
		MovePoints:      p.Stats.SuccessfulMoves * e.cfg.PointsPerSuccessfulMove,
		RecommendationPoints: p.Stats.RecommendationsSent*e.cfg.PointsPerRecommendationSent +
			p.Stats.RecommendationsAccepted*e.cfg.PointsPerRecommendationAccepted,
	}
	if gameSuccess {
		b.CompletionBonus = e.cfg.CompletionBonus
	}
	b.TotalScore = b.TriviaPoints + b.SpecialtyPoints + b.CompletionBonus +
		b.MovePoints + b.RecommendationPoints
	return b
}

// buildLeaderboard sorts players by score descending with competition
// ranking (1, 2, 2, 4); ties order alphabetically by playerName.
func (e *Engine) buildLeaderboard(scores map[string]protocol.ScoreBreakdown) []protocol.LeaderboardEntry {
	entries := make([]protocol.LeaderboardEntry, 0, len(e.players))
	for id, p := range e.players {
		entries = append(entries, protocol.LeaderboardEntry{
			PlayerID:   id,
			PlayerName: p.Name,
			TotalScore: scores[id].TotalScore,
			Role:       p.Role,
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].TotalScore != entries[j].TotalScore {
			return entries[i].TotalScore > entries[j].TotalScore
		}
		return entries[i].PlayerName < entries[j].PlayerName
	})
	for i := range entries {
		if i > 0 && entries[i].TotalScore == entries[i-1].TotalScore {
			entries[i].Rank = entries[i-1].Rank
		} else {
			entries[i].Rank = i + 1
		}
	}
	return entries
}

// individualRanks ranks real 2A completions by solve time ascending
// (competition ranking); auto-solved players get rank 0.
func (e *Engine) individualRanks() map[string]int {
	type solve struct {
		id   string
		time float64
	}
	var solves []solve
	for id, p := range e.players {
		if p.Stats.CompletedIndividual {
			solves = append(solves, solve{id, p.Stats.IndividualSolveTime})
		}
	}
	sort.Slice(solves, func(i, j int) bool { return solves[i].time < solves[j].time })

	ranks := map[string]int{}
	for i, s := range solves {
		if i > 0 && s.time == solves[i-1].time {
			ranks[s.id] = ranks[solves[i-1].id]
		} else {
			ranks[s.id] = i + 1
		}
	}
	return ranks
}
