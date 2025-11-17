package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// GameConfig holds all game configuration values
type GameConfig struct {
	// Player Limits
	MinPlayers int `json:"minPlayers"`
	MaxPlayers int `json:"maxPlayers"`

	// Phase Timing
	ResourceGatheringRounds        int           `json:"resourceGatheringRounds"`
	ResourceGatheringRoundDuration int           `json:"resourceGatheringRoundDuration"`
	TriviaAnswerTime               int           `json:"triviaAnswerTime"`
	TriviaGraceTime                int           `json:"triviaGraceTime"`
	PuzzleBaseTime                 int           `json:"puzzleBaseTime"`
	PostGameDuration               int           `json:"postGameDuration"`
	GridUpdateInterval             int           `json:"gridUpdateInterval"`
	GridUpdateIntervalDuration     time.Duration `json:"-"` // Computed from GridUpdateInterval

	// Token Economics
	BaseTokensPerCorrectAnswer int     `json:"baseTokensPerCorrectAnswer"`
	RoleResourceMultiplier     float64 `json:"roleResourceMultiplier"`
	SpecialtyPointMultiplier   float64 `json:"specialtyPointMultiplier"`
	AnchorTokenThreshold       int     `json:"anchorTokenThreshold"`
	ChronosTokenThreshold      int     `json:"chronosTokenThreshold"`
	GuideTokenThreshold        int     `json:"guideTokenThreshold"`
	ClarityTokenThreshold      int     `json:"clarityTokenThreshold"`
	TokenThreshold1            int     `json:"tokenThreshold1"`
	TokenThreshold2            int     `json:"tokenThreshold2"`
	TokenThreshold3            int     `json:"tokenThreshold3"`
	MaxThresholds              int     `json:"maxThresholds"`

	// Puzzle Configuration
	IndividualPuzzlePieces      int `json:"individualPuzzlePieces"`
	FragmentMoveCooldown        int `json:"fragmentMoveCooldown"`
	MaxSpecialtiesPerPlayer     int `json:"maxSpecialtiesPerPlayer"`
	ClarityBasePreviewTime      int `json:"clarityBasePreviewTime"`
	PiecesPreSolvedPerThreshold int `json:"piecesPreSolvedPerThreshold"`
	MaxPreSolvedPieces          int `json:"maxPreSolvedPieces"`
	TimeExtensionPerThreshold   int `json:"timeExtensionPerThreshold"`
	PreviewTimePerThreshold     int `json:"previewTimePerThreshold"`

	// Difficulty Modifiers
	EasyTimeMultiplier         float64 `json:"easyTimeMultiplier"`
	MediumTimeMultiplier       float64 `json:"mediumTimeMultiplier"`
	HardTimeMultiplier         float64 `json:"hardTimeMultiplier"`
	EasyThresholdMultiplier    float64 `json:"easyThresholdMultiplier"`
	MediumThresholdMultiplier  float64 `json:"mediumThresholdMultiplier"`
	HardThresholdMultiplier    float64 `json:"hardThresholdMultiplier"`
	EasySpecialtyProbability   float64 `json:"easySpecialtyProbability"`
	MediumSpecialtyProbability float64 `json:"mediumSpecialtyProbability"`
	HardSpecialtyProbability   float64 `json:"hardSpecialtyProbability"`

	// Scoring Algorithm
	PointsPerCorrectAnswer          int `json:"pointsPerCorrectAnswer"`
	SpecialtyBonusPoints            int `json:"specialtyBonusPoints"`
	CompletionBonus                 int `json:"completionBonus"`
	MaxSpeedBonus                   int `json:"maxSpeedBonus"`
	PointsPerSuccessfulMove         int `json:"pointsPerSuccessfulMove"`
	PointsPerRecommendationSent     int `json:"pointsPerRecommendationSent"`
	PointsPerRecommendationAccepted int `json:"pointsPerRecommendationAccepted"`

	// Specialty Question Frequency
	SpecialtyQFreqEasy   float64 `json:"specialtyQFreqEasy"`
	SpecialtyQFreqMedium float64 `json:"specialtyQFreqMedium"`
	SpecialtyQFreqHard   float64 `json:"specialtyQFreqHard"`
}

// Global config instance
var currentConfig *GameConfig

// Initialize with default values
func init() {
	currentConfig = getDefaultConfig()

	// Check for custom config directory
	if configDir := os.Getenv("CANVAS_CONFIG_DIR"); configDir != "" {
		if err := LoadConfigFromDirectory(configDir); err != nil {
			fmt.Printf("Warning: Failed to load config from %s: %v\n", configDir, err)
		}
	}

	// Update config variables after loading
	UpdateConfigVariables()
}

// getDefaultConfig returns the default configuration values
func getDefaultConfig() *GameConfig {
	return &GameConfig{
		// Player Limits
		MinPlayers: 4,
		MaxPlayers: 64,

		// Phase Timing
		ResourceGatheringRounds:        5,
		ResourceGatheringRoundDuration: 60,
		TriviaAnswerTime:               30,
		TriviaGraceTime:                30,
		PuzzleBaseTime:                 300,
		PostGameDuration:               300,
		GridUpdateInterval:             3,
		GridUpdateIntervalDuration:     3 * time.Second,

		// Token Economics
		BaseTokensPerCorrectAnswer: 20,
		RoleResourceMultiplier:     1.5,
		SpecialtyPointMultiplier:   2.0,
		AnchorTokenThreshold:       25,
		ChronosTokenThreshold:      20,
		GuideTokenThreshold:        15,
		ClarityTokenThreshold:      30,
		TokenThreshold1:            10,
		TokenThreshold2:            25,
		TokenThreshold3:            45,
		MaxThresholds:              6,

		// Puzzle Configuration
		IndividualPuzzlePieces:      16,
		FragmentMoveCooldown:        2500,
		MaxSpecialtiesPerPlayer:     1,
		ClarityBasePreviewTime:      3,
		PiecesPreSolvedPerThreshold: 2,
		MaxPreSolvedPieces:          12,
		TimeExtensionPerThreshold:   20,
		PreviewTimePerThreshold:     1,

		// Difficulty Modifiers
		EasyTimeMultiplier:         1.2,
		MediumTimeMultiplier:       1.0,
		HardTimeMultiplier:         0.8,
		EasyThresholdMultiplier:    0.8,
		MediumThresholdMultiplier:  1.0,
		HardThresholdMultiplier:    1.2,
		EasySpecialtyProbability:   0.2,
		MediumSpecialtyProbability: 0.3,
		HardSpecialtyProbability:   0.4,

		// Scoring Algorithm
		PointsPerCorrectAnswer:          10,
		SpecialtyBonusPoints:            2,
		CompletionBonus:                 100,
		MaxSpeedBonus:                   300,
		PointsPerSuccessfulMove:         5,
		PointsPerRecommendationSent:     3,
		PointsPerRecommendationAccepted: 8,

		// Specialty Question Frequency
		SpecialtyQFreqEasy:   0.2,
		SpecialtyQFreqMedium: 0.3,
		SpecialtyQFreqHard:   0.4,
	}
}

// LoadConfigFromDirectory loads configuration from JSON files in the specified directory
func LoadConfigFromDirectory(configDir string) error {
	configFile := filepath.Join(configDir, "game.json")

	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return fmt.Errorf("config file not found: %s", configFile)
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file: %v", err)
	}

	var config GameConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config file: %v", err)
	}

	// Compute derived values
	config.GridUpdateIntervalDuration = time.Duration(config.GridUpdateInterval) * time.Second

	currentConfig = &config

	// Update config variables
	UpdateConfigVariables()

	return nil
}

// GetConfig returns the current configuration
func GetConfig() *GameConfig {
	return currentConfig
}

// UpdateConfigVariables updates all config variables from the current config
func UpdateConfigVariables() {
	config := GetConfig()

	// Player Limits
	MinPlayers = config.MinPlayers
	MaxPlayers = config.MaxPlayers

	// Phase Timing
	ResourceGatheringRounds = config.ResourceGatheringRounds
	ResourceGatheringRoundDuration = config.ResourceGatheringRoundDuration
	TriviaAnswerTime = config.TriviaAnswerTime
	TriviaGraceTime = config.TriviaGraceTime
	PuzzleBaseTime = config.PuzzleBaseTime
	PostGameDuration = config.PostGameDuration
	GridUpdateInterval = config.GridUpdateInterval
	GridUpdateIntervalDuration = config.GridUpdateIntervalDuration

	// Token Economics
	BaseTokensPerCorrectAnswer = config.BaseTokensPerCorrectAnswer
	RoleResourceMultiplier = config.RoleResourceMultiplier
	SpecialtyPointMultiplier = config.SpecialtyPointMultiplier
	AnchorTokenThreshold = config.AnchorTokenThreshold
	ChronosTokenThreshold = config.ChronosTokenThreshold
	GuideTokenThreshold = config.GuideTokenThreshold
	ClarityTokenThreshold = config.ClarityTokenThreshold
	TokenThreshold1 = config.TokenThreshold1
	TokenThreshold2 = config.TokenThreshold2
	TokenThreshold3 = config.TokenThreshold3
	MaxThresholds = config.MaxThresholds

	// Puzzle Configuration
	IndividualPuzzlePieces = config.IndividualPuzzlePieces
	FragmentMoveCooldown = config.FragmentMoveCooldown
	MaxSpecialtiesPerPlayer = config.MaxSpecialtiesPerPlayer
	ClarityBasePreviewTime = config.ClarityBasePreviewTime
	PiecesPreSolvedPerThreshold = config.PiecesPreSolvedPerThreshold
	MaxPreSolvedPieces = config.MaxPreSolvedPieces
	TimeExtensionPerThreshold = config.TimeExtensionPerThreshold
	PreviewTimePerThreshold = config.PreviewTimePerThreshold

	// Difficulty Modifiers
	EasyTimeMultiplier = config.EasyTimeMultiplier
	MediumTimeMultiplier = config.MediumTimeMultiplier
	HardTimeMultiplier = config.HardTimeMultiplier
	EasyThresholdMultiplier = config.EasyThresholdMultiplier
	MediumThresholdMultiplier = config.MediumThresholdMultiplier
	HardThresholdMultiplier = config.HardThresholdMultiplier
	EasySpecialtyProbability = config.EasySpecialtyProbability
	MediumSpecialtyProbability = config.MediumSpecialtyProbability
	HardSpecialtyProbability = config.HardSpecialtyProbability

	// Scoring Algorithm
	PointsPerCorrectAnswer = config.PointsPerCorrectAnswer
	SpecialtyBonusPoints = config.SpecialtyBonusPoints
	CompletionBonus = config.CompletionBonus
	MaxSpeedBonus = config.MaxSpeedBonus
	PointsPerSuccessfulMove = config.PointsPerSuccessfulMove
	PointsPerRecommendationSent = config.PointsPerRecommendationSent
	PointsPerRecommendationAccepted = config.PointsPerRecommendationAccepted

	// Specialty Question Frequency
	SpecialtyQFreqEasy = config.SpecialtyQFreqEasy
	SpecialtyQFreqMedium = config.SpecialtyQFreqMedium
	SpecialtyQFreqHard = config.SpecialtyQFreqHard
}
