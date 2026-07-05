package integration

import (
	"testing"

	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/protocol"
)

// Full preparation flow: PREPARING → READY → PHASE_LOAD to host and
// players, then the host start signal opens puzzle assembly.
func TestPrepFlowAndPhaseLoad(t *testing.T) {
	t.Parallel()
	h := Start(t, noSpecialty, nil)
	host, players := JoinConfigured(t, h, 2)
	a, b := players[0], players[1]
	host.StartGame()

	// a (art_enthusiast) farms clarity: 2 rounds × 30 = 60 → 2 thresholds.
	complete := PlayResource(t, host, players, 2, map[int]string{0: "hash-clarity"})
	if complete.ThresholdAchievements.Clarity != 2 {
		t.Fatalf("clarity thresholds = %d, want 2", complete.ThresholdAchievements.Clarity)
	}

	host.Expect(protocol.PuzzleToHostPreparing)
	host.Expect(protocol.PuzzleToHostReady)
	hostLoad := payloadAs[protocol.HostPuzzlePhaseLoad](t, host.Expect(protocol.PuzzleToHostPhaseLoad))
	if hostLoad.CentralGridSize != 3 || hostLoad.TotalFragments != 9 || hostLoad.PlayerCount != 2 {
		t.Errorf("host phase load = %+v", hostLoad)
	}
	// Join-order row-major assignment.
	if hostLoad.PlayerSegmentAssignments[a.ID] != "segment_a1" ||
		hostLoad.PlayerSegmentAssignments[b.ID] != "segment_a2" {
		t.Errorf("assignments = %v", hostLoad.PlayerSegmentAssignments)
	}
	if hostLoad.BonusEffects != complete.BonusEffects {
		t.Errorf("host bonusEffects %+v != phase-complete %+v", hostLoad.BonusEffects, complete.BonusEffects)
	}

	loadA := payloadAs[protocol.PuzzlePhaseLoad](t, a.Expect(protocol.PuzzleToClientPhaseLoad))
	if loadA.AssignedSegmentID != "segment_a1" || loadA.IndividualPuzzleSize != 4 ||
		loadA.CentralGridSize != 3 || loadA.Phase != protocol.PhasePuzzlePreparation {
		t.Errorf("player phase load = %+v", loadA)
	}
	if loadA.ClarityPreviewDuration != 0.5 { // 0.3 + 2×0.1
		t.Errorf("clarityPreviewDuration = %v, want 0.5", loadA.ClarityPreviewDuration)
	}

	// Host starts the timer; players reveal the UI.
	hostStart := host.StartPuzzle()
	if !hostStart.TimerActive || hostStart.PlayersInPhase2A != 2 || hostStart.PlayersInPhase2B != 0 {
		t.Errorf("host phase start = %+v", hostStart)
	}
	start := payloadAs[protocol.PuzzlePhaseStart](t, a.Expect(protocol.PuzzleToClientPhaseStart))
	if start.TotalTime != 3 || start.BaseTime != 3 || start.ChronosBonus != 0 { // 3s base × medium 1.0
		t.Errorf("times = %v/%v/%v", start.TotalTime, start.BaseTime, start.ChronosBonus)
	}
	if !start.ClarityPreviewActive || start.ClarityPreviewDuration != 0.5 {
		t.Errorf("clarity fields = %v/%v", start.ClarityPreviewActive, start.ClarityPreviewDuration)
	}
	if len(start.PlayerPhases.Phase2A) != 2 || len(start.PlayerPhases.Phase2B) != 0 {
		t.Errorf("playerPhases = %+v", start.PlayerPhases)
	}
}

// Chronos thresholds extend the authoritative totalTime.
func TestChronosBonusExtendsTotalTime(t *testing.T) {
	t.Parallel()
	h := Start(t, noSpecialty, nil)
	host, players := JoinConfigured(t, h, 3)
	host.StartGame()

	// players[2] is a tourist (chronos bonus): 2 rounds × 30 = 60 chronos
	// tokens → floor(60/20) = 3 thresholds → +1.5s (3 × 0.5s).
	complete := PlayResource(t, host, players, 2, map[int]string{2: "hash-chronos"})
	if complete.ThresholdAchievements.Chronos != 3 {
		t.Fatalf("chronos thresholds = %d", complete.ThresholdAchievements.Chronos)
	}
	if complete.BonusEffects.ChronosTimeBonus != 1.5 {
		t.Fatalf("chronos bonus = %v, want 1.5", complete.BonusEffects.ChronosTimeBonus)
	}

	host.Expect(protocol.PuzzleToHostReady)
	hostStart := host.StartPuzzle()
	if hostStart.TotalTime != 4.5 { // (3 + 1.5) × 1.0
		t.Errorf("totalTime = %v, want 4.5", hostStart.TotalTime)
	}
	if hostStart.BaseTime != 3 || hostStart.ChronosBonus != 1.5 {
		t.Errorf("decomposition = %v + %v", hostStart.BaseTime, hostStart.ChronosBonus)
	}
}

func TestPuzzleStartRejectedOutsidePrep(t *testing.T) {
	t.Parallel()
	h := Start(t, noSpecialty, nil)
	host, _ := JoinConfigured(t, h, 2)
	host.StartGame()

	// Still in resource gathering: rejected.
	host.Send(string(protocol.PuzzleToServerPhaseStart), host.UUID, struct{}{})
	host.ExpectError(protocol.SystemToHostError, protocol.ErrForbiddenPhase)
}

func TestDuplicatePuzzleStartRejected(t *testing.T) {
	t.Parallel()
	h := Start(t, noSpecialty, nil)
	host, players := JoinConfigured(t, h, 2)
	host.StartGame()
	PlayResource(t, host, players, 2, nil)
	host.Expect(protocol.PuzzleToHostReady)

	host.StartPuzzle()
	host.Send(string(protocol.PuzzleToServerPhaseStart), host.UUID, struct{}{})
	host.ExpectError(protocol.SystemToHostError, protocol.ErrForbiddenPhase)
}

// Reconnections during preparation replay the load state on the new socket.
func TestPrepReconnectReplays(t *testing.T) {
	t.Parallel()
	h := Start(t, noSpecialty, nil)
	host, players := JoinConfigured(t, h, 2)
	a := players[0]
	host.StartGame()
	PlayResource(t, host, players, 2, nil)

	a.Expect(protocol.PuzzleToClientPhaseLoad)
	host.Expect(protocol.PuzzleToHostReady)

	// Player drops and returns: PHASE_LOAD replays with their assignment.
	a.Close()
	host.Expect(protocol.SystemToHostPlayerDisconnected)
	a2, confirmed := ReconnectPlayer(t, h, a.ID)
	if confirmed.CurrentPhase != protocol.PhasePuzzlePreparation {
		t.Fatalf("phase = %s", confirmed.CurrentPhase)
	}
	load := payloadAs[protocol.PuzzlePhaseLoad](t, a2.Expect(protocol.PuzzleToClientPhaseLoad))
	if load.AssignedSegmentID != "segment_a1" {
		t.Errorf("replayed assignment = %s", load.AssignedSegmentID)
	}

	// Host drops and returns: READY + HOST_PHASE_LOAD replay.
	host.Close()
	a2.Expect(protocol.SystemToClientHostDisconnected)
	host2, hostConfirmed := ConnectHost(t, h)
	if hostConfirmed.CurrentPhase != protocol.PhasePuzzlePreparation {
		t.Fatalf("host phase = %s", hostConfirmed.CurrentPhase)
	}
	host2.Expect(protocol.PuzzleToHostReady)
	host2.Expect(protocol.PuzzleToHostPhaseLoad)

	// The reconnected host can start the game.
	host2.StartPuzzle()
}
