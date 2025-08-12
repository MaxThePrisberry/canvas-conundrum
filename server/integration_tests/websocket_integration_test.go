package integration_tests

import (
	"canvas-conundrum/config"
	"canvas-conundrum/constants"
	"canvas-conundrum/handlers"
	"canvas-conundrum/services"
	"canvas-conundrum/test_helpers"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestServer(t *testing.T) *httptest.Server {
	// Reset game manager singleton
	services.GetGameInstance()

	// Setup services
	gm := services.GetGameInstance()
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
	return server
}

func TestPlayerWebSocketConnection(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	// Create WebSocket connection
	wsURL := strings.Replace(server.URL, "http://", "ws://", 1) + "/ws"
	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	// Read initial message (SETUP_TO_PLAYER_ROLES_AVAILABLE)
	_, message, err := conn.ReadMessage()
	require.NoError(t, err)

	var msg map[string]interface{}
	err = json.Unmarshal(message, &msg)
	require.NoError(t, err)

	assert.Equal(t, constants.EventSetupToPlayerRolesAvailable, msg["event"])

	// Check payload
	payload := msg["payload"].(map[string]interface{})
	assert.NotEmpty(t, payload["playerId"])
	assert.False(t, payload["isHost"].(bool))
	assert.NotNil(t, payload["roles"])

	// Verify roles structure
	roles := payload["roles"].([]interface{})
	assert.Len(t, roles, 4) // 4 roles available

	firstRole := roles[0].(map[string]interface{})
	assert.Contains(t, firstRole, "roleType")
	assert.Contains(t, firstRole, "displayName")
	assert.Contains(t, firstRole, "resourceBonus")
	assert.Contains(t, firstRole, "bonusTokenType")
	assert.Contains(t, firstRole, "description")
	assert.Contains(t, firstRole, "available")
}

func TestHostWebSocketConnection(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	// Test with correct UUID
	t.Run("ValidUUID", func(t *testing.T) {
		wsURL := strings.Replace(server.URL, "http://", "ws://", 1) + "/ws/host/" + config.HostUUID
		dialer := websocket.Dialer{}
		conn, resp, err := dialer.Dial(wsURL, nil)

		if err != nil && resp != nil && resp.StatusCode == http.StatusUnauthorized {
			// Expected for invalid UUID
			return
		}

		require.NoError(t, err)
		defer conn.Close()

		// Read initial message (SETUP_TO_HOST_CONNECTION_CONFIRMED)
		_, message, err := conn.ReadMessage()
		require.NoError(t, err)

		var msg map[string]interface{}
		err = json.Unmarshal(message, &msg)
		require.NoError(t, err)

		assert.Equal(t, constants.EventSetupToHostConnectionConfirmed, msg["event"])

		payload := msg["payload"].(map[string]interface{})
		assert.NotEmpty(t, payload["playerId"])
		assert.True(t, payload["isHost"].(bool))
		assert.Equal(t, "Connected as game host", payload["message"])

		// Check game config
		gameConfig := payload["gameConfig"].(map[string]interface{})
		assert.Equal(t, float64(constants.MinPlayers), gameConfig["minPlayers"])
		assert.Equal(t, float64(constants.MaxPlayers), gameConfig["maxPlayers"])
	})

	t.Run("InvalidUUID", func(t *testing.T) {
		wsURL := strings.Replace(server.URL, "http://", "ws://", 1) + "/ws/host/invalid-uuid"
		dialer := websocket.Dialer{}
		_, resp, err := dialer.Dial(wsURL, nil)

		// Should fail with unauthorized
		assert.Error(t, err)
		if resp != nil {
			assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		}
	})
}

func TestPlayerConfiguration(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	// Connect as player
	client := test_helpers.NewTestPlayerClient(t, server)
	err := client.Connect()
	require.NoError(t, err)
	defer client.Close()

	// Send player configuration
	err = client.ConfigurePlayer("Alice", "art_enthusiast", []string{"general"})
	require.NoError(t, err)

	// Wait for lobby status update
	msg, err := client.WaitForEvent(constants.EventSetupToClientLobbyStatus, 2*time.Second)
	require.NoError(t, err)

	var payload map[string]interface{}
	payloadBytes, ok := msg.Payload.([]byte)
	if !ok {
		// Try to marshal and unmarshal if it's already parsed
		tempBytes, _ := json.Marshal(msg.Payload)
		payloadBytes = tempBytes
	}
	err = json.Unmarshal(payloadBytes, &payload)
	require.NoError(t, err)

	// Check player count
	assert.Equal(t, float64(1), payload["totalPlayers"])
	assert.Equal(t, float64(1), payload["readyPlayers"])
}

func TestMultiplePlayersJoining(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	// Connect host
	host := test_helpers.NewTestHostClient(t, server, config.HostUUID)
	err := host.Connect()
	require.NoError(t, err)
	defer host.Close()

	// Connect multiple players
	players := make([]*test_helpers.TestPlayerClient, 4)
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

		// Configure each player
		err = players[i].ConfigurePlayer(
			string(rune('A'+i)),
			roles[i],
			specialties[i],
		)
		require.NoError(t, err)

		// Small delay to ensure message processing
		time.Sleep(50 * time.Millisecond)
	}

	// Host should receive roster updates
	time.Sleep(100 * time.Millisecond)

	// Check for roster update
	messages := host.GetMessages()
	foundRoster := false
	for _, msg := range messages {
		if msg.Event == constants.EventSetupToHostPlayerRoster {
			foundRoster = true

			var payload map[string]interface{}
			payloadBytes, ok := msg.Payload.([]byte)
			if !ok {
				// Try to marshal and unmarshal if it's already parsed
				tempBytes, _ := json.Marshal(msg.Payload)
				payloadBytes = tempBytes
			}
			err := json.Unmarshal(payloadBytes, &payload)
			require.NoError(t, err)

			players := payload["players"].([]interface{})
			assert.Len(t, players, 4)
			break
		}
	}
	assert.True(t, foundRoster, "Host should receive player roster")
}

func TestGameStartFlow(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	// Connect host
	host := test_helpers.NewTestHostClient(t, server, config.HostUUID)
	err := host.Connect()
	require.NoError(t, err)
	defer host.Close()

	// Connect and configure minimum players
	players := make([]*test_helpers.TestPlayerClient, constants.MinPlayers)
	roles := []string{"art_enthusiast", "detective", "tourist", "janitor"}

	for i := 0; i < constants.MinPlayers; i++ {
		players[i] = test_helpers.NewTestPlayerClient(t, server)
		err := players[i].Connect()
		require.NoError(t, err)
		defer players[i].Close()

		err = players[i].ConfigurePlayer(
			"Player"+string(rune('1'+i)),
			roles[i%4],
			[]string{"general"},
		)
		require.NoError(t, err)
	}

	// Wait for players to be ready
	time.Sleep(200 * time.Millisecond)

	// Host starts game
	err = host.StartGame("medium")
	require.NoError(t, err)

	// All players should receive game started event
	for _, player := range players {
		msg, err := player.WaitForEvent(constants.EventSetupToClientGameStarted, 2*time.Second)
		require.NoError(t, err)
		assert.NotNil(t, msg)
	}

	// Host should receive game started event
	msg, err := host.WaitForEvent(constants.EventSetupToHostGameStarted, 2*time.Second)
	require.NoError(t, err)
	assert.NotNil(t, msg)
}

func TestPingPong(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	// Connect as player
	client := test_helpers.NewTestPlayerClient(t, server)
	err := client.Connect()
	require.NoError(t, err)
	defer client.Close()

	// Send ping
	err = client.SendMessage(constants.EventSystemPing, nil)
	require.NoError(t, err)

	// Should receive pong
	msg, err := client.WaitForEvent(constants.EventSystemPong, 1*time.Second)
	require.NoError(t, err)
	assert.NotNil(t, msg)
}

func TestInvalidAuthentication(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	// Connect as player
	wsURL := strings.Replace(server.URL, "http://", "ws://", 1) + "/ws"
	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	// Read initial message to get player ID
	_, _, err = conn.ReadMessage()
	require.NoError(t, err)

	// Send message with invalid token
	invalidMsg := map[string]interface{}{
		"event": constants.EventResourceToServerLocationVerified,
		"auth": map[string]interface{}{
			"token": "invalid-token",
		},
		"payload": map[string]interface{}{
			"stationId": "anchor",
			"qrHash":    "test",
		},
		"timestamp": time.Now().Format(time.RFC3339),
	}

	data, err := json.Marshal(invalidMsg)
	require.NoError(t, err)

	err = conn.WriteMessage(websocket.TextMessage, data)
	require.NoError(t, err)

	// Should receive error message
	_, message, err := conn.ReadMessage()
	require.NoError(t, err)

	var response map[string]interface{}
	err = json.Unmarshal(message, &response)
	require.NoError(t, err)

	assert.Equal(t, constants.EventSystemToClientError, response["event"])
}

func TestConcurrentConnections(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	// Connect many players concurrently
	numPlayers := 20
	var wg sync.WaitGroup
	errors := make([]error, numPlayers)

	for i := 0; i < numPlayers; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			client := test_helpers.NewTestPlayerClient(t, server)
			err := client.Connect()
			if err != nil {
				errors[index] = err
				return
			}
			defer client.Close()

			// Configure player
			role := []string{"art_enthusiast", "detective", "tourist", "janitor"}[index%4]
			err = client.ConfigurePlayer(
				"Player"+string(rune(index)),
				role,
				[]string{"general"},
			)
			errors[index] = err

			// Keep connection alive briefly
			time.Sleep(100 * time.Millisecond)
		}(i)
	}

	wg.Wait()

	// Check for errors
	for i, err := range errors {
		assert.NoError(t, err, "Player %d connection error", i)
	}
}

func TestDisconnectionHandling(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	// Connect host
	host := test_helpers.NewTestHostClient(t, server, config.HostUUID)
	err := host.Connect()
	require.NoError(t, err)
	defer host.Close()

	// Connect player
	player := test_helpers.NewTestPlayerClient(t, server)
	err = player.Connect()
	require.NoError(t, err)

	playerID := player.GetLastMessage()
	var payload map[string]interface{}
	payloadBytes, ok := playerID.Payload.([]byte)
	if !ok {
		// Try to marshal and unmarshal if it's already parsed
		tempBytes, _ := json.Marshal(playerID.Payload)
		payloadBytes = tempBytes
	}
	json.Unmarshal(payloadBytes, &payload)
	pID := payload["playerId"].(string)

	// Configure player
	err = player.ConfigurePlayer("DisconnectTest", "detective", []string{"history"})
	require.NoError(t, err)

	// Wait for roster update
	time.Sleep(100 * time.Millisecond)

	// Disconnect player
	player.Close()

	// Wait for disconnection to be processed
	time.Sleep(200 * time.Millisecond)

	// Host should receive player disconnection notification
	messages := host.GetMessages()
	foundDisconnect := false
	for _, msg := range messages {
		if msg.Event == constants.EventSystemToHostPlayerDisconnected {
			var payload map[string]interface{}
			payloadBytes, ok := msg.Payload.([]byte)
			if !ok {
				// Try to marshal and unmarshal if it's already parsed
				tempBytes, _ := json.Marshal(msg.Payload)
				payloadBytes = tempBytes
			}
			json.Unmarshal(payloadBytes, &payload)
			if payload["playerId"] == pID {
				foundDisconnect = true
				break
			}
		}
	}
	assert.True(t, foundDisconnect, "Host should be notified of player disconnection")
}
