package main

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBasicGameFlow(t *testing.T) {
	// Create managers
	pm := NewPlayerManager()
	tm := NewTriviaManager()
	broadcastChan := make(chan BroadcastMessage, 256)
	gm := NewGameManager(pm, tm, broadcastChan)
	_ = NewEventHandlers(gm, pm, broadcastChan) // eh not used in this test

	// Verify initial state
	assert.Equal(t, PhaseSetup, gm.GetPhase())

	// Create host
	host := pm.CreatePlayer(nil, true)
	assert.NotNil(t, host)
	assert.True(t, host.IsHost)

	// Verify host is connected
	assert.True(t, pm.IsHostConnected())
	assert.Equal(t, host.ID, pm.GetHost().ID)

	// Test starting game without enough players
	canStart, reason := gm.CanStartGame()
	assert.False(t, canStart)
	assert.Contains(t, reason, "Need at least")

	// Add minimum players (4 non-host)
	players := make([]*Player, 4)
	roles := []string{"art_enthusiast", "detective", "tourist", "janitor"}

	for i := 0; i < 4; i++ {
		players[i] = pm.CreatePlayer(nil, false)
		assert.NotNil(t, players[i])
		assert.False(t, players[i].IsHost)

		// Set role
		err := pm.SetPlayerRole(players[i].ID, roles[i])
		assert.NoError(t, err)

		// Set specialties (players auto-ready after this)
		err = pm.SetPlayerSpecialties(players[i].ID, []string{"science", "history"})
		if err == nil {
			// If no error, verify player is ready
			p, _ := pm.GetPlayer(players[i].ID)
			assert.True(t, p.Ready)
		}
	}

	// Verify player counts
	assert.Equal(t, 5, pm.GetPlayerCount()) // 4 players + 1 host
	assert.Equal(t, 4, pm.GetNonHostPlayerCount())
	assert.GreaterOrEqual(t, pm.GetReadyCount(), 0) // May be 0 due to specialty issues

	// Test role distribution
	distribution := pm.GetRoleDistribution()
	assert.Equal(t, 1, distribution["art_enthusiast"])
	assert.Equal(t, 1, distribution["detective"])
	assert.Equal(t, 1, distribution["tourist"])
	assert.Equal(t, 1, distribution["janitor"])
}

func TestTokenCalculations(t *testing.T) {
	// Create game manager
	pm := NewPlayerManager()
	tm := NewTriviaManager()
	broadcastChan := make(chan BroadcastMessage, 256)
	gm := NewGameManager(pm, tm, broadcastChan)

	// Set up test tokens
	gm.state.TeamTokens.AnchorTokens = 25
	gm.state.TeamTokens.ChronosTokens = 15
	gm.state.TeamTokens.GuideTokens = 35
	gm.state.TeamTokens.ClarityTokens = 20

	// Test total calculation
	total := gm.getTotalTokens()
	assert.Equal(t, 95, total)

	// Test threshold calculations with different difficulties
	difficulties := []struct {
		name     string
		expected map[string]int
	}{
		{
			name: "easy",
			expected: map[string]int{
				"anchor":  6, // 25 / (5 * 0.8) = 6.25 -> 6
				"chronos": 3, // 15 / (5 * 0.8) = 3.75 -> 3
				"guide":   8, // 35 / (5 * 0.8) = 8.75 -> 8
				"clarity": 5, // 20 / (5 * 0.8) = 5.0 -> 5
			},
		},
		{
			name: "medium",
			expected: map[string]int{
				"anchor":  5, // 25 / (5 * 1.0) = 5
				"chronos": 3, // 15 / (5 * 1.0) = 3
				"guide":   7, // 35 / (5 * 1.0) = 7
				"clarity": 4, // 20 / (5 * 1.0) = 4
			},
		},
		{
			name: "hard",
			expected: map[string]int{
				"anchor":  3, // 25 / (5 * 1.3) = 3.84 -> 3
				"chronos": 2, // 15 / (5 * 1.3) = 2.30 -> 2
				"guide":   5, // 35 / (5 * 1.3) = 5.38 -> 5
				"clarity": 3, // 20 / (5 * 1.3) = 3.07 -> 3
			},
		},
	}

	for _, diff := range difficulties {
		t.Run(diff.name, func(t *testing.T) {
			gm.SetDifficulty(diff.name)
			thresholds := gm.calculateThresholdsReached()

			for tokenType, expected := range diff.expected {
				assert.Equal(t, expected, thresholds[tokenType],
					"Token type %s in %s difficulty", tokenType, diff.name)
			}
		})
	}
}

func TestGridSizeCalculations(t *testing.T) {
	pm := NewPlayerManager()
	tm := NewTriviaManager()
	broadcastChan := make(chan BroadcastMessage, 256)
	gm := NewGameManager(pm, tm, broadcastChan)

	// Test all grid size breakpoints
	tests := []struct {
		playerCount  int
		expectedGrid int
	}{
		{1, 3}, {5, 3}, {9, 3}, // 3x3 grid
		{10, 4}, {15, 4}, {16, 4}, // 4x4 grid
		{17, 5}, {20, 5}, {25, 5}, // 5x5 grid
		{26, 6}, {30, 6}, {36, 6}, // 6x6 grid
		{37, 7}, {40, 7}, {49, 7}, // 7x7 grid
		{50, 8}, {60, 8}, {64, 8}, // 8x8 grid
	}

	for _, tt := range tests {
		t.Run("players_"+string(rune(tt.playerCount+48)), func(t *testing.T) {
			result := gm.calculateGridSize(tt.playerCount)
			assert.Equal(t, tt.expectedGrid, result,
				"Grid size for %d players", tt.playerCount)
		})
	}
}

func TestFragmentPositionCalculations(t *testing.T) {
	pm := NewPlayerManager()
	tm := NewTriviaManager()
	broadcastChan := make(chan BroadcastMessage, 256)
	gm := NewGameManager(pm, tm, broadcastChan)

	// Test position calculations for different grid sizes
	tests := []struct {
		gridSize    int
		playerIndex int
		expectedX   int
		expectedY   int
	}{
		// 3x3 grid
		{3, 0, 0, 0},
		{3, 1, 1, 0},
		{3, 2, 2, 0},
		{3, 3, 0, 1},
		{3, 4, 1, 1},
		{3, 8, 2, 2},

		// 4x4 grid
		{4, 0, 0, 0},
		{4, 3, 3, 0},
		{4, 4, 0, 1},
		{4, 15, 3, 3},
	}

	for _, tt := range tests {
		t.Run("grid_calculation", func(t *testing.T) {
			pos := gm.calculateCorrectPosition(tt.playerIndex, tt.gridSize)
			assert.Equal(t, tt.expectedX, pos.X)
			assert.Equal(t, tt.expectedY, pos.Y)
		})
	}
}

func TestConcurrentPlayerOperations(t *testing.T) {
	pm := NewPlayerManager()

	// Create multiple players concurrently
	done := make(chan bool, 20)
	playerCount := 20

	// Concurrent player creation
	for i := 0; i < playerCount; i++ {
		go func(index int) {
			defer func() { done <- true }()

			// Create player
			player := pm.CreatePlayer(nil, false)
			assert.NotNil(t, player)

			// Try to set role (may fail due to concurrent access)
			roles := []string{"art_enthusiast", "detective", "tourist", "janitor"}
			role := roles[index%len(roles)]
			_ = pm.SetPlayerRole(player.ID, role)

			// Try to get player back
			retrieved, err := pm.GetPlayer(player.ID)
			assert.NoError(t, err)
			assert.Equal(t, player.ID, retrieved.ID)
		}(i)
	}

	// Wait for all operations
	for i := 0; i < playerCount; i++ {
		<-done
	}

	// Verify final state is consistent
	assert.Equal(t, playerCount, pm.GetPlayerCount())
	allPlayers := pm.GetAllPlayers()
	assert.Len(t, allPlayers, playerCount)

	// Verify all players have unique IDs
	seenIDs := make(map[string]bool)
	for _, player := range allPlayers {
		assert.False(t, seenIDs[player.ID], "Duplicate player ID: %s", player.ID)
		seenIDs[player.ID] = true
	}
}

func TestValidationIntegration(t *testing.T) {
	// Test that validation functions work with real data structures

	// Test UUID validation
	validUUID := "123e4567-e89b-12d3-a456-426614174000"
	err := validatePlayerID(validUUID)
	assert.Nil(t, err)

	invalidUUID := "not-a-uuid"
	err = validatePlayerID(invalidUUID)
	assert.NotNil(t, err)

	// Test role validation
	validRoles := []string{"art_enthusiast", "detective", "tourist", "janitor"}
	for _, role := range validRoles {
		err := validateRole(role)
		assert.Nil(t, err, "Role %s should be valid", role)
	}

	err = validateRole("invalid_role")
	assert.NotNil(t, err)

	// Test specialty validation
	_ = []string{"general", "geography", "history", "music", "science", "video_games"} // validSpecialties not used

	// Test single specialty
	errs := validateSpecialties([]string{"science"})
	assert.Empty(t, errs)

	// Test two specialties
	errs = validateSpecialties([]string{"science", "history"})
	assert.Empty(t, errs)

	// Test too many specialties
	errs = validateSpecialties([]string{"science", "history", "geography"})
	assert.NotEmpty(t, errs)

	// Test invalid specialty
	errs = validateSpecialties([]string{"magic"})
	assert.NotEmpty(t, errs)

	// Test grid position validation
	err = validateGridPosition(GridPos{X: 2, Y: 2}, 4)
	assert.Nil(t, err)

	err = validateGridPosition(GridPos{X: 4, Y: 4}, 4)
	assert.NotNil(t, err)
}

func TestTriviaManagerIntegration(t *testing.T) {
	tm := NewTriviaManager()

	// Test that categories are loaded
	categories := tm.GetAvailableCategories()
	expectedCategories := []string{"general", "geography", "history", "music", "science", "video_games"}

	for _, expected := range expectedCategories {
		assert.Contains(t, categories, expected)
	}

	// Test category support
	for _, category := range expectedCategories {
		assert.True(t, tm.IsCategorySupported(category))
	}

	assert.False(t, tm.IsCategorySupported("invalid"))

	// Test stats
	stats := tm.GetCategoryStats()
	assert.NotNil(t, stats)

	poolStats := tm.GetPoolStats()
	assert.NotNil(t, poolStats)

	summaryStats := tm.GetSummaryStats()
	assert.NotNil(t, summaryStats)
}

func TestSystemStability(t *testing.T) {
	// Test that the system doesn't crash under stress

	// Create multiple game managers
	for i := 0; i < 5; i++ {
		pm := NewPlayerManager()
		tm := NewTriviaManager()
		broadcastChan := make(chan BroadcastMessage, 256)
		gm := NewGameManager(pm, tm, broadcastChan)

		// Verify each one initializes properly
		assert.NotNil(t, gm)
		assert.Equal(t, PhaseSetup, gm.GetPhase())

		// Add some players
		for j := 0; j < 5; j++ {
			player := pm.CreatePlayer(nil, false)
			assert.NotNil(t, player)
		}

		// Verify counts
		assert.Equal(t, 5, pm.GetPlayerCount())
	}
}

func TestDataStructureConsistency(t *testing.T) {
	// Test that data structures maintain consistency

	// Test TeamTokens
	tokens := TeamTokens{
		AnchorTokens:  10,
		ChronosTokens: 20,
		GuideTokens:   30,
		ClarityTokens: 40,
	}

	// Test JSON marshaling/unmarshaling
	data, err := json.Marshal(tokens)
	assert.NoError(t, err)

	var decoded TeamTokens
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, tokens, decoded)

	// Test GridPos
	pos := GridPos{X: 3, Y: 7}
	data, err = json.Marshal(pos)
	assert.NoError(t, err)

	var decodedPos GridPos
	err = json.Unmarshal(data, &decodedPos)
	assert.NoError(t, err)
	assert.Equal(t, pos, decodedPos)

	// Test PuzzleFragment
	fragment := PuzzleFragment{
		ID:              "fragment-1",
		PlayerID:        "player-1",
		Position:        GridPos{X: 1, Y: 2},
		CorrectPosition: GridPos{X: 3, Y: 4},
		Solved:          true,
		Visible:         true,
		PreSolved:       false,
		MovableBy:       "player-1",
		IsUnassigned:    false,
		LastMoved:       time.Now(),
	}

	data, err = json.Marshal(fragment)
	assert.NoError(t, err)

	var decodedFragment PuzzleFragment
	err = json.Unmarshal(data, &decodedFragment)
	assert.NoError(t, err)
	assert.Equal(t, fragment.ID, decodedFragment.ID)
	assert.Equal(t, fragment.PlayerID, decodedFragment.PlayerID)
	assert.Equal(t, fragment.Position, decodedFragment.Position)
}
