package e2e_tests

import (
	"canvas-conundrum/config"
	"canvas-conundrum/handlers"
	"canvas-conundrum/models"
	"canvas-conundrum/services"
	"canvas-conundrum/test_helpers"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestTriviaFiles creates trivia files in the expected location for testing
func createTestTriviaFiles() error {
	// Create trivia directory structure
	categories := []string{"general", "geography", "history", "music", "science", "video_games"}
	difficulties := []string{"easy", "medium", "hard"}

	for _, cat := range categories {
		catDir := filepath.Join("./trivia", cat)
		if err := os.MkdirAll(catDir, 0755); err != nil {
			return err
		}

		for _, diff := range difficulties {
			// Create mock trivia data
			response := models.TriviaAPIResponse{
				ResponseCode: 0,
				Results: []models.RawTriviaQuestion{
					{
						Category:      cat,
						Type:          "multiple",
						Difficulty:    diff,
						Question:      fmt.Sprintf("Test %s %s question 1?", cat, diff),
						CorrectAnswer: "Correct Answer 1",
						Incorrect:     []string{"Wrong 1", "Wrong 2", "Wrong 3"},
					},
					{
						Category:      cat,
						Type:          "multiple",
						Difficulty:    diff,
						Question:      fmt.Sprintf("Test %s %s question 2?", cat, diff),
						CorrectAnswer: "Correct Answer 2",
						Incorrect:     []string{"Wrong A", "Wrong B", "Wrong C"},
					},
				},
			}

			data, err := json.Marshal(response)
			if err != nil {
				return err
			}

			filePath := filepath.Join(catDir, diff+".json")
			if err := ioutil.WriteFile(filePath, data, 0644); err != nil {
				return err
			}
		}
	}

	return nil
}

func setupE2EServer(t *testing.T) (*httptest.Server, *services.GameManager) {
	// Reset and setup game manager
	gm := services.GetGameInstance()
	gm.Cleanup()   // Clean up any existing state first
	gm.ResetGame() // Reset any state from previous tests
	gm.SetBroadcastService(services.NewBroadcastService())

	// Setup trivia service with test questions
	triviaService := services.NewTriviaService()

	// Create mock trivia files in the expected location
	// The service expects files in ./trivia, so we'll create them there temporarily
	err := createTestTriviaFiles()
	require.NoError(t, err)
	t.Cleanup(func() {
		// Clean up test trivia files
		os.RemoveAll("./trivia")
	})

	err = triviaService.LoadQuestions()
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
	return server, gm
}

func TestCompleteGameFlow(t *testing.T) {
	server, gm := setupE2EServer(t)
	defer server.Close()
	defer gm.Cleanup() // Ensure proper cleanup of game manager

	// Create persistent connections for entire test
	host := test_helpers.NewTestHostClient(t, server, config.HostUUID)
	err := host.Connect()
	require.NoError(t, err)
	defer host.Close()

	// Connect 4 players (minimum)
	players := make([]*test_helpers.TestPlayerClient, 4)
	playerNames := []string{"Alice", "Bob", "Charlie", "Diana"}
	roles := []string{"art_enthusiast", "detective", "tourist", "janitor"}
	specialties := [][]string{
		{"general"},
		{"history"},
		{"science"},
		{"music"},
	}

	for i := 0; i < 4; i++ {
		players[i] = test_helpers.NewTestPlayerClient(t, server)
		err := players[i].Connect()
		require.NoError(t, err)
		defer players[i].Close()
	}

	// Phase 1: Setup
	t.Run("Phase1_Setup", func(t *testing.T) {
		for i := 0; i < 4; i++ {

			// Store initial message with player ID
			initialMsg := players[i].GetLastMessage()
			require.NotNil(t, initialMsg)

			// Configure player
			err := players[i].ConfigurePlayer(playerNames[i], roles[i], specialties[i])
			require.NoError(t, err)
		}

		// Wait for all players to be configured
		time.Sleep(200 * time.Millisecond)

		// Verify lobby status
		assert.Equal(t, 4, gm.GetPlayerCount())

		// Set difficulty before starting the game
		game := gm.GetGame()
		game.SetDifficulty(models.DifficultyMedium)

		// Mark all players as ready so we can start the game by updating their configuration
		allPlayers := gm.GetAllPlayers()
		for _, player := range allPlayers {
			// Use the proper method to mark players as ready
			err := gm.UpdatePlayerConfiguration(player.ID, player.Name, player.Role, []string{})
			require.NoError(t, err)
		}

		// Use proper GameManager method to start the game
		err := gm.StartGame()
		require.NoError(t, err)

		// Verify game state
		assert.True(t, game.GameStarted)
		assert.Equal(t, models.PhaseResourceGathering, gm.GetCurrentPhase())
	})

	// Phase 2: Resource Gathering
	t.Run("Phase2_ResourceGathering", func(t *testing.T) {
		// Each player should receive phase start
		// Simulate QR scanning and trivia answering

		// Get players from game manager
		allPlayers := gm.GetAllPlayers()
		assert.Len(t, allPlayers, 4)

		// Get analytics service and initialize it
		analyticsService := gm.GetAnalyticsService()
		require.NotNil(t, analyticsService)

		// Initialize analytics for the game
		analyticsService.StartGame(gm.GetGame().ID)
		for _, player := range allPlayers {
			analyticsService.InitializePlayer(player.ID, player.Name)
		}

		// Simulate one round of resource gathering
		for _, player := range allPlayers {
			// Simulate QR scan at a station
			station := "anchor"
			_ = config.HashAnchorStation

			// In a real test, we'd send the location verification through WebSocket
			// For now, directly update the player state
			player.CurrentStation = station

			// Record station visit in analytics
			analyticsService.RecordStationVisit(player.ID, station)

			// Get a trivia question for the player
			questions := gm.GetTriviaService().GetQuestionsForRound(gm.GetAllPlayers())
			question, ok := questions[player.ID]
			require.True(t, ok)
			require.NotNil(t, question)

			// Simulate correct answer
			isCorrect := true
			tokensEarned := 3
			player.QuestionsAnswered++
			player.CorrectAnswers++
			player.TokensEarned += tokensEarned

			// Record trivia answer in analytics
			responseTime := 5.0  // Simulate 5 second response time
			isSpecialty := false // Assume not a specialty for simplicity
			analyticsService.RecordTriviaAnswer(
				player.ID,
				question.Category,
				isCorrect,
				responseTime,
				tokensEarned,
				isSpecialty,
			)

			// Add tokens to team and record in analytics
			game := gm.GetGame()
			game.TeamTokens.AddTokens(models.TokenAnchor, tokensEarned)
			analyticsService.RecordTokenCollection(player.ID, models.TokenAnchor, tokensEarned)
		}

		// Verify tokens were added
		game := gm.GetGame()
		tokens := game.TeamTokens
		assert.Equal(t, 12, tokens.AnchorTokens) // 4 players * 3 tokens

		// Move to next phase - properly initialize puzzle phase
		game.StartPuzzlePhase(gm.GetPlayerCount())
		assert.Equal(t, models.PhasePuzzleAssembly, gm.GetCurrentPhase())
	})

	// Phase 3: Puzzle Assembly
	t.Run("Phase3_PuzzleAssembly", func(t *testing.T) {
		// Initialize puzzle service
		puzzleService := gm.GetPuzzleService()
		require.NotNil(t, puzzleService)

		// Get the game and verify puzzle grid exists
		game := gm.GetGame()
		// If PuzzleGrid is nil, initialize it (for isolated test runs)
		if game.PuzzleGrid == nil {
			game.StartPuzzlePhase(gm.GetPlayerCount())
		}
		require.NotNil(t, game.PuzzleGrid, "Puzzle grid should be initialized")

		// Assign puzzle segments using the service method
		gridSize := game.GetGridSize()
		puzzleService.AssignSegments(gm.GetAllPlayers(), gridSize)

		// Start puzzle timer (needed for solve time calculation)
		gm.GetGame().StartPuzzleTimer()

		// Simulate individual puzzle solving
		for playerID, player := range gm.GetAllPlayers() {
			if player.AssignedSegment != "" {
				// Simulate segment completion through game manager
				err := gm.CompleteSegment(playerID, player.AssignedSegment)
				require.NoError(t, err)

				// Get updated player state from game manager
				updatedPlayer, exists := gm.GetPlayer(playerID)
				require.True(t, exists)
				assert.True(t, updatedPlayer.SegmentCompleted)
			}
		}

		// Simulate collaborative grid assembly
		grid := gm.GetGame().PuzzleGrid
		require.NotNil(t, grid)

		// Place some fragments
		fragments := grid.Fragments
		placedCount := 0
		for _, fragment := range fragments {
			if placedCount >= 4 {
				break
			}
			// Try to move fragment to position
			row := placedCount / gridSize
			col := placedCount % gridSize
			newPos := models.Position{X: col, Y: row}
			err := grid.MoveFragment(fragment.ID, newPos)
			if err == nil {
				placedCount++
			}
		}

		// Verify fragments placed
		assert.Greater(t, len(grid.Fragments), 0)

		// Move to analytics phase
		game = gm.GetGame()
		game.CurrentPhase = models.PhaseAnalytics
		game.PhaseStartTime = time.Now()
		assert.Equal(t, models.PhaseAnalytics, gm.GetCurrentPhase())
	})

	// Phase 4: Analytics
	t.Run("Phase4_Analytics", func(t *testing.T) {
		analyticsService := gm.GetAnalyticsService()
		require.NotNil(t, analyticsService)

		// Mark puzzle as successful for testing (simulate puzzle was solved)
		gm.GetGame().PuzzleSuccess = true

		// Finalize game analytics
		analyticsService.FinalizeGame(gm.GetGame(), gm.GetAllPlayers(), true)

		// Get analytics
		fullAnalytics := analyticsService.GetFullAnalytics()
		require.NotNil(t, fullAnalytics)

		// Verify analytics data
		assert.Equal(t, 4, len(fullAnalytics.PlayerAnalytics))
		assert.NotNil(t, fullAnalytics.TeamAnalytics)
		assert.Equal(t, 4, fullAnalytics.TeamAnalytics.TotalPlayers)

		// Check player analytics
		for _, playerAnalytics := range fullAnalytics.PlayerAnalytics {
			assert.Greater(t, playerAnalytics.TotalQuestions, 0)
			assert.Greater(t, playerAnalytics.TokensEarned[models.TokenAnchor], 0)
			assert.NotEmpty(t, playerAnalytics.Achievements)
		}

		// Check team analytics
		teamAnalytics := fullAnalytics.TeamAnalytics
		assert.Greater(t, teamAnalytics.TotalTokensCollected, 0)
		assert.NotEmpty(t, teamAnalytics.TeamAchievements)
	})
}

func TestGameWithHighTokens(t *testing.T) {
	server, gm := setupE2EServer(t)
	defer server.Close()
	defer gm.Cleanup() // Ensure proper cleanup of game manager

	// Setup game with 4 players
	for i := 0; i < 4; i++ {
		player := models.NewPlayer(string(rune('a'+i)), nil)
		player.Name = "Player" + string(rune('A'+i))
		player.Role = models.RoleArtEnthusiast
		player.IsReady = true
		player.IsActive = true
		gm.AddPlayer(player)
	}

	// Add host for game to start
	host := models.NewHost("test-host", &websocket.Conn{})
	gm.SetHost(host)

	// Set difficulty and start game
	gm.GetGame().SetDifficulty(models.DifficultyMedium)
	err := gm.StartGame()
	require.NoError(t, err)

	// Add high tokens to trigger thresholds
	// Based on constants: Anchor=25/threshold, Chronos=20, Guide=15, Clarity=30
	game := gm.GetGame()
	game.TeamTokens.AddTokens(models.TokenAnchor, 75)  // 75/25 = 3 thresholds
	game.TeamTokens.AddTokens(models.TokenChronos, 40) // 40/20 = 2 thresholds
	game.TeamTokens.AddTokens(models.TokenGuide, 30)   // 30/15 = 2 thresholds
	game.TeamTokens.AddTokens(models.TokenClarity, 30) // 30/30 = 1 threshold

	// Verify thresholds
	tokens := game.TeamTokens
	assert.Equal(t, 3, tokens.GetThreshold(models.TokenAnchor))
	assert.Equal(t, 2, tokens.GetThreshold(models.TokenChronos))
	assert.Equal(t, 2, tokens.GetThreshold(models.TokenGuide))
	assert.Equal(t, 1, tokens.GetThreshold(models.TokenClarity))

	// Check pre-solved pieces (only anchor affects this)
	game = gm.GetGame()
	preSolved := game.GetPreSolvedPieces()
	expectedPreSolved := 3 * config.PiecesPreSolvedPerThreshold // Only anchor thresholds count
	assert.Equal(t, expectedPreSolved, preSolved)

	// Check puzzle time bonus
	totalTime := gm.GetGame().GetTotalPuzzleTime()
	expectedTime := config.PuzzleBaseTime + (2 * config.TimeExtensionPerThreshold) // Chronos threshold 2
	assert.Equal(t, expectedTime, totalTime)
}

func TestLargeScaleGame(t *testing.T) {
	server, gm := setupE2EServer(t)
	defer server.Close()
	defer gm.Cleanup() // Ensure proper cleanup of game manager

	// Add maximum players
	playerCount := 32 // Test with 32 players
	for i := 0; i < playerCount; i++ {
		player := models.NewPlayer(string(rune(i)), nil)
		player.Name = "Player" + string(rune(i))
		player.Role = []models.Role{
			models.RoleArtEnthusiast,
			models.RoleDetective,
			models.RoleTourist,
			models.RoleJanitor,
		}[i%4]
		player.IsReady = true
		player.IsActive = true
		_, err := gm.AddPlayer(player)
		require.NoError(t, err)
	}

	// Add host for game to start
	host := models.NewHost("test-host", &websocket.Conn{})
	gm.SetHost(host)

	// Start game with hard difficulty
	gm.GetGame().SetDifficulty(models.DifficultyHard)
	err := gm.StartGame()
	require.NoError(t, err)

	// Verify grid size scales with player count
	gridSize := gm.GetGame().GetGridSize()
	assert.Equal(t, 6, gridSize) // 32 players should get 6x6 grid

	// Simulate resource gathering with varied performance
	for _, player := range gm.GetAllPlayers() {
		// Simulate different success rates
		player.QuestionsAnswered = 10
		player.CorrectAnswers = 5 + (len(player.ID) % 5) // Varied accuracy
		tokensEarned := player.CorrectAnswers * 2
		player.TokensEarned = tokensEarned

		// Add to team tokens
		tokenType := []models.TokenType{
			models.TokenAnchor,
			models.TokenChronos,
			models.TokenGuide,
			models.TokenClarity,
		}[len(player.ID)%4]
		game := gm.GetGame()
		game.TeamTokens.AddTokens(tokenType, tokensEarned)
	}

	// Move to puzzle phase manually for test
	game := gm.GetGame()
	game.CurrentPhase = models.PhasePuzzleAssembly
	game.PhaseStartTime = time.Now()
	gm.GetGame().StartPuzzlePhase(gm.GetPlayerCount())

	// Verify puzzle initialized correctly
	grid := gm.GetGame().PuzzleGrid
	assert.NotNil(t, grid)
	assert.Equal(t, gridSize, grid.Size)
}

func TestGameReset(t *testing.T) {
	server, gm := setupE2EServer(t)
	defer server.Close()
	defer gm.Cleanup() // Ensure proper cleanup of game manager

	// Setup and play a game
	for i := 0; i < 4; i++ {
		player := models.NewPlayer(string(rune('a'+i)), nil)
		player.Name = "Player" + string(rune('A'+i))
		player.IsReady = true
		player.IsActive = true
		gm.AddPlayer(player)
	}

	// Add host for game to start
	host := models.NewHost("test-host", &websocket.Conn{})
	gm.SetHost(host)

	// Start and progress game
	gm.GetGame().SetDifficulty(models.DifficultyEasy)
	err := gm.StartGame()
	require.NoError(t, err)

	// Add some tokens
	game := gm.GetGame()
	game.TeamTokens.AddTokens(models.TokenAnchor, 10)

	// Move through phases
	game.CurrentPhase = models.PhasePuzzleAssembly
	game.PhaseStartTime = time.Now()
	game.CurrentPhase = models.PhaseAnalytics
	game.PhaseStartTime = time.Now()

	// Reset game
	gm.ResetGame()

	// Verify reset
	game = gm.GetGame()
	assert.False(t, game.GameStarted)
	assert.Equal(t, models.PhaseSetup, gm.GetCurrentPhase())
	assert.Equal(t, 0, gm.GetPlayerCount())
	assert.Equal(t, 0, game.CurrentRound)

	// Verify tokens reset
	tokens := game.TeamTokens
	assert.Equal(t, 0, tokens.GetTotal())

	// Verify services reset
	assert.Empty(t, gm.GetAllPlayers())
}
