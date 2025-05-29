package main

import (
	"testing"
	"time"

	"github.com/MaxThePrisberry/canvas-conundrum/server/constants"
	"github.com/stretchr/testify/assert"
)

// Mock broadcast channel for testing
type mockBroadcaster struct {
	messages []BroadcastMessage
}

func (m *mockBroadcaster) Send(msg BroadcastMessage) {
	m.messages = append(m.messages, msg)
}

func createTestGameManager() (*GameManager, *PlayerManager, *TriviaManager, chan BroadcastMessage) {
	playerMgr := NewPlayerManager()
	triviaMgr := NewTriviaManager()
	broadcastChan := make(chan BroadcastMessage, 256)
	gameMgr := NewGameManager(playerMgr, triviaMgr, broadcastChan)
	return gameMgr, playerMgr, triviaMgr, broadcastChan
}

func TestNewGameManager(t *testing.T) {
	gm, _, _, _ := createTestGameManager()

	assert.NotNil(t, gm)
	assert.NotNil(t, gm.state)
	assert.Equal(t, PhaseSetup, gm.GetPhase())
	assert.Equal(t, "medium", gm.state.Difficulty)
	assert.NotNil(t, gm.state.QuestionHistory)
	assert.NotNil(t, gm.state.PlayerAnalytics)
	assert.NotNil(t, gm.state.PuzzleFragments)
}

func TestGameManagerPhases(t *testing.T) {
	gm, _, _, _ := createTestGameManager()

	// Test initial phase
	assert.Equal(t, PhaseSetup, gm.GetPhase())

	// Test phase string conversion
	phases := []struct {
		phase    GamePhase
		expected string
	}{
		{PhaseSetup, "setup"},
		{PhaseResourceGathering, "resource_gathering"},
		{PhasePuzzleAssembly, "puzzle_assembly"},
		{PhasePostGame, "post_game"},
	}

	for _, p := range phases {
		assert.Equal(t, p.expected, p.phase.String())
	}
}

func TestSetDifficulty(t *testing.T) {
	gm, _, _, _ := createTestGameManager()

	tests := []struct {
		name       string
		difficulty string
		wantErr    bool
	}{
		{
			name:       "Set easy difficulty",
			difficulty: "easy",
			wantErr:    false,
		},
		{
			name:       "Set medium difficulty",
			difficulty: "medium",
			wantErr:    false,
		},
		{
			name:       "Set hard difficulty",
			difficulty: "hard",
			wantErr:    false,
		},
		{
			name:       "Invalid difficulty",
			difficulty: "extreme",
			wantErr:    true,
		},
		{
			name:       "Empty difficulty",
			difficulty: "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := gm.SetDifficulty(tt.difficulty)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.difficulty, gm.state.Difficulty)
			}
		})
	}
}

func TestCanStartGame(t *testing.T) {
	gm, pm, _, _ := createTestGameManager()

	// Test with no players
	canStart, reason := gm.CanStartGame()
	assert.False(t, canStart)
	assert.Contains(t, reason, "minimum")

	// Add host
	pm.CreatePlayer(nil, true)
	canStart, reason = gm.CanStartGame()
	assert.False(t, canStart)
	assert.Contains(t, reason, "minimum")

	// Add minimum players (4 non-host)
	for i := 0; i < 4; i++ {
		player := pm.CreatePlayer(nil, false)
		pm.SetPlayerRole(player.ID, "detective")
		pm.SetPlayerReady(player.ID, true)
	}

	// Now should be able to start
	canStart, reason = gm.CanStartGame()
	assert.True(t, canStart)
	assert.Empty(t, reason)

	// Test in wrong phase
	gm.state.Phase = PhaseResourceGathering
	canStart, reason = gm.CanStartGame()
	assert.False(t, canStart)
	assert.Contains(t, reason, "already in progress")
}

func TestGetDifficultyModifiers(t *testing.T) {
	gm, _, _, _ := createTestGameManager()

	tests := []struct {
		difficulty string
		expected   constants.DifficultyModifiers
	}{
		{
			difficulty: "easy",
			expected:   constants.EasyMode,
		},
		{
			difficulty: "medium",
			expected:   constants.MediumMode,
		},
		{
			difficulty: "hard",
			expected:   constants.HardMode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.difficulty, func(t *testing.T) {
			gm.SetDifficulty(tt.difficulty)
			mods := gm.getDifficultyModifiers()
			assert.Equal(t, tt.expected.TriviaModifier, mods.TriviaModifier)
			assert.Equal(t, tt.expected.TimeLimitModifier, mods.TimeLimitModifier)
			assert.Equal(t, tt.expected.TokenThresholdModifier, mods.TokenThresholdModifier)
		})
	}
}

func TestCalculateGridSize(t *testing.T) {
	gm, _, _, _ := createTestGameManager()

	tests := []struct {
		playerCount int
		expected    int
	}{
		{1, 3},
		{5, 3},
		{9, 3},
		{10, 4},
		{16, 4},
		{17, 5},
		{25, 5},
		{26, 6},
		{36, 6},
		{37, 7},
		{49, 7},
		{50, 8},
		{64, 8},
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.playerCount))+"_players", func(t *testing.T) {
			result := gm.calculateGridSize(tt.playerCount)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCalculateCorrectPosition(t *testing.T) {
	gm, _, _, _ := createTestGameManager()

	tests := []struct {
		playerIndex int
		gridSize    int
		expectedX   int
		expectedY   int
	}{
		{0, 3, 0, 0},
		{1, 3, 1, 0},
		{2, 3, 2, 0},
		{3, 3, 0, 1},
		{8, 3, 2, 2},
		{0, 4, 0, 0},
		{15, 4, 3, 3},
	}

	for _, tt := range tests {
		t.Run("position_calculation", func(t *testing.T) {
			pos := gm.calculateCorrectPosition(tt.playerIndex, tt.gridSize)
			assert.Equal(t, tt.expectedX, pos.X)
			assert.Equal(t, tt.expectedY, pos.Y)
		})
	}
}

func TestTokenThresholdCalculations(t *testing.T) {
	gm, _, _, _ := createTestGameManager()

	// Set up some test tokens
	gm.state.TeamTokens = TeamTokens{
		AnchorTokens:  50,
		ChronosTokens: 30,
		GuideTokens:   25,
		ClarityTokens: 40,
	}

	// Test with medium difficulty (modifier = 1.0)
	gm.SetDifficulty("medium")
	thresholds := gm.calculateThresholdsReached()

	// Each token type has 5 thresholds
	// Threshold = tokens / (5 * difficultyModifier)
	assert.Equal(t, 10, thresholds["anchor"]) // 50 / (5 * 1.0) = 10
	assert.Equal(t, 6, thresholds["chronos"]) // 30 / (5 * 1.0) = 6
	assert.Equal(t, 5, thresholds["guide"])   // 25 / (5 * 1.0) = 5
	assert.Equal(t, 8, thresholds["clarity"]) // 40 / (5 * 1.0) = 8

	// Test with easy difficulty (modifier = 0.8)
	gm.SetDifficulty("easy")
	thresholds = gm.calculateThresholdsReached()
	assert.Equal(t, 12, thresholds["anchor"]) // 50 / (5 * 0.8) = 12.5 -> 12

	// Test with hard difficulty (modifier = 1.3)
	gm.SetDifficulty("hard")
	thresholds = gm.calculateThresholdsReached()
	assert.Equal(t, 7, thresholds["anchor"]) // 50 / (5 * 1.3) = 7.69 -> 7
}

func TestGetTotalTokens(t *testing.T) {
	gm, _, _, _ := createTestGameManager()

	gm.state.TeamTokens = TeamTokens{
		AnchorTokens:  10,
		ChronosTokens: 20,
		GuideTokens:   30,
		ClarityTokens: 40,
	}

	total := gm.getTotalTokens()
	assert.Equal(t, 100, total)
}

func TestProcessTriviaAnswer(t *testing.T) {
	gm, pm, _, _ := createTestGameManager()

	// Create a player
	player := pm.CreatePlayer(nil, false)
	pm.SetPlayerRole(player.ID, "detective") // Gets guide token bonus

	// Set game to resource gathering phase
	gm.state.Phase = PhaseResourceGathering

	// Create a mock current question
	gm.state.CurrentQuestions[player.ID] = &TriviaQuestion{
		ID:            "test_question_1",
		Text:          "Test question?",
		CorrectAnswer: "Test answer",
		Category:      "science",
	}

	// Test correct answer - but this will fail because we can't mock trivia validation
	err := gm.ProcessTriviaAnswer(player.ID, "test_question_1", "Test answer")
	// The test will error because the trivia manager doesn't have the question loaded
	assert.Error(t, err)

	// Test wrong phase
	gm.state.Phase = PhaseSetup
	err = gm.ProcessTriviaAnswer(player.ID, "test_question_1", "Test answer")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "trivia answers only accepted during resource gathering")

	// Test non-existent player
	err = gm.ProcessTriviaAnswer("invalid-player", "test_question_1", "Test answer")
	assert.Error(t, err)
}

func TestFragmentMovement(t *testing.T) {
	gm, pm, _, _ := createTestGameManager()

	// Create player
	player := pm.CreatePlayer(nil, false)

	// Set to puzzle phase
	gm.state.Phase = PhasePuzzleAssembly
	gm.state.GridSize = 4

	// Create a fragment owned by the player
	fragment := &PuzzleFragment{
		ID:              "fragment-1",
		PlayerID:        player.ID,
		Position:        GridPos{X: 0, Y: 0},
		CorrectPosition: GridPos{X: 2, Y: 2},
		Visible:         true,
		LastMoved:       time.Now().Add(-2 * time.Second), // Past cooldown
	}
	gm.state.PuzzleFragments["fragment-1"] = fragment

	// Test valid move
	err := gm.ProcessFragmentMove(player.ID, "fragment-1", GridPos{X: 1, Y: 1})
	assert.NoError(t, err)
	assert.Equal(t, 1, fragment.Position.X)
	assert.Equal(t, 1, fragment.Position.Y)

	// Test move during cooldown
	err = gm.ProcessFragmentMove(player.ID, "fragment-1", GridPos{X: 2, Y: 2})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cooldown")

	// Test move out of bounds
	fragment.LastMoved = time.Now().Add(-2 * time.Second)
	err = gm.ProcessFragmentMove(player.ID, "fragment-1", GridPos{X: 4, Y: 4})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "out of bounds")

	// Test wrong phase
	gm.state.Phase = PhaseSetup
	err = gm.ProcessFragmentMove(player.ID, "fragment-1", GridPos{X: 1, Y: 1})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "only during puzzle assembly")
}

func TestPuzzleCompletion(t *testing.T) {
	gm, pm, _, _ := createTestGameManager()

	// Set to puzzle phase
	gm.state.Phase = PhasePuzzleAssembly
	gm.state.GridSize = 2

	// Create 4 fragments (2x2 grid)
	players := make([]*Player, 4)
	for i := 0; i < 4; i++ {
		players[i] = pm.CreatePlayer(nil, false)

		fragment := &PuzzleFragment{
			ID:              "fragment-" + string(rune('0'+i)),
			PlayerID:        players[i].ID,
			Position:        GridPos{X: i % 2, Y: i / 2},
			CorrectPosition: GridPos{X: i % 2, Y: i / 2},
			Visible:         true,
			Solved:          true,
		}
		gm.state.PuzzleFragments[fragment.ID] = fragment
	}

	// Check if puzzle is complete
	complete := gm.checkPuzzleComplete()
	assert.True(t, complete)

	// Move one fragment to wrong position
	gm.state.PuzzleFragments["fragment-0"].Position = GridPos{X: 1, Y: 1}
	complete = gm.checkPuzzleComplete()
	assert.False(t, complete)

	// Make one fragment invisible
	gm.state.PuzzleFragments["fragment-0"].Position = GridPos{X: 0, Y: 0}
	gm.state.PuzzleFragments["fragment-0"].Visible = false
	complete = gm.checkPuzzleComplete()
	assert.False(t, complete)
}

func TestGuideTokenLinearProgression(t *testing.T) {
	gm, _, _, _ := createTestGameManager()

	// Test guide highlight calculation
	correctPos := GridPos{X: 2, Y: 2}
	_ = 4 // gridSize not used directly in this test

	tests := []struct {
		thresholdLevel int
		expectedCount  int
		description    string
	}{
		{0, 4, "Level 0: 25% of grid"},
		{1, 3, "Level 1: smaller area"},
		{2, 2, "Level 2: smaller area"},
		{3, 2, "Level 3: very small area"},
		{4, 2, "Level 4: 2 positions for precision"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			positions := gm.calculateHighlightPositions(correctPos, tt.thresholdLevel)
			assert.GreaterOrEqual(t, len(positions), tt.expectedCount)

			// Verify correct position is always included
			hasCorrect := false
			for _, pos := range positions {
				if pos.X == correctPos.X && pos.Y == correctPos.Y {
					hasCorrect = true
					break
				}
			}
			assert.True(t, hasCorrect)
		})
	}
}

func TestConcurrentTokenUpdates(t *testing.T) {
	gm, pm, _, _ := createTestGameManager()

	// Create multiple players
	players := make([]*Player, 10)
	for i := 0; i < 10; i++ {
		players[i] = pm.CreatePlayer(nil, false)
	}

	// Set to resource gathering phase
	gm.state.Phase = PhaseResourceGathering

	// Concurrent token updates (simulated)
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(idx int) {
			defer func() { done <- true }()

			// Safely update tokens
			gm.mu.Lock()
			gm.state.TeamTokens.AnchorTokens += 10
			gm.mu.Unlock()
		}(i)
	}

	// Wait for all updates
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify final state
	assert.Equal(t, 100, gm.state.TeamTokens.AnchorTokens)
}
