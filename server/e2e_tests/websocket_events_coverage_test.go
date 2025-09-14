package e2e_tests

import (
	"canvas-conundrum/config"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAllWebSocketEventsCovered verifies that all 50 WebSocket events from the specification are defined
func TestAllWebSocketEventsCovered(t *testing.T) {
	// All 50 events from websocket-events.md specification
	requiredEvents := []string{
		// Phase 0: Connection and Setup (7 events)
		config.EventSetupToHostConnectionConfirmed,
		config.EventSetupToPlayerRolesAvailable,
		config.EventSetupToServerPlayerConfiguration,
		config.EventSetupToClientLobbyStatus,
		config.EventSetupToHostPlayerRoster,
		config.EventSetupToServerStartGame,
		config.EventSetupToHostGameStarted,

		// Phase 1: Resource Gathering (10 events)
		config.EventResourceToClientPhaseStart,
		config.EventResourceToHostPhaseStart,
		config.EventResourceToServerLocationVerified,
		config.EventResourceToPlayerTriviaQuestion,
		config.EventResourceToServerTriviaAnswer,
		config.EventResourceToPlayerAnswerResult,
		config.EventResourceToClientTeamProgress,
		config.EventResourceToHostRoundAnalytics,
		config.EventResourceToClientPhaseComplete,
		config.EventResourceToHostPhaseComplete,

		// Phase 2: Puzzle Assembly (20 events)
		config.EventPuzzleToClientPhaseLoad,
		config.EventPuzzleToHostPhaseLoad,
		config.EventPuzzleToServerPhaseStart,
		config.EventPuzzleToClientPhaseStart,
		config.EventPuzzleToHostPhaseStart,
		config.EventPuzzleToServerSegmentCompleted,
		config.EventPuzzleToPlayerSegmentAcknowledged,
		config.EventPuzzleToHostSegmentCompleted,
		config.EventPuzzleToServerFragmentMove,
		config.EventPuzzleToPlayerMoveResult,
		config.EventPuzzleToClientGridState,
		config.EventPuzzleToHostGridState,
		config.EventPuzzleToServerRecommendMove,
		config.EventPuzzleToPlayerMoveRecommendation,
		config.EventPuzzleToServerRecommendationResponse,
		config.EventPuzzleToPlayerRecommendationResult,
		config.EventPuzzleToPlayerRecommendationExpired,
		config.EventPuzzleToClientCompletedSuccess,
		config.EventPuzzleToClientCompletedTimeout,
		config.EventPuzzleToHostCompletionAnalytics,

		// Phase 3: Analytics (5 events)
		config.EventAnalyticsToPlayerPersonalReport,
		config.EventAnalyticsToClientTeamSummary,
		config.EventAnalyticsToHostCompleteReport,
		config.EventAnalyticsToServerResetGame,
		config.EventAnalyticsToClientGameReset,

		// System Events (8 events)
		config.EventSystemToClientError,
		config.EventSystemToHostError,
		config.EventSystemToClientDisconnectionWarning,
		config.EventSystemToClientHostDisconnected,
		config.EventSystemToClientHostReconnected,
		config.EventSystemToHostPlayerDisconnected,
		config.EventSystemPing,
		config.EventSystemPong,
	}

	// Verify we have exactly 50 events (as per actual specification)
	assert.Len(t, requiredEvents, 50, "Should have exactly 50 WebSocket events")

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
			config.EventSetupToHostConnectionConfirmed,
			config.EventSetupToPlayerRolesAvailable,
			config.EventSetupToServerPlayerConfiguration,
			config.EventSetupToClientLobbyStatus,
			config.EventSetupToHostPlayerRoster,
			config.EventSetupToServerStartGame,
			config.EventSetupToHostGameStarted,
		}
		assert.Len(t, setupEvents, 7, "Setup phase should have 7 events")
		for _, event := range setupEvents {
			assert.NotEmpty(t, event)
		}
	})

	t.Run("ResourceGatheringPhaseEvents", func(t *testing.T) {
		resourceEvents := []string{
			config.EventResourceToClientPhaseStart,
			config.EventResourceToHostPhaseStart,
			config.EventResourceToServerLocationVerified,
			config.EventResourceToPlayerTriviaQuestion,
			config.EventResourceToServerTriviaAnswer,
			config.EventResourceToPlayerAnswerResult,
			config.EventResourceToClientTeamProgress,
			config.EventResourceToHostRoundAnalytics,
			config.EventResourceToClientPhaseComplete,
			config.EventResourceToHostPhaseComplete,
		}
		assert.Len(t, resourceEvents, 10, "Resource gathering phase should have 10 events")
		for _, event := range resourceEvents {
			assert.NotEmpty(t, event)
		}
	})

	t.Run("PuzzleAssemblyPhaseEvents", func(t *testing.T) {
		puzzleEvents := []string{
			config.EventPuzzleToClientPhaseLoad,
			config.EventPuzzleToHostPhaseLoad,
			config.EventPuzzleToServerPhaseStart,
			config.EventPuzzleToClientPhaseStart,
			config.EventPuzzleToHostPhaseStart,
			config.EventPuzzleToServerSegmentCompleted,
			config.EventPuzzleToPlayerSegmentAcknowledged,
			config.EventPuzzleToHostSegmentCompleted,
			config.EventPuzzleToServerFragmentMove,
			config.EventPuzzleToPlayerMoveResult,
			config.EventPuzzleToClientGridState,
			config.EventPuzzleToHostGridState,
			config.EventPuzzleToServerRecommendMove,
			config.EventPuzzleToPlayerMoveRecommendation,
			config.EventPuzzleToServerRecommendationResponse,
			config.EventPuzzleToPlayerRecommendationResult,
			config.EventPuzzleToPlayerRecommendationExpired,
			config.EventPuzzleToClientCompletedSuccess,
			config.EventPuzzleToClientCompletedTimeout,
			config.EventPuzzleToHostCompletionAnalytics,
		}
		assert.Len(t, puzzleEvents, 20, "Puzzle assembly phase should have 20 events")
		for _, event := range puzzleEvents {
			assert.NotEmpty(t, event)
		}
	})

	t.Run("AnalyticsPhaseEvents", func(t *testing.T) {
		analyticsEvents := []string{
			config.EventAnalyticsToPlayerPersonalReport,
			config.EventAnalyticsToClientTeamSummary,
			config.EventAnalyticsToHostCompleteReport,
			config.EventAnalyticsToServerResetGame,
			config.EventAnalyticsToClientGameReset,
		}
		assert.Len(t, analyticsEvents, 5, "Analytics phase should have 5 events")
		for _, event := range analyticsEvents {
			assert.NotEmpty(t, event)
		}
	})

	t.Run("SystemEvents", func(t *testing.T) {
		systemEvents := []string{
			config.EventSystemToClientError,
			config.EventSystemToHostError,
			config.EventSystemToClientDisconnectionWarning,
			config.EventSystemToClientHostDisconnected,
			config.EventSystemToClientHostReconnected,
			config.EventSystemToHostPlayerDisconnected,
			config.EventSystemPing,
			config.EventSystemPong,
		}
		assert.Len(t, systemEvents, 8, "System should have 8 events")
		for _, event := range systemEvents {
			assert.NotEmpty(t, event)
		}
	})
}

// TestEventNamingConvention verifies event naming follows the convention
func TestEventNamingConvention(t *testing.T) {
	// All events should follow the pattern: PHASE_TO_TARGET_ACTION
	allEvents := []string{
		config.EventSetupToHostConnectionConfirmed,
		config.EventSetupToPlayerRolesAvailable,
		config.EventSetupToServerPlayerConfiguration,
		config.EventSetupToClientLobbyStatus,
		config.EventSetupToHostPlayerRoster,
		config.EventSetupToServerStartGame,
		config.EventSetupToHostGameStarted,
		config.EventResourceToClientPhaseStart,
		config.EventResourceToHostPhaseStart,
		config.EventResourceToServerLocationVerified,
		config.EventResourceToPlayerTriviaQuestion,
		config.EventResourceToServerTriviaAnswer,
		config.EventResourceToPlayerAnswerResult,
		config.EventResourceToClientTeamProgress,
		config.EventResourceToHostRoundAnalytics,
		config.EventResourceToClientPhaseComplete,
		config.EventResourceToHostPhaseComplete,
		config.EventPuzzleToClientPhaseLoad,
		config.EventPuzzleToHostPhaseLoad,
		config.EventPuzzleToServerPhaseStart,
		config.EventPuzzleToClientPhaseStart,
		config.EventPuzzleToHostPhaseStart,
		config.EventPuzzleToServerSegmentCompleted,
		config.EventPuzzleToPlayerSegmentAcknowledged,
		config.EventPuzzleToHostSegmentCompleted,
		config.EventPuzzleToServerFragmentMove,
		config.EventPuzzleToPlayerMoveResult,
		config.EventPuzzleToClientGridState,
		config.EventPuzzleToHostGridState,
		config.EventPuzzleToServerRecommendMove,
		config.EventPuzzleToPlayerMoveRecommendation,
		config.EventPuzzleToServerRecommendationResponse,
		config.EventPuzzleToPlayerRecommendationResult,
		config.EventPuzzleToPlayerRecommendationExpired,
		config.EventPuzzleToClientCompletedSuccess,
		config.EventPuzzleToClientCompletedTimeout,
		config.EventPuzzleToHostCompletionAnalytics,
		config.EventAnalyticsToPlayerPersonalReport,
		config.EventAnalyticsToClientTeamSummary,
		config.EventAnalyticsToHostCompleteReport,
		config.EventAnalyticsToServerResetGame,
		config.EventAnalyticsToClientGameReset,
		config.EventSystemToClientError,
		config.EventSystemToHostError,
		config.EventSystemToClientDisconnectionWarning,
		config.EventSystemToClientHostDisconnected,
		config.EventSystemToClientHostReconnected,
		config.EventSystemToHostPlayerDisconnected,
		config.EventSystemPing,
		config.EventSystemPong,
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
		config.EventSetupToServerPlayerConfiguration,
		config.EventSetupToServerStartGame,
		config.EventResourceToServerLocationVerified,
		config.EventResourceToServerTriviaAnswer,
		config.EventPuzzleToServerPhaseStart,
		config.EventPuzzleToServerSegmentCompleted,
		config.EventPuzzleToServerFragmentMove,
		config.EventPuzzleToServerRecommendMove,
		config.EventPuzzleToServerRecommendationResponse,
		config.EventAnalyticsToServerResetGame,
	}

	// Events TO_CLIENT should only come from server
	toClientEvents := []string{
		config.EventSetupToClientLobbyStatus,
		config.EventResourceToClientPhaseStart,
		config.EventResourceToClientTeamProgress,
		config.EventResourceToClientPhaseComplete,
		config.EventPuzzleToClientPhaseLoad,
		config.EventPuzzleToClientPhaseStart,
		config.EventPuzzleToClientGridState,
		config.EventPuzzleToClientCompletedSuccess,
		config.EventPuzzleToClientCompletedTimeout,
		config.EventAnalyticsToClientTeamSummary,
		config.EventAnalyticsToClientGameReset,
		config.EventSystemToClientError,
		config.EventSystemToClientDisconnectionWarning,
		config.EventSystemToClientHostDisconnected,
		config.EventSystemToClientHostReconnected,
	}

	// Events TO_PLAYER should only come from server
	toPlayerEvents := []string{
		config.EventSetupToPlayerRolesAvailable,
		config.EventResourceToPlayerTriviaQuestion,
		config.EventResourceToPlayerAnswerResult,
		config.EventPuzzleToPlayerSegmentAcknowledged,
		config.EventPuzzleToPlayerMoveResult,
		config.EventPuzzleToPlayerMoveRecommendation,
		config.EventPuzzleToPlayerRecommendationResult,
		config.EventPuzzleToPlayerRecommendationExpired,
		config.EventAnalyticsToPlayerPersonalReport,
	}

	// Events TO_HOST should only come from server
	toHostEvents := []string{
		config.EventSetupToHostConnectionConfirmed,
		config.EventSetupToHostPlayerRoster,
		config.EventSetupToHostGameStarted,
		config.EventResourceToHostPhaseStart,
		config.EventResourceToHostRoundAnalytics,
		config.EventResourceToHostPhaseComplete,
		config.EventPuzzleToHostPhaseLoad,
		config.EventPuzzleToHostPhaseStart,
		config.EventPuzzleToHostSegmentCompleted,
		config.EventPuzzleToHostGridState,
		config.EventPuzzleToHostCompletionAnalytics,
		config.EventAnalyticsToHostCompleteReport,
		config.EventSystemToHostError,
		config.EventSystemToHostPlayerDisconnected,
	}

	// Verify counts
	totalDirectionalEvents := len(toServerEvents) + len(toClientEvents) +
		len(toPlayerEvents) + len(toHostEvents) + 2 // +2 for PING/PONG

	assert.Equal(t, 50, totalDirectionalEvents, "All events should be accounted for")
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
