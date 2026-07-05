package integration

import (
	"testing"
	"time"

	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/app"
	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/protocol"
)

func TestPingPongEcho(t *testing.T) {
	t.Parallel()
	h := Start(t, nil, nil)
	p := ConnectNewPlayer(t, h)

	stamp := protocol.Timestamp(time.Now())
	p.Send(string(protocol.SystemPing), p.ID, protocol.Ping{ClientTimestamp: stamp, SequenceNumber: 42})
	pong := payloadAs[protocol.Pong](t, p.Expect(protocol.SystemPong))
	if pong.ClientTimestamp != stamp || pong.SequenceNumber != 42 {
		t.Errorf("pong = %+v", pong)
	}

	host, _ := ConnectHost(t, h)
	host.Send(string(protocol.SystemPing), host.UUID, protocol.Ping{ClientTimestamp: stamp, SequenceNumber: 1})
	hostPong := payloadAs[protocol.Pong](t, host.Expect(protocol.SystemPong))
	if hostPong.SequenceNumber != 1 {
		t.Errorf("host pong = %+v", hostPong)
	}
}

// A silent client is treated as disconnected after the configured window,
// while a pinging client survives. Reconnection with the same token works
// afterwards.
func TestHeartbeatSilenceDisconnects(t *testing.T) {
	t.Parallel()
	h := Start(t, nil, func(o *app.Options) { o.DisconnectAfter = 250 * time.Millisecond })
	host, _ := ConnectHost(t, h)
	host.StartPinger(h.HostUUID, 50*time.Millisecond)

	quiet := ConnectNewPlayer(t, h)
	chatty := ConnectNewPlayer(t, h)
	chatty.StartPinger(chatty.ID, 50*time.Millisecond)

	// The quiet player must be swept out...
	notice := payloadAs[protocol.PlayerDisconnected](t, host.Expect(protocol.SystemToHostPlayerDisconnected))
	if notice.PlayerID != quiet.ID {
		t.Fatalf("disconnected player = %s, want the quiet one %s", notice.PlayerID, quiet.ID)
	}

	// ...with a close code that allows auto-reconnect (1001, not 1000/4xxx).
	quiet.ExpectClose(protocol.CloseGoingAway)

	// The pinging player must still be connected: a ping still pongs.
	chatty.Send(string(protocol.SystemPing), chatty.ID, protocol.Ping{SequenceNumber: 99})
	for {
		pong := payloadAs[protocol.Pong](t, chatty.Expect(protocol.SystemPong))
		if pong.SequenceNumber == 99 {
			break
		}
	}

	// The swept player's token remains valid for reconnection.
	_, confirmed := ReconnectPlayer(t, h, quiet.ID)
	if !confirmed.IsReconnection {
		t.Error("expected a reconnection handshake")
	}
}
