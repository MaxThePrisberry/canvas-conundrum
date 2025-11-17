package integration_tests

import (
	"canvas-conundrum/config"
	"canvas-conundrum/handlers"
	"canvas-conundrum/services"
	"canvas-conundrum/test_helpers"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

// createTestTriviaFiles creates test trivia files for integration tests
func createTestTriviaFiles(t *testing.T) {
	categories := []string{"general", "geography", "history", "music", "science", "video_games"}
	difficulties := []string{"easy", "medium", "hard"}

	for _, category := range categories {
		for _, difficulty := range difficulties {
			dir := filepath.Join("trivia", category)
			err := os.MkdirAll(dir, 0755)
			require.NoError(t, err)

			response := map[string]interface{}{
				"response_code": 0,
				"results": []map[string]interface{}{
					{
						"category":          category,
						"type":              "multiple",
						"difficulty":        difficulty,
						"question":          fmt.Sprintf("Test %s %s question 1?", category, difficulty),
						"correct_answer":    "Correct Answer 1",
						"incorrect_answers": []string{"Wrong 1", "Wrong 2", "Wrong 3"},
					},
					{
						"category":          category,
						"type":              "multiple",
						"difficulty":        difficulty,
						"question":          fmt.Sprintf("Test %s %s question 2?", category, difficulty),
						"correct_answer":    "Correct Answer 2",
						"incorrect_answers": []string{"Wrong A", "Wrong B", "Wrong C"},
					},
					{
						"category":          category,
						"type":              "multiple",
						"difficulty":        difficulty,
						"question":          fmt.Sprintf("Test %s %s question 3?", category, difficulty),
						"correct_answer":    "Correct Answer 3",
						"incorrect_answers": []string{"Wrong X", "Wrong Y", "Wrong Z"},
					},
				},
			}

			data, err := json.Marshal(response)
			require.NoError(t, err)

			filename := filepath.Join(dir, fmt.Sprintf("%s.json", difficulty))
			err = os.WriteFile(filename, data, 0644)
			require.NoError(t, err)
		}
	}
}

// cleanupTestTriviaFiles removes test trivia files after integration tests
func cleanupTestTriviaFiles(t *testing.T) {
	err := os.RemoveAll("trivia")
	if err != nil {
		t.Logf("Warning: Failed to cleanup trivia files: %v", err)
	}
}

// setupTestServer creates a test server for integration tests - matches websocket_integration_test.go
func setupTestServer(t *testing.T) (*httptest.Server, func()) {
	// Get game manager singleton and reset it
	gm := services.GetGameInstance()

	// Cleanup and reset for this test
	gm.Cleanup()
	gm.ResetGame()

	// Setup services
	gm.SetBroadcastService(services.NewBroadcastService())
	gm.SetTriviaService(services.NewTriviaService())
	gm.SetPuzzleService(services.NewPuzzleService())
	gm.SetAnalyticsService(services.NewAnalyticsService())

	// Create router
	r := mux.NewRouter()
	r.HandleFunc("/ws", handlers.HandlePlayerWebSocket).Methods("GET")
	r.HandleFunc("/ws/host/{uuid}", handlers.HandleHostWebSocket).Methods("GET")

	// Create test server
	server := httptest.NewServer(r)

	// Return cleanup function that resets game
	cleanup := func() {
		server.Close()
		gm.Cleanup()   // Properly cleanup game manager
		gm.ResetGame() // Reset state after cleanup
	}

	return server, cleanup
}

// setupTestServerWithTrivia creates a test server with trivia files for tests requiring trivia
func setupTestServerWithTrivia(t *testing.T) (*httptest.Server, func()) {
	// Set test config directory to use faster round duration
	configDir, err := filepath.Abs("../config/test")
	require.NoError(t, err)
	originalConfigDir := os.Getenv("CANVAS_CONFIG_DIR")
	os.Setenv("CANVAS_CONFIG_DIR", configDir)

	// Reload config with test values
	err = config.LoadConfigFromDirectory(configDir)
	require.NoError(t, err)

	// Create trivia files first
	createTestTriviaFiles(t)

	// Get game manager singleton and reset it
	gm := services.GetGameInstance()
	gm.Cleanup()
	gm.ResetGame()

	// Setup services
	gm.SetBroadcastService(services.NewBroadcastService())

	// Setup trivia service with test questions
	triviaService := services.NewTriviaService()
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

	// Return cleanup function that resets game, config, and cleans up trivia files
	cleanup := func() {
		server.Close()
		gm.Cleanup()
		gm.ResetGame()
		cleanupTestTriviaFiles(t)

		// Restore original config
		if originalConfigDir != "" {
			os.Setenv("CANVAS_CONFIG_DIR", originalConfigDir)
			config.LoadConfigFromDirectory(originalConfigDir)
		} else {
			os.Unsetenv("CANVAS_CONFIG_DIR")
			// Reload default config
			config.LoadConfigFromDirectory("")
		}
	}

	return server, cleanup
}

// createAndConfigurePlayer creates a player, connects them, and configures their role/specialty
func createAndConfigurePlayer(t *testing.T, server *httptest.Server, name, role string, specialties []string) *test_helpers.TestPlayerClient {
	player := test_helpers.NewTestPlayerClient(t, server)
	err := player.Connect()
	require.NoError(t, err)

	player.ClearMessages()
	err = player.ConfigurePlayer(name, role, specialties)
	require.NoError(t, err)

	// Wait for lobby status update
	_, err = player.WaitForEvent(config.EventSetupToClientLobbyStatus, 2*time.Second)
	require.NoError(t, err)

	return player
}

// setupMinimalGameScenario sets up a game with host and minimum number of players configured
func setupMinimalGameScenario(t *testing.T, server *httptest.Server) (*test_helpers.TestHostClient, []*test_helpers.TestPlayerClient, func()) {
	// Connect host
	host := test_helpers.NewTestHostClient(t, server, config.HostUUID)
	err := host.Connect()
	require.NoError(t, err)

	// Connect minimum number of players (4)
	players := make([]*test_helpers.TestPlayerClient, 4)
	roles := []string{"art_enthusiast", "detective", "tourist", "janitor"}
	specialties := []string{"general", "history", "science", "geography"}

	for i := 0; i < 4; i++ {
		players[i] = test_helpers.NewTestPlayerClient(t, server)
		err = players[i].Connect()
		require.NoError(t, err)

		players[i].ClearMessages()
		err = players[i].ConfigurePlayer(fmt.Sprintf("Player%d", i+1), roles[i], []string{specialties[i]})
		require.NoError(t, err)

		// Wait for lobby status update (configuration is confirmed when this is received)
		_, err = players[i].WaitForEvent(config.EventSetupToClientLobbyStatus, 2*time.Second)
		require.NoError(t, err)
	}

	cleanup := func() {
		host.Close()
		for _, player := range players {
			player.Close()
		}
	}

	return host, players, cleanup
}

// waitForGameToStart waits for the game to start and reach resource gathering phase
func waitForGameToStart(t *testing.T, host *test_helpers.TestHostClient, players []*test_helpers.TestPlayerClient) {
	// Host starts the game
	err := host.StartGame("medium")
	require.NoError(t, err)

	// Wait for resource phase to start for all players
	for _, player := range players {
		_, err = player.WaitForEvent(config.EventResourceToClientPhaseStart, 5*time.Second)
		require.NoError(t, err)
	}
}

// advanceToResourcePhase advances the game from setup to resource gathering phase
func advanceToResourcePhase(t *testing.T, host *test_helpers.TestHostClient, players []*test_helpers.TestPlayerClient) {
	waitForGameToStart(t, host, players)
}

// simulateTriviaRound simulates a trivia round with all players answering correctly
func simulateTriviaRound(t *testing.T, players []*test_helpers.TestPlayerClient, stationHashes map[string]string) {
	// Wait for trivia question to be sent to all players
	// Use longer timeout for the first round since questions start after 5 seconds
	for _, player := range players {
		_, err := player.WaitForEvent(config.EventResourceToPlayerTriviaQuestion, 10*time.Second)
		require.NoError(t, err)
	}

	// Each player moves to their preferred station and answers
	stations := []string{"anchor", "chronos", "guide", "clarity"}
	for i, player := range players {
		station := stations[i%len(stations)]
		hash := stationHashes[station]

		// Send location update
		err := player.VerifyLocation(station, hash)
		require.NoError(t, err)

		// Submit answer (always first option for test simplicity)
		err = player.AnswerTrivia("", 0, 10.0)
		require.NoError(t, err)
	}

	// Wait for round to complete
	time.Sleep(1 * time.Second)
}

// getTestStationHashes returns the test station hashes
func getTestStationHashes() map[string]string {
	return map[string]string{
		"anchor":  config.HashAnchorStation,
		"chronos": config.HashChronosStation,
		"guide":   config.HashGuideStation,
		"clarity": config.HashClarityStation,
	}
}

// simulateResourceGatheringPhase waits for the resource gathering phase to complete naturally
// using the actual game flow with real WebSocket events and timers
func simulateResourceGatheringPhase(t *testing.T, host *test_helpers.TestHostClient, players []*test_helpers.TestPlayerClient) {
	// With test config: 5 rounds * 5 seconds each = 25 seconds total
	// Wait for resource phase to complete naturally - server will automatically
	// send PUZZLE_TO_CLIENT_PHASE_LOAD when resource gathering completes
	for _, player := range players {
		_, err := player.WaitForEvent(config.EventPuzzleToClientPhaseLoad, 40*time.Second)
		require.NoError(t, err)
	}

	// According to the specification, host needs to start the puzzle phase timer
	// to send PUZZLE_TO_CLIENT_PHASE_START to players
	err := host.StartPuzzlePhase()
	require.NoError(t, err)

	// Wait for the actual puzzle phase start event
	for _, player := range players {
		_, err := player.WaitForEvent(config.EventPuzzleToClientPhaseStart, 5*time.Second)
		require.NoError(t, err)
	}
}

// connectPlayersWithConfiguration connects specified number of players with given roles
func connectPlayersWithConfiguration(t *testing.T, server *httptest.Server, configs []PlayerConfig) []*test_helpers.TestPlayerClient {
	players := make([]*test_helpers.TestPlayerClient, len(configs))

	for i, playerConfig := range configs {
		players[i] = test_helpers.NewTestPlayerClient(t, server)
		err := players[i].Connect()
		require.NoError(t, err)

		players[i].ClearMessages()
		err = players[i].ConfigurePlayer(playerConfig.Name, playerConfig.Role, []string{playerConfig.Specialty})
		require.NoError(t, err)

		// Wait for lobby status update (configuration is confirmed when this is received)
		_, err = players[i].WaitForEvent(config.EventSetupToClientLobbyStatus, 2*time.Second)
		require.NoError(t, err)
	}

	return players
}

// PlayerConfig represents a player configuration for testing
type PlayerConfig struct {
	Name      string
	Role      string
	Specialty string
}

// GetDefaultPlayerConfigs returns default player configurations for testing
func GetDefaultPlayerConfigs(count int) []PlayerConfig {
	roles := []string{"art_enthusiast", "detective", "tourist", "janitor"}
	specialties := []string{"general", "history", "science", "geography", "music", "video_games"}

	configs := make([]PlayerConfig, count)
	for i := 0; i < count; i++ {
		configs[i] = PlayerConfig{
			Name:      fmt.Sprintf("Player%d", i+1),
			Role:      roles[i%len(roles)],
			Specialty: specialties[i%len(specialties)],
		}
	}

	return configs
}
