package utils

import (
	"encoding/json"
	"time"
)

// Message represents a WebSocket message structure
type Message struct {
	Event     string          `json:"event"`
	Auth      *Auth           `json:"auth,omitempty"`
	Payload   json.RawMessage `json:"payload"`
	Timestamp string          `json:"timestamp"`
}

// Auth represents the authentication portion of a message
type Auth struct {
	Token string `json:"token"`
}

// ServerMessage represents a server-to-client message
type ServerMessage struct {
	Event     string      `json:"event"`
	Payload   interface{} `json:"payload"`
	Timestamp string      `json:"timestamp"`
}

// NewServerMessage creates a new server message
func NewServerMessage(event string, payload interface{}) *ServerMessage {
	return &ServerMessage{
		Event:     event,
		Payload:   payload,
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

// Marshal converts a ServerMessage to JSON bytes
func (m *ServerMessage) Marshal() ([]byte, error) {
	return json.Marshal(m)
}

// ParseMessage parses a raw WebSocket message
func ParseMessage(data []byte) (*Message, error) {
	var msg Message
	err := json.Unmarshal(data, &msg)
	if err != nil {
		return nil, err
	}

	// Set timestamp if not provided
	if msg.Timestamp == "" {
		msg.Timestamp = time.Now().Format(time.RFC3339)
	}

	return &msg, nil
}

// CreateErrorPayload creates a standard error payload
func CreateErrorPayload(errorType, errorCode, message, details string) map[string]interface{} {
	return map[string]interface{}{
		"errorType": errorType,
		"errorCode": errorCode,
		"message":   message,
		"details":   details,
		"retryable": true,
		"severity":  "error",
	}
}

// MarshalJSON is a helper to marshal any object to JSON bytes
func MarshalJSON(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// UnmarshalJSON is a helper to unmarshal JSON bytes to an object
func UnmarshalJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
