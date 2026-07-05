package integration

import (
	"strings"
	"testing"
	"time"

	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/app"
	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/config"
	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/protocol"
	"github.com/google/uuid"
)

// The upgrade must complete before an invalid host token is rejected, so
// browser clients can read the 4001 close code.
func TestHostConnectUnknownUUIDCloses4001(t *testing.T) {
	h := Start(t, nil, nil)
	c := DialHost(t, h, uuid.NewString())
	c.ExpectClose(protocol.CloseUnauthorized)
}

func TestPlayerConnectFrameDeadline(t *testing.T) {
	h := Start(t, nil, func(o *app.Options) { o.ConnectDeadline = 100 * time.Millisecond })
	c := DialPlayer(t, h)
	// Send nothing: the 10s (here 100ms) deadline must close with 4001.
	c.ExpectClose(protocol.CloseUnauthorized)
}

func TestPlayerFirstFrameMustBeConnect(t *testing.T) {
	h := Start(t, nil, nil)
	c := DialPlayer(t, h)
	c.SendUnauthenticated(string(protocol.SystemPing), protocol.Ping{SequenceNumber: 1})
	c.ExpectClose(protocol.CloseUnauthorized)
}

func TestPlayerReconnectUnknownTokenCloses4001(t *testing.T) {
	h := Start(t, nil, nil)
	c := DialPlayer(t, h)
	c.Send(string(protocol.SetupToServerPlayerConnect), uuid.NewString(), struct{}{})
	c.ExpectClose(protocol.CloseUnauthorized)
}

func TestNewJoinAtMaxPlayersCloses4002(t *testing.T) {
	h := Start(t, func(c *config.Config) { c.MaxPlayers = 2 }, nil)
	ConnectNewPlayer(t, h)
	ConnectNewPlayer(t, h)

	c := DialPlayer(t, h)
	c.SendUnauthenticated(string(protocol.SetupToServerPlayerConnect), struct{}{})
	c.ExpectClose(protocol.CloseJoinRejected)
}

func TestNewJoinPastSetupCloses4002(t *testing.T) {
	h := Start(t, nil, nil)
	host, _ := JoinConfigured(t, h, 2)
	host.StartGame()

	c := DialPlayer(t, h)
	c.SendUnauthenticated(string(protocol.SetupToServerPlayerConnect), struct{}{})
	c.ExpectClose(protocol.CloseJoinRejected)
}

// Frames above 8 KB get a MALFORMED_PAYLOAD error and the connection
// survives (the spec forbids closing for this).
func TestOversizedFrameRejectedConnectionSurvives(t *testing.T) {
	h := Start(t, nil, nil)
	p := ConnectNewPlayer(t, h)

	big := `{"event":"SYSTEM_PING","auth":{"token":"` + p.ID + `"},"payload":{"clientTimestamp":"` +
		strings.Repeat("x", protocol.MaxClientMessageBytes) + `"},"timestamp":"2025-06-15T14:23:05.000Z"}`
	p.SendRaw([]byte(big))
	p.ExpectError(protocol.SystemToClientError, protocol.ErrMalformedPayload)

	// Connection still works.
	p.Send(string(protocol.SystemPing), p.ID, protocol.Ping{SequenceNumber: 7})
	pong := payloadAs[protocol.Pong](t, p.Expect(protocol.SystemPong))
	if pong.SequenceNumber != 7 {
		t.Errorf("pong sequence = %d, want 7", pong.SequenceNumber)
	}
}

func TestFrameWithoutAuthRejected(t *testing.T) {
	h := Start(t, nil, nil)
	p := ConnectNewPlayer(t, h)
	p.SendUnauthenticated(string(protocol.SystemPing), protocol.Ping{SequenceNumber: 1})
	errPayload := p.ExpectError(protocol.SystemToClientError, protocol.ErrUnauthorized)
	if errPayload.ErrorType != protocol.ErrorTypeAuth {
		t.Errorf("errorType = %s, want auth_error", errPayload.ErrorType)
	}
}

func TestFrameWithWrongTokenRejected(t *testing.T) {
	h := Start(t, nil, nil)
	p := ConnectNewPlayer(t, h)
	p.Send(string(protocol.SystemPing), uuid.NewString(), protocol.Ping{SequenceNumber: 1})
	p.ExpectError(protocol.SystemToClientError, protocol.ErrUnauthorized)
}

func TestUnknownEventRejected(t *testing.T) {
	h := Start(t, nil, nil)
	p := ConnectNewPlayer(t, h)
	p.Send("TOTALLY_MADE_UP_EVENT", p.ID, struct{}{})
	p.ExpectError(protocol.SystemToClientError, protocol.ErrMalformedPayload)
}

func TestMalformedJSONRejected(t *testing.T) {
	h := Start(t, nil, nil)
	p := ConnectNewPlayer(t, h)
	p.SendRaw([]byte(`{not json`))
	p.ExpectError(protocol.SystemToClientError, protocol.ErrMalformedPayload)
}

func TestPlayerNameLengthLimit(t *testing.T) {
	h := Start(t, nil, nil)
	p := ConnectNewPlayer(t, h)
	p.Expect(protocol.SetupToPlayerRolesAvailable)

	p.Send(string(protocol.SetupToServerPlayerConfiguration), p.ID, protocol.PlayerConfiguration{
		SelectedRole:        "detective",
		SelectedSpecialties: []string{fixtureCategories[0]},
		PlayerName:          strings.Repeat("x", protocol.MaxPlayerNameLen+1),
	})
	p.ExpectError(protocol.SystemToClientError, protocol.ErrMalformedPayload)

	// Empty name is below the 1-character minimum.
	p.Send(string(protocol.SetupToServerPlayerConfiguration), p.ID, protocol.PlayerConfiguration{
		SelectedRole:        "detective",
		SelectedSpecialties: []string{fixtureCategories[0]},
		PlayerName:          "",
	})
	p.ExpectError(protocol.SystemToClientError, protocol.ErrMalformedPayload)
}
