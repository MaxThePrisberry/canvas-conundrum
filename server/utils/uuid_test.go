package utils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateUUID(t *testing.T) {
	uuid1 := GenerateUUID()
	uuid2 := GenerateUUID()

	// UUIDs should be different
	assert.NotEqual(t, uuid1, uuid2)

	// UUIDs should be valid format (36 characters with dashes)
	assert.Len(t, uuid1, 36)
	assert.Len(t, uuid2, 36)

	// Should contain dashes in correct positions
	assert.Equal(t, "-", string(uuid1[8]))
	assert.Equal(t, "-", string(uuid1[13]))
	assert.Equal(t, "-", string(uuid1[18]))
	assert.Equal(t, "-", string(uuid1[23]))
}

func TestGeneratePlayerID(t *testing.T) {
	playerID1 := GeneratePlayerID()
	playerID2 := GeneratePlayerID()

	// Player IDs should be different
	assert.NotEqual(t, playerID1, playerID2)

	// Should be valid UUID format (36 characters with dashes)
	assert.Len(t, playerID1, 36)
	assert.Len(t, playerID2, 36)

	// Should contain dashes in correct positions
	assert.Equal(t, "-", string(playerID1[8]))
	assert.Equal(t, "-", string(playerID1[13]))
	assert.Equal(t, "-", string(playerID1[18]))
	assert.Equal(t, "-", string(playerID1[23]))
}

func TestGenerateFragmentID(t *testing.T) {
	fragmentID1 := GenerateFragmentID()
	fragmentID2 := GenerateFragmentID()

	// Fragment IDs should be different
	assert.NotEqual(t, fragmentID1, fragmentID2)

	// Should start with "fragment-"
	assert.True(t, strings.HasPrefix(fragmentID1, "fragment-"))
	assert.True(t, strings.HasPrefix(fragmentID2, "fragment-"))

	// Should be longer than just the prefix
	assert.Greater(t, len(fragmentID1), 9)
	assert.Greater(t, len(fragmentID2), 9)

	// The UUID part should be valid format
	uuidPart := fragmentID1[9:] // Remove "fragment-" prefix
	assert.Len(t, uuidPart, 36)
}

func TestGenerateMoveID(t *testing.T) {
	moveID1 := GenerateMoveID()
	moveID2 := GenerateMoveID()

	// Move IDs should be different
	assert.NotEqual(t, moveID1, moveID2)

	// Should start with "move-"
	assert.True(t, strings.HasPrefix(moveID1, "move-"))
	assert.True(t, strings.HasPrefix(moveID2, "move-"))

	// Should be longer than just the prefix
	assert.Greater(t, len(moveID1), 5)
	assert.Greater(t, len(moveID2), 5)

	// The UUID part should be valid format
	uuidPart := moveID1[5:] // Remove "move-" prefix
	assert.Len(t, uuidPart, 36)
}

func TestIDUniqueness(t *testing.T) {
	// Test that different ID generators produce unique IDs
	playerID := GeneratePlayerID()
	fragmentID := GenerateFragmentID()
	moveID := GenerateMoveID()
	uuid := GenerateUUID()

	// All should be different
	assert.NotEqual(t, playerID, fragmentID)
	assert.NotEqual(t, playerID, moveID)
	assert.NotEqual(t, playerID, uuid)
	assert.NotEqual(t, fragmentID, moveID)
	assert.NotEqual(t, fragmentID, uuid)
	assert.NotEqual(t, moveID, uuid)
}

func TestGenerateMultipleUUIDs(t *testing.T) {
	// Generate multiple UUIDs and ensure they're all unique
	uuids := make(map[string]bool)
	count := 1000

	for i := 0; i < count; i++ {
		uuid := GenerateUUID()
		assert.False(t, uuids[uuid], "Duplicate UUID found: %s", uuid)
		uuids[uuid] = true
	}

	assert.Len(t, uuids, count)
}
