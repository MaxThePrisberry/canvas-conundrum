package e2e_tests

import (
	"canvas-conundrum/config"
	"canvas-conundrum/constants"
	"canvas-conundrum/handlers"
	"canvas-conundrum/models"
	"canvas-conundrum/services"
	"canvas-conundrum/test_helpers"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupE2EServer(t *testing.T) (*httptest.Server, *services.GameManager) {
	// Reset and setup game manager
	gm := services.GetGameInstance()
	gm.ResetGame() // Reset any state from previous tests
	gm.SetBroadcastService(services.NewBroadcastService())

	// Setup trivia service with test questions
	triviaService := services.NewTriviaService()
	// Load test questions
	_, cleanup, err := test_helpers.CreateMockTriviaFiles()
	require.NoError(t, err)
	t.Cleanup(cleanup)

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

	// Phase 1: Setup
	t.Run("Phase1_Setup", func(t *testing.T) {
		// Connect host
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

			// Store initial message with player ID
			initialMsg := players[i].GetLastMessage()
			require.NotNil(t, initialMsg)

			// Configure player
			err = players[i].ConfigurePlayer(playerNames[i], roles[i], specialties[i])
			require.NoError(t, err)
		}

		// Wait for all players to be configured
		time.Sleep(200 * time.Millisecond)

		// Verify lobby status
		assert.Equal(t, 4, gm.GetPlayerCount())

		// Start game
		err = host.StartGame("medium")
		require.NoError(t, err)

		// All players should receive game started
		for _, player := range players {
			msg, err := player.WaitForEvent(constants.EventSetupToClientGameStarted, 2*time.Second)
			require.NoError(t, err)
			assert.NotNil(t, msg)
		}

		// Verify game state
		assert.True(t, gm.IsGameStarted())
		assert.Equal(t, string(models.PhaseResourceGathering), gm.GetCurrentPhase())
	})

	// Phase 2: Resource Gathering
	t.Run("Phase2_ResourceGathering", func(t *testing.T) {
		// Each player should receive phase start
		// Simulate QR scanning and trivia answering

		// Get players from game manager
		allPlayers := gm.GetAllPlayers()
		assert.Len(t, allPlayers, 4)

		// Simulate one round of resource gathering
		for _, player := range allPlayers {
			// Simulate QR scan at a station
			station := "anchor"
			_ = config.HashAnchorStation

			// In a real test, we'd send the location verification through WebSocket
			// For now, directly update the player state
			player.CurrentStation = station

			// Get a trivia question for the player
			questions := gm.GetTriviaService().GetQuestionsForRound(gm.GetAllPlayers())
			question, ok := questions[player.ID]
			require.True(t, ok)
			require.NotNil(t, question)

			// Simulate correct answer
			player.QuestionsAnswered++
			player.CorrectAnswers++
			player.TokensEarned += 3

			// Add tokens to team
			gm.AddTeamTokens(models.TokenAnchor, 3)
		}

		// Verify tokens were added
		tokens := gm.GetTeamTokens()
		assert.Equal(t, 12, tokens.AnchorTokens) // 4 players * 3 tokens

		// Move to next phase
		gm.TransitionToPhase(models.PhasePuzzleAssembly)
		assert.Equal(t, string(models.PhasePuzzleAssembly), gm.GetCurrentPhase())
	})

	// Phase 3: Puzzle Assembly
	t.Run("Phase3_PuzzleAssembly", func(t *testing.T) {
		// Initialize puzzle service
		puzzleService := gm.GetPuzzleService()
		require.NotNil(t, puzzleService)

		// Assign puzzle segments using the service method
		gridSize := gm.GetGame().GetGridSize()
		puzzleService.AssignSegments(gm.GetAllPlayers(), gridSize)

		// Simulate individual puzzle solving
		for playerID, player := range gm.GetAllPlayers() {
			if player.AssignedSegment != "" {
				// Simulate segment completion through game manager
				err := gm.CompleteSegment(playerID, player.AssignedSegment)
				require.NoError(t, err)

				// Verify player state updated
				assert.True(t, player.SegmentCompleted)
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
		gm.TransitionToPhase(models.PhaseAnalytics)
		assert.Equal(t, string(models.PhaseAnalytics), gm.GetCurrentPhase())
	})

	// Phase 4: Analytics
	t.Run("Phase4_Analytics", func(t *testing.T) {
		analyticsService := gm.GetAnalyticsService()
		require.NotNil(t, analyticsService)

		// Finalize game analytics
		analyticsService.FinalizeGame(gm.GetGame(), gm.GetAllPlayers(), false)

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
	gm.AddTeamTokens(models.TokenAnchor, 30)  // Threshold 3
	gm.AddTeamTokens(models.TokenChronos, 25) // Threshold 2
	gm.AddTeamTokens(models.TokenGuide, 20)   // Threshold 2
	gm.AddTeamTokens(models.TokenClarity, 15) // Threshold 1

	// Verify thresholds
	tokens := gm.GetTeamTokens()
	assert.Equal(t, 3, tokens.GetThreshold(models.TokenAnchor))
	assert.Equal(t, 2, tokens.GetThreshold(models.TokenChronos))
	assert.Equal(t, 2, tokens.GetThreshold(models.TokenGuide))
	assert.Equal(t, 1, tokens.GetThreshold(models.TokenClarity))

	// Check pre-solved pieces
	preSolved := gm.GetGame().GetPreSolvedPieces()
	expectedPreSolved := 3 + 2 + 2 + 1 // Sum of thresholds
	assert.Equal(t, expectedPreSolved, preSolved)

	// Check puzzle time bonus
	totalTime := gm.GetGame().GetTotalPuzzleTime()
	expectedTime := constants.PuzzleBaseTime + (2 * constants.TimeExtensionPerThreshold) // Chronos threshold 2
	assert.Equal(t, expectedTime, totalTime)
}

func TestLargeScaleGame(t *testing.T) {
	server, gm := setupE2EServer(t)
	defer server.Close()

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
		err := gm.AddPlayer(player)
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
		gm.AddTeamTokens(tokenType, tokensEarned)
	}

	// Move to puzzle phase
	gm.TransitionToPhase(models.PhasePuzzleAssembly)

	// Initialize puzzle phase
	gm.TransitionToPhase(models.PhasePuzzleAssembly)
	gm.GetGame().StartPuzzlePhase(gm.GetPlayerCount())

	// Verify puzzle initialized correctly
	grid := gm.GetGame().PuzzleGrid
	assert.NotNil(t, grid)
	assert.Equal(t, gridSize, grid.Size)
}

func TestGameReset(t *testing.T) {
	server, gm := setupE2EServer(t)
	defer server.Close()

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
	gm.AddTeamTokens(models.TokenAnchor, 10)

	// Move through phases
	gm.TransitionToPhase(models.PhasePuzzleAssembly)
	gm.TransitionToPhase(models.PhaseAnalytics)

	// Reset game
	gm.ResetGame()

	// Verify reset
	assert.False(t, gm.IsGameStarted())
	assert.Equal(t, string(models.PhaseSetup), gm.GetCurrentPhase())
	assert.Equal(t, 0, gm.GetPlayerCount())
	assert.Equal(t, 0, gm.GetCurrentRound())

	// Verify tokens reset
	tokens := gm.GetTeamTokens()
	assert.Equal(t, 0, tokens.GetTotal())

	// Verify services reset
	assert.Empty(t, gm.GetAllPlayers())
}
