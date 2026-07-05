package config

import (
	"encoding/json"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestLoadRealConfig parses the repo's committed game-config.json and
// validates it against the committed puzzle sources, proving the production
// artifacts and this package agree.
func TestLoadRealConfig(t *testing.T) {
	path := filepath.Join("..", "..", "..", "game-config.json")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("repo config not present: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.MinPlayers != 4 || cfg.MaxPlayers != 64 {
		t.Errorf("player bounds = %d..%d, want 4..64", cfg.MinPlayers, cfg.MaxPlayers)
	}
	if got := cfg.TriviaAnswerTime.Duration(); got != 30*time.Second {
		t.Errorf("triviaAnswerTime = %v, want 30s", got)
	}
	if got := cfg.FragmentMoveCooldown.Duration(); got != 2500*time.Millisecond {
		t.Errorf("fragmentMoveCooldown = %v, want 2.5s", got)
	}
	if cfg.TimeMultiplier() != 1.0 || cfg.SpecialtyProbability() != 0.3 {
		t.Errorf("medium difficulty accessors wrong: time=%v prob=%v",
			cfg.TimeMultiplier(), cfg.SpecialtyProbability())
	}
	if cfg.TokenThreshold("clarity") != 30 {
		t.Errorf("clarity threshold = %d, want 30", cfg.TokenThreshold("clarity"))
	}

	sources := filepath.Join("..", "..", "..", "assets", "puzzle-sources")
	if err := cfg.Validate(sources); err != nil {
		t.Errorf("committed config fails validation: %v", err)
	}
}

func TestFractionalSecondsParse(t *testing.T) {
	var s Seconds
	if err := json.Unmarshal([]byte("0.2"), &s); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if s.Duration() != 200*time.Millisecond {
		t.Errorf("0.2s parsed as %v, want 200ms", s.Duration())
	}

	var m Millis
	if err := json.Unmarshal([]byte("2500"), &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if m.Duration() != 2500*time.Millisecond {
		t.Errorf("2500ms parsed as %v", m.Duration())
	}
}

func TestLoadRejectsUnknownKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"minPlayers": 4, "typoedKey": 1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Error("Load accepted a config with an unknown key")
	}
}

// validConfig returns a fully valid config plus a puzzle-sources dir
// containing a decodable image, for mutation in validation tests.
func validConfig(t *testing.T) (*Config, string) {
	t.Helper()

	dir := t.TempDir()
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for i := range img.Pix {
		img.Pix[i] = 0xff
	}
	f, err := os.Create(filepath.Join(dir, "test.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	f.Close()

	return &Config{
		MinPlayers:                      2,
		MaxPlayers:                      8,
		DifficultyMode:                  "medium",
		PuzzleImage:                     "test.png",
		ResourceGatheringRounds:         2,
		TriviaAnswerTime:                Seconds(time.Second),
		TriviaGraceTime:                 Seconds(time.Second),
		PuzzleBaseTime:                  Seconds(10 * time.Second),
		GridUpdateInterval:              Seconds(time.Second),
		BaseTokensPerCorrectAnswer:      20,
		RoleResourceMultiplier:          1.5,
		SpecialtyPointMultiplier:        2.0,
		AnchorTokenThreshold:            25,
		ChronosTokenThreshold:           20,
		GuideTokenThreshold:             15,
		ClarityTokenThreshold:           30,
		MaxThresholds:                   6,
		IndividualPuzzlePieces:          16,
		FragmentMoveCooldown:            Millis(100 * time.Millisecond),
		RecommendationTimeout:           Seconds(time.Second),
		MaxSpecialtiesPerPlayer:         1,
		ClarityBasePreviewTime:          Seconds(time.Second),
		TimeExtensionPerThreshold:       Seconds(time.Second),
		PreviewTimePerThreshold:         Seconds(time.Second),
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
		StationHashes: StationHashes{
			Anchor: "a", Chronos: "c", Guide: "g", Clarity: "l",
		},
	}, dir
}

func TestValidateAcceptsValidConfig(t *testing.T) {
	cfg, dir := validConfig(t)
	if err := cfg.Validate(dir); err != nil {
		t.Errorf("valid config rejected: %v", err)
	}
}

func TestValidateFailures(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*Config)
		wantSub string
	}{
		{"pieces not square", func(c *Config) { c.IndividualPuzzlePieces = 15 }, "perfect square"},
		{"min above max", func(c *Config) { c.MinPlayers = 9 }, "minPlayers"},
		{"players above table", func(c *Config) { c.MaxPlayers = 65 }, "maxPlayers"},
		{"bad difficulty", func(c *Config) { c.DifficultyMode = "extreme" }, "difficultyMode"},
		{"empty station hash", func(c *Config) { c.StationHashes.Guide = "" }, "stationHashes.guide"},
		{"duplicate station hashes", func(c *Config) { c.StationHashes.Guide = c.StationHashes.Anchor }, "duplicates"},
		{"missing image", func(c *Config) { c.PuzzleImage = "nope.png" }, "puzzle image"},
		{"probability above 1", func(c *Config) { c.MediumSpecialtyProbability = 1.5 }, "SpecialtyProbability"},
		{"zero answer time", func(c *Config) { c.TriviaAnswerTime = 0 }, "triviaAnswerTime"},
		{"zero max thresholds", func(c *Config) { c.MaxThresholds = 0 }, "maxThresholds"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, dir := validConfig(t)
			tc.mutate(cfg)
			err := cfg.Validate(dir)
			if err == nil {
				t.Fatal("expected validation error, got nil")
			}
			if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tc.wantSub)) {
				t.Errorf("error %q does not mention %q", err, tc.wantSub)
			}
		})
	}
}

func TestValidateRejectsUndecodableImage(t *testing.T) {
	cfg, dir := validConfig(t)
	if err := os.WriteFile(filepath.Join(dir, "bad.png"), []byte("not a png"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg.PuzzleImage = "bad.png"
	if err := cfg.Validate(dir); err == nil {
		t.Error("undecodable image accepted")
	}
}
