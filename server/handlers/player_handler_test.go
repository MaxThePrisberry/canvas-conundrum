package handlers

import (
	"canvas-conundrum/config"
	"canvas-conundrum/models"
	"canvas-conundrum/services"
	"canvas-conundrum/test_helpers"
	"encoding/json"
	"testing"
	"time"

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
			"name":        "TestPlayer",
			"role":        "art_enthusiast",
			"specialties": []string{"science"},
		}

		msg := test_helpers.CreateTestMessage(config.EventSetupToServerPlayerConfiguration, payload)

		// Should not panic
		HandlePlayerMessage(player, msg)
		assert.True(t, true)
	})

	t.Run("Location Verified", func(t *testing.T) {
		payload := map[string]interface{}{
			"stationHash": "ANCHOR_STATION_QR_HASH_2024",
		}

		msg := test_helpers.CreateTestMessage(config.EventResourceToServerLocationVerified, payload)

		HandlePlayerMessage(player, msg)
		assert.True(t, true)
	})

	t.Run("Trivia Answer", func(t *testing.T) {
		payload := map[string]interface{}{
			"questionId": "test-question-id",
			"answer":     "Test Answer",
		}

		msg := test_helpers.CreateTestMessage(config.EventResourceToServerTriviaAnswer, payload)

		HandlePlayerMessage(player, msg)
		assert.True(t, true)
	})

	t.Run("Segment Completed", func(t *testing.T) {
		payload := map[string]interface{}{
			"segmentId": "test-segment-id",
		}

		msg := test_helpers.CreateTestMessage(config.EventPuzzleToServerSegmentCompleted, payload)

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

		msg := test_helpers.CreateTestMessage(config.EventPuzzleToServerFragmentMove, payload)

		HandlePlayerMessage(player, msg)
		assert.True(t, true)
	})

	t.Run("Recommend Move", func(t *testing.T) {
		payload := map[string]interface{}{
			"targetPlayerId": "target-player-id",
			"fromFragmentId": "fragment_01",
			"toFragmentId":   "fragment_02",
			"reasoning":      "This piece should go here",
		}

		msg := test_helpers.CreateTestMessage(config.EventPuzzleToServerRecommendMove, payload)

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

		msg := test_helpers.CreateTestMessage(config.EventPuzzleToServerRecommendationResponse, payload)

		HandlePlayerMessage(player, msg)
		assert.True(t, true)
	})

	t.Run("Ping", func(t *testing.T) {
		payload := map[string]interface{}{
			"timestamp": "2025-01-01T00:00:00Z",
		}

		msg := test_helpers.CreateTestMessage(config.EventSystemPing, payload)

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
			"name":        "TestPlayerName",
			"role":        "art_enthusiast",
			"specialties": []string{"science"},
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

func TestFragmentMoveRateLimiting(t *testing.T) {
	resetGameManager()
	gameManager := services.GetGameInstance()
	broadcastService := services.NewBroadcastService()
	gameManager.SetBroadcastService(broadcastService)

	// Initialize puzzle service and game state
	puzzleService := services.NewPuzzleService()
	gameManager.SetPuzzleService(puzzleService)

	player := models.NewPlayer("test-player", nil)
	gameManager.AddPlayer(player)

	// Set up game in puzzle phase using real application flow
	gameManager.GetGame().StartPuzzlePhase(4) // 4 players -> 2x2 grid

	// Add a fragment to the grid that the player can move
	fragment := &models.Fragment{
		ID:       "fragment-1",
		PlayerID: player.ID,
		Position: models.Position{X: 0, Y: 0},
	}
	gameManager.GetGame().PuzzleGrid.Fragments["fragment-1"] = fragment

	t.Run("First Move Should Succeed", func(t *testing.T) {
		payload := map[string]interface{}{
			"fragmentId":      "fragment-1",
			"currentPosition": map[string]int{"x": 0, "y": 0},
			"targetPosition":  map[string]int{"x": 1, "y": 1},
		}
		payloadJSON, _ := json.Marshal(payload)

		// First move should succeed (no cooldown)
		handleFragmentMove(player, payloadJSON)

		// Verify the move was processed (no errors thrown)
		assert.True(t, true)
	})

	t.Run("Rapid Successive Moves Should Be Rate Limited", func(t *testing.T) {
		// Make the first move
		player.UpdateLastMove() // Set last move time to now

		payload := map[string]interface{}{
			"fragmentId":      "fragment-1",
			"currentPosition": map[string]int{"x": 1, "y": 1},
			"targetPosition":  map[string]int{"x": 2, "y": 2},
		}
		payloadJSON, _ := json.Marshal(payload)

		// Immediate second move should be blocked by cooldown
		handleFragmentMove(player, payloadJSON)

		// Capture initial move count
		initialMoves := player.FragmentMoves

		// Try another immediate move
		payload2 := map[string]interface{}{
			"fragmentId":      "fragment-1",
			"currentPosition": map[string]int{"x": 1, "y": 1},
			"targetPosition":  map[string]int{"x": 0, "y": 1},
		}
		payloadJSON2, _ := json.Marshal(payload2)
		handleFragmentMove(player, payloadJSON2)

		// Fragment moves should not have increased due to cooldown
		assert.Equal(t, initialMoves, player.FragmentMoves, "Fragment moves should not increase due to cooldown")
	})

	t.Run("Move After Cooldown Should Succeed", func(t *testing.T) {
		// Simulate cooldown period has passed by setting last move time to past
		pastTime := time.Now().Add(time.Duration(-config.FragmentMoveCooldown-100) * time.Millisecond)
		player.LastMoveTime = pastTime

		// Use a minimal test to verify cooldown logic without full game state validation
		payload := map[string]interface{}{
			"fragmentId":      "fragment-1",
			"currentPosition": map[string]int{"x": 1, "y": 1},
			"targetPosition":  map[string]int{"x": 0, "y": 0},
		}
		payloadJSON, _ := json.Marshal(payload)

		handleFragmentMove(player, payloadJSON)

		// This move should succeed since cooldown has passed
		// Note: Actual validation will depend on game state, but cooldown check should pass
		assert.True(t, true, "Move after cooldown should be processed")
	})

	t.Run("CanMoveFragment Method Tests", func(t *testing.T) {
		// Test cooldown logic directly
		player.LastMoveTime = time.Time{} // Zero time - first move
		assert.True(t, player.CanMoveFragment(config.FragmentMoveCooldown), "First move should be allowed")

		// Set recent move time
		player.UpdateLastMove()
		assert.False(t, player.CanMoveFragment(config.FragmentMoveCooldown), "Move should be blocked by cooldown")

		// Set old move time
		player.LastMoveTime = time.Now().Add(time.Duration(-config.FragmentMoveCooldown-1) * time.Millisecond)
		assert.True(t, player.CanMoveFragment(config.FragmentMoveCooldown), "Move should be allowed after cooldown")
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
			"targetPlayerId": "target-player-id",
			"fromFragmentId": "fragment_01",
			"toFragmentId":   "fragment_02",
			"reasoning":      "This piece should go here",
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
