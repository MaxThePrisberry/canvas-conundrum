package integration_tests

import (
	"canvas-conundrum/config"
	"canvas-conundrum/handlers"
	"canvas-conundrum/services"
	"canvas-conundrum/test_helpers"
	"fmt"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupQRStationTestServer creates a server for QR station system testing
func setupQRStationTestServer(t *testing.T) (*httptest.Server, *test_helpers.TestHostClient, func()) {
	// Reset game manager
	gm := services.GetGameInstance()
	gm.Cleanup()
	gm.ResetGame()

	// Setup all services
	gm.SetBroadcastService(services.NewBroadcastService())

	triviaService := services.NewTriviaService()
	createTestTriviaFiles(t)
	err := triviaService.LoadQuestions()
	require.NoError(t, err)
	gm.SetTriviaService(triviaService)

	gm.SetPuzzleService(services.NewPuzzleService())
	gm.SetAnalyticsService(services.NewAnalyticsService())

	// Create router
	r := mux.NewRouter()
	r.HandleFunc("/ws", handlers.HandlePlayerWebSocket).Methods("GET")
	r.HandleFunc("/ws/host/{uuid}", handlers.HandleHostWebSocket).Methods("GET")

	// Create test server
	server := httptest.NewServer(r)

	// Create and connect host
	hostClient := test_helpers.NewTestHostClient(t, server, config.HostUUID)
	err = hostClient.Connect()
	require.NoError(t, err)

	_, err = hostClient.WaitForEvent(config.EventSetupToHostConnectionConfirmed, 2*time.Second)
	require.NoError(t, err)

	cleanup := func() {
		hostClient.Close()
		server.Close()
		gm.Cleanup()
		gm.ResetGame()
		os.RemoveAll("./trivia")
	}

	return server, hostClient, cleanup
}

// createTestTriviaFiles creates trivia files for testing

// createAndConfigurePlayer creates a player with specified role and specialty

// TestQRStationHashValidation tests the QR code hash validation system
func TestQRStationHashValidation(t *testing.T) {
	server, hostClient, cleanup := setupQRStationTestServer(t)
	defer cleanup()

	// Create players with different roles for testing different station bonuses
	players := make([]*test_helpers.TestPlayerClient, 4)
	roles := []string{"janitor", "tourist", "detective", "art_enthusiast"}
	expectedStations := []string{"anchor", "chronos", "guide", "clarity"}

	for i := 0; i < 4; i++ {
		players[i] = createAndConfigurePlayer(t, server,
			fmt.Sprintf("Player%d", i+1), roles[i], []string{"science"})
		defer players[i].Close()
	}

	// Start game
	err := hostClient.StartGame("medium")
	require.NoError(t, err)

	// Wait for resource phase start
	for i := 0; i < 4; i++ {
		phaseMsg, err := players[i].WaitForEvent(config.EventResourceToClientPhaseStart, 3*time.Second)
		require.NoError(t, err)

		// Verify station hashes are provided
		phasePayload := phaseMsg.Payload.(map[string]interface{})
		stationHashes := phasePayload["resourceStationHashes"].(map[string]interface{})

		// Verify all 4 station hashes are provided
		assert.Contains(t, stationHashes, "anchor", "Should provide anchor station hash")
		assert.Contains(t, stationHashes, "chronos", "Should provide chronos station hash")
		assert.Contains(t, stationHashes, "guide", "Should provide guide station hash")
		assert.Contains(t, stationHashes, "clarity", "Should provide clarity station hash")

		// Verify hashes match constants
		assert.Equal(t, config.HashAnchorStation, stationHashes["anchor"].(string))
		assert.Equal(t, config.HashChronosStation, stationHashes["chronos"].(string))
		assert.Equal(t, config.HashGuideStation, stationHashes["guide"].(string))
		assert.Equal(t, config.HashClarityStation, stationHashes["clarity"].(string))

		// Verify all hashes are different
		hashes := []string{
			stationHashes["anchor"].(string),
			stationHashes["chronos"].(string),
			stationHashes["guide"].(string),
			stationHashes["clarity"].(string),
		}

		for j := 0; j < len(hashes); j++ {
			for k := j + 1; k < len(hashes); k++ {
				assert.NotEqual(t, hashes[j], hashes[k],
					"Station hashes should be unique: %s vs %s", hashes[j], hashes[k])
			}
		}
	}

	t.Run("ValidHashVerification", func(t *testing.T) {
		gm := services.GetGameInstance()
		_ = gm.GetGame()

		// Test each player verifying their role's optimal station
		for i, player := range players {
			playerID := player.GetPlayerID()
			playerObj, exists := gm.GetPlayer(playerID)
			require.True(t, exists)
			require.NotNil(t, playerObj)

			stationName := expectedStations[i]
			stationHash := getStationHashByName(stationName)

			// Verify location initially unknown
			assert.Empty(t, playerObj.CurrentStation, "Player should start with unknown location")

			// Send location verification
			err := player.VerifyLocation(stationName, stationHash)
			require.NoError(t, err)

			// Wait for verification to be processed
			time.Sleep(200 * time.Millisecond)

			// Verify player location was updated
			playerObj, exists = gm.GetPlayer(playerID) // Refresh player object
			require.True(t, exists)
			assert.Equal(t, stationName, playerObj.CurrentStation,
				"Player %d should be at %s station", i+1, stationName)

			// Verify host receives station distribution update
			analyticsMsg, err := hostClient.WaitForEvent(config.EventResourceToHostRoundAnalytics, 3*time.Second)
			if err == nil { // May not always trigger depending on timing
				analyticsPayload := analyticsMsg.Payload.(map[string]interface{})
				stationDistribution := analyticsPayload["stationDistribution"].(map[string]interface{})

				// Verify this station has at least 1 player
				stationCount := int(stationDistribution[stationName].(float64))
				assert.GreaterOrEqual(t, stationCount, 1,
					"Station %s should have at least 1 player", stationName)
			}
		}
	})

	t.Run("InvalidHashRejection", func(t *testing.T) {
		gm := services.GetGameInstance()
		_ = gm.GetGame()

		player := players[0]
		playerID := player.GetPlayerID()
		playerObj, exists := gm.GetPlayer(playerID)
		require.True(t, exists)
		require.NotNil(t, playerObj)

		// Record initial location
		initialLocation := playerObj.CurrentStation

		// Try to verify with invalid hash
		invalidHash := "invalid_hash_12345"

		// Send invalid location verification
		err := player.VerifyLocation("anchor", invalidHash)
		require.NoError(t, err) // Message sending should succeed

		// Wait for processing
		time.Sleep(200 * time.Millisecond)

		// Verify player location was NOT updated
		playerObj, exists = gm.GetPlayer(playerID) // Refresh player object
		require.True(t, exists)
		assert.Equal(t, initialLocation, playerObj.CurrentStation,
			"Player location should not change with invalid hash")

		// Player might receive an error message (depending on implementation)
		// This would be tested in error handling tests
	})

	t.Run("StationSwitching", func(t *testing.T) {
		gm := services.GetGameInstance()
		_ = gm.GetGame()

		player := players[0]
		playerID := player.GetPlayerID()

		// Test switching between different stations
		stations := []struct {
			name string
			hash string
		}{
			{"anchor", config.HashAnchorStation},
			{"chronos", config.HashChronosStation},
			{"guide", config.HashGuideStation},
			{"clarity", config.HashClarityStation},
		}

		for _, station := range stations {
			// Verify location at new station
			err := player.VerifyLocation(station.name, station.hash)
			require.NoError(t, err)

			time.Sleep(200 * time.Millisecond)

			// Verify player location was updated
			playerObj, exists := gm.GetPlayer(playerID)
			require.True(t, exists)
			require.NotNil(t, playerObj)
			assert.Equal(t, station.name, playerObj.CurrentStation,
				"Player should be at %s station", station.name)
		}
	})
}

// TestLocationVerificationWorkflow tests the complete location verification workflow
func TestLocationVerificationWorkflow(t *testing.T) {
	server, hostClient, cleanup := setupQRStationTestServer(t)
	defer cleanup()

	// Create players
	players := make([]*test_helpers.TestPlayerClient, 4)
	roles := []string{"janitor", "tourist", "detective", "art_enthusiast"}

	for i := 0; i < 4; i++ {
		players[i] = createAndConfigurePlayer(t, server,
			fmt.Sprintf("Player%d", i+1), roles[i], []string{"science"})
		defer players[i].Close()
	}

	// Start game
	err := hostClient.StartGame("medium")
	require.NoError(t, err)

	// Wait for resource phase
	for i := 0; i < 4; i++ {
		_, err = players[i].WaitForEvent(config.EventResourceToClientPhaseStart, 3*time.Second)
		require.NoError(t, err)
	}

	t.Run("LocationVerificationDuringResourceGathering", func(t *testing.T) {
		gm := services.GetGameInstance()
		_ = gm.GetGame()

		// Have each player verify their location before answering trivia
		stationAssignments := map[int]string{
			0: "anchor",  // janitor
			1: "chronos", // tourist
			2: "guide",   // detective
			3: "clarity", // art_enthusiast
		}

		for i, player := range players {
			stationName := stationAssignments[i]
			stationHash := getStationHashByName(stationName)

			// Verify location
			err := player.VerifyLocation(stationName, stationHash)
			require.NoError(t, err)

			time.Sleep(200 * time.Millisecond)

			// Wait for trivia question
			triviaMsg, err := player.WaitForEvent(config.EventResourceToPlayerTriviaQuestion, 5*time.Second)
			require.NoError(t, err)

			triviaPayload := triviaMsg.Payload.(map[string]interface{})
			questionID := triviaPayload["questionId"].(string)

			// Answer correctly
			err = player.AnswerTrivia(questionID, 0, 15.0)
			require.NoError(t, err)

			// Wait for answer result
			resultMsg, err := player.WaitForEvent(config.EventResourceToPlayerAnswerResult, 3*time.Second)
			require.NoError(t, err)

			resultPayload := resultMsg.Payload.(map[string]interface{})

			// Verify location is reported in answer result
			currentLocation := resultPayload["currentLocation"].(string)
			assert.Equal(t, stationName, currentLocation,
				"Answer result should report current location")

			// Verify role bonus was applied (player is at their optimal station)
			bonuses := resultPayload["bonuses"].(map[string]interface{})
			roleBonus := bonuses["roleBonus"].(bool)
			assert.True(t, roleBonus, "Should receive role bonus at optimal station")

			roleBonusTokens := int(bonuses["roleBonusTokens"].(float64))
			assert.Greater(t, roleBonusTokens, 0, "Role bonus tokens should be positive")
		}
	})

	t.Run("LocationRequiredOnlyForChanges", func(t *testing.T) {
		// According to spec: "Location verification only required when changing stations"
		gm := services.GetGameInstance()
		_ = gm.GetGame()

		player := players[0]
		playerID := player.GetPlayerID()

		// Set initial location
		err := player.VerifyLocation("anchor", config.HashAnchorStation)
		require.NoError(t, err)

		time.Sleep(200 * time.Millisecond)

		// Answer multiple trivia questions without re-verifying location
		for round := 0; round < 3; round++ {
			triviaMsg, err := player.WaitForEvent(config.EventResourceToPlayerTriviaQuestion, 5*time.Second)
			if err != nil {
				break // May run out of rounds
			}

			triviaPayload := triviaMsg.Payload.(map[string]interface{})
			questionID := triviaPayload["questionId"].(string)

			// Answer without sending location verification
			err = player.AnswerTrivia(questionID, 0, 15.0)
			require.NoError(t, err)

			resultMsg, err := player.WaitForEvent(config.EventResourceToPlayerAnswerResult, 3*time.Second)
			require.NoError(t, err)

			resultPayload := resultMsg.Payload.(map[string]interface{})

			// Player should still be at anchor station
			currentLocation := resultPayload["currentLocation"].(string)
			assert.Equal(t, "anchor", currentLocation,
				"Player should remain at anchor station without re-verification")

			// Should still receive role bonus
			bonuses := resultPayload["bonuses"].(map[string]interface{})
			roleBonus := bonuses["roleBonus"].(bool)
			assert.True(t, roleBonus, "Should continue receiving role bonus")
		}

		// Now switch to different station
		err = player.VerifyLocation("chronos", config.HashChronosStation)
		require.NoError(t, err)

		time.Sleep(200 * time.Millisecond)

		// Verify location changed
		playerObj, exists := gm.GetPlayer(playerID)
		require.True(t, exists)
		assert.Equal(t, "chronos", playerObj.CurrentStation, "Player should be at chronos station")
	})
}

// TestStationDistributionTracking tests that the host can track player distribution across stations
func TestStationDistributionTracking(t *testing.T) {
	server, hostClient, cleanup := setupQRStationTestServer(t)
	defer cleanup()

	// Create 8 players to test distribution tracking
	players := make([]*test_helpers.TestPlayerClient, 8)
	roles := []string{"janitor", "tourist", "detective", "art_enthusiast",
		"janitor", "tourist", "detective", "art_enthusiast"}

	for i := 0; i < 8; i++ {
		players[i] = createAndConfigurePlayer(t, server,
			fmt.Sprintf("Player%d", i+1), roles[i], []string{"science"})
		defer players[i].Close()
	}

	// Start game
	err := hostClient.StartGame("medium")
	require.NoError(t, err)

	// Wait for resource phase
	for i := 0; i < 8; i++ {
		_, err = players[i].WaitForEvent(config.EventResourceToClientPhaseStart, 3*time.Second)
		require.NoError(t, err)
	}

	t.Run("InitialDistribution", func(t *testing.T) {
		// Initially all players should be at "unknown" location
		hostPhaseMsg, err := hostClient.WaitForEvent(config.EventResourceToHostPhaseStart, 3*time.Second)
		require.NoError(t, err)

		hostPayload := hostPhaseMsg.Payload.(map[string]interface{})
		monitoringDashboard := hostPayload["monitoringDashboard"].(map[string]interface{})
		playerDistribution := monitoringDashboard["playerDistribution"].(map[string]interface{})

		// All players should start as unknown
		assert.Equal(t, float64(8), playerDistribution["unknown"].(float64))
		assert.Equal(t, float64(0), playerDistribution["anchor"].(float64))
		assert.Equal(t, float64(0), playerDistribution["chronos"].(float64))
		assert.Equal(t, float64(0), playerDistribution["guide"].(float64))
		assert.Equal(t, float64(0), playerDistribution["clarity"].(float64))
	})

	t.Run("DistributionAfterLocationVerification", func(t *testing.T) {
		// Have players verify locations at different stations
		stationAssignments := []string{
			"anchor", "anchor", // 2 at anchor
			"chronos", "chronos", "chronos", // 3 at chronos
			"guide",              // 1 at guide
			"clarity", "clarity", // 2 at clarity
		}

		for i, stationName := range stationAssignments {
			stationHash := getStationHashByName(stationName)
			err := players[i].VerifyLocation(stationName, stationHash)
			require.NoError(t, err)
		}

		// Wait for distribution to be updated
		time.Sleep(500 * time.Millisecond)

		// Host should receive round analytics with updated distribution
		analyticsMsg, err := hostClient.WaitForEvent(config.EventResourceToHostRoundAnalytics, 3*time.Second)
		require.NoError(t, err)

		analyticsPayload := analyticsMsg.Payload.(map[string]interface{})
		stationDistribution := analyticsPayload["stationDistribution"].(map[string]interface{})

		// Verify distribution matches assignments
		assert.Equal(t, 2, int(stationDistribution["anchor"].(float64)))
		assert.Equal(t, 3, int(stationDistribution["chronos"].(float64)))
		assert.Equal(t, 1, int(stationDistribution["guide"].(float64)))
		assert.Equal(t, 2, int(stationDistribution["clarity"].(float64)))

		// Total should equal number of players
		total := int(stationDistribution["anchor"].(float64)) +
			int(stationDistribution["chronos"].(float64)) +
			int(stationDistribution["guide"].(float64)) +
			int(stationDistribution["clarity"].(float64))
		assert.Equal(t, 8, total, "Total distributed players should equal total players")
	})

	t.Run("DynamicDistributionTracking", func(t *testing.T) {
		// Move players between stations and verify tracking
		// Move 2 players from chronos to anchor
		err := players[2].VerifyLocation("anchor", config.HashAnchorStation)
		require.NoError(t, err)

		err = players[3].VerifyLocation("anchor", config.HashAnchorStation)
		require.NoError(t, err)

		time.Sleep(500 * time.Millisecond)

		// Check updated distribution
		analyticsMsg, err := hostClient.WaitForEvent(config.EventResourceToHostRoundAnalytics, 3*time.Second)
		require.NoError(t, err)

		analyticsPayload := analyticsMsg.Payload.(map[string]interface{})
		stationDistribution := analyticsPayload["stationDistribution"].(map[string]interface{})

		// Should now have 4 at anchor, 1 at chronos
		assert.Equal(t, 4, int(stationDistribution["anchor"].(float64)))
		assert.Equal(t, 1, int(stationDistribution["chronos"].(float64)))
		assert.Equal(t, 1, int(stationDistribution["guide"].(float64)))
		assert.Equal(t, 2, int(stationDistribution["clarity"].(float64)))
	})
}

// TestStationHashConsistency tests that station hashes remain consistent throughout the game
func TestStationHashConsistency(t *testing.T) {
	server, hostClient, cleanup := setupQRStationTestServer(t)
	defer cleanup()

	player := createAndConfigurePlayer(t, server, "TestPlayer", "detective", []string{"science"})
	defer player.Close()

	// Start game
	err := hostClient.StartGame("medium")
	require.NoError(t, err)

	// Get initial station hashes
	phaseMsg, err := player.WaitForEvent(config.EventResourceToClientPhaseStart, 3*time.Second)
	require.NoError(t, err)

	phasePayload := phaseMsg.Payload.(map[string]interface{})
	initialHashes := phasePayload["resourceStationHashes"].(map[string]interface{})

	t.Run("HashesMatchConstants", func(t *testing.T) {
		// Verify hashes match the constants defined in config
		assert.Equal(t, config.HashAnchorStation, initialHashes["anchor"].(string))
		assert.Equal(t, config.HashChronosStation, initialHashes["chronos"].(string))
		assert.Equal(t, config.HashGuideStation, initialHashes["guide"].(string))
		assert.Equal(t, config.HashClarityStation, initialHashes["clarity"].(string))
	})

	t.Run("HashesRemainConsistentAcrossReconnections", func(t *testing.T) {
		// Test that hashes don't change if host reconnects
		err := hostClient.Close()
		require.NoError(t, err)

		// Reconnect host
		newHostClient := test_helpers.NewTestHostClient(t, server, config.HostUUID)
		err = newHostClient.Connect()
		require.NoError(t, err)
		defer newHostClient.Close()

		// Player should still be able to use same hashes
		err = player.VerifyLocation("anchor", config.HashAnchorStation)
		require.NoError(t, err)

		time.Sleep(200 * time.Millisecond)

		// Verification should work with original hashes
		gm := services.GetGameInstance()
		_ = gm.GetGame()
		playerObj, exists := gm.GetPlayer(player.GetPlayerID())
		require.True(t, exists)
		assert.Equal(t, "anchor", playerObj.CurrentStation, "Location verification should work with consistent hashes")
	})

	t.Run("HashValidationAccuracy", func(t *testing.T) {
		// Test various hash validation scenarios
		testCases := []struct {
			station     string
			hash        string
			shouldWork  bool
			description string
		}{
			{"anchor", config.HashAnchorStation, true, "Correct anchor hash"},
			{"chronos", config.HashChronosStation, true, "Correct chronos hash"},
			{"guide", config.HashGuideStation, true, "Correct guide hash"},
			{"clarity", config.HashClarityStation, true, "Correct clarity hash"},
			{"anchor", config.HashChronosStation, false, "Wrong hash for anchor"},
			{"chronos", config.HashAnchorStation, false, "Wrong hash for chronos"},
			{"anchor", "invalid_hash", false, "Completely invalid hash"},
			{"unknown_station", config.HashAnchorStation, false, "Invalid station name"},
		}

		gm := services.GetGameInstance()
		_ = gm.GetGame()
		playerID := player.GetPlayerID()

		for _, tc := range testCases {
			// Record initial location
			initialPlayerObj, exists := gm.GetPlayer(playerID)
			require.True(t, exists)
			initialLocation := initialPlayerObj.CurrentStation

			// Attempt verification
			err := player.VerifyLocation(tc.station, tc.hash)
			require.NoError(t, err, "Message sending should succeed for: %s", tc.description)

			time.Sleep(200 * time.Millisecond)

			// Check if location changed
			finalPlayerObj, exists := gm.GetPlayer(playerID)
			require.True(t, exists)
			finalLocation := finalPlayerObj.CurrentStation

			if tc.shouldWork {
				assert.Equal(t, tc.station, finalLocation,
					"Location should change for valid case: %s", tc.description)
			} else {
				assert.Equal(t, initialLocation, finalLocation,
					"Location should NOT change for invalid case: %s", tc.description)
			}
		}
	})
}

// getStationHashByName returns the hash constant for a given station name
func getStationHashByName(stationName string) string {
	switch stationName {
	case "anchor":
		return config.HashAnchorStation
	case "chronos":
		return config.HashChronosStation
	case "guide":
		return config.HashGuideStation
	case "clarity":
		return config.HashClarityStation
	default:
		return ""
	}
}

// TestQRCodeTextValueValidation tests that QR codes' text values are validated as hashes
func TestQRCodeTextValueValidation(t *testing.T) {
	server, hostClient, cleanup := setupQRStationTestServer(t)
	defer cleanup()

	player := createAndConfigurePlayer(t, server, "TestPlayer", "detective", []string{"science"})
	defer player.Close()

	// Start game
	err := hostClient.StartGame("medium")
	require.NoError(t, err)

	_, err = player.WaitForEvent(config.EventResourceToClientPhaseStart, 3*time.Second)
	require.NoError(t, err)

	t.Run("QRCodeHashFormat", func(t *testing.T) {
		// According to spec: "QR codes' text value is the hash sent to the server for validation"
		// Test that the hash format is appropriate for QR codes

		stationHashes := []string{
			config.HashAnchorStation,
			config.HashChronosStation,
			config.HashGuideStation,
			config.HashClarityStation,
		}

		for _, hash := range stationHashes {
			// Verify hash is reasonable length for QR codes
			assert.Greater(t, len(hash), 8, "Hash should be reasonably long for security")
			assert.Less(t, len(hash), 200, "Hash should not be too long for QR codes")

			// Verify hash doesn't contain problematic characters for QR codes
			assert.NotContains(t, hash, " ", "Hash should not contain spaces")
			assert.NotContains(t, hash, "\n", "Hash should not contain newlines")
			assert.NotContains(t, hash, "\t", "Hash should not contain tabs")

			// Verify hash is not empty or obviously weak
			assert.NotEqual(t, "", hash, "Hash should not be empty")
			assert.NotEqual(t, "12345", hash, "Hash should not be obviously weak")
			assert.NotEqual(t, "test", hash, "Hash should not be obviously weak")
		}
	})

	t.Run("HashUniquenessAcrossStations", func(t *testing.T) {
		// All station hashes should be unique
		hashes := []string{
			config.HashAnchorStation,
			config.HashChronosStation,
			config.HashGuideStation,
			config.HashClarityStation,
		}

		for i := 0; i < len(hashes); i++ {
			for j := i + 1; j < len(hashes); j++ {
				assert.NotEqual(t, hashes[i], hashes[j],
					"Station hashes must be unique: position %d vs %d", i, j)
			}
		}
	})

	t.Run("HashValidationTiming", func(t *testing.T) {
		// Test that hash validation happens quickly (for good UX)
		gm := services.GetGameInstance()
		_ = gm.GetGame()
		playerID := player.GetPlayerID()

		start := time.Now()

		// Send location verification
		err := player.VerifyLocation("anchor", config.HashAnchorStation)
		require.NoError(t, err)

		// Wait for validation to complete
		time.Sleep(100 * time.Millisecond)

		// Check that location was updated
		playerObj, exists := gm.GetPlayer(playerID)
		require.True(t, exists)
		if playerObj.CurrentStation == "anchor" {
			duration := time.Since(start)
			assert.Less(t, duration, 1*time.Second,
				"Hash validation should complete quickly for good UX")
		}
	})
}
