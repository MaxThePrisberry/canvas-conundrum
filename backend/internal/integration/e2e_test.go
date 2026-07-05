package integration

import (
	"testing"
	"time"

	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/config"
	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/protocol"
)

// TestE2EFullGameWin plays an entire winning game with four real clients:
// setup → resource gathering (scans, mixed answers) → preparation →
// assembly (completions, moves, swaps, recommendations) → analytics.
func TestE2EFullGameWin(t *testing.T) {
	t.Parallel()
	h := Start(t, func(c *config.Config) {
		noSpecialty(c)
		c.MinPlayers = 4
		c.ResourceGatheringRounds = 3
		c.PuzzleBaseTime = config.Seconds(30 * time.Second)
	}, nil)
	host, players := JoinConfigured(t, h, 4)
	started := host.StartGame()
	if started.TotalPlayers != 4 {
		t.Fatalf("game started = %+v", started)
	}

	// Every player scans their role's bonus station during the silent wait.
	stations := []string{"hash-clarity", "hash-guide", "hash-chronos", "hash-anchor"}
	for i, p := range players {
		p.Expect(protocol.ResourceToClientPhaseStart)
		p.Scan(stations[i])
	}

	// Three rounds; player 3 always answers wrong, the rest correctly.
	for round := 1; round <= 3; round++ {
		for i, p := range players {
			q := p.ExpectQuestion()
			if q.RoundNumber != round {
				t.Fatalf("round %d question = %+v", round, q)
			}
			p.Answer(q, i != 3)
		}
	}

	complete := payloadAs[protocol.ResourcePhaseComplete](t,
		players[0].Expect(protocol.ResourceToClientPhaseComplete))
	// 3 correct answers × 30 tokens at bonus stations for players 0-2.
	if complete.FinalTokenTotals.ClarityTokens != 90 || complete.FinalTokenTotals.GuideTokens != 90 ||
		complete.FinalTokenTotals.ChronosTokens != 90 || complete.FinalTokenTotals.AnchorTokens != 0 {
		t.Fatalf("token totals = %+v", complete.FinalTokenTotals)
	}
	// chronos floor(90/20)=4 thresholds → +2s; clarity floor(90/30)=3 → window.
	if complete.ThresholdAchievements.Chronos != 4 || complete.ThresholdAchievements.Clarity != 3 {
		t.Fatalf("thresholds = %+v", complete.ThresholdAchievements)
	}

	host.Expect(protocol.PuzzleToHostPreparing)
	host.Expect(protocol.PuzzleToHostReady)
	hostStart := host.StartPuzzle()
	if hostStart.TotalTime != 32 { // (30 + 4×0.5) × 1.0
		t.Fatalf("totalTime = %v, want 32", hostStart.TotalTime)
	}

	// Everyone fetches their tile and completes their individual puzzle.
	for i, p := range players {
		load := payloadAs[protocol.PuzzlePhaseLoad](t, p.Expect(protocol.PuzzleToClientPhaseLoad))
		resp, body := fetchAsset(t, h, "/api/segments/"+load.AssignedSegmentID, p.ID)
		assertPNG(t, resp, body, 32)
		p.Expect(protocol.PuzzleToClientPhaseStart)
		p.CompleteSegment(load.AssignedSegmentID, float64(i+1))
	}

	success := AssembleToVictory(t, players)
	if !success.Success || success.FinalGridState.CorrectFragments != 9 {
		t.Fatalf("success = %+v", success)
	}

	// Analytics: every player gets a personal report and the summary.
	for i, p := range players {
		report := payloadAs[protocol.PersonalReport](t, p.Expect(protocol.AnalyticsToPlayerPersonalReport))
		if !report.GameSuccess || report.ScoreBreakdown.CompletionBonus != 100 {
			t.Errorf("player %d report = %+v", i, report.ScoreBreakdown)
		}
		wantTrivia := 30 // 3 correct
		if i == 3 {
			wantTrivia = 0
		}
		if report.ScoreBreakdown.TriviaPoints != wantTrivia {
			t.Errorf("player %d triviaPoints = %d, want %d", i, report.ScoreBreakdown.TriviaPoints, wantTrivia)
		}
		summary := payloadAs[protocol.TeamSummary](t, p.Expect(protocol.AnalyticsToClientTeamSummary))
		if !summary.GameSuccess || len(summary.Leaderboard) != 4 {
			t.Errorf("summary = %+v", summary)
		}
	}
	report := payloadAs[protocol.HostCompleteReport](t, host.Expect(protocol.AnalyticsToHostCompleteReport))
	if !report.GameSuccess || report.PuzzleAssemblyAnalytics.TotalTime != 32 {
		t.Errorf("host report = %+v", report.PuzzleAssemblyAnalytics)
	}
}

// TestE2EFullGameLoss plays a losing game end-to-end: partial completions,
// the timer expires, and analytics reports the failure without bonuses.
func TestE2EFullGameLoss(t *testing.T) {
	t.Parallel()
	h := Start(t, func(c *config.Config) {
		noSpecialty(c)
		c.MinPlayers = 4
		c.PuzzleBaseTime = config.Seconds(800 * time.Millisecond)
	}, nil)
	host, players := ReachAssembly(t, h, 4, 2, map[int]string{1: "hash-guide"})

	// Two of four complete; time runs out.
	players[0].CompleteSegment("segment_a1", 1)
	players[1].CompleteSegment("segment_a2", 2)

	timeout := payloadAs[protocol.CompletedTimeout](t, players[0].Expect(protocol.PuzzleToClientCompletedTimeout))
	if timeout.Success || timeout.FinalStats.TotalFragments != 9 {
		t.Fatalf("timeout = %+v", timeout)
	}
	// k=2 of 4 → ceil(18/4) = 5 fragments were on the grid.
	if timeout.FinalStats.FragmentsPlaced != 5 {
		t.Errorf("fragmentsPlaced = %d, want 5", timeout.FinalStats.FragmentsPlaced)
	}

	for _, p := range players {
		report := payloadAs[protocol.PersonalReport](t, p.Expect(protocol.AnalyticsToPlayerPersonalReport))
		if report.GameSuccess || report.ScoreBreakdown.CompletionBonus != 0 {
			t.Errorf("loss report = %+v", report.ScoreBreakdown)
		}
	}
	summary := payloadAs[protocol.TeamSummary](t, players[0].Expect(protocol.AnalyticsToClientTeamSummary))
	if summary.GameSuccess {
		t.Error("summary must report failure")
	}
	hostReport := payloadAs[protocol.HostCompleteReport](t, host.Expect(protocol.AnalyticsToHostCompleteReport))
	if hostReport.GameSuccess || hostReport.OverallPerformance.CompletionRate >= 1.0 {
		t.Errorf("host loss report = %+v", hostReport.OverallPerformance)
	}
}
