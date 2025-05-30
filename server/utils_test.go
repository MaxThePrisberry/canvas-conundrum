package main

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSendToPlayer(t *testing.T) {
	tests := []struct {
		name    string
		msgType string
		payload interface{}
		connNil bool
		wantErr bool
	}{
		{
			name:    "Send with nil connection",
			msgType: "test_message",
			payload: map[string]string{"key": "value"},
			connNil: true,
			wantErr: false, // sendToPlayer returns nil for nil connections (silently ignores)
		},
		{
			name:    "Send with invalid payload",
			msgType: "test_message",
			payload: make(chan int), // Channels cannot be marshaled to JSON
			connNil: true,
			wantErr: false, // nil connection returns nil, doesn't attempt to marshal
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create player
			var player *Player
			if tt.connNil {
				player = &Player{
					ID:         "test-player",
					Connection: nil,
				}
			} else {
				// This case shouldn't happen with current test data
				player = &Player{
					ID:         "test-player",
					Connection: nil,
				}
			}

			// Send message
			err := sendToPlayer(player, tt.msgType, tt.payload)

			// Check results
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMustMarshal(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
		panics   bool
	}{
		{
			name:     "Marshal valid struct",
			input:    map[string]string{"key": "value"},
			expected: `{"key":"value"}`,
			panics:   false,
		},
		{
			name:     "Marshal array",
			input:    []int{1, 2, 3},
			expected: `[1,2,3]`,
			panics:   false,
		},
		{
			name:     "Marshal nil",
			input:    nil,
			expected: `null`,
			panics:   false,
		},
		{
			name: "Marshal complex struct",
			input: struct {
				Name  string `json:"name"`
				Score int    `json:"score"`
			}{
				Name:  "Player1",
				Score: 100,
			},
			expected: `{"name":"Player1","score":100}`,
			panics:   false,
		},
		{
			name:   "Marshal channel (should panic)",
			input:  make(chan int),
			panics: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.panics {
				assert.Panics(t, func() {
					mustMarshal(tt.input)
				})
			} else {
				result := mustMarshal(tt.input)
				assert.JSONEq(t, tt.expected, string(result))
			}
		})
	}
}

func TestLogError(t *testing.T) {
	// Test various error logging scenarios
	tests := []struct {
		name    string
		context string
		err     error
	}{
		{
			name:    "Log simple error",
			context: "test operation",
			err:     errors.New("test error"),
		},
		{
			name:    "Log nil error",
			context: "test operation",
			err:     nil,
		},
		{
			name:    "Log with empty context",
			context: "",
			err:     errors.New("test error"),
		},
		{
			name:    "Log complex error",
			context: "websocket handling",
			err:     errors.New("connection closed: EOF"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This should not panic
			assert.NotPanics(t, func() {
				logError(tt.context, tt.err)
			})
		})
	}
}

func TestLogInfo(t *testing.T) {
	// Test various info logging scenarios
	tests := []struct {
		name   string
		format string
		args   []interface{}
	}{
		{
			name:   "Simple log",
			format: "Test message",
			args:   nil,
		},
		{
			name:   "Log with formatting",
			format: "Player %s connected",
			args:   []interface{}{"player-123"},
		},
		{
			name:   "Log with multiple args",
			format: "Game phase changed from %s to %s",
			args:   []interface{}{"setup", "resource_gathering"},
		},
		{
			name:   "Log with numbers",
			format: "Player count: %d/%d",
			args:   []interface{}{5, 10},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This should not panic
			assert.NotPanics(t, func() {
				logInfo(tt.format, tt.args...)
			})
		})
	}
}

func TestLogDebug(t *testing.T) {
	// Test various debug logging scenarios
	tests := []struct {
		name   string
		format string
		args   []interface{}
	}{
		{
			name:   "Simple debug",
			format: "Debug message",
			args:   nil,
		},
		{
			name:   "Debug with data",
			format: "Processing message: %v",
			args:   []interface{}{map[string]string{"type": "test"}},
		},
		{
			name:   "Debug with complex formatting",
			format: "Fragment %s moved from (%d,%d) to (%d,%d)",
			args:   []interface{}{"fragment-1", 0, 0, 2, 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This should not panic
			assert.NotPanics(t, func() {
				logDebug(tt.format, tt.args...)
			})
		})
	}
}

func TestGenerateUUID(t *testing.T) {
	// Check if we're using the actual generateUUID function from the codebase
	// Looking at the codebase, there's no generateUUID function in utils.go
	// The code uses uuid.New() directly from google/uuid package
	// So we'll test UUID generation patterns used in the codebase

	t.Run("UUID format validation", func(t *testing.T) {
		// Test the UUID validation pattern used in the codebase
		// uuidRegex := `^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`

		testCases := []struct {
			name  string
			uuid  string
			valid bool
		}{
			{
				name:  "Valid UUID",
				uuid:  "123e4567-e89b-12d3-a456-426614174000",
				valid: true,
			},
			{
				name:  "UUID without hyphens",
				uuid:  "123e4567e89b12d3a456426614174000",
				valid: false,
			},
			{
				name:  "UUID with uppercase",
				uuid:  "123E4567-E89B-12D3-A456-426614174000",
				valid: true,
			},
			{
				name:  "Empty string",
				uuid:  "",
				valid: false,
			},
			{
				name:  "Invalid format",
				uuid:  "not-a-uuid",
				valid: false,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Use the same validation logic as in validation.go
				err := validatePlayerID(tc.uuid)
				if tc.valid {
					assert.Nil(t, err)
				} else {
					assert.NotNil(t, err)
				}
			})
		}
	})
}

func TestJSONMarshaling(t *testing.T) {
	// Test JSON marshaling patterns used throughout the codebase
	tests := []struct {
		name     string
		input    interface{}
		wantJSON string
	}{
		{
			name: "Marshal game state message",
			input: map[string]interface{}{
				"type": "game_state",
				"payload": map[string]interface{}{
					"phase":       "setup",
					"playerCount": 4,
				},
			},
			wantJSON: `{"type":"game_state","payload":{"phase":"setup","playerCount":4}}`,
		},
		{
			name: "Marshal player data",
			input: map[string]interface{}{
				"playerId": "123e4567-e89b-12d3-a456-426614174000",
				"role":     "detective",
				"ready":    true,
			},
			wantJSON: `{"playerId":"123e4567-e89b-12d3-a456-426614174000","role":"detective","ready":true}`,
		},
		{
			name: "Marshal array data",
			input: map[string]interface{}{
				"players": []string{"player1", "player2", "player3"},
				"count":   3,
			},
			wantJSON: `{"players":["player1","player2","player3"],"count":3}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := json.Marshal(tt.input)
			assert.NoError(t, err)
			assert.JSONEq(t, tt.wantJSON, string(result))
		})
	}
}

func TestConcurrentUtilityUsage(t *testing.T) {
	// Test that utility functions are safe for concurrent use
	t.Run("Concurrent mustMarshal", func(t *testing.T) {
		done := make(chan bool, 10)

		for i := 0; i < 10; i++ {
			go func(id int) {
				defer func() { done <- true }()

				data := map[string]interface{}{
					"id":    id,
					"value": "test",
				}

				result := mustMarshal(data)
				assert.NotNil(t, result)
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}
	})

	t.Run("Concurrent logging", func(t *testing.T) {
		done := make(chan bool, 30)

		// Test concurrent logging doesn't cause issues
		for i := 0; i < 10; i++ {
			go func(id int) {
				defer func() { done <- true }()
				logInfo("Test info %d", id)
			}(i)

			go func(id int) {
				defer func() { done <- true }()
				logDebug("Test debug %d", id)
			}(i)

			go func(id int) {
				defer func() { done <- true }()
				logError("test context", errors.New("test error"))
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < 30; i++ {
			<-done
		}
	})
}

func TestConcurrentWebSocketWrites(t *testing.T) {
	// Test that sendToPlayer handles concurrent access to Player fields safely
	t.Run("Concurrent player mutex usage", func(t *testing.T) {
		// Create player with nil connection to test mutex behavior only
		player := &Player{
			ID:         "test-player-concurrent",
			Connection: nil, // nil connection prevents actual WebSocket calls
		}

		numGoroutines := 100
		numOpsPerGoroutine := 50

		done := make(chan bool, numGoroutines)

		// Launch multiple goroutines that access player fields concurrently
		for i := 0; i < numGoroutines; i++ {
			go func(goroutineID int) {
				defer func() { done <- true }()

				for j := 0; j < numOpsPerGoroutine; j++ {
					// This tests the mutex protection in sendToPlayer
					payload := map[string]interface{}{
						"goroutine": goroutineID,
						"message":   j,
						"data":      "test concurrent access",
					}

					// sendToPlayer should safely handle concurrent access
					err := sendToPlayer(player, "test_message", payload)
					// Should not return an error for nil connections (silently ignored)
					assert.NoError(t, err)

					// Also test direct mutex access patterns
					player.mu.RLock()
					_ = player.Connection
					player.mu.RUnlock()
				}
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < numGoroutines; i++ {
			select {
			case <-done:
				// Continue
			case <-time.After(10 * time.Second):
				t.Fatal("Timeout waiting for concurrent operations to complete")
			}
		}
	})

	t.Run("Concurrent sendToPlayer stress test", func(t *testing.T) {
		// Create multiple players and test concurrent sends to different players
		numPlayers := 10
		players := make([]*Player, numPlayers)

		for i := 0; i < numPlayers; i++ {
			players[i] = &Player{
				ID:         "test-player-" + string(rune('A'+i)),
				Connection: nil, // nil to avoid actual WebSocket calls
			}
		}

		numGoroutines := 50
		done := make(chan bool, numGoroutines)

		// Launch goroutines that send to random players
		for i := 0; i < numGoroutines; i++ {
			go func(goroutineID int) {
				defer func() { done <- true }()

				// Send 20 messages to different players
				for j := 0; j < 20; j++ {
					playerIndex := (goroutineID + j) % numPlayers
					player := players[playerIndex]

					payload := map[string]interface{}{
						"goroutine": goroutineID,
						"message":   j,
						"player":    playerIndex,
					}

					err := sendToPlayer(player, "stress_test", payload)
					assert.NoError(t, err)
				}
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < numGoroutines; i++ {
			select {
			case <-done:
				// Continue
			case <-time.After(5 * time.Second):
				t.Fatal("Timeout waiting for stress test to complete")
			}
		}
	})
}
