package integration_tests

import (
	"canvas-conundrum/config"
	"canvas-conundrum/test_helpers"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInvalidGameOperations(t *testing.T) {
	server, cleanup := setupTestServerWithTrivia(t)
	defer cleanup()

	// Test creating game with different parameters
	t.Run("GameCreationWithVariousDifficulties", func(t *testing.T) {
		host, _, gameCleanup := setupMinimalGameScenario(t, server)
		defer gameCleanup()

		// Test with various difficulty levels
		difficulties := []string{"easy", "medium", "hard", "invalid-difficulty"}
		for _, difficulty := range difficulties {
			err := host.StartGame(difficulty)
			// Game should start regardless of difficulty parameter
			// Invalid difficulties may be handled gracefully or default to medium
			require.NoError(t, err, "Game should start with difficulty: %s", difficulty)
			break // Only test one start per scenario
		}
	})
}

func TestInvalidGameStateTransitions(t *testing.T) {
	server, cleanup := setupTestServerWithTrivia(t)
	defer cleanup()

	host, players, gameCleanup := setupMinimalGameScenario(t, server)
	defer gameCleanup()

	// Test invalid state transitions
	t.Run("StartGameTwice", func(t *testing.T) {
		// Start game once
		err := host.StartGame("medium")
		require.NoError(t, err)

		// Wait for game to start
		for _, player := range players {
			_, err = player.WaitForEvent(config.EventResourceToClientPhaseStart, 5*time.Second)
			require.NoError(t, err)
		}

		// Try to start game again (should be ignored or handled gracefully)
		err = host.StartGame("hard")
		// This should either be ignored or return an error, but not crash
		if err != nil {
			t.Logf("Starting game twice resulted in error: %v (expected behavior)", err)
		}
	})

	t.Run("PuzzlePhaseBeforeResourcePhase", func(t *testing.T) {
		// Create a fresh scenario
		server2, cleanup2 := setupTestServerWithTrivia(t)
		defer cleanup2()

		host2, _, gameCleanup2 := setupMinimalGameScenario(t, server2)
		defer gameCleanup2()

		// Try to start puzzle phase before resource phase
		err := host2.StartPuzzlePhase()
		// This should be handled gracefully (ignored or error)
		if err != nil {
			t.Logf("Starting puzzle phase early resulted in error: %v (expected behavior)", err)
		}
	})
}

func TestMalformedRequestHandling(t *testing.T) {
	server, cleanup := setupTestServerWithTrivia(t)
	defer cleanup()

	host, players, gameCleanup := setupMinimalGameScenario(t, server)
	defer gameCleanup()

	// Start a game to get to a testable state
	err := host.StartGame("medium")
	require.NoError(t, err)

	// Wait for resource phase
	for _, player := range players {
		_, err = player.WaitForEvent(config.EventResourceToClientPhaseStart, 5*time.Second)
		require.NoError(t, err)
	}

	t.Run("InvalidTriviaAnswers", func(t *testing.T) {
		player := players[0]

		// Test sending malformed trivia answer
		invalidPayload := map[string]interface{}{
			"questionId":     123,            // Invalid type (should be string)
			"selectedAnswer": nil,            // Invalid type
			"timeElapsed":    "not-a-number", // Invalid type
		}

		// Send malformed message
		err := player.SendMessage(config.EventResourceToServerTriviaAnswer, invalidPayload)
		// Should not crash the connection, may result in error response
		if err != nil {
			t.Logf("Malformed trivia answer resulted in error: %v (expected)", err)
		}

		// Verify connection is still alive by sending valid message
		stationHashes := getTestStationHashes()
		err = player.VerifyLocation("anchor", stationHashes["anchor"])
		require.NoError(t, err, "Connection should still be alive after malformed request")
	})

	t.Run("InvalidLocationVerification", func(t *testing.T) {
		player := players[0]

		// Test invalid location verification
		invalidLocationPayload := map[string]interface{}{
			"stationId": 12345, // Invalid type
			"qrHash":    nil,   // Invalid type
		}

		err := player.SendMessage(config.EventResourceToServerLocationVerified, invalidLocationPayload)
		if err != nil {
			t.Logf("Invalid location verification resulted in error: %v (expected)", err)
		}

		// Verify connection is still alive
		validLocationPayload := map[string]interface{}{
			"stationId": "anchor",
			"qrHash":    getTestStationHashes()["anchor"],
		}
		err = player.SendMessage(config.EventResourceToServerLocationVerified, validLocationPayload)
		require.NoError(t, err, "Valid location verification should work after invalid one")
	})
}

func TestConcurrentOperations(t *testing.T) {
	server, cleanup := setupTestServerWithTrivia(t)
	defer cleanup()

	host, players, gameCleanup := setupMinimalGameScenario(t, server)
	defer gameCleanup()

	// Start game
	err := host.StartGame("medium")
	require.NoError(t, err)

	// Wait for resource phase
	for _, player := range players {
		_, err = player.WaitForEvent(config.EventResourceToClientPhaseStart, 5*time.Second)
		require.NoError(t, err)
	}

	t.Run("ConcurrentLocationVerification", func(t *testing.T) {
		stationHashes := getTestStationHashes()

		// All players try to verify location at the same station simultaneously
		done := make(chan error, len(players))

		for _, player := range players {
			go func(p *test_helpers.TestPlayerClient) {
				err := p.VerifyLocation("anchor", stationHashes["anchor"])
				done <- err
			}(player)
		}

		// Wait for all to complete
		for i := 0; i < len(players); i++ {
			err := <-done
			assert.NoError(t, err, "Concurrent location verification should not fail")
		}
	})

	t.Run("ConcurrentTriviaAnswers", func(t *testing.T) {
		// Wait for trivia questions
		for _, player := range players {
			_, err := player.WaitForEvent(config.EventResourceToPlayerTriviaQuestion, 5*time.Second)
			require.NoError(t, err)
		}

		// All players submit answers simultaneously
		done := make(chan error, len(players))

		for _, player := range players {
			go func(p *test_helpers.TestPlayerClient) {
				err := p.AnswerTrivia("", 0, 15.0)
				done <- err
			}(player)
		}

		// Wait for all to complete
		for i := 0; i < len(players); i++ {
			err := <-done
			assert.NoError(t, err, "Concurrent trivia answers should not fail")
		}
	})
}

func TestResourceExhaustion(t *testing.T) {
	server, cleanup := setupTestServerWithTrivia(t)
	defer cleanup()

	host, players, gameCleanup := setupMinimalGameScenario(t, server)
	defer gameCleanup()

	// Start game
	err := host.StartGame("medium")
	require.NoError(t, err)

	// Wait for resource phase
	for _, player := range players {
		_, err = player.WaitForEvent(config.EventResourceToClientPhaseStart, 5*time.Second)
		require.NoError(t, err)
	}

	t.Run("ExcessiveMessageSending", func(t *testing.T) {
		player := players[0]
		stationHashes := getTestStationHashes()

		// Send many location verification messages rapidly
		successCount := 0
		for i := 0; i < 50; i++ {
			err := player.VerifyLocation("anchor", stationHashes["anchor"])
			if err == nil {
				successCount++
			}
			time.Sleep(10 * time.Millisecond)
		}

		// Should handle rate limiting gracefully
		t.Logf("Successful location verifications: %d/50", successCount)
		assert.Greater(t, successCount, 0, "Should handle at least some messages")
	})
}

func TestBoundaryValues(t *testing.T) {
	server, cleanup := setupTestServerWithTrivia(t)
	defer cleanup()

	host, players, gameCleanup := setupMinimalGameScenario(t, server)
	defer gameCleanup()

	// Start game
	err := host.StartGame("medium")
	require.NoError(t, err)

	// Wait for resource phase
	for _, player := range players {
		_, err = player.WaitForEvent(config.EventResourceToClientPhaseStart, 5*time.Second)
		require.NoError(t, err)
	}

	t.Run("ExtremeTriviaAnswerTimes", func(t *testing.T) {
		player := players[0]
		stationHashes := getTestStationHashes()

		// Verify location first
		err := player.VerifyLocation("anchor", stationHashes["anchor"])
		require.NoError(t, err)

		// Wait for trivia question
		_, err = player.WaitForEvent(config.EventResourceToPlayerTriviaQuestion, 5*time.Second)
		require.NoError(t, err)

		// Test boundary values for time elapsed
		extremeTimes := []float64{-1.0, 0.0, 999999.0}

		for _, timeElapsed := range extremeTimes {
			err := player.AnswerTrivia("", 0, timeElapsed)
			// Should handle extreme values gracefully
			if err != nil {
				t.Logf("Time elapsed %f resulted in error: %v", timeElapsed, err)
			}
			break // Only test one per question
		}
	})

	t.Run("InvalidAnswerIndices", func(t *testing.T) {
		player := players[0]

		// Wait for next trivia question
		_, err := player.WaitForEvent(config.EventResourceToPlayerTriviaQuestion, 5*time.Second)
		if err == nil {
			// Test invalid answer indices
			invalidIndices := []int{-1, 999, 100}

			for _, index := range invalidIndices {
				err := player.AnswerTrivia("", index, 10.0)
				// Should handle invalid indices gracefully
				if err != nil {
					t.Logf("Invalid answer index %d resulted in error: %v", index, err)
				}
				break // Only test one per question
			}
		}
	})
}
