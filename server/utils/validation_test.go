package utils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidatePlayerName(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		description string
	}{
		{"Valid name", "TestPlayer", false, "valid player name"},
		{"Valid name with spaces", "Test Player", false, "valid name with spaces"},
		{"Valid name with numbers", "Player123", false, "valid name with numbers"},
		{"Valid name with special chars", "Player-_.", false, "valid name with special characters"},
		{"Empty string", "", true, "empty string should fail"},
		{"Only spaces", "   ", true, "only spaces should fail"},
		{"Too long", strings.Repeat("a", MaxPlayerNameLength+1), true, "too long name should fail"},
		{"Control character", "Test\x00Player", true, "control character should fail"},
		{"Control character (tab)", "Test\tPlayer", true, "tab character should fail"},
		{"Control character (newline)", "Test\nPlayer", true, "newline character should fail"},
		{"Delete character", "Test\x7FPlayer", true, "delete character should fail"},
		{"Valid Unicode", "TestПлеер", false, "valid Unicode should pass"},
		{"Minimum length", "A", false, "minimum length should pass"},
		{"Maximum length", strings.Repeat("a", MaxPlayerNameLength), false, "maximum length should pass"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePlayerName(tt.input)
			if tt.expectError {
				assert.Error(t, err, tt.description)
			} else {
				assert.NoError(t, err, tt.description)
			}
		})
	}
}

func TestValidateRole(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
	}{
		{"Art enthusiast", "art_enthusiast", false},
		{"Detective", "detective", false},
		{"Tourist", "tourist", false},
		{"Janitor", "janitor", false},
		{"Invalid role", "invalid_role", true},
		{"Empty string", "", true},
		{"Wrong case", "ART_ENTHUSIAST", true},
		{"Partial match", "art", true},
		{"With spaces", "art enthusiast", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRole(tt.input)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateSpecialties(t *testing.T) {
	tests := []struct {
		name        string
		input       []string
		expectError bool
		description string
	}{
		{"Valid single specialty", []string{"science"}, false, "single valid specialty"},
		{"Valid multiple specialties", []string{"science", "history"}, false, "multiple valid specialties"},
		{"Valid all specialties", []string{"general", "geography", "history"}, false, "all valid specialties within limit"},
		{"Empty array", []string{}, false, "empty array should be valid"},
		{"Too many specialties", []string{"science", "history", "geography", "music"}, true, "too many specialties"},
		{"Invalid specialty", []string{"invalid"}, true, "invalid specialty name"},
		{"Duplicate specialty", []string{"science", "science"}, true, "duplicate specialty"},
		{"Valid and invalid mix", []string{"science", "invalid"}, true, "mix of valid and invalid"},
		{"Case sensitive", []string{"SCIENCE"}, true, "wrong case should fail"},
		{"All valid categories", []string{"general", "geography", "history"}, false, "all valid categories"},
		{"Maximum allowed", []string{"science", "history", "music"}, false, "maximum allowed specialties"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSpecialties(tt.input)
			if tt.expectError {
				assert.Error(t, err, tt.description)
			} else {
				assert.NoError(t, err, tt.description)
			}
		})
	}
}

func TestSanitizeString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Normal string", "Hello World", "Hello World"},
		{"String with tabs", "Hello\tWorld", "HelloWorld"},
		{"String with newlines", "Hello\nWorld", "HelloWorld"},
		{"String with control chars", "Hello\x00\x01World", "HelloWorld"},
		{"String with leading/trailing spaces", "  Hello World  ", "Hello World"},
		{"String with delete char", "Hello\x7FWorld", "HelloWorld"},
		{"Empty string", "", ""},
		{"Only control chars", "\x00\x01\x02", ""},
		{"Only spaces", "   ", ""},
		{"Unicode string", "Привет мир", "Привет мир"},
		{"Mixed content", "\x00  Hello\tWorld\n  \x7F", "HelloWorld"},
		{"String with valid special chars", "Hello-World_123!@#", "Hello-World_123!@#"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidationConstants(t *testing.T) {
	// Test that constants are reasonable values
	assert.Greater(t, MaxPlayerNameLength, MinPlayerNameLength)
	assert.Greater(t, MaxMessageLength, 0)
	assert.Greater(t, MaxSpecialties, 0)
	assert.GreaterOrEqual(t, MinPlayerNameLength, 1)
	assert.GreaterOrEqual(t, MinTimeElapsed, 0.0)
	assert.Greater(t, MaxTimeElapsed, MinTimeElapsed)
	assert.GreaterOrEqual(t, MinCoordinateValue, 0)
	assert.Greater(t, MaxCoordinateValue, MinCoordinateValue)
}

func TestValidatePlayerNameEdgeCases(t *testing.T) {
	// Test various Unicode scenarios
	t.Run("Invalid UTF-8", func(t *testing.T) {
		invalidUTF8 := string([]byte{0xff, 0xfe, 0xfd})
		err := ValidatePlayerName(invalidUTF8)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid UTF-8")
	})

	t.Run("Exactly minimum length", func(t *testing.T) {
		name := strings.Repeat("a", MinPlayerNameLength)
		err := ValidatePlayerName(name)
		assert.NoError(t, err)
	})

	t.Run("Exactly maximum length", func(t *testing.T) {
		name := strings.Repeat("a", MaxPlayerNameLength)
		err := ValidatePlayerName(name)
		assert.NoError(t, err)
	})
}

func TestValidateSpecialtiesEdgeCases(t *testing.T) {
	t.Run("Exactly max specialties", func(t *testing.T) {
		specialties := []string{"science", "history", "geography"}
		err := ValidateSpecialties(specialties)
		assert.NoError(t, err)
	})

	t.Run("All valid categories", func(t *testing.T) {
		allValid := []string{"general", "geography", "history"}
		err := ValidateSpecialties(allValid)
		assert.NoError(t, err)
	})

	t.Run("Case sensitivity", func(t *testing.T) {
		mixedCase := []string{"Science", "HISTORY"}
		err := ValidateSpecialties(mixedCase)
		assert.Error(t, err)
	})
}