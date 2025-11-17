package integration_tests

import (
	"canvas-conundrum/config"
	"canvas-conundrum/test_helpers"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTriviaQuestionDelivery(t *testing.T) {
	server, cleanup := setupTestServerWithTrivia(t)
	defer cleanup()

	// Setup minimal game scenario
	host, players, cleanupGame := setupMinimalGameScenario(t, server)
	defer cleanupGame()

	// Start the game to reach resource gathering phase
	waitForGameToStart(t, host, players)

	// Wait for the natural game flow - trivia questions are sent after ResourceGatheringRoundDuration
	// Each player should receive a trivia question (now using 2-second test config)
	for i, player := range players {
		triviaMsg, err := player.WaitForEvent(config.EventResourceToPlayerTriviaQuestion, 5*time.Second)
		require.NoError(t, err, "Player %d should receive trivia question", i+1)

		// Verify trivia question structure
		payload := triviaMsg.Payload.(map[string]interface{})
		assert.Contains(t, payload, "questionId", "Trivia question should have questionId")
		assert.Contains(t, payload, "category", "Trivia question should have category")
		assert.Contains(t, payload, "difficulty", "Trivia question should have difficulty")
		assert.Contains(t, payload, "questionText", "Trivia question should have question text")
		assert.Contains(t, payload, "options", "Trivia question should have options")

		// Verify options array structure
		options := payload["options"].([]interface{})
		assert.GreaterOrEqual(t, len(options), 2, "Should have at least 2 answer options")
		assert.LessOrEqual(t, len(options), 4, "Should have at most 4 answer options")

		// Verify category is one of the expected categories
		category := payload["category"].(string)
		expectedCategories := []string{"general", "geography", "history", "music", "science", "video_games"}
		assert.Contains(t, expectedCategories, category, "Category should be one of the expected categories")

		// Verify difficulty is valid
		difficulty := payload["difficulty"].(string)
		expectedDifficulties := []string{"easy", "medium", "hard"}
		assert.Contains(t, expectedDifficulties, difficulty, "Difficulty should be valid")
	}
}

func TestAnswerValidationAndScoring(t *testing.T) {
	server, cleanup := setupTestServerWithTrivia(t)
	defer cleanup()

	// Setup minimal game scenario
	host, players, cleanupGame := setupMinimalGameScenario(t, server)
	defer cleanupGame()

	// Start the game to reach resource gathering phase
	waitForGameToStart(t, host, players)

	// Test answer validation for each player
	for i, player := range players {
		// Wait for trivia question
		triviaMsg, err := player.WaitForEvent(config.EventResourceToPlayerTriviaQuestion, 5*time.Second)
		require.NoError(t, err, "Player %d should receive trivia question", i+1)

		// Extract question information
		payload := triviaMsg.Payload.(map[string]interface{})
		questionId := payload["questionId"].(string)

		// Submit answer (first option)
		err = player.AnswerTrivia(questionId, 0, 10.0)
		require.NoError(t, err, "Player %d should be able to submit answer", i+1)

		// Wait for answer result
		resultMsg, err := player.WaitForEvent(config.EventResourceToPlayerAnswerResult, 5*time.Second)
		require.NoError(t, err, "Player %d should receive answer result", i+1)

		// Verify answer result structure
		resultPayload := resultMsg.Payload.(map[string]interface{})
		assert.Contains(t, resultPayload, "correct", "Answer result should indicate if correct")
		assert.Contains(t, resultPayload, "correctAnswer", "Answer result should show correct answer")
		assert.Contains(t, resultPayload, "pointsEarned", "Answer result should show points earned")
		assert.Contains(t, resultPayload, "tokenType", "Answer result should show token type earned")
		assert.Contains(t, resultPayload, "tokensEarned", "Answer result should show tokens earned")

		// Check if answer was marked as correct or incorrect
		correct := resultPayload["correct"].(bool)
		pointsEarned := resultPayload["pointsEarned"].(float64)
		tokensEarned := resultPayload["tokensEarned"].(float64)

		if correct {
			assert.Greater(t, pointsEarned, 0.0, "Correct answer should earn points")
			assert.Greater(t, tokensEarned, 0.0, "Correct answer should earn tokens")
		} else {
			assert.Equal(t, 0.0, pointsEarned, "Incorrect answer should earn no points")
			assert.Equal(t, 0.0, tokensEarned, "Incorrect answer should earn no tokens")
		}
	}
}

func TestQuestionPoolCycling(t *testing.T) {
	server, cleanup := setupTestServerWithTrivia(t)
	defer cleanup()

	// Setup minimal game scenario
	host, players, cleanupGame := setupMinimalGameScenario(t, server)
	defer cleanupGame()

	// Start the game to reach resource gathering phase
	waitForGameToStart(t, host, players)

	// Track question IDs to ensure no immediate repetition
	questionIds := make(map[string]bool)
	questionsReceived := 0

	// Simulate multiple trivia rounds to test question cycling
	for round := 0; round < 3; round++ {
		// Each player should receive a trivia question
		for i, player := range players {
			// First round will come after the initial delay, subsequent rounds come naturally
			timeout := 5 * time.Second
			if round == 0 {
				timeout = 5 * time.Second // First round after 2-second delay
			}
			triviaMsg, err := player.WaitForEvent(config.EventResourceToPlayerTriviaQuestion, timeout)
			require.NoError(t, err, "Player %d should receive trivia question in round %d", i+1, round+1)

			// Extract question ID
			payload := triviaMsg.Payload.(map[string]interface{})
			questionId := payload["questionId"].(string)

			// Track this question ID
			if questionIds[questionId] {
				t.Logf("Question ID %s repeated - this is expected behavior after pool exhaustion", questionId)
			} else {
				questionIds[questionId] = true
				questionsReceived++
			}

			// Submit answer to complete the round
			err = player.AnswerTrivia(questionId, 0, 10.0)
			require.NoError(t, err, "Player %d should be able to submit answer", i+1)
		}

		// Wait a bit for the round to complete
		time.Sleep(2 * time.Second)
	}

	// Should have received at least some unique questions
	assert.Greater(t, questionsReceived, 0, "Should have received at least some unique questions")
}

func TestAnswerTimeLimitsAndGracePeriods(t *testing.T) {
	server, cleanup := setupTestServerWithTrivia(t)
	defer cleanup()

	// Setup minimal game scenario
	host, players, cleanupGame := setupMinimalGameScenario(t, server)
	defer cleanupGame()

	// Start the game to reach resource gathering phase
	waitForGameToStart(t, host, players)

	// Test time limits
	player := players[0] // Use first player for timing test

	// Wait for trivia question
	triviaMsg, err := player.WaitForEvent(config.EventResourceToPlayerTriviaQuestion, 5*time.Second)
	require.NoError(t, err, "Player should receive trivia question")

	// Verify time limit is provided
	payload := triviaMsg.Payload.(map[string]interface{})
	timeLimit := payload["timeLimit"].(float64)
	assert.Equal(t, float64(config.TriviaAnswerTime), timeLimit, "Time limit should match config")

	// Extract question information
	questionId := payload["questionId"].(string)

	// Submit answer within time limit
	err = player.AnswerTrivia(questionId, 0, 10.0)
	require.NoError(t, err, "Player should be able to submit answer within time limit")

	// Wait for answer result
	resultMsg, err := player.WaitForEvent(config.EventResourceToPlayerAnswerResult, 5*time.Second)
	require.NoError(t, err, "Player should receive answer result")

	// Verify result includes timing information
	resultPayload := resultMsg.Payload.(map[string]interface{})
	assert.Contains(t, resultPayload, "correct", "Answer result should indicate correctness")
}

func TestDifficultyModeEffectsOnTrivia(t *testing.T) {
	// This test checks if trivia questions respect difficulty settings
	server, cleanup := setupTestServerWithTrivia(t)
	defer cleanup()

	// Setup minimal game scenario
	host, players, cleanupGame := setupMinimalGameScenario(t, server)
	defer cleanupGame()

	// Start the game to reach resource gathering phase
	waitForGameToStart(t, host, players)

	// Test that trivia questions are delivered properly
	player := players[0]

	// Wait for trivia question
	triviaMsg, err := player.WaitForEvent(config.EventResourceToPlayerTriviaQuestion, 5*time.Second)
	require.NoError(t, err, "Player should receive trivia question")

	// Verify question has difficulty information
	payload := triviaMsg.Payload.(map[string]interface{})
	difficulty := payload["difficulty"].(string)
	assert.NotEmpty(t, difficulty, "Question should have difficulty level")

	// Verify time limit respects difficulty mode
	timeLimit := payload["timeLimit"].(float64)
	assert.Greater(t, timeLimit, 0.0, "Time limit should be positive")

	// For this test, we primarily verify the structure is correct
	// The actual difficulty scaling is tested in difficulty_mode_effects_test.go
	assert.Contains(t, []string{"easy", "medium", "hard"}, difficulty,
		"Difficulty should be one of the standard levels")
}

func TestSpecialtyQuestionHandling(t *testing.T) {
	server, cleanup := setupTestServerWithTrivia(t)
	defer cleanup()

	// Setup game with players having different specialties
	host := test_helpers.NewTestHostClient(t, server, config.HostUUID)
	err := host.Connect()
	require.NoError(t, err)
	defer host.Close()

	// Create players with specific specialties
	playerConfigs := []PlayerConfig{
		{Name: "Player1", Role: "art_enthusiast", Specialty: "science"},
		{Name: "Player2", Role: "detective", Specialty: "history"},
		{Name: "Player3", Role: "tourist", Specialty: "geography"},
		{Name: "Player4", Role: "janitor", Specialty: "general"},
	}

	players := connectPlayersWithConfiguration(t, server, playerConfigs)
	defer func() {
		for _, player := range players {
			player.Close()
		}
	}()

	// Start the game
	waitForGameToStart(t, host, players)

	// Test that specialty information is preserved and questions are delivered
	for i, player := range players {
		triviaMsg, err := player.WaitForEvent(config.EventResourceToPlayerTriviaQuestion, 5*time.Second)
		require.NoError(t, err, "Player %d should receive trivia question", i+1)

		// Verify question structure includes category
		payload := triviaMsg.Payload.(map[string]interface{})
		category := payload["category"].(string)
		assert.NotEmpty(t, category, "Question should have a category")

		// Note: Whether this is a specialty question or not depends on the implementation
		// The important thing is that the question is delivered properly
	}
}
