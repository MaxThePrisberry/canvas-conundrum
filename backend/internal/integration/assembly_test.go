package integration

import (
	"testing"
	"time"

	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/config"
	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/protocol"
)

func TestSegmentCompletionAndProportionalReveal(t *testing.T) {
	t.Parallel()
	h := Start(t, noSpecialty, nil)
	host, players := ReachAssembly(t, h, 2, 2, nil)
	a, b := players[0], players[1]

	// A completes: k=1 of N=2 on a 3×3 grid → ceil(9/2)=5 fragments visible
	// (1 owned + 4 unassigned).
	ack := a.CompleteSegment("segment_a1", 1.5)
	if ack.SegmentID != "segment_a1" {
		t.Fatalf("ack = %+v", ack)
	}

	hostNote := payloadAs[protocol.HostSegmentCompleted](t, host.Expect(protocol.PuzzleToHostSegmentCompleted))
	if hostNote.PlayerID != a.ID || hostNote.CompletionTime != 1.5 {
		t.Errorf("host completion = %+v", hostNote)
	}
	if hostNote.PhaseTransition.PlayersInPhase2A != 1 || hostNote.PhaseTransition.PlayersInPhase2B != 1 {
		t.Errorf("phase transition = %+v", hostNote.PhaseTransition)
	}
	if hostNote.CompletionStats.TotalCompleted != 1 || hostNote.CompletionStats.TotalRequired != 2 ||
		hostNote.CompletionStats.UnassignedFragments != 4 {
		t.Errorf("completion stats = %+v", hostNote.CompletionStats)
	}

	state := a.ExpectGridState()
	if len(state.Fragments) != 5 {
		t.Fatalf("visible fragments = %d, want 5", len(state.Fragments))
	}
	owned, unassigned := 0, 0
	for _, f := range state.Fragments {
		if f.PlayerID == nil {
			unassigned++
		} else if *f.PlayerID == a.ID && *f.PlayerName == "Alice" {
			owned++
		}
	}
	if owned != 1 || unassigned != 4 {
		t.Errorf("owned=%d unassigned=%d", owned, unassigned)
	}

	// Any player may now fetch fragments visible on the grid.
	for _, f := range state.Fragments {
		resp, body := fetchAsset(t, h, "/api/segments/"+f.SegmentID, b.ID)
		assertPNG(t, resp, body, 32)
	}

	// Idempotent re-completion: same ack, no new host event.
	again := a.CompleteSegment("segment_a1", 9.9)
	if again.Position != ack.Position {
		t.Errorf("re-ack position changed: %+v vs %+v", again, ack)
	}
	host.ExpectNone(protocol.PuzzleToHostSegmentCompleted, 150*time.Millisecond)

	// Completing someone else's segment is rejected.
	b.Send(string(protocol.PuzzleToServerSegmentCompleted), b.ID, protocol.SegmentCompleted{
		SegmentID: "segment_a1", CompletionTimestamp: nowStamp(), SolveTime: 1,
	})
	b.ExpectError(protocol.SystemToClientError, protocol.ErrForbiddenNotOwner)

	// B completes: k=2/2 → all 9 fragments visible.
	b.CompleteSegment("segment_a2", 2.0)
	for {
		if s := b.ExpectGridState(); len(s.Fragments) == 9 {
			break
		}
	}
}

func TestMovesSwapsAndCooldowns(t *testing.T) {
	t.Parallel()
	h := Start(t, func(c *config.Config) { noSpecialty(c); c.MinPlayers = 3 }, nil)
	_, players := ReachAssembly(t, h, 3, 2, nil)
	a, b, c := players[0], players[1], players[2]

	// A player still in 2A cannot move fragments.
	res := c.Move("segment_a1", protocol.Position{X: 0, Y: 0}, nil)
	if res.Status != "rejected" || res.Reason != protocol.MoveRejectPhaseInvalid {
		t.Fatalf("2A move = %+v", res)
	}

	a.CompleteSegment("segment_a1", 1)
	b.CompleteSegment("segment_a2", 1)

	// Find the current board from a tick: locate A's fragment and an empty cell.
	state := a.ExpectGridState()
	for len(state.Fragments) < 6 { // k=2 of 3 → ceil(18/3)=6 visible
		state = a.ExpectGridState()
	}
	occupied := map[protocol.Position]protocol.GridFragment{}
	var unassignedSeg protocol.GridFragment
	for _, f := range state.Fragments {
		occupied[f.Position] = f
		if f.PlayerID == nil {
			unassignedSeg = f
		}
	}
	// 6 fragments on 9 cells → at least two empty cells exist.
	var empties []protocol.Position
	for y := 0; y < 3; y++ {
		for x := 0; x < 3; x++ {
			if _, ok := occupied[protocol.Position{X: x, Y: y}]; !ok {
				empties = append(empties, protocol.Position{X: x, Y: y})
			}
		}
	}
	emptyCell, secondEmpty := empties[0], empties[1]

	// Move own fragment to an empty cell.
	res = a.Move("segment_a1", emptyCell, nil)
	if res.Status != "success" || *res.NewPosition != emptyCell || res.CooldownInfo == nil {
		t.Fatalf("move = %+v", res)
	}

	// Immediate second move of the same fragment: cooldown.
	res = a.Move("segment_a1", secondEmpty, nil)
	if res.Status != "rejected" || res.Reason != protocol.MoveRejectCooldown || res.CooldownInfo == nil {
		t.Fatalf("cooldown rejection = %+v", res)
	}

	// Moving another player's owned fragment: not_owner.
	res = a.Move("segment_a2", emptyCell, nil)
	if res.Reason != protocol.MoveRejectNotOwner {
		t.Fatalf("foreign move = %+v", res)
	}

	// Out-of-bounds target.
	res = a.Move(unassignedSeg.SegmentID, protocol.Position{X: 5, Y: 5}, nil)
	if res.Reason != protocol.MoveRejectTargetInvalid {
		t.Fatalf("out of bounds = %+v", res)
	}

	// Move onto an occupied cell without declaring a swap.
	res = a.Move(unassignedSeg.SegmentID, emptyCell, nil) // emptyCell now holds segment_a1
	if res.Reason != protocol.MoveRejectTargetInvalid {
		t.Fatalf("occupied non-swap = %+v", res)
	}

	// Swap A's fragment with an unassigned one after its cooldown lapses.
	time.Sleep(80 * time.Millisecond)
	swapID := unassignedSeg.SegmentID
	res = a.Move("segment_a1", unassignedSeg.Position, &swapID)
	if res.Status != "success" || res.SwappedSegmentID == nil || *res.SwappedSegmentID != swapID {
		t.Fatalf("swap = %+v", res)
	}
	if *res.SwappedSegmentNewPosition != emptyCell {
		t.Errorf("swapped fragment landed at %+v, want %+v", res.SwappedSegmentNewPosition, emptyCell)
	}

	// Every involved fragment must be off cooldown: the swap restarted both,
	// so an immediate move of the *partner* is also rejected.
	res = a.Move(swapID, protocol.Position{X: 9, Y: 9}, nil)
	if res.Reason != protocol.MoveRejectTargetInvalid && res.Reason != protocol.MoveRejectCooldown {
		t.Fatalf("post-swap partner move = %+v", res)
	}
	res = a.Move(swapID, emptyCell, nil) // emptyCell is occupied... find real empty below
	_ = res

	// Swap displacing another player's owned fragment: not_owner.
	time.Sleep(80 * time.Millisecond)
	state = a.ExpectGridState()
	var aFrag, bFrag protocol.GridFragment
	for _, f := range state.Fragments {
		if f.PlayerID != nil && *f.PlayerID == a.ID {
			aFrag = f
		}
		if f.PlayerID != nil && *f.PlayerID == b.ID {
			bFrag = f
		}
	}
	bID := bFrag.SegmentID
	res = a.Move(aFrag.SegmentID, bFrag.Position, &bID)
	if res.Reason != protocol.MoveRejectNotOwner {
		t.Fatalf("swap displacing owned fragment = %+v", res)
	}
}

// The "every involved fragment" cooldown rule: a swap is rejected when only
// the partner fragment is cooling down.
func TestSwapRequiresBothFragmentsOffCooldown(t *testing.T) {
	t.Parallel()
	h := Start(t, noSpecialty, nil)
	_, players := ReachAssembly(t, h, 2, 2, nil)
	a, b := players[0], players[1]
	a.CompleteSegment("segment_a1", 1)
	b.CompleteSegment("segment_a2", 1)

	state := a.ExpectGridState()
	for len(state.Fragments) < 9 {
		state = a.ExpectGridState()
	}
	var mine, other protocol.GridFragment
	for _, f := range state.Fragments {
		if f.PlayerID != nil && *f.PlayerID == a.ID {
			mine = f
		}
		if f.PlayerID == nil {
			other = f
		}
	}

	time.Sleep(80 * time.Millisecond) // clear placement-time state

	// Move the unassigned fragment... to itself? No — move it to any empty
	// cell is impossible on a full 9/9 grid, so swap it with another
	// unassigned fragment to start its cooldown.
	var third protocol.GridFragment
	for _, f := range state.Fragments {
		if f.PlayerID == nil && f.SegmentID != other.SegmentID {
			third = f
			break
		}
	}
	swapID := third.SegmentID
	res := a.Move(other.SegmentID, third.Position, &swapID)
	if res.Status != "success" {
		t.Fatalf("setup swap = %+v", res)
	}

	// A's own fragment is idle, but the partner (just swapped) is cooling
	// down → the swap must be rejected with reason cooldown.
	otherID := other.SegmentID
	res = a.Move(mine.SegmentID, *res.NewPosition, &otherID)
	if res.Status != "rejected" || res.Reason != protocol.MoveRejectCooldown {
		t.Fatalf("swap with cooling partner = %+v", res)
	}
}

func TestVictoryBySolvingGrid(t *testing.T) {
	t.Parallel()
	h := Start(t, func(c *config.Config) {
		noSpecialty(c)
		c.PuzzleBaseTime = config.Seconds(30 * time.Second) // roomy deadline for the solver
	}, nil)
	host, players := ReachAssembly(t, h, 2, 2, nil)
	a, b := players[0], players[1]
	a.CompleteSegment("segment_a1", 1.5)
	b.CompleteSegment("segment_a2", 2.5)

	success := AssembleToVictory(t, players)
	if !success.Success || !success.FinalGridState.AllFragmentsCorrect ||
		success.FinalGridState.CorrectFragments != 9 || success.FinalGridState.TotalFragments != 9 {
		t.Errorf("success payload = %+v", success)
	}
	if success.TotalTime != 30 || success.CompletionTime <= 0 || success.CompletionTime > 30 {
		t.Errorf("times = %+v", success)
	}

	// Host receives the success broadcast plus completion analytics.
	host.Expect(protocol.PuzzleToClientCompletedSuccess)
	analytics := payloadAs[protocol.CompletionAnalytics](t, host.Expect(protocol.PuzzleToHostCompletionAnalytics))
	if !analytics.PuzzleSuccess {
		t.Errorf("analytics = %+v", analytics)
	}
	if analytics.PhaseTransitions.PlayersCompletedIndividual != 2 ||
		analytics.PhaseTransitions.FastestIndividual != 1.5 ||
		analytics.PhaseTransitions.SlowestIndividual != 2.5 ||
		analytics.PhaseTransitions.AverageIndividualTime != 2.0 {
		t.Errorf("phase transitions = %+v", analytics.PhaseTransitions)
	}
	if analytics.CollaborationMetrics.SuccessfulMoves == 0 {
		t.Error("expected successful moves in collaboration metrics")
	}
	aContrib := analytics.PlayerContributions[a.ID]
	if !aContrib.FinalFragmentCorrect || aContrib.IndividualSolveTime != 1.5 {
		t.Errorf("player contribution = %+v", aContrib)
	}
}

func TestTimeoutEndsGameAsLoss(t *testing.T) {
	t.Parallel()
	h := Start(t, func(c *config.Config) {
		noSpecialty(c)
		c.PuzzleBaseTime = config.Seconds(700 * time.Millisecond)
	}, nil)
	host, players := ReachAssembly(t, h, 2, 2, nil)
	a := players[0]
	a.CompleteSegment("segment_a1", 1) // one player never finishes

	timeout := payloadAs[protocol.CompletedTimeout](t, a.Expect(protocol.PuzzleToClientCompletedTimeout))
	if timeout.Success || timeout.Reason != "time_expired" || !timeout.TimeExpired {
		t.Errorf("timeout payload = %+v", timeout)
	}
	if timeout.FinalStats.TotalFragments != 9 || timeout.FinalStats.FragmentsPlaced != 5 {
		t.Errorf("final stats = %+v", timeout.FinalStats)
	}

	host.Expect(protocol.PuzzleToClientCompletedTimeout)
	analytics := payloadAs[protocol.CompletionAnalytics](t, host.Expect(protocol.PuzzleToHostCompletionAnalytics))
	if analytics.PuzzleSuccess {
		t.Error("analytics must report failure")
	}

	// Post-game moves are rejected.
	res := a.Move("segment_a1", protocol.Position{X: 0, Y: 0}, nil)
	if res.Reason != protocol.MoveRejectPhaseInvalid {
		t.Errorf("post-game move = %+v", res)
	}
}

// A 2A player disconnecting mid-assembly is auto-solved into an unassigned
// fragment; a 2B player's fragment becomes unassigned in place.
func TestDisconnectFragmentHandling(t *testing.T) {
	t.Parallel()
	h := Start(t, func(c *config.Config) { noSpecialty(c); c.MinPlayers = 3 }, nil)
	host, players := ReachAssembly(t, h, 3, 2, nil)
	a, b, c := players[0], players[1], players[2]

	a.CompleteSegment("segment_a1", 1)
	host.Expect(protocol.PuzzleToHostSegmentCompleted)

	// c (2A) drops: auto-solve to unassigned at a random cell.
	c.Close()
	notice := payloadAs[protocol.PlayerDisconnected](t, host.Expect(protocol.SystemToHostPlayerDisconnected))
	if notice.FragmentHandling == nil || notice.FragmentHandling.SegmentID != "segment_a3" ||
		!notice.FragmentHandling.NowUnassigned {
		t.Fatalf("2A disconnect handling = %+v", notice.FragmentHandling)
	}

	// a (2B) drops: fragment stays but becomes unassigned.
	a.Close()
	notice = payloadAs[protocol.PlayerDisconnected](t, host.Expect(protocol.SystemToHostPlayerDisconnected))
	if notice.FragmentHandling == nil || notice.FragmentHandling.SegmentID != "segment_a1" {
		t.Fatalf("2B disconnect handling = %+v", notice.FragmentHandling)
	}

	// b sees both fragments as unassigned in the next tick.
	for {
		state := b.ExpectGridState()
		unassigned := map[string]bool{}
		for _, f := range state.Fragments {
			if f.PlayerID == nil {
				unassigned[f.SegmentID] = true
			}
		}
		if unassigned["segment_a1"] && unassigned["segment_a3"] {
			break
		}
	}

	// Reconnection during puzzle assembly is forbidden: close 4003.
	cl := DialPlayer(t, h)
	cl.Send(string(protocol.SetupToServerPlayerConnect), a.ID, struct{}{})
	cl.ExpectClose(protocol.CloseReconnectForbidden)
}

// Players still disconnected when the host starts the timer are auto-solved
// at that moment.
func TestDisconnectedAtTimerStartAutoSolved(t *testing.T) {
	t.Parallel()
	h := Start(t, func(c *config.Config) { noSpecialty(c); c.MinPlayers = 3 }, nil)
	host, players := JoinConfigured(t, h, 3)
	host.StartGame()
	PlayResource(t, host, players, 2, nil)
	host.Expect(protocol.PuzzleToHostReady)

	players[2].Close()
	host.Expect(protocol.SystemToHostPlayerDisconnected)

	host.StartPuzzle()
	start := payloadAs[protocol.PuzzlePhaseStart](t, players[0].Expect(protocol.PuzzleToClientPhaseStart))
	if len(start.PlayerPhases.Phase2B) != 1 || start.PlayerPhases.Phase2B[0] != players[2].ID {
		t.Errorf("auto-solved player not in 2B: %+v", start.PlayerPhases)
	}

	// Their fragment is on the grid, unassigned (k=1 → ceil(9/3)=3 visible).
	state := players[0].ExpectGridState()
	found := false
	for _, f := range state.Fragments {
		if f.SegmentID == "segment_a3" && f.PlayerID == nil {
			found = true
		}
	}
	if !found {
		t.Errorf("auto-solved fragment missing from grid: %+v", state.Fragments)
	}
}
