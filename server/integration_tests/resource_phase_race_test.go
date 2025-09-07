package integration_tests

import (
	"canvas-conundrum/config"
	"canvas-conundrum/handlers"
	"canvas-conundrum/services"
	"canvas-conundrum/test_helpers"
	"fmt"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupResourcePhaseTestServer(t *testing.T) (*httptest.Server, *services.GameManager, func()) {
	// Reset and setup game manager
	gm := services.GetGameInstance()
	gm.Cleanup()
	gm.ResetGame()

	// Setup all real services (not mocks)
	gm.SetBroadcastService(services.NewBroadcastService())

	// Setup trivia service - use the actual service which will have empty pools but won't crash
	triviaService := services.NewTriviaService()
	gm.SetTriviaService(triviaService)

	gm.SetPuzzleService(services.NewPuzzleService())
	gm.SetAnalyticsService(services.NewAnalyticsService())

	// Create router with real handlers
	r := mux.NewRouter()
	r.HandleFunc("/ws", handlers.HandlePlayerWebSocket).Methods("GET")
	r.HandleFunc("/ws/host/{uuid}", handlers.HandleHostWebSocket).Methods("GET")

	server := httptest.NewServer(r)

	cleanup := func() {
		server.Close()
		gm.Cleanup()
		gm.ResetGame()
	}

	return server, gm, cleanup
}

// TestResourcePhaseStartRaceCondition reproduces the exact scenario from Issue #10
// where players get disconnected with "websocket: close 1005" when resource phase starts
func TestResourcePhaseStartRaceCondition(t *testing.T) {
	server, gm, cleanup := setupResourcePhaseTestServer(t)
	defer cleanup()

	// Track websocket errors
	var websocketErrors []error
	var errorsMu sync.Mutex

	// Create 4 player WebSocket connections using the test helper
	players := make([]*test_helpers.TestWebSocketClient, 4)
	playerNames := []string{"Maxwell", "White", "Babe", "Alice"}

	// Connect all players
	for i := 0; i < 4; i++ {
		client := test_helpers.NewTestWebSocketClient(t, server, "/ws")
		err := client.Connect()
		require.NoError(t, err)
		players[i] = client

		// Monitor for websocket errors on this connection
		go func(playerIdx int, playerName string, client *test_helpers.TestWebSocketClient) {
			// Keep the connection alive and monitor for errors
			time.Sleep(10 * time.Second) // Monitor for the duration of the test

			// Check for read errors that occurred during the test
			defer func() {
				// Use defer to catch any errors that occurred
				if r := recover(); r != nil {
					errorsMu.Lock()
					websocketErrors = append(websocketErrors, fmt.Errorf("Player %s (%d) panic: %v", playerName, playerIdx, r))
					errorsMu.Unlock()
				}
			}()
		}(i, playerNames[i], client)

		// Let connection stabilize
		time.Sleep(50 * time.Millisecond)
	}

	// Get player IDs from the game manager after connections are established
	time.Sleep(200 * time.Millisecond)
	allPlayers := gm.GetAllPlayers()
	require.Equal(t, 4, len(allPlayers))

	// Configure each player with role and mark as ready
	for i, client := range players {
		// Configure player using the test helper
		configPayload := map[string]interface{}{
			"playerName": playerNames[i],
			"roleType":   "anchor", // Use a valid role
		}

		err := client.SendMessage(config.EventSetupToServerPlayerConfiguration, configPayload)
		require.NoError(t, err)

		// Mark player as ready in the game manager directly
		playerID := client.GetPlayerID()
		if player, exists := gm.GetPlayer(playerID); exists {
			player.IsReady = true
			player.Name = playerNames[i]
		}
	}

	// Wait for all players to be configured
	time.Sleep(300 * time.Millisecond)

	// Connect host using test helper
	host := test_helpers.NewTestHostClient(t, server, config.HostUUID)
	err := host.Connect()
	require.NoError(t, err)
	defer host.Close()

	// Wait for host connection to stabilize
	time.Sleep(200 * time.Millisecond)

	// Start the game - this is the critical moment that triggers the race condition
	t.Logf("Starting game - this should trigger StartResourceRound after 5 seconds")
	err = host.StartGame("medium")
	require.NoError(t, err)

	// This is where the bug happens:
	// 1. StartGame() transitions to resource phase
	// 2. After 5 seconds, StartResourceRound() is called in a goroutine
	// 3. StartResourceRound stores `players := gm.players` (reference to map)
	// 4. Releases mutex
	// 5. BroadcastTriviaQuestions iterates over the map without lock protection
	// 6. Race condition causes websocket close 1005 errors

	// Wait for the critical period when StartResourceRound is executing
	t.Logf("Waiting for StartResourceRound to be triggered...")
	time.Sleep(6 * time.Second) // Wait past the 5-second delay

	// Send some messages during the critical period to increase chance of race condition
	for i, client := range players {
		if client != nil {
			err := client.SendMessage("SYSTEM_PING", map[string]interface{}{
				"clientTimestamp": time.Now().Format(time.RFC3339),
				"sequenceNumber":  i,
			})
			if err != nil {
				t.Logf("Error sending ping to player %d: %v", i, err)
			}
		}
	}

	// Wait a bit more to let any errors manifest
	time.Sleep(2 * time.Second)

	// Check if any players lost connection or have read errors
	errorsMu.Lock()
	defer errorsMu.Unlock()

	// Also check the test client's built-in error tracking
	for i, client := range players {
		if client != nil {
			// Use the test helper's error checking
			defer func(playerIdx int, playerClient *test_helpers.TestWebSocketClient) {
				// This will detect websocket read errors including close 1005
				playerClient.AssertNoErrors(t)
			}(i, client)
		}
	}

	if len(websocketErrors) > 0 {
		t.Errorf("Detected websocket errors during resource phase start:")
		for _, err := range websocketErrors {
			t.Logf("  %v", err)
		}

		// Check for the specific "close 1005" error mentioned in the issue
		for _, err := range websocketErrors {
			if strings.Contains(err.Error(), "close 1005") {
				t.Errorf("Found the specific 'close 1005' error from Issue #10: %v", err)
			}
		}
	}

	// Clean up connections
	for _, player := range players {
		if player != nil {
			player.Close()
		}
	}
}

// TestResourcePhaseStartConcurrentModification tests concurrent modification
// of the players map while StartResourceRound is executing
func TestResourcePhaseStartConcurrentModification(t *testing.T) {
	server, gm, cleanup := setupResourcePhaseTestServer(t)
	defer cleanup()

	// Create initial players using test helpers
	players := make([]*test_helpers.TestWebSocketClient, 3)
	for i := 0; i < 3; i++ {
		client := test_helpers.NewTestWebSocketClient(t, server, "/ws")
		err := client.Connect()
		require.NoError(t, err)
		players[i] = client
	}

	time.Sleep(200 * time.Millisecond)

	// Configure players and mark them ready
	for i, client := range players {
		configPayload := map[string]interface{}{
			"playerName": fmt.Sprintf("Player%d", i),
			"roleType":   "anchor",
		}

		err := client.SendMessage(config.EventSetupToServerPlayerConfiguration, configPayload)
		require.NoError(t, err)

		// Mark player as ready in the game manager directly
		playerID := client.GetPlayerID()
		if player, exists := gm.GetPlayer(playerID); exists {
			player.IsReady = true
			player.Name = fmt.Sprintf("Player%d", i)
		}
	}

	// Connect host using test helper
	host := test_helpers.NewTestHostClient(t, server, config.HostUUID)
	err := host.Connect()
	require.NoError(t, err)
	defer host.Close()

	time.Sleep(200 * time.Millisecond)

	// Track any panics or races
	var panicOccurred int32

	// Start monitoring for concurrent access
	go func() {
		defer func() {
			if r := recover(); r != nil {
				atomic.StoreInt32(&panicOccurred, 1)
				t.Logf("Panic detected during concurrent operations: %v", r)
			}
		}()

		// Wait for game to start and resource round to begin
		time.Sleep(6 * time.Second)

		// Try to add a new player while StartResourceRound might be executing
		newClient := test_helpers.NewTestWebSocketClient(t, server, "/ws")
		err := newClient.Connect()
		if err != nil {
			t.Logf("Expected error connecting new player during game: %v", err)
		} else {
			newClient.Close()
		}

		// Try to modify existing players - this creates the race condition
		for playerID, player := range gm.GetAllPlayers() {
			// This could race with the map access in BroadcastTriviaQuestions
			player.TokensEarned++
			_ = playerID
		}
	}()

	// Start the game
	err = host.StartGame("medium")
	require.NoError(t, err)

	// Wait for operations to complete
	time.Sleep(8 * time.Second)

	// Check results
	assert.Equal(t, int32(0), atomic.LoadInt32(&panicOccurred), "No panics should occur")

	// Clean up
	for _, client := range players {
		if client != nil {
			client.Close()
		}
	}
}
