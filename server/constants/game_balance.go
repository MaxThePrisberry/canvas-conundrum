package constants

import "time"

// Character Role Settings
const (
	// RoleResourceMultiplier - Multiplier applied to resource collection for character role bonuses
	// Used in: game_manager.go ProcessTriviaAnswer() and player_manager.go GetAvailableRoles()
	RoleResourceMultiplier float64 = 1.5
)

// Role Types - All used in player_manager.go and game_manager.go
const (
	RoleArtEnthusiast = "art_enthusiast"
	RoleDetective     = "detective"
	RoleTourist       = "tourist"
	RoleJanitor       = "janitor"
)

// Token Types - All used throughout game_manager.go
const (
	TokenAnchor  = "anchor"
	TokenChronos = "chronos"
	TokenGuide   = "guide"
	TokenClarity = "clarity"
)

// Role to Token Type Associations - Used in game_manager.go ProcessTriviaAnswer()
var RoleTokenBonuses = map[string]string{
	RoleArtEnthusiast: TokenClarity,
	RoleDetective:     TokenGuide,
	RoleTourist:       TokenChronos,
	RoleJanitor:       TokenAnchor,
}

// Trivia Categories - Used in trivia_manager.go and event_handlers.go
var TriviaCategories = []string{
	"general",
	"geography",
	"history",
	"music",
	"science",
	"video_games",
}

// Trivia Mechanics - All used in game_manager.go and trivia_manager.go
const (
	// SpecialtyPointMultiplier - Point multiplier for correctly answering specialty trivia questions
	// Used in: game_manager.go ProcessTriviaAnswer()
	SpecialtyPointMultiplier float64 = 2.0

	// BaseTokensPerCorrectAnswer - Base number of tokens earned for correct answer
	// Used in: game_manager.go ProcessTriviaAnswer()
	BaseTokensPerCorrectAnswer int = 10

	// TriviaAnswerTimeout - Time limit for answering trivia questions (seconds)
	// Used in: trivia_manager.go loadQuestionsFromFile() and GetQuestion()
	// Note: Same timeout applies to both regular and specialty questions
	TriviaAnswerTimeout int = 30

	// MaxSpecialtiesPerPlayer - Maximum number of specialty categories per player
	// Used in: player_manager.go SetPlayerSpecialties()
	MaxSpecialtiesPerPlayer int = 2
)

// Resource Token Settings - All used in game_manager.go for token threshold calculations
const (
	// AnchorTokenThresholds - Number of anchor token thresholds available
	// Used in: game_manager.go startPuzzlePhase() and calculateThresholdsReached()
	AnchorTokenThresholds int = 5

	// ChronosTokenThresholds - Number of chronos token thresholds available
	// Used in: game_manager.go StartPuzzle() and calculateThresholdsReached()
	ChronosTokenThresholds int = 5

	// ChronosTimeBonus - Time bonus added per chronos token threshold reached (seconds)
	// Used in: game_manager.go StartPuzzle()
	ChronosTimeBonus int = 20

	// GuideTokenThresholds - Number of guide token thresholds available
	// Used in: game_manager.go sendGuideHints() and calculateThresholdsReached()
	GuideTokenThresholds int = 5

	// ClarityTokenThresholds - Number of clarity token thresholds available
	// Used in: game_manager.go startPuzzlePhase() and calculateThresholdsReached()
	ClarityTokenThresholds int = 5

	// ClarityTimeBonus - Additional image display time per clarity token threshold (seconds)
	// Used in: game_manager.go startPuzzlePhase()
	ClarityTimeBonus int = 1
)

// Guide Token Linear Progression - NEW: Implementation for linear guide highlighting
const (
	// GuideTokenMaxThresholds - Maximum number of guide token thresholds for linear progression
	GuideTokenMaxThresholds int = 5

	// GuideTokenLinearSteps - Number of linear progression steps for guide highlighting
	GuideTokenLinearSteps int = 5
)

// GuideHighlightSizes - Linear progression from large area (25%) to precise (2 positions)
// Index corresponds to threshold level (0-4), values are grid percentage coverage
var GuideHighlightSizes = []float64{
	0.25, // Level 0: 25% of grid (very vague)
	0.16, // Level 1: 16% of grid
	0.09, // Level 2: 9% of grid
	0.04, // Level 3: 4% of grid
	0.02, // Level 4: 2% of grid (2 positions for precision)
}

// Puzzle Mechanics - All used in game_manager.go
const (
	// FragmentMovementCooldown - Cooldown period after moving a puzzle fragment (milliseconds)
	// Used in: game_manager.go ProcessFragmentMove()
	FragmentMovementCooldown int = 1000

	// IndividualPuzzlePieces - Number of pieces in each individual player puzzle fragment
	// Used in: game_manager.go startPuzzlePhase() for anchor token calculations
	IndividualPuzzlePieces int = 16
)

// Player Limits - Used in event_handlers.go and main.go
const (
	MinPlayers = 4
	MaxPlayers = 64
)

// Phase Durations - All used in game_manager.go and event_handlers.go
const (
	// LobbyCountdownDuration - Time after minimum players reached before game can start (seconds)
	// Used in: event_handlers.go startGameCountdown()
	LobbyCountdownDuration int = 30

	// ResourceGatheringRounds - Number of rounds in resource gathering phase
	// Used in: game_manager.go runResourceGatheringPhase() and sendTeamProgressUpdate()
	ResourceGatheringRounds int = 5

	// ResourceGatheringRoundDuration - Duration of each resource gathering round (seconds)
	// FIXED: Changed from 180 to 60 to match documentation requirement
	// Used in: game_manager.go runTriviaRound() and sendHostUpdate()
	ResourceGatheringRoundDuration int = 60

	// PuzzleAssemblyBaseTime - Base time for puzzle assembly phase (seconds)
	// Used in: game_manager.go StartPuzzle()
	PuzzleAssemblyBaseTime int = 300

	// PostGameAnalyticsDuration - Time to display analytics before reset (seconds)
	// Used in: game_manager.go endGame()
	PostGameAnalyticsDuration int = 60
)

// Resource Station Hashes - Used in game_manager.go and player_manager.go
var ResourceStationHashes = map[string]string{
	TokenAnchor:  "HASH_ANCHOR_STATION_2025",
	TokenChronos: "HASH_CHRONOS_STATION_2025",
	TokenGuide:   "HASH_GUIDE_STATION_2025",
	TokenClarity: "HASH_CLARITY_STATION_2025",
}

// Puzzle Configuration - Used in game_manager.go startPuzzlePhase()
const (
	// Number of different puzzle images available
	AvailablePuzzleImages = 10
)

// Grid Scaling Configuration - Used in game_manager.go calculateGridSize()
type GridBreakpoint struct {
	MinPlayers     int
	MaxPlayers     int
	GridSize       int
	TotalFragments int
}

// GridSizeBreakpoints - Player count breakpoints for determining puzzle grid size
var GridSizeBreakpoints = []GridBreakpoint{
	{MinPlayers: 1, MaxPlayers: 9, GridSize: 3, TotalFragments: 9},
	{MinPlayers: 10, MaxPlayers: 16, GridSize: 4, TotalFragments: 16},
	{MinPlayers: 17, MaxPlayers: 25, GridSize: 5, TotalFragments: 25},
	{MinPlayers: 26, MaxPlayers: 36, GridSize: 6, TotalFragments: 36},
	{MinPlayers: 37, MaxPlayers: 49, GridSize: 7, TotalFragments: 49},
	{MinPlayers: 50, MaxPlayers: 64, GridSize: 8, TotalFragments: 64},
}

// Difficulty Level Modifiers - All used in game_manager.go and trivia_manager.go
type DifficultyModifiers struct {
	TriviaModifier         float64 // Affects question difficulty selection
	TimeLimitModifier      float64 // Affects time limits for all phases
	TokenThresholdModifier float64 // Affects token requirements for thresholds
}

// Difficulty settings - All used in game_manager.go getDifficultyModifiers()
var (
	// EasyMode - Modifiers applied for easy difficulty level
	EasyMode = DifficultyModifiers{
		TriviaModifier:         0.7, // Easier questions
		TimeLimitModifier:      1.3, // More time
		TokenThresholdModifier: 0.8, // Lower token requirements
	}

	// MediumMode - Baseline modifiers for medium difficulty level
	MediumMode = DifficultyModifiers{
		TriviaModifier:         1.0, // Normal questions
		TimeLimitModifier:      1.0, // Normal time
		TokenThresholdModifier: 1.0, // Normal token requirements
	}

	// HardMode - Modifiers applied for hard difficulty level
	HardMode = DifficultyModifiers{
		TriviaModifier:         1.4, // Harder questions
		TimeLimitModifier:      0.7, // Less time
		TokenThresholdModifier: 1.3, // Higher token requirements
	}
)

// Timing Constants - All used in websocket_handlers.go and main.go
const (
	WebSocketPingInterval    = 30 * time.Second
	WebSocketPongTimeout     = 60 * time.Second
	BroadcastChannelBuffer   = 256
	PlayerEventChannelBuffer = 64
)

// Error Messages - Used throughout the application for consistent error handling
const (
	// Fragment ownership errors
	ErrFragmentOwnership  = "you can only move your own fragment or unassigned fragments"
	ErrFragmentNotVisible = "fragment is not yet visible"
	ErrFragmentUnassigned = "invalid unassigned fragment access"

	// Recommendation errors
	ErrInvalidRecommendation = "can only recommend moves for unassigned fragments"
	ErrRecommendationAuth    = "not authorized to respond to this recommendation"

	// Phase errors
	ErrWrongPhase            = "action not allowed in current game phase"
	ErrReconnectionForbidden = "reconnection not allowed during puzzle assembly phase"

	// Host errors
	ErrHostOnly   = "only host can perform this action"
	ErrHostExists = "a host is already connected to this game"

	// Validation errors
	ErrInvalidOwnership = "invalid fragment ownership format"
)
