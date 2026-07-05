// Package integration boots the real App (real config load, trivia load,
// tile generation, TCP listener, WebSocket upgrades) and drives it with real
// WebSocket clients. There are no mocks: phase pacing comes from tiny
// duration values in the test config, so full games play out in real time in
// a couple of seconds.
package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/app"
	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/config"
	"github.com/google/uuid"
)

// Harness is one running server instance on an ephemeral port.
type Harness struct {
	T        *testing.T
	BaseURL  string // http://127.0.0.1:port
	WSURL    string // ws://127.0.0.1:port
	HostUUID string
}

// fastConfig is the default test configuration: real production shape,
// miniature durations. Individual tests mutate it before boot.
func fastConfig() *config.Config {
	return &config.Config{
		MinPlayers:                      2,
		MaxPlayers:                      8,
		DifficultyMode:                  "medium",
		PuzzleImage:                     "test.png",
		ResourceGatheringRounds:         2,
		TriviaAnswerTime:                config.Seconds(150 * time.Millisecond),
		TriviaGraceTime:                 config.Seconds(50 * time.Millisecond),
		PuzzleBaseTime:                  config.Seconds(3 * time.Second),
		GridUpdateInterval:              config.Seconds(50 * time.Millisecond),
		BaseTokensPerCorrectAnswer:      20,
		RoleResourceMultiplier:          1.5,
		SpecialtyPointMultiplier:        2.0,
		AnchorTokenThreshold:            25,
		ChronosTokenThreshold:           20,
		GuideTokenThreshold:             15,
		ClarityTokenThreshold:           30,
		MaxThresholds:                   6,
		IndividualPuzzlePieces:          4,
		FragmentMoveCooldown:            config.Millis(60 * time.Millisecond),
		RecommendationTimeout:           config.Seconds(500 * time.Millisecond),
		MaxSpecialtiesPerPlayer:         2,
		ClarityBasePreviewTime:          config.Seconds(300 * time.Millisecond),
		TimeExtensionPerThreshold:       config.Seconds(500 * time.Millisecond),
		PreviewTimePerThreshold:         config.Seconds(100 * time.Millisecond),
		EasyTimeMultiplier:              1.2,
		MediumTimeMultiplier:            1.0,
		HardTimeMultiplier:              0.8,
		EasyThresholdMultiplier:         0.8,
		MediumThresholdMultiplier:       1.0,
		HardThresholdMultiplier:         1.2,
		EasySpecialtyProbability:        0.2,
		MediumSpecialtyProbability:      0.3,
		HardSpecialtyProbability:        0.4,
		PointsPerCorrectAnswer:          10,
		SpecialtyBonusPoints:            2,
		CompletionBonus:                 100,
		PointsPerSuccessfulMove:         5,
		PointsPerRecommendationSent:     3,
		PointsPerRecommendationAccepted: 8,
		StationHashes: config.StationHashes{
			Anchor:  "hash-anchor",
			Chronos: "hash-chronos",
			Guide:   "hash-guide",
			Clarity: "hash-clarity",
		},
	}
}

// Categories in the generated trivia fixture.
var fixtureCategories = []string{"alpha", "beta", "gamma"}

// Start boots a server. mutate (nillable) adjusts the config fixture;
// tweak (nillable) adjusts app options (protocol timing overrides).
func Start(t *testing.T, mutate func(*config.Config), tweak func(*app.Options)) *Harness {
	t.Helper()

	dir := t.TempDir()
	cfg := fastConfig()
	if mutate != nil {
		mutate(cfg)
	}

	sourcesDir := filepath.Join(dir, "puzzle-sources")
	triviaDir := filepath.Join(dir, "trivia")
	writeTestPNG(t, sourcesDir, cfg.PuzzleImage)
	writeTriviaFixture(t, triviaDir)

	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(dir, "game-config.json")
	if err := os.WriteFile(configPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}

	opts := app.Options{
		Environment:      "test",
		ConfigPath:       configPath,
		TriviaDir:        triviaDir,
		PuzzleSourcesDir: sourcesDir,
		HostUUID:         uuid.NewString(),
		ConnectDeadline:  5 * time.Second,
		DisconnectAfter:  30 * time.Second,
		Logger:           slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	if tweak != nil {
		tweak(&opts)
	}

	a, err := app.New(opts)
	if err != nil {
		t.Fatalf("app.New: %v", err)
	}

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = a.Serve(ctx, l)
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
		}
	})

	addr := l.Addr().String()
	return &Harness{
		T:        t,
		BaseURL:  "http://" + addr,
		WSURL:    "ws://" + addr,
		HostUUID: opts.HostUUID,
	}
}

// writeTestPNG writes a size×size image whose pixels encode coordinates
// (R=x, G=y), so tile-content assertions can verify exact crops.
func writeTestPNG(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	const size = 96
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 7, A: 0xff})
		}
	}
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

// writeTriviaFixture writes 3 categories × 3 difficulties × 2 questions in
// the raw Open Trivia DB export shape, HTML entities included, so the real
// loader path (parse + entity decode) is exercised.
func writeTriviaFixture(t *testing.T, dir string) {
	t.Helper()
	for _, cat := range fixtureCategories {
		if err := os.MkdirAll(filepath.Join(dir, cat), 0o755); err != nil {
			t.Fatal(err)
		}
		for _, diff := range []string{"easy", "medium", "hard"} {
			content := fmt.Sprintf(`{"response_code":0,"results":[
 {"type":"multiple","difficulty":%[1]q,"category":"X","question":"%[2]s %[1]s q0: 2+2?","correct_answer":"4","incorrect_answers":["3","5","6"]},
 {"type":"multiple","difficulty":%[1]q,"category":"X","question":"%[2]s %[1]s q1: &quot;easy&quot;?","correct_answer":"yes","incorrect_answers":["no","maybe","n/a"]}
]}`, diff, cat)
			path := filepath.Join(dir, cat, diff+".json")
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
}
