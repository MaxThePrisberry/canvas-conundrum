package integration

import (
	"testing"
	"time"

	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/protocol"
)

func nowStamp() string { return protocol.Timestamp(time.Now()) }

// Player couples a live client with its server-issued identity.
type Player struct {
	*Client
	ID   string
	Name string
	Role string
}

// Roles in assignment order for multi-player joins.
var joinRoles = []string{"art_enthusiast", "detective", "tourist", "janitor"}

// ConnectNewPlayer dials, performs the join handshake, and returns the
// player with its issued ID (not yet configured).
func ConnectNewPlayer(t *testing.T, h *Harness) *Player {
	t.Helper()
	c := DialPlayer(t, h)
	c.SendUnauthenticated(string(protocol.SetupToServerPlayerConnect), struct{}{})
	confirmed := payloadAs[protocol.PlayerConnectionConfirmed](t, c.Expect(protocol.SetupToPlayerConnectionConfirmed))
	if confirmed.PlayerID == "" || confirmed.IsReconnection {
		t.Fatalf("unexpected join handshake: %+v", confirmed)
	}
	return &Player{Client: c, ID: confirmed.PlayerID}
}

// ReconnectPlayer dials and reconnects with an existing token.
func ReconnectPlayer(t *testing.T, h *Harness, token string) (*Player, protocol.PlayerConnectionConfirmed) {
	t.Helper()
	c := DialPlayer(t, h)
	c.Send(string(protocol.SetupToServerPlayerConnect), token, struct{}{})
	confirmed := payloadAs[protocol.PlayerConnectionConfirmed](t, c.Expect(protocol.SetupToPlayerConnectionConfirmed))
	return &Player{Client: c, ID: confirmed.PlayerID}, confirmed
}

// Configure submits a configuration (role+specialties+name → ready).
func (p *Player) Configure(role, name string, specialties ...string) {
	p.t.Helper()
	if len(specialties) == 0 {
		specialties = []string{fixtureCategories[0]}
	}
	p.Send(string(protocol.SetupToServerPlayerConfiguration), p.ID, protocol.PlayerConfiguration{
		SelectedRole:        role,
		SelectedSpecialties: specialties,
		PlayerName:          name,
	})
	p.Name, p.Role = name, role
}

// Host couples a live host client with the host UUID.
type Host struct {
	*Client
	UUID string
}

// ConnectHost dials the host endpoint and completes the handshake.
func ConnectHost(t *testing.T, h *Harness) (*Host, protocol.HostConnectionConfirmed) {
	t.Helper()
	c := DialHost(t, h, h.HostUUID)
	confirmed := payloadAs[protocol.HostConnectionConfirmed](t, c.Expect(protocol.SetupToHostConnectionConfirmed))
	return &Host{Client: c, UUID: h.HostUUID}, confirmed
}

// JoinConfigured connects a host plus n configured players (roles cycle
// through the four types; capacity max(1,ceil(n/4)) always admits this).
func JoinConfigured(t *testing.T, h *Harness, n int) (*Host, []*Player) {
	t.Helper()
	host, _ := ConnectHost(t, h)
	players := make([]*Player, n)
	for i := range players {
		players[i] = ConnectNewPlayer(t, h)
		players[i].Expect(protocol.SetupToPlayerRolesAvailable)
	}
	// Configure after all joins so role capacity is already at its final
	// value and cycling assignments cannot hit ROLE_FULL.
	for i, p := range players {
		p.Configure(joinRoles[i%len(joinRoles)], playerName(i))
	}
	// Wait until every configuration has been processed so callers can
	// immediately start the game without racing the ready state.
	for {
		roster := payloadAs[protocol.PlayerRoster](t, host.Expect(protocol.SetupToHostPlayerRoster))
		if roster.ReadyPlayers == n {
			break
		}
	}
	return host, players
}

func playerName(i int) string {
	names := []string{"Alice", "Bob", "Charlie", "Diana", "Eve", "Frank", "Grace", "Heidi"}
	if i < len(names) {
		return names[i]
	}
	return names[i%len(names)] + "2"
}

// StartGame sends the host start signal and waits for the confirmation.
func (host *Host) StartGame() protocol.GameStarted {
	host.t.Helper()
	host.Send(string(protocol.SetupToServerStartGame), host.UUID, struct{}{})
	return payloadAs[protocol.GameStarted](host.t, host.Expect(protocol.SetupToHostGameStarted))
}

// ── Resource-phase helpers ─────────────────────────────────────────────────

// Scan submits a station QR hash and waits for the confirmation.
func (p *Player) Scan(hash string) string {
	p.t.Helper()
	p.Send(string(protocol.ResourceToServerLocationVerified), p.ID, protocol.LocationVerified{
		StationHash:   hash,
		ScanTimestamp: nowStamp(),
	})
	confirmed := payloadAs[protocol.LocationConfirmed](p.t, p.Expect(protocol.ResourceToPlayerLocationConfirmed))
	return confirmed.NewLocation
}

// ExpectQuestion waits for this round's trivia question.
func (p *Player) ExpectQuestion() protocol.TriviaQuestion {
	p.t.Helper()
	return payloadAs[protocol.TriviaQuestion](p.t, p.Expect(protocol.ResourceToPlayerTriviaQuestion))
}

// fixtureAnswerIndex locates the correct option ("4" or "yes" in every
// fixture pool) and returns its index, or a wrong index when correct=false.
func fixtureAnswerIndex(t *testing.T, q protocol.TriviaQuestion, correct bool) int {
	t.Helper()
	correctIdx := -1
	for i, o := range q.Options {
		if o == "4" || o == "yes" {
			correctIdx = i
			break
		}
	}
	if correctIdx < 0 {
		t.Fatalf("no known correct option in %v", q.Options)
	}
	if correct {
		return correctIdx
	}
	return (correctIdx + 1) % len(q.Options)
}

// Answer submits an answer to q, correct or deliberately wrong.
func (p *Player) Answer(q protocol.TriviaQuestion, correct bool) {
	p.t.Helper()
	p.Send(string(protocol.ResourceToServerTriviaAnswer), p.ID, protocol.TriviaAnswer{
		QuestionID:  q.QuestionID,
		AnswerIndex: fixtureAnswerIndex(p.t, q, correct),
		TimeElapsed: 0.05,
	})
}

// ExpectAnswerResult waits for the end-of-window marking result.
func (p *Player) ExpectAnswerResult() protocol.AnswerResult {
	p.t.Helper()
	return payloadAs[protocol.AnswerResult](p.t, p.Expect(protocol.ResourceToPlayerAnswerResult))
}

// PlayResource plays every resource round with all players answering
// correctly. scans maps player index → station hash, scanned during the
// silent pre-round wait. Returns the phase-complete payload.
func PlayResource(t *testing.T, host *Host, players []*Player, rounds int, scans map[int]string) protocol.ResourcePhaseComplete {
	t.Helper()
	for idx, hash := range scans {
		players[idx].Expect(protocol.ResourceToClientPhaseStart)
		players[idx].Scan(hash)
	}
	for round := 1; round <= rounds; round++ {
		for _, p := range players {
			q := p.ExpectQuestion()
			p.Answer(q, true)
		}
	}
	return payloadAs[protocol.ResourcePhaseComplete](t, players[0].Expect(protocol.ResourceToClientPhaseComplete))
}

// StartPuzzle sends the host's puzzle start signal and waits for its
// PHASE_START confirmation.
func (host *Host) StartPuzzle() protocol.HostPuzzlePhaseStart {
	host.t.Helper()
	host.Send(string(protocol.PuzzleToServerPhaseStart), host.UUID, struct{}{})
	return payloadAs[protocol.HostPuzzlePhaseStart](host.t, host.Expect(protocol.PuzzleToHostPhaseStart))
}

// ── Assembly-phase helpers ─────────────────────────────────────────────────

// ReachAssembly drives a fresh harness through setup + resource gathering
// (rounds must match the config) and starts the puzzle timer. Every player
// consumes their PHASE_START so their inboxes are aligned.
func ReachAssembly(t *testing.T, h *Harness, n, rounds int, scans map[int]string) (*Host, []*Player) {
	t.Helper()
	host, players := JoinConfigured(t, h, n)
	host.StartGame()
	PlayResource(t, host, players, rounds, scans)
	host.Expect(protocol.PuzzleToHostReady)
	host.StartPuzzle()
	for _, p := range players {
		p.Expect(protocol.PuzzleToClientPhaseStart)
	}
	return host, players
}

// CompleteSegment reports the player's individual puzzle solved and returns
// the acknowledged fragment position.
func (p *Player) CompleteSegment(segmentID string, solveTime float64) protocol.SegmentAcknowledged {
	p.t.Helper()
	p.Send(string(protocol.PuzzleToServerSegmentCompleted), p.ID, protocol.SegmentCompleted{
		SegmentID:           segmentID,
		CompletionTimestamp: nowStamp(),
		SolveTime:           solveTime,
		ManualPiecesSolved:  4,
		PreSolvedPieces:     0,
	})
	return payloadAs[protocol.SegmentAcknowledged](p.t, p.Expect(protocol.PuzzleToPlayerSegmentAcknowledged))
}

// Move requests a fragment move/swap and returns the server's result.
func (p *Player) Move(segmentID string, target protocol.Position, swapWith *string) protocol.MoveResult {
	p.t.Helper()
	p.Send(string(protocol.PuzzleToServerFragmentMove), p.ID, protocol.FragmentMove{
		SegmentID:         segmentID,
		TargetPosition:    &target,
		SwapWithSegmentID: swapWith,
	})
	return payloadAs[protocol.MoveResult](p.t, p.Expect(protocol.PuzzleToPlayerMoveResult))
}

// ExpectGridState waits for the next periodic grid tick.
func (p *Player) ExpectGridState() protocol.GridState {
	p.t.Helper()
	return payloadAs[protocol.GridState](p.t, p.Expect(protocol.PuzzleToClientGridState))
}

// correctPosFor derives a segment's correct cell from its id
// (segment_{letter}{col} → {col-1, letter-index}).
func correctPosFor(segmentID string) protocol.Position {
	rest := segmentID[len("segment_"):]
	row := int(rest[0] - 'a')
	col := int(rest[1] - '1')
	return protocol.Position{X: col, Y: row}
}

// AssembleToVictory drives the grid to the solved state with real moves,
// swaps, and (when two owned fragments block each other) recommendations,
// then returns the success payload seen by players[0]. Owners move their own
// fragments; players[0] moves unassigned ones.
func AssembleToVictory(t *testing.T, players []*Player) protocol.CompletedSuccess {
	t.Helper()
	byID := map[string]*Player{}
	for _, p := range players {
		byID[p.ID] = p
	}
	driver := players[0]

	for attempt := 0; attempt < 300; attempt++ {
		frame := driver.ExpectOneOf(protocol.PuzzleToClientGridState, protocol.PuzzleToClientCompletedSuccess)
		if frame.Event == string(protocol.PuzzleToClientCompletedSuccess) {
			return payloadAs[protocol.CompletedSuccess](t, frame.Payload)
		}
		state := payloadAs[protocol.GridState](t, frame.Payload)

		occupant := map[protocol.Position]protocol.GridFragment{}
		for _, f := range state.Fragments {
			occupant[f.Position] = f
		}

		acted := false
		for _, f := range state.Fragments {
			want := correctPosFor(f.SegmentID)
			if f.Position == want {
				continue
			}
			mover := driver
			if f.PlayerID != nil {
				mover = byID[*f.PlayerID]
			}

			blocker, occupied := occupant[want]
			if !occupied {
				if res := mover.Move(f.SegmentID, want, nil); res.Status == "success" {
					acted = true
					break
				}
				continue
			}
			// Occupied: swap if the mover controls both fragments, else
			// route through a recommendation to the blocker's owner.
			if blocker.PlayerID == nil || *blocker.PlayerID == mover.ID {
				id := blocker.SegmentID
				if res := mover.Move(f.SegmentID, want, &id); res.Status == "success" {
					acted = true
					break
				}
				continue
			}
			if recommendSwap(t, mover, byID[*blocker.PlayerID], f.SegmentID, blocker.SegmentID) {
				acted = true
				break
			}
		}

		if acted {
			// Let cooldowns lapse before the next mutation.
			time.Sleep(80 * time.Millisecond)
		}
	}
	t.Fatal("grid did not reach victory within the attempt budget")
	return protocol.CompletedSuccess{}
}

// recommendSwap runs the full recommendation round-trip: sender proposes,
// target accepts, sender sees the executed swap. Returns false if the
// proposal was rejected (e.g. cooldown) so the solver can retry later.
func recommendSwap(t *testing.T, sender, target *Player, fromSegment, toSegment string) bool {
	t.Helper()
	sender.Send(string(protocol.PuzzleToServerRecommendMove), sender.ID, protocol.RecommendMove{
		TargetPlayerID: target.ID,
		FromSegmentID:  fromSegment,
		ToSegmentID:    toSegment,
		Reasoning:      "solver: both fragments belong on each other's cells",
	})

	frame := target.ExpectOneOf(protocol.PuzzleToPlayerMoveRecommendation)
	rec := payloadAs[protocol.MoveRecommendation](t, frame.Payload)
	target.Send(string(protocol.PuzzleToServerRecommendationResponse), target.ID, protocol.RecommendationResponse{
		MoveID:   rec.MoveID,
		Response: "accept",
	})

	result := payloadAs[protocol.RecommendationResult](t,
		sender.Expect(protocol.PuzzleToPlayerRecommendationResult))
	return result.SwapExecuted != nil
}
