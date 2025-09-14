package handlers

import (
	"canvas-conundrum/config"
	"canvas-conundrum/models"
	"canvas-conundrum/services"
	"canvas-conundrum/test_helpers"
	"canvas-conundrum/utils"
	"encoding/json"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MessageCapture represents a captured WebSocket message
type MessageCapture struct {
	Event   string
	Payload map[string]interface{}
	Raw     []byte
}

// HostReconnectionTestHelper provides utilities for testing host reconnection
type HostReconnectionTestHelper struct {
	gameManager      *services.GameManager
	broadcastService *services.BroadcastService
	host             *models.Host
	capturedMessages []MessageCapture
}

// NewHostReconnectionTestHelper creates a new test helper
func NewHostReconnectionTestHelper(t *testing.T) *HostReconnectionTestHelper {
	// Reset game manager for clean test
	services.ResetGameManagerInstance()

	gameManager := services.GetGameInstance()
	broadcastService := services.NewBroadcastService()
	gameManager.SetBroadcastService(broadcastService)

	return &HostReconnectionTestHelper{
		gameManager:      gameManager,
		broadcastService: broadcastService,
		capturedMessages: make([]MessageCapture, 0),
	}
}

// CreateAndConnectHost creates a host and connects it
func (h *HostReconnectionTestHelper) CreateAndConnectHost(t *testing.T) {
	h.host = test_helpers.CreateTestHost(config.HostUUID)

	// Set up a mock connection for HasConnection() to return true
	h.host.Connection = &websocket.Conn{}

	// Override the Send channel to capture messages
	h.host.Send = make(chan []byte, 256)

	isReconnection, err := h.gameManager.SetHost(h.host)
	require.NoError(t, err)
	require.False(t, isReconnection, "Initial connection should not be a reconnection")
}

// DisconnectHost simulates host disconnection
func (h *HostReconnectionTestHelper) DisconnectHost(t *testing.T) {
	if h.host != nil {
		h.gameManager.RemoveHost()
		// Clear any existing messages
		h.DrainMessages()
	}
}

// ReconnectHost simulates host reconnection
func (h *HostReconnectionTestHelper) ReconnectHost(t *testing.T) bool {
	// Create new host with same ID but new connection
	newHost := test_helpers.CreateTestHost(config.HostUUID)

	// Set up a mock connection for HasConnection() to return true
	newHost.Connection = &websocket.Conn{}

	// Set up a new Send channel to capture messages
	newSendChannel := make(chan []byte, 256)
	newHost.Send = newSendChannel

	isReconnection, err := h.gameManager.SetHost(newHost)
	require.NoError(t, err)

	// For reconnections, the game manager keeps the old host object but updates its Connection
	// We need to get the actual host object from the game manager and update its Send channel
	if isReconnection {
		actualHost := h.gameManager.GetHost()
		if actualHost != nil {
			// Update the actual host's Send channel to our new one so we can capture messages
			actualHost.Send = newSendChannel
			h.host = actualHost
		}
	} else {
		h.host = newHost
	}

	return isReconnection
}

// DrainMessages collects all messages from the host's Send channel
func (h *HostReconnectionTestHelper) DrainMessages() {
	h.capturedMessages = make([]MessageCapture, 0)

	if h.host == nil || h.host.Send == nil {
		return
	}

	// Drain all messages with a timeout
	timeout := time.After(100 * time.Millisecond)

	for {
		select {
		case msg := <-h.host.Send:
			capture := h.parseMessage(msg)
			if capture != nil {
				h.capturedMessages = append(h.capturedMessages, *capture)
			}
		case <-timeout:
			return
		default:
			return
		}
	}
}

// WaitForMessages waits for a specific number of messages with timeout
func (h *HostReconnectionTestHelper) WaitForMessages(expectedCount int, timeoutMs int) {
	h.capturedMessages = make([]MessageCapture, 0)

	if h.host == nil || h.host.Send == nil {
		return
	}

	timeout := time.After(time.Duration(timeoutMs) * time.Millisecond)

	for len(h.capturedMessages) < expectedCount {
		select {
		case msg := <-h.host.Send:
			capture := h.parseMessage(msg)
			if capture != nil {
				// Debug: log each message as we receive it
				// fmt.Printf("[DEBUG] Received message: %s\n", capture.Event)
				h.capturedMessages = append(h.capturedMessages, *capture)
			}
		case <-timeout:
			// Debug: log when we timeout
			// fmt.Printf("[DEBUG] Timeout waiting for messages. Got %d/%d\n", len(h.capturedMessages), expectedCount)
			return
		}
	}
}

// parseMessage parses a raw message into MessageCapture
func (h *HostReconnectionTestHelper) parseMessage(rawMsg []byte) *MessageCapture {
	msg, err := utils.ParseMessage(rawMsg)
	if err != nil {
		return nil
	}

	var payload map[string]interface{}
	if len(msg.Payload) > 0 {
		json.Unmarshal(msg.Payload, &payload)
	}

	return &MessageCapture{
		Event:   msg.Event,
		Payload: payload,
		Raw:     rawMsg,
	}
}

// GetCapturedMessages returns all captured messages
func (h *HostReconnectionTestHelper) GetCapturedMessages() []MessageCapture {
	return h.capturedMessages
}

// AssertMessageReceived checks if a specific event was received
func (h *HostReconnectionTestHelper) AssertMessageReceived(t *testing.T, expectedEvent string) *MessageCapture {
	for _, msg := range h.capturedMessages {
		if msg.Event == expectedEvent {
			return &msg
		}
	}

	t.Errorf("Expected event %s was not received", expectedEvent)
	t.Logf("Received messages: %v", h.getReceivedEventNames())
	return nil
}

// AssertMessageCount checks if the correct number of messages were received
func (h *HostReconnectionTestHelper) AssertMessageCount(t *testing.T, expectedCount int) {
	actualCount := len(h.capturedMessages)
	if actualCount != expectedCount {
		t.Errorf("Expected %d messages, got %d", expectedCount, actualCount)
		t.Logf("Received messages: %v", h.getReceivedEventNames())
	}
}

// getReceivedEventNames returns a list of received event names for logging
func (h *HostReconnectionTestHelper) getReceivedEventNames() []string {
	events := make([]string, len(h.capturedMessages))
	for i, msg := range h.capturedMessages {
		events[i] = msg.Event
	}
	return events
}

// SetPhase sets the game to a specific phase
func (h *HostReconnectionTestHelper) SetPhase(phase models.GamePhase) {
	game := h.gameManager.GetGame()
	game.CurrentPhase = phase
}

// AddTestPlayers adds some test players to the game
func (h *HostReconnectionTestHelper) AddTestPlayers(t *testing.T, count int) {
	for i := 0; i < count; i++ {
		player := test_helpers.CreateTestPlayer("player-" + string(rune('1'+i)))
		player.Name = "Test Player " + string(rune('1'+i))
		player.Role = models.RoleDetective
		player.IsReady = true
		player.IsActive = true

		_, err := h.gameManager.AddPlayer(player)
		require.NoError(t, err)
	}
}

// TestHostReconnectionPhase0Setup tests host reconnection during setup phase
func TestHostReconnectionPhase0Setup(t *testing.T) {
	helper := NewHostReconnectionTestHelper(t)

	// Ensure we're in setup phase
	helper.SetPhase(models.PhaseSetup)

	// Add some test players
	helper.AddTestPlayers(t, 2)

	t.Run("Initial Connection", func(t *testing.T) {
		helper.CreateAndConnectHost(t)

		// Simulate the flow that happens in HandleHostWebSocket
		sendHostConnectionConfirmed(helper.host, false)

		// Wait for messages
		helper.WaitForMessages(2, 200) // CONNECTION_CONFIRMED + PLAYER_ROSTER

		// Should receive at least the connection confirmed message
		helper.AssertMessageReceived(t, config.EventSetupToHostConnectionConfirmed)
		// BroadcastLobbyStatus is called, which should send PLAYER_ROSTER
		helper.AssertMessageReceived(t, config.EventSetupToHostPlayerRoster)

		t.Logf("Initial connection: received %d messages", len(helper.GetCapturedMessages()))
	})

	t.Run("First Reconnection", func(t *testing.T) {
		helper.DisconnectHost(t)

		isReconnection := helper.ReconnectHost(t)
		assert.True(t, isReconnection, "Should detect reconnection")

		// Simulate the flow that happens in HandleHostWebSocket for reconnection
		sendHostConnectionConfirmed(helper.host, true)
		sendHostPhaseRestoration(helper.host) // This should send PLAYER_ROSTER again

		// Wait for messages
		helper.WaitForMessages(2, 200) // CONNECTION_CONFIRMED + PLAYER_ROSTER

		// Verify we receive the expected messages
		helper.AssertMessageReceived(t, config.EventSetupToHostConnectionConfirmed)
		helper.AssertMessageReceived(t, config.EventSetupToHostPlayerRoster)

		t.Logf("First reconnection: received %d messages", len(helper.GetCapturedMessages()))
	})

	t.Run("Multiple Reconnections", func(t *testing.T) {
		// This is the scenario mentioned in the GitHub issue comments
		for i := 2; i <= 4; i++ {
			helper.DisconnectHost(t)

			isReconnection := helper.ReconnectHost(t)
			assert.True(t, isReconnection, "Should detect reconnection #%d", i)

			// Simulate the reconnection flow
			sendHostConnectionConfirmed(helper.host, true)
			sendHostPhaseRestoration(helper.host)

			// Wait for messages
			helper.WaitForMessages(2, 200)

			// Verify we still receive all expected messages
			connectionMsg := helper.AssertMessageReceived(t, config.EventSetupToHostConnectionConfirmed)
			rosterMsg := helper.AssertMessageReceived(t, config.EventSetupToHostPlayerRoster)

			assert.NotNil(t, connectionMsg, "Reconnection #%d should receive CONNECTION_CONFIRMED", i)
			assert.NotNil(t, rosterMsg, "Reconnection #%d should receive PLAYER_ROSTER", i)

			t.Logf("Reconnection #%d: received %d messages", i, len(helper.GetCapturedMessages()))
		}
	})
}

// TestHostReconnectionPhase1ResourceGathering tests host reconnection during resource gathering phase
func TestHostReconnectionPhase1ResourceGathering(t *testing.T) {
	helper := NewHostReconnectionTestHelper(t)

	// Add players during setup phase first (required)
	helper.SetPhase(models.PhaseSetup)
	helper.AddTestPlayers(t, 2)

	// Then set up resource gathering phase
	helper.SetPhase(models.PhaseResourceGathering)
	game := helper.gameManager.GetGame()
	game.StartResourceGathering() // Start resource gathering phase (this resets CurrentRound to 0)
	game.CurrentRound = 2         // Set to round 2 to test mid-phase reconnection

	t.Run("Reconnection During Resource Phase", func(t *testing.T) {
		helper.CreateAndConnectHost(t)
		helper.DisconnectHost(t)

		isReconnection := helper.ReconnectHost(t)
		assert.True(t, isReconnection, "Should detect reconnection")

		// Simulate the reconnection flow
		sendHostConnectionConfirmed(helper.host, true)
		sendHostPhaseRestoration(helper.host) // Should send RESOURCE_TO_HOST_PHASE_START

		// Wait for messages
		helper.WaitForMessages(2, 300) // CONNECTION_CONFIRMED + PHASE_START

		// Verify messages
		helper.AssertMessageReceived(t, config.EventSetupToHostConnectionConfirmed)
		phaseStartMsg := helper.AssertMessageReceived(t, config.EventResourceToHostPhaseStart)

		// Verify phase start payload contains current round info
		if phaseStartMsg != nil {
			assert.Contains(t, phaseStartMsg.Payload, "currentRound")
			assert.Contains(t, phaseStartMsg.Payload, "totalRounds")
			assert.Equal(t, float64(2), phaseStartMsg.Payload["currentRound"])
		}

		t.Logf("Resource phase reconnection: received %d messages", len(helper.GetCapturedMessages()))
	})
}

// TestHostReconnectionPhase2PuzzleAssembly tests host reconnection during puzzle assembly phase
func TestHostReconnectionPhase2PuzzleAssembly(t *testing.T) {
	helper := NewHostReconnectionTestHelper(t)

	// Add players during setup phase first (required)
	helper.SetPhase(models.PhaseSetup)
	helper.AddTestPlayers(t, 2)

	// Then set up puzzle assembly phase
	helper.SetPhase(models.PhasePuzzleAssembly)

	// Set up puzzle grid
	game := helper.gameManager.GetGame()
	game.PuzzleGrid = models.NewPuzzleGrid(3) // 3x3 grid

	// Add some test fragments
	game.PuzzleGrid.AddFragment("segment1", "player1")
	game.PuzzleGrid.AddFragment("segment2", "player2")

	t.Run("Reconnection During Puzzle Phase", func(t *testing.T) {
		helper.CreateAndConnectHost(t)
		helper.DisconnectHost(t)

		isReconnection := helper.ReconnectHost(t)
		assert.True(t, isReconnection, "Should detect reconnection")

		// Simulate the reconnection flow
		sendHostConnectionConfirmed(helper.host, true)
		sendHostPhaseRestoration(helper.host) // Should send PUZZLE_TO_HOST_PHASE_LOAD and GRID_STATE

		// Wait for messages - could be up to 3 messages
		helper.WaitForMessages(3, 300) // CONNECTION_CONFIRMED + PHASE_LOAD + GRID_STATE

		// Verify messages
		helper.AssertMessageReceived(t, config.EventSetupToHostConnectionConfirmed)
		phaseLoadMsg := helper.AssertMessageReceived(t, config.EventPuzzleToHostPhaseLoad)
		gridStateMsg := helper.AssertMessageReceived(t, config.EventPuzzleToHostGridState)

		// Verify phase load payload
		if phaseLoadMsg != nil {
			assert.Contains(t, phaseLoadMsg.Payload, "gridSize")
			assert.Contains(t, phaseLoadMsg.Payload, "totalFragments")
		}

		// Verify grid state payload
		if gridStateMsg != nil {
			assert.Contains(t, gridStateMsg.Payload, "fragments")
			assert.Contains(t, gridStateMsg.Payload, "gridSize")
		}

		t.Logf("Puzzle phase reconnection: received %d messages", len(helper.GetCapturedMessages()))
	})
}

// TestHostReconnectionPhase3Analytics tests host reconnection during analytics phase
func TestHostReconnectionPhase3Analytics(t *testing.T) {
	helper := NewHostReconnectionTestHelper(t)

	// Add players during setup phase first (required)
	helper.SetPhase(models.PhaseSetup)
	helper.AddTestPlayers(t, 2)

	// Then set up analytics phase
	helper.SetPhase(models.PhaseAnalytics)

	// Set up analytics service with some mock data
	analyticsService := services.NewAnalyticsService()
	helper.gameManager.SetAnalyticsService(analyticsService)

	// Initialize analytics for the game
	analyticsService.StartGame("test-game-id")

	t.Run("Reconnection During Analytics Phase", func(t *testing.T) {
		helper.CreateAndConnectHost(t)
		helper.DisconnectHost(t)

		isReconnection := helper.ReconnectHost(t)
		assert.True(t, isReconnection, "Should detect reconnection")

		// Simulate the reconnection flow
		sendHostConnectionConfirmed(helper.host, true)
		sendHostPhaseRestoration(helper.host) // Should send ANALYTICS_TO_HOST_COMPLETE_REPORT

		// Wait for messages
		helper.WaitForMessages(2, 300) // CONNECTION_CONFIRMED + COMPLETE_REPORT

		// Verify messages
		helper.AssertMessageReceived(t, config.EventSetupToHostConnectionConfirmed)
		reportMsg := helper.AssertMessageReceived(t, config.EventAnalyticsToHostCompleteReport)

		// Verify analytics report payload
		if reportMsg != nil {
			assert.Contains(t, reportMsg.Payload, "gameAnalytics")
			assert.Contains(t, reportMsg.Payload, "teamPerformance")
		}

		t.Logf("Analytics phase reconnection: received %d messages", len(helper.GetCapturedMessages()))
	})
}

// TestRaceConditionDiagnosis specifically tests for race condition issues
func TestRaceConditionDiagnosis(t *testing.T) {
	helper := NewHostReconnectionTestHelper(t)
	helper.SetPhase(models.PhaseSetup)
	helper.AddTestPlayers(t, 2)

	t.Run("Message Timing Race Condition Test", func(t *testing.T) {
		// This test specifically checks if messages are lost due to race conditions
		// by reconnecting rapidly multiple times

		helper.CreateAndConnectHost(t)

		for i := 1; i <= 5; i++ {
			helper.DisconnectHost(t)

			isReconnection := helper.ReconnectHost(t)
			assert.True(t, isReconnection, "Should detect reconnection #%d", i)

			// Send messages immediately without delay (simulating race condition)
			sendHostConnectionConfirmed(helper.host, true)
			sendHostPhaseRestoration(helper.host)

			// Give minimal time for message processing
			helper.WaitForMessages(2, 50) // Very short timeout to stress test

			messages := helper.GetCapturedMessages()

			// Log detailed information about what we received
			t.Logf("Reconnection #%d: Expected 2 messages, got %d", i, len(messages))

			if len(messages) < 2 {
				t.Errorf("Race condition detected on reconnection #%d: only received %d/2 messages", i, len(messages))
				for j, msg := range messages {
					t.Logf("  Message %d: %s", j+1, msg.Event)
				}

				// This is the key diagnostic: if we consistently miss messages,
				// it's likely a race condition
				break
			}

			// Verify we got the expected messages
			helper.AssertMessageReceived(t, config.EventSetupToHostConnectionConfirmed)
			helper.AssertMessageReceived(t, config.EventSetupToHostPlayerRoster)
		}
	})
}
