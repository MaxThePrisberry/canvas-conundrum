package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

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

	// Send available roles
	roles := eh.playerManager.GetAvailableRoles()

	response := map[string]interface{}{
		"playerId":         player.ID,
		"roles":            roles,
		"triviaCategories": constants.TriviaCategories,
	}

	return sendToPlayer(player, MsgAvailableRoles, response)
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

	// Set player ready status
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

	// Update player location
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

	// Process the answer
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

	// Process segment completion
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

// IMPLEMENTED: Handle piece recommendation requests
func (eh *EventHandlers) HandlePieceRecommendationRequest(playerID string, payload json.RawMessage) error {
	var data struct {
		ToPlayerID       string  `json:"toPlayerId"`
		FromFragmentID   string  `json:"fromFragmentId"`
		ToFragmentID     string  `json:"toFragmentId"`
		SuggestedFromPos GridPos `json:"suggestedFromPos"`
		SuggestedToPos   GridPos `json:"suggestedToPos"`
		Message          string  `json:"message"`
	}

	if err := json.Unmarshal(payload, &data); err != nil {
		return fmt.Errorf("invalid payload: %v", err)
	}

	// Validate that target player exists
	if _, err := eh.playerManager.GetPlayer(data.ToPlayerID); err != nil {
		return fmt.Errorf("target player not found")
	}

	// Process the recommendation
	return eh.gameManager.ProcessPieceRecommendation(
		playerID,
		data.ToPlayerID,
		data.Message,
		data.FromFragmentID,
		data.ToFragmentID,
		data.SuggestedFromPos,
		data.SuggestedToPos,
	)
}

// IMPLEMENTED: Handle piece recommendation responses
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

	if connectedCount < constants.MinPlayers {
		waitingMessage = fmt.Sprintf("Waiting for %d more players...", constants.MinPlayers-connectedCount)
	} else if readyCount < connectedCount {
		waitingMessage = fmt.Sprintf("Waiting for all players to be ready (%d/%d)...", readyCount, connectedCount)
	} else {
		canStart, reason := eh.gameManager.CanStartGame()
		if !canStart {
			waitingMessage = reason
		} else {
			waitingMessage = "Ready to start!"
		}
	}

	status := map[string]interface{}{
		"currentPlayers": connectedCount,
		"playerRoles":    roleDistribution,
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
	// Start countdown if minimum players reached and all are ready
	canStart, _ := eh.gameManager.CanStartGame()
	return canStart && eh.gameManager.GetPhase() == PhaseSetup
}

// startGameCountdown starts the automatic game countdown
func (eh *EventHandlers) startGameCountdown() {
	countdown := constants.LobbyCountdownDuration

	// Cancel any existing countdown
	select {
	case eh.gameManager.countdownCancel <- struct{}{}:
	default:
	}

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for countdown > 0 {
		select {
		case <-ticker.C:
			countdown--

			// Send countdown update
			eh.broadcastChan <- BroadcastMessage{
				Type: MsgCountdown,
				Payload: map[string]interface{}{
					"seconds":  countdown,
					"message":  fmt.Sprintf("Game starting in %d seconds...", countdown),
					"canAbort": true,
				},
			}

			// Update lobby status
			status := map[string]interface{}{
				"currentPlayers": eh.playerManager.GetConnectedCount(),
				"playerRoles":    eh.playerManager.GetRoleDistribution(),
				"gameStarting":   true,
				"waitingMessage": fmt.Sprintf("Game starting in %d seconds...", countdown),
			}

			eh.broadcastChan <- BroadcastMessage{
				Type:    MsgGameLobbyStatus,
				Payload: status,
			}

		case <-eh.gameManager.countdownCancel:
			// Countdown cancelled
			log.Println("Game countdown cancelled")
			eh.broadcastLobbyStatus()
			return
		}

		// Check if we still can start
		canStart, _ := eh.gameManager.CanStartGame()
		if !canStart {
			log.Println("Game start conditions no longer met")
			eh.broadcastLobbyStatus()
			return
		}
	}

	// Start the game
	if err := eh.gameManager.StartGame(); err != nil {
		log.Printf("Error starting game: %v", err)
		eh.broadcastLobbyStatus()
	}
}
