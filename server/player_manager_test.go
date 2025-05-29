package main

import (
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

func TestNewPlayerManager(t *testing.T) {
	pm := NewPlayerManager()

	assert.NotNil(t, pm)
	assert.NotNil(t, pm.players)
	assert.Empty(t, pm.players)
}

func TestCreatePlayer(t *testing.T) {
	pm := NewPlayerManager()

	tests := []struct {
		name   string
		isHost bool
	}{
		{
			name:   "Create regular player",
			isHost: false,
		},
		{
			name:   "Create host player",
			isHost: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For simplicity, we'll pass nil connection since WebSocket mocking is complex
			player := pm.CreatePlayer(nil, tt.isHost)

			assert.NotNil(t, player)
			assert.NotEmpty(t, player.ID)
			assert.Equal(t, tt.isHost, player.IsHost)
			assert.Equal(t, StateConnected, player.State)
			assert.Empty(t, player.Role)
			assert.Empty(t, player.Specialties)
			assert.False(t, player.Ready)

			// Verify player was added to manager
			storedPlayer, err := pm.GetPlayer(player.ID)
			assert.NoError(t, err)
			assert.Equal(t, player.ID, storedPlayer.ID)
		})
	}
}

func TestPlayerManagerGetPlayer(t *testing.T) {
	pm := NewPlayerManager()

	// Create a test player
	player := pm.CreatePlayer(nil, false)
	playerID := player.ID

	tests := []struct {
		name     string
		playerID string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "Get existing player",
			playerID: playerID,
			wantErr:  false,
		},
		{
			name:     "Get non-existent player",
			playerID: uuid.New().String(),
			wantErr:  true,
			errMsg:   "player not found",
		},
		{
			name:     "Get with empty ID",
			playerID: "",
			wantErr:  true,
			errMsg:   "player not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			player, err := pm.GetPlayer(tt.playerID)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, player)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, player)
				assert.Equal(t, tt.playerID, player.ID)
			}
		})
	}
}

func TestPlayerManagerGetCounts(t *testing.T) {
	pm := NewPlayerManager()

	// Initial state
	assert.Equal(t, 0, pm.GetPlayerCount())
	assert.Equal(t, 0, pm.GetConnectedCount())
	assert.Equal(t, 0, pm.GetReadyCount())
	assert.Equal(t, 0, pm.GetNonHostPlayerCount())

	// Add host
	host := pm.CreatePlayer(nil, true)
	assert.Equal(t, 1, pm.GetPlayerCount())
	assert.Equal(t, 1, pm.GetConnectedCount())
	assert.Equal(t, 0, pm.GetReadyCount())
	assert.Equal(t, 0, pm.GetNonHostPlayerCount())

	// Add regular player
	player1 := pm.CreatePlayer(nil, false)
	assert.Equal(t, 2, pm.GetPlayerCount())
	assert.Equal(t, 2, pm.GetConnectedCount())
	assert.Equal(t, 0, pm.GetReadyCount())
	assert.Equal(t, 1, pm.GetNonHostPlayerCount())

	// Set player ready
	err := pm.SetPlayerReady(player1.ID, true)
	assert.NoError(t, err)
	assert.Equal(t, 1, pm.GetReadyCount())

	// Disconnect player
	err = pm.DisconnectPlayer(player1.ID)
	assert.NoError(t, err)
	assert.Equal(t, 2, pm.GetPlayerCount())    // Still in system
	assert.Equal(t, 1, pm.GetConnectedCount()) // But not connected

	// Verify host operations
	assert.True(t, pm.IsHostConnected())
	assert.Equal(t, host.ID, pm.GetHost().ID)
}

func TestPlayerManagerRoleManagement(t *testing.T) {
	pm := NewPlayerManager()

	// Create players
	players := make([]*Player, 5)
	for i := 0; i < 5; i++ {
		players[i] = pm.CreatePlayer(nil, false)
	}

	// Check initial available roles
	roles := pm.GetAvailableRoles()
	assert.Len(t, roles, 4)
	for _, role := range roles {
		assert.True(t, role.Available)
		assert.Equal(t, 1.5, role.ResourceBonus)
	}

	// Assign roles
	assert.NoError(t, pm.SetPlayerRole(players[0].ID, "art_enthusiast"))
	assert.NoError(t, pm.SetPlayerRole(players[1].ID, "art_enthusiast"))
	assert.NoError(t, pm.SetPlayerRole(players[2].ID, "detective"))
	assert.NoError(t, pm.SetPlayerRole(players[3].ID, "tourist"))
	assert.NoError(t, pm.SetPlayerRole(players[4].ID, "janitor"))

	// Check role distribution
	distribution := pm.GetRoleDistribution()
	assert.Equal(t, 2, distribution["art_enthusiast"])
	assert.Equal(t, 1, distribution["detective"])
	assert.Equal(t, 1, distribution["tourist"])
	assert.Equal(t, 1, distribution["janitor"])

	// Check available roles after assignment
	roles = pm.GetAvailableRoles()
	for _, role := range roles {
		if role.Role == "art_enthusiast" && len(players) <= 8 {
			// With 5 players, max per role is (5+3)/4 = 2
			assert.False(t, role.Available)
		} else {
			assert.True(t, role.Available)
		}
	}

	// Test invalid role
	err := pm.SetPlayerRole(players[0].ID, "invalid_role")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid role")

	// Test setting role for non-existent player
	err = pm.SetPlayerRole(uuid.New().String(), "detective")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "player not found")
}

func TestPlayerManagerSpecialtyManagement(t *testing.T) {
	pm := NewPlayerManager()
	player := pm.CreatePlayer(nil, false)

	tests := []struct {
		name        string
		specialties []string
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "Valid single specialty",
			specialties: []string{"science"},
			wantErr:     false,
		},
		{
			name:        "Valid two specialties",
			specialties: []string{"history", "geography"},
			wantErr:     false,
		},
		{
			name:        "Too many specialties",
			specialties: []string{"science", "history", "geography"},
			wantErr:     true,
			errMsg:      "must select 1-2 specialties",
		},
		{
			name:        "Empty specialties",
			specialties: []string{},
			wantErr:     true,
			errMsg:      "must select 1-2 specialties",
		},
		{
			name:        "Invalid specialty",
			specialties: []string{"magic"},
			wantErr:     true,
			errMsg:      "invalid specialty",
		},
		{
			name:        "Duplicate specialties",
			specialties: []string{"science", "science"},
			wantErr:     true,
			errMsg:      "duplicate specialty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := pm.SetPlayerSpecialties(player.ID, tt.specialties)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
				// Verify specialties were set
				p, _ := pm.GetPlayer(player.ID)
				assert.Equal(t, tt.specialties, p.Specialties)
				// Player should be ready after specialty selection
				assert.True(t, p.Ready)
			}
		})
	}
}

func TestPlayerManagerLocationTracking(t *testing.T) {
	pm := NewPlayerManager()
	player := pm.CreatePlayer(nil, false)

	// Test valid location hashes
	validHashes := []string{
		"HASH_ANCHOR_STATION_2025",
		"HASH_CHRONOS_STATION_2025",
		"HASH_GUIDE_STATION_2025",
		"HASH_CLARITY_STATION_2025",
	}

	for _, hash := range validHashes {
		err := pm.UpdatePlayerLocation(player.ID, hash)
		assert.NoError(t, err)

		location, err := pm.GetPlayerLocation(player.ID)
		assert.NoError(t, err)
		assert.Equal(t, hash, location)
	}

	// Test invalid location hash
	err := pm.UpdatePlayerLocation(player.ID, "INVALID_HASH")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid resource station hash")

	// Test non-existent player
	err = pm.UpdatePlayerLocation(uuid.New().String(), validHashes[0])
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "player not found")
}

func TestPlayerManagerDisconnectReconnect(t *testing.T) {
	pm := NewPlayerManager()

	// Create player
	player := pm.CreatePlayer(nil, false)
	playerID := player.ID

	// Set some state
	assert.NoError(t, pm.SetPlayerRole(playerID, "detective"))
	assert.NoError(t, pm.SetPlayerSpecialties(playerID, []string{"science", "history"}))

	// Disconnect player
	err := pm.DisconnectPlayer(playerID)
	assert.NoError(t, err)

	// Verify state after disconnect
	p, err := pm.GetPlayer(playerID)
	assert.NoError(t, err)
	assert.Equal(t, StateDisconnected, p.State)
	assert.Equal(t, "detective", p.Role)                           // Role preserved
	assert.Equal(t, []string{"science", "history"}, p.Specialties) // Specialties preserved

	// Reconnect player
	newConn := &websocket.Conn{} // Mock connection
	err = pm.ReconnectPlayer(playerID, newConn)
	assert.NoError(t, err)

	// Verify state after reconnect
	p, err = pm.GetPlayer(playerID)
	assert.NoError(t, err)
	assert.Equal(t, StateConnected, p.State)
	assert.Equal(t, newConn, p.Connection)

	// Test reconnect non-existent player
	err = pm.ReconnectPlayer(uuid.New().String(), newConn)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "player not found")
}

func TestPlayerManagerGetPlayerLists(t *testing.T) {
	pm := NewPlayerManager()

	// Create mix of players
	host := pm.CreatePlayer(nil, true)
	player1 := pm.CreatePlayer(nil, false)
	player2 := pm.CreatePlayer(nil, false)
	player3 := pm.CreatePlayer(nil, false)

	// Set different states
	pm.SetPlayerReady(player1.ID, true)
	pm.SetPlayerReady(player2.ID, true)
	pm.DisconnectPlayer(player3.ID)

	// Test GetAllPlayers
	allPlayers := pm.GetAllPlayers()
	assert.Len(t, allPlayers, 4)

	// Test GetConnectedPlayers
	connectedPlayers := pm.GetConnectedPlayers()
	assert.Len(t, connectedPlayers, 3) // host + player1 + player2

	// Test GetReadyPlayers
	readyPlayers := pm.GetReadyPlayers()
	assert.Len(t, readyPlayers, 2) // player1 + player2

	// Test GetConnectedNonHostPlayers
	nonHostPlayers := pm.GetConnectedNonHostPlayers()
	assert.Len(t, nonHostPlayers, 2) // player1 + player2

	// Test GetReadyNonHostPlayers
	readyNonHost := pm.GetReadyNonHostPlayers()
	assert.Len(t, readyNonHost, 2) // player1 + player2

	// Verify host is not in non-host lists
	for _, p := range nonHostPlayers {
		assert.NotEqual(t, host.ID, p.ID)
	}
}

func TestPlayerManagerConcurrency(t *testing.T) {
	pm := NewPlayerManager()

	// Create players concurrently
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			player := pm.CreatePlayer(nil, false)
			assert.NotNil(t, player)
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all players were created
	assert.Equal(t, 10, pm.GetPlayerCount())

	// Test concurrent reads and writes
	players := pm.GetAllPlayers()
	done2 := make(chan bool, len(players)*2)

	// Concurrent reads
	for _, p := range players {
		go func(playerID string) {
			defer func() { done2 <- true }()
			player, err := pm.GetPlayer(playerID)
			assert.NoError(t, err)
			assert.NotNil(t, player)
		}(p.ID)
	}

	// Concurrent writes
	roles := []string{"art_enthusiast", "detective", "tourist", "janitor"}
	for i, p := range players {
		go func(playerID string, idx int) {
			defer func() { done2 <- true }()
			role := roles[idx%len(roles)]
			err := pm.SetPlayerRole(playerID, role)
			// Some may fail due to role limits, but should not panic
			_ = err
		}(p.ID, i)
	}

	// Wait for all operations
	for i := 0; i < len(players)*2; i++ {
		<-done2
	}

	// Verify state is consistent
	finalCount := pm.GetPlayerCount()
	assert.Equal(t, 10, finalCount)
}
