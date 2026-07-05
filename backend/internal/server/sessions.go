// Package server is the transport layer: WebSocket endpoints, HTTP asset
// endpoints, and the framing/auth/routing between sockets and the game
// engine.
package server

import (
	"log/slog"
	"sync"
	"time"

	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/protocol"
	"github.com/gorilla/websocket"
)

const (
	outboundBuffer = 256
	writeTimeout   = 10 * time.Second
)

// Session owns one WebSocket connection. It implements game.Client: Send
// enqueues onto the write pump and CloseWithCode performs an async close
// handshake — both non-blocking, callable from the engine goroutine.
type Session struct {
	conn *websocket.Conn
	log  *slog.Logger

	out     chan protocol.ServerFrame
	closing chan int
	once    sync.Once
}

func newSession(conn *websocket.Conn, log *slog.Logger) *Session {
	s := &Session{
		conn:    conn,
		log:     log,
		out:     make(chan protocol.ServerFrame, outboundBuffer),
		closing: make(chan int, 1),
	}
	go s.writePump()
	return s
}

// Send enqueues one frame. If the client is too slow to drain its buffer the
// connection is closed rather than blocking the engine.
func (s *Session) Send(event protocol.EventType, payload any) {
	select {
	case s.out <- protocol.NewServerFrame(event, payload):
	default:
		s.log.Warn("outbound buffer full, closing slow client")
		s.CloseWithCode(protocol.CloseGoingAway)
	}
}

// CloseWithCode signals the write pump to send a close frame with the given
// code and tear down the connection. Idempotent.
func (s *Session) CloseWithCode(code int) {
	s.once.Do(func() { s.closing <- code })
}

func (s *Session) writePump() {
	defer s.conn.Close()
	for {
		select {
		case frame := <-s.out:
			s.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := s.conn.WriteJSON(frame); err != nil {
				return
			}
		case code := <-s.closing:
			msg := websocket.FormatCloseMessage(code, "")
			deadline := time.Now().Add(time.Second)
			s.conn.SetWriteDeadline(deadline)
			_ = s.conn.WriteControl(websocket.CloseMessage, msg, deadline)
			return
		}
	}
}
