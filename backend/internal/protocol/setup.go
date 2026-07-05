package protocol

// ── Client → Server ────────────────────────────────────────────────────────

// PlayerConnect is the payload of SETUP_TO_SERVER_PLAYER_CONNECT — empty;
// the token (or its absence) in the envelope's auth carries the meaning.
type PlayerConnect struct{}

func (PlayerConnect) Validate() error { return nil }

// PlayerConfiguration is SETUP_TO_SERVER_PLAYER_CONFIGURATION: role +
// specialties + display name, submitted atomically (also marks ready).
type PlayerConfiguration struct {
	SelectedRole        string   `json:"selectedRole"`
	SelectedSpecialties []string `json:"selectedSpecialties"`
	PlayerName          string   `json:"playerName"`
}

func (p PlayerConfiguration) Validate() error {
	return validateText("playerName", p.PlayerName, 1, MaxPlayerNameLen)
}

// StartGame is SETUP_TO_SERVER_START_GAME (empty payload).
type StartGame struct{}

func (StartGame) Validate() error { return nil }

// ── Server → Client ────────────────────────────────────────────────────────

// HostGameConfig is the config subset echoed in the host handshake.
type HostGameConfig struct {
	MinPlayers              int     `json:"minPlayers"`
	MaxPlayers              int     `json:"maxPlayers"`
	ResourceGatheringRounds int     `json:"resourceGatheringRounds"`
	TriviaAnswerTime        float64 `json:"triviaAnswerTime"`
	TriviaGraceTime         float64 `json:"triviaGraceTime"`
	PuzzleBaseTime          float64 `json:"puzzleBaseTime"`
	DifficultyMode          string  `json:"difficultyMode"`
}

// HostConnectionConfirmed is SETUP_TO_HOST_CONNECTION_CONFIRMED.
type HostConnectionConfirmed struct {
	HostID         string         `json:"hostId"`
	CurrentPhase   string         `json:"currentPhase"`
	IsReconnection bool           `json:"isReconnection"`
	GameConfig     HostGameConfig `json:"gameConfig"`
}

// ExistingConfiguration is the preserved player state in a reconnection
// handshake. SelectedRole is null if the role's slot was lost while
// disconnected (setup phase only).
type ExistingConfiguration struct {
	SelectedRole        *string  `json:"selectedRole"`
	SelectedSpecialties []string `json:"selectedSpecialties"`
	PlayerName          string   `json:"playerName"`
	Ready               bool     `json:"ready"`
}

// PlayerConnectionConfirmed is SETUP_TO_PLAYER_CONNECTION_CONFIRMED.
type PlayerConnectionConfirmed struct {
	PlayerID              string                 `json:"playerId"`
	CurrentPhase          string                 `json:"currentPhase"`
	IsReconnection        bool                   `json:"isReconnection"`
	ExistingConfiguration *ExistingConfiguration `json:"existingConfiguration"`
}

// RoleInfo describes one selectable role in SETUP_TO_PLAYER_ROLES_AVAILABLE.
type RoleInfo struct {
	RoleType       string  `json:"roleType"`
	DisplayName    string  `json:"displayName"`
	ResourceBonus  float64 `json:"resourceBonus"`
	BonusTokenType string  `json:"bonusTokenType"`
	Description    string  `json:"description"`
	Available      bool    `json:"available"`
}

// RolesAvailable is SETUP_TO_PLAYER_ROLES_AVAILABLE (unready players only).
type RolesAvailable struct {
	Roles            []RoleInfo `json:"roles"`
	TriviaCategories []string   `json:"triviaCategories"`
	MaxSpecialties   int        `json:"maxSpecialties"`
}

// LobbyStatus is SETUP_TO_CLIENT_LOBBY_STATUS. PlayerRoles counts only
// configured (ready) players, so it always sums to ReadyPlayers.
type LobbyStatus struct {
	CurrentPlayers    int            `json:"currentPlayers"`
	MinPlayers        int            `json:"minPlayers"`
	MaxPlayers        int            `json:"maxPlayers"`
	PlayerRoles       map[string]int `json:"playerRoles"`
	HasHost           bool           `json:"hasHost"`
	AllPlayersReady   bool           `json:"allPlayersReady"`
	ReadyPlayers      int            `json:"readyPlayers"`
	GameStartEligible bool           `json:"gameStartEligible"`
	WaitingMessage    string         `json:"waitingMessage"`
}

// PlayerStatus is one roster entry in SETUP_TO_HOST_PLAYER_ROSTER.
type PlayerStatus struct {
	PlayerName   string   `json:"playerName"`
	Role         *string  `json:"role"`
	Specialties  []string `json:"specialties"`
	Connected    bool     `json:"connected"`
	Ready        bool     `json:"ready"`
	LastActivity string   `json:"lastActivity"`
}

// PlayerRoster is SETUP_TO_HOST_PLAYER_ROSTER.
type PlayerRoster struct {
	Phase             string                  `json:"phase"`
	ConnectedPlayers  int                     `json:"connectedPlayers"`
	ReadyPlayers      int                     `json:"readyPlayers"`
	GameStartEligible bool                    `json:"gameStartEligible"`
	PlayerStatuses    map[string]PlayerStatus `json:"playerStatuses"`
	RoleDistribution  map[string]int          `json:"roleDistribution"`
}

// TeamTokens is the four-pool token counter used across phases.
type TeamTokens struct {
	AnchorTokens  int `json:"anchorTokens"`
	ChronosTokens int `json:"chronosTokens"`
	GuideTokens   int `json:"guideTokens"`
	ClarityTokens int `json:"clarityTokens"`
}

// GameStarted is SETUP_TO_HOST_GAME_STARTED.
type GameStarted struct {
	Phase             string     `json:"phase"`
	TotalPlayers      int        `json:"totalPlayers"`
	InitialTeamTokens TeamTokens `json:"initialTeamTokens"`
}
