package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

// MockWebSocketConn implements a mock WebSocket connection for testing
type MockWebSocketConn struct {
	WriteMessages [][]byte
	ReadMessages  [][]byte
	ReadIndex     int
	WriteClosed   bool
	Closed        bool
	CloseError    error
	WriteError    error
	ReadError     error
	PingHandler   func(string) error
	PongHandler   func(string) error
	CloseHandler  func(int, string) error
	LocalAddr     string
	RemoteAddr    string
	WriteDeadline time.Time
	ReadDeadline  time.Time
}

func NewMockWebSocketConn() *MockWebSocketConn {
	return &MockWebSocketConn{
		WriteMessages: make([][]byte, 0),
		ReadMessages:  make([][]byte, 0),
		ReadIndex:     0,
	}
}

func (m *MockWebSocketConn) Close() error {
	m.Closed = true
	return m.CloseError
}

func (m *MockWebSocketConn) WriteMessage(messageType int, data []byte) error {
	if m.WriteError != nil {
		return m.WriteError
	}
	m.WriteMessages = append(m.WriteMessages, data)
	return nil
}

func (m *MockWebSocketConn) ReadMessage() (messageType int, p []byte, err error) {
	if m.ReadError != nil {
		return 0, nil, m.ReadError
	}
	if m.ReadIndex >= len(m.ReadMessages) {
		return 0, nil, &websocket.CloseError{Code: websocket.CloseNormalClosure}
	}
	msg := m.ReadMessages[m.ReadIndex]
	m.ReadIndex++
	return websocket.TextMessage, msg, nil
}

func (m *MockWebSocketConn) WriteControl(messageType int, data []byte, deadline time.Time) error {
	return nil
}

func (m *MockWebSocketConn) SetReadDeadline(t time.Time) error {
	m.ReadDeadline = t
	return nil
}

func (m *MockWebSocketConn) SetWriteDeadline(t time.Time) error {
	m.WriteDeadline = t
	return nil
}

func (m *MockWebSocketConn) SetPingHandler(h func(appData string) error) {
	m.PingHandler = h
}

func (m *MockWebSocketConn) SetPongHandler(h func(appData string) error) {
	m.PongHandler = h
}

func (m *MockWebSocketConn) SetCloseHandler(h func(code int, text string) error) {
	m.CloseHandler = h
}

// Test helper functions

// CreateTestGame creates a game with test managers
func CreateTestGame(t *testing.T, playerCount int) (*GameManager, *PlayerManager, *TriviaManager) {
	playerMgr := NewPlayerManager()
	triviaMgr := NewTriviaManager()
	broadcastChan := make(chan BroadcastMessage, 256)
	gameMgr := NewGameManager(playerMgr, triviaMgr, broadcastChan)

	// Create host
	hostConn := &websocket.Conn{} // We'll use mock connection methods later
	host := playerMgr.CreatePlayer(hostConn, true)
	assert.NotNil(t, host)

	// Create players
	for i := 0; i < playerCount; i++ {
		conn := &websocket.Conn{} // We'll use mock connection methods later
		player := playerMgr.CreatePlayer(conn, false)
		assert.NotNil(t, player)
	}

	return gameMgr, playerMgr, triviaMgr
}

// CreateMockPlayer creates a mock player with a valid UUID
func CreateMockPlayer(isHost bool) *Player {
	return &Player{
		ID:              uuid.New().String(),
		Name:            "Test Player",
		Role:            "",
		Specialties:     []string{},
		State:           StateConnected,
		Connection:      &websocket.Conn{},
		CurrentLocation: "",
		IsHost:          isHost,
		Ready:           false,
		LastSeen:        time.Now(),
	}
}

// GenerateTestPlayers creates multiple test players
func GenerateTestPlayers(count int) []*Player {
	players := make([]*Player, count)
	for i := 0; i < count; i++ {
		players[i] = CreateMockPlayer(false)
	}
	return players
}

// CreateAuthWrapper creates a properly formatted auth wrapper for testing
func CreateAuthWrapper(playerID string, payload interface{}) []byte {
	payloadBytes, _ := json.Marshal(payload)
	wrapper := AuthWrapper{
		Auth: AuthData{
			PlayerID: playerID,
		},
		Payload: json.RawMessage(payloadBytes),
	}
	data, _ := json.Marshal(wrapper)
	return data
}

// CreateWebSocketTestServer creates a test HTTP server with WebSocket upgrade
func CreateWebSocketTestServer(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}

// UpgradeWebSocket upgrades an HTTP connection to WebSocket for testing
func UpgradeWebSocket(serverURL string) (*websocket.Conn, *http.Response, error) {
	wsURL := "ws" + strings.TrimPrefix(serverURL, "http")
	return websocket.DefaultDialer.Dial(wsURL, nil)
}

// SimulateGamePhase moves the game to a specific phase for testing
func SimulateGamePhase(g *GameManager, phase GamePhase) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.state.Phase = phase
}

// AssertJSONEqual asserts that two JSON strings are equal
func AssertJSONEqual(t *testing.T, expected, actual string) {
	var expectedObj, actualObj interface{}
	assert.NoError(t, json.Unmarshal([]byte(expected), &expectedObj))
	assert.NoError(t, json.Unmarshal([]byte(actual), &actualObj))
	assert.Equal(t, expectedObj, actualObj)
}

// WaitForCondition waits for a condition to be true with timeout
func WaitForCondition(t *testing.T, condition func() bool, timeout time.Duration, message string) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("Condition not met within timeout: %s", message)
}
