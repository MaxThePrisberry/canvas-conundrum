package config

// WebSocket Event Names - Client to Server
const (
	// Setup Phase Events
	EventSetupToServerPlayerConfiguration = "SETUP_TO_SERVER_PLAYER_CONFIGURATION"
	EventSetupToServerStartGame           = "SETUP_TO_SERVER_START_GAME"

	// Resource Gathering Phase Events
	EventResourceToServerLocationVerified = "RESOURCE_TO_SERVER_LOCATION_VERIFIED"
	EventResourceToServerTriviaAnswer     = "RESOURCE_TO_SERVER_TRIVIA_ANSWER"

	// Puzzle Assembly Phase Events
	EventPuzzleToServerStartTimer             = "PUZZLE_TO_SERVER_START_TIMER"
	EventPuzzleToServerSegmentCompleted       = "PUZZLE_TO_SERVER_SEGMENT_COMPLETED"
	EventPuzzleToServerFragmentMove           = "PUZZLE_TO_SERVER_FRAGMENT_MOVE"
	EventPuzzleToServerRecommendMove          = "PUZZLE_TO_SERVER_RECOMMEND_MOVE"
	EventPuzzleToServerRecommendationResponse = "PUZZLE_TO_SERVER_RECOMMENDATION_RESPONSE"

	// Analytics Phase Events
	EventAnalyticsToServerResetGame = "ANALYTICS_TO_SERVER_RESET_GAME"

	// System Events
	EventSystemPing = "SYSTEM_PING"
)

// WebSocket Event Names - Server to Client
const (
	// Setup Phase Events
	EventSetupToHostConnectionConfirmed = "SETUP_TO_HOST_CONNECTION_CONFIRMED"
	EventSetupToPlayerRolesAvailable    = "SETUP_TO_PLAYER_ROLES_AVAILABLE"
	EventSetupToClientLobbyStatus       = "SETUP_TO_CLIENT_LOBBY_STATUS"
	EventSetupToHostPlayerRoster        = "SETUP_TO_HOST_PLAYER_ROSTER"
	EventSetupToClientGameStarted       = "SETUP_TO_CLIENT_GAME_STARTED"
	EventSetupToHostGameStarted         = "SETUP_TO_HOST_GAME_STARTED"

	// Resource Gathering Phase Events
	EventResourceToClientPhaseStart     = "RESOURCE_TO_CLIENT_PHASE_START"
	EventResourceToHostPhaseStart       = "RESOURCE_TO_HOST_PHASE_START"
	EventResourceToPlayerTriviaQuestion = "RESOURCE_TO_PLAYER_TRIVIA_QUESTION"
	EventResourceToPlayerAnswerResult   = "RESOURCE_TO_PLAYER_ANSWER_RESULT"
	EventResourceToClientTeamProgress   = "RESOURCE_TO_CLIENT_TEAM_PROGRESS"
	EventResourceToHostRoundAnalytics   = "RESOURCE_TO_HOST_ROUND_ANALYTICS"
	EventResourceToClientPhaseComplete  = "RESOURCE_TO_CLIENT_PHASE_COMPLETE"
	EventResourceToHostPhaseComplete    = "RESOURCE_TO_HOST_PHASE_COMPLETE"

	// Puzzle Assembly Phase Events
	EventPuzzleToClientPhaseLoad             = "PUZZLE_TO_CLIENT_PHASE_LOAD"
	EventPuzzleToHostPhaseLoad               = "PUZZLE_TO_HOST_PHASE_LOAD"
	EventPuzzleToClientTimerStart            = "PUZZLE_TO_CLIENT_TIMER_START"
	EventPuzzleToHostTimerStart              = "PUZZLE_TO_HOST_TIMER_START"
	EventPuzzleToPlayerSegmentAcknowledged   = "PUZZLE_TO_PLAYER_SEGMENT_ACKNOWLEDGED"
	EventPuzzleToHostSegmentCompleted        = "PUZZLE_TO_HOST_SEGMENT_COMPLETED"
	EventPuzzleToPlayerPersonalState         = "PUZZLE_TO_PLAYER_PERSONAL_STATE"
	EventPuzzleToPlayerMoveResult            = "PUZZLE_TO_PLAYER_MOVE_RESULT"
	EventPuzzleToClientGridState             = "PUZZLE_TO_CLIENT_GRID_STATE"
	EventPuzzleToHostGridState               = "PUZZLE_TO_HOST_GRID_STATE"
	EventPuzzleToPlayerMoveRecommendation    = "PUZZLE_TO_PLAYER_MOVE_RECOMMENDATION"
	EventPuzzleToPlayerRecommendationResult  = "PUZZLE_TO_PLAYER_RECOMMENDATION_RESULT"
	EventPuzzleToPlayerRecommendationExpired = "PUZZLE_TO_PLAYER_RECOMMENDATION_EXPIRED"
	EventPuzzleToClientCompletedSuccess      = "PUZZLE_TO_CLIENT_COMPLETED_SUCCESS"
	EventPuzzleToClientCompletedTimeout      = "PUZZLE_TO_CLIENT_COMPLETED_TIMEOUT"
	EventPuzzleToHostCompletionAnalytics     = "PUZZLE_TO_HOST_COMPLETION_ANALYTICS"

	// Analytics Phase Events
	EventAnalyticsToPlayerPersonalReport = "ANALYTICS_TO_PLAYER_PERSONAL_REPORT"
	EventAnalyticsToClientTeamSummary    = "ANALYTICS_TO_CLIENT_TEAM_SUMMARY"
	EventAnalyticsToHostCompleteReport   = "ANALYTICS_TO_HOST_COMPLETE_REPORT"
	EventAnalyticsToClientGameReset      = "ANALYTICS_TO_CLIENT_GAME_RESET"

	// System-Wide Events
	EventSystemToClientError                = "SYSTEM_TO_CLIENT_ERROR"
	EventSystemToHostError                  = "SYSTEM_TO_HOST_ERROR"
	EventSystemToClientDisconnectionWarning = "SYSTEM_TO_CLIENT_DISCONNECTION_WARNING"
	EventSystemToClientHostDisconnected     = "SYSTEM_TO_CLIENT_HOST_DISCONNECTED"
	EventSystemToClientHostReconnected      = "SYSTEM_TO_CLIENT_HOST_RECONNECTED"
	EventSystemToHostPlayerDisconnected     = "SYSTEM_TO_HOST_PLAYER_DISCONNECTED"
	EventSystemPong                         = "SYSTEM_PONG"
	EventSystemToClientPhaseTransition      = "SYSTEM_TO_CLIENT_PHASE_TRANSITION"
	EventSystemToHostPhaseTransition        = "SYSTEM_TO_HOST_PHASE_TRANSITION"
)
