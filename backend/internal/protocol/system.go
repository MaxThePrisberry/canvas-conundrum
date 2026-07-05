package protocol

// Ping is SYSTEM_PING (client → server, every 30s, auth included).
type Ping struct {
	ClientTimestamp string `json:"clientTimestamp"`
	SequenceNumber  int    `json:"sequenceNumber"`
}

func (Ping) Validate() error { return nil }

// Pong is SYSTEM_PONG. ClientTimestamp echoes the ping so the client can
// compute round-trip time.
type Pong struct {
	ServerTimestamp string `json:"serverTimestamp"`
	ClientTimestamp string `json:"clientTimestamp"`
	SequenceNumber  int    `json:"sequenceNumber"`
}

// GameImpact describes what a host disconnect does to the current phase.
type GameImpact struct {
	CanContinue      bool     `json:"canContinue"`
	AffectedFeatures []string `json:"affectedFeatures"`
}

// HostDisconnected is SYSTEM_TO_CLIENT_HOST_DISCONNECTED. TimerPausedAt is
// optional — present only when the puzzle timer pauses (puzzle_assembly).
type HostDisconnected struct {
	HostStatus    string     `json:"hostStatus"`
	CurrentPhase  string     `json:"currentPhase"`
	GameImpact    GameImpact `json:"gameImpact"`
	TimerPausedAt string     `json:"timerPausedAt,omitempty"`
}

// HostReconnected is SYSTEM_TO_CLIENT_HOST_RECONNECTED. TimeRemaining is
// optional — present only when the puzzle timer resumes.
type HostReconnected struct {
	HostStatus       string   `json:"hostStatus"`
	CurrentPhase     string   `json:"currentPhase"`
	RestoredFeatures []string `json:"restoredFeatures"`
	TimeRemaining    *float64 `json:"timeRemaining,omitempty"`
}

// DisconnectCounts is the setup-phase extra in SYSTEM_TO_HOST_PLAYER_DISCONNECTED.
type DisconnectCounts struct {
	ConnectedPlayers int            `json:"connectedPlayers"`
	ReadyPlayers     int            `json:"readyPlayers"`
	RoleDistribution map[string]int `json:"roleDistribution"`
}

// FragmentHandling is the puzzle-assembly extra in SYSTEM_TO_HOST_PLAYER_DISCONNECTED.
type FragmentHandling struct {
	SegmentID     string `json:"segmentId"`
	NewPosition   any    `json:"newPosition"`
	NowUnassigned bool   `json:"nowUnassigned"`
}

// PlayerDisconnected is SYSTEM_TO_HOST_PLAYER_DISCONNECTED. Exactly one of
// the phase-specific blocks is present: UpdatedCounts during setup;
// FragmentHandling + UpdatedPlayerCount during puzzle_assembly;
// UpdatedPlayerCount alone during resource_gathering/analytics.
type PlayerDisconnected struct {
	PlayerID           string            `json:"playerId"`
	PlayerName         string            `json:"playerName"`
	DisconnectionTime  string            `json:"disconnectionTime"`
	CurrentPhase       string            `json:"currentPhase"`
	UpdatedCounts      *DisconnectCounts `json:"updatedCounts,omitempty"`
	FragmentHandling   *FragmentHandling `json:"fragmentHandling,omitempty"`
	UpdatedPlayerCount *int              `json:"updatedPlayerCount,omitempty"`
}
