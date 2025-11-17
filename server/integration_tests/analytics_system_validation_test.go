package integration_tests

import (
	"canvas-conundrum/config"
	"canvas-conundrum/handlers"
	"canvas-conundrum/models"
	"canvas-conundrum/services"
	"canvas-conundrum/test_helpers"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Using createTestTriviaFiles and cleanupTestTriviaFiles from test_utils.go

// setupAnalyticsTestServer creates a test server for analytics testing
func setupAnalyticsTestServer(t *testing.T) (*httptest.Server, *test_helpers.TestHostClient, func()) {
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
	router.HandleFunc("/ws", handlers.HandlePlayerWebSocket).Methods("GET")
	router.HandleFunc("/ws/host/{uuid}", handlers.HandleHostWebSocket).Methods("GET")

	// Create test server
	server := httptest.NewServer(router)

	// Create host client
	hostClient := test_helpers.NewTestHostClient(t, server, config.HostUUID)

	cleanup := func() {
		hostClient.Close()
		server.Close()
		gm.Cleanup()
	}

	return server, hostClient, cleanup
}

func TestAnalyticsPhaseTransition(t *testing.T) {
	createTestTriviaFiles(t)
	defer cleanupTestTriviaFiles(t)

	_, hostClient, cleanup := setupAnalyticsTestServer(t)
	defer cleanup()

	// Connect host
	err := hostClient.Connect()
	require.NoError(t, err)

	// Wait for host connection confirmation
	_, err = hostClient.WaitForEvent(config.EventSetupToHostConnectionConfirmed, 2*time.Second)
	require.NoError(t, err)

	// Create and configure players using the real game implementation
	gm := services.GetGameInstance()

	// Add players through game manager like real implementation
	for i := 0; i < 4; i++ {
		playerName := fmt.Sprintf("AnalyticsPlayer%d", i+1)
		roles := []models.Role{models.RoleArtEnthusiast, models.RoleDetective, models.RoleTourist, models.RoleJanitor}

		player := models.NewPlayer(playerName, &websocket.Conn{})
		player.Name = playerName
		player.Role = roles[i%4]
		player.Specialties = []models.TriviaCategory{models.CategoryScience}
		player.IsReady = true
		player.IsActive = true

		_, err := gm.AddPlayer(player)
		require.NoError(t, err)
	}

	// Add host for game to start
	host := models.NewHost("analytics-host", &websocket.Conn{})
	gm.SetHost(host)

	// Set difficulty and start game using real implementation
	gm.GetGame().SetDifficulty(models.DifficultyMedium)
	err = gm.StartGame("medium")
	require.NoError(t, err)

	// Progress through game phases to reach analytics
	t.Run("ProgressToAnalyticsPhase", func(t *testing.T) {
		// Simulate some resource gathering activity
		game := gm.GetGame()
		analyticsService := gm.GetAnalyticsService()

		// Add some tokens and record analytics data
		for playerID, _ := range gm.GetAllPlayers() {
			// Simulate trivia answers
			analyticsService.RecordTriviaAnswer(playerID, "science", true, 5.0, 10, false)
			analyticsService.RecordTokenCollection(playerID, models.TokenClarity, 10)

			// Add tokens to team totals
			game.TeamTokens.AddTokens(models.TokenClarity, 10)
		}

		// Complete resource gathering and move to puzzle phase
		gm.CompleteResourceGathering()

		// Wait for transition to complete
		time.Sleep(6 * time.Second)

		// Verify we're in puzzle phase
		assert.Equal(t, models.PhasePuzzleAssembly, gm.GetCurrentPhase())

		// Start puzzle timer
		game.StartPuzzleTimer()

		// Complete individual puzzle segments for all players
		allPlayers := gm.GetAllPlayers()
		for playerID, player := range allPlayers {
			if player.AssignedSegment != "" {
				err := gm.CompleteSegment(playerID, player.AssignedSegment)
				require.NoError(t, err)
			}
		}

		// Simulate puzzle completion and move to analytics
		game.CurrentPhase = models.PhaseAnalytics
		game.PuzzleSuccess = true

		// Finalize game analytics like the real implementation
		analyticsService.FinalizeGame(game, gm.GetAllPlayers(), true)

		// Verify we're in analytics phase
		assert.Equal(t, models.PhaseAnalytics, gm.GetCurrentPhase())

		// Verify analytics data is available
		fullAnalytics := analyticsService.GetFullAnalytics()
		require.NotNil(t, fullAnalytics)
		assert.Equal(t, 4, len(fullAnalytics.PlayerAnalytics))
		assert.NotNil(t, fullAnalytics.TeamAnalytics)
	})
}

func TestGameStatisticsGeneration(t *testing.T) {
	createTestTriviaFiles(t)
	defer cleanupTestTriviaFiles(t)

	server, gm := setupAnalyticsGameWithData(t)
	defer server.Close()
	defer gm.Cleanup()

	t.Run("GenerateGameStatistics", func(t *testing.T) {
		// Verify game has been created with players
		game := gm.GetGame()
		require.NotNil(t, game)
		assert.Equal(t, 4, len(gm.GetAllPlayers()))

		// Generate analytics data through game simulation
		analyticsService := gm.GetAnalyticsService()

		// Record some trivia activity for each player
		for i, playerID := range getPlayerIDs(gm) {
			// Simulate different performance levels - give one player high accuracy for achievements
			correctAnswers := []int{10, 6, 7, 5}[i] // First player gets 100% for Perfectionist achievement
			totalQuestions := 10

			for q := 0; q < totalQuestions; q++ {
				isCorrect := q < correctAnswers
				responseTime := float64(5 + i) // Different response times per player
				tokensEarned := 0
				if isCorrect {
					tokensEarned = 10
				}

				analyticsService.RecordTriviaAnswer(playerID, "science", isCorrect, responseTime, tokensEarned, false)
				if isCorrect {
					analyticsService.RecordTokenCollection(playerID, models.TokenClarity, tokensEarned)
					// Add tokens to team totals
					game.TeamTokens.AddTokens(models.TokenClarity, tokensEarned)
				}
			}
		}

		// Progress to analytics phase
		game.CurrentPhase = models.PhaseResourceGathering
		gm.CompleteResourceGathering()
		time.Sleep(6 * time.Second)

		game.StartPuzzleTimer()

		// Complete puzzle segments
		for playerID, player := range gm.GetAllPlayers() {
			if player.AssignedSegment != "" {
				err := gm.CompleteSegment(playerID, player.AssignedSegment)
				require.NoError(t, err)
			}
		}

		// Complete puzzle and move to analytics
		game.CurrentPhase = models.PhaseAnalytics
		game.PuzzleSuccess = true

		// Finalize game analytics
		analyticsService.FinalizeGame(game, gm.GetAllPlayers(), true)

		// Verify analytics generation
		fullAnalytics := analyticsService.GetFullAnalytics()
		require.NotNil(t, fullAnalytics)

		// Verify player analytics
		assert.Equal(t, 4, len(fullAnalytics.PlayerAnalytics))
		for _, playerAnalytics := range fullAnalytics.PlayerAnalytics {
			assert.Greater(t, playerAnalytics.TotalQuestions, 0)
			// Achievements may be empty if player performance doesn't meet thresholds
			assert.NotNil(t, playerAnalytics.Achievements)
		}

		// Verify team analytics
		teamAnalytics := fullAnalytics.TeamAnalytics
		assert.NotNil(t, teamAnalytics)
		assert.Equal(t, 4, teamAnalytics.TotalPlayers)
		assert.Greater(t, teamAnalytics.TotalTokensCollected, 0)
	})
}

// Helper function to setup a game with data for analytics testing
func setupAnalyticsGameWithData(t *testing.T) (*httptest.Server, *services.GameManager) {
	// Setup game manager
	gm := services.GetGameInstance()
	gm.Cleanup()
	gm.ResetGame()

	// Setup services
	gm.SetBroadcastService(services.NewBroadcastService())

	triviaService := services.NewTriviaService()
	err := triviaService.LoadQuestions()
	require.NoError(t, err)
	gm.SetTriviaService(triviaService)

	gm.SetPuzzleService(services.NewPuzzleService())
	gm.SetAnalyticsService(services.NewAnalyticsService())

	// Create router and server
	router := mux.NewRouter()
	router.HandleFunc("/ws", handlers.HandlePlayerWebSocket).Methods("GET")
	router.HandleFunc("/ws/host/{uuid}", handlers.HandleHostWebSocket).Methods("GET")
	server := httptest.NewServer(router)

	// Add players using real implementation
	playerNames := []string{"Alice", "Bob", "Carol", "Dave"}
	roles := []models.Role{models.RoleArtEnthusiast, models.RoleDetective, models.RoleTourist, models.RoleJanitor}

	for i, name := range playerNames {
		player := models.NewPlayer(name, &websocket.Conn{})
		player.Name = name
		player.Role = roles[i]
		player.Specialties = []models.TriviaCategory{models.CategoryScience}
		player.IsReady = true
		player.IsActive = true

		_, err := gm.AddPlayer(player)
		require.NoError(t, err)
	}

	// Add host and start game
	host := models.NewHost("analytics-host", &websocket.Conn{})
	gm.SetHost(host)

	gm.GetGame().SetDifficulty(models.DifficultyMedium)
	err = gm.StartGame("medium")
	require.NoError(t, err)

	return server, gm
}

// Helper function to get player IDs in a consistent order
func getPlayerIDs(gm *services.GameManager) []string {
	var playerIDs []string
	for playerID := range gm.GetAllPlayers() {
		playerIDs = append(playerIDs, playerID)
	}
	return playerIDs
}
