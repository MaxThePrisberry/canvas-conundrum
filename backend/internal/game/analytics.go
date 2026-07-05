package game

import (
	"time"

	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/protocol"
)

// analyticsState captures the game outcome for the analytics phase.
type analyticsState struct {
	success           bool
	completionSeconds float64
	endedAt           time.Time
}

// enterAnalytics transitions puzzle_assembly → analytics. The report
// payloads (personal, team summary, host complete report) land in M7.
func (e *Engine) enterAnalytics(success bool, completionSeconds float64) {
	e.phase = protocol.PhaseAnalytics
	e.analytics = analyticsState{
		success:           success,
		completionSeconds: completionSeconds,
		endedAt:           time.Now(),
	}
}
