package config

// Error Codes
const (
	ErrorCodeInvalidRole          = "INVALID_ROLE_SELECTION"
	ErrorCodeInsufficientPlayers  = "INSUFFICIENT_PLAYERS"
	ErrorCodeInvalidToken         = "INVALID_TOKEN"
	ErrorCodeRateLimited          = "RATE_LIMITED"
	ErrorCodeInvalidMove          = "INVALID_MOVE"
	ErrorCodeGameInProgress       = "GAME_IN_PROGRESS"
	ErrorCodeHostAlreadyConnected = "HOST_ALREADY_CONNECTED"
)

// Error Messages
const (
	ErrorMessageInvalidRole          = "Selected role is not available"
	ErrorMessageInsufficientPlayers  = "Need at least 4 players to start"
	ErrorMessageInvalidToken         = "Invalid authentication token"
	ErrorMessageRateLimited          = "Too many requests, please wait"
	ErrorMessageInvalidMove          = "Invalid fragment move"
	ErrorMessageGameInProgress       = "Game is already in progress"
	ErrorMessageHostAlreadyConnected = "A host is already connected"
)
