package models

import (
	"time"

	"github.com/gorilla/websocket"
)

// Role represents a player's chosen role
type Role string

const (
	RoleArtEnthusiast Role = "art_enthusiast"
	RoleDetective     Role = "detective"
	RoleTourist       Role = "tourist"
	RoleJanitor       Role = "janitor"
	RoleNone          Role = ""
)

// GetBonusTokenType returns the token type this role gets a bonus for
func (r Role) GetBonusTokenType() TokenType {
	switch r {
	case RoleArtEnthusiast:
		return TokenClarity
	case RoleDetective:
		return TokenGuide
	case RoleTourist:
		return TokenChronos
	case RoleJanitor:
		return TokenAnchor
	default:
		return ""
	}
}

// Player represents a connected player
type Player struct {
	ID                string           `json:"id"`
	Name              string           `json:"name"`
	Connection        *websocket.Conn  `json:"-"`
	Role              Role             `json:"role"`
	Specialties       []TriviaCategory `json:"specialties"`
	IsReady           bool             `json:"isReady"`
	IsActive          bool             `json:"isActive"`
	CurrentStation    string           `json:"currentStation"`
	TokensEarned      int              `json:"tokensEarned"`
	QuestionsAnswered int              `json:"questionsAnswered"`
	CorrectAnswers    int              `json:"correctAnswers"`
	JoinedAt          time.Time        `json:"joinedAt"`
	LastSeen          time.Time        `json:"lastSeen"`

	// Puzzle phase specific
	AssignedSegment  string            `json:"assignedSegment"`
	IndividualPuzzle *IndividualPuzzle `json:"individualPuzzle,omitempty"`
	PuzzlePhase      string            `json:"puzzlePhase"` // "2A" (individual) or "2B" (collaborative)
	SegmentCompleted bool              `json:"segmentCompleted"`
	SegmentSolveTime float64           `json:"segmentSolveTime"`
	FragmentID       string            `json:"fragmentId"`
	FragmentMoves    int               `json:"fragmentMoves"`
	LastMoveTime     time.Time         `json:"-"`

	// Analytics
	RecommendationsSent     int `json:"recommendationsSent"`
	RecommendationsReceived int `json:"recommendationsReceived"`
	RecommendationsAccepted int `json:"recommendationsAccepted"`
	TotalScore              int `json:"totalScore"`

	// Channels for async operations
	Send chan []byte   `json:"-"`
	Done chan struct{} `json:"-"`
}

// NewPlayer creates a new player instance
func NewPlayer(id string, conn *websocket.Conn) *Player {
	return &Player{
		ID:           id,
		Connection:   conn,
		IsActive:     conn != nil,
		IsReady:      false,
		JoinedAt:     time.Now(),
		TokensEarned: 0,
		Specialties:  []TriviaCategory{},
		Send:         make(chan []byte, 256),
		Done:         make(chan struct{}),
	}
}

// GetAccuracy returns the player's trivia accuracy
func (p *Player) GetAccuracy() float64 {
	if p.QuestionsAnswered == 0 {
		return 0
	}
	return float64(p.CorrectAnswers) / float64(p.QuestionsAnswered)
}

// CanMoveFragment checks if enough time has passed since last move
func (p *Player) CanMoveFragment(cooldownMs int) bool {
	if p.LastMoveTime.IsZero() {
		return true
	}
	elapsed := time.Since(p.LastMoveTime).Milliseconds()
	return elapsed >= int64(cooldownMs)
}

// UpdateLastMove updates the last move timestamp
func (p *Player) UpdateLastMove() {
	p.LastMoveTime = time.Now()
	p.FragmentMoves++
}

// Host represents the game host
type Host struct {
	ID          string          `json:"id"`
	Connection  *websocket.Conn `json:"-"`
	ConnectedAt time.Time       `json:"connectedAt"`
	Send        chan []byte     `json:"-"`
	Done        chan struct{}   `json:"-"`
}

// NewHost creates a new host instance
func NewHost(id string, conn *websocket.Conn) *Host {
	return &Host{
		ID:          id,
		Connection:  conn,
		ConnectedAt: time.Now(),
		Send:        make(chan []byte, 256),
		Done:        make(chan struct{}),
	}
}
