package integration

import (
	"bytes"
	"encoding/json"
	"image/png"
	"io"
	"net/http"
	"testing"

	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/protocol"
)

// fetchAsset GETs an /api path with an optional bearer token and returns the
// response plus body.
func fetchAsset(t *testing.T, h *Harness, path, token string) (*http.Response, []byte) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, h.BaseURL+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	return resp, body
}

func assertAssetError(t *testing.T, resp *http.Response, body []byte, status int, code protocol.ErrorCode) {
	t.Helper()
	if resp.StatusCode != status {
		t.Fatalf("status = %d (%s), want %d", resp.StatusCode, body, status)
	}
	var e struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &e); err != nil {
		t.Fatalf("error body is not JSON: %s", body)
	}
	if e.Error != string(code) {
		t.Errorf("error code = %s, want %s", e.Error, code)
	}
	if e.Message == "" {
		t.Error("error message must be non-empty")
	}
}

func assertPNG(t *testing.T, resp *http.Response, body []byte, wantSide int) {
	t.Helper()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d: %s", resp.StatusCode, body)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "image/png" {
		t.Errorf("content-type = %s", ct)
	}
	if cc := resp.Header.Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("cache-control = %s", cc)
	}
	img, err := png.Decode(bytes.NewReader(body))
	if err != nil {
		t.Fatalf("body is not PNG: %v", err)
	}
	if wantSide > 0 && img.Bounds().Dx() != wantSide {
		t.Errorf("image side = %d, want %d", img.Bounds().Dx(), wantSide)
	}
}

func TestSegmentEndpointGating(t *testing.T) {
	t.Parallel()
	h := Start(t, noSpecialty, nil)
	host, players := JoinConfigured(t, h, 2)
	a, b := players[0], players[1]
	host.StartGame()

	// During resource gathering: tiles don't exist yet → FORBIDDEN_PHASE.
	resp, body := fetchAsset(t, h, "/api/segments/segment_a1", a.ID)
	assertAssetError(t, resp, body, http.StatusForbidden, protocol.ErrForbiddenPhase)

	PlayResource(t, host, players, 2, nil)
	a.Expect(protocol.PuzzleToClientPhaseLoad)

	// Own segment: 200 PNG (96px source / 3 grid = 32px tiles).
	resp, body = fetchAsset(t, h, "/api/segments/segment_a1", a.ID)
	assertPNG(t, resp, body, 32)

	// Another player's segment, not visible on the grid → FORBIDDEN_NOT_OWNER.
	resp, body = fetchAsset(t, h, "/api/segments/segment_a2", a.ID)
	assertAssetError(t, resp, body, http.StatusForbidden, protocol.ErrForbiddenNotOwner)
	// ...but its owner may fetch it.
	resp, body = fetchAsset(t, h, "/api/segments/segment_a2", b.ID)
	assertPNG(t, resp, body, 32)

	// The host has read-only access to everything.
	resp, body = fetchAsset(t, h, "/api/segments/segment_c3", h.HostUUID)
	assertPNG(t, resp, body, 32)

	// Unknown segment id → NOT_FOUND (c4 is off a 3×3 grid).
	resp, body = fetchAsset(t, h, "/api/segments/segment_c4", a.ID)
	assertAssetError(t, resp, body, http.StatusNotFound, protocol.ErrNotFound)

	// Missing and unknown tokens → 401.
	resp, body = fetchAsset(t, h, "/api/segments/segment_a1", "")
	assertAssetError(t, resp, body, http.StatusUnauthorized, protocol.ErrUnauthorized)
	resp, body = fetchAsset(t, h, "/api/segments/segment_a1", "not-a-token")
	assertAssetError(t, resp, body, http.StatusUnauthorized, protocol.ErrUnauthorized)

	// Query-string tokens are deliberately not accepted.
	resp, body = fetchAsset(t, h, "/api/segments/segment_a1?token="+a.ID, "")
	assertAssetError(t, resp, body, http.StatusUnauthorized, protocol.ErrUnauthorized)
}

func TestPreviewWindow(t *testing.T) {
	t.Parallel()
	h := Start(t, noSpecialty, nil)
	host, players := JoinConfigured(t, h, 2)
	a := players[0]
	host.StartGame()

	// a (art_enthusiast at clarity) earns 2 clarity thresholds → 0.5s window.
	PlayResource(t, host, players, 2, map[int]string{0: "hash-clarity"})
	host.Expect(protocol.PuzzleToHostReady)

	// Preview is assembly-only: during preparation → FORBIDDEN_PHASE.
	resp, body := fetchAsset(t, h, "/api/preview/full", a.ID)
	assertAssetError(t, resp, body, http.StatusForbidden, protocol.ErrForbiddenPhase)

	host.StartPuzzle()
	a.Expect(protocol.PuzzleToClientPhaseStart)

	// Inside the window: the full un-cropped source (96×96 test image).
	resp, body = fetchAsset(t, h, "/api/preview/full", a.ID)
	assertPNG(t, resp, body, 96)

	// The expiry broadcast closes the endpoint for the rest of the game.
	a.Expect(protocol.PuzzleToClientPreviewExpired)
	resp, body = fetchAsset(t, h, "/api/preview/full", a.ID)
	assertAssetError(t, resp, body, http.StatusForbidden, protocol.ErrForbiddenPreviewWindowClosed)
}

// With zero clarity thresholds no window ever opens: the phase-start
// broadcast says so and the endpoint stays 403 the whole phase.
func TestPreviewZeroThresholds(t *testing.T) {
	t.Parallel()
	h := Start(t, noSpecialty, nil)
	host, players := JoinConfigured(t, h, 2)
	a := players[0]
	host.StartGame()
	PlayResource(t, host, players, 2, nil) // nobody scans clarity
	host.Expect(protocol.PuzzleToHostReady)
	host.StartPuzzle()

	start := payloadAs[protocol.PuzzlePhaseStart](t, a.Expect(protocol.PuzzleToClientPhaseStart))
	if start.ClarityPreviewActive || start.ClarityPreviewDuration != 0 {
		t.Errorf("clarity fields = %+v", start)
	}

	resp, body := fetchAsset(t, h, "/api/preview/full", a.ID)
	assertAssetError(t, resp, body, http.StatusForbidden, protocol.ErrForbiddenPreviewWindowClosed)

	// No PREVIEW_EXPIRED is ever emitted in a zero-threshold game.
	a.ExpectNone(protocol.PuzzleToClientPreviewExpired, 400e6)
}
