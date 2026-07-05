package protocol

import "fmt"

// Position is a 0-based grid coordinate (x = column, y = row, origin
// top-left).
type Position struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// ── Client → Server ────────────────────────────────────────────────────────

// PuzzleStart is PUZZLE_TO_SERVER_PHASE_START (host starts the timer).
// Rejected before tile generation completes or after the timer is running.
type PuzzleStart struct{}

func (PuzzleStart) Validate() error { return nil }

// ── Server → Client ────────────────────────────────────────────────────────

// PuzzlePhaseLoad is PUZZLE_TO_CLIENT_PHASE_LOAD: tiles are fetchable, the
// UI stays hidden until the host starts the timer.
type PuzzlePhaseLoad struct {
	Phase                  string  `json:"phase"`
	ImageID                string  `json:"imageId"`
	AssignedSegmentID      string  `json:"assignedSegmentId"`
	IndividualPuzzleSize   int     `json:"individualPuzzleSize"`
	AnchorPreSolvedPieces  int     `json:"anchorPreSolvedPieces"`
	CentralGridSize        int     `json:"centralGridSize"`
	TotalFragments         int     `json:"totalFragments"`
	ClarityPreviewDuration float64 `json:"clarityPreviewDuration"`
	GuideHighlightCount    int     `json:"guideHighlightCount"`
}

// HostPuzzlePhaseLoad is PUZZLE_TO_HOST_PHASE_LOAD.
type HostPuzzlePhaseLoad struct {
	Phase                    string            `json:"phase"`
	ImageID                  string            `json:"imageId"`
	CentralGridSize          int               `json:"centralGridSize"`
	TotalFragments           int               `json:"totalFragments"`
	PlayerCount              int               `json:"playerCount"`
	PlayerSegmentAssignments map[string]string `json:"playerSegmentAssignments"`
	BonusEffects             BonusEffects      `json:"bonusEffects"`
}

// PlayerPhases partitions players by puzzle sub-phase.
type PlayerPhases struct {
	Phase2A []string `json:"phase2a"`
	Phase2B []string `json:"phase2b"`
}

// PuzzlePhaseStart is PUZZLE_TO_CLIENT_PHASE_START. TotalTime is
// authoritative and already includes the difficulty multiplier;
// baseTime/chronosBonus are display-only decomposition.
type PuzzlePhaseStart struct {
	StartTimestamp         string       `json:"startTimestamp"`
	TotalTime              float64      `json:"totalTime"`
	BaseTime               float64      `json:"baseTime"`
	ChronosBonus           float64      `json:"chronosBonus"`
	ClarityPreviewActive   bool         `json:"clarityPreviewActive"`
	ClarityPreviewDuration float64      `json:"clarityPreviewDuration"`
	PlayerPhases           PlayerPhases `json:"playerPhases"`
}

// HostPuzzlePhaseStart is PUZZLE_TO_HOST_PHASE_START. On host-reconnect
// replay, TotalTime carries the remaining seconds (countdown anchor) while
// BaseTime/ChronosBonus keep their original values.
type HostPuzzlePhaseStart struct {
	TimerActive      bool    `json:"timerActive"`
	StartTimestamp   string  `json:"startTimestamp"`
	TotalTime        float64 `json:"totalTime"`
	BaseTime         float64 `json:"baseTime"`
	ChronosBonus     float64 `json:"chronosBonus"`
	PlayersInPhase2A int     `json:"playersInPhase2a"`
	PlayersInPhase2B int     `json:"playersInPhase2b"`
}

// ── Phase 2A: individual puzzles ───────────────────────────────────────────

// SegmentCompleted is PUZZLE_TO_SERVER_SEGMENT_COMPLETED — the client is
// authoritative for its own individual puzzle.
type SegmentCompleted struct {
	SegmentID           string  `json:"segmentId"`
	CompletionTimestamp string  `json:"completionTimestamp"`
	SolveTime           float64 `json:"solveTime"`
	ManualPiecesSolved  int     `json:"manualPiecesSolved"`
	PreSolvedPieces     int     `json:"preSolvedPieces"`
}

func (s SegmentCompleted) Validate() error {
	if s.SegmentID == "" {
		return fmt.Errorf("segmentId is required")
	}
	return nil
}

// SegmentAcknowledged is PUZZLE_TO_PLAYER_SEGMENT_ACKNOWLEDGED.
type SegmentAcknowledged struct {
	SegmentID string   `json:"segmentId"`
	Position  Position `json:"position"`
}

// PhaseTransition counts players by sub-phase.
type PhaseTransition struct {
	PlayersInPhase2A int `json:"playersInPhase2a"`
	PlayersInPhase2B int `json:"playersInPhase2b"`
}

// CompletionStats summarizes grid progress for the host.
type CompletionStats struct {
	TotalCompleted      int `json:"totalCompleted"`
	TotalRequired       int `json:"totalRequired"`
	UnassignedFragments int `json:"unassignedFragments"`
}

// HostSegmentCompleted is PUZZLE_TO_HOST_SEGMENT_COMPLETED.
type HostSegmentCompleted struct {
	PlayerID        string          `json:"playerId"`
	PlayerName      string          `json:"playerName"`
	SegmentID       string          `json:"segmentId"`
	CompletionTime  float64         `json:"completionTime"`
	Position        Position        `json:"position"`
	PhaseTransition PhaseTransition `json:"phaseTransition"`
	CompletionStats CompletionStats `json:"completionStats"`
}

// ── Phase 2B: fragment movement ────────────────────────────────────────────

// FragmentMove is PUZZLE_TO_SERVER_FRAGMENT_MOVE. SwapWithSegmentID nil
// means "move to empty cell".
type FragmentMove struct {
	SegmentID         string    `json:"segmentId"`
	TargetPosition    *Position `json:"targetPosition"`
	SwapWithSegmentID *string   `json:"swapWithSegmentId"`
}

func (m FragmentMove) Validate() error {
	if m.SegmentID == "" {
		return fmt.Errorf("segmentId is required")
	}
	if m.TargetPosition == nil {
		return fmt.Errorf("targetPosition is required")
	}
	return nil
}

// Move rejection reasons.
const (
	MoveRejectCooldown      = "cooldown"
	MoveRejectNotOwner      = "not_owner"
	MoveRejectTargetInvalid = "target_invalid"
	MoveRejectPhaseInvalid  = "phase_invalid"
)

// CooldownInfo describes when a fragment becomes movable again.
type CooldownInfo struct {
	NextMoveAvailable string  `json:"nextMoveAvailable"`
	CooldownRemaining float64 `json:"cooldownRemaining"`
}

// MoveResult is PUZZLE_TO_PLAYER_MOVE_RESULT. Status is "success" or
// "rejected"; the optional fields' presence depends on it.
type MoveResult struct {
	MoveID                    string        `json:"moveId"`
	Status                    string        `json:"status"`
	SegmentID                 string        `json:"segmentId"`
	NewPosition               *Position     `json:"newPosition,omitempty"`
	SwappedSegmentID          *string       `json:"swappedSegmentId,omitempty"`
	SwappedSegmentNewPosition *Position     `json:"swappedSegmentNewPosition,omitempty"`
	Reason                    string        `json:"reason,omitempty"`
	CooldownInfo              *CooldownInfo `json:"cooldownInfo,omitempty"`
}

// GridFragment is one visible fragment in PUZZLE_TO_CLIENT_GRID_STATE.
// PlayerID/PlayerName are null for unassigned fragments.
type GridFragment struct {
	SegmentID  string   `json:"segmentId"`
	PlayerID   *string  `json:"playerId"`
	PlayerName *string  `json:"playerName"`
	Position   Position `json:"position"`
}

// GridState is PUZZLE_TO_CLIENT_GRID_STATE.
type GridState struct {
	Fragments     []GridFragment `json:"fragments"`
	TimeRemaining float64        `json:"timeRemaining"`
}

// HostGridFragment is one fragment row in PUZZLE_TO_HOST_GRID_STATE.
type HostGridFragment struct {
	PlayerID   *string  `json:"playerId"`
	PlayerName *string  `json:"playerName"`
	SegmentID  string   `json:"segmentId"`
	Position   Position `json:"position"`
	LastMoved  string   `json:"lastMoved"`
	MoveCount  int      `json:"moveCount"`
}

// HostPlayerMetric is one player's row in PUZZLE_TO_HOST_GRID_STATE.
type HostPlayerMetric struct {
	Phase            string `json:"phase"`
	FragmentsOwned   int    `json:"fragmentsOwned"`
	MovesContributed int    `json:"movesContributed"`
	SuccessfulMoves  int    `json:"successfulMoves"`
	LastActivity     string `json:"lastActivity"`
}

// HostGridState is PUZZLE_TO_HOST_GRID_STATE (immediate on any change).
type HostGridState struct {
	Fragments             []HostGridFragment          `json:"fragments"`
	PlayerMetrics         map[string]HostPlayerMetric `json:"playerMetrics"`
	ActiveRecommendations int                         `json:"activeRecommendations"`
	TimeRemaining         float64                     `json:"timeRemaining"`
}

// PersonalState is PUZZLE_TO_PLAYER_PERSONAL_STATE — the only channel for
// guide highlights, which are private per player and immutable once drawn.
type PersonalState struct {
	GuideHighlights []Position `json:"guideHighlights"`
}

// ── Recommendations ────────────────────────────────────────────────────────

// RecommendMove is PUZZLE_TO_SERVER_RECOMMEND_MOVE.
type RecommendMove struct {
	TargetPlayerID string `json:"targetPlayerId"`
	FromSegmentID  string `json:"fromSegmentId"`
	ToSegmentID    string `json:"toSegmentId"`
	Reasoning      string `json:"reasoning"`
}

func (r RecommendMove) Validate() error {
	if r.TargetPlayerID == "" || r.FromSegmentID == "" || r.ToSegmentID == "" {
		return fmt.Errorf("targetPlayerId, fromSegmentId, and toSegmentId are required")
	}
	return validateText("reasoning", r.Reasoning, 0, MaxReasoningLen)
}

// MoveRecommendation is PUZZLE_TO_PLAYER_MOVE_RECOMMENDATION (to target).
type MoveRecommendation struct {
	MoveID         string `json:"moveId"`
	FromPlayerID   string `json:"fromPlayerId"`
	FromPlayerName string `json:"fromPlayerName"`
	TargetPlayerID string `json:"targetPlayerId"`
	FromSegmentID  string `json:"fromSegmentId"`
	ToSegmentID    string `json:"toSegmentId"`
	Reasoning      string `json:"reasoning"`
	ExpiresAt      string `json:"expiresAt"`
}

// RecommendationResponse is PUZZLE_TO_SERVER_RECOMMENDATION_RESPONSE.
// ResponseReason is optional.
type RecommendationResponse struct {
	MoveID         string  `json:"moveId"`
	Response       string  `json:"response"`
	ResponseReason *string `json:"responseReason"`
}

func (r RecommendationResponse) Validate() error {
	if r.MoveID == "" {
		return fmt.Errorf("moveId is required")
	}
	if r.Response != "accept" && r.Response != "reject" {
		return fmt.Errorf("response must be \"accept\" or \"reject\"")
	}
	if r.ResponseReason != nil {
		return validateText("responseReason", *r.ResponseReason, 0, MaxReasoningLen)
	}
	return nil
}

// SwapExecuted describes an executed recommendation swap.
type SwapExecuted struct {
	Segment1ID          string   `json:"segment1Id"`
	Segment1OldPosition Position `json:"segment1OldPosition"`
	Segment1NewPosition Position `json:"segment1NewPosition"`
	Segment2ID          string   `json:"segment2Id"`
	Segment2OldPosition Position `json:"segment2OldPosition"`
	Segment2NewPosition Position `json:"segment2NewPosition"`
}

// RecommendationResult is PUZZLE_TO_PLAYER_RECOMMENDATION_RESULT (to the
// recommender). SwapExecuted is present only on accept.
type RecommendationResult struct {
	MoveID           string        `json:"moveId"`
	TargetPlayerID   string        `json:"targetPlayerId"`
	TargetPlayerName string        `json:"targetPlayerName"`
	Response         string        `json:"response"`
	ResponseReason   string        `json:"responseReason,omitempty"`
	SwapExecuted     *SwapExecuted `json:"swapExecuted,omitempty"`
}

// RecommendationExpired is PUZZLE_TO_PLAYER_RECOMMENDATION_EXPIRED (to both
// parties). Reason is "timeout" or "player_disconnected".
type RecommendationExpired struct {
	MoveID string `json:"moveId"`
	Reason string `json:"reason"`
}

// ── Completion ─────────────────────────────────────────────────────────────

// FinalGridState summarizes the winning board.
type FinalGridState struct {
	AllFragmentsCorrect bool `json:"allFragmentsCorrect"`
	TotalFragments      int  `json:"totalFragments"`
	CorrectFragments    int  `json:"correctFragments"`
}

// CompletedSuccess is PUZZLE_TO_CLIENT_COMPLETED_SUCCESS (all participants).
type CompletedSuccess struct {
	Success        bool           `json:"success"`
	CompletionTime float64        `json:"completionTime"`
	TotalTime      float64        `json:"totalTime"`
	TimeRemaining  float64        `json:"timeRemaining"`
	FinalGridState FinalGridState `json:"finalGridState"`
}

// TimeoutFinalStats summarizes the losing board.
type TimeoutFinalStats struct {
	FragmentsPlaced      int     `json:"fragmentsPlaced"`
	TotalFragments       int     `json:"totalFragments"`
	CorrectlyPlaced      int     `json:"correctlyPlaced"`
	CompletionPercentage float64 `json:"completionPercentage"`
}

// CompletedTimeout is PUZZLE_TO_CLIENT_COMPLETED_TIMEOUT (all participants).
type CompletedTimeout struct {
	Success     bool              `json:"success"`
	Reason      string            `json:"reason"`
	TotalTime   float64           `json:"totalTime"`
	TimeExpired bool              `json:"timeExpired"`
	FinalStats  TimeoutFinalStats `json:"finalStats"`
}

// PlayerContribution is one player's row in PUZZLE_TO_HOST_COMPLETION_ANALYTICS.
type PlayerContribution struct {
	IndividualSolveTime     float64 `json:"individualSolveTime"`
	FragmentMoves           int     `json:"fragmentMoves"`
	SuccessfulMoves         int     `json:"successfulMoves"`
	RecommendationsSent     int     `json:"recommendationsSent"`
	RecommendationsReceived int     `json:"recommendationsReceived"`
	RecommendationsAccepted int     `json:"recommendationsAccepted"`
	FinalFragmentCorrect    bool    `json:"finalFragmentCorrect"`
}

// CollaborationMetrics aggregates team collaboration.
type CollaborationMetrics struct {
	TotalMoves              int     `json:"totalMoves"`
	SuccessfulMoves         int     `json:"successfulMoves"`
	TotalRecommendations    int     `json:"totalRecommendations"`
	AcceptedRecommendations int     `json:"acceptedRecommendations"`
	AverageResponseTime     float64 `json:"averageResponseTime"`
}

// IndividualPhaseTransitions summarizes 2A completion times (real
// completions only, not auto-solves).
type IndividualPhaseTransitions struct {
	PlayersCompletedIndividual int     `json:"playersCompletedIndividual"`
	AverageIndividualTime      float64 `json:"averageIndividualTime"`
	FastestIndividual          float64 `json:"fastestIndividual"`
	SlowestIndividual          float64 `json:"slowestIndividual"`
}

// CompletionAnalytics is PUZZLE_TO_HOST_COMPLETION_ANALYTICS.
type CompletionAnalytics struct {
	PuzzleSuccess        bool                          `json:"puzzleSuccess"`
	CompletionTime       float64                       `json:"completionTime"`
	TotalTime            float64                       `json:"totalTime"`
	PlayerContributions  map[string]PlayerContribution `json:"playerContributions"`
	CollaborationMetrics CollaborationMetrics          `json:"collaborationMetrics"`
	PhaseTransitions     IndividualPhaseTransitions    `json:"phaseTransitions"`
}
