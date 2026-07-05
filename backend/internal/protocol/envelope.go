package protocol

import (
	"encoding/json"
	"time"
)

// MaxClientMessageBytes is the client-to-server frame cap. Violations are
// rejected with MALFORMED_PAYLOAD; the connection stays open.
const MaxClientMessageBytes = 8 * 1024

// TimestampLayout renders the ISO 8601 UTC millisecond format used by every
// timestamp in the protocol.
const TimestampLayout = "2006-01-02T15:04:05.000Z"

// Timestamp formats t in the wire timestamp format.
func Timestamp(t time.Time) string {
	return t.UTC().Format(TimestampLayout)
}

// Auth is the client authentication wrapper.
type Auth struct {
	Token string `json:"token"`
}

// ClientFrame is any client-to-server message. Auth is required on every
// frame except SETUP_TO_SERVER_PLAYER_CONNECT, where its absence means "new
// player joining".
type ClientFrame struct {
	Event     EventType       `json:"event"`
	Auth      *Auth           `json:"auth,omitempty"`
	Payload   json.RawMessage `json:"payload"`
	Timestamp string          `json:"timestamp"`
}

// ServerFrame is any server-to-client message.
type ServerFrame struct {
	Event     EventType `json:"event"`
	Payload   any       `json:"payload"`
	Timestamp string    `json:"timestamp"`
}

// NewServerFrame wraps a payload in the outbound envelope, stamped now.
func NewServerFrame(event EventType, payload any) ServerFrame {
	return ServerFrame{Event: event, Payload: payload, Timestamp: Timestamp(time.Now())}
}
