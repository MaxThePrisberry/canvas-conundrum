package game

import (
	"time"

	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/protocol"
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
	Station string // current station on record; "" = unknown
	Stats   PlayerStats
}

// PlayerStats accumulates per-player metrics used for scoring and analytics.
type PlayerStats struct {
	QuestionsDelivered  int
	QuestionsAnswered   int
	CorrectAnswers      int
	SpecialtyDelivered  int
	SpecialtyCorrect    int
	TokensEarned        int
	TotalResponseTime   float64 // seconds, over answered questions
	CorrectByCategory   map[string]int
	QuestionsByCategory map[string]int
	StationVisits       map[string]int
}

func (p *Player) send(event protocol.EventType, payload any) {
	if p.client != nil {
		p.client.Send(event, payload)
	}
}
