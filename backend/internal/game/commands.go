// Package game implements the game engine: a single goroutine (actor) that
// owns all game state and processes commands from the transport layer,
// timers, and HTTP handlers. Single ownership makes the spec's serial
// guarantees (first-wins configuration, victory check after every mutation,
// atomic pause) hold by construction.
package game

import (
	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/protocol"
)

// Client is the engine's handle on one connected WebSocket. Implementations
// must be safe to call from the engine goroutine and never block (buffered
// send, async close).
type Client interface {
	// Send enqueues one server frame.
	Send(event protocol.EventType, payload any)
	// CloseWithCode performs the WebSocket close handshake with code and
	// tears the connection down.
	CloseWithCode(code int)
}

type command interface{ isCommand() }

// cmdHostConnect attaches a new host socket (URL token already verified by
// the transport). Supersedes any existing host socket.
type cmdHostConnect struct{ client Client }

// cmdPlayerConnect processes a SETUP_TO_SERVER_PLAYER_CONNECT frame.
// HasToken distinguishes reconnection (true) from a new join.
type cmdPlayerConnect struct {
	token    string
	hasToken bool
	client   Client
	reply    chan<- PlayerConnectResult
}

// PlayerConnectResult tells the read pump whether the connection was
// accepted and, if so, which player identity owns it. On rejection the
// engine has already closed the socket.
type PlayerConnectResult struct {
	OK       bool
	PlayerID string
}

// cmdPlayerFrame is a decoded, validated post-handshake player frame.
type cmdPlayerFrame struct {
	playerID string
	client   Client
	event    protocol.EventType
	payload  any
}

// cmdHostFrame is a decoded, validated post-handshake host frame.
type cmdHostFrame struct {
	client  Client
	event   protocol.EventType
	payload any
}

// cmdPlayerClosed reports a player socket read-pump exit.
type cmdPlayerClosed struct {
	playerID string
	client   Client
}

// cmdHostClosed reports a host socket read-pump exit.
type cmdHostClosed struct{ client Client }

// cmdTimer is posted by the timer service when a named timer fires.
type cmdTimer struct {
	name string
	gen  uint64
}

func (cmdHostConnect) isCommand()   {}
func (cmdPlayerConnect) isCommand() {}
func (cmdPlayerFrame) isCommand()   {}
func (cmdHostFrame) isCommand()     {}
func (cmdPlayerClosed) isCommand()  {}
func (cmdHostClosed) isCommand()    {}
func (cmdTimer) isCommand()         {}
