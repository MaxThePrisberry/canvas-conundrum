package protocol

// ErrorCode is a stable identifier from the error-code registry, shared by
// WebSocket error events and HTTP error bodies.
type ErrorCode string

const (
	ErrUnauthorized                 ErrorCode = "UNAUTHORIZED"
	ErrMalformedRequest             ErrorCode = "MALFORMED_REQUEST"
	ErrMalformedPayload             ErrorCode = "MALFORMED_PAYLOAD"
	ErrInvalidRoleSelection         ErrorCode = "INVALID_ROLE_SELECTION"
	ErrInvalidSpecialtySelection    ErrorCode = "INVALID_SPECIALTY_SELECTION"
	ErrRoleFull                     ErrorCode = "ROLE_FULL"
	ErrConfigurationLocked          ErrorCode = "CONFIGURATION_LOCKED"
	ErrInvalidStationHash           ErrorCode = "INVALID_STATION_HASH"
	ErrInsufficientPlayers          ErrorCode = "INSUFFICIENT_PLAYERS"
	ErrCooldownActive               ErrorCode = "COOLDOWN_ACTIVE"
	ErrRecommendationPending        ErrorCode = "RECOMMENDATION_PENDING"
	ErrForbiddenPhase               ErrorCode = "FORBIDDEN_PHASE"
	ErrForbiddenNotOwner            ErrorCode = "FORBIDDEN_NOT_OWNER"
	ErrForbiddenPreviewWindowClosed ErrorCode = "FORBIDDEN_PREVIEW_WINDOW_CLOSED"
	ErrNotFound                     ErrorCode = "NOT_FOUND"
)

// Error types — the coarse category in error payloads, for log filtering.
const (
	ErrorTypeAuth       = "auth_error"
	ErrorTypeValidation = "validation_error"
	ErrorTypeGameState  = "game_state_error"
)

// ErrorPayload is the payload of SYSTEM_TO_CLIENT_ERROR and
// SYSTEM_TO_HOST_ERROR. Details, Context, and SuggestedActions are optional.
type ErrorPayload struct {
	ErrorType        string    `json:"errorType"`
	ErrorCode        ErrorCode `json:"errorCode"`
	Message          string    `json:"message"`
	Details          string    `json:"details,omitempty"`
	Context          any       `json:"context,omitempty"`
	SuggestedActions []string  `json:"suggestedActions,omitempty"`
}

// WebSocket close codes (websocket-events.md § WebSocket close codes).
// 4001–4003 are terminal: clients must not auto-reconnect after them.
const (
	CloseNormal             = 1000 // reset, graceful disconnect, superseded host socket
	CloseGoingAway          = 1001 // planned shutdown
	CloseUnauthorized       = 4001 // bad/unknown/invalidated token, or no connect frame in time
	CloseJoinRejected       = 4002 // new join past setup or at maxPlayers
	CloseReconnectForbidden = 4003 // player reconnection during puzzle_assembly
)
