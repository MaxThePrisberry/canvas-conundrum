package config

import "time"

// Player Limits
var (
	MinPlayers int = 4  // Minimum players required (excluding host)
	MaxPlayers int = 64 // Maximum players allowed (excluding host)
)

// Phase Timing
var (
	ResourceGatheringRounds        int           = 5               // Number of resource gathering rounds
	ResourceGatheringRoundDuration int           = 60              // Seconds per resource gathering round
	TriviaAnswerTime               int           = 30              // Seconds to answer trivia question
	TriviaGraceTime                int           = 30              // Seconds of grace period after answer
	PuzzleBaseTime                 int           = 300             // Base seconds for puzzle assembly phase (5 minutes)
	PostGameDuration               int           = 300             // Seconds for post-game analytics display
	GridUpdateInterval             int           = 3               // Seconds between grid state updates for players
	GridUpdateIntervalDuration     time.Duration = 3 * time.Second // Duration version for timers
)

// Token Economics
var (
	BaseTokensPerCorrectAnswer int     = 20  // Base tokens awarded per correct answer
	RoleResourceMultiplier     float64 = 1.5 // Multiplier when at matching station
	SpecialtyPointMultiplier   float64 = 2.0 // Multiplier for specialty questions

	// Token Thresholds (tokens needed per threshold level)
	AnchorTokenThreshold  int = 25 // Tokens per anchor threshold
	ChronosTokenThreshold int = 20 // Tokens per chronos threshold
	GuideTokenThreshold   int = 15 // Tokens per guide threshold
	ClarityTokenThreshold int = 30 // Tokens per clarity threshold

	// Token threshold levels
	TokenThreshold1 int = 10 // First threshold level
	TokenThreshold2 int = 25 // Second threshold level
	TokenThreshold3 int = 45 // Third threshold level

	// Maximum Thresholds
	MaxThresholds int = 6 // Maximum threshold levels per token type
)

// Puzzle Configuration
var (
	IndividualPuzzlePieces  int = 16   // Pieces per individual puzzle
	FragmentMoveCooldown    int = 2500 // Milliseconds cooldown between fragment moves
	MaxSpecialtiesPerPlayer int = 1    // Maximum specialties a player can select
	ClarityBasePreviewTime  int = 3    // Base seconds for clarity preview

	// Anchor token effects
	PiecesPreSolvedPerThreshold int = 2  // Pieces pre-solved per anchor threshold
	MaxPreSolvedPieces          int = 12 // Maximum pieces that can be pre-solved

	// Chronos token effects
	TimeExtensionPerThreshold int = 20 // Seconds added per chronos threshold

	// Clarity token effects
	PreviewTimePerThreshold int = 1 // Seconds of preview per clarity threshold
)

// Difficulty Modifiers
var (
	// Time Multipliers
	EasyTimeMultiplier   float64 = 1.2
	MediumTimeMultiplier float64 = 1.0
	HardTimeMultiplier   float64 = 0.8

	// Threshold Multipliers (lower = easier to achieve)
	EasyThresholdMultiplier   float64 = 0.8
	MediumThresholdMultiplier float64 = 1.0
	HardThresholdMultiplier   float64 = 1.2

	// Specialty Question Probability
	EasySpecialtyProbability   float64 = 0.2
	MediumSpecialtyProbability float64 = 0.3
	HardSpecialtyProbability   float64 = 0.4
)

// Scoring Algorithm
var (
	PointsPerCorrectAnswer          int = 10
	SpecialtyBonusPoints            int = 2
	CompletionBonus                 int = 100
	MaxSpeedBonus                   int = 300
	PointsPerSuccessfulMove         int = 5
	PointsPerRecommendationSent     int = 3
	PointsPerRecommendationAccepted int = 8
)

// Specialty Question Frequency
// These determine how often specialty questions appear by difficulty
var (
	SpecialtyQFreqEasy   float64 = 0.2 // 20% chance in easy mode
	SpecialtyQFreqMedium float64 = 0.3 // 30% chance in medium mode
	SpecialtyQFreqHard   float64 = 0.4 // 40% chance in hard mode
)

// Grid Size Calculation
func GetGridSizeForPlayerCount(playerCount int) int {
	switch {
	case playerCount <= 9:
		return 3
	case playerCount <= 16:
		return 4
	case playerCount <= 25:
		return 5
	case playerCount <= 36:
		return 6
	case playerCount <= 49:
		return 7
	default:
		return 8
	}
}
