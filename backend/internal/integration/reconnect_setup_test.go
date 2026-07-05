package integration

import (
	"testing"

	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/protocol"
)

func TestSetupDisconnectUpdatesCountsAndNotifiesHost(t *testing.T) {
	h := Start(t, nil, nil)
	host, _ := ConnectHost(t, h)
	a := ConnectNewPlayer(t, h)
	b := ConnectNewPlayer(t, h)
	a.Expect(protocol.SetupToPlayerRolesAvailable)
	b.Expect(protocol.SetupToPlayerRolesAvailable)
	a.Configure("detective", "Alice")
	// Ensure the configuration landed before the disconnect races it.
	for {
		lobby := payloadAs[protocol.LobbyStatus](t, a.Expect(protocol.SetupToClientLobbyStatus))
		if lobby.ReadyPlayers == 1 {
			break
		}
	}

	b.Close()

	notice := payloadAs[protocol.PlayerDisconnected](t, host.Expect(protocol.SystemToHostPlayerDisconnected))
	if notice.PlayerID != b.ID || notice.CurrentPhase != protocol.PhaseSetup {
		t.Errorf("disconnect notice = %+v", notice)
	}
	if notice.UpdatedCounts == nil {
		t.Fatal("setup disconnect must carry updatedCounts")
	}
	if notice.UpdatedCounts.ConnectedPlayers != 1 || notice.UpdatedCounts.ReadyPlayers != 1 {
		t.Errorf("updatedCounts = %+v", notice.UpdatedCounts)
	}

	// Remaining players see the shrunken lobby.
	for {
		lobby := payloadAs[protocol.LobbyStatus](t, a.Expect(protocol.SetupToClientLobbyStatus))
		if lobby.CurrentPlayers == 1 {
			break
		}
	}
}

func TestSetupReconnectRestoresConfiguration(t *testing.T) {
	h := Start(t, nil, nil)
	host, _ := ConnectHost(t, h)
	a := ConnectNewPlayer(t, h)
	a.Expect(protocol.SetupToPlayerRolesAvailable)
	a.Configure("detective", "Alice", fixtureCategories[0], fixtureCategories[1])
	host.Expect(protocol.SetupToHostPlayerRoster)

	a.Close()
	host.Expect(protocol.SystemToHostPlayerDisconnected)

	a2, confirmed := ReconnectPlayer(t, h, a.ID)
	if !confirmed.IsReconnection || confirmed.PlayerID != a.ID {
		t.Fatalf("handshake = %+v", confirmed)
	}
	cfg := confirmed.ExistingConfiguration
	if cfg == nil {
		t.Fatal("existingConfiguration missing on reconnect")
	}
	if cfg.SelectedRole == nil || *cfg.SelectedRole != "detective" || !cfg.Ready ||
		cfg.PlayerName != "Alice" || len(cfg.SelectedSpecialties) != 2 {
		t.Errorf("existingConfiguration = %+v", cfg)
	}

	// Ready players are configuration-locked, so no ROLES_AVAILABLE replay.
	a2.ExpectNone(protocol.SetupToPlayerRolesAvailable, 150e6)
}

// If the reconnecting player's role slot was taken while they were away, the
// role (and ready state) is dropped and they must reselect.
func TestSetupReconnectRoleSlotLost(t *testing.T) {
	h := Start(t, nil, nil)
	a := ConnectNewPlayer(t, h)
	a.Expect(protocol.SetupToPlayerRolesAvailable)
	a.Configure("detective", "Alice")

	a.Close()

	// With Alice gone (1 connected → capacity 1), Bob takes the slot.
	b := ConnectNewPlayer(t, h)
	b.Expect(protocol.SetupToPlayerRolesAvailable)
	b.Configure("detective", "Bob")
	for {
		lobby := payloadAs[protocol.LobbyStatus](t, b.Expect(protocol.SetupToClientLobbyStatus))
		if lobby.PlayerRoles["detective"] == 1 {
			break
		}
	}

	a2, confirmed := ReconnectPlayer(t, h, a.ID)
	cfg := confirmed.ExistingConfiguration
	if cfg == nil {
		t.Fatal("existingConfiguration missing")
	}
	if cfg.SelectedRole != nil {
		t.Errorf("selectedRole = %v, want null (slot lost)", *cfg.SelectedRole)
	}
	if cfg.Ready {
		t.Error("ready must be false after losing the role slot")
	}
	if cfg.PlayerName != "Alice" || len(cfg.SelectedSpecialties) != 1 {
		t.Errorf("preserved data wrong: %+v", cfg)
	}

	// Unready again → receives availability; detective shows unavailable
	// (2 connected → capacity 1, occupied by Bob).
	ra := payloadAs[protocol.RolesAvailable](t, a2.Expect(protocol.SetupToPlayerRolesAvailable))
	for _, r := range ra.Roles {
		if r.RoleType == "detective" && r.Available {
			t.Error("detective should be unavailable after Bob took the slot")
		}
	}

	// Reselecting a different role works.
	a2.Configure("janitor", "Alice")
	for {
		lobby := payloadAs[protocol.LobbyStatus](t, a2.Expect(protocol.SetupToClientLobbyStatus))
		if lobby.PlayerRoles["janitor"] == 1 && lobby.ReadyPlayers == 2 {
			break
		}
	}
}

// A new connection presenting the valid host UUID supersedes the old host
// socket, which is closed with 1000 (the no-auto-reconnect code).
func TestHostSupersede(t *testing.T) {
	h := Start(t, nil, nil)
	host1, confirmed1 := ConnectHost(t, h)
	if confirmed1.IsReconnection {
		t.Error("first host connection must not be a reconnection")
	}
	p := ConnectNewPlayer(t, h)
	p.Expect(protocol.SetupToPlayerRolesAvailable)

	host2, confirmed2 := ConnectHost(t, h)
	if !confirmed2.IsReconnection {
		t.Error("superseding connection must be a reconnection")
	}
	host1.ExpectClose(protocol.CloseNormal)

	// Players are told the host is back.
	reconnected := payloadAs[protocol.HostReconnected](t, p.Expect(protocol.SystemToClientHostReconnected))
	if reconnected.CurrentPhase != protocol.PhaseSetup {
		t.Errorf("hostReconnected = %+v", reconnected)
	}

	// The new socket receives the setup replay (roster) and works normally.
	host2.Expect(protocol.SetupToHostPlayerRoster)
}

func TestHostDisconnectNotifiesPlayers(t *testing.T) {
	h := Start(t, nil, nil)
	host, _ := ConnectHost(t, h)
	p := ConnectNewPlayer(t, h)
	p.Expect(protocol.SetupToPlayerRolesAvailable)

	host.Close()

	notice := payloadAs[protocol.HostDisconnected](t, p.Expect(protocol.SystemToClientHostDisconnected))
	if notice.CurrentPhase != protocol.PhaseSetup {
		t.Errorf("currentPhase = %s", notice.CurrentPhase)
	}
	if !notice.GameImpact.CanContinue {
		t.Error("setup host disconnect must be canContinue=true")
	}
	if notice.TimerPausedAt != "" {
		t.Error("timerPausedAt must be absent outside puzzle_assembly")
	}
}
