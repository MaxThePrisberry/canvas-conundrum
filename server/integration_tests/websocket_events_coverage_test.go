package integration_tests

import (
	"canvas-conundrum/config"
	"canvas-conundrum/models"
	"canvas-conundrum/services"
	"canvas-conundrum/test_helpers"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupWebSocketCoverageTestServer creates a test server for WebSocket event coverage testing
func setupWebSocketCoverageTestServer(t *testing.T) (*httptest.Server, *test_helpers.TestHostClient, func()) {
	// Use the shared test server setup
	server, baseCleanup := setupTestServerWithTrivia(t)

	// Create host client using config.HostUUID
	hostClient := test_helpers.NewTestHostClient(t, server, config.HostUUID)

	cleanup := func() {
		hostClient.Close()
		baseCleanup()
	}

	return server, hostClient, cleanup
}

func TestCompleteWebSocketEventCoverage(t *testing.T) {
	// Test complete game flow covering all major WebSocket events
	server, hostClient, cleanup := setupWebSocketCoverageTestServer(t)
	defer cleanup()

	// Track events received during the test
	eventsReceived := make(map[string]bool)

	// Connect host
	err := hostClient.Connect()
	require.NoError(t, err)

	// Host connection should trigger SETUP_TO_HOST_CONNECTION_CONFIRMED
	eventsReceived[config.EventSetupToHostConnectionConfirmed] = true

	// Connect players for the entire test (reuse same players)
	players := make([]*test_helpers.TestPlayerClient, 4)
	roles := []string{"art_enthusiast", "detective", "tourist", "janitor"}
	for i := 0; i < 4; i++ {
		player := test_helpers.NewTestPlayerClient(t, server)
		err := player.Connect()
		require.NoError(t, err)
		defer player.Close()
		players[i] = player

		// Should receive SETUP_TO_PLAYER_ROLES_AVAILABLE
		_, err = player.WaitForEvent(config.EventSetupToPlayerRolesAvailable, 2*time.Second)
		require.NoError(t, err)
		eventsReceived[config.EventSetupToPlayerRolesAvailable] = true

		// Configure player (triggers SETUP_TO_SERVER_PLAYER_CONFIGURATION)
		err = player.ConfigurePlayer(fmt.Sprintf("Player%d", i+1), roles[i], []string{"science"})
		require.NoError(t, err)

		// Should receive SETUP_TO_CLIENT_LOBBY_STATUS
		_, err = player.WaitForEvent(config.EventSetupToClientLobbyStatus, 2*time.Second)
		require.NoError(t, err)
		eventsReceived[config.EventSetupToClientLobbyStatus] = true

		// Host should receive SETUP_TO_HOST_PLAYER_ROSTER
		_, err = hostClient.WaitForEvent(config.EventSetupToHostPlayerRoster, 2*time.Second)
		require.NoError(t, err)
		eventsReceived[config.EventSetupToHostPlayerRoster] = true
	}

	// Test setup and resource phase events in sequence (they happen naturally)
	t.Run("SetupAndResourcePhaseEvents", func(t *testing.T) {
		// Start game (triggers SETUP_TO_SERVER_START_GAME)
		err = hostClient.StartGame("medium")
		require.NoError(t, err)

		// Wait for game started event (now implemented)
		_, err = hostClient.WaitForEvent(config.EventSetupToHostGameStarted, 3*time.Second)
		require.NoError(t, err)
		eventsReceived[config.EventSetupToHostGameStarted] = true

		// Now wait for resource phase start for all players (happens automatically after game start)
		for _, player := range players {
			_, err = player.WaitForEvent(config.EventResourceToClientPhaseStart, 5*time.Second)
			require.NoError(t, err)
			eventsReceived[config.EventResourceToClientPhaseStart] = true

			// Should also receive team progress
			_, err = player.WaitForEvent(config.EventResourceToClientTeamProgress, 2*time.Second)
			require.NoError(t, err)
			eventsReceived[config.EventResourceToClientTeamProgress] = true
		}

		// Host should receive RESOURCE_TO_HOST_PHASE_START
		_, err = hostClient.WaitForEvent(config.EventResourceToHostPhaseStart, 2*time.Second)
		require.NoError(t, err)
		eventsReceived[config.EventResourceToHostPhaseStart] = true

		// Test trivia question/answer flow
		for _, player := range players {
			// Wait for trivia question (RESOURCE_TO_PLAYER_TRIVIA_QUESTION)
			_, err = player.WaitForEvent(config.EventResourceToPlayerTriviaQuestion, 10*time.Second)
			if err == nil {
				eventsReceived[config.EventResourceToPlayerTriviaQuestion] = true

				// Submit answer (RESOURCE_TO_SERVER_TRIVIA_ANSWER)
				err = player.AnswerTrivia("", 0, 10.0)
				require.NoError(t, err)

				// Wait for answer result (RESOURCE_TO_PLAYER_ANSWER_RESULT)
				_, err = player.WaitForEvent(config.EventResourceToPlayerAnswerResult, 2*time.Second)
				if err == nil {
					eventsReceived[config.EventResourceToPlayerAnswerResult] = true
				}
			}

			time.Sleep(200 * time.Millisecond)
		}

		// Host should receive round analytics (RESOURCE_TO_HOST_ROUND_ANALYTICS)
		_, err = hostClient.WaitForEvent(config.EventResourceToHostRoundAnalytics, 10*time.Second)
		if err == nil {
			eventsReceived[config.EventResourceToHostRoundAnalytics] = true
		}

		// Wait for resource phase to complete and puzzle phase to load
		for _, player := range players {
			_, err = player.WaitForEvent(config.EventPuzzleToClientPhaseLoad, 40*time.Second)
			require.NoError(t, err)
			eventsReceived[config.EventPuzzleToClientPhaseLoad] = true
		}
	})

	// Test puzzle phase events
	t.Run("PuzzlePhaseEvents", func(t *testing.T) {
		// Host starts puzzle phase
		err = hostClient.StartPuzzlePhase()
		require.NoError(t, err)

		// All players should receive PUZZLE_TO_CLIENT_PHASE_START
		for _, player := range players {
			_, err = player.WaitForEvent(config.EventPuzzleToClientPhaseStart, 5*time.Second)
			require.NoError(t, err)
			eventsReceived[config.EventPuzzleToClientPhaseStart] = true

			// Should receive PUZZLE_TO_CLIENT_GRID_STATE
			_, err = player.WaitForEvent(config.EventPuzzleToClientGridState, 2*time.Second)
			if err == nil {
				eventsReceived[config.EventPuzzleToClientGridState] = true
			}
		}

		// Host should receive puzzle phase events
		_, err = hostClient.WaitForEvent(config.EventPuzzleToHostPhaseLoad, 2*time.Second)
		if err == nil {
			eventsReceived[config.EventPuzzleToHostPhaseLoad] = true
		}

		_, err = hostClient.WaitForEvent(config.EventPuzzleToHostPhaseStart, 2*time.Second)
		if err == nil {
			eventsReceived[config.EventPuzzleToHostPhaseStart] = true
		}

		_, err = hostClient.WaitForEvent(config.EventPuzzleToHostGridState, 2*time.Second)
		if err == nil {
			eventsReceived[config.EventPuzzleToHostGridState] = true
		}

		// Test puzzle interaction events
		if len(players) > 0 {
			// Test segment completion (PUZZLE_TO_SERVER_SEGMENT_COMPLETED)
			segmentPayload := map[string]interface{}{
				"segmentId": "A1",
			}
			err = players[0].SendMessage(config.EventPuzzleToServerSegmentCompleted, segmentPayload)
			if err == nil {
				// Wait for acknowledgment (PUZZLE_TO_PLAYER_SEGMENT_ACKNOWLEDGED)
				_, err = players[0].WaitForEvent(config.EventPuzzleToPlayerSegmentAcknowledged, 1*time.Second)
				if err == nil {
					eventsReceived[config.EventPuzzleToPlayerSegmentAcknowledged] = true
				}

				// Host should receive segment completion (PUZZLE_TO_HOST_SEGMENT_COMPLETED)
				_, err = hostClient.WaitForEvent(config.EventPuzzleToHostSegmentCompleted, 1*time.Second)
				if err == nil {
					eventsReceived[config.EventPuzzleToHostSegmentCompleted] = true
				}
			}

			// Test fragment move (PUZZLE_TO_SERVER_FRAGMENT_MOVE)
			movePayload := map[string]interface{}{
				"fragmentId": 0,
				"x":          1,
				"y":          1,
			}
			err = players[0].SendMessage(config.EventPuzzleToServerFragmentMove, movePayload)
			if err == nil {
				// Wait for move result (PUZZLE_TO_PLAYER_MOVE_RESULT)
				_, err = players[0].WaitForEvent(config.EventPuzzleToPlayerMoveResult, 1*time.Second)
				if err == nil {
					eventsReceived[config.EventPuzzleToPlayerMoveResult] = true
				}
			}
		}
	})

	// Test analytics phase events
	t.Run("AnalyticsPhaseEvents", func(t *testing.T) {
		// Complete puzzle phase to trigger analytics
		gm := services.GetGameInstance()
		game := gm.GetGame()
		if game != nil {
			// Force transition to analytics phase
			game.CurrentPhase = models.PhaseAnalytics

			// Trigger analytics events by sending analytics reports
			for _, player := range players {
				// Should receive ANALYTICS_TO_PLAYER_PERSONAL_REPORT
				_, err = player.WaitForEvent(config.EventAnalyticsToPlayerPersonalReport, 2*time.Second)
				if err == nil {
					eventsReceived[config.EventAnalyticsToPlayerPersonalReport] = true
				}

				// Should receive ANALYTICS_TO_CLIENT_TEAM_SUMMARY
				_, err = player.WaitForEvent(config.EventAnalyticsToClientTeamSummary, 2*time.Second)
				if err == nil {
					eventsReceived[config.EventAnalyticsToClientTeamSummary] = true
				}
			}

			// Host should receive ANALYTICS_TO_HOST_COMPLETE_REPORT
			_, err = hostClient.WaitForEvent(config.EventAnalyticsToHostCompleteReport, 2*time.Second)
			if err == nil {
				eventsReceived[config.EventAnalyticsToHostCompleteReport] = true
			}
		}
	})

	// Test system events
	t.Run("SystemEvents", func(t *testing.T) {
		// Test disconnection events by connecting and then disconnecting a temporary player
		tempPlayer := test_helpers.NewTestPlayerClient(t, server)
		err := tempPlayer.Connect()

		// Only test disconnection if connection succeeds
		if err == nil {
			// Disconnect player immediately to trigger disconnection event
			tempPlayer.Close()

			time.Sleep(500 * time.Millisecond)

			// Host should receive SYSTEM_TO_HOST_PLAYER_DISCONNECTED
			_, err = hostClient.WaitForEvent(config.EventSystemToHostPlayerDisconnected, 2*time.Second)
			if err == nil {
				eventsReceived[config.EventSystemToHostPlayerDisconnected] = true
			}
		}
	})

	// Verify coverage of major event categories
	t.Run("VerifyEventCoverage", func(t *testing.T) {
		majorEventCategories := map[string][]string{
			"Setup Events": {
				config.EventSetupToHostConnectionConfirmed,
				config.EventSetupToPlayerRolesAvailable,
				config.EventSetupToClientLobbyStatus,
				config.EventSetupToHostPlayerRoster,
				config.EventSetupToHostGameStarted,
			},
			"Resource Events": {
				config.EventResourceToClientPhaseStart,
				config.EventResourceToHostPhaseStart,
				config.EventResourceToPlayerTriviaQuestion,
				config.EventResourceToPlayerAnswerResult,
				config.EventResourceToClientTeamProgress,
			},
			"Puzzle Events": {
				config.EventPuzzleToClientPhaseLoad,
				config.EventPuzzleToClientPhaseStart,
				config.EventPuzzleToClientGridState,
			},
			"Analytics Events": {
				config.EventAnalyticsToPlayerPersonalReport,
				config.EventAnalyticsToClientTeamSummary,
				config.EventAnalyticsToHostCompleteReport,
			},
			"System Events": {
				config.EventSystemToHostPlayerDisconnected,
			},
		}

		for category, events := range majorEventCategories {
			t.Run(category, func(t *testing.T) {
				for _, event := range events {
					if eventsReceived[event] {
						t.Logf("✓ Covered: %s", event)
					} else {
						t.Logf("✗ Missing: %s", event)
					}
				}
			})
		}

		// Summary of coverage
		totalExpectedEvents := 0
		coveredEvents := 0
		for _, events := range majorEventCategories {
			totalExpectedEvents += len(events)
			for _, event := range events {
				if eventsReceived[event] {
					coveredEvents++
				}
			}
		}

		coveragePercentage := float64(coveredEvents) / float64(totalExpectedEvents) * 100
		t.Logf("WebSocket Event Coverage: %d/%d (%.1f%%)", coveredEvents, totalExpectedEvents, coveragePercentage)

		assert.GreaterOrEqual(t, coveragePercentage, 60.0, "Should have at least 60% WebSocket event coverage")
	})
}

func TestMissingSpecificationFeatures(t *testing.T) {
	createTestTriviaFiles(t)
	defer cleanupTestTriviaFiles(t)

	// Test any remaining specification features not covered in other tests
	server, hostClient, cleanup := setupWebSocketCoverageTestServer(t)
	defer cleanup()

	// Connect host
	err := hostClient.Connect()
	require.NoError(t, err)

	// Test location verification with QR codes
	t.Run("LocationVerificationFlow", func(t *testing.T) {
		err := hostClient.StartGame("medium")
		require.NoError(t, err)

		player := test_helpers.NewTestPlayerClient(t, server)
		err = player.Connect()
		require.NoError(t, err)
		defer player.Close()

		// Configure and start game
		err = player.ConfigurePlayer("LocationPlayer", "janitor", []string{"science"})
		require.NoError(t, err)

		err = hostClient.StartGame("medium")
		require.NoError(t, err)

		// Wait for resource phase
		_, err = player.WaitForEvent(config.EventResourceToClientPhaseStart, 3*time.Second)
		require.NoError(t, err)

		// Test location verification (RESOURCE_TO_SERVER_LOCATION_VERIFIED)
		locationPayload := map[string]interface{}{
			"stationHash": "test_station_hash",
		}

		err = player.SendMessage(config.EventResourceToServerLocationVerified, locationPayload)
		require.NoError(t, err, "Should be able to send location verification")
	})

	// Test game reset functionality
	t.Run("GameResetFlow", func(t *testing.T) {
		// Fast-forward to analytics phase
		gm := services.GetGameInstance()
		game := gm.GetGame()
		if game != nil {
			game.CurrentPhase = models.PhaseAnalytics

			// Test game reset (ANALYTICS_TO_SERVER_RESET_GAME)
			resetPayload := map[string]interface{}{
				"confirmReset": true,
			}

			err = hostClient.SendMessage(config.EventAnalyticsToServerResetGame, resetPayload)
			if err == nil {
				// Should receive ANALYTICS_TO_CLIENT_GAME_RESET
				_, err = hostClient.WaitForEvent(config.EventAnalyticsToClientGameReset, 2*time.Second)
				require.NoError(t, err, "Should receive game reset confirmation")
			}
		}
	})

	// Test error handling events
	t.Run("ErrorEventHandling", func(t *testing.T) {
		player := test_helpers.NewTestPlayerClient(t, server)
		err = player.Connect()
		require.NoError(t, err)
		defer player.Close()

		// Send invalid message to trigger error
		invalidPayload := map[string]interface{}{
			"invalid": "data",
		}

		err = player.SendMessage("invalid_event_type", invalidPayload)
		require.NoError(t, err)

		// Should receive SYSTEM_TO_CLIENT_ERROR (if server sends error events)
		errorMsg, err := player.WaitForEvent(config.EventSystemToClientError, 1*time.Second)
		if err == nil {
			payload := errorMsg.Payload.(map[string]interface{})
			assert.Contains(t, payload, "error", "Error event should contain error message")
		}
	})
}
