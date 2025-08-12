package models

import (
	"canvas-conundrum/constants"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGame(t *testing.T) {
	game := NewGame()

	assert.NotNil(t, game)
	assert.NotEmpty(t, game.ID)
	assert.Equal(t, PhaseSetup, game.CurrentPhase)
	assert.Equal(t, DifficultyMedium, game.Difficulty)
	assert.Equal(t, 0, game.PlayerCount)
	assert.False(t, game.GameStarted)
	assert.False(t, game.PuzzleSuccess)
	assert.NotNil(t, game.TeamTokens)
	assert.Equal(t, 0, game.CurrentRound)
}

func TestGamePhaseTransitions(t *testing.T) {
	tests := []struct {
		name         string
		startPhase   GamePhase
		expectedNext GamePhase
		shouldEnd    bool
	}{
		{
			name:         "Setup to Resource",
			startPhase:   PhaseSetup,
			expectedNext: PhaseResourceGathering,
			shouldEnd:    false,
		},
		{
			name:         "Resource to Puzzle",
			startPhase:   PhaseResourceGathering,
			expectedNext: PhasePuzzleAssembly,
			shouldEnd:    false,
		},
		{
			name:         "Puzzle to Analytics",
			startPhase:   PhasePuzzleAssembly,
			expectedNext: PhaseAnalytics,
			shouldEnd:    false,
		},
		{
			name:         "Analytics ends game",
			startPhase:   PhaseAnalytics,
			expectedNext: PhaseAnalytics,
			shouldEnd:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			game := NewGame()
			game.CurrentPhase = tt.startPhase

			next := game.NextPhase()

			if !tt.shouldEnd {
				assert.Equal(t, tt.expectedNext, next)
				assert.Equal(t, tt.expectedNext, game.CurrentPhase)
			} else {
				assert.Equal(t, tt.expectedNext, next)
			}
		})
	}
}

func TestGameSetDifficulty(t *testing.T) {
	tests := []struct {
		difficulty   DifficultyMode
		playerCount  int
		expectedGrid int
	}{
		{DifficultyEasy, 4, 3},
		{DifficultyEasy, 8, 3},
		{DifficultyMedium, 16, 4},
		{DifficultyMedium, 24, 5},
		{DifficultyHard, 36, 6},
		{DifficultyHard, 64, 8},
	}

	for _, tt := range tests {
		t.Run(string(tt.difficulty), func(t *testing.T) {
			game := NewGame()
			game.SetDifficulty(tt.difficulty)
			game.PlayerCount = tt.playerCount

			assert.Equal(t, tt.difficulty, game.Difficulty)
			assert.Equal(t, tt.expectedGrid, game.GetGridSize())
		})
	}
}

func TestGetGridSize(t *testing.T) {
	tests := []struct {
		playerCount  int
		expectedSize int
	}{
		{4, 3},
		{8, 3},
		{12, 4},
		{16, 4},
		{20, 5},
		{24, 5},
		{28, 6},
		{32, 6},
		{36, 6},
		{40, 7},
		{48, 7},
		{56, 8},
		{64, 8},
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.playerCount)), func(t *testing.T) {
			game := NewGame()
			game.PlayerCount = tt.playerCount

			size := game.GetGridSize()
			assert.Equal(t, tt.expectedSize, size)
		})
	}
}

func TestGetPreSolvedPieces(t *testing.T) {
	tests := []struct {
		name           string
		tokens         map[TokenType]int
		expectedPieces int
	}{
		{
			name: "No tokens",
			tokens: map[TokenType]int{
				TokenAnchor:  0,
				TokenChronos: 0,
				TokenGuide:   0,
				TokenClarity: 0,
			},
			expectedPieces: 0,
		},
		{
			name: "All threshold 1",
			tokens: map[TokenType]int{
				TokenAnchor:  10,
				TokenChronos: 10,
				TokenGuide:   10,
				TokenClarity: 10,
			},
			expectedPieces: 2,
		},
		{
			name: "All threshold 2",
			tokens: map[TokenType]int{
				TokenAnchor:  25, // Threshold 2
				TokenChronos: 25,
				TokenGuide:   25,
				TokenClarity: 25,
			},
			expectedPieces: 4,
		},
		{
			name: "Mixed thresholds",
			tokens: map[TokenType]int{
				TokenAnchor:  30, // Threshold 2
				TokenChronos: 20, // Threshold 1
				TokenGuide:   10, // Threshold 1
				TokenClarity: 5,  // Threshold 0
			},
			expectedPieces: 4, // 2 * 2 for anchor threshold 2
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			game := NewGame()
			game.TeamTokens.AnchorTokens = tt.tokens[TokenAnchor]
			game.TeamTokens.ChronosTokens = tt.tokens[TokenChronos]
			game.TeamTokens.GuideTokens = tt.tokens[TokenGuide]
			game.TeamTokens.ClarityTokens = tt.tokens[TokenClarity]

			pieces := game.GetPreSolvedPieces()
			assert.Equal(t, tt.expectedPieces, pieces)
		})
	}
}

func TestGetTotalPuzzleTime(t *testing.T) {
	tests := []struct {
		name         string
		tokens       map[TokenType]int
		playerCount  int
		expectedTime int
	}{
		{
			name: "Base time only",
			tokens: map[TokenType]int{
				TokenAnchor:  0,
				TokenChronos: 0,
				TokenGuide:   0,
				TokenClarity: 0,
			},
			playerCount:  4,
			expectedTime: constants.PuzzleBaseTime,
		},
		{
			name: "With chronos bonus",
			tokens: map[TokenType]int{
				TokenAnchor:  0,
				TokenChronos: 15, // Threshold 1
				TokenGuide:   0,
				TokenClarity: 0,
			},
			playerCount:  4,
			expectedTime: constants.PuzzleBaseTime + constants.TimeExtensionPerThreshold,
		},
		{
			name: "Max chronos bonus",
			tokens: map[TokenType]int{
				TokenAnchor:  0,
				TokenChronos: 45, // Threshold 3
				TokenGuide:   0,
				TokenClarity: 0,
			},
			playerCount:  4,
			expectedTime: constants.PuzzleBaseTime + (3 * constants.TimeExtensionPerThreshold),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			game := NewGame()
			game.TeamTokens.AnchorTokens = tt.tokens[TokenAnchor]
			game.TeamTokens.ChronosTokens = tt.tokens[TokenChronos]
			game.TeamTokens.GuideTokens = tt.tokens[TokenGuide]
			game.TeamTokens.ClarityTokens = tt.tokens[TokenClarity]
			game.PlayerCount = tt.playerCount

			time := game.GetTotalPuzzleTime()
			assert.Equal(t, tt.expectedTime, time)
		})
	}
}

func TestTeamTokens(t *testing.T) {
	t.Run("NewTeamTokens", func(t *testing.T) {
		tokens := NewTeamTokens()
		assert.NotNil(t, tokens)
		assert.Equal(t, 0, tokens.AnchorTokens)
		assert.Equal(t, 0, tokens.ChronosTokens)
		assert.Equal(t, 0, tokens.GuideTokens)
		assert.Equal(t, 0, tokens.ClarityTokens)
	})

	t.Run("AddTokens", func(t *testing.T) {
		tokens := NewTeamTokens()

		tokens.AddTokens(TokenAnchor, 5)
		assert.Equal(t, 5, tokens.AnchorTokens)

		tokens.AddTokens(TokenChronos, 10)
		assert.Equal(t, 10, tokens.ChronosTokens)

		tokens.AddTokens(TokenGuide, 15)
		assert.Equal(t, 15, tokens.GuideTokens)

		tokens.AddTokens(TokenClarity, 20)
		assert.Equal(t, 20, tokens.ClarityTokens)
	})

	t.Run("GetThreshold", func(t *testing.T) {
		tests := []struct {
			tokenCount int
			expected   int
		}{
			{0, 0},
			{5, 0},
			{10, 1},
			{15, 1},
			{20, 1},
			{25, 2},
			{30, 2},
			{35, 2},
			{40, 2},
			{45, 3},
			{50, 3}, // Max threshold
		}

		for _, tt := range tests {
			tokens := NewTeamTokens()
			tokens.AnchorTokens = tt.tokenCount

			threshold := tokens.GetThreshold(TokenAnchor)
			assert.Equal(t, tt.expected, threshold, "Token count %d should have threshold %d", tt.tokenCount, tt.expected)
		}
	})

	t.Run("GetTotal", func(t *testing.T) {
		tokens := NewTeamTokens()
		tokens.AnchorTokens = 10
		tokens.ChronosTokens = 20
		tokens.GuideTokens = 15
		tokens.ClarityTokens = 25

		total := tokens.GetTotal()
		assert.Equal(t, 70, total)
	})
}

func TestPuzzleGrid(t *testing.T) {
	t.Run("NewPuzzleGrid", func(t *testing.T) {
		grid := NewPuzzleGrid(4)

		assert.NotNil(t, grid)
		assert.Equal(t, 4, grid.Size)
		assert.Len(t, grid.Grid, 4)
		assert.NotNil(t, grid.Fragments)

		for i := range grid.Grid {
			assert.Len(t, grid.Grid[i], 4)
			for j := range grid.Grid[i] {
				assert.Nil(t, grid.Grid[i][j]) // Grid starts empty
			}
		}
	})

	t.Run("AddFragment", func(t *testing.T) {
		grid := NewPuzzleGrid(3)

		// Add a fragment
		frag := grid.AddFragment("B2", "player1")
		assert.NotNil(t, frag)
		assert.Equal(t, "B2", frag.SegmentID)
		assert.Equal(t, "player1", frag.PlayerID)
		assert.Equal(t, "fragment_player1", frag.ID)

		// Check correct position was calculated
		assert.Equal(t, 1, frag.CorrectPosition.X) // Column 2 -> index 1
		assert.Equal(t, 1, frag.CorrectPosition.Y) // Row B -> index 1

		// Fragment should be placed at random position
		assert.NotNil(t, grid.Fragments[frag.ID])
	})

	t.Run("MoveFragment", func(t *testing.T) {
		grid := NewPuzzleGrid(3)

		// Add initial fragment
		frag1 := grid.AddFragment("A1", "player1")
		require.NotNil(t, frag1)

		// Move to an empty position
		newPos := Position{X: 2, Y: 2}
		err := grid.MoveFragment(frag1.ID, newPos)
		assert.NoError(t, err)
		assert.Equal(t, 2, frag1.Position.X)
		assert.Equal(t, 2, frag1.Position.Y)

		// Invalid - position out of bounds
		err = grid.MoveFragment(frag1.ID, Position{X: -1, Y: 0})
		assert.Error(t, err)

		err = grid.MoveFragment(frag1.ID, Position{X: 3, Y: 0})
		assert.Error(t, err)

		// Add another fragment
		frag2 := grid.AddFragment("B2", "player2")
		require.NotNil(t, frag2)

		// Invalid - destination occupied
		err = grid.MoveFragment(frag2.ID, frag1.Position)
		assert.Error(t, err)
	})

	t.Run("SwapFragments", func(t *testing.T) {
		grid := NewPuzzleGrid(3)

		// Add two fragments
		frag1 := grid.AddFragment("A1", "player1")
		frag2 := grid.AddFragment("B2", "player2")

		pos1Before := frag1.Position
		pos2Before := frag2.Position

		// Swap fragments
		err := grid.SwapFragments(frag1.ID, frag2.ID)
		assert.NoError(t, err)

		// Positions should be swapped
		assert.Equal(t, pos2Before, frag1.Position)
		assert.Equal(t, pos1Before, frag2.Position)
	})

	t.Run("CheckCompletion", func(t *testing.T) {
		grid := NewPuzzleGrid(2)

		// Not complete - empty
		assert.False(t, grid.CheckCompletion())

		// Manually place fragments in correct positions
		frag1 := &Fragment{
			ID:              "f1",
			SegmentID:       "A1",
			Position:        Position{X: 0, Y: 0},
			CorrectPosition: Position{X: 0, Y: 0},
		}
		frag2 := &Fragment{
			ID:              "f2",
			SegmentID:       "A2",
			Position:        Position{X: 1, Y: 0},
			CorrectPosition: Position{X: 1, Y: 0},
		}
		frag3 := &Fragment{
			ID:              "f3",
			SegmentID:       "B1",
			Position:        Position{X: 0, Y: 1},
			CorrectPosition: Position{X: 0, Y: 1},
		}
		frag4 := &Fragment{
			ID:              "f4",
			SegmentID:       "B2",
			Position:        Position{X: 1, Y: 1},
			CorrectPosition: Position{X: 1, Y: 1},
		}

		grid.Fragments[frag1.ID] = frag1
		grid.Fragments[frag2.ID] = frag2
		grid.Fragments[frag3.ID] = frag3
		grid.Fragments[frag4.ID] = frag4

		grid.Grid[0][0] = frag1
		grid.Grid[0][1] = frag2
		grid.Grid[1][0] = frag3
		grid.Grid[1][1] = frag4

		// Now complete
		assert.True(t, grid.CheckCompletion())
	})
}
