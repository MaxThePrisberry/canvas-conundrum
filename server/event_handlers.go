package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/MaxThePrisberry/canvas-conundrum/server/constants"
)

// EventHandlers contains all WebSocket event handler functions
type EventHandlers struct {
	gameManager   *GameManager
	playerManager *PlayerManager
	broadcastChan chan BroadcastMessage
}

// NewEventHandlers creates a new event handlers instance
func NewEventHandlers(gm *GameManager, pm *PlayerManager, bc chan BroadcastMessage) *EventHandlers {
	return &EventHandlers{
		gameManager:   gm,
		playerManager: pm,
		broadcastChan: bc,
	}
}

// HandlePlayerJoin handles new player connections
func (eh *EventHandlers) HandlePlayerJoin(player *Player, payload json.RawMessage) error {
	// Player is already created in WebSocket handler

	// Send available roles (only relevant for non-host players)
	player.mu.RLock()
	isHost := player.IsHost
	player.mu.RUnlock()

	if isHost {
		// Host gets a different response - no roles or specialties needed
		response := map[string]interface{}{
			"playerId": player.ID,
			"isHost":   true,
			"message":  "Connected as game host",
		}
		return sendToPlayer(player, MsgAvailableRoles, response)
	} else {
		// Regular player gets roles and trivia categories
		roles := eh.playerManager.GetAvailableRoles()
		response := map[string]interface{}{
			"playerId":         player.ID,
			"isHost":           false,
			"roles":            roles,
			"triviaCategories": constants.TriviaCategories,
		}
		return sendToPlayer(player, MsgAvailableRoles, response)
	}
}

// HandleRoleSelection handles player role selection
func (eh *EventHandlers) HandleRoleSelection(playerID string, payload json.RawMessage) error {
	var data struct {
		Role string `json:"role"`
	}

	if err := json.Unmarshal(payload, &data); err != nil {
		return fmt.Errorf("invalid payload: %v", err)
	}

	// Set player role
	if err := eh.playerManager.SetPlayerRole(playerID, data.Role); err != nil {
		return err
	}

	// Broadcast lobby status update
	eh.broadcastLobbyStatus()

	return nil
}

// HandleTriviaSpecialtySelection handles player specialty selection
func (eh *EventHandlers) HandleTriviaSpecialtySelection(playerID string, payload json.RawMessage) error {
	var data struct {
		Specialties []string `json:"specialties"`
	}

	if err := json.Unmarshal(payload, &data); err != nil {
		return fmt.Errorf("invalid payload: %v", err)
	}

	// Set player specialties
	if err := eh.playerManager.SetPlayerSpecialties(playerID, data.Specialties); err != nil {
		return err
	}

	// Auto-mark player as ready after selecting specialties
	eh.playerManager.SetPlayerReady(playerID, true)

	// Broadcast lobby status update
	eh.broadcastLobbyStatus()

	return nil
}

// HandlePlayerReady handles player ready status
func (eh *EventHandlers) HandlePlayerReady(playerID string, payload json.RawMessage) error {
	var data struct {
		Ready bool `json:"ready"`
	}

	if err := json.Unmarshal(payload, &data); err != nil {
		return fmt.Errorf("invalid payload: %v", err)
	}

	// Set player ready status (this will fail for hosts, which is intended)
	if err := eh.playerManager.SetPlayerReady(playerID, data.Ready); err != nil {
		return err
	}

	// Broadcast lobby status update
	eh.broadcastLobbyStatus()

	// Check if we should start countdown
	if eh.shouldStartCountdown() {
		go eh.startGameCountdown()
	}

	return nil
}

// HandleHostStartGame handles host starting the game
func (eh *EventHandlers) HandleHostStartGame(playerID string, payload json.RawMessage) error {
	// Verify player is host
	player, err := eh.playerManager.GetPlayer(playerID)
	if err != nil {
		return err
	}

	if !player.IsHost {
		return fmt.Errorf("only host can start the game")
	}

	// Check if game can be started
	canStart, reason := eh.gameManager.CanStartGame()
	if !canStart {
		return fmt.Errorf("cannot start game: %s", reason)
	}

	// Start the game
	return eh.gameManager.StartGame()
}

// HandleResourceLocationVerified handles player location verification
func (eh *EventHandlers) HandleResourceLocationVerified(playerID string, payload json.RawMessage) error {
	var data struct {
		VerifiedHash string `json:"verifiedHash"`
	}

	if err := json.Unmarshal(payload, &data); err != nil {
		return fmt.Errorf("invalid payload: %v", err)
	}

	// Update player location (will fail for hosts, which is intended)
	return eh.playerManager.UpdatePlayerLocation(playerID, data.VerifiedHash)
}

// HandleTriviaAnswer handles player trivia answers
func (eh *EventHandlers) HandleTriviaAnswer(playerID string, payload json.RawMessage) error {
	var data struct {
		QuestionID string `json:"questionId"`
		Answer     string `json:"answer"`
		Timestamp  int64  `json:"timestamp"`
	}

	if err := json.Unmarshal(payload, &data); err != nil {
		return fmt.Errorf("invalid payload: %v", err)
	}

	// Process the answer (hosts don't participate in trivia)
	return eh.gameManager.ProcessTriviaAnswer(playerID, data.QuestionID, data.Answer)
}

// HandleSegmentCompleted handles puzzle segment completion
func (eh *EventHandlers) HandleSegmentCompleted(playerID string, payload json.RawMessage) error {
	var data struct {
		SegmentID           string `json:"segmentId"`
		CompletionTimestamp int64  `json:"completionTimestamp"`
	}

	if err := json.Unmarshal(payload, &data); err != nil {
		return fmt.Errorf("invalid payload: %v", err)
	}

	// Process segment completion (hosts don't have puzzle segments)
	return eh.gameManager.ProcessSegmentCompleted(playerID, data.SegmentID)
}

// HandleFragmentMoveRequest handles puzzle fragment movement
func (eh *EventHandlers) HandleFragmentMoveRequest(playerID string, payload json.RawMessage) error {
	var data struct {
		FragmentID  string  `json:"fragmentId"`
		NewPosition GridPos `json:"newPosition"`
		Timestamp   int64   `json:"timestamp"`
	}

	if err := json.Unmarshal(payload, &data); err != nil {
		return fmt.Errorf("invalid payload: %v", err)
	}

	// Process fragment move
	return eh.gameManager.ProcessFragmentMove(playerID, data.FragmentID, data.NewPosition)
}

// HandleHostStartPuzzle handles host starting the puzzle phase
func (eh *EventHandlers) HandleHostStartPuzzle(playerID string, payload json.RawMessage) error {
	// Verify player is host
	player, err := eh.playerManager.GetPlayer(playerID)
	if err != nil {
		return err
	}

	if !player.IsHost {
		return fmt.Errorf("only host can start the puzzle")
	}

	// Start puzzle timer
	return eh.gameManager.StartPuzzle()
}

// HandlePieceRecommendationRequest handles piece recommendation requests
func (eh *EventHandlers) HandlePieceRecommendationRequest(playerID string, payload json.RawMessage) error {
	var data struct {
		ToPlayerID       string  `json:"toPlayerId"`
		FromFragmentID   string  `json:"fromFragmentId"`
		ToFragmentID     string  `json:"toFragmentId"`
		SuggestedFromPos GridPos `json:"suggestedFromPos"`
		SuggestedToPos   GridPos `json:"suggestedToPos"`
	}

	if err := json.Unmarshal(payload, &data); err != nil {
		return fmt.Errorf("invalid payload: %v", err)
	}

	// Validate that target player exists
	if _, err := eh.playerManager.GetPlayer(data.ToPlayerID); err != nil {
		return fmt.Errorf("target player not found")
	}

	// Process the recommendation (no custom message support)
	// FIXED: Removed empty string parameter that was passed for message
	return eh.gameManager.ProcessPieceRecommendation(
		playerID,
		data.ToPlayerID,
		data.FromFragmentID,
		data.ToFragmentID,
		data.SuggestedFromPos,
		data.SuggestedToPos,
	)
}

// HandlePieceRecommendationResponse handles piece recommendation responses
func (eh *EventHandlers) HandlePieceRecommendationResponse(playerID string, payload json.RawMessage) error {
	var data struct {
		RecommendationID string `json:"recommendationId"`
		Accepted         bool   `json:"accepted"`
	}

	if err := json.Unmarshal(payload, &data); err != nil {
		return fmt.Errorf("invalid payload: %v", err)
	}

	// Process the response
	return eh.gameManager.ProcessPieceRecommendationResponse(playerID, data.RecommendationID, data.Accepted)
}

// Helper functions

// broadcastLobbyStatus sends lobby status to all players
func (eh *EventHandlers) broadcastLobbyStatus() {
	roleDistribution := eh.playerManager.GetRoleDistribution()

	// Determine waiting message
	waitingMessage := ""
	connectedCount := eh.playerManager.GetConnectedCount()
	readyCount := eh.playerManager.GetReadyCount()
	nonHostCount := eh.playerManager.GetConnectedNonHostPlayers()
	hasHost := eh.playerManager.IsHostConnected()

	// Check for host
	if !hasHost {
		waitingMessage = "Waiting for host to connect..."
	} else if len(nonHostCount) < constants.MinPlayers {
		waitingMessage = fmt.Sprintf("Waiting for %d more players...", constants.MinPlayers-len(nonHostCount))
	} else if readyCount < connectedCount {
		waitingMessage = fmt.Sprintf("Waiting for all players to be ready (%d/%d)...", readyCount, connectedCount)
	} else {
		canStart, reason := eh.gameManager.CanStartGame()
		if !canStart {
			waitingMessage = reason
		} else {
			waitingMessage = "Ready to start! (Host can begin the game)"
		}
	}

	status := map[string]interface{}{
		"currentPlayers": connectedCount,
		"nonHostPlayers": len(nonHostCount),
		"playerRoles":    roleDistribution,
		"hasHost":        hasHost,
		"gameStarting":   false,
		"waitingMessage": waitingMessage,
	}

	eh.broadcastChan <- BroadcastMessage{
		Type:    MsgGameLobbyStatus,
		Payload: status,
	}

	// Send host update
	eh.gameManager.sendHostUpdate()
}

// shouldStartCountdown checks if automatic countdown should start
func (eh *EventHandlers) shouldStartCountdown() bool {
	// Only start countdown if host manually starts the game
	// No automatic countdown with the new host system
	return false
}

// startGameCountdown starts the automatic game countdown (disabled with host system)
func (eh *EventHandlers) startGameCountdown() {
	// This method is now disabled since only hosts can start games
	log.Println("Automatic countdown disabled - only host can start games")
}
