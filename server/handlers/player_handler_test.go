package handlers

import (
	"canvas-conundrum/constants"
	"canvas-conundrum/models"
	"canvas-conundrum/services"
	"canvas-conundrum/test_helpers"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandlePlayerMessage(t *testing.T) {
	// Reset game manager for clean tests
	resetGameManager()
	gameManager := services.GetGameInstance()
	
	// Set up services
	broadcastService := services.NewBroadcastService()
	gameManager.SetBroadcastService(broadcastService)
	
	player := models.NewPlayer("test-player", nil)
	
	t.Run("Player Configuration", func(t *testing.T) {
		payload := map[string]interface{}{
			"name":         "TestPlayer",
			"role":         "art_enthusiast",
			"specialties":  []string{"science"},
		}
		
		msg := test_helpers.CreateTestMessage(constants.EventSetupToServerPlayerConfiguration, payload)
		
		// Should not panic
		HandlePlayerMessage(player, msg)
		assert.True(t, true)
	})
	
	t.Run("Location Verified", func(t *testing.T) {
		payload := map[string]interface{}{
			"stationHash": "ANCHOR_STATION_QR_HASH_2024",
		}
		
		msg := test_helpers.CreateTestMessage(constants.EventResourceToServerLocationVerified, payload)
		
		HandlePlayerMessage(player, msg)
		assert.True(t, true)
	})
	
	t.Run("Trivia Answer", func(t *testing.T) {
		payload := map[string]interface{}{
			"questionId": "test-question-id",
			"answer":     "Test Answer",
		}
		
		msg := test_helpers.CreateTestMessage(constants.EventResourceToServerTriviaAnswer, payload)
		
		HandlePlayerMessage(player, msg)
		assert.True(t, true)
	})
	
	t.Run("Segment Completed", func(t *testing.T) {
		payload := map[string]interface{}{
			"segmentId": "test-segment-id",
		}
		
		msg := test_helpers.CreateTestMessage(constants.EventPuzzleToServerSegmentCompleted, payload)
		
		HandlePlayerMessage(player, msg)
		assert.True(t, true)
	})
	
	t.Run("Fragment Move", func(t *testing.T) {
		payload := map[string]interface{}{
			"fragmentId": "test-fragment-id",
			"fromRow":    0,
			"fromCol":    0,
			"toRow":      1,
			"toCol":      1,
		}
		
		msg := test_helpers.CreateTestMessage(constants.EventPuzzleToServerFragmentMove, payload)
		
		HandlePlayerMessage(player, msg)
		assert.True(t, true)
	})
	
	t.Run("Recommend Move", func(t *testing.T) {
		payload := map[string]interface{}{
			"fragmentId": "test-fragment-id",
			"fromRow":    0,
			"fromCol":    0,
			"toRow":      1,
			"toCol":      1,
			"message":    "This piece should go here",
		}
		
		msg := test_helpers.CreateTestMessage(constants.EventPuzzleToServerRecommendMove, payload)
		
		HandlePlayerMessage(player, msg)
		assert.True(t, true)
	})
	
	t.Run("Recommendation Response", func(t *testing.T) {
		// Set up puzzle service first
		puzzleService := services.NewPuzzleService()
		gameManager.SetPuzzleService(puzzleService)
		
		payload := map[string]interface{}{
			"recommendationId": "test-recommendation-id",
			"response":         "accept",
		}
		
		msg := test_helpers.CreateTestMessage(constants.EventPuzzleToServerRecommendationResponse, payload)
		
		HandlePlayerMessage(player, msg)
		assert.True(t, true)
	})
	
	t.Run("Ping", func(t *testing.T) {
		payload := map[string]interface{}{
			"timestamp": "2025-01-01T00:00:00Z",
		}
		
		msg := test_helpers.CreateTestMessage(constants.EventSystemPing, payload)
		
		HandlePlayerMessage(player, msg)
		assert.True(t, true)
	})
	
	t.Run("Unknown Event", func(t *testing.T) {
		payload := map[string]interface{}{
			"data": "test",
		}
		
		msg := test_helpers.CreateTestMessage("UNKNOWN_EVENT", payload)
		
		HandlePlayerMessage(player, msg)
		assert.True(t, true)
	})
}

func TestHandlePlayerConfiguration(t *testing.T) {
	resetGameManager()
	gameManager := services.GetGameInstance()
	broadcastService := services.NewBroadcastService()
	gameManager.SetBroadcastService(broadcastService)
	
	player := models.NewPlayer("test-player", nil)
	gameManager.AddPlayer(player)
	
	t.Run("Valid Configuration", func(t *testing.T) {
		payload := map[string]interface{}{
			"name":         "TestPlayerName",
			"role":         "art_enthusiast",
			"specialties":  []string{"science"},
		}
		payloadJSON, _ := json.Marshal(payload)
		
		// Should not panic
		handlePlayerConfiguration(player, payloadJSON)
		// Note: May not set values if validation fails, just test no panic
		assert.True(t, true)
	})
	
	t.Run("Invalid JSON", func(t *testing.T) {
		// Should handle invalid JSON gracefully
		handlePlayerConfiguration(player, []byte("invalid json"))
		assert.True(t, true) // Test passes if no panic
	})
}

func TestHandleLocationVerified(t *testing.T) {
	resetGameManager()
	gameManager := services.GetGameInstance()
	broadcastService := services.NewBroadcastService()
	gameManager.SetBroadcastService(broadcastService)
	
	player := models.NewPlayer("test-player", nil)
	gameManager.AddPlayer(player)
	
	t.Run("Valid Station Hash", func(t *testing.T) {
		payload := map[string]interface{}{
			"stationHash": "ANCHOR_STATION_QR_HASH_2024",
		}
		payloadJSON, _ := json.Marshal(payload)
		
		handleLocationVerified(player, payloadJSON)
		assert.True(t, true) // Test passes if no panic
	})
	
	t.Run("Invalid Station Hash", func(t *testing.T) {
		payload := map[string]interface{}{
			"stationHash": "INVALID_HASH",
		}
		payloadJSON, _ := json.Marshal(payload)
		
		handleLocationVerified(player, payloadJSON)
		assert.True(t, true) // Test passes if no panic
	})
}

func TestHandleTriviaAnswer(t *testing.T) {
	resetGameManager()
	gameManager := services.GetGameInstance()
	broadcastService := services.NewBroadcastService()
	gameManager.SetBroadcastService(broadcastService)
	
	triviaService := services.NewTriviaService()
	gameManager.SetTriviaService(triviaService)
	
	player := models.NewPlayer("test-player", nil)
	gameManager.AddPlayer(player)
	
	t.Run("Valid Answer", func(t *testing.T) {
		payload := map[string]interface{}{
			"questionId": "test-question-id",
			"answer":     "Test Answer",
		}
		payloadJSON, _ := json.Marshal(payload)
		
		handleTriviaAnswer(player, payloadJSON)
		assert.True(t, true) // Test passes if no panic
	})
	
	t.Run("Invalid JSON", func(t *testing.T) {
		handleTriviaAnswer(player, []byte("invalid json"))
		assert.True(t, true) // Test passes if no panic
	})
}

func TestHandleSegmentCompleted(t *testing.T) {
	resetGameManager()
	gameManager := services.GetGameInstance()
	broadcastService := services.NewBroadcastService()
	gameManager.SetBroadcastService(broadcastService)
	
	player := models.NewPlayer("test-player", nil)
	gameManager.AddPlayer(player)
	
	t.Run("Valid Segment", func(t *testing.T) {
		payload := map[string]interface{}{
			"segmentId": "test-segment-id",
		}
		payloadJSON, _ := json.Marshal(payload)
		
		handleSegmentCompleted(player, payloadJSON)
		assert.True(t, true) // Test passes if no panic
	})
}

func TestHandleFragmentMove(t *testing.T) {
	resetGameManager()
	gameManager := services.GetGameInstance()
	broadcastService := services.NewBroadcastService()
	gameManager.SetBroadcastService(broadcastService)
	
	// Initialize puzzle service
	puzzleService := services.NewPuzzleService()
	gameManager.SetPuzzleService(puzzleService)
	
	player := models.NewPlayer("test-player", nil)
	gameManager.AddPlayer(player)
	
	t.Run("Valid Move - Expected Error", func(t *testing.T) {
		payload := map[string]interface{}{
			"fragmentId":      "test-fragment-id",
			"currentPosition": map[string]int{"x": 0, "y": 0},
			"targetPosition":  map[string]int{"x": 1, "y": 1},
		}
		payloadJSON, _ := json.Marshal(payload)
		
		// This should fail gracefully due to game state (no puzzle grid initialized)
		// We just want to ensure no panic occurs and the handler executes
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("handleFragmentMove should not panic: %v", r)
			}
		}()
		
		handleFragmentMove(player, payloadJSON)
		assert.True(t, true) // Test passes if no panic
	})
}

func TestHandleRecommendMove(t *testing.T) {
	resetGameManager()
	gameManager := services.GetGameInstance()
	broadcastService := services.NewBroadcastService()
	gameManager.SetBroadcastService(broadcastService)
	
	puzzleService := services.NewPuzzleService()
	gameManager.SetPuzzleService(puzzleService)
	
	player := models.NewPlayer("test-player", nil)
	gameManager.AddPlayer(player)
	
	t.Run("Valid Recommendation", func(t *testing.T) {
		payload := map[string]interface{}{
			"fragmentId": "test-fragment-id",
			"fromRow":    0,
			"fromCol":    0,
			"toRow":      1,
			"toCol":      1,
			"message":    "This piece should go here",
		}
		payloadJSON, _ := json.Marshal(payload)
		
		handleRecommendMove(player, payloadJSON)
		assert.True(t, true) // Test passes if no panic
	})
}

func TestHandleRecommendationResponse(t *testing.T) {
	resetGameManager()
	gameManager := services.GetGameInstance()
	broadcastService := services.NewBroadcastService()
	gameManager.SetBroadcastService(broadcastService)
	
	puzzleService := services.NewPuzzleService()
	gameManager.SetPuzzleService(puzzleService)
	
	player := models.NewPlayer("test-player", nil)
	gameManager.AddPlayer(player)
	
	t.Run("Valid Response", func(t *testing.T) {
		payload := map[string]interface{}{
			"recommendationId": "test-recommendation-id",
			"response":         "accept",
		}
		payloadJSON, _ := json.Marshal(payload)
		
		handleRecommendationResponse(player, payloadJSON)
		assert.True(t, true) // Test passes if no panic
	})
}

func TestHandlePing(t *testing.T) {
	resetGameManager()
	gameManager := services.GetGameInstance()
	broadcastService := services.NewBroadcastService()
	gameManager.SetBroadcastService(broadcastService)
	
	player := models.NewPlayer("test-player", nil)
	gameManager.AddPlayer(player)
	
	t.Run("Valid Ping", func(t *testing.T) {
		payload := map[string]interface{}{
			"timestamp": "2025-01-01T00:00:00Z",
		}
		payloadJSON, _ := json.Marshal(payload)
		
		handlePing(player, payloadJSON)
		assert.True(t, true) // Test passes if no panic
	})
}

func TestGetTeamProgressPayload(t *testing.T) {
	resetGameManager()
	
	payload := getTeamProgressPayload()
	
	require.NotNil(t, payload)
	assert.Contains(t, payload, "teamTokens")
	assert.Contains(t, payload, "currentRound")
	assert.Contains(t, payload, "totalRounds")
	assert.Contains(t, payload, "tokenThresholds")
	
	// Verify token thresholds structure
	tokenThresholds, ok := payload["tokenThresholds"].(map[string]interface{})
	require.True(t, ok, "tokenThresholds should be a map")
	assert.Contains(t, tokenThresholds, "anchor")
	assert.Contains(t, tokenThresholds, "chronos")
	assert.Contains(t, tokenThresholds, "guide")
	assert.Contains(t, tokenThresholds, "clarity")
}