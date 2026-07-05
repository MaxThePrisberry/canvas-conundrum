package protocol

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
