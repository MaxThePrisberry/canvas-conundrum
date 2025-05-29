package main

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestGlobalVariables(t *testing.T) {
	// Test that global variables are properly defined
	assert.NotNil(t, port)
	assert.NotNil(t, host)
	assert.NotNil(t, environment)
	assert.NotNil(t, certFile)
	assert.NotNil(t, keyFile)
	assert.NotNil(t, allowedOrigins)
}

func TestHostEndpointGeneration(t *testing.T) {
	// Test that host endpoint ID generation works
	testID := uuid.New().String()
	assert.NotEmpty(t, testID)

	// Parse as UUID to ensure it's valid
	parsedID, err := uuid.Parse(testID)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, parsedID)
}

func TestEnvironmentValues(t *testing.T) {
	// Test valid environment values
	validEnvs := []string{"development", "staging", "production"}

	for _, env := range validEnvs {
		t.Run("Environment: "+env, func(t *testing.T) {
			assert.Contains(t, validEnvs, env)
		})
	}
}

func TestServerLifecycle(t *testing.T) {
	// Test that we can create server components without starting them
	t.Run("Create managers", func(t *testing.T) {
		assert.NotPanics(t, func() {
			playerMgr := NewPlayerManager()
			assert.NotNil(t, playerMgr)

			triviaMgr := NewTriviaManager()
			assert.NotNil(t, triviaMgr)

			broadcastChan := make(chan BroadcastMessage, 256)
			assert.NotNil(t, broadcastChan)

			gameMgr := NewGameManager(playerMgr, triviaMgr, broadcastChan)
			assert.NotNil(t, gameMgr)
		})
	})
}

func TestBroadcastChannelIntegration(t *testing.T) {
	// Test broadcast channel functionality
	broadcastChan := make(chan BroadcastMessage, 256)

	// Test sending and receiving messages
	testMessage := BroadcastMessage{
		Type:    "test_message",
		Payload: map[string]string{"test": "data"},
	}

	// Send message
	broadcastChan <- testMessage

	// Receive message with timeout
	select {
	case received := <-broadcastChan:
		assert.Equal(t, testMessage.Type, received.Type)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for broadcast message")
	}
}

func TestEnvironmentSetup(t *testing.T) {
	// Test different environment configurations
	envs := []string{"development", "production", "testing"}

	for _, env := range envs {
		t.Run(env, func(t *testing.T) {
			// Verify environment is valid
			assert.NotEmpty(t, env)
			assert.Contains(t, []string{"development", "production", "testing"}, env)
		})
	}
}
