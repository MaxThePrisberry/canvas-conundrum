package main

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGamePhaseString(t *testing.T) {
	tests := []struct {
		name     string
		phase    GamePhase
		expected string
	}{
		{
			name:     "Setup phase",
			phase:    PhaseSetup,
			expected: "setup",
		},
		{
			name:     "Resource gathering phase",
			phase:    PhaseResourceGathering,
			expected: "resource_gathering",
		},
		{
			name:     "Puzzle assembly phase",
			phase:    PhasePuzzleAssembly,
			expected: "puzzle_assembly",
		},
		{
			name:     "Post game phase",
			phase:    PhasePostGame,
			expected: "post_game",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.phase.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBaseMessageMarshaling(t *testing.T) {
	tests := []struct {
		name    string
		message BaseMessage
		wantErr bool
	}{
		{
			name: "Valid message",
			message: BaseMessage{
				Type:    "test_type",
				Payload: json.RawMessage(`{"key": "value"}`),
			},
			wantErr: false,
		},
		{
			name: "Empty type",
			message: BaseMessage{
				Type:    "",
				Payload: json.RawMessage(`{}`),
			},
			wantErr: false,
		},
		{
			name: "Null payload",
			message: BaseMessage{
				Type:    "test",
				Payload: nil,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			data, err := json.Marshal(tt.message)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			// Test unmarshaling
			var decoded BaseMessage
			err = json.Unmarshal(data, &decoded)
			assert.NoError(t, err)
			assert.Equal(t, tt.message.Type, decoded.Type)
		})
	}
}

func TestAuthWrapperValidation(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr bool
		wantID  string
	}{
		{
			name:    "Valid auth wrapper",
			json:    `{"auth": {"playerId": "123e4567-e89b-12d3-a456-426614174000"}, "payload": {"test": true}}`,
			wantErr: false,
			wantID:  "123e4567-e89b-12d3-a456-426614174000",
		},
		{
			name:    "Missing auth field",
			json:    `{"payload": {"test": true}}`,
			wantErr: true,
		},
		{
			name:    "Missing playerId",
			json:    `{"auth": {}, "payload": {"test": true}}`,
			wantErr: true,
		},
		{
			name:    "Invalid JSON",
			json:    `{"auth": {"playerId": "test"`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapper, errors := validateAuthWrapper([]byte(tt.json))

			if tt.wantErr {
				assert.True(t, len(errors) > 0, "Expected validation errors but got none")
				return
			}

			assert.Equal(t, 0, len(errors), "Expected no validation errors")
			assert.Equal(t, tt.wantID, wrapper.Auth.PlayerID)
		})
	}
}

func TestPlayerThreadSafety(t *testing.T) {
	player := &Player{
		ID:    "test-id",
		State: StateConnected,
	}

	// Test concurrent access to player fields
	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			player.mu.Lock()
			if i%2 == 0 {
				player.State = StateConnected
			} else {
				player.State = StateDisconnected
			}
			player.mu.Unlock()
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			player.mu.RLock()
			_ = player.State
			player.mu.RUnlock()
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done
}

func TestTeamTokensCalculations(t *testing.T) {
	tokens := &TeamTokens{
		AnchorTokens:  50,
		ChronosTokens: 40,
		GuideTokens:   30,
		ClarityTokens: 20,
	}

	// Test JSON marshaling
	data, err := json.Marshal(tokens)
	assert.NoError(t, err)

	var decoded TeamTokens
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, tokens.AnchorTokens, decoded.AnchorTokens)
	assert.Equal(t, tokens.ChronosTokens, decoded.ChronosTokens)
	assert.Equal(t, tokens.GuideTokens, decoded.GuideTokens)
	assert.Equal(t, tokens.ClarityTokens, decoded.ClarityTokens)
}

func TestGridPosValidation(t *testing.T) {
	tests := []struct {
		name     string
		pos      GridPos
		gridSize int
		valid    bool
	}{
		{
			name:     "Valid position",
			pos:      GridPos{X: 2, Y: 2},
			gridSize: 4,
			valid:    true,
		},
		{
			name:     "X out of bounds",
			pos:      GridPos{X: 4, Y: 2},
			gridSize: 4,
			valid:    false,
		},
		{
			name:     "Y out of bounds",
			pos:      GridPos{X: 2, Y: 4},
			gridSize: 4,
			valid:    false,
		},
		{
			name:     "Negative X",
			pos:      GridPos{X: -1, Y: 2},
			gridSize: 4,
			valid:    false,
		},
		{
			name:     "Negative Y",
			pos:      GridPos{X: 2, Y: -1},
			gridSize: 4,
			valid:    false,
		},
		{
			name:     "Zero position",
			pos:      GridPos{X: 0, Y: 0},
			gridSize: 4,
			valid:    true,
		},
		{
			name:     "Max valid position",
			pos:      GridPos{X: 3, Y: 3},
			gridSize: 4,
			valid:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := tt.pos.X >= 0 && tt.pos.X < tt.gridSize &&
				tt.pos.Y >= 0 && tt.pos.Y < tt.gridSize
			assert.Equal(t, tt.valid, valid)
		})
	}
}

func TestPuzzleFragmentState(t *testing.T) {
	fragment := &PuzzleFragment{
		ID:              "fragment-1",
		PlayerID:        "player-1",
		Position:        GridPos{X: 1, Y: 1},
		CorrectPosition: GridPos{X: 2, Y: 2},
		Solved:          false,
		PreSolved:       false,
		Visible:         true,
		LastMoved:       time.Now(),
		MovableBy:       "player-1",
		IsUnassigned:    false,
	}

	// Test JSON marshaling
	data, err := json.Marshal(fragment)
	assert.NoError(t, err)

	var decoded PuzzleFragment
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, fragment.ID, decoded.ID)
	assert.Equal(t, fragment.PlayerID, decoded.PlayerID)
	assert.Equal(t, fragment.Position, decoded.Position)
	assert.Equal(t, fragment.CorrectPosition, decoded.CorrectPosition)
	assert.Equal(t, fragment.Visible, decoded.Visible)
}

func TestPieceRecommendation(t *testing.T) {
	rec := &PieceRecommendation{
		ID:               "rec-1",
		FromPlayerID:     "player-1",
		ToPlayerID:       "player-2",
		FromFragmentID:   "fragment-1",
		ToFragmentID:     "fragment-2",
		SuggestedFromPos: GridPos{X: 1, Y: 1},
		SuggestedToPos:   GridPos{X: 2, Y: 2},
		Message:          "Try this position",
		Timestamp:        time.Now(),
	}

	// Test JSON marshaling
	data, err := json.Marshal(rec)
	assert.NoError(t, err)

	var decoded PieceRecommendation
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, rec.ID, decoded.ID)
	assert.Equal(t, rec.FromPlayerID, decoded.FromPlayerID)
	assert.Equal(t, rec.ToPlayerID, decoded.ToPlayerID)
}

func TestGameStateConcurrency(t *testing.T) {
	state := &GameState{
		Phase:                PhaseSetup,
		Difficulty:           "medium",
		CurrentRound:         0,
		RoundStartTime:       time.Now(),
		PuzzleStartTime:      time.Now(),
		PuzzleFragments:      make(map[string]*PuzzleFragment),
		PlayerAnalytics:      make(map[string]*PlayerAnalytics),
		QuestionHistory:      make(map[string]map[string]bool),
		PieceRecommendations: make(map[string]*PieceRecommendation),
		CurrentQuestions:     make(map[string]*TriviaQuestion),
	}

	// Test concurrent access
	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			state.Phase = PhaseResourceGathering
			state.CurrentRound = i
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			_ = state.Phase
			_ = state.CurrentRound
		}
		done <- true
	}()

	<-done
	<-done
}

func TestPlayerAnalytics(t *testing.T) {
	analytics := &PlayerAnalytics{
		PlayerID:   "player-1",
		PlayerName: "Test Player",
		TokenCollection: map[string]int{
			"anchor":  20,
			"chronos": 15,
			"guide":   10,
			"clarity": 5,
		},
		TriviaPerformance: TriviaPerformance{
			TotalQuestions: 15,
			CorrectAnswers: 10,
			AccuracyByCategory: map[string]float64{
				"science": 0.8,
				"history": 0.6,
			},
			SpecialtyBonus:   50,
			SpecialtyCorrect: 5,
			SpecialtyTotal:   7,
		},
		PuzzleMetrics: PuzzleSolvingMetrics{
			FragmentSolveTime:       180,
			MovesContributed:        8,
			SuccessfulMoves:         7,
			RecommendationsSent:     3,
			RecommendationsReceived: 2,
			RecommendationsAccepted: 2,
		},
	}

	// Calculate accuracy
	accuracy := float64(analytics.TriviaPerformance.CorrectAnswers) / float64(analytics.TriviaPerformance.TotalQuestions)
	assert.InDelta(t, 0.666, accuracy, 0.001)

	// Test JSON marshaling
	data, err := json.Marshal(analytics)
	assert.NoError(t, err)

	var decoded PlayerAnalytics
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, analytics.PlayerID, decoded.PlayerID)
	assert.Equal(t, analytics.TriviaPerformance.CorrectAnswers, decoded.TriviaPerformance.CorrectAnswers)
	assert.Equal(t, analytics.TokenCollection["anchor"], decoded.TokenCollection["anchor"])
}
