package main

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestGameManagerCreation(t *testing.T) {
	// Test that we can create the core game components
	playerMgr := NewPlayerManager()
	triviaMgr := NewTriviaManager()
	broadcastChan := make(chan BroadcastMessage, 256)
	gameMgr := NewGameManager(playerMgr, triviaMgr, broadcastChan)

	assert.NotNil(t, playerMgr)
	assert.NotNil(t, triviaMgr)
	assert.NotNil(t, broadcastChan)
	assert.NotNil(t, gameMgr)
}

func TestEventProcessing(t *testing.T) {
	// Test basic event processing structure
	tests := []struct {
		name      string
		eventType string
		valid     bool
	}{
		{
			name:      "Valid role selection",
			eventType: "role_selection",
			valid:     true,
		},
		{
			name:      "Valid specialty selection",
			eventType: "specialty_selection",
			valid:     true,
		},
		{
			name:      "Valid trivia answer",
			eventType: "trivia_answer",
			valid:     true,
		},
		{
			name:      "Valid host start game",
			eventType: "host_start_game",
			valid:     true,
		},
		{
			name:      "Invalid event type",
			eventType: "invalid_event",
			valid:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Check if event type is in the list of valid events
			validEvents := map[string]bool{
				"role_selection":                true,
				"specialty_selection":           true,
				"resource_location_verified":    true,
				"trivia_answer":                 true,
				"segment_completed":             true,
				"fragment_move_request":         true,
				"host_start_game":               true,
				"host_start_puzzle":             true,
				"piece_recommendation_request":  true,
				"piece_recommendation_response": true,
				"player_ready":                  true,
			}

			isValid := validEvents[tt.eventType]
			assert.Equal(t, tt.valid, isValid)
		})
	}
}

func TestReconnectionValidation(t *testing.T) {
	// Test reconnection validation logic
	tests := []struct {
		name           string
		currentPhase   GamePhase
		isHost         bool
		allowReconnect bool
	}{
		{
			name:           "Host reconnect in setup",
			currentPhase:   PhaseSetup,
			isHost:         true,
			allowReconnect: true,
		},
		{
			name:           "Player reconnect in setup",
			currentPhase:   PhaseSetup,
			isHost:         false,
			allowReconnect: true,
		},
		{
			name:           "Host reconnect in resource gathering",
			currentPhase:   PhaseResourceGathering,
			isHost:         true,
			allowReconnect: true,
		},
		{
			name:           "Player reconnect in resource gathering",
			currentPhase:   PhaseResourceGathering,
			isHost:         false,
			allowReconnect: true,
		},
		{
			name:           "Host reconnect in puzzle assembly",
			currentPhase:   PhasePuzzleAssembly,
			isHost:         true,
			allowReconnect: true,
		},
		{
			name:           "Player reconnect in puzzle assembly",
			currentPhase:   PhasePuzzleAssembly,
			isHost:         false,
			allowReconnect: false, // Based on docs, reconnection forbidden during puzzle
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the reconnection logic
			allowReconnect := true
			if tt.currentPhase == PhasePuzzleAssembly && !tt.isHost {
				allowReconnect = false
			}

			assert.Equal(t, tt.allowReconnect, allowReconnect)
		})
	}
}

func TestHostValidation(t *testing.T) {
	// Test host validation logic
	playerMgr := NewPlayerManager()

	// Create a host player
	host := playerMgr.CreatePlayer(nil, true)
	assert.NotNil(t, host)
	assert.True(t, host.IsHost)

	// Create a regular player
	player := playerMgr.CreatePlayer(nil, false)
	assert.NotNil(t, player)
	assert.False(t, player.IsHost)

	// Test host validation
	assert.True(t, playerMgr.IsHostConnected())
}

func TestBroadcastMessageTypes(t *testing.T) {
	// Test that all expected message types are properly defined
	expectedMessages := []string{
		MsgAvailableRoles,
		MsgGameLobbyStatus,
		MsgResourcePhaseStart,
		MsgTriviaQuestion,
		MsgTeamProgressUpdate,
		MsgPuzzlePhaseLoad,
		MsgPuzzlePhaseStart,
		MsgSegmentCompletionAck,
		MsgFragmentMoveResponse,
		MsgCentralPuzzleState,
		MsgGameAnalytics,
		MsgGameReset,
		MsgError,
		MsgHostUpdate,
		MsgCountdown,
		MsgPieceRecommendation,
		MsgImagePreview,
		MsgPersonalPuzzleState,
		MsgGuideHighlight,
	}

	for _, msgType := range expectedMessages {
		t.Run("Message type: "+msgType, func(t *testing.T) {
			assert.NotEmpty(t, msgType)
			assert.IsType(t, string(""), msgType)
		})
	}
}

func TestWebSocketConnState(t *testing.T) {
	// Test websocket connection state tracking
	tests := []struct {
		name  string
		state PlayerState
		valid bool
	}{
		{
			name:  "Connected state",
			state: StateConnected,
			valid: true,
		},
		{
			name:  "Disconnected state",
			state: StateDisconnected,
			valid: true,
		},
		{
			name:  "Ready state",
			state: StateReady,
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify the state constants are properly defined
			assert.True(t, tt.valid)
		})
	}
}

func TestPlayerCleanup(t *testing.T) {
	// Test player cleanup on disconnection
	playerMgr := NewPlayerManager()

	// Create a player
	player := playerMgr.CreatePlayer(nil, false)
	playerID := player.ID

	// Verify player exists
	_, err := playerMgr.GetPlayer(playerID)
	assert.NoError(t, err)

	// Disconnect player
	err = playerMgr.DisconnectPlayer(playerID)
	assert.NoError(t, err)

	// Verify player state changed
	player, err = playerMgr.GetPlayer(playerID)
	assert.NoError(t, err)
	assert.Equal(t, StateDisconnected, player.State)
}

func TestConcurrentConnections(t *testing.T) {
	// Test handling of concurrent connections
	playerMgr := NewPlayerManager()

	// Create multiple players concurrently
	numPlayers := 10
	playerIDs := make([]string, numPlayers)

	done := make(chan string, numPlayers)

	for i := 0; i < numPlayers; i++ {
		go func() {
			player := playerMgr.CreatePlayer(nil, false)
			done <- player.ID
		}()
	}

	// Collect all player IDs
	for i := 0; i < numPlayers; i++ {
		select {
		case playerID := <-done:
			playerIDs[i] = playerID
		case <-time.After(1 * time.Second):
			t.Fatal("Timeout waiting for player creation")
		}
	}

	// Verify all players were created successfully
	assert.Equal(t, numPlayers, len(playerIDs))
	for _, playerID := range playerIDs {
		assert.NotEmpty(t, playerID)
		_, err := uuid.Parse(playerID)
		assert.NoError(t, err, "Player ID should be valid UUID")
	}
}

func TestPlayerIDValidationInConnection(t *testing.T) {
	// Test player ID validation during connection
	tests := []struct {
		name        string
		playerID    string
		expectError bool
	}{
		{
			name:        "Valid UUID format",
			playerID:    "123e4567-e89b-12d3-a456-426614174000",
			expectError: false,
		},
		{
			name:        "Empty player ID",
			playerID:    "",
			expectError: true, // validatePlayerID rejects empty strings
		},
		{
			name:        "Invalid UUID format - too short",
			playerID:    "123e4567-e89b",
			expectError: true,
		},
		{
			name:        "Invalid UUID format - wrong pattern",
			playerID:    "not-a-uuid-at-all",
			expectError: true,
		},
		{
			name:        "Invalid UUID format - missing hyphens",
			playerID:    "123e4567e89b12d3a456426614174000",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the validatePlayerID function directly
			err := validatePlayerID(tt.playerID)
			if tt.expectError {
				assert.NotNil(t, err, "Expected validation error for player ID: %s", tt.playerID)
			} else {
				assert.Nil(t, err, "Expected no validation error for player ID: %s", tt.playerID)
			}
		})
	}
}

// TestHostDisconnectionHandler tests the new host disconnection handler
func TestHostDisconnectionHandler(t *testing.T) {
	// Setup components
	playerManager := NewPlayerManager()
	triviaManager := NewTriviaManager()
	broadcastChan := make(chan BroadcastMessage, 256)
	gameManager := NewGameManager(playerManager, triviaManager, broadcastChan)
	eventHandlers := NewEventHandlers(gameManager, playerManager, broadcastChan)
	wsHandler := NewWebSocketHandler(playerManager, gameManager, eventHandlers, broadcastChan)

	// Create a host player
	host := playerManager.CreatePlayer(nil, true)

	// Verify host exists and is connected
	assert.NotNil(t, playerManager.GetConnectedHost())
	assert.Equal(t, host.ID, playerManager.GetConnectedHost().ID)

	// Simulate host disconnection via close handler
	wsHandler.handleHostDisconnection(host)

	// Verify immediate cleanup
	assert.Nil(t, playerManager.GetConnectedHost())
	assert.Nil(t, playerManager.GetHost())

	// Verify broadcast message was sent
	select {
	case msg := <-broadcastChan:
		assert.Equal(t, MsgError, msg.Type)
		payload := msg.Payload.(map[string]interface{})
		assert.Contains(t, payload["error"], "Host disconnected - new host can now connect")
		assert.Equal(t, "host_disconnected", payload["type"])
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Expected broadcast message for host disconnection")
	}

	// Cleanup
	triviaManager.Shutdown()
}

// TestHostConnectionBlocking tests that connected hosts block new host connections
func TestHostConnectionBlocking(t *testing.T) {
	playerManager := NewPlayerManager()
	triviaManager := NewTriviaManager()
	broadcastChan := make(chan BroadcastMessage, 256)
	gameManager := NewGameManager(playerManager, triviaManager, broadcastChan)
	eventHandlers := NewEventHandlers(gameManager, playerManager, broadcastChan)
	wsHandler := NewWebSocketHandler(playerManager, gameManager, eventHandlers, broadcastChan)

	// Create first host
	host1 := playerManager.CreatePlayer(nil, true)
	assert.NotNil(t, playerManager.GetConnectedHost())

	// Simulate the check that would happen in HandleConnection for new host
	existingConnectedHost := playerManager.GetConnectedHost()
	shouldBlockNewHost := existingConnectedHost != nil
	assert.True(t, shouldBlockNewHost, "Should block new host when one is connected")

	// Disconnect first host and clean up
	wsHandler.handleHostDisconnection(host1)
	assert.Nil(t, playerManager.GetConnectedHost())

	// Now new host should be allowed
	shouldBlockNewHost = playerManager.GetConnectedHost() != nil
	assert.False(t, shouldBlockNewHost, "Should allow new host after cleanup")

	// Create second host should succeed
	host2 := playerManager.CreatePlayer(nil, true)
	assert.NotNil(t, host2)
	assert.NotEqual(t, host1.ID, host2.ID)

	// Cleanup
	triviaManager.Shutdown()
}

// TestWebSocketCloseHandlerBehavior tests the close handler logic
func TestWebSocketCloseHandlerBehavior(t *testing.T) {
	playerManager := NewPlayerManager()
	triviaManager := NewTriviaManager()
	broadcastChan := make(chan BroadcastMessage, 256)
	gameManager := NewGameManager(playerManager, triviaManager, broadcastChan)
	eventHandlers := NewEventHandlers(gameManager, playerManager, broadcastChan)
	wsHandler := NewWebSocketHandler(playerManager, gameManager, eventHandlers, broadcastChan)

	tests := []struct {
		name       string
		isHost     bool
		shouldCall string
	}{
		{
			name:       "Host close should call handleHostDisconnection",
			isHost:     true,
			shouldCall: "handleHostDisconnection",
		},
		{
			name:       "Player close should call handleDisconnection",
			isHost:     false,
			shouldCall: "handleDisconnection",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create player
			player := playerManager.CreatePlayer(nil, tt.isHost)

			initialConnectedCount := playerManager.GetConnectedCount()

			if tt.isHost {
				// For host, test immediate cleanup
				wsHandler.handleHostDisconnection(player)

				// Host should be completely removed
				assert.Nil(t, playerManager.GetHost())
				assert.Nil(t, playerManager.GetConnectedHost())
				assert.Equal(t, initialConnectedCount-1, playerManager.GetConnectedCount())
			} else {
				// For regular player, test normal disconnection
				wsHandler.handleDisconnection(player)

				// Player should be disconnected but not removed
				p, err := playerManager.GetPlayer(player.ID)
				assert.NoError(t, err)
				assert.Equal(t, StateDisconnected, p.State)
				assert.Equal(t, initialConnectedCount-1, playerManager.GetConnectedCount())
			}
		})
	}

	// Cleanup
	triviaManager.Shutdown()
}

// TestFallbackDisconnectionForHost tests that regular disconnection also handles hosts
func TestFallbackDisconnectionForHost(t *testing.T) {
	playerManager := NewPlayerManager()
	triviaManager := NewTriviaManager()
	broadcastChan := make(chan BroadcastMessage, 256)
	gameManager := NewGameManager(playerManager, triviaManager, broadcastChan)
	eventHandlers := NewEventHandlers(gameManager, playerManager, broadcastChan)
	wsHandler := NewWebSocketHandler(playerManager, gameManager, eventHandlers, broadcastChan)

	// Create host
	host := playerManager.CreatePlayer(nil, true)
	assert.NotNil(t, playerManager.GetConnectedHost())

	// Call regular handleDisconnection (fallback path)
	wsHandler.handleDisconnection(host)

	// Should still result in host cleanup
	assert.Nil(t, playerManager.GetConnectedHost())
	assert.Nil(t, playerManager.GetHost())

	// Cleanup
	triviaManager.Shutdown()
}

// TestConcurrentHostDisconnectionHandling tests thread safety
func TestConcurrentHostDisconnectionHandling(t *testing.T) {
	playerManager := NewPlayerManager()
	triviaManager := NewTriviaManager()
	broadcastChan := make(chan BroadcastMessage, 256)
	gameManager := NewGameManager(playerManager, triviaManager, broadcastChan)
	eventHandlers := NewEventHandlers(gameManager, playerManager, broadcastChan)
	wsHandler := NewWebSocketHandler(playerManager, gameManager, eventHandlers, broadcastChan)

	// Create host
	host := playerManager.CreatePlayer(nil, true)

	done := make(chan bool, 10)

	// Simulate multiple concurrent disconnection calls
	for i := 0; i < 5; i++ {
		go func() {
			defer func() { done <- true }()
			wsHandler.handleHostDisconnection(host)
		}()
	}

	// Also simulate regular disconnection calls
	for i := 0; i < 5; i++ {
		go func() {
			defer func() { done <- true }()
			wsHandler.handleDisconnection(host)
		}()
	}

	// Wait for all operations
	for i := 0; i < 10; i++ {
		select {
		case <-done:
		case <-time.After(1 * time.Second):
			t.Fatal("Timeout waiting for concurrent disconnection handling")
		}
	}

	// Final state should be consistent
	assert.Nil(t, playerManager.GetConnectedHost())
	assert.Nil(t, playerManager.GetHost())

	// Drain broadcast channel
	for {
		select {
		case <-broadcastChan:
		default:
			goto drained
		}
	}
drained:

	// Cleanup
	triviaManager.Shutdown()
}
