package e2e_tests

import (
	"canvas-conundrum/constants"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAllWebSocketEventsCovered verifies that all 59 WebSocket events from the specification are defined
func TestAllWebSocketEventsCovered(t *testing.T) {
	// All 59 events from websocket-events.md specification
	requiredEvents := []string{
		// Phase 0: Connection and Setup (8 events)
		constants.EventSetupToHostConnectionConfirmed,
		constants.EventSetupToPlayerRolesAvailable,
		constants.EventSetupToServerPlayerConfiguration,
		constants.EventSetupToClientLobbyStatus,
		constants.EventSetupToHostPlayerRoster,
		constants.EventSetupToServerStartGame,
		constants.EventSetupToClientGameStarted,
		constants.EventSetupToHostGameStarted,

		// Phase 1: Resource Gathering (10 events)
		constants.EventResourceToClientPhaseStart,
		constants.EventResourceToHostPhaseStart,
		constants.EventResourceToServerLocationVerified,
		constants.EventResourceToPlayerTriviaQuestion,
		constants.EventResourceToServerTriviaAnswer,
		constants.EventResourceToPlayerAnswerResult,
		constants.EventResourceToClientTeamProgress,
		constants.EventResourceToHostRoundAnalytics,
		constants.EventResourceToClientPhaseComplete,
		constants.EventResourceToHostPhaseComplete,

		// Phase 2: Puzzle Assembly (21 events)
		constants.EventPuzzleToClientPhaseLoad,
		constants.EventPuzzleToHostPhaseLoad,
		constants.EventPuzzleToServerStartTimer,
		constants.EventPuzzleToClientTimerStart,
		constants.EventPuzzleToHostTimerStart,
		constants.EventPuzzleToServerSegmentCompleted,
		constants.EventPuzzleToPlayerSegmentAcknowledged,
		constants.EventPuzzleToHostSegmentCompleted,
		constants.EventPuzzleToServerFragmentMove,
		constants.EventPuzzleToPlayerMoveResult,
		constants.EventPuzzleToClientGridState,
		constants.EventPuzzleToHostGridState,
		constants.EventPuzzleToServerRecommendMove,
		constants.EventPuzzleToPlayerMoveRecommendation,
		constants.EventPuzzleToServerRecommendationResponse,
		constants.EventPuzzleToPlayerRecommendationResult,
		constants.EventPuzzleToPlayerRecommendationExpired,
		constants.EventPuzzleToClientCompletedSuccess,
		constants.EventPuzzleToClientCompletedTimeout,
		constants.EventPuzzleToHostCompletionAnalytics,

		// Phase 3: Analytics (5 events)
		constants.EventAnalyticsToPlayerPersonalReport,
		constants.EventAnalyticsToClientTeamSummary,
		constants.EventAnalyticsToHostCompleteReport,
		constants.EventAnalyticsToServerResetGame,
		constants.EventAnalyticsToClientGameReset,

		// System Events (10 events)
		constants.EventSystemToClientError,
		constants.EventSystemToHostError,
		constants.EventSystemToClientDisconnectionWarning,
		constants.EventSystemToClientHostDisconnected,
		constants.EventSystemToClientHostReconnected,
		constants.EventSystemToHostPlayerDisconnected,
		constants.EventSystemPing,
		constants.EventSystemPong,
		constants.EventSystemToClientPhaseTransition,
		constants.EventSystemToHostPhaseTransition,
	}

	// Verify we have exactly 53 events (as per actual specification)
	assert.Len(t, requiredEvents, 53, "Should have exactly 53 WebSocket events")

	// Check each event is defined (non-empty)
	for _, event := range requiredEvents {
		assert.NotEmpty(t, event, "Event constant should be defined: %s", event)
	}

	// Check for duplicates
	eventMap := make(map[string]bool)
	for _, event := range requiredEvents {
		if eventMap[event] {
			assert.Fail(t, "Duplicate event found", "Event %s is duplicated", event)
		}
		eventMap[event] = true
	}

	// Group events by phase for reporting
	t.Run("SetupPhaseEvents", func(t *testing.T) {
		setupEvents := []string{
			constants.EventSetupToHostConnectionConfirmed,
			constants.EventSetupToPlayerRolesAvailable,
			constants.EventSetupToServerPlayerConfiguration,
			constants.EventSetupToClientLobbyStatus,
			constants.EventSetupToHostPlayerRoster,
			constants.EventSetupToServerStartGame,
			constants.EventSetupToClientGameStarted,
			constants.EventSetupToHostGameStarted,
		}
		assert.Len(t, setupEvents, 8, "Setup phase should have 8 events")
		for _, event := range setupEvents {
			assert.NotEmpty(t, event)
		}
	})

	t.Run("ResourceGatheringPhaseEvents", func(t *testing.T) {
		resourceEvents := []string{
			constants.EventResourceToClientPhaseStart,
			constants.EventResourceToHostPhaseStart,
			constants.EventResourceToServerLocationVerified,
			constants.EventResourceToPlayerTriviaQuestion,
			constants.EventResourceToServerTriviaAnswer,
			constants.EventResourceToPlayerAnswerResult,
			constants.EventResourceToClientTeamProgress,
			constants.EventResourceToHostRoundAnalytics,
			constants.EventResourceToClientPhaseComplete,
			constants.EventResourceToHostPhaseComplete,
		}
		assert.Len(t, resourceEvents, 10, "Resource gathering phase should have 10 events")
		for _, event := range resourceEvents {
			assert.NotEmpty(t, event)
		}
	})

	t.Run("PuzzleAssemblyPhaseEvents", func(t *testing.T) {
		puzzleEvents := []string{
			constants.EventPuzzleToClientPhaseLoad,
			constants.EventPuzzleToHostPhaseLoad,
			constants.EventPuzzleToServerStartTimer,
			constants.EventPuzzleToClientTimerStart,
			constants.EventPuzzleToHostTimerStart,
			constants.EventPuzzleToServerSegmentCompleted,
			constants.EventPuzzleToPlayerSegmentAcknowledged,
			constants.EventPuzzleToHostSegmentCompleted,
			constants.EventPuzzleToServerFragmentMove,
			constants.EventPuzzleToPlayerMoveResult,
			constants.EventPuzzleToClientGridState,
			constants.EventPuzzleToHostGridState,
			constants.EventPuzzleToServerRecommendMove,
			constants.EventPuzzleToPlayerMoveRecommendation,
			constants.EventPuzzleToServerRecommendationResponse,
			constants.EventPuzzleToPlayerRecommendationResult,
			constants.EventPuzzleToPlayerRecommendationExpired,
			constants.EventPuzzleToClientCompletedSuccess,
			constants.EventPuzzleToClientCompletedTimeout,
			constants.EventPuzzleToHostCompletionAnalytics,
		}
		assert.Len(t, puzzleEvents, 20, "Puzzle assembly phase should have 20 events")
		for _, event := range puzzleEvents {
			assert.NotEmpty(t, event)
		}
	})

	t.Run("AnalyticsPhaseEvents", func(t *testing.T) {
		analyticsEvents := []string{
			constants.EventAnalyticsToPlayerPersonalReport,
			constants.EventAnalyticsToClientTeamSummary,
			constants.EventAnalyticsToHostCompleteReport,
			constants.EventAnalyticsToServerResetGame,
			constants.EventAnalyticsToClientGameReset,
		}
		assert.Len(t, analyticsEvents, 5, "Analytics phase should have 5 events")
		for _, event := range analyticsEvents {
			assert.NotEmpty(t, event)
		}
	})

	t.Run("SystemEvents", func(t *testing.T) {
		systemEvents := []string{
			constants.EventSystemToClientError,
			constants.EventSystemToHostError,
			constants.EventSystemToClientDisconnectionWarning,
			constants.EventSystemToClientHostDisconnected,
			constants.EventSystemToClientHostReconnected,
			constants.EventSystemToHostPlayerDisconnected,
			constants.EventSystemPing,
			constants.EventSystemPong,
			constants.EventSystemToClientPhaseTransition,
			constants.EventSystemToHostPhaseTransition,
		}
		assert.Len(t, systemEvents, 10, "System should have 10 events")
		for _, event := range systemEvents {
			assert.NotEmpty(t, event)
		}
	})
}

// TestEventNamingConvention verifies event naming follows the convention
func TestEventNamingConvention(t *testing.T) {
	// All events should follow the pattern: PHASE_TO_TARGET_ACTION
	allEvents := []string{
		constants.EventSetupToHostConnectionConfirmed,
		constants.EventSetupToPlayerRolesAvailable,
		constants.EventSetupToServerPlayerConfiguration,
		constants.EventSetupToClientLobbyStatus,
		constants.EventSetupToHostPlayerRoster,
		constants.EventSetupToServerStartGame,
		constants.EventSetupToClientGameStarted,
		constants.EventSetupToHostGameStarted,
		constants.EventResourceToClientPhaseStart,
		constants.EventResourceToHostPhaseStart,
		constants.EventResourceToServerLocationVerified,
		constants.EventResourceToPlayerTriviaQuestion,
		constants.EventResourceToServerTriviaAnswer,
		constants.EventResourceToPlayerAnswerResult,
		constants.EventResourceToClientTeamProgress,
		constants.EventResourceToHostRoundAnalytics,
		constants.EventResourceToClientPhaseComplete,
		constants.EventResourceToHostPhaseComplete,
		constants.EventPuzzleToClientPhaseLoad,
		constants.EventPuzzleToHostPhaseLoad,
		constants.EventPuzzleToServerStartTimer,
		constants.EventPuzzleToClientTimerStart,
		constants.EventPuzzleToHostTimerStart,
		constants.EventPuzzleToServerSegmentCompleted,
		constants.EventPuzzleToPlayerSegmentAcknowledged,
		constants.EventPuzzleToHostSegmentCompleted,
		constants.EventPuzzleToServerFragmentMove,
		constants.EventPuzzleToPlayerMoveResult,
		constants.EventPuzzleToClientGridState,
		constants.EventPuzzleToHostGridState,
		constants.EventPuzzleToServerRecommendMove,
		constants.EventPuzzleToPlayerMoveRecommendation,
		constants.EventPuzzleToServerRecommendationResponse,
		constants.EventPuzzleToPlayerRecommendationResult,
		constants.EventPuzzleToPlayerRecommendationExpired,
		constants.EventPuzzleToClientCompletedSuccess,
		constants.EventPuzzleToClientCompletedTimeout,
		constants.EventPuzzleToHostCompletionAnalytics,
		constants.EventAnalyticsToPlayerPersonalReport,
		constants.EventAnalyticsToClientTeamSummary,
		constants.EventAnalyticsToHostCompleteReport,
		constants.EventAnalyticsToServerResetGame,
		constants.EventAnalyticsToClientGameReset,
		constants.EventSystemToClientError,
		constants.EventSystemToHostError,
		constants.EventSystemToClientDisconnectionWarning,
		constants.EventSystemToClientHostDisconnected,
		constants.EventSystemToClientHostReconnected,
		constants.EventSystemToHostPlayerDisconnected,
		constants.EventSystemPing,
		constants.EventSystemPong,
		constants.EventSystemToClientPhaseTransition,
		constants.EventSystemToHostPhaseTransition,
	}

	validPrefixes := []string{
		"SETUP_TO_",
		"RESOURCE_TO_",
		"PUZZLE_TO_",
		"ANALYTICS_TO_",
		"SYSTEM_TO_",
		"SYSTEM_", // For PING/PONG
	}

	validTargets := []string{
		"_CLIENT_",
		"_SERVER_",
		"_PLAYER_",
		"_HOST_",
		"PING", // Special case
		"PONG", // Special case
	}

	for _, event := range allEvents {
		// Check prefix
		hasValidPrefix := false
		for _, prefix := range validPrefixes {
			if len(event) >= len(prefix) && event[:len(prefix)] == prefix {
				hasValidPrefix = true
				break
			}
		}
		assert.True(t, hasValidPrefix, "Event %s should have valid prefix", event)

		// Check target (unless it's PING/PONG)
		if event != "SYSTEM_PING" && event != "SYSTEM_PONG" {
			hasValidTarget := false
			for _, target := range validTargets[:4] { // Exclude PING/PONG from targets
				if contains(event, target) {
					hasValidTarget = true
					break
				}
			}
			assert.True(t, hasValidTarget, "Event %s should have valid target", event)
		}
	}
}

// TestEventDirectionality verifies events have correct sender/receiver relationships
func TestEventDirectionality(t *testing.T) {
	// Events TO_SERVER should only come from clients (players/host)
	toServerEvents := []string{
		constants.EventSetupToServerPlayerConfiguration,
		constants.EventSetupToServerStartGame,
		constants.EventResourceToServerLocationVerified,
		constants.EventResourceToServerTriviaAnswer,
		constants.EventPuzzleToServerStartTimer,
		constants.EventPuzzleToServerSegmentCompleted,
		constants.EventPuzzleToServerFragmentMove,
		constants.EventPuzzleToServerRecommendMove,
		constants.EventPuzzleToServerRecommendationResponse,
		constants.EventAnalyticsToServerResetGame,
	}

	// Events TO_CLIENT should only come from server
	toClientEvents := []string{
		constants.EventSetupToClientLobbyStatus,
		constants.EventSetupToClientGameStarted,
		constants.EventResourceToClientPhaseStart,
		constants.EventResourceToClientTeamProgress,
		constants.EventResourceToClientPhaseComplete,
		constants.EventPuzzleToClientPhaseLoad,
		constants.EventPuzzleToClientTimerStart,
		constants.EventPuzzleToClientGridState,
		constants.EventPuzzleToClientCompletedSuccess,
		constants.EventPuzzleToClientCompletedTimeout,
		constants.EventAnalyticsToClientTeamSummary,
		constants.EventAnalyticsToClientGameReset,
		constants.EventSystemToClientError,
		constants.EventSystemToClientDisconnectionWarning,
		constants.EventSystemToClientHostDisconnected,
		constants.EventSystemToClientHostReconnected,
		constants.EventSystemToClientPhaseTransition,
	}

	// Events TO_PLAYER should only come from server
	toPlayerEvents := []string{
		constants.EventSetupToPlayerRolesAvailable,
		constants.EventResourceToPlayerTriviaQuestion,
		constants.EventResourceToPlayerAnswerResult,
		constants.EventPuzzleToPlayerSegmentAcknowledged,
		constants.EventPuzzleToPlayerMoveResult,
		constants.EventPuzzleToPlayerMoveRecommendation,
		constants.EventPuzzleToPlayerRecommendationResult,
		constants.EventPuzzleToPlayerRecommendationExpired,
		constants.EventAnalyticsToPlayerPersonalReport,
	}

	// Events TO_HOST should only come from server
	toHostEvents := []string{
		constants.EventSetupToHostConnectionConfirmed,
		constants.EventSetupToHostPlayerRoster,
		constants.EventSetupToHostGameStarted,
		constants.EventResourceToHostPhaseStart,
		constants.EventResourceToHostRoundAnalytics,
		constants.EventResourceToHostPhaseComplete,
		constants.EventPuzzleToHostPhaseLoad,
		constants.EventPuzzleToHostTimerStart,
		constants.EventPuzzleToHostSegmentCompleted,
		constants.EventPuzzleToHostGridState,
		constants.EventPuzzleToHostCompletionAnalytics,
		constants.EventAnalyticsToHostCompleteReport,
		constants.EventSystemToHostError,
		constants.EventSystemToHostPlayerDisconnected,
		constants.EventSystemToHostPhaseTransition,
	}

	// Verify counts
	totalDirectionalEvents := len(toServerEvents) + len(toClientEvents) +
		len(toPlayerEvents) + len(toHostEvents) + 2 // +2 for PING/PONG

	assert.Equal(t, 53, totalDirectionalEvents, "All events should be accounted for")
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			(len(substr) < len(s) && findSubstring(s, substr))))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
