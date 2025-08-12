package utils

import (
	"github.com/google/uuid"
)

// GenerateUUID generates a new UUID v4
func GenerateUUID() string {
	return uuid.New().String()
}

// ValidateUUID checks if a string is a valid UUID
func ValidateUUID(id string) bool {
	_, err := uuid.Parse(id)
	return err == nil
}

// GeneratePlayerID generates a new player ID
func GeneratePlayerID() string {
	return "player-" + GenerateUUID()
}

// GenerateFragmentID generates a new fragment ID
func GenerateFragmentID() string {
	return "fragment-" + GenerateUUID()
}

// GenerateMoveID generates a new move/recommendation ID
func GenerateMoveID() string {
	return "move-" + GenerateUUID()
}
