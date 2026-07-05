package integration

import (
	"strings"
	"testing"
	"time"

	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/config"
	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/protocol"
)

// noSpecialty makes question selection deterministic for token-math tests.
func noSpecialty(c *config.Config) { c.MediumSpecialtyProbability = 0 }

func TestResourcePhaseStartAndSilentWait(t *testing.T) {
	t.Parallel()
	h := Start(t, noSpecialty, nil)
	host, players := JoinConfigured(t, h, 2)
	host.StartGame()

	start := payloadAs[protocol.ResourcePhaseStart](t, players[0].Expect(protocol.ResourceToClientPhaseStart))
	if start.Phase != protocol.PhaseResourceGathering || start.TotalRounds != 2 {
		t.Errorf("phase start = %+v", start)
	}
	if start.RoundDuration != 0.2 || start.AnswerTime != 0.15 || start.GraceTime != 0.05 {
		t.Errorf("durations = %v/%v/%v", start.RoundDuration, start.AnswerTime, start.GraceTime)
	}
	if start.TokenThresholds != (protocol.ThresholdSet{Anchor: 25, Chronos: 20, Guide: 15, Clarity: 30}) {
		t.Errorf("thresholds = %+v", start.TokenThresholds)
	}
	if start.DifficultySettings.Mode != "medium" || start.DifficultySettings.TimeMultiplier != 1.0 {
		t.Errorf("difficulty = %+v", start.DifficultySettings)
	}

	hostStart := payloadAs[protocol.HostResourcePhaseStart](t, host.Expect(protocol.ResourceToHostPhaseStart))
	if hostStart.MonitoringDashboard.CurrentRound != 0 {
		t.Errorf("dashboard round = %d, want 0 during the wait", hostStart.MonitoringDashboard.CurrentRound)
	}
	if hostStart.MonitoringDashboard.PlayerDistribution["unknown"] != 2 {
		t.Errorf("distribution = %v", hostStart.MonitoringDashboard.PlayerDistribution)
	}

	// The silent round-length wait (200ms here): no question inside it.
	players[0].ExpectNone(protocol.ResourceToPlayerTriviaQuestion, 120*time.Millisecond)
	q := players[0].ExpectQuestion()
	if q.RoundNumber != 1 || q.TotalRounds != 2 || q.IsSpecialty {
		t.Errorf("round 1 question = %+v", q)
	}
	if q.Difficulty != "medium" {
		t.Errorf("difficulty = %s, want base medium", q.Difficulty)
	}
	if strings.Contains(q.QuestionText, "&quot;") || strings.Contains(q.QuestionText, "&amp;") {
		t.Errorf("undecoded HTML entities in question: %q", q.QuestionText)
	}
}

// Full two-round token-economy test: role multipliers, unknown station,
// station-on-record at window close, thresholds, and both PHASE_COMPLETE
// payloads.
func TestTokenEconomyAcrossRounds(t *testing.T) {
	t.Parallel()
	h := Start(t, noSpecialty, nil)
	host, players := JoinConfigured(t, h, 2)
	a, b := players[0], players[1] // a: art_enthusiast (clarity), b: detective (guide)
	host.StartGame()

	// During the silent wait, b scans its bonus station; a stays unknown.
	b.Expect(protocol.ResourceToClientPhaseStart)
	if got := b.Scan("hash-guide"); got != "guide" {
		t.Fatalf("scan → %s, want guide", got)
	}

	// Round 1: both answer correctly.
	qa, qb := a.ExpectQuestion(), b.ExpectQuestion()
	a.Answer(qa, true)
	b.Answer(qb, true)

	ra := a.ExpectAnswerResult()
	if !ra.Correct || ra.TokensEarned != 0 || ra.CurrentLocation != "unknown" {
		t.Errorf("unknown-station result = %+v", ra)
	}
	rb := b.ExpectAnswerResult()
	if !rb.Correct || rb.TokensEarned != 30 || rb.BaseTokens != 20 {
		t.Errorf("guide-station result = %+v", rb)
	}
	if !rb.Bonuses.RoleBonus || rb.Bonuses.RoleBonusTokens != 10 || rb.Bonuses.SpecialtyBonus {
		t.Errorf("bonuses = %+v", rb.Bonuses)
	}
	if rb.CurrentLocation != "guide" {
		t.Errorf("currentLocation = %s", rb.CurrentLocation)
	}

	progress := payloadAs[protocol.TeamProgress](t, a.Expect(protocol.ResourceToClientTeamProgress))
	if progress.TeamTokens.GuideTokens != 30 || progress.TeamTokens.ClarityTokens != 0 {
		t.Errorf("team tokens = %+v", progress.TeamTokens)
	}
	if progress.QuestionsAnswered != 2 || progress.TotalQuestions != 4 {
		t.Errorf("progress counts = %+v", progress)
	}
	if progress.TeamPerformance.AverageAccuracy != 1.0 {
		t.Errorf("accuracy = %v", progress.TeamPerformance.AverageAccuracy)
	}
	if progress.CurrentThresholds.Guide != 2 { // floor(30/15) = 2
		t.Errorf("guide thresholds = %d", progress.CurrentThresholds.Guide)
	}

	analytics := payloadAs[protocol.RoundAnalytics](t, host.Expect(protocol.ResourceToHostRoundAnalytics))
	if analytics.CurrentRound != 1 || analytics.RoundResults.QuestionsDelivered != 2 ||
		analytics.RoundResults.AnswersReceived != 2 || analytics.RoundResults.CorrectAnswers != 2 ||
		analytics.RoundResults.TokensAwarded != 30 {
		t.Errorf("round analytics = %+v", analytics.RoundResults)
	}
	if analytics.PlayerPerformance[b.ID].Location != "guide" || analytics.PlayerPerformance[b.ID].TokensEarned != 30 {
		t.Errorf("player performance = %+v", analytics.PlayerPerformance[b.ID])
	}
	if analytics.StationDistribution["guide"] != 1 || analytics.StationDistribution["unknown"] != 1 {
		t.Errorf("station distribution = %v", analytics.StationDistribution)
	}

	// Round 2: a scans clarity mid-window — the award goes to the station on
	// record at window close. Both answer correctly again.
	qa2, qb2 := a.ExpectQuestion(), b.ExpectQuestion()
	if got := a.Scan("hash-clarity"); got != "clarity" {
		t.Fatalf("scan → %s", got)
	}
	a.Answer(qa2, true)
	b.Answer(qb2, true)

	ra2 := a.ExpectAnswerResult()
	if ra2.TokensEarned != 30 || ra2.CurrentLocation != "clarity" || !ra2.Bonuses.RoleBonus {
		t.Errorf("mid-window scan result = %+v", ra2)
	}

	// Phase completes after the final grace period.
	complete := payloadAs[protocol.ResourcePhaseComplete](t, a.Expect(protocol.ResourceToClientPhaseComplete))
	if complete.NextPhase != protocol.PhasePuzzlePreparation {
		t.Errorf("nextPhase = %s", complete.NextPhase)
	}
	if complete.FinalTokenTotals.GuideTokens != 60 || complete.FinalTokenTotals.ClarityTokens != 30 {
		t.Errorf("final tokens = %+v", complete.FinalTokenTotals)
	}
	want := protocol.ThresholdSet{Guide: 4, Clarity: 1} // floor(60/15), floor(30/30)
	if complete.ThresholdAchievements != want {
		t.Errorf("thresholds = %+v, want %+v", complete.ThresholdAchievements, want)
	}
	// Grid 3×3, guide N=4 of max 6 → ceil(9×(1−4/6)) = 3 highlights;
	// clarity N=1 → 0.3 + 1×0.1 = 0.4s preview; no anchor/chronos effects.
	be := complete.BonusEffects
	if be.GuideHighlightCount != 3 || be.ClarityPreviewDuration != 0.4 ||
		be.AnchorPreSolved != 0 || be.ChronosTimeBonus != 0 {
		t.Errorf("bonusEffects = %+v", be)
	}

	hostComplete := payloadAs[protocol.HostResourcePhaseComplete](t, host.Expect(protocol.ResourceToHostPhaseComplete))
	if hostComplete.TotalQuestionsAnswered != 4 || hostComplete.TeamPerformance.OverallAccuracy != 1.0 ||
		hostComplete.TeamPerformance.TotalTokensEarned != 90 {
		t.Errorf("host phase complete = %+v", hostComplete.TeamPerformance)
	}
	pa := hostComplete.PlayerAnalytics[a.ID]
	if pa.QuestionsAnswered != 2 || pa.CorrectAnswers != 2 || pa.Accuracy != 1.0 || pa.TokensEarned != 30 {
		t.Errorf("player analytics = %+v", pa)
	}
	if !hostComplete.ReadyForPuzzlePhase {
		t.Error("readyForPuzzlePhase must be true")
	}
}

// With specialtyProbability forced to 1.0, every question is a specialty:
// one difficulty above base (capped hard), drawn from the player's own
// specialties, and worth the specialty multiplier.
func TestSpecialtyQuestionsForced(t *testing.T) {
	t.Parallel()
	h := Start(t, func(c *config.Config) { c.MediumSpecialtyProbability = 1.0 }, nil)
	host, players := JoinConfigured(t, h, 2)
	a := players[0]
	host.StartGame()

	a.Expect(protocol.ResourceToClientPhaseStart)
	a.Scan("hash-clarity") // a is art_enthusiast → clarity is the bonus station

	q := a.ExpectQuestion()
	if !q.IsSpecialty {
		t.Fatal("expected a specialty question at probability 1.0")
	}
	if q.Difficulty != "hard" {
		t.Errorf("difficulty = %s, want hard (medium+1)", q.Difficulty)
	}
	if q.Category != fixtureCategories[0] {
		t.Errorf("category = %s, want the player's specialty %s", q.Category, fixtureCategories[0])
	}

	a.Answer(q, true)
	r := a.ExpectAnswerResult()
	// 20 × 1.5 (role at station) × 2.0 (specialty) = 60
	if r.TokensEarned != 60 {
		t.Errorf("tokensEarned = %d, want 60", r.TokensEarned)
	}
	if !r.Bonuses.SpecialtyBonus || r.Bonuses.SpecialtyBonusTokens != 30 || r.Bonuses.RoleBonusTokens != 10 {
		t.Errorf("bonuses = %+v", r.Bonuses)
	}
}

func TestInvalidStationHash(t *testing.T) {
	t.Parallel()
	h := Start(t, noSpecialty, nil)
	host, players := JoinConfigured(t, h, 2)
	a := players[0]
	host.StartGame()

	a.Expect(protocol.ResourceToClientPhaseStart)
	a.Send(string(protocol.ResourceToServerLocationVerified), a.ID, protocol.LocationVerified{
		StationHash:   "not-a-real-station",
		ScanTimestamp: nowStamp(),
	})
	a.ExpectError(protocol.SystemToClientError, protocol.ErrInvalidStationHash)

	// Station unchanged: the next answer result reports unknown and earns 0.
	q := a.ExpectQuestion()
	a.Answer(q, true)
	r := a.ExpectAnswerResult()
	if r.CurrentLocation != "unknown" || r.TokensEarned != 0 {
		t.Errorf("result after bad scan = %+v", r)
	}
}

func TestAnswerResubmissionLastCounts(t *testing.T) {
	t.Parallel()
	h := Start(t, noSpecialty, nil)
	host, players := JoinConfigured(t, h, 2)
	a := players[0]
	host.StartGame()

	q := a.ExpectQuestion()
	a.Answer(q, false)
	a.Answer(q, true) // resubmission before the deadline overwrites
	r := a.ExpectAnswerResult()
	if !r.Correct {
		t.Errorf("resubmitted correct answer marked wrong: %+v", r)
	}
}

func TestUnansweredAndLateAnswers(t *testing.T) {
	t.Parallel()
	h := Start(t, noSpecialty, nil)
	host, players := JoinConfigured(t, h, 2)
	a := players[0]
	host.StartGame()

	q := a.ExpectQuestion() // never answered
	r := a.ExpectAnswerResult()
	if r.Correct || r.SelectedAnswer != nil {
		t.Errorf("unanswered result = %+v", r)
	}

	// A late answer for the closed question is silently ignored.
	a.Send(string(protocol.ResourceToServerTriviaAnswer), a.ID, protocol.TriviaAnswer{
		QuestionID:  q.QuestionID,
		AnswerIndex: 0,
		TimeElapsed: 0.2,
	})
	a.ExpectNone(protocol.SystemToClientError, 100*time.Millisecond)
}

// Disconnected players receive no questions; their round slots count as
// incorrect in team accuracy, and they keep receiving phase events after
// reconnecting.
func TestDisconnectedPlayerAccounting(t *testing.T) {
	t.Parallel()
	h := Start(t, func(c *config.Config) {
		noSpecialty(c)
		c.MinPlayers = 3
	}, nil)
	host, players := JoinConfigured(t, h, 3)
	a, b, c := players[0], players[1], players[2]
	host.StartGame()

	a.Expect(protocol.ResourceToClientPhaseStart)
	c.Close() // gone before round 1
	host.Expect(protocol.SystemToHostPlayerDisconnected)

	qa, qb := a.ExpectQuestion(), b.ExpectQuestion()
	a.Answer(qa, true)
	b.Answer(qb, true)

	progress := payloadAs[protocol.TeamProgress](t, a.Expect(protocol.ResourceToClientTeamProgress))
	if progress.TotalQuestions != 6 { // 3 players × 2 rounds, disconnected included
		t.Errorf("totalQuestions = %d, want 6", progress.TotalQuestions)
	}
	if progress.TeamPerformance.AverageAccuracy != 0.67 { // 2 correct / 3 slots
		t.Errorf("accuracy = %v, want 0.67", progress.TeamPerformance.AverageAccuracy)
	}

	analytics := payloadAs[protocol.RoundAnalytics](t, host.Expect(protocol.ResourceToHostRoundAnalytics))
	if analytics.RoundResults.QuestionsDelivered != 2 {
		t.Errorf("questionsDelivered = %d, want 2 (disconnected player skipped)", analytics.RoundResults.QuestionsDelivered)
	}
	row, ok := analytics.PlayerPerformance[c.ID]
	if !ok || row.AnswerCorrect {
		t.Errorf("disconnected player's row = %+v ok=%t", row, ok)
	}

	// Reconnect mid-phase: phase context replays; no question was delivered
	// this round, so none is replayed.
	c2, confirmed := ReconnectPlayer(t, h, c.ID)
	if !confirmed.IsReconnection || confirmed.CurrentPhase != protocol.PhaseResourceGathering {
		t.Fatalf("reconnect handshake = %+v", confirmed)
	}
	c2.Expect(protocol.ResourceToClientPhaseStart)
	c2.Expect(protocol.ResourceToClientTeamProgress)

	// The reconnected player is back in the loop for phase completion.
	c2.Expect(protocol.ResourceToClientPhaseComplete)
}

// A player who received a question, dropped, and reconnected mid-round gets
// the same question replayed with its original deadline and can still answer.
func TestReconnectMidRoundReplaysQuestion(t *testing.T) {
	t.Parallel()
	h := Start(t, func(c *config.Config) {
		noSpecialty(c)
		c.TriviaAnswerTime = config.Seconds(time.Second) // roomy answer window
	}, nil)
	host, players := JoinConfigured(t, h, 2)
	a := players[0]
	host.StartGame()

	q := a.ExpectQuestion()
	a.Close()
	host.Expect(protocol.SystemToHostPlayerDisconnected)

	a2, _ := ReconnectPlayer(t, h, a.ID)
	a2.Expect(protocol.ResourceToClientPhaseStart)
	a2.Expect(protocol.ResourceToClientTeamProgress)
	replay := a2.ExpectQuestion()
	if replay.QuestionID != q.QuestionID {
		t.Errorf("replayed question = %s, want %s", replay.QuestionID, q.QuestionID)
	}
	if replay.AnswerDeadline != q.AnswerDeadline {
		t.Errorf("replayed deadline = %s, want original %s", replay.AnswerDeadline, q.AnswerDeadline)
	}

	a2.Answer(replay, true)
	r := a2.ExpectAnswerResult()
	if !r.Correct {
		t.Errorf("answer after reconnect = %+v", r)
	}
}

// Host reconnection mid-phase replays the dashboard and the latest round
// analytics.
func TestHostReconnectDuringResource(t *testing.T) {
	t.Parallel()
	h := Start(t, func(c *config.Config) {
		noSpecialty(c)
		c.ResourceGatheringRounds = 4
	}, nil)
	host, players := JoinConfigured(t, h, 2)
	a, b := players[0], players[1]
	host.StartGame()

	// Play round 1 so lastAnalytics exists.
	qa, qb := a.ExpectQuestion(), b.ExpectQuestion()
	a.Answer(qa, true)
	b.Answer(qb, true)
	host.Expect(protocol.ResourceToHostRoundAnalytics)

	host.Close()
	a.Expect(protocol.SystemToClientHostDisconnected)

	host2, confirmed := ConnectHost(t, h)
	if !confirmed.IsReconnection || confirmed.CurrentPhase != protocol.PhaseResourceGathering {
		t.Fatalf("host handshake = %+v", confirmed)
	}
	replay := payloadAs[protocol.HostResourcePhaseStart](t, host2.Expect(protocol.ResourceToHostPhaseStart))
	if replay.MonitoringDashboard.CurrentRound < 1 {
		t.Errorf("dashboard round = %d, want >= 1", replay.MonitoringDashboard.CurrentRound)
	}
	analytics := payloadAs[protocol.RoundAnalytics](t, host2.Expect(protocol.ResourceToHostRoundAnalytics))
	if analytics.CurrentRound < 1 {
		t.Errorf("replayed analytics = %+v", analytics)
	}
	a.Expect(protocol.SystemToClientHostReconnected)
}
