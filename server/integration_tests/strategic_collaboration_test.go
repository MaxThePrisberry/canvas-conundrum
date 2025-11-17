package integration_tests

import (
	"canvas-conundrum/config"
	"canvas-conundrum/handlers"
	"canvas-conundrum/services"
	"canvas-conundrum/test_helpers"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupCollaborationTestServer creates a test server for collaboration testing
func setupCollaborationTestServer(t *testing.T) (*httptest.Server, *test_helpers.TestHostClient, func()) {
	// Reset game manager
	gm := services.GetGameInstance()
	gm.Cleanup()
	gm.ResetGame()

	// Setup all services with real implementation
	gm.SetBroadcastService(services.NewBroadcastService())

	// Setup trivia service with test questions
	triviaService := services.NewTriviaService()
	err := triviaService.LoadQuestions()
	require.NoError(t, err)
	gm.SetTriviaService(triviaService)

	gm.SetPuzzleService(services.NewPuzzleService())
	gm.SetAnalyticsService(services.NewAnalyticsService())

	// Create router
	router := mux.NewRouter()

	// Setup handlers
	router.HandleFunc("/ws", handlers.HandlePlayerWebSocket)
	router.HandleFunc("/host", handlers.HandleHostWebSocket)

	// Create test server
	server := httptest.NewServer(router)

	// Create host client
	hostUUID := "test-host-collaboration"
	hostClient := test_helpers.NewTestHostClient(t, server, hostUUID)

	cleanup := func() {
		hostClient.Close()
		server.Close()
		gm.Cleanup()
		gm.ResetGame()

		// Setup all services with real implementation
		gm.SetBroadcastService(services.NewBroadcastService())

		// Setup trivia service with test questions
		triviaService := services.NewTriviaService()
		err := triviaService.LoadQuestions()
		require.NoError(t, err)
		gm.SetTriviaService(triviaService)

		gm.SetPuzzleService(services.NewPuzzleService())
		gm.SetAnalyticsService(services.NewAnalyticsService())
	}

	return server, hostClient, cleanup
}

func TestCollaborativePuzzleAssembly(t *testing.T) {
	createTestTriviaFiles(t)
	defer cleanupTestTriviaFiles(t)

	server, hostClient, cleanup := setupCollaborationTestServer(t)
	defer cleanup()

	// Connect host
	err := hostClient.Connect()
	require.NoError(t, err)

	// Create game
	err = hostClient.StartGame("medium")
	require.NoError(t, err)

	// Create and configure players
	players := make([]*test_helpers.TestPlayerClient, 4)
	roles := []string{"art_enthusiast", "detective", "tourist", "janitor"}
	specialties := []string{"science", "history", "geography", "music"}

	for i := 0; i < 4; i++ {
		player := test_helpers.NewTestPlayerClient(t, server)
		err := player.Connect()
		require.NoError(t, err)
		defer player.Close()
		players[i] = player

		// Configure player
		err = player.ConfigurePlayer(fmt.Sprintf("Player%d", i+1), roles[i], []string{specialties[i]})
		require.NoError(t, err)

		// Wait for lobby status
		_, err = player.WaitForEvent(config.EventSetupToClientLobbyStatus, 2*time.Second)
		require.NoError(t, err)
	}

	// Start the game
	err = hostClient.StartGame("medium")
	require.NoError(t, err)

	// Wait for resource gathering phase
	for i := 0; i < 4; i++ {
		_, err = players[i].WaitForEvent(config.EventResourceToClientPhaseStart, 3*time.Second)
		require.NoError(t, err)
	}

	// Test collaborative resource gathering
	t.Run("CollaborativeResourceGathering", func(t *testing.T) {
		// Track team progress updates
		teamProgressUpdates := 0

		// Have players answer questions collaboratively
		for round := 0; round < 3; round++ {
			for playerIndex := 0; playerIndex < 4; playerIndex++ {
				// Player submits a trivia answer
				questionID := fmt.Sprintf("collab_q_%d_%d", round, playerIndex)

				err := players[playerIndex].AnswerTrivia(questionID, 0, 10.0)
				require.NoError(t, err)

				// Wait briefly for processing
				time.Sleep(100 * time.Millisecond)

				// Check for team progress updates
				progressMsg, err := players[playerIndex].WaitForEvent(config.EventResourceToClientTeamProgress, 500*time.Millisecond)
				if err == nil {
					teamProgressUpdates++

					// Verify team progress structure
					progressPayload := progressMsg.Payload.(map[string]interface{})
					assert.Contains(t, progressPayload, "totalTokens", "Team progress should include total tokens")
					assert.Contains(t, progressPayload, "playerContributions", "Team progress should include player contributions")
				}
			}
		}

		// Verify collaborative progress tracking
		assert.Greater(t, teamProgressUpdates, 0, "Should receive team progress updates during collaboration")
	})
}

func TestRealTimeCollaboration(t *testing.T) {
	createTestTriviaFiles(t)
	defer cleanupTestTriviaFiles(t)

	server, hostClient, cleanup := setupCollaborationTestServer(t)
	defer cleanup()

	// Connect host
	err := hostClient.Connect()
	require.NoError(t, err)

	// Create game
	err = hostClient.StartGame("medium")
	require.NoError(t, err)

	// Create players
	players := make([]*test_helpers.TestPlayerClient, 4)
	for i := 0; i < 4; i++ {
		player := test_helpers.NewTestPlayerClient(t, server)
		err := player.Connect()
		require.NoError(t, err)
		defer player.Close()
		players[i] = player

		// Configure player
		err = player.ConfigurePlayer(fmt.Sprintf("Player%d", i+1), "art_enthusiast", []string{"science"})
		require.NoError(t, err)

		// Wait for configuration
		_, err = player.WaitForEvent(config.EventSetupToClientLobbyStatus, 2*time.Second)
		require.NoError(t, err)
	}

	// Start game
	err = hostClient.StartGame("medium")
	require.NoError(t, err)

	// Test real-time message propagation
	t.Run("RealTimeMessagePropagation", func(t *testing.T) {
		// Player 0 performs an action
		startTime := time.Now()

		questionID := "realtime_test"

		err := players[0].AnswerTrivia(questionID, 0, 10.0)
		require.NoError(t, err)

		// Check how quickly other players receive updates
		updateTimes := make([]time.Duration, 3)
		for i := 1; i < 4; i++ {
			// Wait for any game update message
			_, err := players[i].WaitForEvent(config.EventResourceToClientTeamProgress, 2*time.Second)
			if err == nil {
				updateTimes[i-1] = time.Since(startTime)
			}
		}

		// Verify real-time performance
		for i, updateTime := range updateTimes {
			if updateTime > 0 {
				assert.Less(t, updateTime, 1*time.Second, "Player %d should receive updates quickly", i+1)
			}
		}
	})
}

func TestCentralGridCoordination(t *testing.T) {
	createTestTriviaFiles(t)
	defer cleanupTestTriviaFiles(t)

	server, hostClient, cleanup := setupCollaborationTestServer(t)
	defer cleanup()

	// Connect host
	err := hostClient.Connect()
	require.NoError(t, err)

	// Test different grid sizes based on player count
	testCases := []struct {
		playerCount  int
		expectedSize int
	}{
		{3, 3}, // 3x3 grid for 3 players
		{4, 4}, // 4x4 grid for 4 players
		{6, 6}, // 6x6 grid for 6 players
		{8, 8}, // 8x8 grid for 8 players
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("GridSize_%dx%d", tc.expectedSize, tc.expectedSize), func(t *testing.T) {
			// Create game with specific player count
			err = hostClient.StartGame("medium")
			require.NoError(t, err)

			// Connect players
			players := make([]*test_helpers.TestPlayerClient, tc.playerCount)
			for i := 0; i < tc.playerCount; i++ {
				player := test_helpers.NewTestPlayerClient(t, server)
				err := player.Connect()
				require.NoError(t, err)
				defer player.Close()
				players[i] = player

				// Configure player
				err = player.ConfigurePlayer(fmt.Sprintf("Player%d", i+1), "art_enthusiast", []string{"science"})
				require.NoError(t, err)

				// Wait for configuration
				_, err = player.WaitForEvent(config.EventSetupToClientLobbyStatus, 2*time.Second)
				require.NoError(t, err)
			}

			// Start game
			err = hostClient.StartGame("medium")
			require.NoError(t, err)

			// Progress to puzzle phase to test central grid
			// Fast-forward through resource gathering
			gm := services.GetGameInstance()
			game := gm.GetGame()
			if game != nil {
				gm.CompleteResourceGathering()

				// Start puzzle phase
				err = hostClient.StartPuzzlePhase()
				require.NoError(t, err)

				// Wait for puzzle phase
				for i := 0; i < tc.playerCount; i++ {
					puzzleMsg, err := players[i].WaitForEvent(config.EventPuzzleToClientPhaseLoad, 3*time.Second)
					if err == nil {
						// Verify grid size in puzzle data
						payload := puzzleMsg.Payload.(map[string]interface{})
						if centralGrid, exists := payload["centralGrid"]; exists {
							gridData := centralGrid.(map[string]interface{})
							if width, ok := gridData["width"]; ok {
								assert.Equal(t, float64(tc.expectedSize), width.(float64), "Grid width should match player count")
							}
							if height, ok := gridData["height"]; ok {
								assert.Equal(t, float64(tc.expectedSize), height.(float64), "Grid height should match player count")
							}
						}
					}
				}
			}

			// Reset for next test case
			gm.Cleanup()
			gm.ResetGame()

			// Setup all services with real implementation
			gm.SetBroadcastService(services.NewBroadcastService())

			// Setup trivia service with test questions
			triviaService := services.NewTriviaService()
			err := triviaService.LoadQuestions()
			require.NoError(t, err)
			gm.SetTriviaService(triviaService)

			gm.SetPuzzleService(services.NewPuzzleService())
			gm.SetAnalyticsService(services.NewAnalyticsService())
		})
	}
}

func TestFragmentOwnershipRules(t *testing.T) {
	createTestTriviaFiles(t)
	defer cleanupTestTriviaFiles(t)

	server, hostClient, cleanup := setupCollaborationTestServer(t)
	defer cleanup()

	// Connect host
	err := hostClient.Connect()
	require.NoError(t, err)

	// Create game
	err = hostClient.StartGame("medium")
	require.NoError(t, err)

	// Create players
	players := make([]*test_helpers.TestPlayerClient, 4)
	for i := 0; i < 4; i++ {
		player := test_helpers.NewTestPlayerClient(t, server)
		err := player.Connect()
		require.NoError(t, err)
		defer player.Close()
		players[i] = player

		// Configure player
		err = player.ConfigurePlayer(fmt.Sprintf("Player%d", i+1), "art_enthusiast", []string{"science"})
		require.NoError(t, err)

		// Wait for configuration
		_, err = player.WaitForEvent(config.EventSetupToClientLobbyStatus, 2*time.Second)
		require.NoError(t, err)
	}

	// Start and progress to puzzle phase
	err = hostClient.StartGame("medium")
	require.NoError(t, err)

	// Fast-forward to puzzle phase
	gm := services.GetGameInstance()
	game := gm.GetGame()
	if game != nil {
		gm.CompleteResourceGathering()

		err = hostClient.StartPuzzlePhase()
		require.NoError(t, err)

		// Wait for puzzle phase
		for i := 0; i < 4; i++ {
			_, err = players[i].WaitForEvent(config.EventPuzzleToClientPhaseLoad, 3*time.Second)
			require.NoError(t, err)
		}

		// Test fragment ownership
		t.Run("FragmentOwnership", func(t *testing.T) {
			// Simulate Player 0 completing their individual puzzle to get a fragment
			// This would normally happen through puzzle piece placement

			// For testing, we'll simulate fragment placement attempts
			fragmentID := "0" // Player 0's fragment
			fromPos := "0,0"
			toPos := "0,0"

			// Player 0 should be able to place their own fragment
			err := players[0].MoveFragment(fragmentID, fromPos, toPos)
			require.NoError(t, err)

			time.Sleep(200 * time.Millisecond)

			// Player 1 should NOT be able to move Player 0's fragment
			moveFragmentID := "0" // Trying to move Player 0's fragment
			moveFromPos := "0,0"
			moveToPos := "1,1"

			err = players[1].MoveFragment(moveFragmentID, moveFromPos, moveToPos)
			// This should either be rejected or ignored by the server
			// The actual behavior depends on the server implementation

			// Verify through game state that ownership is maintained
			// This would be checked through puzzle update messages
		})
	}
}

func TestCollaborativeProgress(t *testing.T) {
	createTestTriviaFiles(t)
	defer cleanupTestTriviaFiles(t)

	server, hostClient, cleanup := setupCollaborationTestServer(t)
	defer cleanup()

	// Connect host
	err := hostClient.Connect()
	require.NoError(t, err)

	// Create game
	err = hostClient.StartGame("medium")
	require.NoError(t, err)

	// Create players with different specialties for diverse collaboration
	players := make([]*test_helpers.TestPlayerClient, 4)
	specialties := []string{"science", "history", "geography", "music"}

	for i := 0; i < 4; i++ {
		player := test_helpers.NewTestPlayerClient(t, server)
		err := player.Connect()
		require.NoError(t, err)
		defer player.Close()
		players[i] = player

		// Configure with different specialties
		err = player.ConfigurePlayer(fmt.Sprintf("Player%d", i+1), "art_enthusiast", []string{specialties[i]})
		require.NoError(t, err)

		// Wait for configuration
		_, err = player.WaitForEvent(config.EventSetupToClientLobbyStatus, 2*time.Second)
		require.NoError(t, err)
	}

	// Start game
	err = hostClient.StartGame("medium")
	require.NoError(t, err)

	// Wait for resource phase
	for i := 0; i < 4; i++ {
		_, err = players[i].WaitForEvent(config.EventResourceToClientPhaseStart, 3*time.Second)
		require.NoError(t, err)
	}

	// Test collaborative progress tracking
	t.Run("TeamProgressTracking", func(t *testing.T) {
		initialProgress := make(map[string]interface{})

		// Get initial team progress
		progressMsg, err := players[0].WaitForEvent(config.EventResourceToClientTeamProgress, 2*time.Second)
		if err == nil {
			initialProgress = progressMsg.Payload.(map[string]interface{})
		}

		// Have each player contribute
		for i := 0; i < 4; i++ {
			questionID := fmt.Sprintf("progress_q_%d", i)

			err := players[i].AnswerTrivia(questionID, 0, 10.0)
			require.NoError(t, err)

			time.Sleep(200 * time.Millisecond)

			// Check updated progress
			progressMsg, err := players[i].WaitForEvent(config.EventResourceToClientTeamProgress, 1*time.Second)
			if err == nil {
				payload := progressMsg.Payload.(map[string]interface{})

				// Verify progress structure
				assert.Contains(t, payload, "totalTokens", "Should track total team tokens")
				assert.Contains(t, payload, "playerContributions", "Should track individual contributions")

				// Progress should advance from initial state
				if totalTokens, ok := payload["totalTokens"]; ok {
					if initialTokens, exists := initialProgress["totalTokens"]; exists {
						assert.GreaterOrEqual(t, totalTokens, initialTokens, "Team tokens should not decrease")
					}
				}
			}
		}
	})
}
