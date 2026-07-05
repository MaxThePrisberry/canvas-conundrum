package config

import (
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"path/filepath"
)

// MaxSupportedPlayers is the top of the grid-scaling table in game-design.md
// § Dynamic Grid System.
const MaxSupportedPlayers = 64

// Validate performs the startup checks required by game-design.md: the
// server must refuse to start on any failure. puzzleSourcesDir is where
// cfg.PuzzleImage must exist and decode.
func (c *Config) Validate(puzzleSourcesDir string) error {
	var errs []error
	fail := func(format string, args ...any) {
		errs = append(errs, fmt.Errorf(format, args...))
	}

	if c.MinPlayers < 1 {
		fail("minPlayers must be >= 1, got %d", c.MinPlayers)
	}
	if c.MaxPlayers < c.MinPlayers {
		fail("maxPlayers (%d) must be >= minPlayers (%d)", c.MaxPlayers, c.MinPlayers)
	}
	if c.MaxPlayers > MaxSupportedPlayers {
		fail("maxPlayers must be <= %d, got %d", MaxSupportedPlayers, c.MaxPlayers)
	}

	switch c.DifficultyMode {
	case "easy", "medium", "hard":
	default:
		fail("difficultyMode must be easy|medium|hard, got %q", c.DifficultyMode)
	}

	if !isPerfectSquare(c.IndividualPuzzlePieces) {
		fail("individualPuzzlePieces must be a perfect square, got %d", c.IndividualPuzzlePieces)
	}

	if c.ResourceGatheringRounds < 1 {
		fail("resourceGatheringRounds must be >= 1, got %d", c.ResourceGatheringRounds)
	}
	for name, d := range map[string]Seconds{
		"triviaAnswerTime":   c.TriviaAnswerTime,
		"triviaGraceTime":    c.TriviaGraceTime,
		"puzzleBaseTime":     c.PuzzleBaseTime,
		"gridUpdateInterval": c.GridUpdateInterval,
	} {
		if d.Duration() <= 0 {
			fail("%s must be > 0", name)
		}
	}
	if c.RecommendationTimeout.Duration() <= 0 {
		fail("recommendationTimeout must be > 0")
	}
	if c.FragmentMoveCooldown.Duration() < 0 {
		fail("fragmentMoveCooldown must be >= 0")
	}

	if c.BaseTokensPerCorrectAnswer <= 0 {
		fail("baseTokensPerCorrectAnswer must be > 0")
	}
	for name, m := range map[string]float64{
		"roleResourceMultiplier":    c.RoleResourceMultiplier,
		"specialtyPointMultiplier":  c.SpecialtyPointMultiplier,
		"easyTimeMultiplier":        c.EasyTimeMultiplier,
		"mediumTimeMultiplier":      c.MediumTimeMultiplier,
		"hardTimeMultiplier":        c.HardTimeMultiplier,
		"easyThresholdMultiplier":   c.EasyThresholdMultiplier,
		"mediumThresholdMultiplier": c.MediumThresholdMultiplier,
		"hardThresholdMultiplier":   c.HardThresholdMultiplier,
	} {
		if m <= 0 {
			fail("%s must be > 0", name)
		}
	}
	for name, p := range map[string]float64{
		"easySpecialtyProbability":   c.EasySpecialtyProbability,
		"mediumSpecialtyProbability": c.MediumSpecialtyProbability,
		"hardSpecialtyProbability":   c.HardSpecialtyProbability,
	} {
		if p < 0 || p > 1 {
			fail("%s must be in [0,1], got %v", name, p)
		}
	}

	for name, t := range map[string]int{
		"anchorTokenThreshold":  c.AnchorTokenThreshold,
		"chronosTokenThreshold": c.ChronosTokenThreshold,
		"guideTokenThreshold":   c.GuideTokenThreshold,
		"clarityTokenThreshold": c.ClarityTokenThreshold,
	} {
		if t <= 0 {
			fail("%s must be > 0", name)
		}
	}
	if c.MaxThresholds < 1 {
		fail("maxThresholds must be >= 1, got %d", c.MaxThresholds)
	}
	if c.MaxSpecialtiesPerPlayer < 1 {
		fail("maxSpecialtiesPerPlayer must be >= 1, got %d", c.MaxSpecialtiesPerPlayer)
	}

	if err := c.validateStationHashes(); err != nil {
		errs = append(errs, err)
	}
	if err := c.validatePuzzleImage(puzzleSourcesDir); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

func (c *Config) validateStationHashes() error {
	hashes := map[string]string{
		"anchor":  c.StationHashes.Anchor,
		"chronos": c.StationHashes.Chronos,
		"guide":   c.StationHashes.Guide,
		"clarity": c.StationHashes.Clarity,
	}
	seen := map[string]string{}
	for station, h := range hashes {
		if h == "" {
			return fmt.Errorf("stationHashes.%s must not be empty", station)
		}
		if other, dup := seen[h]; dup {
			return fmt.Errorf("stationHashes.%s duplicates stationHashes.%s", station, other)
		}
		seen[h] = station
	}
	return nil
}

func (c *Config) validatePuzzleImage(dir string) error {
	if c.PuzzleImage == "" {
		return errors.New("puzzleImage must not be empty")
	}
	path := filepath.Join(dir, c.PuzzleImage)
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("puzzle image %s: %w", path, err)
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return fmt.Errorf("puzzle image %s does not decode: %w", path, err)
	}
	b := img.Bounds()
	if b.Dx() < 1 || b.Dy() < 1 {
		return fmt.Errorf("puzzle image %s has empty bounds", path)
	}
	return nil
}

func isPerfectSquare(n int) bool {
	if n < 1 {
		return false
	}
	r := int(math.Sqrt(float64(n)))
	return r*r == n || (r+1)*(r+1) == n
}
