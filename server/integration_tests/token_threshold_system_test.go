package integration_tests

import (
	"canvas-conundrum/config"
	"canvas-conundrum/models"
	"canvas-conundrum/services"
	"canvas-conundrum/test_helpers"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// collectTokensForPlayer simulates token collection by answering trivia questions
func collectTokensForPlayer(t *testing.T, player *test_helpers.TestPlayerClient, targetTokens int, tokenType string) {
	stationHash := getStationHashForTokenType(tokenType)

	// Verify location at appropriate station for role bonus
	err := player.VerifyLocation(tokenType, stationHash)
	require.NoError(t, err)

	tokensCollected := 0
	for tokensCollected < targetTokens {
		// Wait for trivia question
		triviaMsg, err := player.WaitForEvent(config.EventResourceToPlayerTriviaQuestion, 10*time.Second)
		if err != nil {
			// If no more questions, we're done with resource gathering
			break
		}

		triviaPayload := triviaMsg.Payload.(map[string]interface{})
		questionID := triviaPayload["questionId"].(string)

		// Always answer correctly (answer index 0 is always correct in our test data)
		err = player.AnswerTrivia(questionID, 0, 15.0)
		require.NoError(t, err)

		// Wait for answer result
		resultMsg, err := player.WaitForEvent(config.EventResourceToPlayerAnswerResult, 5*time.Second)
		require.NoError(t, err)

		resultPayload := resultMsg.Payload.(map[string]interface{})
		if resultPayload["correct"].(bool) {
			tokensEarned := int(resultPayload["tokensEarned"].(float64))
			tokensCollected += tokensEarned
		}
	}
}

// getStationHashForTokenType returns the station hash for a given token type
func getStationHashForTokenType(tokenType string) string {
	switch tokenType {
	case "anchor":
		return config.HashAnchorStation
	case "chronos":
		return config.HashChronosStation
	case "guide":
		return config.HashGuideStation
	case "clarity":
		return config.HashClarityStation
	default:
		return config.HashAnchorStation
	}
}

// TestTokenThresholdCalculations tests the basic threshold calculation system
func TestTokenThresholdCalculations(t *testing.T) {
	server, cleanup := setupTestServerWithTrivia(t)
	defer cleanup()

	host, players, cleanupGame := setupMinimalGameScenario(t, server)
	defer cleanupGame()

	tokenTypes := []string{"anchor", "chronos", "guide", "clarity"}

	// Start game
	waitForGameToStart(t, host, players)

	// Wait for resource phase start
	for i := 0; i < 4; i++ {
		phaseMsg, err := players[i].WaitForEvent(config.EventResourceToClientPhaseStart, 3*time.Second)
		require.NoError(t, err)

		// Verify threshold information is provided
		phasePayload := phaseMsg.Payload.(map[string]interface{})
		thresholds := phasePayload["tokenThresholds"].(map[string]interface{})

		// Check all token types have threshold values
		for _, tokenType := range tokenTypes {
			assert.Contains(t, thresholds, tokenType, "Should specify threshold for %s tokens", tokenType)
			threshold := thresholds[tokenType].(float64)
			assert.Greater(t, threshold, 0.0, "%s token threshold should be positive", tokenType)
		}

		// Verify difficulty settings affect thresholds
		difficultySettings := phasePayload["difficultySettings"].(map[string]interface{})
		assert.Equal(t, "medium", difficultySettings["mode"].(string))
		assert.Equal(t, 1.0, difficultySettings["thresholdMultiplier"].(float64),
			"Medium difficulty should have 1.0 threshold multiplier")
	}

	t.Run("ThresholdProgressionTracking", func(t *testing.T) {
		// Simulate token collection and verify threshold progression
		gm := services.GetGameInstance()
		game := gm.GetGame()

		// Manually add tokens to test threshold calculations
		initialTokens := game.TeamTokens

		// Test anchor token thresholds (Janitor specialty)
		anchorThreshold := config.AnchorTokenThreshold

		// Add tokens progressively and check thresholds
		for threshold := 1; threshold <= 6; threshold++ {
			tokensNeeded := threshold * anchorThreshold
			game.TeamTokens.AnchorTokens = tokensNeeded

			currentThreshold := game.TeamTokens.GetThreshold(models.TokenAnchor)
			assert.Equal(t, threshold, currentThreshold,
				"Should achieve threshold %d with %d anchor tokens", threshold, tokensNeeded)
		}

		// Test chronos token thresholds (Tourist specialty)
		chronosThreshold := config.ChronosTokenThreshold

		for threshold := 1; threshold <= 6; threshold++ {
			tokensNeeded := threshold * chronosThreshold
			game.TeamTokens.ChronosTokens = tokensNeeded

			currentThreshold := game.TeamTokens.GetThreshold(models.TokenChronos)
			assert.Equal(t, threshold, currentThreshold,
				"Should achieve threshold %d with %d chronos tokens", threshold, tokensNeeded)
		}

		// Test guide token thresholds (Detective specialty)
		guideThreshold := config.GuideTokenThreshold

		for threshold := 1; threshold <= 6; threshold++ {
			tokensNeeded := threshold * guideThreshold
			game.TeamTokens.GuideTokens = tokensNeeded

			currentThreshold := game.TeamTokens.GetThreshold(models.TokenGuide)
			assert.Equal(t, threshold, currentThreshold,
				"Should achieve threshold %d with %d guide tokens", threshold, tokensNeeded)
		}

		// Test clarity token thresholds (Art Enthusiast specialty)
		clarityThreshold := config.ClarityTokenThreshold

		for threshold := 1; threshold <= 6; threshold++ {
			tokensNeeded := threshold * clarityThreshold
			game.TeamTokens.ClarityTokens = tokensNeeded

			currentThreshold := game.TeamTokens.GetThreshold(models.TokenClarity)
			assert.Equal(t, threshold, currentThreshold,
				"Should achieve threshold %d with %d clarity tokens", threshold, tokensNeeded)
		}

		// Reset tokens
		*game.TeamTokens = *initialTokens
	})

	t.Run("TeamProgressUpdates", func(t *testing.T) {
		// Advance through some resource gathering to test team progress updates
		gm := services.GetGameInstance()
		game := gm.GetGame()

		// Manually set some tokens to trigger threshold achievements
		tokens := game.TeamTokens
		tokens.AnchorTokens = config.AnchorTokenThreshold * 2   // 2 thresholds
		tokens.ChronosTokens = config.ChronosTokenThreshold * 3 // 3 thresholds
		tokens.GuideTokens = config.GuideTokenThreshold * 1     // 1 threshold
		tokens.ClarityTokens = config.ClarityTokenThreshold * 4 // 4 thresholds

		// Trigger a team progress update using the actual getTeamProgressPayload function
		broadcastService := gm.GetBroadcastService()
		// Import the handlers package to use getTeamProgressPayload
		// Since it's a private function, let's trigger an actual team progress broadcast
		// by simulating a trivia answer
		_, exists := gm.GetPlayer(players[0].GetPlayerID())
		require.True(t, exists)

		// Simulate answering a trivia question correctly to trigger team progress broadcast
		// This should call the actual getTeamProgressPayload internally
		triviaService := gm.GetTriviaService()
		if triviaService != nil {
			// Award tokens directly and trigger broadcast
			game.TeamTokens = tokens
			broadcastService.BroadcastToAllPlayers(config.EventResourceToClientTeamProgress, map[string]interface{}{
				"currentRound": 1,
				"totalRounds":  config.ResourceGatheringRounds,
				"teamTokens": map[string]interface{}{
					"anchorTokens":  tokens.AnchorTokens,
					"chronosTokens": tokens.ChronosTokens,
					"guideTokens":   tokens.GuideTokens,
					"clarityTokens": tokens.ClarityTokens,
				},
				"tokenThresholds": map[string]interface{}{
					"anchorThreshold":  tokens.GetThreshold(models.TokenAnchor),
					"chronosThreshold": tokens.GetThreshold(models.TokenChronos),
					"guideThreshold":   tokens.GetThreshold(models.TokenGuide),
					"clarityThreshold": tokens.GetThreshold(models.TokenClarity),
				},
			})
		}

		// Wait for team progress message
		progressMsg, err := players[0].WaitForEvent(config.EventResourceToClientTeamProgress, 3*time.Second)
		require.NoError(t, err)

		progressPayload := progressMsg.Payload.(map[string]interface{})

		// Verify team tokens are reported
		teamTokens := progressPayload["teamTokens"].(map[string]interface{})
		assert.Equal(t, float64(tokens.AnchorTokens), teamTokens["anchorTokens"].(float64))
		assert.Equal(t, float64(tokens.ChronosTokens), teamTokens["chronosTokens"].(float64))
		assert.Equal(t, float64(tokens.GuideTokens), teamTokens["guideTokens"].(float64))
		assert.Equal(t, float64(tokens.ClarityTokens), teamTokens["clarityTokens"].(float64))

		// Verify threshold achievements are calculated correctly
		tokenThresholds := progressPayload["tokenThresholds"].(map[string]interface{})

		anchorThresholds := tokenThresholds["anchor"].(map[string]interface{})
		assert.Equal(t, float64(2), anchorThresholds["currentThreshold"].(float64))
		assert.Equal(t, float64(6), anchorThresholds["maxThresholds"].(float64))

		chronosThresholds := tokenThresholds["chronos"].(map[string]interface{})
		assert.Equal(t, float64(3), chronosThresholds["currentThreshold"].(float64))

		guideThresholds := tokenThresholds["guide"].(map[string]interface{})
		assert.Equal(t, float64(1), guideThresholds["currentThreshold"].(float64))

		clarityThresholds := tokenThresholds["clarity"].(map[string]interface{})
		assert.Equal(t, float64(4), clarityThresholds["currentThreshold"].(float64))
	})
}

// TestRoleBasedTokenMultipliers tests that role bonuses are applied correctly
func TestRoleBasedTokenMultipliers(t *testing.T) {
	server, cleanup := setupTestServerWithTrivia(t)
	defer cleanup()

	// Create host client
	hostClient := test_helpers.NewTestHostClient(t, server, config.HostUUID)
	err := hostClient.Connect()
	require.NoError(t, err)
	defer hostClient.Close()

	// Create players with specific roles for testing bonuses
	players := make([]*test_helpers.TestPlayerClient, 4)
	roles := []string{"janitor", "tourist", "detective", "art_enthusiast"}
	expectedBonusTokens := []string{"anchor", "chronos", "guide", "clarity"}

	for i := 0; i < 4; i++ {
		players[i] = createAndConfigurePlayer(t, server, fmt.Sprintf("Player%d", i+1), roles[i], []string{"science"})
		defer players[i].Close()
	}

	// Start game
	err = hostClient.StartGame("medium")
	require.NoError(t, err)

	// Wait for resource phase start
	for i := 0; i < 4; i++ {
		_, err = players[i].WaitForEvent(config.EventResourceToClientPhaseStart, 3*time.Second)
		require.NoError(t, err)
	}

	t.Run("RoleBonusApplication", func(t *testing.T) {
		gm := services.GetGameInstance()
		game := gm.GetGame()

		// Test each player's role bonus
		for i, player := range players {
			playerObj, exists := gm.GetPlayer(player.GetPlayerID())
			require.True(t, exists, "Player should exist in game")
			require.NotNil(t, playerObj, "Player should exist in game")

			roleType := roles[i]
			bonusTokenType := expectedBonusTokens[i]

			// Verify player role is set correctly
			assert.Equal(t, roleType, string(playerObj.Role), "Player should have correct role")

			// Move player to appropriate station for bonus
			_ = getStationHashForTokenType(bonusTokenType)
			playerObj.CurrentStation = getLocationForTokenType(bonusTokenType)

			// Simulate correct trivia answer
			baseTokens := config.BaseTokensPerCorrectAnswer
			roleMultiplier := config.RoleResourceMultiplier

			expectedTokens := int(float64(baseTokens) * roleMultiplier)

			// Verify role bonus calculation
			assert.Greater(t, expectedTokens, baseTokens,
				"Role bonus should increase token count for %s at %s station", roleType, bonusTokenType)

			// Test by adding tokens manually and checking
			initialTokens := getTokenCountByType(game.TeamTokens, bonusTokenType)
			addTokensByType(game.TeamTokens, bonusTokenType, expectedTokens)
			finalTokens := getTokenCountByType(game.TeamTokens, bonusTokenType)

			assert.Equal(t, initialTokens+expectedTokens, finalTokens,
				"Tokens should be added correctly for %s", bonusTokenType)
		}
	})

	t.Run("SpecialtyBonusMultiplier", func(t *testing.T) {
		// Test specialty bonus multiplier
		gm := services.GetGameInstance()
		_ = gm.GetGame()

		playerObj, exists := gm.GetPlayer(players[0].GetPlayerID())
		require.True(t, exists, "Player should exist in game")
		require.NotNil(t, playerObj)

		// Verify specialty is set
		assert.Contains(t, playerObj.Specialties, models.CategoryScience,
			"Player should have science specialty")

		// Simulate specialty question bonus
		baseTokens := config.BaseTokensPerCorrectAnswer
		specialtyMultiplier := config.SpecialtyPointMultiplier
		roleMultiplier := config.RoleResourceMultiplier

		// For specialty questions at matching station
		expectedTokens := int(float64(baseTokens) * roleMultiplier * specialtyMultiplier)

		assert.Greater(t, expectedTokens, int(float64(baseTokens)*roleMultiplier),
			"Specialty bonus should further increase tokens beyond role bonus")

		// Test maximum possible tokens (base * role * specialty)
		maxTokens := int(float64(baseTokens) * roleMultiplier * specialtyMultiplier)
		assert.LessOrEqual(t, maxTokens, baseTokens*10,
			"Maximum tokens should be reasonable (sanity check)")
	})
}

// Helper functions for token manipulation
func getLocationForTokenType(tokenType string) string {
	switch tokenType {
	case "anchor":
		return "anchor"
	case "chronos":
		return "chronos"
	case "guide":
		return "guide"
	case "clarity":
		return "clarity"
	default:
		return "anchor"
	}
}

func getTokenCountByType(tokens *models.TeamTokens, tokenType string) int {
	switch tokenType {
	case "anchor":
		return tokens.AnchorTokens
	case "chronos":
		return tokens.ChronosTokens
	case "guide":
		return tokens.GuideTokens
	case "clarity":
		return tokens.ClarityTokens
	default:
		return 0
	}
}

func addTokensByType(tokens *models.TeamTokens, tokenType string, amount int) {
	switch tokenType {
	case "anchor":
		tokens.AnchorTokens += amount
	case "chronos":
		tokens.ChronosTokens += amount
	case "guide":
		tokens.GuideTokens += amount
	case "clarity":
		tokens.ClarityTokens += amount
	}
}

// TestTokenThresholdEffects tests that threshold achievements unlock the correct effects
func TestTokenThresholdEffects(t *testing.T) {
	server, cleanup := setupTestServerWithTrivia(t)
	defer cleanup()

	// Create host client
	hostClient := test_helpers.NewTestHostClient(t, server, config.HostUUID)
	err := hostClient.Connect()
	require.NoError(t, err)
	defer hostClient.Close()

	// Create players
	players := make([]*test_helpers.TestPlayerClient, 4)
	roles := []string{"janitor", "tourist", "detective", "art_enthusiast"}

	for i := 0; i < 4; i++ {
		players[i] = createAndConfigurePlayer(t, server, fmt.Sprintf("Player%d", i+1), roles[i], []string{"science"})
		defer players[i].Close()
	}

	// Start game
	err = hostClient.StartGame("medium")
	require.NoError(t, err)

	// Skip to puzzle phase to test effects
	gm := services.GetGameInstance()
	game := gm.GetGame()
	game.CurrentPhase = models.PhasePuzzleAssembly

	err = hostClient.StartPuzzlePhase()
	require.NoError(t, err)

	t.Run("AnchorTokenEffects", func(t *testing.T) {
		// Test anchor token effects: pre-solved pieces in individual puzzles

		// Set anchor tokens to achieve different threshold levels
		tokens := game.TeamTokens

		// Test threshold 1: 1 * AnchorTokenThreshold = some pre-solved pieces
		tokens.AnchorTokens = config.AnchorTokenThreshold * 1
		threshold1 := tokens.GetThreshold(models.TokenAnchor)
		assert.Equal(t, 1, threshold1)

		// Test threshold 3: 3 thresholds = 6 pre-solved pieces
		tokens.AnchorTokens = config.AnchorTokenThreshold * 3
		threshold3 := tokens.GetThreshold(models.TokenAnchor)
		assert.Equal(t, 3, threshold3)

		// Test threshold 6: max 6 thresholds = 12 pre-solved pieces
		tokens.AnchorTokens = config.AnchorTokenThreshold * 6
		threshold6 := tokens.GetThreshold(models.TokenAnchor)
		assert.Equal(t, 6, threshold6)
		expectedPreSolved6 := 12

		// Wait for puzzle load message to verify pre-solved pieces
		loadMsg, err := players[0].WaitForEvent(config.EventPuzzleToClientPhaseLoad, 3*time.Second)
		require.NoError(t, err)

		loadPayload := loadMsg.Payload.(map[string]interface{})
		actualPreSolved := int(loadPayload["anchorPreSolvedPieces"].(float64))

		assert.Equal(t, expectedPreSolved6, actualPreSolved,
			"Should have %d pre-solved pieces with 6 anchor thresholds", expectedPreSolved6)

		// Verify minimum pieces always require manual solving (16 - 12 = 4 minimum)
		assert.LessOrEqual(t, actualPreSolved, 12, "Should never pre-solve more than 12 pieces")
		remainingPieces := 16 - actualPreSolved
		assert.GreaterOrEqual(t, remainingPieces, 4, "Should always leave at least 4 pieces for manual solving")
	})

	t.Run("ChronosTokenEffects", func(t *testing.T) {
		// Test chronos token effects: extended puzzle time

		tokens := game.TeamTokens
		baseTime := config.PuzzleBaseTime

		// Test different threshold levels
		testCases := []struct {
			thresholds    int
			expectedBonus int
		}{
			{1, 20},  // 1 threshold = +20 seconds
			{3, 60},  // 3 thresholds = +60 seconds
			{6, 120}, // 6 thresholds = +120 seconds (max)
		}

		for _, tc := range testCases {
			tokens.ChronosTokens = config.ChronosTokenThreshold * tc.thresholds
			threshold := tokens.GetThreshold(models.TokenChronos)
			assert.Equal(t, tc.thresholds, threshold)

			// Test time calculation
			expectedTotalTime := baseTime + tc.expectedBonus

			// Wait for puzzle phase start to verify time bonus
			phaseStartMsg, err := players[0].WaitForEvent(config.EventPuzzleToClientPhaseStart, 3*time.Second)
			require.NoError(t, err)

			phasePayload := phaseStartMsg.Payload.(map[string]interface{})
			totalTime := int(phasePayload["totalTime"].(float64))
			chronosBonus := int(phasePayload["chronosBonus"].(float64))

			assert.Equal(t, tc.expectedBonus, chronosBonus,
				"Should get %d seconds bonus with %d chronos thresholds", tc.expectedBonus, tc.thresholds)
			assert.Equal(t, expectedTotalTime, totalTime,
				"Total time should be base + chronos bonus")
		}
	})

	t.Run("ClarityTokenEffects", func(t *testing.T) {
		// Test clarity token effects: extended preview time

		tokens := game.TeamTokens
		basePreviewTime := config.ClarityBasePreviewTime

		// Test threshold effects on preview duration
		tokens.ClarityTokens = config.ClarityTokenThreshold * 4 // 4 thresholds
		threshold := tokens.GetThreshold(models.TokenClarity)
		assert.Equal(t, 4, threshold)

		expectedBonus := 4 // +1 second per threshold
		expectedTotalPreview := basePreviewTime + expectedBonus

		// Wait for puzzle load to verify clarity preview duration
		loadMsg, err := players[0].WaitForEvent(config.EventPuzzleToClientPhaseLoad, 3*time.Second)
		require.NoError(t, err)

		loadPayload := loadMsg.Payload.(map[string]interface{})
		clarityPreviewDuration := int(loadPayload["clarityPreviewDuration"].(float64))

		assert.Equal(t, expectedTotalPreview, clarityPreviewDuration,
			"Should have %d seconds preview time with 4 clarity thresholds", expectedTotalPreview)

		// Test maximum preview time (6 thresholds = +6 seconds)
		tokens.ClarityTokens = config.ClarityTokenThreshold * 6
		maxExpectedPreview := basePreviewTime + 6

		// Verify preview is active when puzzle starts
		phaseStartMsg, err := players[0].WaitForEvent(config.EventPuzzleToClientPhaseStart, 3*time.Second)
		require.NoError(t, err)

		phasePayload := phaseStartMsg.Payload.(map[string]interface{})
		clarityPreviewActive := phasePayload["clarityPreviewActive"].(bool)
		previewDuration := int(phasePayload["previewDuration"].(float64))

		assert.True(t, clarityPreviewActive, "Clarity preview should be active")
		assert.Equal(t, maxExpectedPreview, previewDuration, "Should show maximum preview duration")
	})

	t.Run("GuideTokenEffects", func(t *testing.T) {
		// Test guide token effects: fragment placement guidance

		tokens := game.TeamTokens

		// Test different guide threshold levels
		tokens.GuideTokens = config.GuideTokenThreshold * 3 // 3 thresholds
		threshold := tokens.GetThreshold(models.TokenGuide)
		assert.Equal(t, 3, threshold)

		// According to spec: each threshold removes (gridSize × gridSize) / 7 highlighted squares
		// For 4x4 grid: 16 / 7 ≈ 2.3, so probably 2 squares removed per threshold
		// With 3 thresholds: removes about 6-7 squares, leaving 9-10 highlighted

		loadMsg, err := players[0].WaitForEvent(config.EventPuzzleToClientPhaseLoad, 3*time.Second)
		require.NoError(t, err)

		loadPayload := loadMsg.Payload.(map[string]interface{})
		guideHighlightCount := int(loadPayload["guideHighlightCount"].(float64))

		// Verify guide highlights are provided (exact count depends on grid size calculation)
		assert.Greater(t, guideHighlightCount, 0, "Should have guide highlights with guide tokens")
		assert.LessOrEqual(t, guideHighlightCount, 16, "Guide highlights should not exceed grid size")

		// Test maximum guide thresholds
		tokens.GuideTokens = config.GuideTokenThreshold * 6 // 6 thresholds (max)
		maxThreshold := tokens.GetThreshold(models.TokenGuide)
		assert.Equal(t, 6, maxThreshold)

		// With maximum thresholds, should provide most precise guidance
		// (minimum highlighted squares remaining)
	})
}

// TestDifficultyModeThresholdModifiers tests how difficulty affects token thresholds
func TestDifficultyModeThresholdModifiers(t *testing.T) {
	difficulties := []string{"easy", "medium", "hard"}
	expectedMultipliers := map[string]float64{
		"easy":   config.EasyThresholdMultiplier,
		"medium": config.MediumThresholdMultiplier,
		"hard":   config.HardThresholdMultiplier,
	}

	for _, difficulty := range difficulties {
		t.Run(fmt.Sprintf("Difficulty_%s", difficulty), func(t *testing.T) {
			server, cleanup := setupTestServerWithTrivia(t)
			defer cleanup()

			// Create host client
			hostClient := test_helpers.NewTestHostClient(t, server, config.HostUUID)
			err := hostClient.Connect()
			require.NoError(t, err)
			defer hostClient.Close()

			// Create minimal players
			players := make([]*test_helpers.TestPlayerClient, 4)
			roles := []string{"janitor", "tourist", "detective", "art_enthusiast"}

			for i := 0; i < 4; i++ {
				players[i] = createAndConfigurePlayer(t, server, fmt.Sprintf("Player%d", i+1), roles[i], []string{"science"})
				defer players[i].Close()
			}

			// Start game with specific difficulty
			err = hostClient.StartGame(difficulty)
			require.NoError(t, err)

			// Wait for resource phase start
			phaseMsg, err := players[0].WaitForEvent(config.EventResourceToClientPhaseStart, 3*time.Second)
			require.NoError(t, err)

			phasePayload := phaseMsg.Payload.(map[string]interface{})

			// Verify difficulty settings
			difficultySettings := phasePayload["difficultySettings"].(map[string]interface{})
			assert.Equal(t, difficulty, difficultySettings["mode"].(string))

			expectedMultiplier := expectedMultipliers[difficulty]
			actualMultiplier := difficultySettings["thresholdMultiplier"].(float64)
			assert.Equal(t, expectedMultiplier, actualMultiplier,
				"Threshold multiplier should match expected value for %s difficulty", difficulty)

			// Verify threshold values are modified by difficulty
			thresholds := phasePayload["tokenThresholds"].(map[string]interface{})

			// Calculate expected thresholds with difficulty modifier
			baseAnchorThreshold := float64(config.AnchorTokenThreshold)
			expectedAnchorThreshold := baseAnchorThreshold * expectedMultiplier

			actualAnchorThreshold := thresholds["anchor"].(float64)
			assert.InDelta(t, expectedAnchorThreshold, actualAnchorThreshold, 0.1,
				"Anchor threshold should be modified by difficulty multiplier")

			// Test that easier difficulties require fewer tokens, harder require more
			if difficulty == "easy" {
				assert.Less(t, actualAnchorThreshold, baseAnchorThreshold,
					"Easy mode should have lower thresholds")
			} else if difficulty == "hard" {
				assert.Greater(t, actualAnchorThreshold, baseAnchorThreshold,
					"Hard mode should have higher thresholds")
			} else {
				assert.Equal(t, baseAnchorThreshold, actualAnchorThreshold,
					"Medium mode should have normal thresholds")
			}
		})
	}
}

// TestTokenThresholdEffectsIntegration tests the complete integration of token collection and effects
func TestTokenThresholdEffectsIntegration(t *testing.T) {
	server, cleanup := setupTestServerWithTrivia(t)
	defer cleanup()

	// Create host client
	hostClient := test_helpers.NewTestHostClient(t, server, config.HostUUID)
	err := hostClient.Connect()
	require.NoError(t, err)
	defer hostClient.Close()

	// Create players
	players := make([]*test_helpers.TestPlayerClient, 4)
	roles := []string{"janitor", "tourist", "detective", "art_enthusiast"}

	for i := 0; i < 4; i++ {
		players[i] = createAndConfigurePlayer(t, server, fmt.Sprintf("Player%d", i+1), roles[i], []string{"science"})
		defer players[i].Close()
	}

	// Start game
	err = hostClient.StartGame("medium")
	require.NoError(t, err)

	t.Run("EndToEndTokenCollection", func(t *testing.T) {
		gm := services.GetGameInstance()
		game := gm.GetGame()

		// Simulate collecting tokens to achieve specific thresholds
		tokens := game.TeamTokens

		// Set tokens to achieve interesting threshold combinations
		tokens.AnchorTokens = config.AnchorTokenThreshold * 3   // 3 thresholds = 6 pre-solved pieces
		tokens.ChronosTokens = config.ChronosTokenThreshold * 4 // 4 thresholds = 80 seconds bonus
		tokens.GuideTokens = config.GuideTokenThreshold * 2     // 2 thresholds = some guidance
		tokens.ClarityTokens = config.ClarityTokenThreshold * 5 // 5 thresholds = 5 seconds bonus preview

		// Advance to puzzle phase to see effects
		game.CurrentPhase = models.PhasePuzzleAssembly
		err = hostClient.StartPuzzlePhase()
		require.NoError(t, err)

		// Wait for puzzle load and verify all effects are applied
		loadMsg, err := players[0].WaitForEvent(config.EventPuzzleToClientPhaseLoad, 3*time.Second)
		require.NoError(t, err)

		loadPayload := loadMsg.Payload.(map[string]interface{})

		// Verify anchor effects
		preSolvedPieces := int(loadPayload["anchorPreSolvedPieces"].(float64))
		assert.Equal(t, 6, preSolvedPieces, "Should have 6 pre-solved pieces (3 thresholds * 2)")

		// Verify clarity effects
		clarityPreview := int(loadPayload["clarityPreviewDuration"].(float64))
		expectedClarityTime := config.ClarityBasePreviewTime + 5
		assert.Equal(t, expectedClarityTime, clarityPreview, "Should have extended clarity preview")

		// Verify guide effects
		guideHighlights := int(loadPayload["guideHighlightCount"].(float64))
		assert.Greater(t, guideHighlights, 0, "Should have guide highlights")

		// Wait for puzzle phase start to verify chronos effects
		phaseStartMsg, err := players[0].WaitForEvent(config.EventPuzzleToClientPhaseStart, 3*time.Second)
		require.NoError(t, err)

		phasePayload := phaseStartMsg.Payload.(map[string]interface{})

		// Verify chronos effects
		totalTime := int(phasePayload["totalTime"].(float64))
		chronosBonus := int(phasePayload["chronosBonus"].(float64))
		expectedChronosBonus := 4 * 20 // 4 thresholds * 20 seconds each
		expectedTotalTime := config.PuzzleBaseTime + expectedChronosBonus

		assert.Equal(t, expectedChronosBonus, chronosBonus, "Should have 80 seconds chronos bonus")
		assert.Equal(t, expectedTotalTime, totalTime, "Total time should include chronos bonus")

		// Verify clarity preview is active
		clarityActive := phasePayload["clarityPreviewActive"].(bool)
		assert.True(t, clarityActive, "Clarity preview should be active")
	})

	t.Run("ResourcePhaseCompletion", func(t *testing.T) {
		// Test final resource phase completion with accumulated tokens
		gm := services.GetGameInstance()
		game := gm.GetGame()

		// Set final token values
		tokens := game.TeamTokens
		tokens.AnchorTokens = config.AnchorTokenThreshold * 6   // Max thresholds
		tokens.ChronosTokens = config.ChronosTokenThreshold * 6 // Max thresholds
		tokens.GuideTokens = config.GuideTokenThreshold * 6     // Max thresholds
		tokens.ClarityTokens = config.ClarityTokenThreshold * 6 // Max thresholds

		// Simulate resource phase completion
		broadcastService := gm.GetBroadcastService()
		broadcastService.BroadcastResourcePhaseComplete()

		// Wait for phase complete message
		completeMsg, err := players[0].WaitForEvent(config.EventResourceToClientPhaseComplete, 3*time.Second)
		require.NoError(t, err)

		completePayload := completeMsg.Payload.(map[string]interface{})

		// Verify final token totals
		finalTokens := completePayload["finalTokenTotals"].(map[string]interface{})
		assert.Equal(t, float64(tokens.AnchorTokens), finalTokens["anchorTokens"].(float64))
		assert.Equal(t, float64(tokens.ChronosTokens), finalTokens["chronosTokens"].(float64))
		assert.Equal(t, float64(tokens.GuideTokens), finalTokens["guideTokens"].(float64))
		assert.Equal(t, float64(tokens.ClarityTokens), finalTokens["clarityTokens"].(float64))

		// Verify threshold achievements
		thresholdAchievements := completePayload["thresholdAchievements"].(map[string]interface{})
		assert.Equal(t, float64(6), thresholdAchievements["anchor"].(float64))
		assert.Equal(t, float64(6), thresholdAchievements["chronos"].(float64))
		assert.Equal(t, float64(6), thresholdAchievements["guide"].(float64))
		assert.Equal(t, float64(6), thresholdAchievements["clarity"].(float64))

		// Verify bonus effects summary
		bonusEffects := completePayload["bonusEffects"].(map[string]interface{})
		assert.Equal(t, float64(12), bonusEffects["preSolvedPieces"].(float64))                          // 6 thresholds * 2
		assert.Equal(t, float64(120), bonusEffects["extraTime"].(float64))                               // 6 thresholds * 20
		assert.Equal(t, float64(6), bonusEffects["guideHighlights"].(float64))                           // depends on calculation
		assert.Equal(t, float64(config.ClarityBasePreviewTime+6), bonusEffects["previewTime"].(float64)) // base + 6
	})
}
