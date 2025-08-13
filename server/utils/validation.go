package utils

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	// Input size limits
	MaxPlayerNameLength  = 50
	MaxMessageLength     = 1000
	MaxReasoningLength   = 500
	MaxStationNameLength = 50
	MaxQuestionIDLength  = 100
	MaxAnswerLength      = 500
	MaxFragmentIDLength  = 100
	MaxSegmentIDLength   = 10
	MinPlayerNameLength  = 1

	// Array size limits
	MaxSpecialties     = 3
	MaxRolesPerMessage = 4

	// Numeric limits
	MaxTimeElapsed     = 3600.0 // 1 hour max
	MinTimeElapsed     = 0.0
	MaxCoordinateValue = 100
	MinCoordinateValue = 0
)

// ValidatePlayerName validates a player name
func ValidatePlayerName(name string) error {
	if !utf8.ValidString(name) {
		return fmt.Errorf("player name contains invalid UTF-8 characters")
	}

	trimmed := strings.TrimSpace(name)
	if len(trimmed) < MinPlayerNameLength {
		return fmt.Errorf("player name is too short (minimum %d characters)", MinPlayerNameLength)
	}
	if len(trimmed) > MaxPlayerNameLength {
		return fmt.Errorf("player name is too long (maximum %d characters)", MaxPlayerNameLength)
	}

	// Check for control characters
	for _, r := range trimmed {
		if r < 32 || r == 127 { // ASCII control characters
			return fmt.Errorf("player name contains invalid control characters")
		}
	}

	return nil
}

// ValidateRole validates a role string
func ValidateRole(role string) error {
	validRoles := map[string]bool{
		"art_enthusiast": true,
		"detective":      true,
		"tourist":        true,
		"janitor":        true,
	}

	if !validRoles[role] {
		return fmt.Errorf("invalid role: %s", role)
	}

	return nil
}

// ValidateSpecialties validates specialties array
func ValidateSpecialties(specialties []string) error {
	if len(specialties) > MaxSpecialties {
		return fmt.Errorf("too many specialties (maximum %d)", MaxSpecialties)
	}

	validCategories := map[string]bool{
		"general":     true,
		"geography":   true,
		"history":     true,
		"music":       true,
		"science":     true,
		"video_games": true,
	}

	for _, specialty := range specialties {
		if !validCategories[specialty] {
			return fmt.Errorf("invalid specialty: %s", specialty)
		}
	}

	// Check for duplicates
	seen := make(map[string]bool)
	for _, specialty := range specialties {
		if seen[specialty] {
			return fmt.Errorf("duplicate specialty: %s", specialty)
		}
		seen[specialty] = true
	}

	return nil
}

// SanitizeString removes control characters and trims whitespace
func SanitizeString(s string) string {
	// Remove control characters
	var result strings.Builder
	for _, r := range s {
		if r >= 32 && r != 127 { // Skip ASCII control characters
			result.WriteRune(r)
		}
	}

	// Trim whitespace
	return strings.TrimSpace(result.String())
}
