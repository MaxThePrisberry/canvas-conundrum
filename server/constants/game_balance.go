package constants

import "time"

// Player Limits
const (
	MinPlayers = 4  // Minimum players required (excluding host)
	MaxPlayers = 64 // Maximum players allowed (excluding host)
)

// Phase Timing
const (
	ResourceGatheringRounds        = 5               // Number of resource gathering rounds
	ResourceGatheringRoundDuration = 60              // Seconds per resource gathering round
	TriviaAnswerTime               = 30              // Seconds to answer trivia question
	TriviaGraceTime                = 30              // Seconds of grace period after answer
	PuzzleBaseTime                 = 300             // Base seconds for puzzle assembly phase (5 minutes)
	PostGameDuration               = 300             // Seconds for post-game analytics display
	GridUpdateInterval             = 3               // Seconds between grid state updates for players
	GridUpdateIntervalDuration     = 3 * time.Second // Duration version for timers
)

// Token Economics
const (
	BaseTokensPerCorrectAnswer = 20  // Base tokens awarded per correct answer
	RoleResourceMultiplier     = 1.5 // Multiplier when at matching station
	SpecialtyPointMultiplier   = 2.0 // Multiplier for specialty questions

	// Token Thresholds (tokens needed per threshold level)
	AnchorTokenThreshold  = 25 // Tokens per anchor threshold
	ChronosTokenThreshold = 20 // Tokens per chronos threshold
	GuideTokenThreshold   = 15 // Tokens per guide threshold
	ClarityTokenThreshold = 30 // Tokens per clarity threshold

	// Token threshold levels
	TokenThreshold1 = 10 // First threshold level
	TokenThreshold2 = 25 // Second threshold level
	TokenThreshold3 = 45 // Third threshold level

	// Maximum Thresholds
	MaxThresholds = 6 // Maximum threshold levels per token type
)

// Puzzle Configuration
const (
	IndividualPuzzlePieces  = 16   // Pieces per individual puzzle
	FragmentMoveCooldown    = 2500 // Milliseconds cooldown between fragment moves
	MaxSpecialtiesPerPlayer = 1    // Maximum specialties a player can select
	ClarityBasePreviewTime  = 3    // Base seconds for clarity preview

	// Anchor token effects
	PiecesPreSolvedPerThreshold = 2  // Pieces pre-solved per anchor threshold
	MaxPreSolvedPieces          = 12 // Maximum pieces that can be pre-solved

	// Chronos token effects
	TimeExtensionPerThreshold = 20 // Seconds added per chronos threshold

	// Clarity token effects
	PreviewTimePerThreshold = 1 // Seconds of preview per clarity threshold
)

// Difficulty Modifiers
const (
	// Time Multipliers
	EasyTimeMultiplier   = 1.2
	MediumTimeMultiplier = 1.0
	HardTimeMultiplier   = 0.8

	// Threshold Multipliers (lower = easier to achieve)
	EasyThresholdMultiplier   = 0.8
	MediumThresholdMultiplier = 1.0
	HardThresholdMultiplier   = 1.2

	// Specialty Question Probability
	EasySpecialtyProbability   = 0.2
	MediumSpecialtyProbability = 0.3
	HardSpecialtyProbability   = 0.4
)

// Scoring Algorithm
const (
	PointsPerCorrectAnswer          = 10
	SpecialtyBonusPoints            = 2
	CompletionBonus                 = 100
	MaxSpeedBonus                   = 300
	PointsPerSuccessfulMove         = 5
	PointsPerRecommendationSent     = 3
	PointsPerRecommendationAccepted = 8
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

// Get total fragments for a grid size
func GetTotalFragments(gridSize int) int {
	return gridSize * gridSize
}
