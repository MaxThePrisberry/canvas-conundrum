package integration

import (
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/protocol"
	"github.com/gorilla/websocket"
)

const expectTimeout = 3 * time.Second

// Frame is one received server frame, payload still raw for per-test typing.
type Frame struct {
	Event     string          `json:"event"`
	Payload   json.RawMessage `json:"payload"`
	Timestamp string          `json:"timestamp"`
}

// Client is a real WebSocket client. A background reader feeds frames into
// an inbox; Expect* helpers consume it.
type Client struct {
	t      *testing.T
	conn   *websocket.Conn
	writeM sync.Mutex // gorilla conns forbid concurrent writers (pinger vs test)
	frames chan Frame
	closed chan int // close code, once
}

// DialPlayer opens a player socket (no connect frame yet).
func DialPlayer(t *testing.T, h *Harness) *Client {
	t.Helper()
	return dial(t, h.WSURL+"/ws")
}

// DialHost opens a host socket for the given UUID.
func DialHost(t *testing.T, h *Harness, hostUUID string) *Client {
	t.Helper()
	return dial(t, h.WSURL+"/ws/host/"+hostUUID)
}

func dial(t *testing.T, url string) *Client {
	t.Helper()
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial %s: %v", url, err)
	}
	c := &Client{
		t:      t,
		conn:   conn,
		frames: make(chan Frame, 256),
		closed: make(chan int, 1),
	}
	go c.readLoop()
	t.Cleanup(c.Close)
	return c
}

func (c *Client) readLoop() {
	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			var ce *websocket.CloseError
			if errors.As(err, &ce) {
				c.closed <- ce.Code
			} else {
				c.closed <- -1 // abnormal / local close
			}
			close(c.frames)
			return
		}
		var f Frame
		if json.Unmarshal(data, &f) == nil {
			c.frames <- f
		}
	}
}

// Close tears the socket down without a close handshake (simulates a
// dropped connection).
func (c *Client) Close() {
	_ = c.conn.Close()
}

// send writes one client frame. token=="" omits auth entirely.
func (c *Client) send(event string, token string, payload any) {
	c.t.Helper()
	frame := map[string]any{
		"event":     event,
		"payload":   payload,
		"timestamp": protocol.Timestamp(time.Now()),
	}
	if token != "" {
		frame["auth"] = map[string]string{"token": token}
	}
	if err := c.writeJSON(frame); err != nil {
		c.t.Fatalf("write %s: %v", event, err)
	}
}

func (c *Client) writeJSON(v any) error {
	c.writeM.Lock()
	defer c.writeM.Unlock()
	return c.conn.WriteJSON(v)
}

// Send writes an authenticated client frame.
func (c *Client) Send(event, token string, payload any) {
	c.t.Helper()
	if token == "" {
		c.t.Fatal("Send requires a token; use SendUnauthenticated")
	}
	c.send(event, token, payload)
}

// SendUnauthenticated writes a frame without an auth block.
func (c *Client) SendUnauthenticated(event string, payload any) {
	c.t.Helper()
	c.send(event, "", payload)
}

// SendRaw writes raw bytes (for malformed / oversized frame tests).
func (c *Client) SendRaw(data []byte) {
	c.t.Helper()
	c.writeM.Lock()
	defer c.writeM.Unlock()
	if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		c.t.Fatalf("write raw: %v", err)
	}
}

// Expect consumes frames until one matches event, failing on timeout or
// socket close. Non-matching frames are discarded (broadcasts arrive
// interleaved; tests assert what they care about).
func (c *Client) Expect(event protocol.EventType) json.RawMessage {
	c.t.Helper()
	deadline := time.After(expectTimeout)
	for {
		select {
		case f, ok := <-c.frames:
			if !ok {
				c.t.Fatalf("socket closed while waiting for %s", event)
			}
			if f.Event == string(event) {
				return f.Payload
			}
		case <-deadline:
			c.t.Fatalf("timed out waiting for %s", event)
		}
	}
}

// ExpectNone asserts no frame of the given event arrives within d.
func (c *Client) ExpectNone(event protocol.EventType, d time.Duration) {
	c.t.Helper()
	deadline := time.After(d)
	for {
		select {
		case f, ok := <-c.frames:
			if !ok {
				return // closed: certainly no more frames
			}
			if f.Event == string(event) {
				c.t.Fatalf("received unwanted %s: %s", event, f.Payload)
			}
		case <-deadline:
			return
		}
	}
}

// ExpectError waits for an error event carrying the given code.
func (c *Client) ExpectError(event protocol.EventType, code protocol.ErrorCode) protocol.ErrorPayload {
	c.t.Helper()
	raw := c.Expect(event)
	var p protocol.ErrorPayload
	mustUnmarshal(c.t, raw, &p)
	if p.ErrorCode != code {
		c.t.Fatalf("error code = %s (%s), want %s", p.ErrorCode, p.Message, code)
	}
	return p
}

// ExpectClose waits for the server to close the socket with the given code.
func (c *Client) ExpectClose(code int) {
	c.t.Helper()
	select {
	case got := <-c.closed:
		if got != code {
			c.t.Fatalf("close code = %d, want %d", got, code)
		}
	case <-time.After(expectTimeout):
		c.t.Fatalf("timed out waiting for close %d", code)
	}
}

// StartPinger sends SYSTEM_PING every interval until the test ends, keeping
// this client alive past heartbeat sweeps.
func (c *Client) StartPinger(token string, interval time.Duration) {
	stop := make(chan struct{})
	c.t.Cleanup(func() { close(stop) })
	go func() {
		seq := 0
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				seq++
				frame := map[string]any{
					"event": string(protocol.SystemPing),
					"auth":  map[string]string{"token": token},
					"payload": protocol.Ping{
						ClientTimestamp: protocol.Timestamp(time.Now()),
						SequenceNumber:  seq,
					},
					"timestamp": protocol.Timestamp(time.Now()),
				}
				if c.writeJSON(frame) != nil {
					return
				}
			}
		}
	}()
}

func mustUnmarshal(t *testing.T, raw json.RawMessage, v any) {
	t.Helper()
	if err := json.Unmarshal(raw, v); err != nil {
		t.Fatalf("unmarshal %T from %s: %v", v, raw, err)
	}
}

// payloadAs decodes a raw payload into T.
func payloadAs[T any](t *testing.T, raw json.RawMessage) T {
	t.Helper()
	var v T
	mustUnmarshal(t, raw, &v)
	return v
}
