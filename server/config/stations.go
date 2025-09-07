package config

// QR Code Station Hashes - These should match the QR codes at physical stations
const (
	HashAnchorStation  = "anchor"
	HashChronosStation = "chronos"
	HashGuideStation   = "guide"
	HashClarityStation = "clarity"
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
