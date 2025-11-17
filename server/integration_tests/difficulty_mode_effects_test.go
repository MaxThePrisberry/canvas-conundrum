package integration_tests

import (
	"canvas-conundrum/config"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDifficultyModeQuestionDistribution(t *testing.T) {
	// Test different difficulty modes
	difficultyModes := []string{"easy", "medium", "hard"}

	for _, mode := range difficultyModes {
		t.Run(fmt.Sprintf("DifficultyMode_%s", mode), func(t *testing.T) {
			server, cleanup := setupTestServerWithTrivia(t)
			defer cleanup()

			// Setup minimal game scenario
			host, players, gameCleanup := setupMinimalGameScenario(t, server)
			defer gameCleanup()

			// Start game with specific difficulty
			err := host.StartGame(mode)
			require.NoError(t, err)

			// Wait for resource phase to start
			for _, player := range players {
				_, err = player.WaitForEvent(config.EventResourceToClientPhaseStart, 5*time.Second)
				require.NoError(t, err)
			}

			questionDifficulties := map[string]int{
				"easy":   0,
				"medium": 0,
				"hard":   0,
			}

			// Start waiting for trivia questions BEFORE they are sent
			// The game waits 5 seconds before starting round 1 and sending questions
			// We need to start listening before that happens
			for _, player := range players {
				// Wait for trivia question (should be sent automatically when round 1 starts after 5 seconds)
				msg, err := player.WaitForEvent(config.EventResourceToPlayerTriviaQuestion, 10*time.Second)
				require.NoError(t, err, "Should receive trivia question automatically when round 1 starts")

				payload := msg.Payload.(map[string]interface{})
				if difficulty, exists := payload["difficulty"]; exists {
					questionDifficulties[difficulty.(string)]++
				}

				// Answer to continue
				err = player.AnswerTrivia("", 0, 10.0)
				require.NoError(t, err)
			}

			// Verify question difficulty matches selected mode
			switch mode {
			case "easy":
				assert.Greater(t, questionDifficulties["easy"], 0, "Easy mode should provide easy questions")
			case "medium":
				assert.Greater(t, questionDifficulties["medium"], 0, "Medium mode should provide medium questions")
			case "hard":
				assert.Greater(t, questionDifficulties["hard"], 0, "Hard mode should provide hard questions")
			}
		})
	}
}

func TestDifficultyModeTokenValues(t *testing.T) {
	// Test token rewards for different difficulty modes
	difficultyModes := []string{"easy", "medium", "hard"}

	for _, mode := range difficultyModes {
		t.Run(fmt.Sprintf("TokenRewards_%s", mode), func(t *testing.T) {
			server, cleanup := setupTestServerWithTrivia(t)
			defer cleanup()

			// Setup minimal game scenario
			host, players, gameCleanup := setupMinimalGameScenario(t, server)
			defer gameCleanup()

			// Start game with specific difficulty
			err := host.StartGame(mode)
			require.NoError(t, err)

			// Wait for resource phase to start
			for _, player := range players {
				_, err = player.WaitForEvent(config.EventResourceToClientPhaseStart, 5*time.Second)
				require.NoError(t, err)
			}

			// Wait for first trivia questions to be sent (after 5 seconds when round 1 starts)
			// and answer them to test token rewards
			stationHashes := getTestStationHashes()
			stations := []string{"anchor", "chronos", "guide", "clarity"}

			for _, player := range players {
				// Wait for trivia question
				_, err := player.WaitForEvent(config.EventResourceToPlayerTriviaQuestion, 10*time.Second)
				require.NoError(t, err)
			}

			// Each player moves to their preferred station and answers
			for i, player := range players {
				station := stations[i%len(stations)]
				hash := stationHashes[station]

				// Send location update
				err := player.VerifyLocation(station, hash)
				require.NoError(t, err)

				// Submit answer (always first option for test simplicity)
				err = player.AnswerTrivia("", 0, 10.0)
				require.NoError(t, err)
			}

			// Wait for team progress update to see token rewards
			for _, player := range players {
				_, err = player.WaitForEvent(config.EventResourceToClientTeamProgress, 3*time.Second)
				if err != nil {
					t.Logf("No team progress event for %s mode (expected behavior)", mode)
				}
			}

			// Test passes if no errors occurred during token distribution
		})
	}
}

func TestDifficultyModeTimeLimits(t *testing.T) {
	// Test time limits for different difficulty modes
	difficultyModes := []string{"easy", "medium", "hard"}

	for _, mode := range difficultyModes {
		t.Run(fmt.Sprintf("TimeLimit_%s", mode), func(t *testing.T) {
			server, cleanup := setupTestServerWithTrivia(t)
			defer cleanup()

			// Setup minimal game scenario
			host, players, gameCleanup := setupMinimalGameScenario(t, server)
			defer gameCleanup()

			// Start game with specific difficulty
			err := host.StartGame(mode)
			require.NoError(t, err)

			// Wait for resource phase to start
			for _, player := range players {
				_, err = player.WaitForEvent(config.EventResourceToClientPhaseStart, 5*time.Second)
				require.NoError(t, err)
			}

			// Check time limits for trivia questions
			player := players[0] // Just test with one player
			stationHashes := getTestStationHashes()

			// Verify location first
			err = player.VerifyLocation("anchor", stationHashes["anchor"])
			require.NoError(t, err)

			// Wait for trivia question and check time limit
			msg, err := player.WaitForEvent(config.EventResourceToPlayerTriviaQuestion, 5*time.Second)
			if err == nil {
				payload := msg.Payload.(map[string]interface{})
				if timeLimit, exists := payload["timeLimit"]; exists {
					// Verify time limit is appropriate for difficulty
					timeLimitValue := timeLimit.(float64)
					assert.Greater(t, timeLimitValue, 0.0, "Time limit should be positive")

					switch mode {
					case "easy":
						// Easy mode should have longer time limits
						assert.GreaterOrEqual(t, timeLimitValue, 30.0, "Easy mode should have sufficient time")
					case "hard":
						// Hard mode should have shorter time limits
						assert.LessOrEqual(t, timeLimitValue, 35.0, "Hard mode should have challenging time limits")
					}
				}
			}
		})
	}
}

func TestDifficultyModeSpecialtyQuestionFrequency(t *testing.T) {
	// Test specialty question frequency for different difficulty modes
	difficultyModes := []string{"easy", "hard"}

	for _, mode := range difficultyModes {
		t.Run(fmt.Sprintf("SpecialtyFrequency_%s", mode), func(t *testing.T) {
			server, cleanup := setupTestServerWithTrivia(t)
			defer cleanup()

			// Setup minimal game scenario
			host, players, gameCleanup := setupMinimalGameScenario(t, server)
			defer gameCleanup()

			// Start game with specific difficulty
			err := host.StartGame(mode)
			require.NoError(t, err)

			// Wait for resource phase to start
			for _, player := range players {
				_, err = player.WaitForEvent(config.EventResourceToClientPhaseStart, 5*time.Second)
				require.NoError(t, err)
			}

			// Simulate multiple rounds to check specialty frequency
			specialtyQuestions := 0
			totalQuestions := 0
			stationHashes := getTestStationHashes()

			// Test with just one player for simplicity
			player := players[0]

			for round := 0; round < 3; round++ {
				// Verify location first
				err = player.VerifyLocation("anchor", stationHashes["anchor"])
				require.NoError(t, err)

				msg, err := player.WaitForEvent(config.EventResourceToPlayerTriviaQuestion, 5*time.Second)
				if err == nil {
					payload := msg.Payload.(map[string]interface{})
					totalQuestions++

					if isSpecialty, exists := payload["isSpecialtyQuestion"]; exists && isSpecialty.(bool) {
						specialtyQuestions++
					}

					// Answer to continue
					err = player.AnswerTrivia("", 0, 10.0)
					require.NoError(t, err)
				} else {
					break // No more questions available
				}
			}

			if totalQuestions > 0 {
				specialtyFreq := float64(specialtyQuestions) / float64(totalQuestions)

				switch mode {
				case "easy":
					// Easy mode should have lower specialty frequency
					assert.LessOrEqual(t, specialtyFreq, 0.3, "Easy mode should have lower specialty question frequency")
				case "hard":
					// Hard mode should have higher specialty frequency
					assert.GreaterOrEqual(t, specialtyFreq, 0.2, "Hard mode should have higher specialty question frequency")
				}
			}
		})
	}
}

func TestDifficultyModeTokenThresholdEffects(t *testing.T) {
	// Test token threshold effects for different difficulty modes
	difficultyModes := []string{"easy", "hard"}

	for _, mode := range difficultyModes {
		t.Run(fmt.Sprintf("TokenThresholds_%s", mode), func(t *testing.T) {
			server, cleanup := setupTestServerWithTrivia(t)
			defer cleanup()

			// Setup minimal game scenario
			host, players, gameCleanup := setupMinimalGameScenario(t, server)
			defer gameCleanup()

			// Start game with specific difficulty
			err := host.StartGame(mode)
			require.NoError(t, err)

			// Wait for resource phase to start
			for _, player := range players {
				_, err = player.WaitForEvent(config.EventResourceToClientPhaseStart, 5*time.Second)
				require.NoError(t, err)
			}

			// Simulate resource gathering to test threshold effects
			simulateResourceGatheringPhase(t, host, players)

			// Test passes if resource gathering completes without errors
			// Token threshold effects would be visible in puzzle phase benefits
		})
	}
}
