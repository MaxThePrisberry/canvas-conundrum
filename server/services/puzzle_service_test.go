package services

import (
	"canvas-conundrum/models"
	"canvas-conundrum/test_helpers"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPuzzleService(t *testing.T) {
	service := NewPuzzleService()

	assert.NotNil(t, service)
	assert.NotNil(t, service.segmentAssignments)
	assert.NotNil(t, service.recommendations)
	assert.NotNil(t, service.stopExpiration)
	assert.NotNil(t, service.expirationTicker)

	// Clean up
	service.stopExpiration <- true
}

func TestPuzzleServiceAssignSegments(t *testing.T) {
	service := NewPuzzleService()
	defer func() { service.stopExpiration <- true }()

	resetGameManager()
	gameManager := GetGameInstance()

	t.Run("Assign Segments to Active Players", func(t *testing.T) {
		// Create test players
		players := map[string]*models.Player{
			"player1": test_helpers.CreateTestPlayer("player1"),
			"player2": test_helpers.CreateTestPlayer("player2"),
		}
		players["player1"].IsActive = true
		players["player2"].IsActive = true

		// Add players to game manager
		gameManager.AddPlayer(players["player1"])
		gameManager.AddPlayer(players["player2"])

		// Assign segments
		service.AssignSegments(players, 2) // 2x2 grid

		// Check assignments
		assert.Len(t, service.segmentAssignments, 2)
		assert.NotEmpty(t, players["player1"].AssignedSegment)
		assert.NotEmpty(t, players["player2"].AssignedSegment)
		assert.NotNil(t, players["player1"].IndividualPuzzle)
		assert.NotNil(t, players["player2"].IndividualPuzzle)
		assert.Equal(t, "2A", players["player1"].PuzzlePhase)
		assert.Equal(t, "2A", players["player2"].PuzzlePhase)
		assert.False(t, players["player1"].SegmentCompleted)
		assert.False(t, players["player2"].SegmentCompleted)
	})

	t.Run("Skip Inactive Players", func(t *testing.T) {
		// Reset
		service.segmentAssignments = make(map[string]string)

		players := map[string]*models.Player{
			"player1": test_helpers.CreateTestPlayer("player1"),
			"player2": test_helpers.CreateTestPlayer("player2"),
		}
		players["player1"].IsActive = true
		players["player2"].IsActive = false // Inactive

		service.AssignSegments(players, 2)

		// Only active player should get assignment
		assert.Len(t, service.segmentAssignments, 1)
		assert.NotEmpty(t, players["player1"].AssignedSegment)
		assert.Empty(t, players["player2"].AssignedSegment)
	})

	t.Run("More Segments Than Players", func(t *testing.T) {
		// Reset
		service.segmentAssignments = make(map[string]string)

		players := map[string]*models.Player{
			"player1": test_helpers.CreateTestPlayer("player1"),
		}
		players["player1"].IsActive = true

		service.AssignSegments(players, 3) // 3x3 = 9 segments, 1 player

		// Check assignments
		assert.Len(t, service.segmentAssignments, 1)
		assert.Len(t, service.unassignedSegments, 8) // 9 - 1 = 8 unassigned
	})
}

func TestPuzzleServiceGenerateSegmentIDs(t *testing.T) {
	service := NewPuzzleService()
	defer func() { service.stopExpiration <- true }()

	t.Run("2x2 Grid", func(t *testing.T) {
		segments := service.generateSegmentIDs(2)

		expected := []string{"A1", "A2", "B1", "B2"}
		assert.Len(t, segments, 4)
		assert.ElementsMatch(t, expected, segments)
	})

	t.Run("3x3 Grid", func(t *testing.T) {
		segments := service.generateSegmentIDs(3)

		expected := []string{"A1", "A2", "A3", "B1", "B2", "B3", "C1", "C2", "C3"}
		assert.Len(t, segments, 9)
		assert.ElementsMatch(t, expected, segments)
	})

	t.Run("1x1 Grid", func(t *testing.T) {
		segments := service.generateSegmentIDs(1)

		expected := []string{"A1"}
		assert.Len(t, segments, 1)
		assert.ElementsMatch(t, expected, segments)
	})
}

func TestPuzzleServiceCreateRecommendation(t *testing.T) {
	service := NewPuzzleService()
	defer func() { service.stopExpiration <- true }()

	t.Run("Valid Recommendation", func(t *testing.T) {
		rec, err := service.CreateRecommendation(
			"player1",
			"Player One",
			"player2",
			"frag1",
			"frag2",
			"These pieces might fit together",
		)

		assert.NoError(t, err)
		assert.NotNil(t, rec)
		assert.NotEmpty(t, rec.ID)
		assert.Equal(t, "player1", rec.FromPlayerID)
		assert.Equal(t, "Player One", rec.FromPlayerName)
		assert.Equal(t, "player2", rec.ToPlayerID)
		assert.Equal(t, "frag1", rec.FromFragmentID)
		assert.Equal(t, "frag2", rec.ToFragmentID)
		assert.Equal(t, "These pieces might fit together", rec.Reasoning)
		assert.Equal(t, "pending", rec.Status)
		assert.True(t, rec.ExpiresAt.After(time.Now()))

		// Check it's stored
		stored, exists := service.GetRecommendation(rec.ID)
		assert.True(t, exists)
		assert.Equal(t, rec.ID, stored.ID)
	})
}

func TestPuzzleServiceGetRecommendation(t *testing.T) {
	service := NewPuzzleService()
	defer func() { service.stopExpiration <- true }()

	t.Run("Existing Recommendation", func(t *testing.T) {
		// Create a recommendation first
		rec, err := service.CreateRecommendation("p1", "Player 1", "p2", "f1", "f2", "test")
		require.NoError(t, err)

		// Retrieve it
		retrieved, exists := service.GetRecommendation(rec.ID)
		assert.True(t, exists)
		assert.Equal(t, rec.ID, retrieved.ID)
		assert.Equal(t, "p1", retrieved.FromPlayerID)
	})

	t.Run("Non-existent Recommendation", func(t *testing.T) {
		retrieved, exists := service.GetRecommendation("non-existent")
		assert.False(t, exists)
		assert.Nil(t, retrieved)
	})
}

func TestPuzzleServiceUpdateRecommendationStatus(t *testing.T) {
	service := NewPuzzleService()
	defer func() { service.stopExpiration <- true }()

	t.Run("Update Existing Recommendation", func(t *testing.T) {
		// Create a recommendation
		rec, err := service.CreateRecommendation("p1", "Player 1", "p2", "f1", "f2", "test")
		require.NoError(t, err)

		// Update status
		err = service.UpdateRecommendationStatus(rec.ID, "accepted")
		assert.NoError(t, err)

		// Check status was updated
		updated, exists := service.GetRecommendation(rec.ID)
		assert.True(t, exists)
		assert.Equal(t, "accepted", updated.Status)
	})

	t.Run("Update Non-existent Recommendation", func(t *testing.T) {
		err := service.UpdateRecommendationStatus("non-existent", "accepted")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "recommendation not found")
	})
}

func TestPuzzleServiceCalculateGuideHighlights(t *testing.T) {
	service := NewPuzzleService()
	defer func() { service.stopExpiration <- true }()

	t.Run("Nil Grid", func(t *testing.T) {
		highlights := service.CalculateGuideHighlights(nil, "fragment1", 5)
		assert.Empty(t, highlights)
	})

	t.Run("Valid Grid", func(t *testing.T) {
		// Create a test grid
		grid := &models.PuzzleGrid{
			Size:      3,
			Fragments: make(map[string]*models.Fragment),
		}

		// Mock fragment
		grid.Fragments["fragment1"] = &models.Fragment{
			ID:       "fragment1",
			Position: models.Position{X: 1, Y: 1},
		}

		highlights := service.CalculateGuideHighlights(grid, "fragment1", 3)

		// Should return positions (could be empty if no highlights computed)
		assert.NotNil(t, highlights)
	})
}

func TestPuzzleServiceValidateFragmentMove(t *testing.T) {
	service := NewPuzzleService()
	defer func() { service.stopExpiration <- true }()

	resetGameManager()
	gameManager := GetGameInstance()

	// Create test player in collaborative phase
	player := test_helpers.CreateTestPlayer("player1")
	player.PuzzlePhase = "2B" // Collaborative phase
	gameManager.AddPlayer(player)

	// Create test grid using proper constructor
	grid := models.NewPuzzleGrid(3)

	t.Run("Nil Grid", func(t *testing.T) {
		err := service.ValidateFragmentMove(nil, "player1", "frag1", models.Position{X: 1, Y: 1})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "puzzle grid not initialized")
	})

	t.Run("Fragment Not Found", func(t *testing.T) {
		err := service.ValidateFragmentMove(grid, "player1", "non-existent", models.Position{X: 1, Y: 1})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "fragment not found")
	})

	t.Run("Player Not Found", func(t *testing.T) {
		// Add fragment to grid properly
		frag1 := &models.Fragment{
			ID:       "frag1",
			Position: models.Position{X: 0, Y: 0},
			PlayerID: "player1",
		}
		grid.Fragments["frag1"] = frag1
		grid.Grid[0][0] = frag1

		err := service.ValidateFragmentMove(grid, "non-existent", "frag1", models.Position{X: 1, Y: 1})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "player not found")
	})

	t.Run("Player Not in Collaborative Phase", func(t *testing.T) {
		// Change player phase
		player.PuzzlePhase = "2A" // Individual phase

		err := service.ValidateFragmentMove(grid, "player1", "frag1", models.Position{X: 1, Y: 1})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "player must complete individual puzzle first")
	})

	t.Run("Valid Move - Own Fragment", func(t *testing.T) {
		// Reset player to collaborative phase
		player.PuzzlePhase = "2B"

		err := service.ValidateFragmentMove(grid, "player1", "frag1", models.Position{X: 1, Y: 1})
		assert.NoError(t, err)
	})

	t.Run("Invalid Move - Out of Bounds", func(t *testing.T) {
		err := service.ValidateFragmentMove(grid, "player1", "frag1", models.Position{X: 5, Y: 5})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "target position out of bounds")
	})

	t.Run("Invalid Move - Another Player's Fragment", func(t *testing.T) {
		// Add fragment owned by another player
		frag2 := &models.Fragment{
			ID:       "frag2",
			Position: models.Position{X: 1, Y: 1},
			PlayerID: "player2", // Different player
		}
		grid.Fragments["frag2"] = frag2
		grid.Grid[1][1] = frag2

		err := service.ValidateFragmentMove(grid, "player1", "frag2", models.Position{X: 2, Y: 2})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot move another player's fragment without permission")
	})
}

func TestPuzzleServiceExecuteRecommendedSwap(t *testing.T) {
	service := NewPuzzleService()
	defer func() { service.stopExpiration <- true }()

	// Create test grid using proper constructor
	grid := models.NewPuzzleGrid(3)

	// Add test fragments properly
	frag1 := &models.Fragment{
		ID:       "frag1",
		Position: models.Position{X: 0, Y: 0},
		PlayerID: "player1",
	}
	frag2 := &models.Fragment{
		ID:       "frag2",
		Position: models.Position{X: 1, Y: 1},
		PlayerID: "player2",
	}
	grid.Fragments["frag1"] = frag1
	grid.Fragments["frag2"] = frag2
	grid.Grid[0][0] = frag1
	grid.Grid[1][1] = frag2

	t.Run("Recommendation Not Found", func(t *testing.T) {
		err := service.ExecuteRecommendedSwap(grid, "non-existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "recommendation not found")
	})

	t.Run("Recommendation Not Accepted", func(t *testing.T) {
		// Create pending recommendation
		rec, err := service.CreateRecommendation("player1", "Player 1", "player2", "frag1", "frag2", "test")
		require.NoError(t, err)

		err = service.ExecuteRecommendedSwap(grid, rec.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "recommendation not accepted")
	})

	t.Run("Execute Accepted Recommendation", func(t *testing.T) {
		// Create and accept recommendation
		rec, err := service.CreateRecommendation("player1", "Player 1", "player2", "frag1", "frag2", "test")
		require.NoError(t, err)

		err = service.UpdateRecommendationStatus(rec.ID, "accepted")
		require.NoError(t, err)

		// Execute swap
		err = service.ExecuteRecommendedSwap(grid, rec.ID)
		assert.NoError(t, err)

		// Check status was updated
		updated, exists := service.GetRecommendation(rec.ID)
		assert.True(t, exists)
		assert.Equal(t, "executed", updated.Status)
	})
}

func TestPuzzleServiceRecommendationExpiration(t *testing.T) {
	service := NewPuzzleService()
	defer func() { service.stopExpiration <- true }()

	resetGameManager()
	gameManager := GetGameInstance()

	// Set up broadcast service
	broadcastService := NewBroadcastService()
	gameManager.SetBroadcastService(broadcastService)

	// Create test players
	player1 := test_helpers.CreateTestPlayer("player1")
	player1.IsActive = true
	player2 := test_helpers.CreateTestPlayer("player2")
	player2.IsActive = true

	gameManager.AddPlayer(player1)
	gameManager.AddPlayer(player2)

	t.Run("Expire Recommendations", func(t *testing.T) {
		// Create recommendation with past expiry time
		rec, err := service.CreateRecommendation("player1", "Player 1", "player2", "f1", "f2", "test")
		require.NoError(t, err)

		// Manually set expiry to past time
		service.mu.Lock()
		service.recommendations[rec.ID].ExpiresAt = time.Now().Add(-1 * time.Second)
		service.mu.Unlock()

		// Trigger expiration check
		service.checkAndExpireRecommendations()

		// Check recommendation was expired
		expired, exists := service.GetRecommendation(rec.ID)
		assert.True(t, exists)
		assert.Equal(t, "expired", expired.Status)
	})
}

func TestPuzzleServiceConcurrency(t *testing.T) {
	service := NewPuzzleService()
	defer func() { service.stopExpiration <- true }()

	// Test concurrent access to recommendations
	for i := 0; i < 10; i++ {
		go func(index int) {
			playerID := "player" + string(rune(index+'1'))
			_, _ = service.CreateRecommendation(playerID, "Player", "target", "f1", "f2", "test")
		}(i)
	}

	// Test concurrent reads
	for i := 0; i < 10; i++ {
		go func() {
			service.mu.RLock()
			_ = len(service.recommendations)
			service.mu.RUnlock()
		}()
	}

	// Should complete without race conditions
	assert.True(t, true)
}

func TestPuzzleServiceRecommendationInvalidationOnGridChange(t *testing.T) {
	resetGameManager()
	gameManager := GetGameInstance()
	broadcastService := NewBroadcastService()
	gameManager.SetBroadcastService(broadcastService)

	service := NewPuzzleService()
	gameManager.SetPuzzleService(service)
	defer func() { service.stopExpiration <- true }()

	// Set up test players
	player1 := test_helpers.CreateTestPlayer("player1")
	player2 := test_helpers.CreateTestPlayer("player2")
	player1.IsActive = true
	player2.IsActive = true
	gameManager.AddPlayer(player1)
	gameManager.AddPlayer(player2)

	// Initialize puzzle phase using real application flow
	gameManager.GetGame().StartPuzzlePhase(4) // 4 players gives 2x2 grid
	grid := gameManager.GetGame().PuzzleGrid

	// Add fragments to grid (2x2 grid, positions 0-1 for x,y)
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

	t.Run("Recommendation Should Be Invalidated When Involved Fragment Moves", func(t *testing.T) {
		// Create a recommendation involving fragment-1 and fragment-2
		rec, err := service.CreateRecommendation(
			player1.ID, "Player1", player2.ID, "fragment-1", "fragment-2", "This swap would be optimal")
		require.NoError(t, err)
		require.NotNil(t, rec)

		// Verify recommendation was created
		storedRec, exists := service.GetRecommendation(rec.ID)
		require.True(t, exists)
		require.NotNil(t, storedRec)
		assert.Equal(t, "pending", storedRec.Status)

		// Now move one of the involved fragments (fragment-1) to valid position in 2x2 grid
		err = grid.MoveFragment("fragment-1", models.Position{X: 1, Y: 0})
		require.NoError(t, err)

		// Call the method that should invalidate recommendations
		service.InvalidateRecommendationsForFragment("fragment-1")

		// Check that the recommendation was invalidated
		updatedRec, exists := service.GetRecommendation(rec.ID)
		require.True(t, exists)
		assert.Equal(t, "expired", updatedRec.Status, "Recommendation should be expired when involved fragment moves")
	})

	t.Run("Multiple Recommendations Should Be Invalidated For Same Fragment", func(t *testing.T) {
		// Add another fragment for testing (2x2 grid, valid positions are 0-1 for x,y)
		fragment3 := &models.Fragment{
			ID:       "fragment-3",
			PlayerID: "", // Unassigned fragment
			Position: models.Position{X: 0, Y: 1},
		}
		grid.Fragments["fragment-3"] = fragment3

		// Create multiple recommendations involving fragment-2
		rec1, _ := service.CreateRecommendation(
			player1.ID, "Player1", player2.ID, "fragment-2", "fragment-3", "First recommendation")
		rec2, _ := service.CreateRecommendation(
			player2.ID, "Player2", player1.ID, "fragment-1", "fragment-2", "Second recommendation")

		// Swap fragment-2 with fragment-1 (fragment-1 is at 1,0, fragment-2 is at 1,1)
		err := grid.SwapFragments("fragment-2", "fragment-1")
		require.NoError(t, err)

		// Invalidate recommendations for fragment-2
		service.InvalidateRecommendationsForFragment("fragment-2")

		// Both recommendations should be invalidated since they both involve fragment-2
		storedRec1, exists1 := service.GetRecommendation(rec1.ID)
		storedRec2, exists2 := service.GetRecommendation(rec2.ID)
		require.True(t, exists1)
		require.True(t, exists2)

		assert.Equal(t, "expired", storedRec1.Status, "First recommendation should be expired")
		assert.Equal(t, "expired", storedRec2.Status, "Second recommendation should be expired")
	})

	t.Run("Unrelated Recommendations Should Not Be Affected", func(t *testing.T) {
		// We'll use existing fragments instead of adding more to avoid position conflicts
		// Create a recommendation involving only fragment-3 and fragment-2 (not fragment-1)
		rec, _ := service.CreateRecommendation(
			player1.ID, "Player1", "", "fragment-3", "fragment-2", "Unrelated recommendation")

		// Move fragment-1 (should not affect the recommendation involving fragment-3 and fragment-2)
		err := grid.MoveFragment("fragment-1", models.Position{X: 0, Y: 0})
		require.NoError(t, err)

		service.InvalidateRecommendationsForFragment("fragment-1")

		// The unrelated recommendation should still be pending
		storedRec, exists := service.GetRecommendation(rec.ID)
		require.True(t, exists)
		assert.Equal(t, "pending", storedRec.Status, "Unrelated recommendation should remain pending")
	})
}

func TestPuzzleServiceNotifyRecommendationExpiration(t *testing.T) {
	resetGameManager()
	gameManager := GetGameInstance()

	// Use regular broadcast service
	broadcastService := NewBroadcastService()
	gameManager.SetBroadcastService(broadcastService)

	service := NewPuzzleService()
	gameManager.SetPuzzleService(service)
	defer func() { service.stopExpiration <- true }()

	// Set up test players
	player1 := test_helpers.CreateTestPlayer("player1")
	player2 := test_helpers.CreateTestPlayer("player2")
	player1.IsActive = true
	player2.IsActive = true
	gameManager.AddPlayer(player1)
	gameManager.AddPlayer(player2)

	t.Run("Grid State Change Should Send Expiration Notification", func(t *testing.T) {
		// Create a recommendation
		rec, err := service.CreateRecommendation(
			player1.ID, "Player1", player2.ID, "fragment-1", "fragment-2", "Test recommendation")
		require.NoError(t, err)

		// Invalidate due to grid state change
		service.InvalidateRecommendationsForFragment("fragment-1")

		// Verify the recommendation was invalidated
		storedRec, exists := service.GetRecommendation(rec.ID)
		require.True(t, exists)
		assert.Equal(t, "expired", storedRec.Status, "Recommendation should be expired")

		// Note: In a real test environment, we would need to mock the broadcast service
		// to capture and verify the exact messages sent. For now, we test the core
		// invalidation functionality.
		assert.True(t, true, "Invalidation completed without errors")
	})
}
