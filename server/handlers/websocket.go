package handlers

import (
	"canvas-conundrum/config"
	"canvas-conundrum/constants"
	"canvas-conundrum/models"
	"canvas-conundrum/services"
	"canvas-conundrum/utils"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  config.WebSocketBufferSize,
	WriteBufferSize: config.WebSocketBufferSize,
	CheckOrigin: func(r *http.Request) bool {
		// In production, implement proper origin checking
		return true
	},
}

// HandlePlayerWebSocket handles player WebSocket connections
func HandlePlayerWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade player connection: %v", err)
		return
	}
	defer conn.Close()

	// Generate player ID
	playerID := utils.GeneratePlayerID()

	// Create player instance
	player := models.NewPlayer(playerID, conn)

	// Add player to game
	gameManager := services.GetGameInstance()
	if err := gameManager.AddPlayer(player); err != nil {
		log.Printf("Failed to add player: %v", err)
		sendConnectionError(conn, err.Error())
		return
	}

	// Send initial roles available message
	sendRolesAvailable(player)

	// Start connection handlers
	go handlePlayerWrite(player)

	// Handle incoming messages
	handlePlayerRead(player)

	// Clean up on disconnect
	gameManager.RemovePlayer(playerID)
}

// HandleHostWebSocket handles host WebSocket connections
func HandleHostWebSocket(w http.ResponseWriter, r *http.Request) {
	// Extract UUID from URL
	vars := mux.Vars(r)
	uuid := vars["uuid"]

	// Validate UUID matches configured host UUID
	if uuid != config.HostUUID {
		http.Error(w, "Invalid host UUID", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade host connection: %v", err)
		return
	}
	defer conn.Close()

	// Use configured host UUID for host ID
	hostID := config.HostUUID

	// Create host instance
	host := models.NewHost(hostID, conn)

	// Set host in game manager
	gameManager := services.GetGameInstance()
	if err := gameManager.SetHost(host); err != nil {
		log.Printf("Failed to set host: %v", err)
		sendConnectionError(conn, err.Error())
		return
	}

	// Send connection confirmed message
	sendHostConnectionConfirmed(host)

	// Start connection handlers
	go handleHostWrite(host)

	// Handle incoming messages
	handleHostRead(host)

	// Clean up on disconnect
	gameManager.RemoveHost()
}

// handlePlayerRead handles incoming messages from a player
func handlePlayerRead(player *models.Player) {
	defer func() {
		player.Connection.Close()
		// Safely close Done channel using recover to handle already closed channels
		func() {
			defer func() {
				if r := recover(); r != nil {
					// Channel was already closed, ignore
				}
			}()
			close(player.Done)
		}()
	}()

	player.Connection.SetReadDeadline(time.Now().Add(time.Duration(config.PongWait) * time.Second))
	player.Connection.SetPongHandler(func(string) error {
		player.Connection.SetReadDeadline(time.Now().Add(time.Duration(config.PongWait) * time.Second))
		return nil
	})

	for {
		_, message, err := player.Connection.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Player %s websocket error: %v", player.ID, err)
			}
			break
		}

		// Parse message
		msg, err := utils.ParseMessage(message)
		if err != nil {
			log.Printf("Failed to parse message from player %s: %v", player.ID, err)
			continue
		}

		// Validate authentication (except for initial setup)
		if msg.Event != constants.EventSetupToServerPlayerConfiguration {
			if msg.Auth == nil || msg.Auth.Token != player.ID {
				sendAuthError(player)
				continue
			}
		}

		// Handle message based on event type
		HandlePlayerMessage(player, msg)
	}
}

// handlePlayerWrite handles outgoing messages to a player
func handlePlayerWrite(player *models.Player) {
	ticker := time.NewTicker(time.Duration(config.PingPeriod) * time.Second)
	defer func() {
		ticker.Stop()
		if player.Connection != nil {
			player.Connection.Close()
		}
	}()

	for {
		select {
		case message, ok := <-player.Send:
			player.Connection.SetWriteDeadline(time.Now().Add(time.Duration(config.WriteWait) * time.Second))
			if !ok {
				player.Connection.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := player.Connection.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			player.Connection.SetWriteDeadline(time.Now().Add(time.Duration(config.WriteWait) * time.Second))
			if err := player.Connection.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}

		case <-player.Done:
			return
		}
	}
}

// handleHostRead handles incoming messages from the host
func handleHostRead(host *models.Host) {
	defer func() {
		host.Connection.Close()
		// Safely close Done channel using recover to handle already closed channels
		func() {
			defer func() {
				if r := recover(); r != nil {
					// Channel was already closed, ignore
				}
			}()
			close(host.Done)
		}()
	}()

	host.Connection.SetReadDeadline(time.Now().Add(time.Duration(config.PongWait) * time.Second))
	host.Connection.SetPongHandler(func(string) error {
		host.Connection.SetReadDeadline(time.Now().Add(time.Duration(config.PongWait) * time.Second))
		return nil
	})

	for {
		_, message, err := host.Connection.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Host websocket error: %v", err)
			}
			break
		}

		// Parse message
		msg, err := utils.ParseMessage(message)
		if err != nil {
			log.Printf("Failed to parse message from host: %v", err)
			continue
		}

		// Validate authentication
		if msg.Auth == nil || msg.Auth.Token != host.ID {
			sendHostAuthError(host)
			continue
		}

		// Handle message based on event type
		HandleHostMessage(host, msg)
	}
}

// handleHostWrite handles outgoing messages to the host
func handleHostWrite(host *models.Host) {
	ticker := time.NewTicker(time.Duration(config.PingPeriod) * time.Second)
	defer func() {
		ticker.Stop()
		if host.Connection != nil {
			host.Connection.Close()
		}
	}()

	for {
		select {
		case message, ok := <-host.Send:
			if host.Connection == nil {
				return
			}
			host.Connection.SetWriteDeadline(time.Now().Add(time.Duration(config.WriteWait) * time.Second))
			if !ok {
				host.Connection.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := host.Connection.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			if host.Connection == nil {
				return
			}
			host.Connection.SetWriteDeadline(time.Now().Add(time.Duration(config.WriteWait) * time.Second))
			if err := host.Connection.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}

		case <-host.Done:
			return
		}
	}
}

// sendConnectionError sends a connection error message
func sendConnectionError(conn *websocket.Conn, errorMsg string) {
	payload := utils.CreateErrorPayload(
		"connection_error",
		constants.ErrorCodeGameInProgress,
		errorMsg,
		"Cannot connect at this time",
	)

	msg := utils.NewServerMessage(constants.EventSystemToClientError, payload)
	data, _ := msg.Marshal()
	conn.WriteMessage(websocket.TextMessage, data)
}

// sendHostConnectionConfirmed sends connection confirmation to the host
func sendHostConnectionConfirmed(host *models.Host) {
	gameManager := services.GetGameInstance()
	game := gameManager.GetGame()

	payload := map[string]interface{}{
		"playerId": host.ID,
		"isHost":   true,
		"message":  "Connected as game host",
		"gameConfig": map[string]interface{}{
			"minPlayers":                     constants.MinPlayers,
			"maxPlayers":                     constants.MaxPlayers,
			"resourceGatheringRounds":        constants.ResourceGatheringRounds,
			"resourceGatheringRoundDuration": constants.ResourceGatheringRoundDuration,
			"puzzleBaseTime":                 constants.PuzzleBaseTime,
			"difficultyMode":                 string(game.Difficulty),
		},
	}

	broadcastService := gameManager.GetBroadcastService()
	if broadcastService != nil {
		broadcastService.SendToHost(host, constants.EventSetupToHostConnectionConfirmed, payload)
	}

	// Also send the initial player roster
	if broadcastService != nil {
		broadcastService.BroadcastLobbyStatus()
	}
}

// sendRolesAvailable sends available roles to a new player
func sendRolesAvailable(player *models.Player) {
	gameManager := services.GetGameInstance()
	roleDistribution := gameManager.GetRoleDistribution()

	// Check role availability
	maxPerRole := (gameManager.GetPlayerCount() + 3) / 4
	if maxPerRole < 1 {
		maxPerRole = 1 // Ensure at least 1 player can select each role
	}

	roles := []map[string]interface{}{
		{
			"roleType":       "art_enthusiast",
			"displayName":    "Art Enthusiast",
			"resourceBonus":  constants.RoleResourceMultiplier,
			"bonusTokenType": "clarity",
			"description":    "Excels at clarity token collection",
			"available":      roleDistribution[models.RoleArtEnthusiast] < maxPerRole,
		},
		{
			"roleType":       "detective",
			"displayName":    "Detective",
			"resourceBonus":  constants.RoleResourceMultiplier,
			"bonusTokenType": "guide",
			"description":    "Excels at guide token collection",
			"available":      roleDistribution[models.RoleDetective] < maxPerRole,
		},
		{
			"roleType":       "tourist",
			"displayName":    "Tourist",
			"resourceBonus":  constants.RoleResourceMultiplier,
			"bonusTokenType": "chronos",
			"description":    "Excels at chronos token collection",
			"available":      roleDistribution[models.RoleTourist] < maxPerRole,
		},
		{
			"roleType":       "janitor",
			"displayName":    "Janitor",
			"resourceBonus":  constants.RoleResourceMultiplier,
			"bonusTokenType": "anchor",
			"description":    "Excels at anchor token collection",
			"available":      roleDistribution[models.RoleJanitor] < maxPerRole,
		},
	}

	payload := map[string]interface{}{
		"playerId": player.ID,
		"isHost":   false,
		"roles":    roles,
		"triviaCategories": []string{
			"general", "geography", "history", "music", "science", "video_games",
		},
		"maxSpecialties": constants.MaxSpecialtiesPerPlayer,
	}

	broadcastService := services.GetGameInstance().GetBroadcastService()
	if broadcastService != nil {
		broadcastService.SendToPlayer(player, constants.EventSetupToPlayerRolesAvailable, payload)
	}
}

// sendAuthError sends authentication error to player
func sendAuthError(player *models.Player) {
	broadcastService := services.GetGameInstance().GetBroadcastService()
	if broadcastService != nil {
		broadcastService.SendError(
			player,
			constants.ErrorCodeInvalidToken,
			constants.ErrorMessageInvalidToken,
			"Authentication token mismatch",
		)
	}
}

// sendHostAuthError sends authentication error to host
func sendHostAuthError(host *models.Host) {
	broadcastService := services.GetGameInstance().GetBroadcastService()
	if broadcastService != nil {
		broadcastService.SendError(
			host,
			constants.ErrorCodeInvalidToken,
			constants.ErrorMessageInvalidToken,
			"Authentication token mismatch",
		)
	}
}
