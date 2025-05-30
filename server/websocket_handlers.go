package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/MaxThePrisberry/canvas-conundrum/server/constants"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Use the new CORS validation function
		return isValidOrigin(r.Header.Get("Origin"))
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Enhanced error handling
	Error: func(w http.ResponseWriter, r *http.Request, status int, reason error) {
		log.Printf("WebSocket upgrade error: %v (status: %d)", reason, status)
		http.Error(w, "WebSocket upgrade failed", status)
	},
}

// WebSocketHandler handles WebSocket connections with enhanced validation
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

// HandleConnection handles incoming WebSocket connections with ENHANCED reconnection restrictions
func (wsh *WebSocketHandler) HandleConnection(w http.ResponseWriter, r *http.Request, isHost bool) {
	// Enhanced connection validation
	if !wsh.validateConnectionRequest(r) {
		http.Error(w, "Invalid connection request", http.StatusBadRequest)
		return
	}

	// Check if reconnecting
	playerID := r.URL.Query().Get("playerId")

	// Validate player ID format if provided
	if playerID != "" {
		if err := validatePlayerID(playerID); err != nil {
			log.Printf("Invalid player ID format in connection: %s", playerID)
			http.Error(w, "Invalid player ID format", http.StatusBadRequest)
			return
		}
	}

	// Upgrade connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	// Set connection limits
	conn.SetReadLimit(8192) // 8KB max message size

	var player *Player

	// Handle reconnection or new connection
	if playerID != "" && !isHost {
		// ENHANCED: Check if reconnection is allowed during current phase
		phase := wsh.gameManager.GetPhase()
		if phase == PhasePuzzleAssembly {
			wsh.sendConnectionError(conn, constants.ErrReconnectionForbidden)
			log.Printf("Blocked reconnection attempt during puzzle assembly phase: player %s", playerID)
			return
		}

		// Attempt regular player reconnection (only allowed in setup and resource gathering)
		if err := wsh.playerManager.ReconnectPlayer(playerID, conn); err != nil {
			log.Printf("Reconnection failed for player %s: %v", playerID, err)
			// Create new player if reconnection fails
			player = wsh.playerManager.CreatePlayer(conn, false)
		} else {
			player, _ = wsh.playerManager.GetPlayer(playerID)
			log.Printf("Player %s reconnected during %s phase", playerID, phase.String())

			// Send current game state on reconnection
			wsh.sendReconnectionState(player)
		}
	} else if playerID != "" && isHost {
		// Host reconnection is ALWAYS allowed
		existingPlayer, err := wsh.playerManager.GetPlayer(playerID)
		if err != nil || !existingPlayer.IsHost {
			log.Printf("Host reconnection failed for player %s: %v", playerID, err)
			// Check if another host is already connected
			if wsh.playerManager.GetHost() != nil {
				wsh.sendConnectionError(conn, constants.ErrHostExists)
				return
			}
			player = wsh.playerManager.CreatePlayer(conn, true)
		} else {
			// Reconnect existing host
			if err := wsh.playerManager.ReconnectPlayer(playerID, conn); err != nil {
				log.Printf("Host reconnection failed for player %s: %v", playerID, err)
				if wsh.playerManager.GetConnectedHost() != nil {
					wsh.sendConnectionError(conn, constants.ErrHostExists)
					return
				}
				player = wsh.playerManager.CreatePlayer(conn, true)
			} else {
				player = existingPlayer
				log.Printf("Host %s reconnected", playerID)
				wsh.sendReconnectionState(player)
			}
		}
	} else {
		// New connection
		if isHost {
			// Check if there's already a host
			existingHost := wsh.playerManager.GetConnectedHost()
			if existingHost != nil {
				wsh.sendConnectionError(conn, constants.ErrHostExists)
				return
			}
			player = wsh.playerManager.CreatePlayer(conn, true)
			log.Printf("New host connected: %s", player.ID)
		} else {
			// New regular player connection validation
			if !wsh.canAcceptNewPlayer() {
				wsh.sendConnectionError(conn, "Cannot join game at this time")
				return
			}
			player = wsh.playerManager.CreatePlayer(conn, false)
			log.Printf("New player connected: %s", player.ID)
		}
	}

	// Set up enhanced ping/pong handlers
	conn.SetReadDeadline(time.Now().Add(constants.WebSocketPongTimeout))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(constants.WebSocketPongTimeout))
		return nil
	})

	// Set up close handler for immediate disconnect detection
	conn.SetCloseHandler(func(code int, text string) error {
		log.Printf("WebSocket closed for player %s (code: %d, reason: %s)", player.ID, code, text)
		// Immediately handle disconnection for hosts to allow new connections
		if player.IsHost {
			wsh.handleHostDisconnection(player)
		} else {
			wsh.handleDisconnection(player)
		}
		return nil
	})

	// Start ping ticker
	pingTicker := time.NewTicker(constants.WebSocketPingInterval)
	defer pingTicker.Stop()

	// Handle initial join
	if err := wsh.eventHandlers.HandlePlayerJoin(player, nil); err != nil {
		log.Printf("Error handling player join: %v", err)
		wsh.sendError(player, "Failed to join game")
		return
	}

	// Broadcast lobby status
	wsh.eventHandlers.broadcastLobbyStatus()

	// Message handling goroutine with enhanced error handling
	messageChan := make(chan error, 1)
	go func() {
		messageChan <- wsh.handlePlayerMessages(player)
	}()

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

		case err := <-messageChan:
			if err != nil {
				log.Printf("Message handling error for player %s: %v", player.ID, err)
			}
			wsh.handleDisconnection(player)
			return
		}
	}
}

// validateConnectionRequest validates the initial connection request - ENHANCED
func (wsh *WebSocketHandler) validateConnectionRequest(r *http.Request) bool {
	// Check request method
	if r.Method != "GET" {
		log.Printf("Invalid connection request method: %s", r.Method)
		return false
	}

	// Check required headers
	if r.Header.Get("Upgrade") != "websocket" {
		log.Printf("Missing or invalid Upgrade header: %s", r.Header.Get("Upgrade"))
		return false
	}

	if r.Header.Get("Connection") != "Upgrade" {
		log.Printf("Missing or invalid Connection header: %s", r.Header.Get("Connection"))
		return false
	}

	// Additional security: check user agent to prevent basic automated attacks
	userAgent := r.Header.Get("User-Agent")
	if userAgent == "" {
		log.Printf("Connection rejected: missing User-Agent header")
		return false
	}

	// Additional security: rate limiting could be added here
	// For now, we'll just log connection attempts
	log.Printf("Valid connection request from %s (User-Agent: %s)", r.RemoteAddr, userAgent)

	return true
}

// canAcceptNewPlayer checks if we can accept a new player connection - ENHANCED
func (wsh *WebSocketHandler) canAcceptNewPlayer() bool {
	// Check game phase - only allow new players during setup
	phase := wsh.gameManager.GetPhase()
	if phase != PhaseSetup {
		log.Printf("Rejected new player connection: game is in %s phase (only setup phase allows new players)", phase.String())
		return false
	}

	// Check player limit
	currentCount := wsh.playerManager.GetPlayerCount()
	if currentCount >= constants.MaxPlayers {
		log.Printf("Rejected new player connection: player limit reached (%d/%d)", currentCount, constants.MaxPlayers)
		return false
	}

	return true
}

// sendConnectionError sends an error during connection setup - ENHANCED
func (wsh *WebSocketHandler) sendConnectionError(conn *websocket.Conn, message string) {
	errorResponse := map[string]interface{}{
		"error":     message,
		"type":      "connection_error",
		"timestamp": time.Now().Unix(),
	}

	// Set write deadline for error message
	conn.SetWriteDeadline(time.Now().Add(5 * time.Second))

	err := conn.WriteJSON(BaseMessage{
		Type:    MsgError,
		Payload: mustMarshal(errorResponse),
	})

	if err != nil {
		log.Printf("Failed to send connection error message: %v", err)
	}

	log.Printf("Sent connection error: %s", message)
}

// handlePlayerMessages handles incoming messages with comprehensive validation
func (wsh *WebSocketHandler) handlePlayerMessages(player *Player) error {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in message handler for player %s: %v", player.ID, r)
		}
	}()

	for {
		var baseMsg BaseMessage
		err := player.Connection.ReadJSON(&baseMsg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error for player %s: %v", player.ID, err)
			}
			return err
		}

		// Update last seen
		player.mu.Lock()
		player.LastSeen = time.Now()
		player.mu.Unlock()

		// Validate message structure
		if err := wsh.validateBaseMessage(baseMsg); err != nil {
			wsh.sendValidationError(player, err)
			continue
		}

		// Route message based on type with validation
		switch baseMsg.Type {
		case MsgRoleSelection, MsgTriviaSpecialtySelection, MsgResourceLocationVerified,
			MsgTriviaAnswer, MsgSegmentCompleted, MsgFragmentMoveRequest,
			MsgPlayerReady, MsgHostStartGame, MsgHostStartPuzzle,
			MsgPieceRecommendationRequest, MsgPieceRecommendationResponse:

			// These messages require authentication and validation
			if err := wsh.handleAuthenticatedMessage(player, baseMsg); err != nil {
				wsh.sendValidationError(player, err)
				continue
			}

		default:
			wsh.sendError(player, fmt.Sprintf("Unknown message type: %s", baseMsg.Type))
		}
	}
}

// validateBaseMessage validates the basic message structure
func (wsh *WebSocketHandler) validateBaseMessage(msg BaseMessage) error {
	if msg.Type == "" {
		return fmt.Errorf("message type cannot be empty")
	}

	if len(msg.Type) > 50 {
		return fmt.Errorf("message type too long")
	}

	if len(msg.Payload) > 8192 { // 8KB limit
		return fmt.Errorf("message payload too large")
	}

	return nil
}

// handleAuthenticatedMessage handles messages that require authentication with validation
func (wsh *WebSocketHandler) handleAuthenticatedMessage(player *Player, baseMsg BaseMessage) error {
	// Validate and parse authentication wrapper
	authWrapper, validationErrors := validateAuthWrapper(baseMsg.Payload)
	if len(validationErrors) > 0 {
		return fmt.Errorf("authentication validation failed: %v", validationErrors[0])
	}

	// Verify authentication
	if authWrapper.Auth.PlayerID != player.ID {
		return fmt.Errorf("authentication failed: player ID mismatch")
	}

	// Route to appropriate handler with validation
	return wsh.routeValidatedMessage(player.ID, baseMsg.Type, authWrapper.Payload)
}

// routeValidatedMessage routes authenticated and validated messages to appropriate handlers
func (wsh *WebSocketHandler) routeValidatedMessage(playerID string, msgType string, payload json.RawMessage) error {
	switch msgType {
	case MsgRoleSelection:
		return wsh.handleRoleSelectionWithValidation(playerID, payload)

	case MsgTriviaSpecialtySelection:
		return wsh.handleSpecialtySelectionWithValidation(playerID, payload)

	case MsgPlayerReady:
		return wsh.handlePlayerReadyWithValidation(playerID, payload)

	case MsgHostStartGame:
		return wsh.handleHostStartGameWithValidation(playerID, payload)

	case MsgResourceLocationVerified:
		return wsh.handleLocationVerificationWithValidation(playerID, payload)

	case MsgTriviaAnswer:
		return wsh.handleTriviaAnswerWithValidation(playerID, payload)

	case MsgSegmentCompleted:
		return wsh.handleSegmentCompletionWithValidation(playerID, payload)

	case MsgFragmentMoveRequest:
		return wsh.handleFragmentMoveWithValidation(playerID, payload)

	case MsgHostStartPuzzle:
		return wsh.handleHostStartPuzzleWithValidation(playerID, payload)

	case MsgPieceRecommendationRequest:
		return wsh.handlePieceRecommendationRequestWithValidation(playerID, payload)

	case MsgPieceRecommendationResponse:
		return wsh.handlePieceRecommendationResponseWithValidation(playerID, payload)

	default:
		return fmt.Errorf("unhandled message type: %s", msgType)
	}
}

// Validated handler methods

func (wsh *WebSocketHandler) handleRoleSelectionWithValidation(playerID string, payload json.RawMessage) error {
	data, errors := ValidateRoleSelection(payload)
	if len(errors) > 0 {
		return fmt.Errorf("validation failed: %v", errors)
	}

	return wsh.eventHandlers.HandleRoleSelection(playerID, mustMarshal(data))
}

func (wsh *WebSocketHandler) handleSpecialtySelectionWithValidation(playerID string, payload json.RawMessage) error {
	data, errors := ValidateSpecialtySelection(payload)
	if len(errors) > 0 {
		return fmt.Errorf("validation failed: %v", errors)
	}

	return wsh.eventHandlers.HandleTriviaSpecialtySelection(playerID, mustMarshal(data))
}

func (wsh *WebSocketHandler) handlePlayerReadyWithValidation(playerID string, payload json.RawMessage) error {
	data, errors := ValidatePlayerReady(payload)
	if len(errors) > 0 {
		return fmt.Errorf("validation failed: %v", errors)
	}

	return wsh.eventHandlers.HandlePlayerReady(playerID, mustMarshal(data))
}

func (wsh *WebSocketHandler) handleHostStartGameWithValidation(playerID string, payload json.RawMessage) error {
	data, errors := ValidateEmptyPayload(payload)
	if len(errors) > 0 {
		return fmt.Errorf("validation failed: %v", errors)
	}

	return wsh.eventHandlers.HandleHostStartGame(playerID, mustMarshal(data))
}

func (wsh *WebSocketHandler) handleLocationVerificationWithValidation(playerID string, payload json.RawMessage) error {
	data, errors := ValidateLocationVerification(payload)
	if len(errors) > 0 {
		return fmt.Errorf("validation failed: %v", errors)
	}

	return wsh.eventHandlers.HandleResourceLocationVerified(playerID, mustMarshal(data))
}

func (wsh *WebSocketHandler) handleTriviaAnswerWithValidation(playerID string, payload json.RawMessage) error {
	data, errors := ValidateTriviaAnswer(payload)
	if len(errors) > 0 {
		return fmt.Errorf("validation failed: %v", errors)
	}

	return wsh.eventHandlers.HandleTriviaAnswer(playerID, mustMarshal(data))
}

func (wsh *WebSocketHandler) handleSegmentCompletionWithValidation(playerID string, payload json.RawMessage) error {
	data, errors := ValidateSegmentCompletion(payload)
	if len(errors) > 0 {
		return fmt.Errorf("validation failed: %v", errors)
	}

	return wsh.eventHandlers.HandleSegmentCompleted(playerID, mustMarshal(data))
}

// handleFragmentMoveWithValidation handles fragment moves with ENHANCED ownership validation
func (wsh *WebSocketHandler) handleFragmentMoveWithValidation(playerID string, payload json.RawMessage) error {
	// Get current grid size for validation
	maxGridSize := 8 // Default max, will be updated by game manager
	if wsh.gameManager.state != nil {
		wsh.gameManager.mu.RLock()
		maxGridSize = wsh.gameManager.state.GridSize
		wsh.gameManager.mu.RUnlock()
	}

	data, errors := ValidateFragmentMove(payload, maxGridSize)
	if len(errors) > 0 {
		return fmt.Errorf("validation failed: %v", errors)
	}

	// ENHANCED: Additional ownership pre-validation before passing to game manager
	wsh.gameManager.mu.RLock()
	fragment, exists := wsh.gameManager.state.PuzzleFragments[data["fragmentId"].(string)]
	wsh.gameManager.mu.RUnlock()

	if exists {
		// Pre-validate ownership to provide more specific error messages
		if err := wsh.gameManager.validateFragmentOwnership(playerID, fragment); err != nil {
			// Send specific ownership error response
			player, _ := wsh.playerManager.GetPlayer(playerID)
			if player != nil {
				sendToPlayer(player, MsgFragmentMoveResponse, map[string]interface{}{
					"status":     "denied",
					"reason":     err.Error(),
					"fragmentId": data["fragmentId"].(string),
					"errorType":  "ownership_violation",
				})
			}

			log.Printf("Fragment move denied for player %s: %v", playerID, err)
			return nil // Don't return error to avoid double error messages
		}
	}

	return wsh.eventHandlers.HandleFragmentMoveRequest(playerID, mustMarshal(data))
}

func (wsh *WebSocketHandler) handleHostStartPuzzleWithValidation(playerID string, payload json.RawMessage) error {
	data, errors := ValidateEmptyPayload(payload)
	if len(errors) > 0 {
		return fmt.Errorf("validation failed: %v", errors)
	}

	return wsh.eventHandlers.HandleHostStartPuzzle(playerID, mustMarshal(data))
}

func (wsh *WebSocketHandler) handlePieceRecommendationRequestWithValidation(playerID string, payload json.RawMessage) error {
	// Get current grid size for validation
	maxGridSize := 8 // Default max
	if wsh.gameManager.state != nil {
		wsh.gameManager.mu.RLock()
		maxGridSize = wsh.gameManager.state.GridSize
		wsh.gameManager.mu.RUnlock()
	}

	data, errors := ValidatePieceRecommendationRequest(payload, maxGridSize)
	if len(errors) > 0 {
		return fmt.Errorf("validation failed: %v", errors)
	}

	return wsh.eventHandlers.HandlePieceRecommendationRequest(playerID, mustMarshal(data))
}

func (wsh *WebSocketHandler) handlePieceRecommendationResponseWithValidation(playerID string, payload json.RawMessage) error {
	data, errors := ValidatePieceRecommendationResponse(payload)
	if len(errors) > 0 {
		return fmt.Errorf("validation failed: %v", errors)
	}

	return wsh.eventHandlers.HandlePieceRecommendationResponse(playerID, mustMarshal(data))
}

// sendValidationError sends a detailed validation error to the player
func (wsh *WebSocketHandler) sendValidationError(player *Player, err error) {
	log.Printf("Validation error for player %s: %v", player.ID, err)

	errorResponse := map[string]interface{}{
		"error":   "Validation failed",
		"details": err.Error(),
		"type":    "validation_error",
	}

	sendToPlayer(player, MsgError, errorResponse)
}

// handleHostDisconnection handles immediate host disconnection cleanup
func (wsh *WebSocketHandler) handleHostDisconnection(player *Player) {
	log.Printf("Host %s disconnected - immediately cleaning up for new host connection", player.ID)

	// Mark as disconnected
	wsh.playerManager.DisconnectPlayer(player.ID)

	// Immediately remove the host to allow new connections
	if removed := wsh.playerManager.RemoveDisconnectedHost(); !removed {
		log.Printf("Warning: Attempted to remove disconnected host but none found")
	}

	// Get current phase for notification
	phase := wsh.gameManager.GetPhase()

	// Notify players that host disconnected
	wsh.broadcastChan <- BroadcastMessage{
		Type: MsgError,
		Payload: map[string]interface{}{
			"error":            "Host disconnected - new host can now connect",
			"type":             "host_disconnected",
			"phase":            phase.String(),
			"reconnectionInfo": "A new host can connect immediately",
		},
	}

	log.Printf("Host %s removed - server is now available for new host connection", player.ID)
}

// handleDisconnection handles player disconnection with ENHANCED fragment ownership handling
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
		log.Printf("Player %s disconnected during resource gathering - can reconnect", player.ID)

	case PhasePuzzleAssembly:
		// ENHANCED: During puzzle assembly, handle fragment ownership transfer
		if !player.IsHost {
			wsh.gameManager.handleFragmentDisconnection(player.ID)
		}

		// Notify others about disconnection and fragment changes
		wsh.broadcastChan <- BroadcastMessage{
			Type: MsgCentralPuzzleState,
			Payload: map[string]interface{}{
				"playerDisconnected":  player.ID,
				"phase":               "puzzle_assembly",
				"reconnectionAllowed": false,
			},
		}

		log.Printf("Player %s disconnected during puzzle assembly - fragment converted to unassigned", player.ID)

	case PhasePostGame:
		// No special handling needed
		log.Printf("Player %s disconnected during post-game phase", player.ID)
	}

	// Handle host disconnection (fallback for non-close events)
	if player.IsHost {
		log.Printf("Host %s disconnected via non-close event - using immediate cleanup", player.ID)
		wsh.handleHostDisconnection(player)
		return
	}
}

// sendReconnectionState sends current game state to reconnecting player - ENHANCED
func (wsh *WebSocketHandler) sendReconnectionState(player *Player) {
	phase := wsh.gameManager.GetPhase()

	player.mu.RLock()
	isHost := player.IsHost
	player.mu.RUnlock()

	if isHost {
		// Host gets comprehensive state information
		sendToPlayer(player, MsgAvailableRoles, map[string]interface{}{
			"playerId": player.ID,
			"isHost":   true,
			"message":  fmt.Sprintf("Reconnected as host during %s phase", phase.String()),
		})
	} else {
		// Regular player gets role information
		roles := wsh.playerManager.GetAvailableRoles()
		sendToPlayer(player, MsgAvailableRoles, map[string]interface{}{
			"playerId":         player.ID,
			"isHost":           false,
			"roles":            roles,
			"triviaCategories": constants.TriviaCategories,
		})
	}

	switch phase {
	case PhaseSetup:
		// Send lobby status
		wsh.eventHandlers.broadcastLobbyStatus()

	case PhaseResourceGathering:
		if !isHost {
			// Send resource phase info to regular players
			sendToPlayer(player, MsgResourcePhaseStart, map[string]interface{}{
				"resourceHashes": constants.ResourceStationHashes,
			})

			// Send current progress
			wsh.gameManager.sendTeamProgressUpdate()
		}

	case PhasePuzzleAssembly:
		if isHost {
			// Host gets complete puzzle state for monitoring
			wsh.gameManager.sendCompletePuzzleStateToHost()
		} else {
			// NOTE: Regular players cannot reconnect during puzzle assembly
			// This case should not occur due to connection restrictions
			log.Printf("WARNING: Regular player %s reconnected during puzzle assembly - this should not happen", player.ID)
		}

	case PhasePostGame:
		// Send current analytics if available
		log.Printf("Player %s reconnected during post-game phase", player.ID)
	}

	log.Printf("Sent reconnection state to %s (host: %v) for phase %s", player.ID, isHost, phase.String())
}

// sendError sends an error message to a player
func (wsh *WebSocketHandler) sendError(player *Player, message string) {
	errorResponse := map[string]interface{}{
		"error": message,
		"type":  "general_error",
	}
	sendToPlayer(player, MsgError, errorResponse)
}

// StartBroadcaster starts the message broadcaster goroutine with enhanced error handling
func (wsh *WebSocketHandler) StartBroadcaster() {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Recovered from panic in broadcaster: %v", r)
			}
		}()

		for msg := range wsh.broadcastChan {
			players := wsh.playerManager.GetAllPlayers()

			successCount := 0
			failureCount := 0

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
						failureCount++
					} else {
						successCount++
					}
				}
			}

			// Log broadcast statistics for monitoring
			if failureCount > 0 {
				log.Printf("Broadcast complete: %d successful, %d failed", successCount, failureCount)
			}
		}
	}()
}
