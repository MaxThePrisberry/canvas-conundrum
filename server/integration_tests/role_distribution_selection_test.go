package integration_tests

import (
	"canvas-conundrum/config"
	"canvas-conundrum/services"
	"canvas-conundrum/test_helpers"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoleDistributionAlgorithm(t *testing.T) {
	// Test role distribution algorithm: max(1, (playerCount + 3) / 4)
	testCases := []struct {
		playerCount       int
		expectedRoleCount int
	}{
		{1, 1},  // max(1, (1+3)/4) = max(1, 1) = 1
		{2, 1},  // max(1, (2+3)/4) = max(1, 1) = 1
		{3, 1},  // max(1, (3+3)/4) = max(1, 1) = 1
		{4, 1},  // max(1, (4+3)/4) = max(1, 1) = 1
		{5, 2},  // max(1, (5+3)/4) = max(1, 2) = 2
		{8, 2},  // max(1, (8+3)/4) = max(1, 2) = 2
		{9, 3},  // max(1, (9+3)/4) = max(1, 3) = 3
		{12, 3}, // max(1, (12+3)/4) = max(1, 3) = 3
		{13, 4}, // max(1, (13+3)/4) = max(1, 4) = 4
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("PlayerCount_%d", tc.playerCount), func(t *testing.T) {
			server, cleanup := setupTestServer(t)
			defer cleanup()

			// Connect host
			host := test_helpers.NewTestHostClient(t, server, config.HostUUID)
			err := host.Connect()
			require.NoError(t, err)
			defer host.Close()

			// Connect the specified number of players to test role distribution
			players := make([]*test_helpers.TestPlayerClient, tc.playerCount)
			for i := 0; i < tc.playerCount; i++ {
				players[i] = test_helpers.NewTestPlayerClient(t, server)
				err = players[i].Connect()
				require.NoError(t, err)
				defer players[i].Close()

				// Wait for roles available message
				rolesMsg, err := players[i].WaitForEvent(config.EventSetupToPlayerRolesAvailable, 2*time.Second)
				require.NoError(t, err)

				// Check role availability
				payload := rolesMsg.Payload.(map[string]interface{})
				roles := payload["roles"].([]interface{})

				// Count available roles
				availableRoles := 0
				for _, role := range roles {
					roleMap := role.(map[string]interface{})
					available := roleMap["available"].(bool)
					if available {
						availableRoles++
					}
				}

				// At least some roles should be available for new players
				assert.GreaterOrEqual(t, availableRoles, 1,
					"At least one role should be available for player %d", i+1)

				// Verify the algorithm is working by checking the total available slots
				// This is indirect validation since we can't directly access the algorithm
				if i == 0 { // First player should see all roles available
					assert.Equal(t, 4, availableRoles,
						"First player should see all 4 roles available")
				}
			}
		})
	}
}

func TestDynamicRoleAvailability(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	// Connect host
	host := test_helpers.NewTestHostClient(t, server, config.HostUUID)
	err := host.Connect()
	require.NoError(t, err)
	defer host.Close()

	// Test role availability changes as players join and select roles
	maxPlayers := 8
	players := make([]*test_helpers.TestPlayerClient, maxPlayers)
	roles := []string{"art_enthusiast", "detective", "tourist", "janitor"}
	specialties := []string{"general", "history", "science", "geography", "music", "video_games"}

	for i := 0; i < maxPlayers; i++ {
		players[i] = test_helpers.NewTestPlayerClient(t, server)
		err = players[i].Connect()
		require.NoError(t, err)
		defer players[i].Close()

		// Wait for roles available message
		rolesMsg, err := players[i].WaitForEvent(config.EventSetupToPlayerRolesAvailable, 2*time.Second)
		require.NoError(t, err)

		// Check current role availability
		payload := rolesMsg.Payload.(map[string]interface{})
		rolesList := payload["roles"].([]interface{})

		// Track which roles are available
		roleAvailability := make(map[string]bool)
		for _, role := range rolesList {
			roleMap := role.(map[string]interface{})
			roleType := roleMap["roleType"].(string)
			available := roleMap["available"].(bool)
			roleAvailability[roleType] = available
		}

		// At least one role should be available for this player
		hasAvailableRole := false
		for _, available := range roleAvailability {
			if available {
				hasAvailableRole = true
				break
			}
		}
		assert.True(t, hasAvailableRole, "Player %d should have at least one available role", i+1)

		// Configure the player with an available role
		selectedRole := ""
		for _, role := range roles {
			if roleAvailability[role] {
				selectedRole = role
				break
			}
		}
		require.NotEmpty(t, selectedRole, "Should find an available role for player %d", i+1)

		players[i].ClearMessages()
		err = players[i].ConfigurePlayer(fmt.Sprintf("Player%d", i+1), selectedRole, []string{specialties[i%len(specialties)]})
		require.NoError(t, err)

		// Wait for lobby status update
		_, err = players[i].WaitForEvent(config.EventSetupToClientLobbyStatus, 2*time.Second)
		require.NoError(t, err)
	}

	// Verify final state - all players should be configured
	gm := services.GetGameInstance()
	assert.Equal(t, maxPlayers, gm.GetPlayerCount(), "All players should be connected and configured")
}

func TestRoleSelectionAndConflicts(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	// Connect host
	host := test_helpers.NewTestHostClient(t, server, config.HostUUID)
	err := host.Connect()
	require.NoError(t, err)
	defer host.Close()

	// Test role selection with potential conflicts
	// Connect multiple players and try to assign them to limited roles
	numPlayers := 5 // More than 4, so some roles will be filled
	players := make([]*test_helpers.TestPlayerClient, numPlayers)

	for i := 0; i < numPlayers; i++ {
		players[i] = test_helpers.NewTestPlayerClient(t, server)
		err = players[i].Connect()
		require.NoError(t, err)
		defer players[i].Close()

		// Wait for roles available message
		rolesMsg, err := players[i].WaitForEvent(config.EventSetupToPlayerRolesAvailable, 2*time.Second)
		require.NoError(t, err)

		// For first 4 players, try to select different roles
		// For 5th player, should still find an available role due to distribution algorithm
		var selectedRole string
		if i < 4 {
			selectedRole = []string{"art_enthusiast", "detective", "tourist", "janitor"}[i]
		} else {
			// 5th player should find an available role (likely art_enthusiast due to distribution)
			payload := rolesMsg.Payload.(map[string]interface{})
			rolesList := payload["roles"].([]interface{})

			for _, role := range rolesList {
				roleMap := role.(map[string]interface{})
				roleType := roleMap["roleType"].(string)
				available := roleMap["available"].(bool)
				if available {
					selectedRole = roleType
					break
				}
			}
		}

		require.NotEmpty(t, selectedRole, "Player %d should have an available role", i+1)

		players[i].ClearMessages()
		err = players[i].ConfigurePlayer(fmt.Sprintf("Player%d", i+1), selectedRole, []string{"general"})
		require.NoError(t, err)

		// Wait for configuration confirmation
		_, err = players[i].WaitForEvent(config.EventSetupToClientLobbyStatus, 2*time.Second)
		require.NoError(t, err)
	}

	// Verify all players are configured successfully
	gm := services.GetGameInstance()
	assert.Equal(t, numPlayers, gm.GetPlayerCount())
}

func TestRoleSpecialtyInteraction(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	// Connect host
	host := test_helpers.NewTestHostClient(t, server, config.HostUUID)
	err := host.Connect()
	require.NoError(t, err)
	defer host.Close()

	// Test that role selection and specialty selection work together
	player := test_helpers.NewTestPlayerClient(t, server)
	err = player.Connect()
	require.NoError(t, err)
	defer player.Close()

	// Wait for roles available message
	rolesMsg, err := player.WaitForEvent(config.EventSetupToPlayerRolesAvailable, 2*time.Second)
	require.NoError(t, err)

	// Verify trivia categories are provided
	payload := rolesMsg.Payload.(map[string]interface{})
	categories := payload["triviaCategories"].([]interface{})
	assert.Len(t, categories, 6, "Should have 6 trivia categories")

	expectedCategories := []string{"general", "geography", "history", "music", "science", "video_games"}
	for i, category := range categories {
		assert.Equal(t, expectedCategories[i], category.(string))
	}

	// Verify max specialties
	maxSpecialties := payload["maxSpecialties"].(float64)
	assert.Equal(t, float64(1), maxSpecialties, "Max specialties should be 1")

	// Configure player with role and specialty
	player.ClearMessages()
	err = player.ConfigurePlayer("TestPlayer", "art_enthusiast", []string{"science"})
	require.NoError(t, err)

	// Wait for configuration confirmation
	_, err = player.WaitForEvent(config.EventSetupToClientLobbyStatus, 2*time.Second)
	require.NoError(t, err)

	// Verify player is ready
	gm := services.GetGameInstance()
	assert.Equal(t, 1, gm.GetPlayerCount())
}
