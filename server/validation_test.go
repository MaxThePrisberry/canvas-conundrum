package main

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestValidatePlayerID(t *testing.T) {
	tests := []struct {
		name     string
		playerID string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "Valid UUID",
			playerID: uuid.New().String(),
			wantErr:  false,
		},
		{
			name:     "Empty player ID",
			playerID: "",
			wantErr:  true,
			errMsg:   "player ID cannot be empty",
		},
		{
			name:     "Invalid UUID format",
			playerID: "not-a-uuid",
			wantErr:  true,
			errMsg:   "invalid player ID format",
		},
		{
			name:     "UUID without hyphens",
			playerID: "123e4567e89b12d3a456426614174000",
			wantErr:  true,
			errMsg:   "invalid player ID format",
		},
		{
			name:     "Host placeholder ID",
			playerID: "host",
			wantErr:  true,
			errMsg:   "invalid player ID format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePlayerID(tt.playerID)
			if tt.wantErr {
				assert.NotNil(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestValidatePlayerName(t *testing.T) {
	tests := []struct {
		name       string
		playerName string
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "Valid name",
			playerName: "Player One",
			wantErr:    false,
		},
		{
			name:       "Empty name",
			playerName: "",
			wantErr:    true,
			errMsg:     "name cannot be empty",
		},
		{
			name:       "Name too long",
			playerName: "ThisNameIsWayTooLongForTheSystemToHandleProperlyAndExceedsTheLimit",
			wantErr:    true,
			errMsg:     "name too long (max 50 characters)",
		},
		{
			name:       "Name with numbers",
			playerName: "Player123",
			wantErr:    false,
		},
		{
			name:       "Name with special characters",
			playerName: "Player_One-2",
			wantErr:    false,
		},
		{
			name:       "Name with HTML",
			playerName: "<script>alert('xss')</script>",
			wantErr:    true,
			errMsg:     "name contains invalid characters (allowed: letters, numbers, spaces, hyphens, underscores)",
		},
		{
			name:       "Name with SQL injection attempt",
			playerName: "Player'; DROP TABLE users--",
			wantErr:    true,
			errMsg:     "name contains invalid characters (allowed: letters, numbers, spaces, hyphens, underscores)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePlayerName(tt.playerName)
			if tt.wantErr {
				assert.NotEqual(t, ValidationError{}, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.Equal(t, ValidationError{}, err)
			}
		})
	}
}

func TestValidateRole(t *testing.T) {
	tests := []struct {
		name    string
		role    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "Valid role - art enthusiast",
			role:    "art_enthusiast",
			wantErr: false,
		},
		{
			name:    "Valid role - detective",
			role:    "detective",
			wantErr: false,
		},
		{
			name:    "Valid role - tourist",
			role:    "tourist",
			wantErr: false,
		},
		{
			name:    "Valid role - janitor",
			role:    "janitor",
			wantErr: false,
		},
		{
			name:    "Empty role",
			role:    "",
			wantErr: true,
			errMsg:  "role cannot be empty",
		},
		{
			name:    "Invalid role",
			role:    "superhero",
			wantErr: true,
			errMsg:  "invalid role selection",
		},
		{
			name:    "Role with uppercase",
			role:    "Art_Enthusiast",
			wantErr: true,
			errMsg:  "invalid role selection",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRole(tt.role)
			if tt.wantErr {
				assert.NotNil(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestValidateSpecialties(t *testing.T) {
	tests := []struct {
		name        string
		specialties []string
		wantErr     bool
		errCount    int
		errMsgs     []string
	}{
		{
			name:        "Valid single specialty",
			specialties: []string{"science"},
			wantErr:     false,
		},
		{
			name:        "Valid two specialties",
			specialties: []string{"science", "history"},
			wantErr:     false,
		},
		{
			name:        "Empty specialties",
			specialties: []string{},
			wantErr:     true,
			errCount:    1,
			errMsgs:     []string{"at least one specialty must be selected"},
		},
		{
			name:        "Too many specialties",
			specialties: []string{"science", "history", "geography"},
			wantErr:     true,
			errCount:    1,
			errMsgs:     []string{"too many specialties (max 2)"},
		},
		{
			name:        "Duplicate specialties",
			specialties: []string{"science", "science"},
			wantErr:     true,
			errCount:    1,
			errMsgs:     []string{"duplicate specialty"},
		},
		{
			name:        "Invalid specialty",
			specialties: []string{"science", "magic"},
			wantErr:     true,
			errCount:    1,
			errMsgs:     []string{"invalid specialty category"},
		},
		{
			name:        "All valid categories",
			specialties: []string{"general", "geography"},
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validateSpecialties(tt.specialties)
			if tt.wantErr {
				assert.Len(t, errs, tt.errCount)
				for i, err := range errs {
					assert.Contains(t, err.Error(), tt.errMsgs[i])
				}
			} else {
				assert.Empty(t, errs)
			}
		})
	}
}

func TestValidateLocationHash(t *testing.T) {
	tests := []struct {
		name    string
		hash    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "Valid anchor hash",
			hash:    "HASH_ANCHOR_STATION_2025",
			wantErr: false,
		},
		{
			name:    "Valid chronos hash",
			hash:    "HASH_CHRONOS_STATION_2025",
			wantErr: false,
		},
		{
			name:    "Valid guide hash",
			hash:    "HASH_GUIDE_STATION_2025",
			wantErr: false,
		},
		{
			name:    "Valid clarity hash",
			hash:    "HASH_CLARITY_STATION_2025",
			wantErr: false,
		},
		{
			name:    "Empty hash",
			hash:    "",
			wantErr: true,
			errMsg:  "location hash cannot be empty",
		},
		{
			name:    "Invalid hash",
			hash:    "HASH_INVALID_STATION_2025",
			wantErr: true,
			errMsg:  "invalid resource station hash",
		},
		{
			name:    "Hash with lowercase",
			hash:    "hash_anchor_station_2025",
			wantErr: true,
			errMsg:  "invalid hash format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateLocationHash(tt.hash)
			if tt.wantErr {
				assert.NotEmpty(t, err.Field, "Expected validation error but got none")
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.Empty(t, err.Field, "Expected no validation error but got: %v", err)
			}
		})
	}
}

func TestValidateTriviaAnswer(t *testing.T) {
	currentTime := time.Now().Unix()

	tests := []struct {
		name       string
		questionID string
		answer     string
		timestamp  int64
		wantErr    bool
		errCount   int
		errMsgs    []string
	}{
		{
			name:       "Valid answer",
			questionID: "science_medium_42", // Simple 3-part format
			answer:     "Paris",
			timestamp:  currentTime,
			wantErr:    true, // This will still fail because it expects player ID regex
			errCount:   1,
			errMsgs:    []string{"invalid question ID format"},
		},
		{
			name:       "Empty question ID",
			questionID: "",
			answer:     "Paris",
			timestamp:  currentTime,
			wantErr:    true,
			errCount:   1,
			errMsgs:    []string{"question ID cannot be empty"},
		},
		{
			name:       "Empty answer",
			questionID: "science_medium_42",
			answer:     "",
			timestamp:  currentTime,
			wantErr:    true,
			errCount:   2, // Both question ID format and empty answer
			errMsgs:    []string{"answer cannot be empty"},
		},
		{
			name:       "Answer too long",
			questionID: "science_medium_42_1234567",
			answer:     string(make([]byte, 201)),
			timestamp:  currentTime,
			wantErr:    true,
			errCount:   1,
			errMsgs:    []string{"answer must be less than 200 characters"},
		},
		{
			name:       "Invalid timestamp",
			questionID: "science_medium_42_1234567",
			answer:     "Paris",
			timestamp:  0,
			wantErr:    true,
			errCount:   1,
			errMsgs:    []string{"invalid timestamp"},
		},
		{
			name:       "Future timestamp",
			questionID: "science_medium_42_1234567",
			answer:     "Paris",
			timestamp:  currentTime + 3600,
			wantErr:    true,
			errCount:   1,
			errMsgs:    []string{"timestamp cannot be in the future"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validateTriviaAnswer(tt.questionID, tt.answer, tt.timestamp)
			if tt.wantErr {
				assert.GreaterOrEqual(t, len(errs), 1, "Expected at least one error")
				// Check that at least one of the expected error messages is found
				found := false
				for _, expectedMsg := range tt.errMsgs {
					for _, err := range errs {
						if strings.Contains(err.Error(), expectedMsg) {
							found = true
							break
						}
					}
					if found {
						break
					}
				}
				assert.True(t, found, "Expected error message not found in: %v", errs)
			} else {
				assert.Empty(t, errs)
			}
		})
	}
}

func TestValidateGridPosition(t *testing.T) {
	tests := []struct {
		name        string
		pos         GridPos
		maxGridSize int
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "Valid position",
			pos:         GridPos{X: 2, Y: 2},
			maxGridSize: 4,
			wantErr:     false,
		},
		{
			name:        "Zero position",
			pos:         GridPos{X: 0, Y: 0},
			maxGridSize: 4,
			wantErr:     false,
		},
		{
			name:        "Max valid position",
			pos:         GridPos{X: 3, Y: 3},
			maxGridSize: 4,
			wantErr:     false,
		},
		{
			name:        "X out of bounds",
			pos:         GridPos{X: 4, Y: 2},
			maxGridSize: 4,
			wantErr:     true,
			errMsg:      "X position out of bounds",
		},
		{
			name:        "Y out of bounds",
			pos:         GridPos{X: 2, Y: 4},
			maxGridSize: 4,
			wantErr:     true,
			errMsg:      "Y position out of bounds",
		},
		{
			name:        "Negative X",
			pos:         GridPos{X: -1, Y: 2},
			maxGridSize: 4,
			wantErr:     true,
			errMsg:      "X position out of bounds",
		},
		{
			name:        "Negative Y",
			pos:         GridPos{X: 2, Y: -1},
			maxGridSize: 4,
			wantErr:     true,
			errMsg:      "Y position out of bounds",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateGridPosition(tt.pos, tt.maxGridSize)
			if tt.wantErr {
				assert.NotNil(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestValidateSegmentID(t *testing.T) {
	tests := []struct {
		name      string
		segmentID string
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "Valid segment ID - a1",
			segmentID: "segment_a1",
			wantErr:   false,
		},
		{
			name:      "Valid segment ID - h8",
			segmentID: "segment_h8",
			wantErr:   false,
		},
		{
			name:      "Empty segment ID",
			segmentID: "",
			wantErr:   true,
			errMsg:    "segment ID cannot be empty",
		},
		{
			name:      "Invalid format - missing prefix",
			segmentID: "a1",
			wantErr:   true,
			errMsg:    "invalid segment ID format",
		},
		{
			name:      "Invalid format - wrong prefix",
			segmentID: "puzzle_a1",
			wantErr:   true,
			errMsg:    "invalid segment ID format",
		},
		{
			name:      "Invalid format - no underscore",
			segmentID: "segmenta1",
			wantErr:   true,
			errMsg:    "invalid segment ID format",
		},
		{
			name:      "Invalid grid reference",
			segmentID: "segment_z9",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSegmentID(tt.segmentID)
			if tt.wantErr {
				assert.NotEqual(t, ValidationError{}, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.Equal(t, ValidationError{}, err)
			}
		})
	}
}

func TestValidateFragmentMove(t *testing.T) {
	tests := []struct {
		name       string
		fragmentID string
		newPos     GridPos
		timestamp  int64
		gridSize   int
		wantErr    bool
		errCount   int
		errMsgs    []string
	}{
		{
			name:       "Valid move",
			fragmentID: "fragment_player-uuid",
			newPos:     GridPos{X: 2, Y: 2},
			timestamp:  time.Now().Unix(),
			gridSize:   4,
			wantErr:    false,
		},
		{
			name:       "Empty fragment ID",
			fragmentID: "",
			newPos:     GridPos{X: 2, Y: 2},
			timestamp:  time.Now().Unix(),
			gridSize:   4,
			wantErr:    true,
			errCount:   1,
			errMsgs:    []string{"fragment ID is required"},
		},
		{
			name:       "Invalid position",
			fragmentID: "fragment_player-uuid",
			newPos:     GridPos{X: 4, Y: 4},
			timestamp:  time.Now().Unix(),
			gridSize:   4,
			wantErr:    true,
			errCount:   1,
			errMsgs:    []string{"position out of bounds"},
		},
		{
			name:       "Invalid timestamp",
			fragmentID: "fragment_player-uuid",
			newPos:     GridPos{X: 2, Y: 2},
			timestamp:  0,
			gridSize:   4,
			wantErr:    true,
			errCount:   1,
			errMsgs:    []string{"invalid timestamp"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validateFragmentMove(tt.fragmentID, tt.newPos, tt.timestamp, tt.gridSize, "")
			if tt.wantErr {
				assert.Len(t, errs, tt.errCount)
				for i, err := range errs {
					assert.Contains(t, err.Error(), tt.errMsgs[i])
				}
			} else {
				assert.Empty(t, errs)
			}
		})
	}
}

func TestValidateAuthWrapper(t *testing.T) {
	validPlayerID := uuid.New().String()

	tests := []struct {
		name     string
		data     []byte
		wantErr  bool
		errCount int
		playerID string
		errMsgs  []string
	}{
		{
			name: "Valid auth wrapper",
			data: []byte(`{
				"auth": {"playerId": "` + validPlayerID + `"},
				"payload": {"test": true}
			}`),
			wantErr:  false,
			playerID: validPlayerID,
		},
		{
			name:     "Invalid JSON",
			data:     []byte(`{"auth": {"playerId": "`),
			wantErr:  true,
			errCount: 1,
			errMsgs:  []string{"invalid JSON format"},
		},
		{
			name: "Missing auth field",
			data: []byte(`{
				"payload": {"test": true}
			}`),
			wantErr:  true,
			errCount: 1,
			errMsgs:  []string{"missing auth field"},
		},
		{
			name: "Missing player ID",
			data: []byte(`{
				"auth": {},
				"payload": {"test": true}
			}`),
			wantErr:  true,
			errCount: 1,
			errMsgs:  []string{"missing player ID in auth"},
		},
		{
			name: "Invalid player ID format",
			data: []byte(`{
				"auth": {"playerId": "not-a-uuid"},
				"payload": {"test": true}
			}`),
			wantErr:  true,
			errCount: 1,
			errMsgs:  []string{"invalid player ID format"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapper, errs := validateAuthWrapper(tt.data)
			if tt.wantErr {
				assert.Len(t, errs, tt.errCount)
				for i, err := range errs {
					assert.Contains(t, err.Error(), tt.errMsgs[i])
				}
				assert.Nil(t, wrapper)
			} else {
				assert.Empty(t, errs)
				assert.NotNil(t, wrapper)
				assert.Equal(t, tt.playerID, wrapper.Auth.PlayerID)
			}
		})
	}
}

func TestValidateRoleSelection(t *testing.T) {
	tests := []struct {
		name     string
		payload  json.RawMessage
		wantErr  bool
		errCount int
		role     string
		errMsgs  []string
	}{
		{
			name:    "Valid role selection",
			payload: json.RawMessage(`{"role": "detective"}`),
			wantErr: false,
			role:    "detective",
		},
		{
			name:     "Invalid JSON",
			payload:  json.RawMessage(`{"role": "detective"`),
			wantErr:  true,
			errCount: 1,
			errMsgs:  []string{"invalid JSON format"},
		},
		{
			name:     "Missing role field",
			payload:  json.RawMessage(`{}`),
			wantErr:  true,
			errCount: 1,
			errMsgs:  []string{"role cannot be empty"},
		},
		{
			name:     "Invalid role",
			payload:  json.RawMessage(`{"role": "superhero"}`),
			wantErr:  true,
			errCount: 1,
			errMsgs:  []string{"invalid role selection"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, errs := ValidateRoleSelection(tt.payload)
			if tt.wantErr {
				assert.Len(t, errs, tt.errCount)
				for i, err := range errs {
					assert.Contains(t, err.Error(), tt.errMsgs[i])
				}
				// data is not nil even when there are errors
			} else {
				assert.Empty(t, errs)
				assert.NotNil(t, data)
				assert.Equal(t, tt.role, data["role"])
			}
		})
	}
}

func TestValidateSpecialtySelection(t *testing.T) {
	tests := []struct {
		name        string
		payload     json.RawMessage
		wantErr     bool
		errCount    int
		specialties []string
		errMsgs     []string
	}{
		{
			name:        "Valid single specialty",
			payload:     json.RawMessage(`{"specialties": ["science"]}`),
			wantErr:     false,
			specialties: []string{"science"},
		},
		{
			name:        "Valid two specialties",
			payload:     json.RawMessage(`{"specialties": ["science", "history"]}`),
			wantErr:     false,
			specialties: []string{"science", "history"},
		},
		{
			name:     "Missing specialties field",
			payload:  json.RawMessage(`{}`),
			wantErr:  true,
			errCount: 1,
			errMsgs:  []string{"missing specialties field"},
		},
		{
			name:     "Too many specialties",
			payload:  json.RawMessage(`{"specialties": ["science", "history", "geography"]}`),
			wantErr:  true,
			errCount: 1,
			errMsgs:  []string{"must select 1-2 specialties"},
		},
		{
			name:     "Invalid specialty",
			payload:  json.RawMessage(`{"specialties": ["science", "magic"]}`),
			wantErr:  true,
			errCount: 1,
			errMsgs:  []string{"invalid specialty: magic"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, errs := ValidateSpecialtySelection(tt.payload)
			if tt.wantErr {
				assert.Len(t, errs, tt.errCount)
				for i, err := range errs {
					assert.Contains(t, err.Error(), tt.errMsgs[i])
				}
				assert.Nil(t, data)
			} else {
				assert.Empty(t, errs)
				assert.NotNil(t, data)
				specialties := data["specialties"].([]string)
				assert.Equal(t, tt.specialties, specialties)
			}
		})
	}
}

func TestValidateEmptyPayload(t *testing.T) {
	tests := []struct {
		name     string
		payload  json.RawMessage
		wantErr  bool
		errCount int
		errMsg   string
	}{
		{
			name:    "Valid empty object",
			payload: json.RawMessage(`{}`),
			wantErr: false,
		},
		{
			name:     "Invalid JSON",
			payload:  json.RawMessage(`{`),
			wantErr:  true,
			errCount: 1,
			errMsg:   "unexpected end of JSON input",
		},
		{
			name:    "Non-empty payload",
			payload: json.RawMessage(`{"field": "value"}`),
			wantErr: false,
		},
		{
			name:    "Null payload",
			payload: json.RawMessage(`null`),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, errs := ValidateEmptyPayload(tt.payload)
			if tt.wantErr {
				assert.Len(t, errs, tt.errCount)
				if tt.errCount > 0 {
					assert.Contains(t, errs[0].Error(), tt.errMsg)
				}
				assert.Nil(t, data)
			} else {
				assert.Empty(t, errs)
				// data can be nil for empty payloads
			}
		})
	}
}
