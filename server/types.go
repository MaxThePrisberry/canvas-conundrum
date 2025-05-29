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
