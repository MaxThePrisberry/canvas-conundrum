package constants

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPlayerLimits(t *testing.T) {
	t.Run("Player count constants", func(t *testing.T) {
		assert.Greater(t, MaxPlayers, MinPlayers, "MaxPlayers should be greater than MinPlayers")
		assert.GreaterOrEqual(t, MinPlayers, 1, "MinPlayers should be at least 1")
		assert.Equal(t, 4, MinPlayers, "MinPlayers should be 4")
		assert.Equal(t, 64, MaxPlayers, "MaxPlayers should be 64")
	})
}

func TestPhaseTiming(t *testing.T) {
	t.Run("Resource gathering timing", func(t *testing.T) {
		assert.Greater(t, ResourceGatheringRounds, 0, "Should have at least 1 round")
		assert.Greater(t, ResourceGatheringRoundDuration, 0, "Round duration should be positive")
		assert.Equal(t, 5, ResourceGatheringRounds, "Should have 5 rounds")
		assert.Equal(t, 60, ResourceGatheringRoundDuration, "Round should be 60 seconds")
	})
	
	t.Run("Trivia timing", func(t *testing.T) {
		assert.Greater(t, TriviaAnswerTime, 0, "Answer time should be positive")
		assert.Greater(t, TriviaGraceTime, 0, "Grace time should be positive")
		assert.Equal(t, 30, TriviaAnswerTime, "Answer time should be 30 seconds")
		assert.Equal(t, 30, TriviaGraceTime, "Grace time should be 30 seconds")
	})
	
	t.Run("Puzzle timing", func(t *testing.T) {
		assert.Greater(t, PuzzleBaseTime, 0, "Puzzle base time should be positive")
		assert.Greater(t, PostGameDuration, 0, "Post game duration should be positive")
		assert.Equal(t, 300, PuzzleBaseTime, "Puzzle base time should be 5 minutes")
		assert.Equal(t, 300, PostGameDuration, "Post game should be 5 minutes")
	})
	
	t.Run("Grid update timing", func(t *testing.T) {
		assert.Greater(t, GridUpdateInterval, 0, "Grid update interval should be positive")
		assert.Equal(t, 3, GridUpdateInterval, "Grid update interval should be 3 seconds")
		assert.Equal(t, 3*time.Second, GridUpdateIntervalDuration, "Duration should match interval")
	})
}

func TestTokenEconomics(t *testing.T) {
	t.Run("Token rewards", func(t *testing.T) {
		assert.Greater(t, BaseTokensPerCorrectAnswer, 0, "Base tokens should be positive")
		assert.Greater(t, RoleResourceMultiplier, 1.0, "Role multiplier should be greater than 1")
		assert.Greater(t, SpecialtyPointMultiplier, 1.0, "Specialty multiplier should be greater than 1")
		
		assert.Equal(t, 20, BaseTokensPerCorrectAnswer, "Base tokens should be 20")
		assert.Equal(t, 1.5, RoleResourceMultiplier, "Role multiplier should be 1.5")
		assert.Equal(t, 2.0, SpecialtyPointMultiplier, "Specialty multiplier should be 2.0")
	})
	
	t.Run("Token thresholds", func(t *testing.T) {
		thresholds := []int{
			AnchorTokenThreshold,
			ChronosTokenThreshold,
			GuideTokenThreshold,
			ClarityTokenThreshold,
		}
		
		for _, threshold := range thresholds {
			assert.Greater(t, threshold, 0, "All thresholds should be positive")
		}
		
		// Test specific values
		assert.Equal(t, 25, AnchorTokenThreshold, "Anchor threshold should be 25")
		assert.Equal(t, 20, ChronosTokenThreshold, "Chronos threshold should be 20")
		assert.Equal(t, 15, GuideTokenThreshold, "Guide threshold should be 15")
		assert.Equal(t, 30, ClarityTokenThreshold, "Clarity threshold should be 30")
	})
}

func TestSpecialtyConfiguration(t *testing.T) {
	t.Run("Specialty frequency", func(t *testing.T) {
		frequencies := []float64{
			SpecialtyQFreqEasy,
			SpecialtyQFreqMedium,
			SpecialtyQFreqHard,
		}
		
		for _, freq := range frequencies {
			assert.GreaterOrEqual(t, freq, 0.0, "Frequency should be non-negative")
			assert.LessOrEqual(t, freq, 1.0, "Frequency should not exceed 1.0")
		}
		
		// Test specific values
		assert.Equal(t, 0.2, SpecialtyQFreqEasy, "Easy specialty frequency should be 0.2")
		assert.Equal(t, 0.3, SpecialtyQFreqMedium, "Medium specialty frequency should be 0.3")
		assert.Equal(t, 0.4, SpecialtyQFreqHard, "Hard specialty frequency should be 0.4")
	})
	
	t.Run("Max specialties", func(t *testing.T) {
		assert.Greater(t, MaxSpecialtiesPerPlayer, 0, "Max specialties should be positive")
		assert.Equal(t, 1, MaxSpecialtiesPerPlayer, "Max specialties should be 1")
	})
}

func TestDifficultyModifiers(t *testing.T) {
	t.Run("Time multipliers", func(t *testing.T) {
		multipliers := []float64{
			EasyTimeMultiplier,
			MediumTimeMultiplier,
			HardTimeMultiplier,
		}
		
		for _, multiplier := range multipliers {
			assert.Greater(t, multiplier, 0.0, "Time multiplier should be positive")
		}
		
		// Test specific values
		assert.Equal(t, 1.2, EasyTimeMultiplier, "Easy multiplier should be 1.2")
		assert.Equal(t, 1.0, MediumTimeMultiplier, "Medium multiplier should be 1.0")
		assert.Equal(t, 0.8, HardTimeMultiplier, "Hard multiplier should be 0.8")
		
		// Easy should give more time than medium, medium more than hard
		assert.Greater(t, EasyTimeMultiplier, MediumTimeMultiplier)
		assert.Greater(t, MediumTimeMultiplier, HardTimeMultiplier)
	})
	
	t.Run("Threshold multipliers", func(t *testing.T) {
		multipliers := []float64{
			EasyThresholdMultiplier,
			MediumThresholdMultiplier,
			HardThresholdMultiplier,
		}
		
		for _, multiplier := range multipliers {
			assert.Greater(t, multiplier, 0.0, "Threshold multiplier should be positive")
		}
		
		// Easy should be easier to achieve (lower multiplier)
		assert.Less(t, EasyThresholdMultiplier, MediumThresholdMultiplier)
		assert.Less(t, MediumThresholdMultiplier, HardThresholdMultiplier)
	})
	
	t.Run("Specialty probability", func(t *testing.T) {
		probabilities := []float64{
			EasySpecialtyProbability,
			MediumSpecialtyProbability,
			HardSpecialtyProbability,
		}
		
		for _, prob := range probabilities {
			assert.GreaterOrEqual(t, prob, 0.0, "Probability should be non-negative")
			assert.LessOrEqual(t, prob, 1.0, "Probability should not exceed 1.0")
		}
		
		// Hard should have more specialty questions
		assert.Less(t, EasySpecialtyProbability, MediumSpecialtyProbability)
		assert.Less(t, MediumSpecialtyProbability, HardSpecialtyProbability)
	})
}

func TestMovementRestrictions(t *testing.T) {
	t.Run("Fragment move cooldown", func(t *testing.T) {
		assert.Greater(t, FragmentMoveCooldown, 0, "Move cooldown should be positive")
		assert.Equal(t, 2500, FragmentMoveCooldown, "Move cooldown should be 2500 milliseconds")
	})
}

func TestGridSizeFunction(t *testing.T) {
	t.Run("Grid size calculation", func(t *testing.T) {
		testCases := []struct {
			playerCount int
			expected    int
		}{
			{4, 3},   // 4 players -> 3x3 (≤9)
			{8, 3},   // 8 players -> 3x3 (≤9)
			{9, 3},   // 9 players -> 3x3 (≤9)
			{10, 4},  // 10 players -> 4x4 (≤16)
			{16, 4},  // 16 players -> 4x4 (≤16)
			{17, 5},  // 17 players -> 5x5 (≤25)
			{25, 5},  // 25 players -> 5x5 (≤25)
			{26, 6},  // 26 players -> 6x6 (≤36)
			{36, 6},  // 36 players -> 6x6 (≤36)
			{37, 7},  // 37 players -> 7x7 (≤49)
			{49, 7},  // 49 players -> 7x7 (≤49)
			{50, 8},  // 50 players -> 8x8 (>49)
			{64, 8},  // 64 players -> 8x8 (>49)
		}
		
		for _, tc := range testCases {
			gridSize := GetGridSizeForPlayerCount(tc.playerCount)
			assert.Equal(t, tc.expected, gridSize, "Grid size should be %d for %d players", tc.expected, tc.playerCount)
		}
	})
	
	t.Run("Edge cases", func(t *testing.T) {
		// Test minimum and maximum player counts
		minGridSize := GetGridSizeForPlayerCount(MinPlayers)
		maxGridSize := GetGridSizeForPlayerCount(MaxPlayers)
		
		assert.Greater(t, minGridSize, 0, "Minimum grid size should be positive")
		assert.Greater(t, maxGridSize, 0, "Maximum grid size should be positive")
		assert.GreaterOrEqual(t, maxGridSize, minGridSize, "Max grid size should be >= min grid size")
	})
}

func TestConstantRelationships(t *testing.T) {
	t.Run("Time relationships", func(t *testing.T) {
		// Trivia answer time should be reasonable compared to round duration
		assert.Less(t, TriviaAnswerTime, ResourceGatheringRoundDuration, "Answer time should be less than round duration")
		
		// Grace time should allow for processing
		assert.Greater(t, TriviaGraceTime, 0, "Grace time should allow for answer processing")
		
		// Grid updates should be frequent enough to feel responsive
		assert.Less(t, GridUpdateInterval, 10, "Grid updates should be frequent")
	})
	
	t.Run("Token economics balance", func(t *testing.T) {
		// Role multiplier should provide meaningful bonus
		assert.Greater(t, RoleResourceMultiplier, 1.25, "Role multiplier should provide meaningful bonus")
		
		// Specialty multiplier should be significant
		assert.Greater(t, SpecialtyPointMultiplier, RoleResourceMultiplier, "Specialty bonus should be greater than role bonus")
		
		// Thresholds should be achievable but require effort
		maxTokensPerRound := float64(BaseTokensPerCorrectAnswer) * SpecialtyPointMultiplier * RoleResourceMultiplier
		assert.Greater(t, maxTokensPerRound*2, float64(AnchorTokenThreshold), "Thresholds should be achievable in a few correct answers")
	})
}

func TestGameBalanceConstants(t *testing.T) {
	t.Run("All constants are defined", func(t *testing.T) {
		// Just ensure we can access all the constants without compilation errors
		constants := []interface{}{
			MinPlayers, MaxPlayers,
			ResourceGatheringRounds, ResourceGatheringRoundDuration,
			TriviaAnswerTime, TriviaGraceTime,
			PuzzleBaseTime, PostGameDuration,
			GridUpdateInterval, GridUpdateIntervalDuration,
			BaseTokensPerCorrectAnswer, RoleResourceMultiplier, SpecialtyPointMultiplier,
			AnchorTokenThreshold, ChronosTokenThreshold, GuideTokenThreshold, ClarityTokenThreshold,
			SpecialtyQFreqEasy, SpecialtyQFreqMedium, SpecialtyQFreqHard,
			MaxSpecialtiesPerPlayer,
			EasyTimeMultiplier, MediumTimeMultiplier, HardTimeMultiplier,
			EasyThresholdMultiplier, MediumThresholdMultiplier, HardThresholdMultiplier,
			EasySpecialtyProbability, MediumSpecialtyProbability, HardSpecialtyProbability,
			FragmentMoveCooldown,
			IndividualPuzzlePieces, ClarityBasePreviewTime,
			PiecesPreSolvedPerThreshold, MaxPreSolvedPieces,
			TimeExtensionPerThreshold, PreviewTimePerThreshold,
		}
		
		for _, constant := range constants {
			assert.NotNil(t, constant, "All constants should be defined")
		}
	})
}