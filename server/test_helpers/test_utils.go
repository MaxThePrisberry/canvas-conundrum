package test_helpers

import (
	"canvas-conundrum/models"
	"canvas-conundrum/utils"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

// CreateTestPlayer creates a test player with mock connection
func CreateTestPlayer(id string) *models.Player {
	if id == "" {
		id = uuid.New().String()
	}
	player := models.NewPlayer(id, nil)
	return player
}

// CreateTestHost creates a test host with mock connection
func CreateTestHost(id string) *models.Host {
	if id == "" {
		id = uuid.New().String()
	}
	host := models.NewHost(id, nil)
	return host
}

// CreateTestGame creates a test game with default settings
func CreateTestGame() *models.Game {
	game := models.NewGame()
	return game
}

// CreateTestMessage creates a test WebSocket message
func CreateTestMessage(event string, payload interface{}) *utils.Message {
	payloadBytes, _ := json.Marshal(payload)
	return &utils.Message{
		Event:     event,
		Payload:   payloadBytes,
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

// CreateTestTriviaQuestion creates a test trivia question
func CreateTestTriviaQuestion(category models.TriviaCategory, difficulty models.TriviaDifficulty) *models.TriviaQuestion {
	return &models.TriviaQuestion{
		ID:            "test-question-" + uuid.New().String(),
		Category:      category,
		Difficulty:    difficulty,
		Question:      "Test question?",
		Options:       []string{"A", "B", "C", "D"},
		CorrectAnswer: "A",
		CorrectIndex:  0,
	}
}

// AssertJSONEqual asserts that two JSON objects are equal
func AssertJSONEqual(t *testing.T, expected, actual interface{}) {
	expectedJSON, err := json.Marshal(expected)
	assert.NoError(t, err)

	actualJSON, err := json.Marshal(actual)
	assert.NoError(t, err)

	assert.JSONEq(t, string(expectedJSON), string(actualJSON))
}

// ParseMessage parses a WebSocket message for testing
func ParseMessage(t *testing.T, data []byte) *utils.ServerMessage {
	var msg utils.ServerMessage
	err := json.Unmarshal(data, &msg)
	assert.NoError(t, err)
	return &msg
}

// CreateMockWebSocketConn creates a mock WebSocket connection for testing
type MockWebSocketConn struct {
	WriteMessages        [][]byte
	ReadMessages         [][]byte
	ReadIndex            int
	Closed               bool
	ReadDeadlineSet      bool
	WriteDeadlineSet     bool
	PongHandlerSet       bool
	MessageWritten       bool
	PingWritten          bool
	NextReadMessageType  int
	NextReadMessageData  []byte
	NextReadMessageError error
}

func NewMockWebSocketConn() *MockWebSocketConn {
	return &MockWebSocketConn{
		WriteMessages:       make([][]byte, 0),
		ReadMessages:        make([][]byte, 0),
		ReadIndex:           0,
		Closed:              false,
		NextReadMessageType: websocket.TextMessage,
	}
}

func (m *MockWebSocketConn) SetNextReadMessage(messageType int, data []byte, err error) {
	m.NextReadMessageType = messageType
	m.NextReadMessageData = data
	m.NextReadMessageError = err
}

func (m *MockWebSocketConn) ReadMessage() (messageType int, p []byte, err error) {
	if m.NextReadMessageError != nil {
		return 0, nil, m.NextReadMessageError
	}
	if m.NextReadMessageData != nil {
		data := m.NextReadMessageData
		m.NextReadMessageData = nil // Only return once
		return m.NextReadMessageType, data, nil
	}
	if m.ReadIndex >= len(m.ReadMessages) {
		return 0, nil, &websocket.CloseError{Code: websocket.CloseNormalClosure}
	}
	msg := m.ReadMessages[m.ReadIndex]
	m.ReadIndex++
	return websocket.TextMessage, msg, nil
}

func (m *MockWebSocketConn) WriteMessage(messageType int, data []byte) error {
	m.WriteMessages = append(m.WriteMessages, data)
	m.MessageWritten = true
	if messageType == websocket.PingMessage {
		m.PingWritten = true
	}
	return nil
}

func (m *MockWebSocketConn) Close() error {
	m.Closed = true
	return nil
}

func (m *MockWebSocketConn) SetReadDeadline(t time.Time) error {
	m.ReadDeadlineSet = true
	return nil
}

func (m *MockWebSocketConn) SetWriteDeadline(t time.Time) error {
	m.WriteDeadlineSet = true
	return nil
}

func (m *MockWebSocketConn) SetPongHandler(h func(string) error) {
	m.PongHandlerSet = true
}

// Helper methods for assertions
func (m *MockWebSocketConn) WasReadDeadlineSet() bool  { return m.ReadDeadlineSet }
func (m *MockWebSocketConn) WasWriteDeadlineSet() bool { return m.WriteDeadlineSet }
func (m *MockWebSocketConn) WasPongHandlerSet() bool   { return m.PongHandlerSet }
func (m *MockWebSocketConn) WasMessageWritten() bool   { return m.MessageWritten }
func (m *MockWebSocketConn) WasPingWritten() bool      { return m.PingWritten }

// GameStateAssertion provides fluent assertions for game state
type GameStateAssertion struct {
	t    *testing.T
	game *models.Game
}

func AssertGameState(t *testing.T, game *models.Game) *GameStateAssertion {
	return &GameStateAssertion{t: t, game: game}
}

func (a *GameStateAssertion) HasPhase(phase models.GamePhase) *GameStateAssertion {
	assert.Equal(a.t, phase, a.game.CurrentPhase, "Expected game phase %v but got %v", phase, a.game.CurrentPhase)
	return a
}

func (a *GameStateAssertion) HasPlayerCount(count int) *GameStateAssertion {
	assert.Equal(a.t, count, a.game.PlayerCount, "Expected %d players but got %d", count, a.game.PlayerCount)
	return a
}

func (a *GameStateAssertion) IsStarted() *GameStateAssertion {
	assert.True(a.t, a.game.GameStarted, "Expected game to be started")
	return a
}

func (a *GameStateAssertion) IsNotStarted() *GameStateAssertion {
	assert.False(a.t, a.game.GameStarted, "Expected game not to be started")
	return a
}

// TokenAssertion provides fluent assertions for tokens
type TokenAssertion struct {
	t      *testing.T
	tokens *models.TeamTokens
}

func AssertTokens(t *testing.T, tokens *models.TeamTokens) *TokenAssertion {
	return &TokenAssertion{t: t, tokens: tokens}
}

func (a *TokenAssertion) HasAnchor(count int) *TokenAssertion {
	assert.Equal(a.t, count, a.tokens.AnchorTokens, "Expected %d anchor tokens but got %d", count, a.tokens.AnchorTokens)
	return a
}

func (a *TokenAssertion) HasChronos(count int) *TokenAssertion {
	assert.Equal(a.t, count, a.tokens.ChronosTokens, "Expected %d chronos tokens but got %d", count, a.tokens.ChronosTokens)
	return a
}

func (a *TokenAssertion) HasGuide(count int) *TokenAssertion {
	assert.Equal(a.t, count, a.tokens.GuideTokens, "Expected %d guide tokens but got %d", count, a.tokens.GuideTokens)
	return a
}

func (a *TokenAssertion) HasClarity(count int) *TokenAssertion {
	assert.Equal(a.t, count, a.tokens.ClarityTokens, "Expected %d clarity tokens but got %d", count, a.tokens.ClarityTokens)
	return a
}

func (a *TokenAssertion) HasThreshold(tokenType models.TokenType, level int) *TokenAssertion {
	actual := a.tokens.GetThreshold(tokenType)
	assert.Equal(a.t, level, actual, "Expected threshold %d for %s but got %d", level, tokenType, actual)
	return a
}
