// Package config loads and validates game-config.json (see game-design.md
// § Configuration Reference).
package config

import (
	"encoding/json"
	"fmt"
	"time"
)

// Seconds is a duration configured as a JSON number of seconds. Production
// config uses integers; fractional values (e.g. 0.2) are supported so tests
// can play through phases quickly with real timers.
type Seconds time.Duration

func (s *Seconds) UnmarshalJSON(b []byte) error {
	var f float64
	if err := json.Unmarshal(b, &f); err != nil {
		return fmt.Errorf("duration must be a number of seconds: %w", err)
	}
	*s = Seconds(time.Duration(f * float64(time.Second)))
	return nil
}

func (s Seconds) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(s).Seconds())
}

func (s Seconds) Duration() time.Duration { return time.Duration(s) }

// Sec returns the value as float seconds, the unit used in wire payloads.
func (s Seconds) Sec() float64 { return time.Duration(s).Seconds() }

// Millis is a duration configured as a JSON number of milliseconds
// (fragmentMoveCooldown).
type Millis time.Duration

func (m *Millis) UnmarshalJSON(b []byte) error {
	var f float64
	if err := json.Unmarshal(b, &f); err != nil {
		return fmt.Errorf("duration must be a number of milliseconds: %w", err)
	}
	*m = Millis(time.Duration(f * float64(time.Millisecond)))
	return nil
}

func (m Millis) MarshalJSON() ([]byte, error) {
	return json.Marshal(float64(time.Duration(m) / time.Millisecond))
}

func (m Millis) Duration() time.Duration { return time.Duration(m) }

// StationHashes are the server-side QR payloads for the four stations. They
// are matched against RESOURCE_TO_SERVER_LOCATION_VERIFIED and must never be
// sent to clients.
type StationHashes struct {
	Anchor  string `json:"anchor"`
	Chronos string `json:"chronos"`
	Guide   string `json:"guide"`
	Clarity string `json:"clarity"`
}

// Config mirrors game-config.json key-for-key.
type Config struct {
	MinPlayers     int    `json:"minPlayers"`
	MaxPlayers     int    `json:"maxPlayers"`
	DifficultyMode string `json:"difficultyMode"`
	PuzzleImage    string `json:"puzzleImage"`

	ResourceGatheringRounds int     `json:"resourceGatheringRounds"`
	TriviaAnswerTime        Seconds `json:"triviaAnswerTime"`
	TriviaGraceTime         Seconds `json:"triviaGraceTime"`
	PuzzleBaseTime          Seconds `json:"puzzleBaseTime"`
	GridUpdateInterval      Seconds `json:"gridUpdateInterval"`

	BaseTokensPerCorrectAnswer int     `json:"baseTokensPerCorrectAnswer"`
	RoleResourceMultiplier     float64 `json:"roleResourceMultiplier"`
	SpecialtyPointMultiplier   float64 `json:"specialtyPointMultiplier"`

	AnchorTokenThreshold  int `json:"anchorTokenThreshold"`
	ChronosTokenThreshold int `json:"chronosTokenThreshold"`
	GuideTokenThreshold   int `json:"guideTokenThreshold"`
	ClarityTokenThreshold int `json:"clarityTokenThreshold"`
	MaxThresholds         int `json:"maxThresholds"`

	IndividualPuzzlePieces  int     `json:"individualPuzzlePieces"`
	FragmentMoveCooldown    Millis  `json:"fragmentMoveCooldown"`
	RecommendationTimeout   Seconds `json:"recommendationTimeout"`
	MaxSpecialtiesPerPlayer int     `json:"maxSpecialtiesPerPlayer"`

	ClarityBasePreviewTime    Seconds `json:"clarityBasePreviewTime"`
	TimeExtensionPerThreshold Seconds `json:"timeExtensionPerThreshold"`
	PreviewTimePerThreshold   Seconds `json:"previewTimePerThreshold"`

	EasyTimeMultiplier   float64 `json:"easyTimeMultiplier"`
	MediumTimeMultiplier float64 `json:"mediumTimeMultiplier"`
	HardTimeMultiplier   float64 `json:"hardTimeMultiplier"`

	EasyThresholdMultiplier   float64 `json:"easyThresholdMultiplier"`
	MediumThresholdMultiplier float64 `json:"mediumThresholdMultiplier"`
	HardThresholdMultiplier   float64 `json:"hardThresholdMultiplier"`

	EasySpecialtyProbability   float64 `json:"easySpecialtyProbability"`
	MediumSpecialtyProbability float64 `json:"mediumSpecialtyProbability"`
	HardSpecialtyProbability   float64 `json:"hardSpecialtyProbability"`

	PointsPerCorrectAnswer          int `json:"pointsPerCorrectAnswer"`
	SpecialtyBonusPoints            int `json:"specialtyBonusPoints"`
	CompletionBonus                 int `json:"completionBonus"`
	PointsPerSuccessfulMove         int `json:"pointsPerSuccessfulMove"`
	PointsPerRecommendationSent     int `json:"pointsPerRecommendationSent"`
	PointsPerRecommendationAccepted int `json:"pointsPerRecommendationAccepted"`

	StationHashes StationHashes `json:"stationHashes"`
}

// TimeMultiplier returns the puzzle-timer multiplier for the configured
// difficulty mode.
func (c *Config) TimeMultiplier() float64 {
	switch c.DifficultyMode {
	case "easy":
		return c.EasyTimeMultiplier
	case "hard":
		return c.HardTimeMultiplier
	default:
		return c.MediumTimeMultiplier
	}
}

// ThresholdMultiplier returns the token-threshold multiplier for the
// configured difficulty mode.
func (c *Config) ThresholdMultiplier() float64 {
	switch c.DifficultyMode {
	case "easy":
		return c.EasyThresholdMultiplier
	case "hard":
		return c.HardThresholdMultiplier
	default:
		return c.MediumThresholdMultiplier
	}
}

// SpecialtyProbability returns the per-player per-round specialty-question
// probability for the configured difficulty mode.
func (c *Config) SpecialtyProbability() float64 {
	switch c.DifficultyMode {
	case "easy":
		return c.EasySpecialtyProbability
	case "hard":
		return c.HardSpecialtyProbability
	default:
		return c.MediumSpecialtyProbability
	}
}

// TokenThreshold returns the base token threshold for one of the four
// station/token types ("anchor", "chronos", "guide", "clarity").
func (c *Config) TokenThreshold(tokenType string) int {
	switch tokenType {
	case "anchor":
		return c.AnchorTokenThreshold
	case "chronos":
		return c.ChronosTokenThreshold
	case "guide":
		return c.GuideTokenThreshold
	case "clarity":
		return c.ClarityTokenThreshold
	}
	return 0
}
