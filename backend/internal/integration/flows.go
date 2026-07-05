package integration

import (
	"testing"

	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/protocol"
)

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
