// Package protocol defines the wire format from websocket-events.md: event
// names, the auth/envelope framing, error codes, close codes, and one typed
// payload struct per event.
package protocol

// EventType names a WebSocket event. Values are copied verbatim from the
// Event Index in websocket-events.md.
type EventType string

const (
	// Phase 0: Connection and Setup
	SetupToHostConnectionConfirmed   EventType = "SETUP_TO_HOST_CONNECTION_CONFIRMED"
	SetupToServerPlayerConnect       EventType = "SETUP_TO_SERVER_PLAYER_CONNECT"
	SetupToPlayerConnectionConfirmed EventType = "SETUP_TO_PLAYER_CONNECTION_CONFIRMED"
	SetupToPlayerRolesAvailable      EventType = "SETUP_TO_PLAYER_ROLES_AVAILABLE"
	SetupToServerPlayerConfiguration EventType = "SETUP_TO_SERVER_PLAYER_CONFIGURATION"
	SetupToClientLobbyStatus         EventType = "SETUP_TO_CLIENT_LOBBY_STATUS"
	SetupToHostPlayerRoster          EventType = "SETUP_TO_HOST_PLAYER_ROSTER"
	SetupToServerStartGame           EventType = "SETUP_TO_SERVER_START_GAME"
	SetupToHostGameStarted           EventType = "SETUP_TO_HOST_GAME_STARTED"

	// Phase 1: Resource Gathering
	ResourceToClientPhaseStart        EventType = "RESOURCE_TO_CLIENT_PHASE_START"
	ResourceToHostPhaseStart          EventType = "RESOURCE_TO_HOST_PHASE_START"
	ResourceToServerLocationVerified  EventType = "RESOURCE_TO_SERVER_LOCATION_VERIFIED"
	ResourceToPlayerLocationConfirmed EventType = "RESOURCE_TO_PLAYER_LOCATION_CONFIRMED"
	ResourceToPlayerTriviaQuestion    EventType = "RESOURCE_TO_PLAYER_TRIVIA_QUESTION"
	ResourceToServerTriviaAnswer      EventType = "RESOURCE_TO_SERVER_TRIVIA_ANSWER"
	ResourceToPlayerAnswerResult      EventType = "RESOURCE_TO_PLAYER_ANSWER_RESULT"
	ResourceToClientTeamProgress      EventType = "RESOURCE_TO_CLIENT_TEAM_PROGRESS"
	ResourceToHostRoundAnalytics      EventType = "RESOURCE_TO_HOST_ROUND_ANALYTICS"
	ResourceToClientPhaseComplete     EventType = "RESOURCE_TO_CLIENT_PHASE_COMPLETE"
	ResourceToHostPhaseComplete       EventType = "RESOURCE_TO_HOST_PHASE_COMPLETE"

	// Puzzle Preparation and Assembly
	PuzzleToHostPreparing                EventType = "PUZZLE_TO_HOST_PREPARING"
	PuzzleToHostReady                    EventType = "PUZZLE_TO_HOST_READY"
	PuzzleToClientPhaseLoad              EventType = "PUZZLE_TO_CLIENT_PHASE_LOAD"
	PuzzleToHostPhaseLoad                EventType = "PUZZLE_TO_HOST_PHASE_LOAD"
	PuzzleToServerPhaseStart             EventType = "PUZZLE_TO_SERVER_PHASE_START"
	PuzzleToClientPhaseStart             EventType = "PUZZLE_TO_CLIENT_PHASE_START"
	PuzzleToHostPhaseStart               EventType = "PUZZLE_TO_HOST_PHASE_START"
	PuzzleToClientPreviewExpired         EventType = "PUZZLE_TO_CLIENT_PREVIEW_EXPIRED"
	PuzzleToServerSegmentCompleted       EventType = "PUZZLE_TO_SERVER_SEGMENT_COMPLETED"
	PuzzleToPlayerSegmentAcknowledged    EventType = "PUZZLE_TO_PLAYER_SEGMENT_ACKNOWLEDGED"
	PuzzleToHostSegmentCompleted         EventType = "PUZZLE_TO_HOST_SEGMENT_COMPLETED"
	PuzzleToServerFragmentMove           EventType = "PUZZLE_TO_SERVER_FRAGMENT_MOVE"
	PuzzleToPlayerMoveResult             EventType = "PUZZLE_TO_PLAYER_MOVE_RESULT"
	PuzzleToClientGridState              EventType = "PUZZLE_TO_CLIENT_GRID_STATE"
	PuzzleToHostGridState                EventType = "PUZZLE_TO_HOST_GRID_STATE"
	PuzzleToPlayerPersonalState          EventType = "PUZZLE_TO_PLAYER_PERSONAL_STATE"
	PuzzleToServerRecommendMove          EventType = "PUZZLE_TO_SERVER_RECOMMEND_MOVE"
	PuzzleToPlayerMoveRecommendation     EventType = "PUZZLE_TO_PLAYER_MOVE_RECOMMENDATION"
	PuzzleToServerRecommendationResponse EventType = "PUZZLE_TO_SERVER_RECOMMENDATION_RESPONSE"
	PuzzleToPlayerRecommendationResult   EventType = "PUZZLE_TO_PLAYER_RECOMMENDATION_RESULT"
	PuzzleToPlayerRecommendationExpired  EventType = "PUZZLE_TO_PLAYER_RECOMMENDATION_EXPIRED"
	PuzzleToClientCompletedSuccess       EventType = "PUZZLE_TO_CLIENT_COMPLETED_SUCCESS"
	PuzzleToClientCompletedTimeout       EventType = "PUZZLE_TO_CLIENT_COMPLETED_TIMEOUT"
	PuzzleToHostCompletionAnalytics      EventType = "PUZZLE_TO_HOST_COMPLETION_ANALYTICS"

	// Phase 3: Analytics
	AnalyticsToPlayerPersonalReport EventType = "ANALYTICS_TO_PLAYER_PERSONAL_REPORT"
	AnalyticsToClientTeamSummary    EventType = "ANALYTICS_TO_CLIENT_TEAM_SUMMARY"
	AnalyticsToHostCompleteReport   EventType = "ANALYTICS_TO_HOST_COMPLETE_REPORT"
	AnalyticsToServerResetGame      EventType = "ANALYTICS_TO_SERVER_RESET_GAME"
	AnalyticsToClientGameReset      EventType = "ANALYTICS_TO_CLIENT_GAME_RESET"

	// System-wide
	SystemToClientError            EventType = "SYSTEM_TO_CLIENT_ERROR"
	SystemToHostError              EventType = "SYSTEM_TO_HOST_ERROR"
	SystemToClientHostDisconnected EventType = "SYSTEM_TO_CLIENT_HOST_DISCONNECTED"
	SystemToClientHostReconnected  EventType = "SYSTEM_TO_CLIENT_HOST_RECONNECTED"
	SystemToHostPlayerDisconnected EventType = "SYSTEM_TO_HOST_PLAYER_DISCONNECTED"
	SystemPing                     EventType = "SYSTEM_PING"
	SystemPong                     EventType = "SYSTEM_PONG"
)

// Game phase identifiers (the `currentPhase` / `phase` enum).
const (
	PhaseSetup             = "setup"
	PhaseResourceGathering = "resource_gathering"
	PhasePuzzlePreparation = "puzzle_preparation"
	PhasePuzzleAssembly    = "puzzle_assembly"
	PhaseAnalytics         = "analytics"
)
