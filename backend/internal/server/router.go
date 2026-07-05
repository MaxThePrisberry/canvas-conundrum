package server

import (
	"encoding/json"

	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/protocol"
)

// decoder turns a raw payload into its typed, validated struct.
type decoder func(json.RawMessage) (any, error)

// payload builds a decoder for one payload type. Unknown fields are
// tolerated (clients may be newer than the server); validation failures are
// reported to the client as MALFORMED_PAYLOAD by the read loop.
func payload[T interface{ Validate() error }]() decoder {
	return func(raw json.RawMessage) (any, error) {
		var p T
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, &p); err != nil {
				return nil, err
			}
		}
		if err := p.Validate(); err != nil {
			return nil, err
		}
		return p, nil
	}
}

// playerDecoders routes every client→server event a player may send after
// the connect handshake.
var playerDecoders = map[protocol.EventType]decoder{
	protocol.SystemPing:                       payload[protocol.Ping](),
	protocol.SetupToServerPlayerConfiguration: payload[protocol.PlayerConfiguration](),
	protocol.ResourceToServerLocationVerified: payload[protocol.LocationVerified](),
	protocol.ResourceToServerTriviaAnswer:     payload[protocol.TriviaAnswer](),
}

// hostDecoders routes every client→server event the host may send.
var hostDecoders = map[protocol.EventType]decoder{
	protocol.SystemPing:               payload[protocol.Ping](),
	protocol.SetupToServerStartGame:   payload[protocol.StartGame](),
	protocol.PuzzleToServerPhaseStart: payload[protocol.PuzzleStart](),
}
