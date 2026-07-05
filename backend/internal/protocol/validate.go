package protocol

import (
	"fmt"
	"unicode/utf8"
)

// Free-text length limits (websocket-events.md § Message limits). Limits
// count characters (runes), and all text must be valid UTF-8.
const (
	MaxPlayerNameLen  = 32
	MaxReasoningLen   = 200
	MaxStationHashLen = 128
)

func validateText(field, value string, minLen, maxLen int) error {
	if !utf8.ValidString(value) {
		return fmt.Errorf("%s is not valid UTF-8", field)
	}
	n := utf8.RuneCountInString(value)
	if n < minLen || n > maxLen {
		return fmt.Errorf("%s must be %d-%d characters, got %d", field, minLen, maxLen, n)
	}
	return nil
}
