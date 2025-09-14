package test_helpers

import (
	"canvas-conundrum/config"
	"canvas-conundrum/utils"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWebSocketClient represents a test WebSocket client
type TestWebSocketClient struct {
	t          *testing.T
	conn       *websocket.Conn
	server     *httptest.Server
	url        string
	playerID   string
	token      string
	messages   []utils.ServerMessage
	mu         sync.Mutex
	done       chan struct{}
	readErrors []error
}

// NewTestWebSocketClient creates a new test WebSocket client
func NewTestWebSocketClient(t *testing.T, server *httptest.Server, path string) *TestWebSocketClient {
	wsURL := strings.Replace(server.URL, "http://", "ws://", 1) + path

	client := &TestWebSocketClient{
		t:        t,
		server:   server,
		url:      wsURL,
		messages: make([]utils.ServerMessage, 0),
		done:     make(chan struct{}),
	}

	return client
}

// Connect establishes WebSocket connection
func (c *TestWebSocketClient) Connect() error {
	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(c.url, nil)
	if err != nil {
		return err
	}

	c.conn = conn

	// Start reading messages
	go c.readMessages()

	// Wait for initial connection message
	time.Sleep(100 * time.Millisecond)

	// Extract player ID and token from first message
	c.mu.Lock()
	if len(c.messages) > 0 {
		if payload, ok := c.messages[0].Payload.(map[string]interface{}); ok {
			if id, ok := payload["playerId"].(string); ok {
				c.playerID = id
				c.token = id // Token is same as player ID initially
			}
		}
	}
	c.mu.Unlock()

	return nil
}

// readMessages continuously reads messages from the WebSocket
func (c *TestWebSocketClient) readMessages() {
	defer close(c.done)

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.mu.Lock()
				c.readErrors = append(c.readErrors, err)
				c.mu.Unlock()
			}
			return
		}

		var msg utils.ServerMessage
		if err := json.Unmarshal(message, &msg); err == nil {
			c.mu.Lock()
			c.messages = append(c.messages, msg)
			c.mu.Unlock()
		}
	}
}

// SendMessage sends a message to the server
func (c *TestWebSocketClient) SendMessage(event string, payload interface{}) error {
	msg := utils.Message{
		Event: event,
		Auth: &utils.Auth{
			Token: c.token,
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	if payload != nil {
		payloadJSON, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		msg.Payload = payloadJSON
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return c.conn.WriteMessage(websocket.TextMessage, data)
}

// WaitForEvent waits for a specific event type
func (c *TestWebSocketClient) WaitForEvent(eventType string, timeout time.Duration) (*utils.ServerMessage, error) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		c.mu.Lock()
		for _, msg := range c.messages {
			if msg.Event == eventType {
				c.mu.Unlock()
				return &msg, nil
			}
		}
		c.mu.Unlock()
		time.Sleep(10 * time.Millisecond)
	}

	return nil, &TimeoutError{Event: eventType}
}

// GetMessages returns all received messages
func (c *TestWebSocketClient) GetMessages() []utils.ServerMessage {
	c.mu.Lock()
	defer c.mu.Unlock()

	result := make([]utils.ServerMessage, len(c.messages))
	copy(result, c.messages)
	return result
}

// GetLastMessage returns the most recent message
func (c *TestWebSocketClient) GetLastMessage() *utils.ServerMessage {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.messages) == 0 {
		return nil
	}
	return &c.messages[len(c.messages)-1]
}

// GetPlayerID returns the player ID
func (c *TestWebSocketClient) GetPlayerID() string {
	return c.playerID
}

// GetToken returns the authentication token
func (c *TestWebSocketClient) GetToken() string {
	return c.token
}

// ClearMessages clears the message buffer
func (c *TestWebSocketClient) ClearMessages() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.messages = make([]utils.ServerMessage, 0)
}

// Close closes the WebSocket connection
func (c *TestWebSocketClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// AssertEventReceived asserts that an event was received
func (c *TestWebSocketClient) AssertEventReceived(t *testing.T, eventType string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, msg := range c.messages {
		if msg.Event == eventType {
			return
		}
	}

	assert.Fail(t, "Event not received", "Expected to receive event: %s", eventType)
}

// AssertNoErrors asserts that no read errors occurred
func (c *TestWebSocketClient) AssertNoErrors(t *testing.T) {
	c.mu.Lock()
	defer c.mu.Unlock()

	assert.Empty(t, c.readErrors, "Unexpected WebSocket read errors")
}

// TimeoutError represents a timeout waiting for an event
type TimeoutError struct {
	Event string
}

func (e *TimeoutError) Error() string {
	return "timeout waiting for event: " + e.Event
}

// TestHostClient represents a test host WebSocket client
type TestHostClient struct {
	*TestWebSocketClient
	hostUUID string
}

// NewTestHostClient creates a new test host client
func NewTestHostClient(t *testing.T, server *httptest.Server, hostUUID string) *TestHostClient {
	path := "/ws/host/" + hostUUID
	baseClient := NewTestWebSocketClient(t, server, path)

	return &TestHostClient{
		TestWebSocketClient: baseClient,
		hostUUID:            hostUUID,
	}
}

// StartGame sends the start game command
func (c *TestHostClient) StartGame(difficulty string) error {
	payload := map[string]interface{}{
		"difficulty": difficulty,
	}
	return c.SendMessage(config.EventSetupToServerStartGame, payload)
}

// StartPuzzlePhase sends the start puzzle phase command
func (c *TestHostClient) StartPuzzlePhase() error {
	return c.SendMessage(config.EventPuzzleToServerPhaseStart, nil)
}

// ResetGame sends the reset game command
func (c *TestHostClient) ResetGame() error {
	return c.SendMessage(config.EventAnalyticsToServerResetGame, nil)
}

// TestPlayerClient represents a test player WebSocket client
type TestPlayerClient struct {
	*TestWebSocketClient
	role        string
	specialties []string
}

// NewTestPlayerClient creates a new test player client
func NewTestPlayerClient(t *testing.T, server *httptest.Server) *TestPlayerClient {
	baseClient := NewTestWebSocketClient(t, server, "/ws")

	return &TestPlayerClient{
		TestWebSocketClient: baseClient,
	}
}

// ConfigurePlayer sends player configuration
func (c *TestPlayerClient) ConfigurePlayer(name, role string, specialties []string) error {
	c.role = role
	c.specialties = specialties

	payload := map[string]interface{}{
		"playerName":   name,
		"selectedRole": role,
		"specialties":  specialties,
	}
	return c.SendMessage(config.EventSetupToServerPlayerConfiguration, payload)
}

// VerifyLocation sends location verification (QR scan)
func (c *TestPlayerClient) VerifyLocation(stationID, qrHash string) error {
	payload := map[string]interface{}{
		"stationId": stationID,
		"qrHash":    qrHash,
	}
	return c.SendMessage(config.EventResourceToServerLocationVerified, payload)
}

// AnswerTrivia sends a trivia answer
func (c *TestPlayerClient) AnswerTrivia(questionID string, answerIndex int, timeElapsed float64) error {
	payload := map[string]interface{}{
		"questionId":  questionID,
		"answerIndex": answerIndex,
		"timeElapsed": timeElapsed,
	}
	return c.SendMessage(config.EventResourceToServerTriviaAnswer, payload)
}

// CompleteSegment sends segment completion
func (c *TestPlayerClient) CompleteSegment(segmentID string, solveTime float64) error {
	payload := map[string]interface{}{
		"segmentId": segmentID,
		"solveTime": solveTime,
	}
	return c.SendMessage(config.EventPuzzleToServerSegmentCompleted, payload)
}

// MoveFragment sends a fragment move
func (c *TestPlayerClient) MoveFragment(fragmentID, fromPosition, toPosition string) error {
	payload := map[string]interface{}{
		"fragmentId":   fragmentID,
		"fromPosition": fromPosition,
		"toPosition":   toPosition,
	}
	return c.SendMessage(config.EventPuzzleToServerFragmentMove, payload)
}

// RecommendMove sends a move recommendation
func (c *TestPlayerClient) RecommendMove(targetPlayerID, fromFragmentID, toFragmentID, reasoning string) error {
	payload := map[string]interface{}{
		"targetPlayerId": targetPlayerID,
		"fromFragmentId": fromFragmentID,
		"toFragmentId":   toFragmentID,
		"reasoning":      reasoning,
	}
	return c.SendMessage(config.EventPuzzleToServerRecommendMove, payload)
}

// RespondToRecommendation responds to a move recommendation
func (c *TestPlayerClient) RespondToRecommendation(recommendationID string, accepted bool) error {
	payload := map[string]interface{}{
		"recommendationId": recommendationID,
		"accepted":         accepted,
	}
	return c.SendMessage(config.EventPuzzleToServerRecommendationResponse, payload)
}

// CreateTestServer creates a test HTTP server with WebSocket support
func CreateTestServer(t *testing.T, handler http.Handler) *httptest.Server {
	server := httptest.NewServer(handler)
	require.NotNil(t, server)
	return server
}

// WaitForCondition waits for a condition to be true
func WaitForCondition(t *testing.T, condition func() bool, timeout time.Duration, message string) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	assert.Fail(t, "Condition not met", message)
}
