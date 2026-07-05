package integration

import (
	"testing"
	"time"

	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/config"
	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/protocol"
)

// recSetup boots a 2-player game where both players are in 2B and returns a
// controllable fragment for A plus B's owned fragment, all off cooldown.
func recSetup(t *testing.T, mutate func(*config.Config)) (*Harness, *Host, *Player, *Player, protocol.GridFragment, protocol.GridFragment) {
	t.Helper()
	h := Start(t, func(c *config.Config) {
		noSpecialty(c)
		if mutate != nil {
			mutate(c)
		}
	}, nil)
	host, players := ReachAssembly(t, h, 2, 2, nil)
	a, b := players[0], players[1]
	a.CompleteSegment("segment_a1", 1)
	b.CompleteSegment("segment_a2", 1)

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
	time.Sleep(80 * time.Millisecond) // placement cooldowns lapse
	return h, host, a, b, unassigned, bFrag
}

func (p *Player) recommend(target *Player, from, to, reasoning string) {
	p.t.Helper()
	p.Send(string(protocol.PuzzleToServerRecommendMove), p.ID, protocol.RecommendMove{
		TargetPlayerID: target.ID,
		FromSegmentID:  from,
		ToSegmentID:    to,
		Reasoning:      reasoning,
	})
}

func TestRecommendationAcceptExecutesSwap(t *testing.T) {
	_, _, a, b, unassigned, bFrag := recSetup(t, nil)

	a.recommend(b, unassigned.SegmentID, bFrag.SegmentID, "swap these")

	rec := payloadAs[protocol.MoveRecommendation](t, b.Expect(protocol.PuzzleToPlayerMoveRecommendation))
	if rec.FromPlayerName != "Alice" || rec.FromSegmentID != unassigned.SegmentID ||
		rec.ToSegmentID != bFrag.SegmentID || rec.Reasoning != "swap these" {
		t.Fatalf("recommendation = %+v", rec)
	}
	if rec.ExpiresAt == "" {
		t.Error("expiresAt missing")
	}

	// A second outgoing recommendation while one is pending is rejected.
	a.recommend(b, unassigned.SegmentID, bFrag.SegmentID, "again")
	a.ExpectError(protocol.SystemToClientError, protocol.ErrRecommendationPending)

	reason := "Good strategic move"
	b.Send(string(protocol.PuzzleToServerRecommendationResponse), b.ID, protocol.RecommendationResponse{
		MoveID: rec.MoveID, Response: "accept", ResponseReason: &reason,
	})

	result := payloadAs[protocol.RecommendationResult](t, a.Expect(protocol.PuzzleToPlayerRecommendationResult))
	if result.Response != "accept" || result.ResponseReason != reason || result.SwapExecuted == nil {
		t.Fatalf("result = %+v", result)
	}
	swap := result.SwapExecuted
	// The swap exchanges the two fragments' positions.
	if swap.Segment1NewPosition != swap.Segment2OldPosition || swap.Segment2NewPosition != swap.Segment1OldPosition {
		t.Errorf("swap positions inconsistent: %+v", swap)
	}
	if swap.Segment1OldPosition != unassigned.Position || swap.Segment2OldPosition != bFrag.Position {
		t.Errorf("swap did not execute at current positions: %+v", swap)
	}

	// The pending slot is free again.
	time.Sleep(80 * time.Millisecond)
	a.recommend(b, unassigned.SegmentID, bFrag.SegmentID, "fresh one")
	b.Expect(protocol.PuzzleToPlayerMoveRecommendation)
}

func TestRecommendationRejectDoesNotSwap(t *testing.T) {
	_, _, a, b, unassigned, bFrag := recSetup(t, nil)

	a.recommend(b, unassigned.SegmentID, bFrag.SegmentID, "please?")
	rec := payloadAs[protocol.MoveRecommendation](t, b.Expect(protocol.PuzzleToPlayerMoveRecommendation))

	b.Send(string(protocol.PuzzleToServerRecommendationResponse), b.ID, protocol.RecommendationResponse{
		MoveID: rec.MoveID, Response: "reject",
	})
	result := payloadAs[protocol.RecommendationResult](t, a.Expect(protocol.PuzzleToPlayerRecommendationResult))
	if result.Response != "reject" || result.SwapExecuted != nil {
		t.Fatalf("reject result = %+v", result)
	}
}

func TestRecommendationCooldownGates(t *testing.T) {
	_, _, a, b, unassigned, bFrag := recSetup(t, nil)

	// Creation gate: move the from-fragment first, then recommend while it
	// cools down.
	res := a.Move(unassigned.SegmentID, bFrag.Position, &bFrag.SegmentID)
	if res.Status != "rejected" || res.Reason != protocol.MoveRejectNotOwner {
		t.Fatalf("expected not_owner for owned swap, got %+v", res)
	}
	// (that didn't start a cooldown — use a real move: swap with another
	// unassigned fragment)
	state := a.ExpectGridState()
	var other protocol.GridFragment
	for _, f := range state.Fragments {
		if f.PlayerID == nil && f.SegmentID != unassigned.SegmentID {
			other = f
			break
		}
	}
	moved := a.Move(unassigned.SegmentID, other.Position, &other.SegmentID)
	if moved.Status != "success" {
		t.Fatalf("setup move = %+v", moved)
	}

	a.recommend(b, unassigned.SegmentID, bFrag.SegmentID, "too soon")
	a.ExpectError(protocol.SystemToClientError, protocol.ErrCooldownActive)

	// After the cooldown, creation succeeds.
	time.Sleep(80 * time.Millisecond)
	a.recommend(b, unassigned.SegmentID, bFrag.SegmentID, "now ok")
	rec := payloadAs[protocol.MoveRecommendation](t, b.Expect(protocol.PuzzleToPlayerMoveRecommendation))

	// Accept gate: the sender moves the from-fragment again before B
	// accepts; the accept hits COOLDOWN_ACTIVE and the recommendation stays
	// pending (retryable). The first swap exchanged the two fragments, so
	// swapping back targets the from-fragment's original cell.
	moved2 := a.Move(unassigned.SegmentID, unassigned.Position, &other.SegmentID)
	if moved2.Status != "success" {
		t.Fatalf("re-move = %+v", moved2)
	}
	b.Send(string(protocol.PuzzleToServerRecommendationResponse), b.ID, protocol.RecommendationResponse{
		MoveID: rec.MoveID, Response: "accept",
	})
	b.ExpectError(protocol.SystemToClientError, protocol.ErrCooldownActive)

	time.Sleep(80 * time.Millisecond)
	b.Send(string(protocol.PuzzleToServerRecommendationResponse), b.ID, protocol.RecommendationResponse{
		MoveID: rec.MoveID, Response: "accept",
	})
	result := payloadAs[protocol.RecommendationResult](t, a.Expect(protocol.PuzzleToPlayerRecommendationResult))
	if result.SwapExecuted == nil {
		t.Fatalf("retried accept failed: %+v", result)
	}
}

func TestRecommendationTimeoutNotifiesBothParties(t *testing.T) {
	_, _, a, b, unassigned, bFrag := recSetup(t, nil)

	a.recommend(b, unassigned.SegmentID, bFrag.SegmentID, "ignore me")
	b.Expect(protocol.PuzzleToPlayerMoveRecommendation)

	// recommendationTimeout is 0.5s in the fixture config.
	expiredA := payloadAs[protocol.RecommendationExpired](t, a.Expect(protocol.PuzzleToPlayerRecommendationExpired))
	expiredB := payloadAs[protocol.RecommendationExpired](t, b.Expect(protocol.PuzzleToPlayerRecommendationExpired))
	if expiredA.Reason != "timeout" || expiredB.Reason != "timeout" || expiredA.MoveID != expiredB.MoveID {
		t.Errorf("expiry payloads = %+v / %+v", expiredA, expiredB)
	}

	// The sender's pending slot is released by the timeout.
	a.recommend(b, unassigned.SegmentID, bFrag.SegmentID, "second try")
	b.Expect(protocol.PuzzleToPlayerMoveRecommendation)
}

func TestRecommendationExpiresOnDisconnect(t *testing.T) {
	_, host, a, b, unassigned, bFrag := recSetup(t, nil)

	a.recommend(b, unassigned.SegmentID, bFrag.SegmentID, "leaving soon")
	b.Expect(protocol.PuzzleToPlayerMoveRecommendation)

	b.Close()
	host.Expect(protocol.SystemToHostPlayerDisconnected)

	expired := payloadAs[protocol.RecommendationExpired](t, a.Expect(protocol.PuzzleToPlayerRecommendationExpired))
	if expired.Reason != "player_disconnected" {
		t.Errorf("expiry reason = %s", expired.Reason)
	}
}

// A recommendation must target a fragment the target player owns, from a
// fragment the sender controls.
func TestRecommendationOwnershipValidation(t *testing.T) {
	_, _, a, b, unassigned, bFrag := recSetup(t, nil)

	// From-fragment owned by B: sender doesn't control it.
	a.recommend(b, bFrag.SegmentID, unassigned.SegmentID, "wrong way around")
	a.ExpectError(protocol.SystemToClientError, protocol.ErrForbiddenNotOwner)

	// To-fragment unassigned: swaps involving no other player's fragment are
	// direct moves, not recommendations.
	state := a.ExpectGridState()
	var other protocol.GridFragment
	for _, f := range state.Fragments {
		if f.PlayerID == nil && f.SegmentID != unassigned.SegmentID {
			other = f
			break
		}
	}
	a.recommend(b, unassigned.SegmentID, other.SegmentID, "not owned by target")
	a.ExpectError(protocol.SystemToClientError, protocol.ErrForbiddenNotOwner)
}
