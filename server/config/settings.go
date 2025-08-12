package config

const (
	// Host UUID - Static identifier for host connections
	HostUUID = "550e8400-e29b-41d4-a716-446655440000"

	// QR Code Station Hashes - These should match the QR codes at physical stations
	HashAnchorStation  = "ANCHOR_STATION_QR_HASH_2024"
	HashChronosStation = "CHRONOS_STATION_QR_HASH_2024"
	HashGuideStation   = "GUIDE_STATION_QR_HASH_2024"
	HashClarityStation = "CLARITY_STATION_QR_HASH_2024"

	// Puzzle Image Settings
	DefaultPuzzleImage = "nature_image"
	PuzzleImagesPath   = "./puzzle_images/puzzle_segments"

	// Server Settings
	DefaultPort         = "8080"
	WebSocketBufferSize = 1024
	MaxMessageSize      = 8192 // 8KB limit for WebSocket messages
	PingPeriod          = 30   // seconds
	PongWait            = 60   // seconds
	WriteWait           = 10   // seconds
)

// Station represents a resource gathering station
type Station string

const (
	AnchorStation  Station = "anchor"
	ChronosStation Station = "chronos"
	GuideStation   Station = "guide"
	ClarityStation Station = "clarity"
	UnknownStation Station = "unknown"
)

// GetStationFromHash returns the station type based on QR code hash
func GetStationFromHash(hash string) Station {
	switch hash {
	case HashAnchorStation:
		return AnchorStation
	case HashChronosStation:
		return ChronosStation
	case HashGuideStation:
		return GuideStation
	case HashClarityStation:
		return ClarityStation
	default:
		return UnknownStation
	}
}

// GetHashFromStation returns the QR code hash for a station
func GetHashFromStation(station Station) string {
	switch station {
	case AnchorStation:
		return HashAnchorStation
	case ChronosStation:
		return HashChronosStation
	case GuideStation:
		return HashGuideStation
	case ClarityStation:
		return HashClarityStation
	default:
		return ""
	}
}
