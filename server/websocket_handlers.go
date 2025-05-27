package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"canvas-conundrum/server/constants"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// In production, implement proper origin checking
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// WebSocketHandler handles WebSocket connections
type WebSocketHandler struct {
	playerManager *PlayerManager
	gameManager   *GameManager
	eventHandlers *EventHandlers
	broadcastChan chan BroadcastMessage
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(pm *PlayerManager, gm *GameManager, eh *EventHandlers, bc chan BroadcastMessage) *WebSocketHandler {
	return &WebSocketHandler{
		playerManager: pm,
		gameManager:   gm,
		eventHandlers: eh,
		broadcastChan: bc,
	}
}

// HandleConnection handles incoming WebSocket connections
func (wsh *WebSocketHandler) HandleConnection(w http.ResponseWriter, r *http.Request) {
	// Check if reconnecting
	playerID := r.URL.Query().Get("playerId")

	// Upgrade connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	var player *Player

	// Handle reconnection or new connection
	if playerID != "" {
		// Attempt reconnection
		if err := wsh.playerManager.ReconnectPlayer(playerID, conn); err != nil {
			log.Printf("Reconnection failed for player %s: %v", playerID, err)
			// Create new player if reconnection fails
			player = wsh.playerManager.CreatePlayer(conn)
		} else {
			player, _ = wsh.playerManager.GetPlayer(playerID)
			log.Printf("Player %s reconnected", playerID)

			// Send current game state on reconnection
			wsh.sendReconnectionState(player)
		}
	} else {
		// New connection
		// Check if we can accept new players
		if wsh.gameManager.GetPhase() != PhaseSetup {
			conn.WriteJSON(BaseMessage{
				Type: MsgError,
				Payload: mustMarshal(map[string]string{
					"error": "Cannot join game in progress",
				}),
			})
			return
		}

		if wsh.playerManager.GetPlayerCount() >= constants.MaxPlayers {
			conn.WriteJSON(BaseMessage{
				Type: MsgError,
				Payload: mustMarshal(map[string]string{
					"error": "Game is full",
				}),
			})
			return
		}

		player = wsh.playerManager.CreatePlayer(conn)
		log.Printf("New player connected: %s", player.ID)
	}

	// Set up ping/pong handlers
	conn.SetReadDeadline(time.Now().Add(constants.WebSocketPongTimeout))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(constants.WebSocketPongTimeout))
		return nil
	})

	// Start ping ticker
	pingTicker := time.NewTicker(constants.WebSocketPingInterval)
	defer pingTicker.Stop()

	// Handle initial join
	if err := wsh.eventHandlers.HandlePlayerJoin(player, nil); err != nil {
		log.Printf("Error handling player join: %v", err)
	}

	// Broadcast lobby status
	wsh.eventHandlers.broadcastLobbyStatus()

	// Message handling goroutine
	go wsh.handlePlayerMessages(player)

	// Keep connection alive and handle pings
	for {
		select {
		case <-pingTicker.C:
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("Ping failed for player %s: %v", player.ID, err)
				wsh.handleDisconnection(player)
				return
			}
		}
	}
}

// handlePlayerMessages handles incoming messages from a player
func (wsh *WebSocketHandler) handlePlayerMessages(player *Player) {
	defer wsh.handleDisconnection(player)

	for {
		var baseMsg BaseMessage
		err := player.Connection.ReadJSON(&baseMsg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error for player %s: %v", player.ID, err)
			}
			return
		}

		// Update last seen
		player.mu.Lock()
		player.LastSeen = time.Now()
		player.mu.Unlock()

		// Route message based on type
		switch baseMsg.Type {
		case MsgRoleSelection, MsgTriviaSpecialtySelection, MsgResourceLocationVerified,
			MsgTriviaAnswer, MsgSegmentCompleted, MsgFragmentMoveRequest,
			MsgPlayerReady, MsgHostStartGame, MsgHostStartPuzzle:

			// These messages require authentication
			var authWrapper AuthWrapper
			if err := json.Unmarshal(baseMsg.Payload, &authWrapper); err != nil {
				wsh.sendError(player, "Invalid message format")
				continue
			}

			// Verify authentication
			if authWrapper.Auth.PlayerID != player.ID {
				wsh.sendError(player, "Authentication failed")
				continue
			}

			// Route to appropriate handler
			wsh.routeAuthenticatedMessage(player.ID, baseMsg.Type, authWrapper.Payload)

		default:
			wsh.sendError(player, fmt.Sprintf("Unknown message type: %s", baseMsg.Type))
		}
	}
}

// routeAuthenticatedMessage routes authenticated messages to appropriate handlers
func (wsh *WebSocketHandler) routeAuthenticatedMessage(playerID string, msgType string, payload json.RawMessage) {
	var err error

	switch msgType {
	case MsgRoleSelection:
		err = wsh.eventHandlers.HandleRoleSelection(playerID, payload)

	case MsgTriviaSpecialtySelection:
		err = wsh.eventHandlers.HandleTriviaSpecialtySelection(playerID, payload)

	case MsgPlayerReady:
		err = wsh.eventHandlers.HandlePlayerReady(playerID, payload)

	case MsgHostStartGame:
		err = wsh.eventHandlers.HandleHostStartGame(playerID, payload)

	case MsgResourceLocationVerified:
		err = wsh.eventHandlers.HandleResourceLocationVerified(playerID, payload)

	case MsgTriviaAnswer:
		err = wsh.eventHandlers.HandleTriviaAnswer(playerID, payload)

	case MsgSegmentCompleted:
		err = wsh.eventHandlers.HandleSegmentCompleted(playerID, payload)

	case MsgFragmentMoveRequest:
		err = wsh.eventHandlers.HandleFragmentMoveRequest(playerID, payload)

	case MsgHostStartPuzzle:
		err = wsh.eventHandlers.HandleHostStartPuzzle(playerID, payload)
	}

	if err != nil {
		log.Printf("Error handling %s for player %s: %v", msgType, playerID, err)
		if player, _ := wsh.playerManager.GetPlayer(playerID); player != nil {
			wsh.sendError(player, err.Error())
		}
	}
}

// handleDisconnection handles player disconnection
func (wsh *WebSocketHandler) handleDisconnection(player *Player) {
	log.Printf("Player %s disconnected", player.ID)

	// Mark as disconnected
	wsh.playerManager.DisconnectPlayer(player.ID)

	// Handle based on game phase
	phase := wsh.gameManager.GetPhase()

	switch phase {
	case PhaseSetup:
		// In setup, we can wait for reconnection
		wsh.eventHandlers.broadcastLobbyStatus()

	case PhaseResourceGathering:
		// Can reconnect during resource gathering
		// Continue game normally

	case PhasePuzzleAssembly:
		// During puzzle assembly, auto-solve their fragment
		wsh.gameManager.mu.Lock()
		fragmentID := fmt.Sprintf("fragment_%s", player.ID)
		if fragment, exists := wsh.gameManager.state.PuzzleFragments[fragmentID]; exists {
			fragment.Solved = true
			// Randomly relocate fragment
			// This is simplified - you might want more sophisticated logic
		}
		wsh.gameManager.mu.Unlock()

		// Notify others
		wsh.broadcastChan <- BroadcastMessage{
			Type: MsgCentralPuzzleState,
			Payload: map[string]interface{}{
				"playerDisconnected": player.ID,
			},
		}

	case PhasePostGame:
		// No special handling needed
	}

	// Update host if needed
	if player.IsHost {
		// Transfer host to another connected player
		players := wsh.playerManager.GetConnectedPlayers()
		if len(players) > 0 {
			newHost := players[0]
			newHost.mu.Lock()
			newHost.IsHost = true
			newHost.mu.Unlock()

			log.Printf("Host transferred to player %s", newHost.ID)

			// Notify new host
			sendToPlayer(newHost, MsgHostUpdate, map[string]interface{}{
				"message": "You are now the host",
			})
		}
	}
}

// sendReconnectionState sends current game state to reconnecting player
func (wsh *WebSocketHandler) sendReconnectionState(player *Player) {
	phase := wsh.gameManager.GetPhase()

	// Always send available roles first
	roles := wsh.playerManager.GetAvailableRoles()
	sendToPlayer(player, MsgAvailableRoles, map[string]interface{}{
		"playerId":         player.ID,
		"roles":            roles,
		"triviaCategories": constants.TriviaCategories,
	})

	switch phase {
	case PhaseSetup:
		// Send lobby status
		wsh.eventHandlers.broadcastLobbyStatus()

	case PhaseResourceGathering:
		// Send resource phase info
		sendToPlayer(player, MsgResourcePhaseStart, map[string]interface{}{
			"resourceHashes": constants.ResourceStationHashes,
		})

		// Send current progress
		wsh.gameManager.sendTeamProgressUpdate()

	case PhasePuzzleAssembly:
		// Send puzzle phase info
		wsh.gameManager.mu.RLock()
		fragmentID := fmt.Sprintf("fragment_%s", player.ID)
		fragment := wsh.gameManager.state.PuzzleFragments[fragmentID]
		segmentID := ""
		if fragment != nil {
			segmentID = fmt.Sprintf("segment_%c%d", 'a'+fragment.Position.Y, fragment.Position.X+1)
		}
		imageID := wsh.gameManager.state.PuzzleImageID
		gridSize := wsh.gameManager.state.GridSize
		wsh.gameManager.mu.RUnlock()

		sendToPlayer(player, MsgPuzzlePhaseLoad, map[string]interface{}{
			"imageId":   imageID,
			"segmentId": segmentID,
			"gridSize":  gridSize,
		})

		// Send current puzzle state
		wsh.gameManager.broadcastPuzzleState()

	case PhasePostGame:
		// Send analytics
		// Game is ending anyway
	}
}

// sendError sends an error message to a player
func (wsh *WebSocketHandler) sendError(player *Player, message string) {
	sendToPlayer(player, MsgError, map[string]interface{}{
		"error": message,
	})
}

// StartBroadcaster starts the message broadcaster goroutine
func (wsh *WebSocketHandler) StartBroadcaster() {
	go func() {
		for msg := range wsh.broadcastChan {
			players := wsh.playerManager.GetAllPlayers()

			for _, player := range players {
				// Apply filter if present
				if msg.Filter != nil && !msg.Filter(player) {
					continue
				}

				// Only send to connected players
				player.mu.RLock()
				connected := player.State == StateConnected
				player.mu.RUnlock()

				if connected {
					if err := sendToPlayer(player, msg.Type, msg.Payload); err != nil {
						log.Printf("Error broadcasting to player %s: %v", player.ID, err)
					}
				}
			}
		}
	}()
}
