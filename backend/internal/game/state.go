package game

import (
	"time"

	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/protocol"
	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/trivia"
)

// Player is everything the server tracks about one player across the whole
// game. Post-setup, players persist through disconnects (Connected=false);
// during setup a disconnected player is excluded from counts and role
// distribution but their selection data is preserved for reconnection.
type Player struct {
	ID          string // the player's session token (server-issued UUID)
	Name        string
	Role        string // "" until configured / after losing the slot
	Specialties []string
	Ready       bool
	Connected   bool

	LastActivity time.Time
	client       Client // nil while disconnected

	// Resource-gathering state.
	Station  string          // current station on record; "" = unknown
	Question *ActiveQuestion // this round's question, nil if none delivered
	Stats    PlayerStats
}

// ActiveQuestion is one delivered trivia question awaiting (or past) its
// answer window.
type ActiveQuestion struct {
	Q            trivia.Question
	Options      []string // shuffled: correct + incorrect
	CorrectIndex int
	IsSpecialty  bool
	Deadline     time.Time
	Closed       bool
	Answer       *SubmittedAnswer
}

// SubmittedAnswer is the player's latest answer (resubmission overwrites).
type SubmittedAnswer struct {
	Index       int
	TimeElapsed float64 // seconds, client-reported
}

// PlayerStats accumulates per-player metrics used for scoring and analytics.
type PlayerStats struct {
	QuestionsDelivered   int
	QuestionsAnswered    int
	CorrectAnswers       int
	SpecialtyDelivered   int
	SpecialtyCorrect     int
	SpecialtyBonusTokens int
	TokensEarned         int
	TotalResponseTime    float64 // seconds, over answered questions
	CorrectByCategory    map[string]int
	QuestionsByCategory  map[string]int
	StationVisits        map[string]int

	// Puzzle-assembly metrics.
	CompletedIndividual     bool    // real completion (not an auto-solve)
	IndividualSolveTime     float64 // seconds, client-reported
	FragmentMoves           int     // move requests processed
	SuccessfulMoves         int
	RecommendationsSent     int
	RecommendationsReceived int
	RecommendationsAccepted int // received and accepted
	RecResponseTimeSum      float64
	RecResponses            int
}

func (s *PlayerStats) countCategory(category string, correct bool) {
	if s.QuestionsByCategory == nil {
		s.QuestionsByCategory = map[string]int{}
		s.CorrectByCategory = map[string]int{}
	}
	s.QuestionsByCategory[category]++
	if correct {
		s.CorrectByCategory[category]++
	}
}

func (p *Player) send(event protocol.EventType, payload any) {
	if p.client != nil {
		p.client.Send(event, payload)
	}
}
