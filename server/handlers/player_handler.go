package handlers

import (
	"canvas-conundrum/config"
	"canvas-conundrum/models"
	"canvas-conundrum/services"
	"canvas-conundrum/utils"
	"encoding/json"
	"log"
	"time"
)

// HandlePlayerMessage handles incoming messages from players
func HandlePlayerMessage(player *models.Player, msg *utils.Message) {
	gameManager := services.GetGameInstance()
	broadcastService := gameManager.GetBroadcastService()

	log.Printf("Player %s sent event: %s", player.ID, msg.Event)

	switch msg.Event {
	case config.EventSetupToServerPlayerConfiguration:
		handlePlayerConfiguration(player, msg.Payload)

	case config.EventResourceToServerLocationVerified:
		handleLocationVerified(player, msg.Payload)

	case config.EventResourceToServerTriviaAnswer:
		handleTriviaAnswer(player, msg.Payload)

	case config.EventPuzzleToServerSegmentCompleted:
		handleSegmentCompleted(player, msg.Payload)

	case config.EventPuzzleToServerFragmentMove:
		handleFragmentMove(player, msg.Payload)

	case config.EventPuzzleToServerRecommendMove:
		handleRecommendMove(player, msg.Payload)

	case config.EventPuzzleToServerRecommendationResponse:
		handleRecommendationResponse(player, msg.Payload)

	case config.EventSystemPing:
		handlePing(player, msg.Payload)

	default:
		log.Printf("Unknown event from player %s: %s", player.ID, msg.Event)
		if broadcastService != nil {
			broadcastService.SendError(
				player,
				"UNKNOWN_EVENT",
				"Unknown event type",
				"Event "+msg.Event+" is not recognized",
			)
		}
	}
}

// handlePlayerConfiguration handles player role and specialty selection
func handlePlayerConfiguration(player *models.Player, payload json.RawMessage) {
	var data struct {
		SelectedRole        string   `json:"selectedRole"`
		SelectedSpecialties []string `json:"selectedSpecialties"`
		PlayerName          string   `json:"playerName"`
	}

	if err := json.Unmarshal(payload, &data); err != nil {
		log.Printf("Failed to parse player configuration: %v", err)
		return
	}

	gameManager := services.GetGameInstance()
	broadcastService := gameManager.GetBroadcastService()

	// Validate and sanitize input
	data.PlayerName = utils.SanitizeString(data.PlayerName)
	if err := utils.ValidatePlayerName(data.PlayerName); err != nil {
		log.Printf("Invalid player name: %v", err)
		if broadcastService != nil {
			broadcastService.SendError(player, "INVALID_NAME", "Invalid player name", err.Error())
		}
		return
	}

	if err := utils.ValidateRole(data.SelectedRole); err != nil {
		log.Printf("Invalid role: %v", err)
		if broadcastService != nil {
			broadcastService.SendError(player, "INVALID_ROLE", "Invalid role", err.Error())
		}
		return
	}

	if err := utils.ValidateSpecialties(data.SelectedSpecialties); err != nil {
		log.Printf("Invalid specialties: %v", err)
		if broadcastService != nil {
			broadcastService.SendError(player, "INVALID_SPECIALTIES", "Invalid specialties", err.Error())
		}
		return
	}

	// Convert string to role
	role := models.Role(data.SelectedRole)

	// Update player configuration
	err := gameManager.UpdatePlayerConfiguration(player.ID, data.PlayerName, role, data.SelectedSpecialties)
	if err != nil {
		log.Printf("Failed to update player configuration: %v", err)
		if broadcastService != nil {
			broadcastService.SendError(
				player,
				config.ErrorCodeInvalidRole,
				config.ErrorMessageInvalidRole,
				err.Error(),
			)
		}
		return
	}

	// Broadcast updated lobby status
	if broadcastService != nil {
		broadcastService.BroadcastLobbyStatus()
	}
}

// handleLocationVerified handles QR code scan verification
func handleLocationVerified(player *models.Player, payload json.RawMessage) {
	var data struct {
		StationHash      string `json:"stationHash"`
		PreviousLocation string `json:"previousLocation"`
		ScanTimestamp    string `json:"scanTimestamp"`
	}

	if err := json.Unmarshal(payload, &data); err != nil {
		log.Printf("Failed to parse location verification: %v", err)
		return
	}

	gameManager := services.GetGameInstance()
	broadcastService := gameManager.GetBroadcastService()
	analyticsService := gameManager.GetAnalyticsService()

	// Verify station hash
	station := config.GetStationFromHash(data.StationHash)
	if station == config.UnknownStation {
		log.Printf("Invalid station hash from player %s: %s", player.ID, data.StationHash)
		if broadcastService != nil {
			broadcastService.SendError(
				player,
				"INVALID_STATION",
				"Invalid QR code",
				"The scanned QR code is not recognized",
			)
		}
		return
	}

	// Update player location
	player.CurrentStation = string(station)
	player.LastSeen = time.Now()

	// Record station visit in analytics
	if analyticsService != nil {
		analyticsService.RecordStationVisit(player.ID, string(station))
	}

	log.Printf("Player %s moved to station %s", player.ID, station)
}

// handleTriviaAnswer handles trivia answer submission
func handleTriviaAnswer(player *models.Player, payload json.RawMessage) {
	var data struct {
		QuestionID      string  `json:"questionId"`
		SelectedAnswer  string  `json:"selectedAnswer"`
		AnswerIndex     int     `json:"answerIndex"`
		TimeElapsed     float64 `json:"timeElapsed"`
		CurrentLocation string  `json:"currentLocation"`
	}

	if err := json.Unmarshal(payload, &data); err != nil {
		log.Printf("Failed to parse trivia answer: %v", err)
		return
	}

	gameManager := services.GetGameInstance()
	broadcastService := gameManager.GetBroadcastService()
	triviaService := gameManager.GetTriviaService()
	analyticsService := gameManager.GetAnalyticsService()
	game := gameManager.GetGame()

	// Process the answer using trivia service
	answer, tokensEarned, tokenType := triviaService.ProcessAnswer(
		player.ID,
		data.QuestionID,
		data.SelectedAnswer,
		data.AnswerIndex,
		data.TimeElapsed,
	)

	if answer == nil {
		log.Printf("Failed to process answer for question: %s", data.QuestionID)
		return
	}

	// Get the original question for additional context
	question := triviaService.GetQuestionByID(data.QuestionID)
	if question == nil {
		log.Printf("Question not found: %s", data.QuestionID)
		return
	}

	// Update player stats
	if answer.Correct {
		player.CorrectAnswers++

		// Update team tokens
		if tokenType != "" {
			player.TokensEarned += tokensEarned
			game.TeamTokens.AddTokens(tokenType, tokensEarned)
		}
	}
	player.QuestionsAnswered++

	// Record in analytics
	if analyticsService != nil {
		analyticsService.RecordTriviaAnswer(
			player.ID,
			question.Category,
			answer.Correct,
			data.TimeElapsed,
			tokensEarned,
			question.IsSpecialty,
		)

		if tokenType != "" && tokensEarned > 0 {
			analyticsService.RecordTokenCollection(player.ID, tokenType, tokensEarned)
		}
	}

	// Send answer result to player
	if broadcastService != nil {
		resultPayload := map[string]interface{}{
			"questionId":     data.QuestionID,
			"correct":        answer.Correct,
			"selectedAnswer": data.SelectedAnswer,
			"correctAnswer":  question.CorrectAnswer,
			"tokensEarned":   tokensEarned,
			"baseTokens":     config.BaseTokensPerCorrectAnswer,
			"bonuses": map[string]interface{}{
				"roleBonus":            player.Role.GetBonusTokenType() == tokenType,
				"roleBonusTokens":      0, // Calculate if needed
				"specialtyBonus":       question.IsSpecialty,
				"specialtyBonusTokens": 0, // Calculate if needed
				"difficultyMultiplier": 1.0,
			},
			"currentLocation": player.CurrentStation,
		}

		broadcastService.SendToPlayer(player, config.EventResourceToPlayerAnswerResult, resultPayload)

		// Broadcast team progress update
		broadcastService.BroadcastToAllPlayers(config.EventResourceToClientTeamProgress, getTeamProgressPayload())
	}
}

// handleSegmentCompleted handles individual puzzle segment completion
func handleSegmentCompleted(player *models.Player, payload json.RawMessage) {
	var data struct {
		SegmentID           string  `json:"segmentId"`
		CompletionTimestamp int64   `json:"completionTimestamp"`
		SolveTime           float64 `json:"solveTime"`
		ManualPiecesSolved  int     `json:"manualPiecesSolved"`
		PreSolvedPieces     int     `json:"preSolvedPieces"`
	}

	if err := json.Unmarshal(payload, &data); err != nil {
		log.Printf("Failed to parse segment completion: %v", err)
		return
	}

	gameManager := services.GetGameInstance()
	broadcastService := gameManager.GetBroadcastService()
	analyticsService := gameManager.GetAnalyticsService()

	// Complete the segment
	err := gameManager.CompleteSegment(player.ID, data.SegmentID)
	if err != nil {
		log.Printf("Failed to complete segment: %v", err)
		if broadcastService != nil {
			broadcastService.SendError(
				player,
				"SEGMENT_COMPLETION_ERROR",
				"Failed to complete segment",
				err.Error(),
			)
		}
		return
	}

	// Record in analytics
	if analyticsService != nil {
		analyticsService.RecordSegmentCompletion(player.ID, data.SolveTime)
	}
}

// handleFragmentMove handles fragment movement requests
func handleFragmentMove(player *models.Player, payload json.RawMessage) {
	var data struct {
		FragmentID         string          `json:"fragmentId"`
		CurrentPosition    models.Position `json:"currentPosition"`
		TargetPosition     models.Position `json:"targetPosition"`
		SwapWithFragmentID string          `json:"swapWithFragmentId"`
	}

	if err := json.Unmarshal(payload, &data); err != nil {
		log.Printf("Failed to parse fragment move: %v", err)
		return
	}

	gameManager := services.GetGameInstance()
	broadcastService := gameManager.GetBroadcastService()
	analyticsService := gameManager.GetAnalyticsService()

	// Execute the move
	err := gameManager.MoveFragment(player.ID, data.FragmentID, data.TargetPosition)
	if err != nil {
		log.Printf("Failed to move fragment: %v", err)
		if broadcastService != nil {
			moveResult := map[string]interface{}{
				"moveId": utils.GenerateMoveID(),
				"status": "failed",
				"error":  err.Error(),
			}
			broadcastService.SendToPlayer(player, config.EventPuzzleToPlayerMoveResult, moveResult)
		}
		return
	}

	// Record move in analytics
	if analyticsService != nil {
		analyticsService.RecordFragmentMove(player.ID, true)
	}

	// Send success result
	if broadcastService != nil {
		moveResult := map[string]interface{}{
			"moveId":      utils.GenerateMoveID(),
			"status":      "success",
			"fragmentId":  data.FragmentID,
			"newPosition": data.TargetPosition,
			"cooldownInfo": map[string]interface{}{
				"nextMoveAvailable": time.Now().Add(time.Duration(config.FragmentMoveCooldown) * time.Millisecond).Unix(),
				"cooldownRemaining": config.FragmentMoveCooldown / 1000.0,
			},
		}

		if data.SwapWithFragmentID != "" {
			moveResult["swappedFragmentId"] = data.SwapWithFragmentID
			moveResult["swappedFragmentNewPosition"] = data.CurrentPosition
		}

		broadcastService.SendToPlayer(player, config.EventPuzzleToPlayerMoveResult, moveResult)
	}
}

// handleRecommendMove handles movement recommendations
func handleRecommendMove(player *models.Player, payload json.RawMessage) {
	var data struct {
		TargetPlayerID string `json:"targetPlayerId"`
		FromFragmentID string `json:"fromFragmentId"`
		ToFragmentID   string `json:"toFragmentId"`
		Reasoning      string `json:"reasoning"`
	}

	if err := json.Unmarshal(payload, &data); err != nil {
		log.Printf("Failed to parse recommendation: %v", err)
		return
	}

	gameManager := services.GetGameInstance()
	broadcastService := gameManager.GetBroadcastService()
	puzzleService := gameManager.GetPuzzleService()
	analyticsService := gameManager.GetAnalyticsService()

	// Get target player
	targetPlayer, exists := gameManager.GetPlayer(data.TargetPlayerID)
	if !exists || !targetPlayer.IsActive {
		log.Printf("Target player not found or disconnected: %s", data.TargetPlayerID)
		return
	}

	// Create recommendation
	rec, err := puzzleService.CreateRecommendation(
		player.ID,
		player.Name,
		data.TargetPlayerID,
		data.FromFragmentID,
		data.ToFragmentID,
		data.Reasoning,
	)
	if err != nil {
		log.Printf("Failed to create recommendation: %v", err)
		return
	}

	// Record in analytics
	if analyticsService != nil {
		analyticsService.RecordRecommendation(player.ID, data.TargetPlayerID, false)
	}

	// Send recommendation to target player
	if broadcastService != nil {
		recPayload := map[string]interface{}{
			"moveId":         rec.ID,
			"fromPlayerId":   player.ID,
			"fromPlayerName": player.Name,
			"toPlayerId":     data.TargetPlayerID,
			"fromFragmentId": data.FromFragmentID,
			"toFragmentId":   data.ToFragmentID,
			"reasoning":      data.Reasoning,
			"expiresAt":      rec.ExpiresAt.Format(time.RFC3339),
		}

		broadcastService.SendToPlayer(targetPlayer, config.EventPuzzleToPlayerMoveRecommendation, recPayload)
	}
}

// handleRecommendationResponse handles responses to recommendations
func handleRecommendationResponse(player *models.Player, payload json.RawMessage) {
	var data struct {
		MoveID         string `json:"moveId"`
		Response       string `json:"response"` // "accept" or "reject"
		ResponseReason string `json:"responseReason"`
	}

	if err := json.Unmarshal(payload, &data); err != nil {
		log.Printf("Failed to parse recommendation response: %v", err)
		return
	}

	gameManager := services.GetGameInstance()
	broadcastService := gameManager.GetBroadcastService()
	puzzleService := gameManager.GetPuzzleService()
	analyticsService := gameManager.GetAnalyticsService()

	// Get recommendation
	rec, exists := puzzleService.GetRecommendation(data.MoveID)
	if !exists {
		log.Printf("Recommendation not found: %s", data.MoveID)
		return
	}

	// Update recommendation status
	puzzleService.UpdateRecommendationStatus(data.MoveID, data.Response+"ed")

	// If accepted, execute the swap
	if data.Response == "accept" {
		game := gameManager.GetGame()
		if game.PuzzleGrid != nil {
			err := puzzleService.ExecuteRecommendedSwap(game.PuzzleGrid, data.MoveID)
			if err != nil {
				log.Printf("Failed to execute recommended swap: %v", err)
			}
		}

		// Record acceptance in analytics
		if analyticsService != nil {
			analyticsService.RecordRecommendation(rec.FromPlayerID, player.ID, true)
		}
	}

	// Send result to original recommender
	if broadcastService != nil {
		fromPlayer, exists := gameManager.GetPlayer(rec.FromPlayerID)
		if exists && fromPlayer.IsActive {
			resultPayload := map[string]interface{}{
				"moveId":           data.MoveID,
				"targetPlayerId":   player.ID,
				"targetPlayerName": player.Name,
				"response":         data.Response,
				"responseReason":   data.ResponseReason,
				"executionStatus":  data.Response + "ed",
			}

			broadcastService.SendToPlayer(fromPlayer, config.EventPuzzleToPlayerRecommendationResult, resultPayload)
		}
	}
}

// handlePing handles ping messages from players
func handlePing(player *models.Player, payload json.RawMessage) {
	var data struct {
		ClientTimestamp   string `json:"clientTimestamp"`
		SequenceNumber    int    `json:"sequenceNumber"`
		ConnectionQuality struct {
			Latency          int `json:"latency"`
			MessagesReceived int `json:"messagesReceived"`
			MessagesSent     int `json:"messagesSent"`
		} `json:"connectionQuality"`
	}

	if err := json.Unmarshal(payload, &data); err != nil {
		log.Printf("Failed to parse ping: %v", err)
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
				"activeConnections": gameManager.GetPlayerCount(), // Only count players, not host
				"serverLoad":        0.15,                         // Placeholder
				"gamePhase":         gameManager.GetCurrentPhase(),
			},
		}

		broadcastService.SendToPlayer(player, config.EventSystemPong, pongPayload)
	}
}

// Helper functions

func getTeamProgressPayload() map[string]interface{} {
	gameManager := services.GetGameInstance()
	game := gameManager.GetGame()

	return map[string]interface{}{
		"currentRound": game.CurrentRound,
		"totalRounds":  config.ResourceGatheringRounds,
		"teamTokens": map[string]int{
			"anchorTokens":  game.TeamTokens.AnchorTokens,
			"chronosTokens": game.TeamTokens.ChronosTokens,
			"guideTokens":   game.TeamTokens.GuideTokens,
			"clarityTokens": game.TeamTokens.ClarityTokens,
		},
		"tokenThresholds": map[string]interface{}{
			"anchor": map[string]interface{}{
				"currentThreshold":   game.TeamTokens.GetThreshold(models.TokenAnchor),
				"maxThresholds":      config.MaxThresholds,
				"tokensPerThreshold": config.AnchorTokenThreshold,
				"effectDescription":  "2 pieces pre-solved per threshold",
			},
			"chronos": map[string]interface{}{
				"currentThreshold":   game.TeamTokens.GetThreshold(models.TokenChronos),
				"maxThresholds":      config.MaxThresholds,
				"tokensPerThreshold": config.ChronosTokenThreshold,
				"effectDescription":  "+20 seconds per threshold",
			},
			"guide": map[string]interface{}{
				"currentThreshold":   game.TeamTokens.GetThreshold(models.TokenGuide),
				"maxThresholds":      config.MaxThresholds,
				"tokensPerThreshold": config.GuideTokenThreshold,
				"effectDescription":  "Remove (gridSize²)/7 squares per threshold",
			},
			"clarity": map[string]interface{}{
				"currentThreshold":   game.TeamTokens.GetThreshold(models.TokenClarity),
				"maxThresholds":      config.MaxThresholds,
				"tokensPerThreshold": config.ClarityTokenThreshold,
				"effectDescription":  "+1 second preview per threshold",
			},
		},
	}
}
