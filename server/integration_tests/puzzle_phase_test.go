package integration_tests

import (
	"canvas-conundrum/models"
	"canvas-conundrum/services"
	"canvas-conundrum/test_helpers"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPuzzlePhaseIntegration(t *testing.T) {
	// Reset game manager for clean test
	resetGameManager()
	gameManager := services.GetGameInstance()

	// Set up services
	broadcastService := services.NewBroadcastService()
	puzzleService := services.NewPuzzleService()
	gameManager.SetBroadcastService(broadcastService)
	gameManager.SetPuzzleService(puzzleService)
	defer puzzleService.Cleanup()

	// Set up host
	host := models.NewHost("test-host", nil)
	gameManager.SetHost(host)

	// Set up test players
	player1 := test_helpers.CreateTestPlayer("player1")
	player2 := test_helpers.CreateTestPlayer("player2")
	player1.IsActive = true
	player2.IsActive = true
	gameManager.AddPlayer(player1)
	gameManager.AddPlayer(player2)

	t.Run("End-to-End Puzzle Phase with Rate Limiting and Recommendation Invalidation", func(t *testing.T) {
		// Initialize puzzle phase using real application flow
		gameManager.GetGame().StartPuzzlePhase(2) // Start puzzle phase with 2 players (creates 2x2 grid)
		grid := gameManager.GetGame().PuzzleGrid

		// Add fragments to grid
		fragment1 := &models.Fragment{
			ID:       "fragment-1",
			PlayerID: player1.ID,
			Position: models.Position{X: 0, Y: 0},
		}
		fragment2 := &models.Fragment{
			ID:       "fragment-2",
			PlayerID: player2.ID,
			Position: models.Position{X: 1, Y: 1},
		}
		grid.Fragments["fragment-1"] = fragment1
		grid.Fragments["fragment-2"] = fragment2

		// Test 1: Rate Limiting (testing cooldown logic directly)
		t.Run("Fragment Move Rate Limiting Works", func(t *testing.T) {
			// Test cooldown logic directly on player object to avoid deadlock issues
			player1.LastMoveTime = time.Time{} // Zero time - first move
			assert.True(t, player1.CanMoveFragment(2500), "First move should be allowed")

			// Update last move time to now
			player1.UpdateLastMove()

			// Immediate second move should be blocked by cooldown
			assert.False(t, player1.CanMoveFragment(2500), "Second immediate move should be blocked by cooldown")

			// Move after cooldown period should be allowed
			pastTime := time.Now().Add(-3 * time.Second) // Past cooldown period
			player1.LastMoveTime = pastTime
			assert.True(t, player1.CanMoveFragment(2500), "Move should be allowed after cooldown period")
		})

		// Test 2: Recommendation System with Grid State Invalidation
		t.Run("Recommendation Invalidation on Grid Changes", func(t *testing.T) {
			// Create a recommendation between two fragments
			rec, err := puzzleService.CreateRecommendation(
				player1.ID, "Player1", player2.ID, "fragment-1", "fragment-2",
				"These fragments should be swapped for better positioning")
			require.NoError(t, err)
			require.NotNil(t, rec)

			// Verify recommendation was created and is pending
			storedRec, exists := puzzleService.GetRecommendation(rec.ID)
			require.True(t, exists)
			assert.Equal(t, "pending", storedRec.Status)

			// Manually trigger invalidation (as would happen when fragment moves)
			puzzleService.InvalidateRecommendationsForFragment("fragment-1")

			// Verify recommendation was invalidated
			updatedRec, exists := puzzleService.GetRecommendation(rec.ID)
			require.True(t, exists)
			assert.Equal(t, "expired", updatedRec.Status,
				"Recommendation should be expired when involved fragment moves")
		})

		// Test 3: Integration - Multiple recommendations with selective invalidation
		t.Run("Selective Recommendation Invalidation", func(t *testing.T) {
			// Create multiple recommendations involving different fragments
			rec1, _ := puzzleService.CreateRecommendation(
				player1.ID, "Player1", player2.ID, "fragment-1", "fragment-2", "Recommendation 1")
			rec2, _ := puzzleService.CreateRecommendation(
				player2.ID, "Player2", player1.ID, "fragment-1", "fragment-2", "Recommendation 2")
			rec3, _ := puzzleService.CreateRecommendation(
				player1.ID, "Player1", player2.ID, "fragment-2", "fragment-1", "Recommendation 3")

			// All should start as pending
			storedRec1, _ := puzzleService.GetRecommendation(rec1.ID)
			storedRec2, _ := puzzleService.GetRecommendation(rec2.ID)
			storedRec3, _ := puzzleService.GetRecommendation(rec3.ID)
			assert.Equal(t, "pending", storedRec1.Status)
			assert.Equal(t, "pending", storedRec2.Status)
			assert.Equal(t, "pending", storedRec3.Status)

			// Invalidate recommendations involving fragment-1
			puzzleService.InvalidateRecommendationsForFragment("fragment-1")

			// All recommendations should be expired since they all involve fragment-1 or fragment-2
			updatedRec1, _ := puzzleService.GetRecommendation(rec1.ID)
			updatedRec2, _ := puzzleService.GetRecommendation(rec2.ID)
			updatedRec3, _ := puzzleService.GetRecommendation(rec3.ID)

			assert.Equal(t, "expired", updatedRec1.Status, "Rec1 should be expired (involves fragment-1)")
			assert.Equal(t, "expired", updatedRec2.Status, "Rec2 should be expired (involves fragment-1)")
			assert.Equal(t, "expired", updatedRec3.Status, "Rec3 should be expired (involves fragment-1)")
		})

		// Test 4: Rate limiting independence between players
		t.Run("Rate Limiting Per-Player Independence", func(t *testing.T) {
			// Reset both players' move times to long ago
			player1.LastMoveTime = time.Now().Add(-5 * time.Second)
			player2.LastMoveTime = time.Now().Add(-5 * time.Second)

			// Both players should be able to move (no cooldown)
			assert.True(t, player1.CanMoveFragment(2500), "Player1 should be able to move")
			assert.True(t, player2.CanMoveFragment(2500), "Player2 should be able to move")

			// Update player1's move time but not player2's
			player1.UpdateLastMove()

			// Player1 should be on cooldown, but player2 should still be able to move
			assert.False(t, player1.CanMoveFragment(2500), "Player1 should be on cooldown")
			assert.True(t, player2.CanMoveFragment(2500), "Player2 should not be affected by player1's cooldown")

			// Update player2's move time
			player2.UpdateLastMove()

			// Now both should be on cooldown
			assert.False(t, player1.CanMoveFragment(2500), "Player1 should still be on cooldown")
			assert.False(t, player2.CanMoveFragment(2500), "Player2 should now be on cooldown")
		})
	})
}

func resetGameManager() {
	// Reset singleton for clean test
	services.ResetGameManagerInstance()
}
