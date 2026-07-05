package game

import (
	"math"
	"time"

	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/protocol"
)

// analyticsState holds the game outcome and the frozen report payloads
// (rebuilt once at phase entry, replayed verbatim on reconnection).
type analyticsState struct {
	success           bool
	completionSeconds float64
	endedAt           time.Time

	personalReports map[string]protocol.PersonalReport
	teamSummary     protocol.TeamSummary
	hostReport      protocol.HostCompleteReport
}

// enterAnalytics transitions puzzle_assembly → analytics: computes scores
// and reports once, then distributes them (host completion analytics was
// already sent by finishPuzzle).
func (e *Engine) enterAnalytics(success bool, completionSeconds float64) {
	e.phase = protocol.PhaseAnalytics
	e.analytics = analyticsState{
		success:           success,
		completionSeconds: completionSeconds,
		endedAt:           time.Now(),
	}
	e.buildReports()

	e.sendHost(protocol.AnalyticsToHostCompleteReport, e.analytics.hostReport)
	for id, p := range e.players {
		if p.Connected {
			p.send(protocol.AnalyticsToPlayerPersonalReport, e.analytics.personalReports[id])
		}
	}
	e.broadcastPlayers(protocol.AnalyticsToClientTeamSummary, e.analytics.teamSummary)
}

func (e *Engine) buildReports() {
	success := e.analytics.success
	rounds := e.cfg.ResourceGatheringRounds
	slots := e.resource.playersAtStart * rounds

	scores := map[string]protocol.ScoreBreakdown{}
	for id, p := range e.players {
		scores[id] = e.scoreBreakdown(p, success)
	}
	leaderboard := e.buildLeaderboard(scores)
	rankByID := map[string]int{}
	for _, entry := range leaderboard {
		rankByID[entry.PlayerID] = entry.Rank
	}
	solveRanks := e.individualRanks()

	totalScore, totalCorrect := 0, 0
	for id, p := range e.players {
		totalScore += scores[id].TotalScore
		totalCorrect += p.Stats.CorrectAnswers
	}
	overallAccuracy := 0.0
	if slots > 0 {
		overallAccuracy = round2(float64(totalCorrect) / float64(slots))
	}
	totalTokens := e.tokens.AnchorTokens + e.tokens.ChronosTokens + e.tokens.GuideTokens + e.tokens.ClarityTokens

	// ── Personal reports ───────────────────────────────────────────────
	e.analytics.personalReports = map[string]protocol.PersonalReport{}
	for id, p := range e.players {
		accuracyByCategory := map[string]float64{}
		for cat, asked := range p.Stats.QuestionsByCategory {
			accuracyByCategory[cat] = round2(float64(p.Stats.CorrectByCategory[cat]) / float64(asked))
		}
		specialtyAccuracy := 0.0
		if p.Stats.SpecialtyDelivered > 0 {
			specialtyAccuracy = round2(float64(p.Stats.SpecialtyCorrect) / float64(p.Stats.SpecialtyDelivered))
		}
		avgResponse := 0.0
		if p.Stats.QuestionsAnswered > 0 {
			avgResponse = round2(p.Stats.TotalResponseTime / float64(p.Stats.QuestionsAnswered))
		}
		moveAccuracy := 0.0
		if p.Stats.FragmentMoves > 0 {
			moveAccuracy = round3(float64(p.Stats.SuccessfulMoves) / float64(p.Stats.FragmentMoves))
		}

		e.analytics.personalReports[id] = protocol.PersonalReport{
			PlayerID:      id,
			PlayerName:    p.Name,
			GameSuccess:   success,
			PersonalScore: scores[id].TotalScore,
			Rank:          rankByID[id],
			TotalPlayers:  len(e.players),
			TokenCollection: protocol.TokenCollection{
				AnchorTokens:  p.Stats.TokensByStation[stationAnchor],
				ChronosTokens: p.Stats.TokensByStation[stationChronos],
				GuideTokens:   p.Stats.TokensByStation[stationGuide],
				ClarityTokens: p.Stats.TokensByStation[stationClarity],
				TotalTokens:   p.Stats.TokensEarned,
			},
			TriviaPerformance: protocol.TriviaPerformance{
				TotalQuestions:     rounds,
				CorrectAnswers:     p.Stats.CorrectAnswers,
				Accuracy:           round2(float64(p.Stats.CorrectAnswers) / float64(rounds)),
				AccuracyByCategory: accuracyByCategory,
				SpecialtyPerformance: protocol.PersonalSpecialtyPerformance{
					SpecialtyQuestions: p.Stats.SpecialtyDelivered,
					SpecialtyCorrect:   p.Stats.SpecialtyCorrect,
					SpecialtyAccuracy:  specialtyAccuracy,
					BonusTokens:        p.Stats.SpecialtyBonusTokens,
				},
				AverageResponseTime: avgResponse,
			},
			PuzzleSolvingMetrics: protocol.PuzzleSolvingMetrics{
				IndividualSolveTime:     p.Stats.IndividualSolveTime,
				IndividualRank:          solveRanks[id],
				FragmentMoves:           p.Stats.FragmentMoves,
				SuccessfulMoves:         p.Stats.SuccessfulMoves,
				MoveAccuracy:            moveAccuracy,
				RecommendationsSent:     p.Stats.RecommendationsSent,
				RecommendationsReceived: p.Stats.RecommendationsReceived,
				RecommendationsAccepted: p.Stats.RecommendationsAccepted,
			},
			ScoreBreakdown: scores[id],
		}
	}

	// ── Team summary ───────────────────────────────────────────────────
	timeline := e.buildTimeline()
	totalGameTime := round2(timeline.SetupPhase + timeline.ResourcePhase +
		timeline.PreparationPhase + timeline.PuzzlePhase)

	e.analytics.teamSummary = protocol.TeamSummary{
		GameSuccess:   success,
		TotalScore:    totalScore,
		TotalPlayers:  len(e.players),
		TotalGameTime: totalGameTime,
		TeamPerformance: protocol.SummaryTeamPerformance{
			OverallAccuracy:       overallAccuracy,
			TotalTokensCollected:  totalTokens,
			ThresholdAchievements: e.puzzle.thresholds,
			PuzzleCompletionTime:  e.analytics.completionSeconds,
		},
		Leaderboard: leaderboard,
	}

	// ── Host complete report ───────────────────────────────────────────
	resourcePerf := map[string]protocol.HostResourcePlayerPerformance{}
	contributions := map[string]protocol.HostPlayerContribution{}
	totalMoves, successfulMoves, totalRecs, acceptedRecs := 0, 0, 0, 0
	completed := 0
	var solveSum, fastest, slowest float64
	categories := map[string]protocol.CategoryPerformance{}

	for id, p := range e.players {
		avgResponse := 0.0
		if p.Stats.QuestionsAnswered > 0 {
			avgResponse = round2(p.Stats.TotalResponseTime / float64(p.Stats.QuestionsAnswered))
		}
		stations := map[string]int{}
		for s, n := range p.Stats.StationVisits {
			stations[s] = n
		}
		resourcePerf[id] = protocol.HostResourcePlayerPerformance{
			QuestionsAnswered:   rounds,
			CorrectAnswers:      p.Stats.CorrectAnswers,
			Accuracy:            round2(float64(p.Stats.CorrectAnswers) / float64(rounds)),
			TokensEarned:        p.Stats.TokensEarned,
			AverageResponseTime: avgResponse,
			SpecialtyPerformance: protocol.SpecialtyPerformance{
				QuestionsReceived: p.Stats.SpecialtyDelivered,
				CorrectAnswers:    p.Stats.SpecialtyCorrect,
				BonusTokens:       p.Stats.SpecialtyBonusTokens,
			},
			StationPreferences: stations,
		}
		contributions[id] = protocol.HostPlayerContribution{
			IndividualSolveTime:     p.Stats.IndividualSolveTime,
			FragmentMoves:           p.Stats.FragmentMoves,
			SuccessfulMoves:         p.Stats.SuccessfulMoves,
			RecommendationsSent:     p.Stats.RecommendationsSent,
			RecommendationsReceived: p.Stats.RecommendationsReceived,
			RecommendationsAccepted: p.Stats.RecommendationsAccepted,
		}

		totalMoves += p.Stats.FragmentMoves
		successfulMoves += p.Stats.SuccessfulMoves
		totalRecs += p.Stats.RecommendationsSent
		acceptedRecs += p.Stats.RecommendationsAccepted
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

		for cat, asked := range p.Stats.QuestionsByCategory {
			cp := categories[cat]
			cp.QuestionsAsked += asked
			cp.CorrectAnswers += p.Stats.CorrectByCategory[cat]
			categories[cat] = cp
		}
	}
	for cat, cp := range categories {
		cp.Accuracy = round3(float64(cp.CorrectAnswers) / float64(cp.QuestionsAsked))
		categories[cat] = cp
	}

	individualMetrics := protocol.IndividualPhaseMetrics{
		PreSolvedPiecesUsed: e.puzzle.effects.AnchorPreSolved * e.puzzle.playersAtPuzzleStart,
	}
	if completed > 0 {
		individualMetrics.AverageSolveTime = round2(solveSum / float64(completed))
		individualMetrics.FastestCompletion = fastest
		individualMetrics.SlowestCompletion = slowest
	}

	collaborative := protocol.CollaborativePhaseMetrics{
		TotalMoves:              totalMoves,
		SuccessfulMoves:         successfulMoves,
		TotalRecommendations:    totalRecs,
		AcceptedRecommendations: acceptedRecs,
	}
	if totalMoves > 0 {
		collaborative.MoveAccuracy = round3(float64(successfulMoves) / float64(totalMoves))
	}
	if totalRecs > 0 {
		collaborative.RecommendationAcceptanceRate = round3(float64(acceptedRecs) / float64(totalRecs))
	}

	totalTime := e.puzzle.totalTime.Seconds()
	completionRate := 1.0
	if !success {
		g2 := e.puzzle.gridSize * e.puzzle.gridSize
		completionRate = round3(float64(e.correctFragments()) / float64(g2))
	}

	e.analytics.hostReport = protocol.HostCompleteReport{
		GameSuccess:    success,
		TotalGameTime:  totalGameTime,
		TotalPlayers:   len(e.players),
		DifficultyMode: e.cfg.DifficultyMode,
		OverallPerformance: protocol.OverallPerformance{
			TotalScore:     totalScore,
			AverageScore:   round2(float64(totalScore) / float64(max(len(e.players), 1))),
			CompletionRate: completionRate,
		},
		ResourceGatheringAnalytics: protocol.ResourceGatheringAnalytics{
			TotalRounds:       rounds,
			QuestionsAnswered: slots,
			OverallAccuracy:   overallAccuracy,
			TokenDistribution: e.tokens,
			PlayerPerformance: resourcePerf,
		},
		PuzzleAssemblyAnalytics: protocol.PuzzleAssemblyAnalytics{
			TotalTime:                 totalTime,
			CompletionTime:            e.analytics.completionSeconds,
			TimeUtilization:           round2(e.analytics.completionSeconds / math.Max(totalTime, 1e-9)),
			IndividualPhaseMetrics:    individualMetrics,
			CollaborativePhaseMetrics: collaborative,
			PlayerContributions:       contributions,
		},
		CategoryPerformance: categories,
		TimelineAnalysis:    timeline,
	}
}

// buildTimeline derives phase durations from the recorded boundaries; they
// sum to totalGameTime by construction.
func (e *Engine) buildTimeline() protocol.TimelineAnalysis {
	return protocol.TimelineAnalysis{
		SetupPhase:       round2(e.resourceStartedAt.Sub(e.setupStartedAt).Seconds()),
		ResourcePhase:    round2(e.prepStartedAt.Sub(e.resourceStartedAt).Seconds()),
		PreparationPhase: round2(e.puzzle.startTime.Sub(e.prepStartedAt).Seconds()),
		PuzzlePhase:      round2(e.analytics.endedAt.Sub(e.puzzle.startTime).Seconds()),
	}
}

// replayPlayerAnalyticsState restores a reconnecting player's reports.
func (e *Engine) replayPlayerAnalyticsState(p *Player) {
	p.send(protocol.AnalyticsToPlayerPersonalReport, e.analytics.personalReports[p.ID])
	p.send(protocol.AnalyticsToClientTeamSummary, e.analytics.teamSummary)
}

// ── Reset ──────────────────────────────────────────────────────────────────

// handleResetGame processes ANALYTICS_TO_SERVER_RESET_GAME: valid only
// during analytics; there is deliberately no mid-game abort.
func (e *Engine) handleResetGame() {
	if e.phase != protocol.PhaseAnalytics {
		e.sendHostError(protocol.ErrorTypeGameState, protocol.ErrForbiddenPhase,
			"reset is only valid during the analytics phase",
			"a stalled game is recovered by a deployment restart, not a mid-game reset")
		return
	}

	reset := protocol.GameReset{
		Reason:                "host_initiated_reset",
		ReconnectRequired:     true,
		ReconnectInstructions: "Refresh your browser and reconnect to join the next game",
		NewGameAvailable:      true,
	}
	e.broadcastAll(protocol.AnalyticsToClientGameReset, reset)

	// Invalidate every player token and close their sockets with 1000 (the
	// close-code table's normal closure for game reset — no auto-reconnect).
	for _, p := range e.players {
		if p.client != nil {
			p.client.CloseWithCode(protocol.CloseNormal)
		}
	}

	// Fresh game state. The host UUID persists (same server process) and
	// the host socket stays attached.
	e.players = map[string]*Player{}
	e.joinOrder = nil
	e.tokens = protocol.TeamTokens{}
	e.resource = resourceState{}
	e.puzzle = puzzleState{}
	e.analytics = analyticsState{}
	e.phase = protocol.PhaseSetup
	e.setupStartedAt = time.Now()
	e.resetOccurred = true
	e.lastRolesSig = ""

	e.sendHost(protocol.SetupToHostPlayerRoster, e.buildRoster())
}

// round3 rounds to three decimals (rate fields such as moveAccuracy and
// recommendationAcceptanceRate).
func round3(x float64) float64 {
	return math.Round(x*1000) / 1000
}
