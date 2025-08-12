package services

import (
	"canvas-conundrum/constants"
	"canvas-conundrum/models"
	"sync"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGameManagerSingleton(t *testing.T) {
	// Reset singleton for testing
	gameInstance = nil
	once = sync.Once{}

	gm1 := GetGameInstance()
	gm2 := GetGameInstance()

	assert.NotNil(t, gm1)
	assert.NotNil(t, gm2)
	assert.Equal(t, gm1, gm2, "Should return same instance")
}

func TestGameManagerServices(t *testing.T) {
	// Reset singleton
	gameInstance = nil
	once = sync.Once{}

	gm := GetGameInstance()

	t.Run("SetTriviaService", func(t *testing.T) {
		trivia := NewTriviaService()
		gm.SetTriviaService(trivia)
		assert.Equal(t, trivia, gm.GetTriviaService())
	})

	t.Run("SetPuzzleService", func(t *testing.T) {
		puzzle := NewPuzzleService()
		gm.SetPuzzleService(puzzle)
		assert.Equal(t, puzzle, gm.GetPuzzleService())
	})

	t.Run("SetBroadcastService", func(t *testing.T) {
		broadcast := NewBroadcastService()
		gm.SetBroadcastService(broadcast)
		assert.Equal(t, broadcast, gm.GetBroadcastService())
	})

	t.Run("SetAnalyticsService", func(t *testing.T) {
		analytics := NewAnalyticsService()
		gm.SetAnalyticsService(analytics)
		assert.Equal(t, analytics, gm.GetAnalyticsService())
	})
}

func TestGameManagerPlayerManagement(t *testing.T) {
	// Reset singleton
	gameInstance = nil
	once = sync.Once{}

	gm := GetGameInstance()

	t.Run("AddPlayer", func(t *testing.T) {
		player1 := models.NewPlayer("player1", nil)
		player1.IsActive = true // Mark as active for testing
		player2 := models.NewPlayer("player2", nil)
		player2.IsActive = true // Mark as active for testing

		err := gm.AddPlayer(player1)
		assert.NoError(t, err)
		assert.Equal(t, 1, gm.GetPlayerCount())

		err = gm.AddPlayer(player2)
		assert.NoError(t, err)
		assert.Equal(t, 2, gm.GetPlayerCount())

		// Try adding duplicate
		err = gm.AddPlayer(player1)
		assert.NoError(t, err) // AddPlayer doesn't check for duplicates
		assert.Equal(t, 2, gm.GetPlayerCount())
	})

	t.Run("GetPlayer", func(t *testing.T) {
		player, exists := gm.GetPlayer("player1")
		assert.True(t, exists)
		assert.NotNil(t, player)
		assert.Equal(t, "player1", player.ID)

		player, exists = gm.GetPlayer("nonexistent")
		assert.False(t, exists)
		assert.Nil(t, player)
	})

	t.Run("RemovePlayer", func(t *testing.T) {
		gm.RemovePlayer("player1")
		assert.Equal(t, 1, gm.GetPlayerCount()) // Only counts active players

		player, exists := gm.GetPlayer("player1")
		assert.True(t, exists)           // Player still exists in map
		assert.False(t, player.IsActive) // But is marked inactive
	})

	t.Run("GetAllPlayers", func(t *testing.T) {
		players := gm.GetAllPlayers()
		assert.Len(t, players, 2) // Both players still in map
		assert.Equal(t, "player1", players["player1"].ID)
		assert.Equal(t, "player2", players["player2"].ID)
		assert.False(t, players["player1"].IsActive) // player1 is inactive
		assert.True(t, players["player2"].IsActive)  // player2 is still active
	})

	t.Run("PlayerReconnection", func(t *testing.T) {
		// Simulate player1 reconnecting
		reconnectPlayer := models.NewPlayer("player1", nil)
		reconnectPlayer.IsActive = true

		err := gm.AddPlayer(reconnectPlayer)
		assert.NoError(t, err)

		// Player should be active again
		player, exists := gm.GetPlayer("player1")
		assert.True(t, exists)
		assert.True(t, player.IsActive)

		// Count should be back to 2
		assert.Equal(t, 2, gm.GetPlayerCount())
	})
}

func TestGameManagerHostManagement(t *testing.T) {
	// Reset singleton
	gameInstance = nil
	once = sync.Once{}

	gm := GetGameInstance()

	t.Run("SetHost", func(t *testing.T) {
		host := models.NewHost("host1", nil)
		err := gm.SetHost(host)
		assert.NoError(t, err)

		retrievedHost := gm.GetHost()
		assert.NotNil(t, retrievedHost)
		assert.Equal(t, "host1", retrievedHost.ID)

		// Try setting another host when one exists (but with nil connection)
		// This should succeed since the current host has no connection
		host2 := models.NewHost("host2", nil)
		err = gm.SetHost(host2)
		assert.NoError(t, err)
		assert.Equal(t, "host2", gm.GetHost().ID)
	})

	t.Run("RemoveHost", func(t *testing.T) {
		gm.RemoveHost()
		// Host object should still exist but with nil connection
		host := gm.GetHost()
		assert.NotNil(t, host)
		assert.Equal(t, "host2", host.ID)
		assert.Nil(t, host.Connection)

		// Host can reconnect with same ID
		reconnectHost := models.NewHost("host2", nil)
		err := gm.SetHost(reconnectHost)
		assert.NoError(t, err)
		assert.Equal(t, "host2", gm.GetHost().ID)

		// Different host can also connect since current one has nil connection
		host3 := models.NewHost("host3", nil)
		err = gm.SetHost(host3)
		assert.NoError(t, err)
		assert.Equal(t, "host3", gm.GetHost().ID)
	})
}

func TestGameManagerGameFlow(t *testing.T) {
	// Reset singleton
	gameInstance = nil
	once = sync.Once{}

	gm := GetGameInstance()
	gm.SetBroadcastService(NewBroadcastService())

	// Add players
	for i := 0; i < 4; i++ {
		player := models.NewPlayer(string(rune('a'+i)), nil)
		player.Name = string(rune('A' + i))
		player.Role = models.Role("role")
		player.IsReady = true
		player.IsActive = true // Mark as active for testing
		gm.AddPlayer(player)
	}

	// Add host for game to start
	host := models.NewHost("test-host", &websocket.Conn{})
	gm.SetHost(host)

	t.Run("StartGame", func(t *testing.T) {
		// Set difficulty before starting
		gm.GetGame().SetDifficulty(models.DifficultyMedium)

		err := gm.StartGame()
		assert.NoError(t, err)
		assert.True(t, gm.GetGame().GameStarted)
		assert.Equal(t, string(models.PhaseResourceGathering), gm.GetCurrentPhase())

		// Can't start again
		err = gm.StartGame()
		assert.Error(t, err)
	})

	t.Run("GetCurrentRound", func(t *testing.T) {
		round := gm.GetCurrentRound()
		assert.Equal(t, 1, round)
	})

	t.Run("NextRound", func(t *testing.T) {
		gm.NextRound()
		assert.Equal(t, 2, gm.GetCurrentRound())
	})

	t.Run("TransitionToPhase", func(t *testing.T) {
		gm.TransitionToPhase(models.PhasePuzzleAssembly)
		assert.Equal(t, string(models.PhasePuzzleAssembly), gm.GetCurrentPhase())

		gm.TransitionToPhase(models.PhaseAnalytics)
		assert.Equal(t, string(models.PhaseAnalytics), gm.GetCurrentPhase())
	})

	t.Run("ResetGame", func(t *testing.T) {
		gm.ResetGame()
		assert.False(t, gm.IsGameStarted())
		assert.Equal(t, string(models.PhaseSetup), gm.GetCurrentPhase())
		assert.Equal(t, 0, gm.GetCurrentRound())
		assert.Equal(t, 0, gm.GetPlayerCount())
	})
}

func TestGameManagerTokenManagement(t *testing.T) {
	// Reset singleton
	gameInstance = nil
	once = sync.Once{}

	gm := GetGameInstance()

	t.Run("AddTeamTokens", func(t *testing.T) {
		gm.AddTeamTokens(models.TokenAnchor, 10)
		gm.AddTeamTokens(models.TokenChronos, 15)
		gm.AddTeamTokens(models.TokenGuide, 20)
		gm.AddTeamTokens(models.TokenClarity, 25)

		tokens := gm.GetTeamTokens()
		assert.Equal(t, 10, tokens.AnchorTokens)
		assert.Equal(t, 15, tokens.ChronosTokens)
		assert.Equal(t, 20, tokens.GuideTokens)
		assert.Equal(t, 25, tokens.ClarityTokens)
	})
}

func TestGameManagerRoleDistribution(t *testing.T) {
	// Reset singleton
	gameInstance = nil
	once = sync.Once{}

	gm := GetGameInstance()

	// Add players with different roles
	roles := []models.Role{
		models.RoleArtEnthusiast,
		models.RoleArtEnthusiast,
		models.RoleDetective,
		models.RoleTourist,
		models.RoleJanitor,
		models.RoleJanitor,
		models.RoleJanitor,
	}

	for i, role := range roles {
		player := models.NewPlayer(string(rune('a'+i)), nil)
		player.Role = role
		gm.AddPlayer(player)
	}

	dist := gm.GetRoleDistribution()

	assert.Equal(t, 2, dist[models.RoleArtEnthusiast])
	assert.Equal(t, 1, dist[models.RoleDetective])
	assert.Equal(t, 1, dist[models.RoleTourist])
	assert.Equal(t, 3, dist[models.RoleJanitor])
}

// TestGameManagerRecommendations tests are commented out as the recommendation system
// is not part of the current specifications in websocket-events.md
/*
func TestGameManagerRecommendations(t *testing.T) {
	// Reset singleton
	gameInstance = nil
	once = sync.Once{}

	gm := GetGameInstance()

	t.Run("AddRecommendation", func(t *testing.T) {
		rec := models.NewMoveRecommendation("rec1", "player1", "player2", "fragment1", "A1")
		gm.AddRecommendation(rec)

		retrieved, exists := gm.GetRecommendation("rec1")
		assert.True(t, exists)
		assert.Equal(t, rec, retrieved)
	})

	t.Run("RemoveRecommendation", func(t *testing.T) {
		gm.RemoveRecommendation("rec1")

		_, exists := gm.GetRecommendation("rec1")
		assert.False(t, exists)
	})

	t.Run("CleanExpiredRecommendations", func(t *testing.T) {
		// Add fresh recommendation
		rec1 := models.NewMoveRecommendation("rec1", "p1", "p2", "f1", "A1")
		gm.AddRecommendation(rec1)

		// Add expired recommendation
		rec2 := models.NewMoveRecommendation("rec2", "p1", "p3", "f2", "B1")
		rec2.Timestamp = time.Now().Add(-constants.RecommendationTimeout - time.Second)
		gm.AddRecommendation(rec2)

		gm.CleanExpiredRecommendations()

		_, exists1 := gm.GetRecommendation("rec1")
		assert.True(t, exists1, "Fresh recommendation should remain")

		_, exists2 := gm.GetRecommendation("rec2")
		assert.False(t, exists2, "Expired recommendation should be removed")
	})
}
*/

func TestGameManagerValidations(t *testing.T) {
	// Reset singleton
	gameInstance = nil
	once = sync.Once{}

	gm := GetGameInstance()

	t.Run("MaxPlayersLimit", func(t *testing.T) {
		// Add maximum players
		for i := 0; i < constants.MaxPlayers; i++ {
			player := models.NewPlayer(string(rune(i)), nil)
			err := gm.AddPlayer(player)
			require.NoError(t, err)
		}

		// Try to add one more
		extraPlayer := models.NewPlayer("extra", nil)
		err := gm.AddPlayer(extraPlayer)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "maximum")
	})

	t.Run("MinPlayersForStart", func(t *testing.T) {
		// Reset
		gameInstance = nil
		once = sync.Once{}
		gm = GetGameInstance()

		// Add less than minimum players
		for i := 0; i < constants.MinPlayers-1; i++ {
			player := models.NewPlayer(string(rune(i)), nil)
			player.IsReady = true
			player.IsActive = true // Mark as active for test
			gm.AddPlayer(player)
		}

		gm.GetGame().SetDifficulty(models.DifficultyMedium)

		// Need a host to start
		host := models.NewHost("test-host", nil)
		host.Connection = &websocket.Conn{} // Non-nil connection
		gm.SetHost(host)

		err := gm.StartGame()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "minimum")

		// Add one more to reach minimum
		player := models.NewPlayer("last", nil)
		player.IsReady = true
		player.IsActive = true // Mark as active for test
		gm.AddPlayer(player)

		err = gm.StartGame()
		assert.NoError(t, err)
	})
}

func TestGameManagerConcurrency(t *testing.T) {
	// Reset singleton
	gameInstance = nil
	once = sync.Once{}

	gm := GetGameInstance()

	// Test concurrent player additions
	var wg sync.WaitGroup
	errors := make([]error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			player := models.NewPlayer(string(rune('a'+index)), nil)
			player.IsActive = true // Mark as active for testing
			errors[index] = gm.AddPlayer(player)
		}(i)
	}

	wg.Wait()

	// All additions should succeed
	for _, err := range errors {
		assert.NoError(t, err)
	}

	assert.Equal(t, 10, gm.GetPlayerCount())
}

// TestGameManagerTimers tests are commented out as they test internal implementation details
// that are not part of the public API
/*
func TestGameManagerTimers(t *testing.T) {
	// Reset singleton
	gameInstance = nil
	once = sync.Once{}

	gm := GetGameInstance()

	t.Run("RoundTimer", func(t *testing.T) {
		// Start round timer
		gm.StartRoundTimer(1 * time.Second)
		assert.NotNil(t, gm.roundTimer)

		// Stop round timer
		gm.StopRoundTimer()
		// Timer should be stopped but not nil
		assert.NotNil(t, gm.roundTimer)
	})

	t.Run("PuzzleTimer", func(t *testing.T) {
		// Start puzzle timer
		gm.StartPuzzleTimer(2 * time.Second)
		assert.NotNil(t, gm.puzzleTimer)

		// Stop puzzle timer
		gm.StopPuzzleTimer()
		// Timer should be stopped but not nil
		assert.NotNil(t, gm.puzzleTimer)
	})
}
*/
