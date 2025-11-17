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
		handleHostStartGame(host, msg.Payload)

	case config.EventPuzzleToServerPhaseStart:
		handleHostStartPuzzlePhase(host)

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
func handleHostStartGame(host *models.Host, payload interface{}) {
	gameManager := services.GetGameInstance()
	broadcastService := gameManager.GetBroadcastService()

	// Extract difficulty from payload
	var difficulty string = "medium" // default
	if payload != nil {
		// Handle both parsed map and raw JSON bytes
		var payloadMap map[string]interface{}

		if pMap, ok := payload.(map[string]interface{}); ok {
			// Already parsed
			payloadMap = pMap
		} else if bytes, ok := payload.([]byte); ok {
			// Raw JSON bytes - need to parse
			if err := json.Unmarshal(bytes, &payloadMap); err != nil {
				log.Printf("Failed to parse JSON payload: %v", err)
			}
		} else if rawMsg, ok := payload.(json.RawMessage); ok {
			// JSON RawMessage - need to parse
			if err := json.Unmarshal(rawMsg, &payloadMap); err != nil {
				log.Printf("Failed to parse RawMessage payload: %v", err)
			}
		}

		if payloadMap != nil {
			if diff, exists := payloadMap["difficulty"]; exists && diff != nil {
				if diffStr, ok := diff.(string); ok {
					difficulty = diffStr
				}
			}
		}
	}

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

	// Start the game with specified difficulty
	err := gameManager.StartGame(difficulty)
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

	// The resource phase start broadcast is handled automatically in StartGame()

	log.Println("Game started by host")
}

// handleHostStartPuzzlePhase handles puzzle phase start request from host
func handleHostStartPuzzlePhase(host *models.Host) {
	gameManager := services.GetGameInstance()
	broadcastService := gameManager.GetBroadcastService()

	// Start puzzle phase
	err := gameManager.StartPuzzleTimer()
	if err != nil {
		log.Printf("Failed to start puzzle timer: %v", err)
		if broadcastService != nil {
			broadcastService.SendError(
				host,
				"PHASE_START_ERROR",
				"Failed to start puzzle phase",
				err.Error(),
			)
		}
		return
	}

	log.Println("Puzzle phase started by host")
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
			"reconnectRequired":     true,
			"reconnectInstructions": "Refresh your browser and reconnect to join the next game",
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
				"gamePhase":         string(gameManager.GetCurrentPhase()),
			},
		}

		broadcastService.SendToHost(host, config.EventSystemPong, pongPayload)
	}
}
