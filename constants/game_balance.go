// Canvas Conundrum - Game Balance Constants
// Package: constants
// Version: 1.0.0
// Last Updated: 2025-05-26

package constants
  
// Character Role Settings
const (
	// RoleResourceMultiplier - Multiplier applied to resource collection for character role bonuses
	// Tuning: Adjust to balance role advantages. Higher values make role selection more impactful.
	RoleResourceMultiplier float64 = 1.5
)

// Trivia Mechanics
const (
	// TriviaQuestionInterval - Time interval between trivia questions during resource gathering phase (seconds)
	// Tuning: Shorter intervals increase game pace but may overwhelm players. Longer intervals may reduce engagement.
	TriviaQuestionInterval int = 60

	// SpecialtyPointMultiplier - Point multiplier for correctly answering specialty trivia questions
	// Tuning: Higher values incentivize specialty focus. Lower values encourage broader knowledge.
	SpecialtyPointMultiplier float64 = 2.0
)

// Resource Token Settings
const (
	// AnchorTokenThresholds - Number of anchor token thresholds available
	// Tuning: More thresholds = easier puzzle assembly. Fewer thresholds = more challenging individual puzzles.
	AnchorTokenThresholds int = 5

	// ChronosTokenThresholds - Number of chronos token thresholds available
	// Tuning: Affects maximum possible time extension for puzzle assembly phase.
	ChronosTokenThresholds int = 5

	// ChronosTimeBonus - Time bonus added per chronos token threshold reached (seconds)
	// Tuning: Balance between rewarding resource gathering and maintaining time pressure.
	ChronosTimeBonus int = 20

	// GuideTokenThresholds - Number of guide token thresholds available
	// Tuning: More thresholds provide better piece placement hints. Adjust based on desired difficulty.
	GuideTokenThresholds int = 5

	// ClarityTokenThresholds - Number of clarity token thresholds available
	// Tuning: Affects maximum initial image display time. Higher thresholds = longer preview.
	ClarityTokenThresholds int = 5

	// ClarityTimeBonus - Additional image display time per clarity token threshold (seconds)
	// Tuning: Longer display times make puzzle assembly easier. Adjust for difficulty balance.
	ClarityTimeBonus int = 1
)

// Puzzle Mechanics
const (
	// FragmentMovementCooldown - Cooldown period after moving a puzzle fragment before next move is allowed (milliseconds)
	// Tuning: Prevents race conditions and excessive rapid movements. Adjust for responsiveness vs stability.
	FragmentMovementCooldown int = 1000

	// IndividualPuzzlePieces - Number of pieces in each individual player puzzle fragment
	// Tuning: More pieces = longer individual solving time. Fewer pieces = faster progression to collaboration phase.
	IndividualPuzzlePieces int = 16
)

// Grid Scaling Configuration
type GridBreakpoint struct {
	MinPlayers     int
	MaxPlayers     int
	GridSize       int
	TotalFragments int
}

// GridSizeBreakpoints - Player count breakpoints for determining puzzle grid size
// Tuning: Adjust breakpoints to balance individual workload vs collaboration complexity.
var GridSizeBreakpoints = []GridBreakpoint{
	{MinPlayers: 1, MaxPlayers: 9, GridSize: 3, TotalFragments: 9},
	{MinPlayers: 10, MaxPlayers: 16, GridSize: 4, TotalFragments: 16},
	{MinPlayers: 17, MaxPlayers: 25, GridSize: 5, TotalFragments: 25},
}

// Difficulty Level Modifiers
type DifficultyModifiers struct {
	TriviaModifier        float64
	TimeLimitModifier     float64
	TokenThresholdModifier float64
}

// Difficulty settings for each game mode
var (
	// EasyMode - Modifiers applied for easy difficulty level
	EasyMode = DifficultyModifiers{
		TriviaModifier:        0.7,
		TimeLimitModifier:     1.3,
		TokenThresholdModifier: 0.8,
	}

	// MediumMode - Baseline modifiers for medium difficulty level
	MediumMode = DifficultyModifiers{
		TriviaModifier:        1.0,
		TimeLimitModifier:     1.0,
		TokenThresholdModifier: 1.0,
	}

	// HardMode - Modifiers applied for hard difficulty level
	HardMode = DifficultyModifiers{
		TriviaModifier:        1.4,
		TimeLimitModifier:     0.7,
		TokenThresholdModifier: 1.3,
	}
)

/*
TUNING GUIDELINES:

Testing Recommendations:
- Test with minimum and maximum expected player counts
- Validate trivia question interval doesn't create bottlenecks
- Ensure fragment movement cooldown feels responsive but prevents spam
- Balance resource token thresholds so they're achievable but meaningful
- Test difficulty modifiers across different group sizes and skill levels

Balance Considerations:
- Role bonuses should feel meaningful but not overpowered
- Specialty questions should reward knowledge without punishing generalists
- Time bonuses should extend gameplay without removing urgency
- Grid scaling should maintain engagement across all player counts
*/
}
