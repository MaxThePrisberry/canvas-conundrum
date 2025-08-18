package utils

import (
	"github.com/google/uuid"
)

// GenerateUUID generates a new UUID v4
func GenerateUUID() string {
	return uuid.New().String()
}

// GeneratePlayerID generates a new player ID
func GeneratePlayerID() string {
	return GenerateUUID()
}

// GenerateFragmentID generates a new fragment ID
func GenerateFragmentID() string {
	return "fragment-" + GenerateUUID()
}

// GenerateMoveID generates a new move/recommendation ID
func GenerateMoveID() string {
	return "move-" + GenerateUUID()
}
