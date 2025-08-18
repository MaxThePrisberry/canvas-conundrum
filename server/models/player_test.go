package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewPlayer(t *testing.T) {
	playerID := "123e4567-e89b-12d3-a456-426614174000"
	player := NewPlayer(playerID, nil)

	assert.NotNil(t, player)
	assert.Equal(t, playerID, player.ID)
	assert.Empty(t, player.Name)
	assert.Equal(t, Role(""), player.Role)
	assert.Empty(t, player.Specialties)
	assert.False(t, player.IsReady)
	assert.False(t, player.IsActive)
	assert.Nil(t, player.Connection)
	assert.NotNil(t, player.Send)
	assert.NotNil(t, player.Done)
	assert.Zero(t, player.TokensEarned)
	assert.Zero(t, player.QuestionsAnswered)
	assert.Zero(t, player.CorrectAnswers)
	assert.Empty(t, player.CurrentStation)
	assert.Empty(t, player.AssignedSegment)
	assert.False(t, player.SegmentCompleted)
	assert.NotZero(t, player.JoinedAt)
}

func TestPlayerRoles(t *testing.T) {
	tests := []struct {
		role     Role
		expected bool
	}{
		{RoleArtEnthusiast, true},
		{RoleDetective, true},
		{RoleTourist, true},
		{RoleJanitor, true},
		{Role("invalid"), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			player := NewPlayer("test", nil)
			player.Role = tt.role

			// Check if role is one of the valid ones
			isValid := player.Role == RoleArtEnthusiast ||
				player.Role == RoleDetective ||
				player.Role == RoleTourist ||
				player.Role == RoleJanitor

			assert.Equal(t, tt.expected, isValid)
		})
	}
}

func TestPlayerSpecialties(t *testing.T) {
	player := NewPlayer("test", nil)

	// Can have no specialties
	assert.Empty(t, player.Specialties)

	// Can have one specialty (as per game rules)
	player.Specialties = []TriviaCategory{CategoryScience}
	assert.Len(t, player.Specialties, 1)
	assert.Equal(t, CategoryScience, player.Specialties[0])

	// Test multiple specialties (though game limits to 1)
	player.Specialties = []TriviaCategory{CategoryScience, CategoryHistory}
	assert.Len(t, player.Specialties, 2)
}

func TestPlayerTokenManagement(t *testing.T) {
	player := NewPlayer("test", nil)

	// Initial state
	assert.Equal(t, 0, player.TokensEarned)

	// Add tokens
	player.TokensEarned = 10
	assert.Equal(t, 10, player.TokensEarned)

	// Add more tokens
	player.TokensEarned += 5
	assert.Equal(t, 15, player.TokensEarned)
}

func TestPlayerTriviaStats(t *testing.T) {
	player := NewPlayer("test", nil)

	// Initial state
	assert.Equal(t, 0, player.QuestionsAnswered)
	assert.Equal(t, 0, player.CorrectAnswers)

	// Answer questions
	player.QuestionsAnswered = 10
	player.CorrectAnswers = 7

	assert.Equal(t, 10, player.QuestionsAnswered)
	assert.Equal(t, 7, player.CorrectAnswers)

	// Calculate accuracy
	accuracy := float64(player.CorrectAnswers) / float64(player.QuestionsAnswered)
	assert.InDelta(t, 0.7, accuracy, 0.01)
}

func TestPlayerStationAssignment(t *testing.T) {
	player := NewPlayer("test", nil)

	// Initial state
	assert.Empty(t, player.CurrentStation)

	// Assign to station
	player.CurrentStation = "anchor"
	assert.Equal(t, "anchor", player.CurrentStation)

	// Change station
	player.CurrentStation = "chronos"
	assert.Equal(t, "chronos", player.CurrentStation)
}

func TestPlayerPuzzleSegment(t *testing.T) {
	player := NewPlayer("test", nil)

	// Initial state
	assert.Empty(t, player.AssignedSegment)
	assert.False(t, player.SegmentCompleted)
	assert.Zero(t, player.SegmentSolveTime)

	// Assign segment
	player.AssignedSegment = "A1"
	assert.Equal(t, "A1", player.AssignedSegment)

	// Complete segment
	player.SegmentCompleted = true
	player.SegmentSolveTime = 45.5

	assert.True(t, player.SegmentCompleted)
	assert.Equal(t, 45.5, player.SegmentSolveTime)
}

func TestPlayerConnectionStatus(t *testing.T) {
	player := NewPlayer("test", nil)

	// Initial state - no connection
	assert.False(t, player.IsActive)
	assert.Nil(t, player.Connection)

	// Simulate connection
	player.IsActive = true
	assert.True(t, player.IsActive)

	// Simulate disconnection
	player.IsActive = false
	player.LastSeen = time.Now()
	assert.False(t, player.IsActive)
	assert.NotZero(t, player.LastSeen)
}

func TestNewHost(t *testing.T) {
	hostID := "987fcdeb-51a2-43d1-9f12-123456789abc"
	host := NewHost(hostID, nil)

	assert.NotNil(t, host)
	assert.Equal(t, hostID, host.ID)
	assert.Nil(t, host.Connection)
	assert.NotNil(t, host.Send)
	assert.NotNil(t, host.Done)
	assert.NotZero(t, host.ConnectedAt)
}

func TestPlayerReadyState(t *testing.T) {
	player := NewPlayer("test", nil)

	// Not ready initially
	assert.False(t, player.IsReady)

	// Configure player
	player.Name = "TestPlayer"
	player.Role = RoleDetective
	player.Specialties = []TriviaCategory{CategoryHistory}
	player.IsReady = true

	assert.True(t, player.IsReady)
	assert.NotEmpty(t, player.Name)
	assert.NotEmpty(t, player.Role)
	assert.NotEmpty(t, player.Specialties)
}

func TestPlayerChannels(t *testing.T) {
	player := NewPlayer("test", nil)

	// Channels should be initialized
	assert.NotNil(t, player.Send)
	assert.NotNil(t, player.Done)

	// Test send channel capacity
	select {
	case player.Send <- []byte("test"):
		// Should be able to send at least one message
		assert.True(t, true)
	default:
		assert.Fail(t, "Send channel should not block immediately")
	}

	// Clean up
	select {
	case <-player.Send:
		// Drain the channel
	default:
	}
}

func TestHostChannels(t *testing.T) {
	host := NewHost("test-host", nil)

	// Channels should be initialized
	assert.NotNil(t, host.Send)
	assert.NotNil(t, host.Done)

	// Test send channel capacity
	select {
	case host.Send <- []byte("test"):
		// Should be able to send at least one message
		assert.True(t, true)
	default:
		assert.Fail(t, "Send channel should not block immediately")
	}

	// Clean up
	select {
	case <-host.Send:
		// Drain the channel
	default:
	}
}

func TestPlayerTimestamps(t *testing.T) {
	player := NewPlayer("test", nil)

	// JoinedAt should be set
	assert.NotZero(t, player.JoinedAt)
	assert.True(t, player.JoinedAt.Before(time.Now().Add(time.Second)))
	assert.True(t, player.JoinedAt.After(time.Now().Add(-time.Minute)))

	// LastSeen should be zero initially
	assert.Zero(t, player.LastSeen)

	// Update LastSeen
	player.LastSeen = time.Now()
	assert.NotZero(t, player.LastSeen)
}

func TestHostTimestamp(t *testing.T) {
	host := NewHost("test-host", nil)

	// ConnectedAt should be set
	assert.NotZero(t, host.ConnectedAt)
	assert.True(t, host.ConnectedAt.Before(time.Now().Add(time.Second)))
	assert.True(t, host.ConnectedAt.After(time.Now().Add(-time.Minute)))
}
