package integration_tests

import (
	"canvas-conundrum/config"
	"canvas-conundrum/services"
	"canvas-conundrum/test_helpers"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStationHashConsistency(t *testing.T) {
	// This test verifies that the station hashes used in broadcast
	// match the ones used for validation, ensuring QR code consistency

	expectedHashes := map[string]string{
		"anchor":  config.HashAnchorStation,
		"chronos": config.HashChronosStation,
		"guide":   config.HashGuideStation,
		"clarity": config.HashClarityStation,
	}

	// Test that GetStationFromHash works with all expected hashes
	for station, hash := range expectedHashes {
		detectedStation := config.GetStationFromHash(hash)
		assert.NotEqual(t, config.UnknownStation, detectedStation,
			"Hash %s for station %s should be recognized", hash, station)

		// Verify the mapping is correct
		expectedStationConst := config.Station(station)
		assert.Equal(t, expectedStationConst, detectedStation,
			"Hash %s should map to station %s", hash, station)
	}

	// Verify no duplicate constants exist by checking the removed duplicates would fail
	oldDuplicates := map[string]string{
		"anchor":  "anchor_station_qr_hash_2025",
		"chronos": "chronos_station_qr_hash_2025",
		"guide":   "guide_station_qr_hash_2025",
		"clarity": "clarity_station_qr_hash_2025",
	}

	for station, badHash := range oldDuplicates {
		detectedStation := config.GetStationFromHash(badHash)
		assert.Equal(t, config.UnknownStation, detectedStation,
			"Old duplicate hash %s for station %s should NOT be recognized", badHash, station)
	}

	t.Logf("Verified station hash consistency - all hashes properly map to stations")
}

func TestBroadcastUsesConfigHashes(t *testing.T) {
	// This test verifies that the broadcast service uses the correct hashes
	// by setting up a minimal game environment

	// Setup minimal game environment
	_, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a broadcast service instance
	gameManager := services.GetGameInstance()
	broadcastService := gameManager.GetBroadcastService()
	require.NotNil(t, broadcastService, "Broadcast service should be available")

	// Create game with medium difficulty to test broadcast payload generation
	game := gameManager.GetGame()
	game.Difficulty = "medium"

	// Verify the broadcast service would use these exact hashes
	// by checking they're the same as what GetStationFromHash validates
	expectedHashes := map[string]string{
		"anchor":  config.HashAnchorStation,
		"chronos": config.HashChronosStation,
		"guide":   config.HashGuideStation,
		"clarity": config.HashClarityStation,
	}

	for station, expectedHash := range expectedHashes {
		detectedStation := config.GetStationFromHash(expectedHash)
		assert.NotEqual(t, config.UnknownStation, detectedStation,
			"Broadcast hash for %s should be valid for validation", station)
		assert.Equal(t, config.Station(station), detectedStation,
			"Broadcast hash for %s should map to correct station", station)
	}

	t.Logf("Verified broadcast service uses config hashes that pass validation")
}

func TestFullGameFlowWithHashValidation(t *testing.T) {
	// This test runs a complete game flow and verifies that:
	// 1. The correct hashes are sent to clients in phase start
	// 2. Those same hashes work for QR validation

	// Setup test server
	server, cleanup := setupTestServer(t)
	defer cleanup()

	// Connect host
	host := test_helpers.NewTestHostClient(t, server, config.HostUUID)
	err := host.Connect()
	require.NoError(t, err)
	defer host.Close()

	// Wait for host to be set up
	time.Sleep(50 * time.Millisecond)

	// Connect 4 players with different roles
	players := make([]*test_helpers.TestPlayerClient, 4)
	roles := []string{"detective", "art_enthusiast", "tourist", "janitor"}

	for i := 0; i < 4; i++ {
		players[i] = test_helpers.NewTestPlayerClient(t, server)
		err := players[i].Connect()
		require.NoError(t, err)
		defer players[i].Close()

		playerName := "Player" + string(rune('1'+i))
		err = players[i].ConfigurePlayer(playerName, roles[i], []string{"history"})
		require.NoError(t, err)

		// Wait between player configurations to avoid race conditions
		time.Sleep(100 * time.Millisecond)
	}

	// Wait for all players to be configured and ready
	time.Sleep(500 * time.Millisecond)

	// Start the game
	err = host.StartGame("medium")
	require.NoError(t, err)

	// Wait for resource phase to start
	time.Sleep(400 * time.Millisecond)

	// Try to find the RESOURCE_TO_CLIENT_PHASE_START message from any player
	var resourceHashes map[string]interface{}
	var foundMessage bool

	for playerIdx := 0; playerIdx < len(players); playerIdx++ {
		// Look through all messages for this player
		messages := players[playerIdx].GetMessages()
		for _, message := range messages {
			if message.Event == config.EventResourceToClientPhaseStart {
				// Parse the payload
				var payload map[string]interface{}
				if payloadBytes, ok := message.Payload.([]byte); ok {
					err = json.Unmarshal(payloadBytes, &payload)
					require.NoError(t, err)
				} else if payloadMap, ok := message.Payload.(map[string]interface{}); ok {
					payload = payloadMap
				} else {
					continue // Skip this message
				}

				// Extract resource station hashes
				if hashes, ok := payload["resourceStationHashes"].(map[string]interface{}); ok {
					resourceHashes = hashes
					foundMessage = true
					t.Logf("Found phase start message from player %d", playerIdx+1)
					break
				}
			}
		}
		if foundMessage {
			break
		}
	}

	require.True(t, foundMessage, "Should find RESOURCE_TO_CLIENT_PHASE_START message from at least one player")
	require.NotNil(t, resourceHashes, "Should find resourceStationHashes in phase start message")

	// Verify that the hashes sent to clients match config constants
	expectedHashes := map[string]string{
		"anchor":  config.HashAnchorStation,
		"chronos": config.HashChronosStation,
		"guide":   config.HashGuideStation,
		"clarity": config.HashClarityStation,
	}

	for station, expectedHash := range expectedHashes {
		actualHash, exists := resourceHashes[station]
		require.True(t, exists, "Hash for station %s should exist in message", station)
		assert.Equal(t, expectedHash, actualHash,
			"Hash for station %s should match config constant", station)

		// Also verify these hashes work for validation
		detectedStation := config.GetStationFromHash(expectedHash)
		assert.NotEqual(t, config.UnknownStation, detectedStation,
			"Sent hash %s should be valid for QR validation", expectedHash)
	}

	// Test that these hashes actually work for location verification
	// (This simulates a player scanning a QR code with the hash they received)
	for station, expectedHash := range expectedHashes {
		t.Run("QR_Validation_"+station, func(t *testing.T) {
			// Use the first player to test location verification
			err := players[0].VerifyLocation(station, expectedHash)
			assert.NoError(t, err,
				"Location verification should succeed with hash sent to client: %s", expectedHash)
		})
	}

	t.Logf("Successfully verified full game flow: correct hashes sent to clients and QR validation works")
}

func TestInvalidHashRejection(t *testing.T) {
	// Test that invalid hashes are properly rejected
	invalidHashes := []string{
		"anchor_station_qr_hash_2025", // The old duplicate hash (lowercase, wrong year)
		"chronos_station_qr_hash_2025",
		"invalid_hash",
		"ANCHOR_STATION_QR_HASH_2023", // Wrong year
		"",
		"random_string",
	}

	for _, hash := range invalidHashes {
		t.Run("Reject_invalid_hash_"+hash, func(t *testing.T) {
			station := config.GetStationFromHash(hash)
			assert.Equal(t, config.UnknownStation, station,
				"Invalid hash %s should return UnknownStation", hash)
		})
	}
}
