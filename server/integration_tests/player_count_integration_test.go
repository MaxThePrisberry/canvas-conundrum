package integration_tests

import (
	"canvas-conundrum/config"
	"canvas-conundrum/handlers"
	"canvas-conundrum/services"
	"canvas-conundrum/test_helpers"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlayerCountingWithHost(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	// Connect host first
	host := test_helpers.NewTestHostClient(t, server, config.HostUUID)
	err := host.Connect()
	require.NoError(t, err)
	defer host.Close()

	// Verify host connection doesn't affect player count
	gm := services.GetGameInstance()
	assert.Equal(t, 0, gm.GetPlayerCount(), "Host connection should not affect player count")
	assert.True(t, gm.IsHostConnected(), "Host should be connected")

	// Connect first player
	player1 := test_helpers.NewTestPlayerClient(t, server)
	err = player1.Connect()
	require.NoError(t, err)
	defer player1.Close()

	// Clear any existing messages to avoid race condition
	player1.ClearMessages()

	// Configure the player
	err = player1.ConfigurePlayer("Player1", "art_enthusiast", []string{"general"})
	require.NoError(t, err)

	// Wait for lobby status update (after configuration)
	msg, err := player1.WaitForEvent(config.EventSetupToClientLobbyStatus, 2*time.Second)
	require.NoError(t, err)

	var payload map[string]interface{}
	payloadBytes, ok := msg.Payload.([]byte)
	if !ok {
		tempBytes, _ := json.Marshal(msg.Payload)
		payloadBytes = tempBytes
	}
	err = json.Unmarshal(payloadBytes, &payload)
	require.NoError(t, err)

	// Verify currentPlayers is 1 (only the actual player, not the host)
	assert.Equal(t, float64(1), payload["currentPlayers"], "currentPlayers should only count actual players")
	assert.Equal(t, float64(1), payload["readyPlayers"], "readyPlayers should be 1")

	// Add second player
	player2 := test_helpers.NewTestPlayerClient(t, server)
	err = player2.Connect()
	require.NoError(t, err)
	defer player2.Close()

	err = player2.ConfigurePlayer("Player2", "detective", []string{"history"})
	require.NoError(t, err)

	// Give time for all lobby updates to propagate
	time.Sleep(200 * time.Millisecond)

	// Check final player count directly from GameManager
	assert.Equal(t, 2, gm.GetPlayerCount(), "GameManager should report 2 players")

	// Also check if we can get the final lobby status from any recent messages
	// Since both players are ready, the last message should show current state
	messages := player1.GetMessages()
	var lastLobbyStatus map[string]interface{}
	found := false

	for _, msg := range messages {
		if msg.Event == config.EventSetupToClientLobbyStatus {
			payloadBytes, ok := msg.Payload.([]byte)
			if !ok {
				tempBytes, _ := json.Marshal(msg.Payload)
				payloadBytes = tempBytes
			}
			var payload map[string]interface{}
			if json.Unmarshal(payloadBytes, &payload) == nil {
				lastLobbyStatus = payload
				found = true
			}
		}
	}

	if found {
		// Verify the last lobby status shows correct counts
		if currentPlayers, ok := lastLobbyStatus["currentPlayers"].(float64); ok {
			assert.Equal(t, float64(2), currentPlayers, "currentPlayers should be 2 (only actual players)")
		}
		if readyPlayers, ok := lastLobbyStatus["readyPlayers"].(float64); ok {
			assert.Equal(t, float64(2), readyPlayers, "readyPlayers should be 2")
		}
	}

	// Verify GameManager reports correct count
	assert.Equal(t, 2, gm.GetPlayerCount(), "GameManager should report 2 players")
}

func TestHostDisconnectionDoesNotAffectPlayerCount(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	gm := services.GetGameInstance()

	// Connect player first
	player := test_helpers.NewTestPlayerClient(t, server)
	err := player.Connect()
	require.NoError(t, err)
	defer player.Close()

	err = player.ConfigurePlayer("Player1", "tourist", []string{"science"})
	require.NoError(t, err)

	// Player count should be 1
	assert.Equal(t, 1, gm.GetPlayerCount(), "Player count should be 1")

	// Connect host
	host := test_helpers.NewTestHostClient(t, server, config.HostUUID)
	err = host.Connect()
	require.NoError(t, err)

	// Player count should still be 1
	assert.Equal(t, 1, gm.GetPlayerCount(), "Player count should still be 1 after host connects")

	// Disconnect host
	host.Close()
	time.Sleep(100 * time.Millisecond) // Allow disconnection to process

	// Player count should still be 1
	assert.Equal(t, 1, gm.GetPlayerCount(), "Player count should still be 1 after host disconnects")
	assert.False(t, gm.IsHostConnected(), "Host should be disconnected")
}

func TestPingResponseActiveConnections(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	// Connect a player
	player := test_helpers.NewTestPlayerClient(t, server)
	err := player.Connect()
	require.NoError(t, err)
	defer player.Close()

	// Wait for initial connection message
	time.Sleep(200 * time.Millisecond)

	// Send ping
	pingPayload := map[string]interface{}{
		"clientTimestamp": time.Now().Format(time.RFC3339),
		"sequenceNumber":  1,
		"connectionQuality": map[string]interface{}{
			"latency":          50,
			"messagesReceived": 1,
			"messagesSent":     1,
		},
	}

	err = player.SendMessage(config.EventSystemPing, pingPayload)
	require.NoError(t, err)

	// Wait for pong response
	msg, err := player.WaitForEvent(config.EventSystemPong, 3*time.Second)
	require.NoError(t, err)

	var pongPayload map[string]interface{}
	payloadBytes, ok := msg.Payload.([]byte)
	if !ok {
		tempBytes, _ := json.Marshal(msg.Payload)
		payloadBytes = tempBytes
	}
	err = json.Unmarshal(payloadBytes, &pongPayload)
	require.NoError(t, err)

	// Verify serverHealth.activeConnections only counts players, not host
	serverHealth, ok := pongPayload["serverHealth"].(map[string]interface{})
	require.True(t, ok, "serverHealth should be present")

	activeConnections, ok := serverHealth["activeConnections"].(float64)
	require.True(t, ok, "activeConnections should be present")

	assert.Equal(t, float64(1), activeConnections, "activeConnections should only count players, not host")
}

func TestHealthEndpointPlayerCount(t *testing.T) {
	_, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a test server with health endpoint
	r := mux.NewRouter()
	r.HandleFunc("/ws", handlers.HandlePlayerWebSocket).Methods("GET")
	r.HandleFunc("/ws/host/{uuid}", handlers.HandleHostWebSocket).Methods("GET")
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		gameManager := services.GetGameInstance()
		health := map[string]interface{}{
			"status":        "healthy",
			"timestamp":     time.Now().Unix(),
			"gamePhase":     string(gameManager.GetCurrentPhase()),
			"playerCount":   gameManager.GetPlayerCount(),
			"hostConnected": gameManager.IsHostConnected(),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(health)
	}).Methods("GET")

	healthServer := httptest.NewServer(r)
	defer healthServer.Close()

	// Connect host
	wsURL := strings.Replace(healthServer.URL, "http://", "ws://", 1) + "/ws/host/" + config.HostUUID
	dialer := websocket.Dialer{}
	hostConn, _, err := dialer.Dial(wsURL, nil)
	if err == nil {
		defer hostConn.Close()
	}

	// Connect player
	wsURL = strings.Replace(healthServer.URL, "http://", "ws://", 1) + "/ws"
	playerConn, _, err := dialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer playerConn.Close()

	// Wait for connections to be processed
	time.Sleep(200 * time.Millisecond)

	// Check health endpoint
	resp, err := http.Get(healthServer.URL + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()

	var health map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&health)
	require.NoError(t, err)

	// Verify playerCount is 1 (only the player, not the host)
	assert.Equal(t, float64(1), health["playerCount"], "Health endpoint should report only players, not host")
}
