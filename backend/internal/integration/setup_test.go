package integration

import (
	"testing"

	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/config"
	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/protocol"
)

func TestJoinHandshakeAndRolesAvailable(t *testing.T) {
	h := Start(t, nil, nil)
	p := ConnectNewPlayer(t, h)

	ra := payloadAs[protocol.RolesAvailable](t, p.Expect(protocol.SetupToPlayerRolesAvailable))
	if len(ra.Roles) != 4 {
		t.Fatalf("got %d roles, want 4", len(ra.Roles))
	}
	wantOrder := []string{"art_enthusiast", "detective", "tourist", "janitor"}
	wantBonus := map[string]string{
		"art_enthusiast": "clarity", "detective": "guide", "tourist": "chronos", "janitor": "anchor",
	}
	for i, r := range ra.Roles {
		if r.RoleType != wantOrder[i] {
			t.Errorf("roles[%d] = %s, want %s", i, r.RoleType, wantOrder[i])
		}
		if r.BonusTokenType != wantBonus[r.RoleType] {
			t.Errorf("%s bonus = %s, want %s", r.RoleType, r.BonusTokenType, wantBonus[r.RoleType])
		}
		if !r.Available {
			t.Errorf("%s should be available in an empty lobby", r.RoleType)
		}
		if r.ResourceBonus != 1.5 {
			t.Errorf("%s resourceBonus = %v, want 1.5", r.RoleType, r.ResourceBonus)
		}
	}
	if len(ra.TriviaCategories) != len(fixtureCategories) {
		t.Errorf("triviaCategories = %v", ra.TriviaCategories)
	}
	if ra.MaxSpecialties != 2 {
		t.Errorf("maxSpecialties = %d, want 2", ra.MaxSpecialties)
	}
}

func TestConfigurationUpdatesLobbyAndRoster(t *testing.T) {
	h := Start(t, nil, nil)
	host, _ := ConnectHost(t, h)
	p := ConnectNewPlayer(t, h)
	p.Expect(protocol.SetupToPlayerRolesAvailable)

	p.Configure("detective", "Alice", fixtureCategories[0])

	// Lobby broadcast reflects the atomic configure-and-ready.
	var lobby protocol.LobbyStatus
	for {
		lobby = payloadAs[protocol.LobbyStatus](t, p.Expect(protocol.SetupToClientLobbyStatus))
		if lobby.ReadyPlayers == 1 {
			break
		}
	}
	if lobby.PlayerRoles["detective"] != 1 {
		t.Errorf("playerRoles = %v", lobby.PlayerRoles)
	}
	if !lobby.HasHost || lobby.AllPlayersReady == false {
		t.Errorf("lobby = %+v", lobby)
	}
	if lobby.GameStartEligible {
		t.Error("1 ready player must not be start-eligible with minPlayers=2")
	}

	// Roster reaches the host with full player detail.
	var roster protocol.PlayerRoster
	for {
		roster = payloadAs[protocol.PlayerRoster](t, host.Expect(protocol.SetupToHostPlayerRoster))
		if roster.ReadyPlayers == 1 {
			break
		}
	}
	st, ok := roster.PlayerStatuses[p.ID]
	if !ok {
		t.Fatalf("roster missing player %s", p.ID)
	}
	if st.PlayerName != "Alice" || st.Role == nil || *st.Role != "detective" || !st.Ready || !st.Connected {
		t.Errorf("roster status = %+v", st)
	}
	if roster.RoleDistribution["detective"] != 1 || roster.RoleDistribution["janitor"] != 0 {
		t.Errorf("roleDistribution = %v", roster.RoleDistribution)
	}
}

func TestInvalidRoleRejected(t *testing.T) {
	h := Start(t, nil, nil)
	p := ConnectNewPlayer(t, h)
	p.Expect(protocol.SetupToPlayerRolesAvailable)

	p.Configure("wizard", "Alice")
	errPayload := p.ExpectError(protocol.SystemToClientError, protocol.ErrInvalidRoleSelection)
	if errPayload.ErrorType != protocol.ErrorTypeValidation {
		t.Errorf("errorType = %s", errPayload.ErrorType)
	}
}

func TestInvalidSpecialtiesRejected(t *testing.T) {
	h := Start(t, nil, nil)
	p := ConnectNewPlayer(t, h)
	p.Expect(protocol.SetupToPlayerRolesAvailable)

	cases := [][]string{
		{},                         // empty
		{"alpha", "alpha"},         // duplicate
		{"alpha", "beta", "gamma"}, // exceeds maxSpecialtiesPerPlayer=2
		{"nonexistent"},            // unknown category
	}
	for _, specialties := range cases {
		p.Send(string(protocol.SetupToServerPlayerConfiguration), p.ID, protocol.PlayerConfiguration{
			SelectedRole:        "detective",
			SelectedSpecialties: specialties,
			PlayerName:          "Alice",
		})
		p.ExpectError(protocol.SystemToClientError, protocol.ErrInvalidSpecialtySelection)
	}
}

// Two players race for the single detective slot (capacity max(1,ceil(2/4))
// = 1): the first configuration to land wins, the loser gets ROLE_FULL and
// succeeds with a different role.
func TestRoleFullFirstWins(t *testing.T) {
	h := Start(t, nil, nil)
	a := ConnectNewPlayer(t, h)
	b := ConnectNewPlayer(t, h)
	a.Expect(protocol.SetupToPlayerRolesAvailable)
	b.Expect(protocol.SetupToPlayerRolesAvailable)

	a.Configure("detective", "Alice")
	// Wait until A's configuration is fully processed before B races in, so
	// the winner is deterministic for assertion purposes.
	for {
		lobby := payloadAs[protocol.LobbyStatus](t, b.Expect(protocol.SetupToClientLobbyStatus))
		if lobby.PlayerRoles["detective"] == 1 {
			break
		}
	}

	b.Configure("detective", "Bob")
	errPayload := b.ExpectError(protocol.SystemToClientError, protocol.ErrRoleFull)
	if errPayload.ErrorType != protocol.ErrorTypeValidation {
		t.Errorf("errorType = %s", errPayload.ErrorType)
	}

	// Loser resubmits a full configuration with another role.
	b.Configure("janitor", "Bob")
	for {
		lobby := payloadAs[protocol.LobbyStatus](t, b.Expect(protocol.SetupToClientLobbyStatus))
		if lobby.PlayerRoles["janitor"] == 1 && lobby.ReadyPlayers == 2 {
			break
		}
	}
}

func TestConfigurationLockedOnceReady(t *testing.T) {
	h := Start(t, nil, nil)
	p := ConnectNewPlayer(t, h)
	p.Expect(protocol.SetupToPlayerRolesAvailable)

	p.Configure("detective", "Alice")
	p.Configure("janitor", "Alice II")
	p.ExpectError(protocol.SystemToClientError, protocol.ErrConfigurationLocked)
}

// With 5 connected players capacity is max(1,ceil(5/4)) = 2, so two players
// can hold the same role.
func TestRoleCapacityScalesWithPlayerCount(t *testing.T) {
	h := Start(t, nil, nil)
	players := make([]*Player, 5)
	for i := range players {
		players[i] = ConnectNewPlayer(t, h)
		players[i].Expect(protocol.SetupToPlayerRolesAvailable)
	}

	players[0].Configure("detective", "Alice")
	players[1].Configure("detective", "Bob")

	for {
		lobby := payloadAs[protocol.LobbyStatus](t, players[2].Expect(protocol.SetupToClientLobbyStatus))
		if lobby.PlayerRoles["detective"] == 2 {
			break
		}
	}

	// Third detective must be rejected at capacity 2.
	players[2].Configure("detective", "Charlie")
	players[2].ExpectError(protocol.SystemToClientError, protocol.ErrRoleFull)
}

func TestStartGameGate(t *testing.T) {
	h := Start(t, func(c *config.Config) { c.MinPlayers = 3 }, nil)
	host, _ := ConnectHost(t, h)

	// No players at all.
	host.Send(string(protocol.SetupToServerStartGame), host.UUID, struct{}{})
	host.ExpectError(protocol.SystemToHostError, protocol.ErrInsufficientPlayers)

	// Two ready players: below minPlayers=3.
	a := ConnectNewPlayer(t, h)
	b := ConnectNewPlayer(t, h)
	a.Expect(protocol.SetupToPlayerRolesAvailable)
	b.Expect(protocol.SetupToPlayerRolesAvailable)
	a.Configure("detective", "Alice")
	b.Configure("janitor", "Bob")
	host.Send(string(protocol.SetupToServerStartGame), host.UUID, struct{}{})
	host.ExpectError(protocol.SystemToHostError, protocol.ErrInsufficientPlayers)

	// Third player connected but not ready: every connected player must be
	// ready even though ready count reaches minPlayers... (2 ready + 1 idle)
	c := ConnectNewPlayer(t, h)
	c.Expect(protocol.SetupToPlayerRolesAvailable)
	host.Send(string(protocol.SetupToServerStartGame), host.UUID, struct{}{})
	host.ExpectError(protocol.SystemToHostError, protocol.ErrInsufficientPlayers)

	// All three ready: start succeeds.
	c.Configure("tourist", "Charlie")
	for {
		roster := payloadAs[protocol.PlayerRoster](t, host.Expect(protocol.SetupToHostPlayerRoster))
		if roster.GameStartEligible {
			break
		}
	}
	started := host.StartGame()
	if started.Phase != protocol.PhaseResourceGathering || started.TotalPlayers != 3 {
		t.Errorf("game started = %+v", started)
	}
	if started.InitialTeamTokens != (protocol.TeamTokens{}) {
		t.Errorf("initial tokens = %+v, want zeros", started.InitialTeamTokens)
	}

	// A second start is rejected: no longer in setup.
	host.Send(string(protocol.SetupToServerStartGame), host.UUID, struct{}{})
	host.ExpectError(protocol.SystemToHostError, protocol.ErrForbiddenPhase)
}
