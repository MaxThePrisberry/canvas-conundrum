package config

const (
	// Host UUID - Static identifier for host connections
	HostUUID = "550e8400-e29b-41d4-a716-446655440000"

	// Server Settings
	DefaultPort         = "8080"
	WebSocketBufferSize = 1024
	MaxMessageSize      = 8192 // 8KB limit for WebSocket messages
	PingPeriod          = 30   // seconds
	PongWait            = 60   // seconds
	WriteWait           = 10   // seconds
)
