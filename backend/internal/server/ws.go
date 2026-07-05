package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/protocol"
	"github.com/gorilla/websocket"
)

// readLimit is a DoS backstop only. The protocol's 8 KB cap is enforced
// per-message in the read loop so violations get a MALFORMED_PAYLOAD error
// frame with the connection surviving, as the spec requires — gorilla's
// SetReadLimit would kill the connection instead.
const readLimit = 64 * 1024

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	// Same-origin is enforced by the deployment topology (nginx/vite proxy);
	// the server itself accepts any origin.
	CheckOrigin: func(*http.Request) bool { return true },
}

// handlePlayer serves /ws. Per the spec's handshake choreography the upgrade
// always completes; rejections then close with 4001/4002/4003 and no
// application frames, so browser clients can read the close code.
func (s *Server) handlePlayer(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	sess := newSession(conn, s.log)
	conn.SetReadLimit(readLimit)

	// First frame must be SETUP_TO_SERVER_PLAYER_CONNECT within the deadline.
	conn.SetReadDeadline(time.Now().Add(s.connectDeadline))
	_, data, err := conn.ReadMessage()
	if err != nil {
		sess.CloseWithCode(protocol.CloseUnauthorized)
		return
	}
	var frame protocol.ClientFrame
	if json.Unmarshal(data, &frame) != nil || frame.Event != protocol.SetupToServerPlayerConnect {
		sess.CloseWithCode(protocol.CloseUnauthorized)
		return
	}

	token, hasToken := "", false
	if frame.Auth != nil && frame.Auth.Token != "" {
		token, hasToken = frame.Auth.Token, true
	}

	res := s.engine.ConnectPlayer(token, hasToken, sess)
	if !res.OK {
		return // engine closed the socket with the appropriate code
	}

	conn.SetReadDeadline(time.Time{}) // heartbeat sweep governs from here on
	s.readLoop(sess, readLoopConfig{
		expectedToken: res.PlayerID,
		decoders:      playerDecoders,
		errorEvent:    protocol.SystemToClientError,
		deliver: func(event protocol.EventType, payload any) {
			s.engine.PlayerFrame(res.PlayerID, sess, event, payload)
		},
		closed: func() { s.engine.PlayerSocketClosed(res.PlayerID, sess) },
	})
}

// handleHost serves /ws/host/{uuid}. The token is judged from the URL
// immediately after the upgrade completes.
func (s *Server) handleHost(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	sess := newSession(conn, s.log)
	conn.SetReadLimit(readLimit)

	if r.PathValue("uuid") != s.hostUUID {
		sess.CloseWithCode(protocol.CloseUnauthorized)
		return
	}

	s.engine.ConnectHost(sess)

	s.readLoop(sess, readLoopConfig{
		expectedToken: s.hostUUID,
		decoders:      hostDecoders,
		errorEvent:    protocol.SystemToHostError,
		deliver: func(event protocol.EventType, payload any) {
			s.engine.HostFrame(sess, event, payload)
		},
		closed: func() { s.engine.HostSocketClosed(sess) },
	})
}

type readLoopConfig struct {
	expectedToken string
	decoders      map[protocol.EventType]decoder
	errorEvent    protocol.EventType
	deliver       func(protocol.EventType, any)
	closed        func()
}

// readLoop drives one authenticated connection until it closes: cap check,
// envelope parse, auth check, payload decode+validate, then hand off to the
// engine. Protocol violations produce error frames, never closes.
func (s *Server) readLoop(sess *Session, cfg readLoopConfig) {
	defer cfg.closed()
	for {
		_, data, err := sess.conn.ReadMessage()
		if err != nil {
			return
		}

		if len(data) > protocol.MaxClientMessageBytes {
			sess.Send(cfg.errorEvent, protocol.ErrorPayload{
				ErrorType: protocol.ErrorTypeValidation,
				ErrorCode: protocol.ErrMalformedPayload,
				Message:   "message exceeds the 8 KB limit",
			})
			continue
		}

		var frame protocol.ClientFrame
		if err := json.Unmarshal(data, &frame); err != nil {
			sess.Send(cfg.errorEvent, protocol.ErrorPayload{
				ErrorType: protocol.ErrorTypeValidation,
				ErrorCode: protocol.ErrMalformedPayload,
				Message:   "message is not a valid client frame",
			})
			continue
		}

		if frame.Auth == nil || frame.Auth.Token != cfg.expectedToken {
			sess.Send(cfg.errorEvent, protocol.ErrorPayload{
				ErrorType: protocol.ErrorTypeAuth,
				ErrorCode: protocol.ErrUnauthorized,
				Message:   "missing or invalid auth token",
			})
			continue
		}

		dec, ok := cfg.decoders[frame.Event]
		if !ok {
			sess.Send(cfg.errorEvent, protocol.ErrorPayload{
				ErrorType: protocol.ErrorTypeValidation,
				ErrorCode: protocol.ErrMalformedPayload,
				Message:   "unknown or misdirected event",
				Details:   string(frame.Event) + " is not a recognized client event on this connection",
			})
			continue
		}
		typed, err := dec(frame.Payload)
		if err != nil {
			sess.Send(cfg.errorEvent, protocol.ErrorPayload{
				ErrorType: protocol.ErrorTypeValidation,
				ErrorCode: protocol.ErrMalformedPayload,
				Message:   "payload validation failed",
				Details:   err.Error(),
			})
			continue
		}

		cfg.deliver(frame.Event, typed)
	}
}
