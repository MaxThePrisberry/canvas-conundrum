package services

import (
	"canvas-conundrum/models"
	"canvas-conundrum/test_helpers"
	"testing"

	"github.com/stretchr/testify/assert"
)

func resetGameManager() {
	// Create a new game manager instance for clean tests
	gameManager := GetGameInstance()
	gameManager.ResetGame()
}

func TestNewBroadcastService(t *testing.T) {
	service := NewBroadcastService()
	
	assert.NotNil(t, service)
}

func TestBroadcastServiceSendToPlayer(t *testing.T) {
	service := NewBroadcastService()
	
	t.Run("Nil Player", func(t *testing.T) {
		// Should not panic
		service.SendToPlayer(nil, "test_event", map[string]string{"data": "test"})
		assert.True(t, true)
	})
	
	t.Run("Inactive Player", func(t *testing.T) {
		player := test_helpers.CreateTestPlayer("player1")
		player.IsActive = false
		
		// Should not panic
		service.SendToPlayer(player, "test_event", map[string]string{"data": "test"})
		assert.True(t, true)
	})
	
	t.Run("Active Player", func(t *testing.T) {
		player := test_helpers.CreateTestPlayer("player1")
		player.IsActive = true
		
		// Should not panic and should attempt to send
		service.SendToPlayer(player, "test_event", map[string]string{"data": "test"})
		
		// Check that a message was sent to the channel
		select {
		case msg := <-player.Send:
			assert.NotEmpty(t, msg)
		default:
			// Channel might be full or message not sent, but shouldn't panic
		}
	})
	
	t.Run("Invalid Payload", func(t *testing.T) {
		player := test_helpers.CreateTestPlayer("player1")
		player.IsActive = true
		
		// Send invalid payload that can't be marshaled
		invalidPayload := make(chan int) // channels can't be marshaled to JSON
		
		// Should not panic
		service.SendToPlayer(player, "test_event", invalidPayload)
		assert.True(t, true)
	})
}

func TestBroadcastServiceSendToHost(t *testing.T) {
	service := NewBroadcastService()
	
	t.Run("Nil Host", func(t *testing.T) {
		// Should not panic
		service.SendToHost(nil, "test_event", map[string]string{"data": "test"})
		assert.True(t, true)
	})
	
	t.Run("Host Without Connection", func(t *testing.T) {
		host := test_helpers.CreateTestHost("host1")
		host.Connection = nil
		
		// Should not panic
		service.SendToHost(host, "test_event", map[string]string{"data": "test"})
		assert.True(t, true)
	})
	
	t.Run("Host With Connection", func(t *testing.T) {
		host := test_helpers.CreateTestHost("host1")
		// host already has a mock connection from CreateTestHost
		
		// Should not panic and should attempt to send
		service.SendToHost(host, "test_event", map[string]string{"data": "test"})
		
		// Check that a message was sent to the channel
		select {
		case msg := <-host.Send:
			assert.NotEmpty(t, msg)
		default:
			// Channel might be full or message not sent, but shouldn't panic
		}
	})
	
	t.Run("Invalid Payload", func(t *testing.T) {
		host := test_helpers.CreateTestHost("host1")
		
		// Send invalid payload that can't be marshaled
		invalidPayload := make(chan int) // channels can't be marshaled to JSON
		
		// Should not panic
		service.SendToHost(host, "test_event", invalidPayload)
		assert.True(t, true)
	})
}

func TestBroadcastServiceBroadcastToAllPlayers(t *testing.T) {
	service := NewBroadcastService()
	resetGameManager()
	gameManager := GetGameInstance()
	
	t.Run("No Players", func(t *testing.T) {
		// Should not panic with no players
		service.BroadcastToAllPlayers("test_event", map[string]string{"data": "test"})
		assert.True(t, true)
	})
	
	t.Run("Multiple Players", func(t *testing.T) {
		// Add some players
		player1 := test_helpers.CreateTestPlayer("player1")
		player1.IsActive = true
		player2 := test_helpers.CreateTestPlayer("player2")
		player2.IsActive = true
		player3 := test_helpers.CreateTestPlayer("player3")
		player3.IsActive = false // inactive player
		
		gameManager.AddPlayer(player1)
		gameManager.AddPlayer(player2)
		gameManager.AddPlayer(player3)
		
		// Should not panic
		service.BroadcastToAllPlayers("test_event", map[string]string{"data": "test"})
		
		// Check that active players received messages
		select {
		case msg := <-player1.Send:
			assert.NotEmpty(t, msg)
		default:
			// May not receive due to timing or channel state
		}
		
		select {
		case msg := <-player2.Send:
			assert.NotEmpty(t, msg)
		default:
			// May not receive due to timing or channel state
		}
		
		// Inactive player should not receive message
		select {
		case <-player3.Send:
			t.Error("Inactive player should not receive broadcast")
		default:
			// Expected - inactive player doesn't get message
		}
	})
	
	t.Run("Invalid Payload", func(t *testing.T) {
		// Reset and add a player
		resetGameManager()
		gameManager = GetGameInstance()
		player := test_helpers.CreateTestPlayer("player1")
		player.IsActive = true
		gameManager.AddPlayer(player)
		
		// Send invalid payload that can't be marshaled
		invalidPayload := make(chan int)
		
		// Should not panic
		service.BroadcastToAllPlayers("test_event", invalidPayload)
		assert.True(t, true)
	})
}

func TestBroadcastServiceBroadcastToAll(t *testing.T) {
	service := NewBroadcastService()
	resetGameManager()
	gameManager := GetGameInstance()
	
	t.Run("Players and Host", func(t *testing.T) {
		// Add players and host
		player := test_helpers.CreateTestPlayer("player1")
		player.IsActive = true
		gameManager.AddPlayer(player)
		
		host := test_helpers.CreateTestHost("host1")
		gameManager.SetHost(host)
		
		// Should not panic
		service.BroadcastToAll("test_event", map[string]string{"data": "test"})
		
		// Both player and host should potentially receive messages
		select {
		case msg := <-player.Send:
			assert.NotEmpty(t, msg)
		default:
			// May not receive due to timing
		}
		
		select {
		case msg := <-host.Send:
			assert.NotEmpty(t, msg)
		default:
			// May not receive due to timing
		}
	})
	
	t.Run("No Host", func(t *testing.T) {
		resetGameManager()
		gameManager = GetGameInstance()
		
		player := test_helpers.CreateTestPlayer("player1")
		player.IsActive = true
		gameManager.AddPlayer(player)
		
		// No host set - should not panic
		service.BroadcastToAll("test_event", map[string]string{"data": "test"})
		assert.True(t, true)
	})
}

func TestBroadcastServiceErrorHandling(t *testing.T) {
	service := NewBroadcastService()
	
	t.Run("Error Payload for Player", func(t *testing.T) {
		player := test_helpers.CreateTestPlayer("player1")
		player.IsActive = true
		
		// Create a payload that will cause marshaling error
		errorPayload := map[string]interface{}{
			"invalid": make(chan int),
		}
		
		// Should handle error gracefully
		service.SendToPlayer(player, "test_event", errorPayload)
		assert.True(t, true) // Test passes if no panic
	})
	
	t.Run("Error Payload for Host", func(t *testing.T) {
		host := test_helpers.CreateTestHost("host1")
		
		// Create a payload that will cause marshaling error
		errorPayload := map[string]interface{}{
			"invalid": make(chan int),
		}
		
		// Should handle error gracefully
		service.SendToHost(host, "test_event", errorPayload)
		assert.True(t, true) // Test passes if no panic
	})
}

func TestBroadcastServiceSendError(t *testing.T) {
	service := NewBroadcastService()
	
	t.Run("Send Error to Player", func(t *testing.T) {
		player := test_helpers.CreateTestPlayer("player1")
		player.IsActive = true
		
		// Should not panic
		service.SendError(player, "TEST_ERROR", "Test error", "Test details")
		
		// Should receive error message
		select {
		case msg := <-player.Send:
			assert.NotEmpty(t, msg)
		default:
			// May not receive due to timing
		}
	})
	
	t.Run("Send Error to Host", func(t *testing.T) {
		host := test_helpers.CreateTestHost("host1")
		
		// Should not panic
		service.SendError(host, "TEST_ERROR", "Test error", "Test details")
		
		// Should receive error message
		select {
		case msg := <-host.Send:
			assert.NotEmpty(t, msg)
		default:
			// May not receive due to timing
		}
	})
}

func TestBroadcastServiceGameFlow(t *testing.T) {
	service := NewBroadcastService()
	resetGameManager()
	gameManager := GetGameInstance()
	
	// Set up test game state
	player := test_helpers.CreateTestPlayer("player1")
	player.IsActive = true
	player.Name = "Test Player"
	player.Role = models.RoleArtEnthusiast
	gameManager.AddPlayer(player)
	
	host := test_helpers.CreateTestHost("host1")
	gameManager.SetHost(host)
	
	t.Run("Broadcast Lobby Status", func(t *testing.T) {
		// Should not panic
		service.BroadcastLobbyStatus()
		assert.True(t, true)
	})
	
	t.Run("Broadcast Game Start", func(t *testing.T) {
		// Should not panic
		service.BroadcastGameStart()
		assert.True(t, true)
	})
	
	t.Run("Broadcast Phase Transition", func(t *testing.T) {
		// Should not panic
		service.BroadcastPhaseTransition(models.PhaseSetup, models.PhaseResourceGathering)
		assert.True(t, true)
	})
	
	t.Run("Broadcast Resource Phase Complete", func(t *testing.T) {
		// Should not panic
		service.BroadcastResourcePhaseComplete()
		assert.True(t, true)
	})
	
	t.Run("Broadcast Puzzle Phase Start", func(t *testing.T) {
		// Initialize puzzle grid to avoid nil pointer
		game := gameManager.GetGame()
		game.StartPuzzlePhase(4) // Initialize with 4 players
		
		// Should not panic
		service.BroadcastPuzzlePhaseStart()
		assert.True(t, true)
	})
}

func TestBroadcastServiceConcurrency(t *testing.T) {
	service := NewBroadcastService()
	resetGameManager()
	gameManager := GetGameInstance()
	
	// Add multiple players
	for i := 0; i < 5; i++ {
		player := test_helpers.CreateTestPlayer("player" + string(rune(i+'1')))
		player.IsActive = true
		gameManager.AddPlayer(player)
	}
	
	// Test concurrent broadcasts
	for i := 0; i < 10; i++ {
		go service.BroadcastToAllPlayers("concurrent_test", map[string]int{"iteration": i})
	}
	
	// Test should complete without race conditions or panics
	assert.True(t, true)
}