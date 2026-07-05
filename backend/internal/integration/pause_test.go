package integration

import (
	"testing"
	"time"

	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/config"
	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/protocol"
)

// Host disconnect during puzzle assembly pauses everything: the deadline,
// puzzle actions, recommendation timeouts, and fragment cooldowns; host
// reconnection resumes with the deadline extended by the pause duration.
func TestHostDisconnectPausesAssembly(t *testing.T) {
	h := Start(t, func(c *config.Config) {
		noSpecialty(c)
		c.PuzzleBaseTime = config.Seconds(1200 * time.Millisecond)
	}, nil)
	host, players := ReachAssembly(t, h, 2, 2, nil)
	a, b := players[0], players[1]
	a.CompleteSegment("segment_a1", 1)
	b.CompleteSegment("segment_a2", 1)

	// Set up a pending recommendation whose 0.5s timeout must freeze.
	state := a.ExpectGridState()
	for len(state.Fragments) < 9 {
		state = a.ExpectGridState()
	}
	var unassigned, bFrag protocol.GridFragment
	for _, f := range state.Fragments {
		if f.PlayerID == nil && unassigned.SegmentID == "" {
			unassigned = f
		}
		if f.PlayerID != nil && *f.PlayerID == b.ID {
			bFrag = f
		}
	}
	time.Sleep(80 * time.Millisecond)
	a.recommend(b, unassigned.SegmentID, bFrag.SegmentID, "pending across pause")
	rec := payloadAs[protocol.MoveRecommendation](t, b.Expect(protocol.PuzzleToPlayerMoveRecommendation))

	host.Close()

	notice := payloadAs[protocol.HostDisconnected](t, a.Expect(protocol.SystemToClientHostDisconnected))
	if notice.GameImpact.CanContinue {
		t.Error("assembly host disconnect must be canContinue=false")
	}
	if notice.TimerPausedAt == "" {
		t.Error("timerPausedAt must be present when the puzzle timer pauses")
	}
	hasTimer := false
	for _, f := range notice.GameImpact.AffectedFeatures {
		if f == "puzzle_timer" {
			hasTimer = true
		}
	}
	if !hasTimer {
		t.Errorf("affectedFeatures = %v", notice.GameImpact.AffectedFeatures)
	}

	// All puzzle actions are rejected while paused.
	res := a.Move(unassigned.SegmentID, protocol.Position{X: 0, Y: 0}, nil)
	if res.Reason != protocol.MoveRejectPhaseInvalid {
		t.Errorf("move during pause = %+v", res)
	}
	a.Send(string(protocol.PuzzleToServerSegmentCompleted), a.ID, protocol.SegmentCompleted{
		SegmentID: "segment_a1", CompletionTimestamp: nowStamp(), SolveTime: 1,
	})
	a.ExpectError(protocol.SystemToClientError, protocol.ErrForbiddenPhase)
	a.recommend(b, unassigned.SegmentID, bFrag.SegmentID, "paused")
	a.ExpectError(protocol.SystemToClientError, protocol.ErrForbiddenPhase)
	b.Send(string(protocol.PuzzleToServerRecommendationResponse), b.ID, protocol.RecommendationResponse{
		MoveID: rec.MoveID, Response: "accept",
	})
	b.ExpectError(protocol.SystemToClientError, protocol.ErrForbiddenPhase)

	// The pause outlasts both the recommendation timeout (0.5s) and the
	// remaining puzzle time (~1.2s): neither may fire while paused.
	time.Sleep(700 * time.Millisecond)
	a.ExpectNone(protocol.PuzzleToPlayerRecommendationExpired, 50*time.Millisecond)
	a.ExpectNone(protocol.PuzzleToClientCompletedTimeout, 50*time.Millisecond)

	// Host returns: players are told the new authoritative remaining time.
	host2, confirmed := ConnectHost(t, h)
	if !confirmed.IsReconnection || confirmed.CurrentPhase != protocol.PhasePuzzleAssembly {
		t.Fatalf("host handshake = %+v", confirmed)
	}

	reconnected := payloadAs[protocol.HostReconnected](t, a.Expect(protocol.SystemToClientHostReconnected))
	if reconnected.TimeRemaining == nil {
		t.Fatal("timeRemaining missing from HOST_RECONNECTED after a pause")
	}
	if *reconnected.TimeRemaining <= 0 || *reconnected.TimeRemaining > 1.2 {
		t.Errorf("timeRemaining = %v", *reconnected.TimeRemaining)
	}

	// Host replay: PHASE_LOAD → GRID_STATE → PHASE_START re-anchored with
	// totalTime = remaining and original decomposition values.
	host2.Expect(protocol.PuzzleToHostPhaseLoad)
	host2.Expect(protocol.PuzzleToHostGridState)
	replay := payloadAs[protocol.HostPuzzlePhaseStart](t, host2.Expect(protocol.PuzzleToHostPhaseStart))
	if !replay.TimerActive || replay.TotalTime > 1.2 || replay.TotalTime <= 0 {
		t.Errorf("replayed phase start = %+v", replay)
	}
	if replay.BaseTime != 1.2 || replay.ChronosBonus != 0 {
		t.Errorf("display decomposition changed: %+v", replay)
	}

	// The frozen recommendation is still pending and can now be accepted.
	b.Send(string(protocol.PuzzleToServerRecommendationResponse), b.ID, protocol.RecommendationResponse{
		MoveID: rec.MoveID, Response: "accept",
	})
	result := payloadAs[protocol.RecommendationResult](t, a.Expect(protocol.PuzzleToPlayerRecommendationResult))
	if result.SwapExecuted == nil {
		t.Fatalf("post-resume accept = %+v", result)
	}

	// The deadline was extended by the pause: the game eventually times out.
	a.Expect(protocol.PuzzleToClientCompletedTimeout)
}
