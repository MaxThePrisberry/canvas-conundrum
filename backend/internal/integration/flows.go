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
