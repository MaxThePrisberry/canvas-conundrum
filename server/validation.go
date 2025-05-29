package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/MaxThePrisberry/canvas-conundrum/server/constants"
)

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (ve ValidationError) Error() string {
	return fmt.Sprintf("validation error on field '%s': %s", ve.Field, ve.Message)
}

// ValidationResult holds validation results
type ValidationResult struct {
	Valid  bool              `json:"valid"`
	Errors []ValidationError `json:"errors,omitempty"`
}

// Input validation constants
const (
	MaxPlayerNameLength     = 50
	MaxMessageLength        = 1000
	MaxSpecialtiesPerPlayer = 2
	MinGridPosition         = 0
	MaxGridPosition         = 10 // Reasonable upper bound
)

// Regular expressions for validation
var (
	playerIDRegex   = regexp.MustCompile(`^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-4[a-fA-F0-9]{3}-[8|9|aA|bB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$`)
	playerNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_\-\s]{1,50}$`)
	hashRegex       = regexp.MustCompile(`^[A-Z_0-9]{10,50}$`)
)

// validatePlayerID validates a player ID format (UUID v4)
func validatePlayerID(playerID string) ValidationError {
	if playerID == "" {
		return ValidationError{Field: "playerId", Message: "player ID cannot be empty"}
	}
	if !playerIDRegex.MatchString(playerID) {
		return ValidationError{Field: "playerId", Message: "invalid player ID format"}
	}
	return ValidationError{}
}

// validatePlayerName validates a player name
func validatePlayerName(name string) ValidationError {
	if name == "" {
		return ValidationError{Field: "name", Message: "name cannot be empty"}
	}
	if !utf8.ValidString(name) {
		return ValidationError{Field: "name", Message: "name contains invalid UTF-8 characters"}
	}
	if len(name) > MaxPlayerNameLength {
		return ValidationError{Field: "name", Message: fmt.Sprintf("name too long (max %d characters)", MaxPlayerNameLength)}
	}
	if !playerNameRegex.MatchString(name) {
		return ValidationError{Field: "name", Message: "name contains invalid characters (allowed: letters, numbers, spaces, hyphens, underscores)"}
	}
	return ValidationError{}
}

// validateRole validates a player role selection
func validateRole(role string) ValidationError {
	if role == "" {
		return ValidationError{Field: "role", Message: "role cannot be empty"}
	}

	validRoles := map[string]bool{
		constants.RoleArtEnthusiast: true,
		constants.RoleDetective:     true,
		constants.RoleTourist:       true,
		constants.RoleJanitor:       true,
	}

	if !validRoles[role] {
		return ValidationError{Field: "role", Message: "invalid role selection"}
	}
	return ValidationError{}
}

// validateSpecialties validates trivia specialty selections
func validateSpecialties(specialties []string) []ValidationError {
	var errors []ValidationError

	if len(specialties) == 0 {
		errors = append(errors, ValidationError{Field: "specialties", Message: "at least one specialty must be selected"})
		return errors
	}

	if len(specialties) > MaxSpecialtiesPerPlayer {
		errors = append(errors, ValidationError{Field: "specialties", Message: fmt.Sprintf("too many specialties (max %d)", MaxSpecialtiesPerPlayer)})
	}

	validCategories := make(map[string]bool)
	for _, cat := range constants.TriviaCategories {
		validCategories[cat] = true
	}

	seen := make(map[string]bool)
	for i, specialty := range specialties {
		if specialty == "" {
			errors = append(errors, ValidationError{Field: fmt.Sprintf("specialties[%d]", i), Message: "specialty cannot be empty"})
			continue
		}

		if seen[specialty] {
			errors = append(errors, ValidationError{Field: fmt.Sprintf("specialties[%d]", i), Message: "duplicate specialty"})
			continue
		}
		seen[specialty] = true

		if !validCategories[specialty] {
			errors = append(errors, ValidationError{Field: fmt.Sprintf("specialties[%d]", i), Message: "invalid specialty category"})
		}
	}

	return errors
}

// validateLocationHash validates a resource station hash
func validateLocationHash(hash string) ValidationError {
	if hash == "" {
		return ValidationError{Field: "verifiedHash", Message: "location hash cannot be empty"}
	}

	if !hashRegex.MatchString(hash) {
		return ValidationError{Field: "verifiedHash", Message: "invalid hash format"}
	}

	// Check if it's a valid resource station hash
	validHash := false
	for _, validHashValue := range constants.ResourceStationHashes {
		if hash == validHashValue {
			validHash = true
			break
		}
	}

	if !validHash {
		return ValidationError{Field: "verifiedHash", Message: "invalid resource station hash"}
	}

	return ValidationError{}
}

// validateTriviaAnswer validates a trivia answer submission
func validateTriviaAnswer(questionID, answer string, timestamp int64) []ValidationError {
	var errors []ValidationError

	if questionID == "" {
		errors = append(errors, ValidationError{Field: "questionId", Message: "question ID cannot be empty"})
	} else if !playerIDRegex.MatchString(strings.Split(questionID, "_")[0]) {
		// Basic format check - should be category_difficulty_number
		parts := strings.Split(questionID, "_")
		if len(parts) != 3 {
			errors = append(errors, ValidationError{Field: "questionId", Message: "invalid question ID format"})
		}
	}

	if answer == "" {
		errors = append(errors, ValidationError{Field: "answer", Message: "answer cannot be empty"})
	} else {
		if !utf8.ValidString(answer) {
			errors = append(errors, ValidationError{Field: "answer", Message: "answer contains invalid UTF-8 characters"})
		}
		if len(answer) > 200 { // Reasonable limit for trivia answers
			errors = append(errors, ValidationError{Field: "answer", Message: "answer too long (max 200 characters)"})
		}
	}

	if timestamp <= 0 {
		errors = append(errors, ValidationError{Field: "timestamp", Message: "invalid timestamp"})
	}

	return errors
}

// validateGridPosition validates a grid position
func validateGridPosition(pos GridPos, maxGridSize int) ValidationError {
	if pos.X < MinGridPosition || pos.X >= maxGridSize {
		return ValidationError{Field: "position.x", Message: fmt.Sprintf("x position out of bounds (0-%d)", maxGridSize-1)}
	}
	if pos.Y < MinGridPosition || pos.Y >= maxGridSize {
		return ValidationError{Field: "position.y", Message: fmt.Sprintf("y position out of bounds (0-%d)", maxGridSize-1)}
	}
	return ValidationError{}
}

// validateSegmentID validates a puzzle segment ID
func validateSegmentID(segmentID string) ValidationError {
	if segmentID == "" {
		return ValidationError{Field: "segmentId", Message: "segment ID cannot be empty"}
	}

	// Format should be: segment_a1, segment_b2, etc.
	segmentRegex := regexp.MustCompile(`^segment_[a-z][0-9]+$`)
	if !segmentRegex.MatchString(segmentID) {
		return ValidationError{Field: "segmentId", Message: "invalid segment ID format"}
	}

	return ValidationError{}
}

// validateFragmentMove validates a fragment movement request
func validateFragmentMove(fragmentID string, newPos GridPos, timestamp int64, maxGridSize int, ownership string) []ValidationError {
	var errors []ValidationError

	if fragmentID == "" {
		errors = append(errors, ValidationError{Field: "fragmentId", Message: "fragment ID cannot be empty"})
	} else if !strings.HasPrefix(fragmentID, "fragment_") {
		errors = append(errors, ValidationError{Field: "fragmentId", Message: "invalid fragment ID format"})
	}

	if posErr := validateGridPosition(newPos, maxGridSize); posErr.Field != "" {
		errors = append(errors, posErr)
	}

	if timestamp <= 0 {
		errors = append(errors, ValidationError{Field: "timestamp", Message: "invalid timestamp"})
	}

	// Validate ownership format if provided
	if ownership != "" && ownership != "anyone" {
		if ownershipErr := validatePlayerID(ownership); ownershipErr.Field != "" {
			ownershipErr.Field = "ownership"
			ownershipErr.Message = "invalid fragment ownership format"
			errors = append(errors, ownershipErr)
		}
	}

	return errors
}

// validatePieceRecommendation validates a piece recommendation request (no message field)
func validatePieceRecommendation(toPlayerID, fromFragmentID, toFragmentID string, fromPos, toPos GridPos, maxGridSize int) []ValidationError {
	var errors []ValidationError

	if playerErr := validatePlayerID(toPlayerID); playerErr.Field != "" {
		playerErr.Field = "toPlayerId"
		errors = append(errors, playerErr)
	}

	if fromFragmentID == "" {
		errors = append(errors, ValidationError{Field: "fromFragmentId", Message: "from fragment ID cannot be empty"})
	} else if !strings.HasPrefix(fromFragmentID, "fragment_") {
		errors = append(errors, ValidationError{Field: "fromFragmentId", Message: "invalid from fragment ID format"})
	}

	if toFragmentID == "" {
		errors = append(errors, ValidationError{Field: "toFragmentId", Message: "to fragment ID cannot be empty"})
	} else if !strings.HasPrefix(toFragmentID, "fragment_") {
		errors = append(errors, ValidationError{Field: "toFragmentId", Message: "invalid to fragment ID format"})
	}

	// Note: Message field removed - no longer supported

	if fromPosErr := validateGridPosition(fromPos, maxGridSize); fromPosErr.Field != "" {
		fromPosErr.Field = "suggestedFromPos"
		errors = append(errors, fromPosErr)
	}

	if toPosErr := validateGridPosition(toPos, maxGridSize); toPosErr.Field != "" {
		toPosErr.Field = "suggestedToPos"
		errors = append(errors, toPosErr)
	}

	return errors
}

// validateAuthWrapper validates the authentication wrapper
func validateAuthWrapper(data []byte) (*AuthWrapper, []ValidationError) {
	var wrapper AuthWrapper
	var errors []ValidationError

	if err := json.Unmarshal(data, &wrapper); err != nil {
		errors = append(errors, ValidationError{Field: "payload", Message: "invalid JSON format"})
		return nil, errors
	}

	if playerErr := validatePlayerID(wrapper.Auth.PlayerID); playerErr.Field != "" {
		errors = append(errors, playerErr)
	}

	if len(wrapper.Payload) == 0 {
		errors = append(errors, ValidationError{Field: "payload", Message: "payload cannot be empty"})
	}

	return &wrapper, errors
}

// validateJSONPayload validates that a payload is valid JSON
func validateJSONPayload(payload json.RawMessage, target interface{}) ValidationError {
	if len(payload) == 0 {
		return ValidationError{Field: "payload", Message: "payload cannot be empty"}
	}

	if err := json.Unmarshal(payload, target); err != nil {
		return ValidationError{Field: "payload", Message: fmt.Sprintf("invalid JSON format: %v", err)}
	}

	return ValidationError{}
}

// Specific validation functions for each WebSocket event type

// ValidateRoleSelection validates role selection payload
func ValidateRoleSelection(payload json.RawMessage) (map[string]interface{}, []ValidationError) {
	var data struct {
		Role string `json:"role"`
	}

	var errors []ValidationError
	if jsonErr := validateJSONPayload(payload, &data); jsonErr.Field != "" {
		errors = append(errors, jsonErr)
		return nil, errors
	}

	if roleErr := validateRole(data.Role); roleErr.Field != "" {
		errors = append(errors, roleErr)
	}

	result := map[string]interface{}{
		"role": data.Role,
	}

	return result, errors
}

// ValidateSpecialtySelection validates specialty selection payload
func ValidateSpecialtySelection(payload json.RawMessage) (map[string]interface{}, []ValidationError) {
	var data struct {
		Specialties []string `json:"specialties"`
	}

	var errors []ValidationError
	if jsonErr := validateJSONPayload(payload, &data); jsonErr.Field != "" {
		errors = append(errors, jsonErr)
		return nil, errors
	}

	errors = append(errors, validateSpecialties(data.Specialties)...)

	result := map[string]interface{}{
		"specialties": data.Specialties,
	}

	return result, errors
}

// ValidateLocationVerification validates location verification payload
func ValidateLocationVerification(payload json.RawMessage) (map[string]interface{}, []ValidationError) {
	var data struct {
		VerifiedHash string `json:"verifiedHash"`
	}

	var errors []ValidationError
	if jsonErr := validateJSONPayload(payload, &data); jsonErr.Field != "" {
		errors = append(errors, jsonErr)
		return nil, errors
	}

	if hashErr := validateLocationHash(data.VerifiedHash); hashErr.Field != "" {
		errors = append(errors, hashErr)
	}

	result := map[string]interface{}{
		"verifiedHash": data.VerifiedHash,
	}

	return result, errors
}

// ValidateTriviaAnswer validates trivia answer payload
func ValidateTriviaAnswer(payload json.RawMessage) (map[string]interface{}, []ValidationError) {
	var data struct {
		QuestionID string `json:"questionId"`
		Answer     string `json:"answer"`
		Timestamp  int64  `json:"timestamp"`
	}

	var errors []ValidationError
	if jsonErr := validateJSONPayload(payload, &data); jsonErr.Field != "" {
		errors = append(errors, jsonErr)
		return nil, errors
	}

	errors = append(errors, validateTriviaAnswer(data.QuestionID, data.Answer, data.Timestamp)...)

	result := map[string]interface{}{
		"questionId": data.QuestionID,
		"answer":     strings.TrimSpace(data.Answer),
		"timestamp":  data.Timestamp,
	}

	return result, errors
}

// ValidateSegmentCompletion validates segment completion payload
func ValidateSegmentCompletion(payload json.RawMessage) (map[string]interface{}, []ValidationError) {
	var data struct {
		SegmentID           string `json:"segmentId"`
		CompletionTimestamp int64  `json:"completionTimestamp"`
	}

	var errors []ValidationError
	if jsonErr := validateJSONPayload(payload, &data); jsonErr.Field != "" {
		errors = append(errors, jsonErr)
		return nil, errors
	}

	if segErr := validateSegmentID(data.SegmentID); segErr.Field != "" {
		errors = append(errors, segErr)
	}

	if data.CompletionTimestamp <= 0 {
		errors = append(errors, ValidationError{Field: "completionTimestamp", Message: "invalid completion timestamp"})
	}

	result := map[string]interface{}{
		"segmentId":           data.SegmentID,
		"completionTimestamp": data.CompletionTimestamp,
	}

	return result, errors
}

// ValidateFragmentMove validates fragment move payload
func ValidateFragmentMove(payload json.RawMessage, maxGridSize int) (map[string]interface{}, []ValidationError) {
	var data struct {
		FragmentID  string  `json:"fragmentId"`
		NewPosition GridPos `json:"newPosition"`
		Timestamp   int64   `json:"timestamp"`
	}

	var errors []ValidationError
	if jsonErr := validateJSONPayload(payload, &data); jsonErr.Field != "" {
		errors = append(errors, jsonErr)
		return nil, errors
	}

	errors = append(errors, validateFragmentMove(data.FragmentID, data.NewPosition, data.Timestamp, maxGridSize)...)

	result := map[string]interface{}{
		"fragmentId":  data.FragmentID,
		"newPosition": data.NewPosition,
		"timestamp":   data.Timestamp,
	}

	return result, errors
}

// ValidatePlayerReady validates player ready payload
func ValidatePlayerReady(payload json.RawMessage) (map[string]interface{}, []ValidationError) {
	var data struct {
		Ready bool `json:"ready"`
	}

	var errors []ValidationError
	if jsonErr := validateJSONPayload(payload, &data); jsonErr.Field != "" {
		errors = append(errors, jsonErr)
		return nil, errors
	}

	result := map[string]interface{}{
		"ready": data.Ready,
	}

	return result, errors
}

// ValidatePieceRecommendationRequest validates piece recommendation request payload
func ValidatePieceRecommendationRequest(payload json.RawMessage, maxGridSize int) (map[string]interface{}, []ValidationError) {
	var data struct {
		ToPlayerID       string  `json:"toPlayerId"`
		FromFragmentID   string  `json:"fromFragmentId"`
		ToFragmentID     string  `json:"toFragmentId"`
		SuggestedFromPos GridPos `json:"suggestedFromPos"`
		SuggestedToPos   GridPos `json:"suggestedToPos"`
		// Message field removed - no longer accepted
	}

	var errors []ValidationError
	if jsonErr := validateJSONPayload(payload, &data); jsonErr.Field != "" {
		errors = append(errors, jsonErr)
		return nil, errors
	}

	errors = append(errors, validatePieceRecommendation(
		data.ToPlayerID,
		data.FromFragmentID,
		data.ToFragmentID,
		data.SuggestedFromPos,
		data.SuggestedToPos,
		maxGridSize,
	)...)

	result := map[string]interface{}{
		"toPlayerId":       data.ToPlayerID,
		"fromFragmentId":   data.FromFragmentID,
		"toFragmentId":     data.ToFragmentID,
		"suggestedFromPos": data.SuggestedFromPos,
		"suggestedToPos":   data.SuggestedToPos,
		// No message field in result
	}

	return result, errors
}

// ValidatePieceRecommendationResponse validates piece recommendation response payload
func ValidatePieceRecommendationResponse(payload json.RawMessage) (map[string]interface{}, []ValidationError) {
	var data struct {
		RecommendationID string `json:"recommendationId"`
		Accepted         bool   `json:"accepted"`
	}

	var errors []ValidationError
	if jsonErr := validateJSONPayload(payload, &data); jsonErr.Field != "" {
		errors = append(errors, jsonErr)
		return nil, errors
	}

	if data.RecommendationID == "" {
		errors = append(errors, ValidationError{Field: "recommendationId", Message: "recommendation ID cannot be empty"})
	} else if !playerIDRegex.MatchString(data.RecommendationID) {
		errors = append(errors, ValidationError{Field: "recommendationId", Message: "invalid recommendation ID format"})
	}

	result := map[string]interface{}{
		"recommendationId": data.RecommendationID,
		"accepted":         data.Accepted,
	}

	return result, errors
}

// ValidateEmptyPayload validates payloads that should be empty (like host actions)
func ValidateEmptyPayload(payload json.RawMessage) (map[string]interface{}, []ValidationError) {
	var data map[string]interface{}

	var errors []ValidationError
	if len(payload) > 0 {
		if jsonErr := validateJSONPayload(payload, &data); jsonErr.Field != "" {
			errors = append(errors, jsonErr)
			return nil, errors
		}
	}

	return data, errors
}
