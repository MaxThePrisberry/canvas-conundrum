package main

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Game Phases
type GamePhase int

const (
	PhaseSetup GamePhase = iota
	PhaseResourceGathering
	PhasePuzzleAssembly
	PhasePostGame
)

// Player States
type PlayerState int

const (
	StateConnected PlayerState = iota
	StateDisconnected
	StateReady
)

// WebSocket Message Types - Server to Client
const (
	MsgAvailableRoles       = "available_roles"
	MsgGameLobbyStatus      = "game_lobby_status"
	MsgResourcePhaseStart   = "resource_phase_start"
	MsgTriviaQuestion       = "trivia_question"
	MsgTeamProgressUpdate   = "team_progress_update"
	MsgPuzzlePhaseLoad      = "puzzle_phase_load"
	MsgPuzzlePhaseStart     = "puzzle_phase_start"
	MsgSegmentCompletionAck = "segment_completion_ack"
	MsgFragmentMoveResponse = "fragment_move_response"
	MsgCentralPuzzleState   = "central_puzzle_state"
	MsgGameAnalytics        = "game_analytics"
	MsgGameReset            = "game_reset"
	MsgError                = "error"
	MsgHostUpdate           = "host_update"
	MsgCountdown            = "countdown"
	MsgPieceRecommendation  = "piece_recommendation"
	MsgImagePreview         = "image_preview"
	MsgPersonalPuzzleState  = "personal_puzzle_state"
	MsgGuideHighlight       = "guide_highlight"
)

// WebSocket Message Types - Client to Server
const (
	MsgPlayerJoin                  = "player_join"
	MsgRoleSelection               = "role_selection"
	MsgTriviaSpecialtySelection    = "trivia_specialty_selection"
	MsgResourceLocationVerified    = "resource_location_verified"
	MsgTriviaAnswer                = "trivia_answer"
	MsgSegmentCompleted            = "segment_completed"
	MsgFragmentMoveRequest         = "fragment_move_request"
	MsgPlayerReady                 = "player_ready"
	MsgHostStartGame               = "host_start_game"
	MsgHostStartPuzzle             = "host_start_puzzle"
	MsgPieceRecommendationRequest  = "piece_recommendation_request"
	MsgPieceRecommendationResponse = "piece_recommendation_response"
)

// Base message structure for all communications
type BaseMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// Client authentication wrapper
type AuthWrapper struct {
	Auth    AuthData        `json:"auth"`
	Payload json.RawMessage `json:"payload"`
}

type AuthData struct {
	PlayerID string `json:"playerId"`
}

// Player represents a connected player
type Player struct {
	ID              string
	Name            string
	Role            string
	Specialties     []string
	State           PlayerState
	Connection      *websocket.Conn
	CurrentLocation string // Resource station hash
	IsHost          bool
	Ready           bool
	LastSeen        time.Time
	mu              sync.RWMutex
	writeMu         sync.Mutex // Protects WebSocket writes
}

// Role information
type RoleInfo struct {
	Role          string  `json:"role"`
	ResourceBonus float64 `json:"resourceBonus"`
	Available     bool    `json:"available"`
}

// Trivia Question
type TriviaQuestion struct {
	ID               string   `json:"questionId"`
	Text             string   `json:"text"`
	Category         string   `json:"category"`
	Difficulty       string   `json:"difficulty"`
	TimeLimit        int      `json:"timeLimit"`
	Options          []string `json:"options,omitempty"`
	CorrectAnswer    string   `json:"-"`
	IncorrectAnswers []string `json:"-"`
	IsSpecialty      bool     `json:"isSpecialty"`
}

// Trivia Question from JSON
type TriviaQuestionJSON struct {
	Type             string   `json:"type"`
	Difficulty       string   `json:"difficulty"`
	Category         string   `json:"category"`
	Question         string   `json:"question"`
	CorrectAnswer    string   `json:"correct_answer"`
	IncorrectAnswers []string `json:"incorrect_answers"`
}

// Team Tokens
type TeamTokens struct {
	AnchorTokens  int `json:"anchorTokens"`
	ChronosTokens int `json:"chronosTokens"`
	GuideTokens   int `json:"guideTokens"`
	ClarityTokens int `json:"clarityTokens"`
}

// Puzzle Fragment
type PuzzleFragment struct {
	ID              string    `json:"id"`
	PlayerID        string    `json:"playerId"`
	Position        GridPos   `json:"position"`
	Solved          bool      `json:"solved"`
	LastMoved       time.Time `json:"-"`
	CorrectPosition GridPos   `json:"correctPosition"`
	PreSolved       bool      `json:"preSolved"`
	Visible         bool      `json:"visible"`
	MovableBy       string    `json:"movableBy"`
	IsUnassigned    bool      `json:"-"`
}

// Grid Position
type GridPos struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// Piece Recommendation
type PieceRecommendation struct {
	ID               string    `json:"id"`
	FromPlayerID     string    `json:"fromPlayerId"`
	ToPlayerID       string    `json:"toPlayerId"`
	FromFragmentID   string    `json:"fromFragmentId"`
	ToFragmentID     string    `json:"toFragmentId"`
	SuggestedFromPos GridPos   `json:"suggestedFromPos"`
	SuggestedToPos   GridPos   `json:"suggestedToPos"`
	Message          string    `json:"message,omitempty"`
	Timestamp        time.Time `json:"timestamp"`
}

// Player Analytics
type PlayerAnalytics struct {
	PlayerID          string               `json:"playerId"`
	PlayerName        string               `json:"playerName"`
	TokenCollection   map[string]int       `json:"tokenCollection"`
	TriviaPerformance TriviaPerformance    `json:"triviaPerformance"`
	PuzzleMetrics     PuzzleSolvingMetrics `json:"puzzleSolvingMetrics"`
}

type TriviaPerformance struct {
	TotalQuestions     int                `json:"totalQuestions"`
	CorrectAnswers     int                `json:"correctAnswers"`
	AccuracyByCategory map[string]float64 `json:"accuracyByCategory"`
	SpecialtyBonus     int                `json:"specialtyBonus"`
	SpecialtyCorrect   int                `json:"specialtyCorrect"`
	SpecialtyTotal     int                `json:"specialtyTotal"`
}

type PuzzleSolvingMetrics struct {
	FragmentSolveTime       int `json:"fragmentSolveTime"`
	MovesContributed        int `json:"movesContributed"`
	SuccessfulMoves         int `json:"successfulMoves"`
	RecommendationsSent     int `json:"recommendationsSent"`
	RecommendationsReceived int `json:"recommendationsReceived"`
	RecommendationsAccepted int `json:"recommendationsAccepted"`
}

// Team Analytics
type TeamAnalytics struct {
	OverallPerformance  TeamPerformance      `json:"overallPerformance"`
	CollaborationScores CollaborationMetrics `json:"collaborationScores"`
	ResourceEfficiency  ResourceMetrics      `json:"resourceEfficiency"`
}

type TeamPerformance struct {
	TotalTime      int     `json:"totalTime"`
	CompletionRate float64 `json:"completionRate"`
	TotalScore     int     `json:"totalScore"`
}

type CollaborationMetrics struct {
	AverageResponseTime     float64 `json:"averageResponseTime"`
	CommunicationScore      float64 `json:"communicationScore"`
	CoordinationScore       float64 `json:"coordinationScore"`
	TotalRecommendations    int     `json:"totalRecommendations"`
	AcceptedRecommendations int     `json:"acceptedRecommendations"`
}

type ResourceMetrics struct {
	TokensPerRound    float64            `json:"tokensPerRound"`
	TokenDistribution map[string]float64 `json:"tokenDistribution"`
	ThresholdsReached map[string]int     `json:"thresholdsReached"`
}

// Leaderboard Entry
type LeaderboardEntry struct {
	PlayerID   string `json:"playerId"`
	PlayerName string `json:"playerName"`
	TotalScore int    `json:"totalScore"`
	Rank       int    `json:"rank"`
}

// Host Update Message
type HostUpdate struct {
	Phase            string                  `json:"phase"`
	ConnectedPlayers int                     `json:"connectedPlayers"`
	ReadyPlayers     int                     `json:"readyPlayers"`
	CurrentRound     int                     `json:"currentRound,omitempty"`
	TimeRemaining    int                     `json:"timeRemaining,omitempty"`
	TeamTokens       TeamTokens              `json:"teamTokens,omitempty"`
	PlayerStatuses   map[string]PlayerStatus `json:"playerStatuses"`
	PuzzleProgress   float64                 `json:"puzzleProgress,omitempty"`
}

type PlayerStatus struct {
	Name      string `json:"name"`
	Role      string `json:"role"`
	Connected bool   `json:"connected"`
	Ready     bool   `json:"ready"`
	Location  string `json:"location,omitempty"`
}

// Game State
type GameState struct {
	Phase                GamePhase
	Difficulty           string
	Players              map[string]*Player
	TeamTokens           TeamTokens
	CurrentRound         int
	RoundStartTime       time.Time
	PuzzleStartTime      time.Time
	PuzzleFragments      map[string]*PuzzleFragment
	GridSize             int
	PuzzleImageID        string
	QuestionHistory      map[string]map[string]bool // playerID -> questionID -> answered
	PlayerAnalytics      map[string]*PlayerAnalytics
	FragmentMoveHistory  []FragmentMove
	PieceRecommendations map[string]*PieceRecommendation // recommendationID -> recommendation
	CurrentQuestions     map[string]*TriviaQuestion      // playerID -> current question
	mu                   sync.RWMutex
}

type FragmentMove struct {
	FragmentID string    `json:"fragmentId"`
	FromPos    GridPos   `json:"fromPos"`
	ToPos      GridPos   `json:"toPos"`
	PlayerID   string    `json:"playerId"`
	Timestamp  time.Time `json:"timestamp"`
}

// Broadcast message structure
type BroadcastMessage struct {
	Type    string
	Payload interface{}
	Filter  func(*Player) bool // Optional filter to send to specific players
}

// Personal Puzzle State - Individual player view of the puzzle
type PersonalPuzzleState struct {
	Fragments        []*PuzzleFragment `json:"fragments"`        // Only visible fragments
	GridSize         int               `json:"gridSize"`         // Grid dimensions
	PlayerFragmentID string            `json:"playerFragmentId"` // Player's own fragment ID
	GuideHighlight   *GuideHighlight   `json:"guideHighlight"`   // Player-specific guide highlighting
}

// Guide Highlight - Linear progression guide token effects
type GuideHighlight struct {
	PlayerID       string    `json:"playerId"`       // Player receiving the highlight
	Positions      []GridPos `json:"positions"`      // Array of highlighted grid positions
	ThresholdLevel int       `json:"thresholdLevel"` // Current guide token threshold level (0-5)
	MaxThresholds  int       `json:"maxThresholds"`  // Maximum possible thresholds
	CoverageSize   float64   `json:"coverageSize"`   // Percentage of grid covered (0.02-0.25)
}

// Complete Puzzle State - Enhanced host view with ownership information
type CompletePuzzleState struct {
	Fragments           []*PuzzleFragment    `json:"fragments"`           // All fragments (visible and invisible)
	GridSize            int                  `json:"gridSize"`            // Grid dimensions
	OwnershipMapping    map[string]string    `json:"ownershipMapping"`    // fragmentId -> playerId or "unassigned"
	UnassignedFragments []string             `json:"unassignedFragments"` // List of unassigned fragment IDs
	VisibilityStatus    map[string]bool      `json:"visibilityStatus"`    // fragmentId -> visible status
	CompletionPercent   float64              `json:"completionPercent"`   // Percentage of puzzle completed
	MovementHistory     []FragmentMove       `json:"movementHistory"`     // Recent movement activity
	CollaborationStats  CollaborationSummary `json:"collaborationStats"`  // Real-time collaboration metrics
}

// Collaboration Summary - Real-time collaboration metrics for host
type CollaborationSummary struct {
	TotalMoves               int              `json:"totalMoves"`               // Total fragment moves
	PlayerOwnedMoves         int              `json:"playerOwnedMoves"`         // Moves of own fragments
	UnassignedMoves          int              `json:"unassignedMoves"`          // Moves of unassigned fragments
	ActiveRecommendations    int              `json:"activeRecommendations"`    // Pending recommendations
	RecommendationAcceptRate float64          `json:"recommendationAcceptRate"` // Acceptance rate %
	MovementsByPlayer        map[string]int   `json:"movementsByPlayer"`        // playerId -> move count
	RecentActivity           []RecentActivity `json:"recentActivity"`           // Last 10 significant events
}

// Recent Activity - Individual activity events for host monitoring
type RecentActivity struct {
	Type        string    `json:"type"`        // "move", "recommendation", "completion", "disconnection"
	PlayerID    string    `json:"playerId"`    // Player involved
	FragmentID  string    `json:"fragmentId"`  // Fragment involved
	Description string    `json:"description"` // Human-readable description
	Timestamp   time.Time `json:"timestamp"`   // When event occurred
}

// Individual Puzzle Progress - Track individual puzzle solving progress
type IndividualPuzzleProgress struct {
	PlayerID         string    `json:"playerId"`         // Player solving the puzzle
	SegmentID        string    `json:"segmentId"`        // Puzzle segment being solved
	PiecesRemaining  int       `json:"piecesRemaining"`  // Pieces left to solve (out of 16)
	PiecesSolved     int       `json:"piecesSolved"`     // Pieces already solved
	StartTime        time.Time `json:"startTime"`        // When player started solving
	EstimatedFinish  time.Time `json:"estimatedFinish"`  // Estimated completion time
	IsPreSolved      bool      `json:"isPreSolved"`      // Whether anchor tokens pre-solved this
	CompletionStatus string    `json:"completionStatus"` // "in_progress", "completed", "pre_solved"
}

// Enhanced Host Update - Add individual puzzle tracking
type EnhancedHostUpdate struct {
	HostUpdate                                          // Embed existing HostUpdate
	IndividualPuzzleProgress []IndividualPuzzleProgress `json:"individualPuzzleProgress"` // Track individual solving
	CompletePuzzleState      CompletePuzzleState        `json:"completePuzzleState"`      // Full puzzle state
	UnassignedFragmentStatus UnassignedFragmentStatus   `json:"unassignedFragmentStatus"` // Unassigned fragment info
}

// Unassigned Fragment Status - Track community fragments
type UnassignedFragmentStatus struct {
	TotalUnassigned    int      `json:"totalUnassigned"`    // Total unassigned fragments
	VisibleUnassigned  int      `json:"visibleUnassigned"`  // Visible unassigned fragments
	PendingRelease     int      `json:"pendingRelease"`     // Fragments waiting to be released
	NextReleaseTime    int64    `json:"nextReleaseTime"`    // Unix timestamp of next release
	UnassignedIDs      []string `json:"unassignedIds"`      // List of unassigned fragment IDs
	CommunityMoveCount int      `json:"communityMoveCount"` // Moves made on unassigned fragments
}

// Enhanced Player Analytics - Add individual puzzle metrics
type EnhancedPlayerAnalytics struct {
	PlayerAnalytics                                 // Embed existing analytics
	IndividualPuzzleMetrics IndividualPuzzleMetrics `json:"individualPuzzleMetrics"` // Individual puzzle solving stats
	CollaborationMetrics    PlayerCollaboration     `json:"collaborationMetrics"`    // Player collaboration stats
	GuideTokenUtilization   GuideTokenStats         `json:"guideTokenUtilization"`   // Guide token effectiveness
}

// Individual Puzzle Metrics - Track individual puzzle solving performance
type IndividualPuzzleMetrics struct {
	SegmentID         string  `json:"segmentId"`         // Assigned puzzle segment
	PiecesManuallySet int     `json:"piecesManuallySet"` // Pieces placed manually (not pre-solved)
	PiecesPreSolved   int     `json:"piecesPreSolved"`   // Pieces pre-solved by anchor tokens
	SolvingStartTime  int64   `json:"solvingStartTime"`  // Unix timestamp when solving started
	CompletionTime    int64   `json:"completionTime"`    // Unix timestamp when completed
	SolvingDuration   int     `json:"solvingDuration"`   // Seconds spent solving individual puzzle
	EfficiencyScore   float64 `json:"efficiencyScore"`   // Efficiency metric (0-1.0)
	DifficultyRating  float64 `json:"difficultyRating"`  // Perceived difficulty based on time
}

// Player Collaboration - Enhanced collaboration tracking
type PlayerCollaboration struct {
	OwnFragmentMoves        int     `json:"ownFragmentMoves"`        // Moves of own fragment
	UnassignedFragmentMoves int     `json:"unassignedFragmentMoves"` // Moves of unassigned fragments
	RecommendationsSent     int     `json:"recommendationsSent"`     // Recommendations sent to others
	RecommendationsReceived int     `json:"recommendationsReceived"` // Recommendations received
	RecommendationsAccepted int     `json:"recommendationsAccepted"` // Recommendations accepted
	AcceptanceRate          float64 `json:"acceptanceRate"`          // Acceptance rate %
	VerbalCoordination      int     `json:"verbalCoordination"`      // Inferred coordination events
	HelpfulnessScore        float64 `json:"helpfulnessScore"`        // Community helpfulness (0-1.0)
}

// Guide Token Stats - Track guide token effectiveness for individual players
type GuideTokenStats struct {
	CurrentThresholdLevel int       `json:"currentThresholdLevel"` // Current guide token level (0-5)
	HighlightPositions    []GridPos `json:"highlightPositions"`    // Current highlighted positions
	HighlightCoverage     float64   `json:"highlightCoverage"`     // Grid coverage percentage
	MovesWithinHighlight  int       `json:"movesWithinHighlight"`  // Moves made within highlighted area
	MovesOutsideHighlight int       `json:"movesOutsideHighlight"` // Moves made outside highlighted area
	GuideEffectiveness    float64   `json:"guideEffectiveness"`    // Effectiveness score (0-1.0)
	LastHighlightUpdate   time.Time `json:"lastHighlightUpdate"`   // When highlight was last updated
}
