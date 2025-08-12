package utils

import (
	"canvas-conundrum/constants"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessage(t *testing.T) {
	t.Run("MarshalMessage", func(t *testing.T) {
		msg := &Message{
			Event: constants.EventSetupToServerPlayerConfiguration,
			Auth: &Auth{
				Token: "test-token",
			},
			Payload:   json.RawMessage(`{"test": "data"}`),
			Timestamp: time.Now().Format(time.RFC3339),
		}

		data, err := json.Marshal(msg)
		require.NoError(t, err)

		// Unmarshal back to verify
		var decoded Message
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, msg.Event, decoded.Event)
		assert.Equal(t, msg.Auth.Token, decoded.Auth.Token)
		assert.JSONEq(t, `{"test": "data"}`, string(decoded.Payload))
	})
}

func TestServerMessage(t *testing.T) {
	t.Run("MarshalServerMessage", func(t *testing.T) {
		msg := &ServerMessage{
			Event:     constants.EventSetupToPlayerRolesAvailable,
			Payload:   map[string]interface{}{"roles": []interface{}{}},
			Timestamp: time.Now().Format(time.RFC3339),
		}

		data, err := msg.Marshal()
		require.NoError(t, err)

		// Unmarshal back to verify
		var decoded ServerMessage
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, msg.Event, decoded.Event)

		// Check payload
		payloadJSON, err := json.Marshal(decoded.Payload)
		require.NoError(t, err)
		assert.JSONEq(t, `{"roles": []}`, string(payloadJSON))
	})
}

func TestParseMessage(t *testing.T) {
	t.Run("ValidMessage", func(t *testing.T) {
		jsonData := `{
			"event": "TEST_EVENT",
			"auth": {
				"token": "player-123"
			},
			"payload": {
				"key": "value"
			},
			"timestamp": "2024-01-01T12:00:00Z"
		}`

		msg, err := ParseMessage([]byte(jsonData))
		require.NoError(t, err)

		assert.Equal(t, "TEST_EVENT", msg.Event)
		assert.Equal(t, "player-123", msg.Auth.Token)
		assert.Contains(t, string(msg.Payload), "key")
		assert.Contains(t, string(msg.Payload), "value")
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		jsonData := `{invalid json}`

		msg, err := ParseMessage([]byte(jsonData))
		assert.Error(t, err)
		assert.Nil(t, msg)
	})

	t.Run("MissingAuth", func(t *testing.T) {
		jsonData := `{
			"event": "TEST_EVENT",
			"payload": {},
			"timestamp": "2024-01-01T12:00:00Z"
		}`

		msg, err := ParseMessage([]byte(jsonData))
		require.NoError(t, err)
		assert.Nil(t, msg.Auth)
	})
}

func TestNewServerMessage(t *testing.T) {
	payload := map[string]interface{}{
		"test":   "data",
		"number": 42,
	}

	msg := NewServerMessage(constants.EventSetupToPlayerRolesAvailable, payload)

	assert.NotNil(t, msg)
	assert.Equal(t, constants.EventSetupToPlayerRolesAvailable, msg.Event)
	assert.NotNil(t, msg.Payload)
	assert.NotZero(t, msg.Timestamp)

	// Verify payload - it's already an interface{}, not RawMessage
	decodedPayload, ok := msg.Payload.(map[string]interface{})
	require.True(t, ok)

	assert.Equal(t, "data", decodedPayload["test"])
	assert.Equal(t, 42, decodedPayload["number"])
}

func TestCreateErrorPayload(t *testing.T) {
	payload := CreateErrorPayload(
		"test_error",
		constants.ErrorCodeInvalidToken,
		"Invalid authentication token",
		"Please reconnect",
	)

	assert.Equal(t, "test_error", payload["errorType"])
	assert.Equal(t, constants.ErrorCodeInvalidToken, payload["errorCode"])
	assert.Equal(t, "Invalid authentication token", payload["message"])
	assert.Equal(t, "Please reconnect", payload["details"])
	assert.True(t, payload["retryable"].(bool))
	assert.Equal(t, "error", payload["severity"])
}

func TestComplexPayloads(t *testing.T) {
	t.Run("NestedStructure", func(t *testing.T) {
		payload := map[string]interface{}{
			"player": map[string]interface{}{
				"id":   "p1",
				"name": "Alice",
				"stats": map[string]interface{}{
					"score":     100,
					"questions": 10,
					"correct":   7,
				},
			},
			"tokens": []int{10, 20, 30},
			"active": true,
		}

		msg := NewServerMessage("TEST", payload)
		data, err := msg.Marshal()
		require.NoError(t, err)

		// Parse back
		var decoded ServerMessage
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		decodedPayload, ok := decoded.Payload.(map[string]interface{})
		require.True(t, ok)

		player := decodedPayload["player"].(map[string]interface{})
		assert.Equal(t, "p1", player["id"])
		assert.Equal(t, "Alice", player["name"])

		stats := player["stats"].(map[string]interface{})
		assert.Equal(t, float64(100), stats["score"])
		assert.Equal(t, float64(10), stats["questions"])
		assert.Equal(t, float64(7), stats["correct"])

		tokens := decodedPayload["tokens"].([]interface{})
		assert.Len(t, tokens, 3)
		assert.Equal(t, float64(10), tokens[0])

		assert.True(t, decodedPayload["active"].(bool))
	})

	t.Run("EmptyPayload", func(t *testing.T) {
		msg := NewServerMessage("EMPTY", nil)

		data, err := msg.Marshal()
		require.NoError(t, err)

		var decoded ServerMessage
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, "EMPTY", decoded.Event)
		assert.Nil(t, decoded.Payload)
	})

	t.Run("ArrayPayload", func(t *testing.T) {
		payload := []map[string]interface{}{
			{"id": "1", "name": "First"},
			{"id": "2", "name": "Second"},
		}

		msg := NewServerMessage("ARRAY", payload)

		data, err := msg.Marshal()
		require.NoError(t, err)

		var decoded ServerMessage
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		decodedPayload, ok := decoded.Payload.([]interface{})
		require.True(t, ok)

		assert.Len(t, decodedPayload, 2)
		firstItem := decodedPayload[0].(map[string]interface{})
		assert.Equal(t, "1", firstItem["id"])
		assert.Equal(t, "First", firstItem["name"])
	})
}

func TestTimestampHandling(t *testing.T) {
	now := time.Now()

	msg := &ServerMessage{
		Event:     "TEST",
		Payload:   map[string]interface{}{},
		Timestamp: now.Format(time.RFC3339),
	}

	data, err := msg.Marshal()
	require.NoError(t, err)

	var decoded ServerMessage
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	// Timestamps should match
	assert.Equal(t, msg.Timestamp, decoded.Timestamp)
}
