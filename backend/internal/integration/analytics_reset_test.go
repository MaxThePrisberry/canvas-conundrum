package integration

import (
	"net/http"
	"testing"
	"time"

	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/config"
	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/protocol"
)

func TestAnalyticsReportsAfterVictory(t *testing.T) {
	t.Parallel()
	h := Start(t, func(c *config.Config) {
		noSpecialty(c)
		c.PuzzleBaseTime = config.Seconds(30 * time.Second)
	}, nil)
	host, players := JoinConfigured(t, h, 2)
	a, b := players[0], players[1] // b: detective (guide bonus)
	host.StartGame()

	// b farms guide (2 × 30 = 60 tokens); a stays unknown and earns none.
	PlayResource(t, host, players, 2, map[int]string{1: "hash-guide"})
	host.Expect(protocol.PuzzleToHostReady)
	host.StartPuzzle()
	for _, p := range players {
		p.Expect(protocol.PuzzleToClientPhaseStart)
	}
	a.CompleteSegment("segment_a1", 1.5)
	b.CompleteSegment("segment_a2", 2.5)
	AssembleToVictory(t, players)

	// ── Personal report (b) ────────────────────────────────────────────
	reportB := payloadAs[protocol.PersonalReport](t, b.Expect(protocol.AnalyticsToPlayerPersonalReport))
	if !reportB.GameSuccess || reportB.PlayerID != b.ID || reportB.PlayerName != "Bob" {
		t.Errorf("report identity = %+v", reportB)
	}
	if reportB.TokenCollection.GuideTokens != 60 || reportB.TokenCollection.TotalTokens != 60 {
		t.Errorf("token collection = %+v", reportB.TokenCollection)
	}
	tp := reportB.TriviaPerformance
	if tp.TotalQuestions != 2 || tp.CorrectAnswers != 2 || tp.Accuracy != 1.0 {
		t.Errorf("trivia performance = %+v", tp)
	}
	psm := reportB.PuzzleSolvingMetrics
	if psm.IndividualSolveTime != 2.5 || psm.IndividualRank != 2 {
		t.Errorf("puzzle metrics = %+v", psm)
	}
	sb := reportB.ScoreBreakdown
	if sb.TriviaPoints != 20 || sb.CompletionBonus != 100 {
		t.Errorf("score breakdown = %+v", sb)
	}
	wantMove := psm.SuccessfulMoves * 5
	wantRec := psm.RecommendationsSent*3 + psm.RecommendationsAccepted*8
	if sb.MovePoints != wantMove || sb.RecommendationPoints != wantRec {
		t.Errorf("score terms = %+v (moves %d, recs %d/%d)", sb,
			psm.SuccessfulMoves, psm.RecommendationsSent, psm.RecommendationsAccepted)
	}
	if sb.TotalScore != sb.TriviaPoints+sb.SpecialtyPoints+sb.CompletionBonus+sb.MovePoints+sb.RecommendationPoints {
		t.Errorf("total mismatch: %+v", sb)
	}
	if reportB.PersonalScore != sb.TotalScore {
		t.Errorf("personalScore %d != breakdown total %d", reportB.PersonalScore, sb.TotalScore)
	}

	reportA := payloadAs[protocol.PersonalReport](t, a.Expect(protocol.AnalyticsToPlayerPersonalReport))
	if reportA.PuzzleSolvingMetrics.IndividualRank != 1 {
		t.Errorf("a's individual rank = %d, want 1", reportA.PuzzleSolvingMetrics.IndividualRank)
	}

	// ── Team summary ───────────────────────────────────────────────────
	summary := payloadAs[protocol.TeamSummary](t, a.Expect(protocol.AnalyticsToClientTeamSummary))
	if !summary.GameSuccess || summary.TotalPlayers != 2 {
		t.Errorf("summary = %+v", summary)
	}
	if summary.TotalScore != reportA.PersonalScore+reportB.PersonalScore {
		t.Errorf("totalScore %d != %d + %d", summary.TotalScore, reportA.PersonalScore, reportB.PersonalScore)
	}
	if summary.TeamPerformance.ThresholdAchievements.Guide != 4 ||
		summary.TeamPerformance.TotalTokensCollected != 60 {
		t.Errorf("team performance = %+v", summary.TeamPerformance)
	}
	if len(summary.Leaderboard) != 2 || summary.Leaderboard[0].TotalScore < summary.Leaderboard[1].TotalScore {
		t.Errorf("leaderboard = %+v", summary.Leaderboard)
	}
	for _, entry := range summary.Leaderboard {
		want := reportA
		if entry.PlayerID == b.ID {
			want = reportB
		}
		if entry.Rank != want.Rank {
			t.Errorf("rank mismatch: leaderboard %+v vs report %d", entry, want.Rank)
		}
	}

	// ── Host complete report ───────────────────────────────────────────
	report := payloadAs[protocol.HostCompleteReport](t, host.Expect(protocol.AnalyticsToHostCompleteReport))
	if !report.GameSuccess || report.DifficultyMode != "medium" || report.TotalPlayers != 2 {
		t.Errorf("host report = %+v", report)
	}
	if report.OverallPerformance.TotalScore != summary.TotalScore ||
		report.OverallPerformance.CompletionRate != 1.0 {
		t.Errorf("overall performance = %+v", report.OverallPerformance)
	}
	rga := report.ResourceGatheringAnalytics
	if rga.QuestionsAnswered != 4 || rga.TokenDistribution.GuideTokens != 60 {
		t.Errorf("resource analytics = %+v", rga)
	}
	if rga.PlayerPerformance[b.ID].StationPreferences["guide"] != 1 {
		t.Errorf("station preferences = %+v", rga.PlayerPerformance[b.ID])
	}
	paa := report.PuzzleAssemblyAnalytics
	if paa.TotalTime != 30 || paa.CompletionTime <= 0 {
		t.Errorf("assembly analytics = %+v", paa)
	}
	if paa.TimeUtilization != round2t(paa.CompletionTime/paa.TotalTime) {
		t.Errorf("timeUtilization = %v for completion %v", paa.TimeUtilization, paa.CompletionTime)
	}
	// Category totals cover every delivered question.
	asked := 0
	for _, cp := range report.CategoryPerformance {
		asked += cp.QuestionsAsked
	}
	if asked != 4 {
		t.Errorf("category questionsAsked sum = %d, want 4", asked)
	}
	// Timeline sums to total game time.
	tl := report.TimelineAnalysis
	sum := tl.SetupPhase + tl.ResourcePhase + tl.PreparationPhase + tl.PuzzlePhase
	if diff := sum - report.TotalGameTime; diff > 0.02 || diff < -0.02 {
		t.Errorf("timeline sum %v != totalGameTime %v", sum, report.TotalGameTime)
	}

	// ── Analytics-phase reconnects replay the reports ──────────────────
	a.Close()
	host.Expect(protocol.SystemToHostPlayerDisconnected)
	a2, confirmed := ReconnectPlayer(t, h, a.ID)
	if confirmed.CurrentPhase != protocol.PhaseAnalytics {
		t.Fatalf("phase = %s", confirmed.CurrentPhase)
	}
	replayed := payloadAs[protocol.PersonalReport](t, a2.Expect(protocol.AnalyticsToPlayerPersonalReport))
	if replayed.PersonalScore != reportA.PersonalScore {
		t.Errorf("replayed report differs: %d vs %d", replayed.PersonalScore, reportA.PersonalScore)
	}
	a2.Expect(protocol.AnalyticsToClientTeamSummary)

	host.Close()
	host2, _ := ConnectHost(t, h)
	host2.Expect(protocol.AnalyticsToHostCompleteReport)
}

func round2t(x float64) float64 {
	return float64(int(x*100+0.5)) / 100
}

// Equal scores share a rank (competition ranking 1,2,2,4), alphabetical
// within the tie.
func TestLeaderboardTieRanking(t *testing.T) {
	t.Parallel()
	h := Start(t, func(c *config.Config) {
		noSpecialty(c)
		c.MinPlayers = 3
		c.PuzzleBaseTime = config.Seconds(500 * time.Millisecond)
	}, nil)
	host, players := JoinConfigured(t, h, 3)
	a, b, c := players[0], players[1], players[2]
	host.StartGame()

	// Alice and Bob answer both rounds correctly (20 points each); Charlie
	// answers wrong (0). Nobody solves the puzzle → timeout, no bonus.
	for round := 0; round < 2; round++ {
		qa, qb, qc := a.ExpectQuestion(), b.ExpectQuestion(), c.ExpectQuestion()
		a.Answer(qa, true)
		b.Answer(qb, true)
		c.Answer(qc, false)
	}
	host.Expect(protocol.PuzzleToHostReady)
	host.StartPuzzle()

	summary := payloadAs[protocol.TeamSummary](t, a.Expect(protocol.AnalyticsToClientTeamSummary))
	lb := summary.Leaderboard
	if len(lb) != 3 {
		t.Fatalf("leaderboard = %+v", lb)
	}
	if lb[0].PlayerName != "Alice" || lb[0].Rank != 1 || lb[0].TotalScore != 20 {
		t.Errorf("lb[0] = %+v", lb[0])
	}
	if lb[1].PlayerName != "Bob" || lb[1].Rank != 1 || lb[1].TotalScore != 20 {
		t.Errorf("lb[1] = %+v", lb[1])
	}
	if lb[2].PlayerName != "Charlie" || lb[2].Rank != 3 || lb[2].TotalScore != 0 {
		t.Errorf("lb[2] = %+v", lb[2])
	}
	if summary.GameSuccess {
		t.Error("timeout game must not be a success")
	}
}

func TestResetOnlyValidInAnalytics(t *testing.T) {
	t.Parallel()
	h := Start(t, noSpecialty, nil)
	host, _ := JoinConfigured(t, h, 2)
	host.StartGame()

	host.Send(string(protocol.AnalyticsToServerResetGame), host.UUID, protocol.ResetGame{ConfirmReset: true})
	host.ExpectError(protocol.SystemToHostError, protocol.ErrForbiddenPhase)
}

func TestResetClearsGameAndInvalidatesTokens(t *testing.T) {
	t.Parallel()
	h := Start(t, func(c *config.Config) {
		noSpecialty(c)
		c.PuzzleBaseTime = config.Seconds(400 * time.Millisecond)
	}, nil)
	host, players := ReachAssembly(t, h, 2, 2, nil)
	a := players[0]

	// Ride the timeout into analytics.
	a.Expect(protocol.PuzzleToClientCompletedTimeout)
	host.Expect(protocol.AnalyticsToHostCompleteReport)

	// confirmReset:false is structurally invalid.
	host.Send(string(protocol.AnalyticsToServerResetGame), host.UUID, protocol.ResetGame{ConfirmReset: false})
	host.ExpectError(protocol.SystemToHostError, protocol.ErrMalformedPayload)

	host.Send(string(protocol.AnalyticsToServerResetGame), host.UUID, protocol.ResetGame{ConfirmReset: true})

	reset := payloadAs[protocol.GameReset](t, a.Expect(protocol.AnalyticsToClientGameReset))
	if !reset.ReconnectRequired || !reset.NewGameAvailable || reset.Reason != "host_initiated_reset" {
		t.Errorf("reset payload = %+v", reset)
	}
	host.Expect(protocol.AnalyticsToClientGameReset)

	// Player sockets close with 1000 (the no-auto-reconnect code).
	a.ExpectClose(protocol.CloseNormal)

	// Invalidated tokens: reconnection refused with 4001...
	stale := DialPlayer(t, h)
	stale.Send(string(protocol.SetupToServerPlayerConnect), a.ID, struct{}{})
	stale.ExpectClose(protocol.CloseUnauthorized)

	// ...and asset requests are refused too (401 for dead tokens, 404 for
	// the host whose token survives but whose tiles were cleared).
	resp, body := fetchAsset(t, h, "/api/segments/segment_a1", a.ID)
	assertAssetError(t, resp, body, http.StatusUnauthorized, protocol.ErrUnauthorized)
	resp, body = fetchAsset(t, h, "/api/segments/segment_a1", h.HostUUID)
	assertAssetError(t, resp, body, http.StatusNotFound, protocol.ErrNotFound)

	// The host socket survives the reset and a fresh game is joinable.
	fresh := ConnectNewPlayer(t, h)
	fresh.Expect(protocol.SetupToPlayerRolesAvailable)
	fresh.Configure("detective", "Nova")
	fresh2 := ConnectNewPlayer(t, h)
	fresh2.Expect(protocol.SetupToPlayerRolesAvailable)
	fresh2.Configure("janitor", "Orbit")
	for {
		roster := payloadAs[protocol.PlayerRoster](t, host.Expect(protocol.SetupToHostPlayerRoster))
		if roster.ReadyPlayers == 2 && roster.GameStartEligible {
			break
		}
	}
	host.StartGame()
}
