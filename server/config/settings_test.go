package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConstants(t *testing.T) {
	t.Run("Host UUID", func(t *testing.T) {
		assert.NotEmpty(t, HostUUID)
		assert.Len(t, HostUUID, 36)       // Standard UUID length
		assert.Contains(t, HostUUID, "-") // Should contain dashes
	})

	t.Run("Station Hashes", func(t *testing.T) {
		hashes := []string{
			HashAnchorStation,
			HashChronosStation,
			HashGuideStation,
			HashClarityStation,
		}

		for _, hash := range hashes {
			assert.NotEmpty(t, hash)
			assert.Contains(t, hash, "STATION")
			assert.Contains(t, hash, "QR_HASH")
			assert.Contains(t, hash, "2024")
		}

		// All hashes should be unique
		hashSet := make(map[string]bool)
		for _, hash := range hashes {
			assert.False(t, hashSet[hash], "Duplicate hash found: %s", hash)
			hashSet[hash] = true
		}
	})

	t.Run("Server Settings", func(t *testing.T) {
		assert.Equal(t, "8080", DefaultPort)
		assert.Greater(t, WebSocketBufferSize, 0)
		assert.Greater(t, MaxMessageSize, 0)
		assert.Greater(t, PingPeriod, 0)
		assert.Greater(t, PongWait, 0)
		assert.Greater(t, WriteWait, 0)

		// Reasonable defaults
		assert.Equal(t, 1024, WebSocketBufferSize)
		assert.Equal(t, 8192, MaxMessageSize)
		assert.Equal(t, 30, PingPeriod)
		assert.Equal(t, 60, PongWait)
		assert.Equal(t, 10, WriteWait)
	})

	t.Run("Puzzle Settings", func(t *testing.T) {
		assert.NotEmpty(t, DefaultPuzzleImage)
		assert.Equal(t, "nature_image", DefaultPuzzleImage)
	})
}

func TestStationEnum(t *testing.T) {
	t.Run("Station constants", func(t *testing.T) {
		stations := []Station{
			AnchorStation,
			ChronosStation,
			GuideStation,
			ClarityStation,
			UnknownStation,
		}

		expectedNames := []string{
			"anchor",
			"chronos",
			"guide",
			"clarity",
			"unknown",
		}

		for i, station := range stations {
			assert.Equal(t, expectedNames[i], string(station))
		}

		// All stations should be unique
		stationSet := make(map[Station]bool)
		for _, station := range stations {
			assert.False(t, stationSet[station], "Duplicate station found: %s", station)
			stationSet[station] = true
		}
	})
}

func TestGetStationFromHash(t *testing.T) {
	tests := []struct {
		name     string
		hash     string
		expected Station
	}{
		{"Anchor station", HashAnchorStation, AnchorStation},
		{"Chronos station", HashChronosStation, ChronosStation},
		{"Guide station", HashGuideStation, GuideStation},
		{"Clarity station", HashClarityStation, ClarityStation},
		{"Unknown hash", "INVALID_HASH", UnknownStation},
		{"Empty hash", "", UnknownStation},
		{"Partial hash", "ANCHOR", UnknownStation},
		{"Wrong format", "anchor_station_qr_hash_2024", UnknownStation},
		{"Case sensitive", "anchor_station_qr_hash_2024", UnknownStation},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetStationFromHash(tt.hash)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetStationFromHashAllStations(t *testing.T) {
	// Test all valid hashes return correct stations
	hashToStation := map[string]Station{
		HashAnchorStation:  AnchorStation,
		HashChronosStation: ChronosStation,
		HashGuideStation:   GuideStation,
		HashClarityStation: ClarityStation,
	}

	for hash, expectedStation := range hashToStation {
		result := GetStationFromHash(hash)
		assert.Equal(t, expectedStation, result, "Hash %s should map to station %s", hash, expectedStation)
	}
}

func TestStationStringValues(t *testing.T) {
	// Test that station string values are as expected
	assert.Equal(t, "anchor", string(AnchorStation))
	assert.Equal(t, "chronos", string(ChronosStation))
	assert.Equal(t, "guide", string(GuideStation))
	assert.Equal(t, "clarity", string(ClarityStation))
	assert.Equal(t, "unknown", string(UnknownStation))
}

func TestHashUniqueness(t *testing.T) {
	// Ensure all hashes are unique
	hashes := []string{
		HashAnchorStation,
		HashChronosStation,
		HashGuideStation,
		HashClarityStation,
	}

	hashSet := make(map[string]bool)
	for _, hash := range hashes {
		assert.False(t, hashSet[hash], "Duplicate hash found: %s", hash)
		hashSet[hash] = true
	}

	assert.Len(t, hashSet, len(hashes), "All hashes should be unique")
}

func TestConstantValues(t *testing.T) {
	// Test specific values that are important for the application
	t.Run("Timeout values are reasonable", func(t *testing.T) {
		assert.True(t, PongWait > PingPeriod, "PongWait should be greater than PingPeriod")
		assert.True(t, WriteWait > 0, "WriteWait should be positive")
		assert.True(t, WriteWait < PingPeriod, "WriteWait should be less than PingPeriod")
	})

	t.Run("Buffer sizes are reasonable", func(t *testing.T) {
		assert.True(t, MaxMessageSize > WebSocketBufferSize, "MaxMessageSize should be larger than WebSocketBufferSize")
		assert.True(t, WebSocketBufferSize >= 512, "WebSocketBufferSize should be at least 512 bytes")
		assert.True(t, MaxMessageSize <= 16384, "MaxMessageSize should not exceed 16KB for reasonable limits")
	})
}
