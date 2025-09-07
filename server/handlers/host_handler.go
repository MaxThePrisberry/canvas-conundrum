package handlers

import (
	"canvas-conundrum/config"
	"canvas-conundrum/models"
	"canvas-conundrum/services"
	"canvas-conundrum/utils"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// HandleHostMessage handles incoming messages from the host
func HandleHostMessage(host *models.Host, msg *utils.Message) {
	gameManager := services.GetGameInstance()
	broadcastService := gameManager.GetBroadcastService()

	log.Printf("Host sent event: %s", msg.Event)

	switch msg.Event {
	case config.EventSetupToServerStartGame:
		handleHostStartGame(host)

	case config.EventPuzzleToServerStartTimer:
		handleHostStartPuzzleTimer(host)

	case config.EventAnalyticsToServerResetGame:
		handleHostResetGame(host, msg.Payload)

	case config.EventSystemPing:
		handleHostPing(host, msg.Payload)

	default:
		log.Printf("Unknown event from host: %s", msg.Event)
		if broadcastService != nil {
			broadcastService.SendError(
				host,
				"UNKNOWN_EVENT",
				"Unknown event type",
				"Event "+msg.Event+" is not recognized",
			)
		}
	}
}

// handleHostStartGame handles game start request from host
func handleHostStartGame(host *models.Host) {
	gameManager := services.GetGameInstance()
	broadcastService := gameManager.GetBroadcastService()

	// Check if game can be started
	if !gameManager.CanStartGame() {
		log.Println("Cannot start game - insufficient ready players")
		if broadcastService != nil {
			broadcastService.SendError(
				host,
				config.ErrorCodeInsufficientPlayers,
				config.ErrorMessageInsufficientPlayers,
				fmt.Sprintf("Need at least %d ready players to start", config.MinPlayers),
			)
		}
		return
	}

	// Start the game
	err := gameManager.StartGame()
	if err != nil {
		log.Printf("Failed to start game: %v", err)
		if broadcastService != nil {
			broadcastService.SendError(
				host,
				config.ErrorCodeGameInProgress,
				"Failed to start game",
				err.Error(),
			)
		}
		return
	}

	// Broadcast game start
	if broadcastService != nil {
		broadcastService.BroadcastGameStart()
	}

	log.Println("Game started by host")
}

// handleHostStartPuzzleTimer handles puzzle timer start request from host
func handleHostStartPuzzleTimer(host *models.Host) {
	gameManager := services.GetGameInstance()
	broadcastService := gameManager.GetBroadcastService()

	// Start puzzle timer
	err := gameManager.StartPuzzleTimer()
	if err != nil {
		log.Printf("Failed to start puzzle timer: %v", err)
		if broadcastService != nil {
			broadcastService.SendError(
				host,
				"TIMER_START_ERROR",
				"Failed to start timer",
				err.Error(),
			)
		}
		return
	}

	log.Println("Puzzle timer started by host")
}

// handleHostResetGame handles game reset request from host
func handleHostResetGame(host *models.Host, payload json.RawMessage) {
	var data struct {
		ConfirmReset  bool `json:"confirmReset"`
		SaveAnalytics bool `json:"saveAnalytics"`
	}

	if err := json.Unmarshal(payload, &data); err != nil {
		log.Printf("Failed to parse reset request: %v", err)
		return
	}

	if !data.ConfirmReset {
		log.Println("Reset not confirmed by host")
		return
	}

	gameManager := services.GetGameInstance()
	broadcastService := gameManager.GetBroadcastService()

	// Analytics are kept in memory only as designed
	// Game state will be completely reset after server restart
	if data.SaveAnalytics {
		log.Println("Analytics save requested - keeping in memory only (no persistence by design)")
	}

	// Broadcast reset to all participants
	if broadcastService != nil {
		resetPayload := map[string]interface{}{
			"reason":                "host_initiated_reset",
			"message":               "Game resetting. Please rejoin to start a new game.",
			"reconnectRequired":     true,
			"reconnectInstructions": "Refresh your browser and reconnect to join the next game",
			"gracePeriod":           30,
			"newGameAvailable":      true,
		}
		broadcastService.BroadcastToAll(config.EventAnalyticsToClientGameReset, resetPayload)
	}

	// Reset the game
	gameManager.ResetGame()

	log.Println("Game reset by host")
}

// handleHostPing handles ping messages from host
func handleHostPing(host *models.Host, payload json.RawMessage) {
	var data struct {
		ClientTimestamp string `json:"clientTimestamp"`
		SequenceNumber  int    `json:"sequenceNumber"`
	}

	if err := json.Unmarshal(payload, &data); err != nil {
		log.Printf("Failed to parse host ping: %v", err)
		return
	}

	gameManager := services.GetGameInstance()
	broadcastService := gameManager.GetBroadcastService()

	// Send pong response
	if broadcastService != nil {
		pongPayload := map[string]interface{}{
			"serverTimestamp": time.Now().Format(time.RFC3339),
			"clientTimestamp": data.ClientTimestamp,
			"sequenceNumber":  data.SequenceNumber,
			"serverHealth": map[string]interface{}{
				"activeConnections": gameManager.GetPlayerCount() + 1,
				"serverLoad":        0.15, // Placeholder
				"gamePhase":         gameManager.GetCurrentPhase(),
			},
		}

		broadcastService.SendToHost(host, config.EventSystemPong, pongPayload)
	}
}
