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
	// Block ALL player connections during puzzle phase before WebSocket upgrade
	gameManager := services.GetGameInstance()
	game := gameManager.GetGame()

	if game.CurrentPhase == "puzzle_assembly" {
		// Return HTTP 403 for any player connection during puzzle phase
		// No distinction between new players and reconnecting players
		http.Error(w, "Player connections not allowed during puzzle assembly phase", http.StatusForbidden)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade player connection: %v", err)
		return
	}
	defer conn.Close()

	// Check for reconnection token in query parameters
	var playerID string
	token := r.URL.Query().Get("token")
	if token != "" {
		// Validate token format (should be UUID)
		if utils.IsValidUUID(token) {
			playerID = token
		} else {
			log.Printf("Invalid token format provided: %s", token)
			sendConnectionError(conn, "Invalid token format")
			return
		}
	} else {
		// Generate new player ID for new connections
		playerID = utils.GeneratePlayerID()
	}

	// Create player instance
	player := models.NewPlayer(playerID, conn)

	// Add player to game
	isReconnection, err := gameManager.AddPlayer(player)
	if err != nil {
		log.Printf("Failed to add player: %v", err)
		sendConnectionError(conn, err.Error())
		return
	}

	// Send initial roles available message
	sendRolesAvailable(player, isReconnection)

	// Send phase-specific restoration events if this is a reconnection
	if isReconnection {
		sendPlayerPhaseRestoration(player)
	}

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
	isReconnection, err := gameManager.SetHost(host)
	if err != nil {
		log.Printf("Failed to set host: %v", err)
		sendConnectionError(conn, err.Error())
		return
	}

	// Send connection confirmed message
	sendHostConnectionConfirmed(host, isReconnection)

	// Send phase-specific restoration events if this is a reconnection
	if isReconnection {
		sendHostPhaseRestoration(host)
	}

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
func sendHostConnectionConfirmed(host *models.Host, isReconnection bool) {
	gameManager := services.GetGameInstance()
	game := gameManager.GetGame()

	payload := map[string]interface{}{
		"playerId":       host.ID,
		"message":        "Connected as game host",
		"currentPhase":   string(game.CurrentPhase),
		"isReconnection": isReconnection,
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
func sendRolesAvailable(player *models.Player, isReconnection bool) {
	gameManager := services.GetGameInstance()
	game := gameManager.GetGame()
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

	// Get existing configuration for reconnecting players
	var existingConfiguration interface{}
	if isReconnection {
		// Get player's current role and specialty
		if player.Role != "" && len(player.Specialties) > 0 {
			existingConfiguration = map[string]interface{}{
				"selectedRole":        string(player.Role),
				"selectedSpecialties": player.Specialties,
				"playerName":          player.Name,
			}
		}
	}

	payload := map[string]interface{}{
		"playerId":              player.ID,
		"currentPhase":          string(game.CurrentPhase),
		"isReconnection":        isReconnection,
		"existingConfiguration": existingConfiguration,
		"roles":                 roles,
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

// sendPlayerPhaseRestoration sends phase-specific restoration events to reconnecting player
func sendPlayerPhaseRestoration(player *models.Player) {
	gameManager := services.GetGameInstance()
	broadcastService := gameManager.GetBroadcastService()
	game := gameManager.GetGame()

	if broadcastService == nil {
		return
	}

	switch game.CurrentPhase {
	case "setup":
		// If player was already configured and ready, mark them as ready again
		if player.Role != "" && len(player.Specialties) > 0 && player.Name != "" {
			player.IsReady = true
			// Broadcast updated lobby status since a configured player is back
			broadcastService.BroadcastLobbyStatus()
		}

	case "resource_gathering":
		// Send RESOURCE_TO_CLIENT_PHASE_START to player using existing infrastructure
		resourcePayload := map[string]interface{}{
			"currentRound":    game.CurrentRound,
			"totalRounds":     constants.ResourceGatheringRounds,
			"roundDuration":   constants.ResourceGatheringRoundDuration,
			"playerStation":   player.CurrentStation,
			"tokenMultiplier": constants.RoleResourceMultiplier,
		}
		broadcastService.SendToPlayer(player, constants.EventResourceToClientPhaseStart, resourcePayload)

		// Send RESOURCE_TO_CLIENT_TEAM_PROGRESS using existing getTeamProgressPayload
		broadcastService.SendToPlayer(player, constants.EventResourceToClientTeamProgress, getTeamProgressPayload())

		// If there's an active trivia question for the current round, send it to the reconnecting player
		triviaService := gameManager.GetTriviaService()
		if triviaService != nil {
			sendCurrentTriviaQuestionIfActive(player, triviaService, broadcastService)
		}

	case "analytics":
		// Get analytics from the analytics service
		analyticsService := gameManager.GetAnalyticsService()
		if analyticsService != nil {
			analytics := analyticsService.GetFullAnalytics()
			if analytics != nil {
				// Send personal report using existing analytics data
				if playerAnalytics, exists := analytics.PlayerAnalytics[player.ID]; exists {
					broadcastService.SendToPlayer(player, constants.EventAnalyticsToPlayerPersonalReport, playerAnalytics)
				}

				// Send team summary using existing analytics data
				teamSummary := map[string]interface{}{
					"gameSuccess":      analytics.TeamAnalytics.GameSuccess,
					"totalPlayers":     analytics.TeamAnalytics.TotalPlayers,
					"totalScore":       analytics.TeamAnalytics.TotalScore,
					"gameTime":         analytics.TeamAnalytics.GameTime,
					"overallAccuracy":  analytics.TeamAnalytics.OverallAccuracy,
					"completionTime":   analytics.TeamAnalytics.PuzzleCompletionTime,
					"teamAchievements": analytics.TeamAnalytics.TeamAchievements,
					"fastestAnswerer":  analytics.TeamAnalytics.FastestAnswerer,
					"mostTokens":       analytics.TeamAnalytics.MostTokens,
					"bestCollaborator": analytics.TeamAnalytics.BestCollaborator,
					"puzzleMVP":        analytics.TeamAnalytics.PuzzleMVP,
					"recommendations":  analytics.RecommendationsForImprovement,
				}
				broadcastService.SendToPlayer(player, constants.EventAnalyticsToClientTeamSummary, teamSummary)
			}
		}

	default:
		// Puzzle phase should already be blocked, but just in case
		log.Printf("Player %s attempted reconnection during unsupported phase: %s", player.ID, game.CurrentPhase)
	}
}

// sendHostPhaseRestoration sends phase-specific restoration events to reconnecting host
func sendHostPhaseRestoration(host *models.Host) {
	gameManager := services.GetGameInstance()
	broadcastService := gameManager.GetBroadcastService()
	game := gameManager.GetGame()

	if broadcastService == nil {
		return
	}

	switch game.CurrentPhase {
	case "setup":
		// Send current player roster
		broadcastService.BroadcastLobbyStatus()

	case "resource_gathering":
		// Send RESOURCE_TO_HOST_PHASE_START using existing resource phase infrastructure
		hostResourcePayload := map[string]interface{}{
			"currentRound":  game.CurrentRound,
			"totalRounds":   constants.ResourceGatheringRounds,
			"roundDuration": constants.ResourceGatheringRoundDuration,
			"gamePhase":     string(game.CurrentPhase),
		}
		broadcastService.SendToHost(host, constants.EventResourceToHostPhaseStart, hostResourcePayload)

		// Round analytics are sent by the existing round management system
		// No need to send placeholder analytics during reconnection

	case "puzzle_assembly":
		// Send PUZZLE_TO_HOST_PHASE_LOAD using existing puzzle infrastructure
		phaseLoadPayload := map[string]interface{}{
			"puzzleImageUrl": "/api/puzzle/image",
			"gridSize":       game.GetGridSize(),
			"totalFragments": game.GetGridSize() * game.GetGridSize(),
			"timeLimit":      constants.PuzzleBaseTime,
		}
		broadcastService.SendToHost(host, constants.EventPuzzleToHostPhaseLoad, phaseLoadPayload)

		// Send current grid state using existing infrastructure
		if game.PuzzleGrid != nil {
			// Use the existing broadcastGridState functionality but send only to host
			fragments := []map[string]interface{}{}
			for _, fragment := range game.PuzzleGrid.Fragments {
				fragments = append(fragments, map[string]interface{}{
					"fragmentId":      fragment.ID,
					"segmentId":       fragment.SegmentID,
					"playerId":        fragment.PlayerID,
					"position":        fragment.Position,
					"correctPosition": fragment.CorrectPosition,
					"visible":         fragment.Visible,
					"lastMoved":       fragment.LastMoved,
					"moveCount":       fragment.MoveCount,
					"isCorrect":       fragment.IsCorrect(),
				})
			}

			gridPayload := map[string]interface{}{
				"gridSize":           game.PuzzleGrid.Size,
				"fragments":          fragments,
				"completedFragments": countCorrectFragments(game.PuzzleGrid),
				"totalFragments":     len(game.PuzzleGrid.Fragments),
				"timeRemaining":      game.GetPuzzleTimeRemaining(),
				"updateType":         "reconnection",
			}
			broadcastService.SendToHost(host, constants.EventPuzzleToHostGridState, gridPayload)
		}

		// Timer events are handled by the existing timer system when active
		// No need to send placeholder timer events during reconnection

	case "analytics":
		// Get complete analytics from analytics service
		analyticsService := gameManager.GetAnalyticsService()
		if analyticsService != nil {
			analytics := analyticsService.GetFullAnalytics()
			if analytics != nil {
				// Send complete report using existing analytics data
				hostReport := map[string]interface{}{
					"gameAnalytics":    analytics,
					"gameCompleted":    analytics.TeamAnalytics.GameSuccess,
					"totalPlayers":     analytics.TeamAnalytics.TotalPlayers,
					"gameTime":         analytics.TeamAnalytics.GameTime,
					"completionTime":   analytics.TeamAnalytics.PuzzleCompletionTime,
					"teamPerformance":  analytics.TeamAnalytics,
					"playerStatistics": analytics.PlayerAnalytics,
					"categoryStats":    analytics.CategoryPerformance,
					"resourceMetrics":  analytics.ResourceGatheringMetrics,
					"puzzleMetrics":    analytics.PuzzleAssemblyMetrics,
					"recommendations":  analytics.RecommendationsForImprovement,
				}
				broadcastService.SendToHost(host, constants.EventAnalyticsToHostCompleteReport, hostReport)
			}
		}

	default:
		log.Printf("Host %s reconnected during unknown phase: %s", host.ID, game.CurrentPhase)
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

// countCorrectFragments counts fragments in their correct positions
func countCorrectFragments(grid *models.PuzzleGrid) int {
	count := 0
	for _, fragment := range grid.Fragments {
		if fragment.IsCorrect() {
			count++
		}
	}
	return count
}

// sendCurrentTriviaQuestionIfActive sends current trivia question to reconnecting player if mid-round
func sendCurrentTriviaQuestionIfActive(player *models.Player, triviaService *services.TriviaService, broadcastService *services.BroadcastService) {
	gameManager := services.GetGameInstance()
	game := gameManager.GetGame()

	// For now, we'll generate a fresh question for the reconnecting player
	// In a more sophisticated implementation, we would track the current active questions
	// and send the exact same question that other players are currently answering

	// Create a map with just this player to get a question
	players := map[string]*models.Player{
		player.ID: player,
	}

	// Get a question for the player
	questions := triviaService.GetQuestionsForRound(players)
	if question, exists := questions[player.ID]; exists && question != nil {
		// Calculate remaining time for this round
		// This is a simplified approach - in a real implementation you'd track round start time
		timeRemaining := constants.TriviaAnswerTime

		questionPayload := map[string]interface{}{
			"questionId":     question.ID,
			"questionText":   question.Question,
			"category":       question.Category,
			"difficulty":     question.Difficulty,
			"isSpecialty":    question.IsSpecialty,
			"specialtyBonus": question.SpecialtyBonus,
			"timeLimit":      timeRemaining,
			"options":        question.Options,
			"roundNumber":    game.CurrentRound,
			"totalRounds":    constants.ResourceGatheringRounds,
			"answerDeadline": time.Now().Add(time.Duration(timeRemaining) * time.Second).Format(time.RFC3339),
		}

		broadcastService.SendToPlayer(player, constants.EventResourceToPlayerTriviaQuestion, questionPayload)
		log.Printf("Sent current trivia question to reconnecting player %s", player.ID)
	}
}
