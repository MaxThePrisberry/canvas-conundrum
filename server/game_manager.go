package main

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/MaxThePrisberry/canvas-conundrum/server/constants"
	"github.com/google/uuid"
)

// GameManager handles the core game logic and state
type GameManager struct {
	state           *GameState
	playerManager   *PlayerManager
	triviaManager   *TriviaManager
	broadcastChan   chan BroadcastMessage
	stopChan        chan struct{}
	countdownCancel chan struct{}
	mu              sync.RWMutex
}

// NewGameManager creates a new game manager instance
func NewGameManager(playerManager *PlayerManager, triviaManager *TriviaManager, broadcastChan chan BroadcastMessage) *GameManager {
	gm := &GameManager{
		state: &GameState{
			Phase:                PhaseSetup,
			Difficulty:           "medium",
			Players:              make(map[string]*Player),
			TeamTokens:           TeamTokens{},
			QuestionHistory:      make(map[string]map[string]bool),
			PlayerAnalytics:      make(map[string]*PlayerAnalytics),
			PieceRecommendations: make(map[string]*PieceRecommendation),
			CurrentQuestions:     make(map[string]*TriviaQuestion),
		},
		playerManager:   playerManager,
		triviaManager:   triviaManager,
		broadcastChan:   broadcastChan,
		stopChan:        make(chan struct{}),
		countdownCancel: make(chan struct{}),
	}

	// Initialize random seed
	rand.Seed(time.Now().UnixNano())

	return gm
}

// GetPhase returns the current game phase
func (gm *GameManager) GetPhase() GamePhase {
	gm.mu.RLock()
	defer gm.mu.RUnlock()
	return gm.state.Phase
}

// SetDifficulty sets the game difficulty
func (gm *GameManager) SetDifficulty(difficulty string) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	if gm.state.Phase != PhaseSetup {
		return fmt.Errorf("can only set difficulty during setup phase")
	}

	validDifficulties := map[string]bool{
		"easy":   true,
		"medium": true,
		"hard":   true,
	}

	if !validDifficulties[difficulty] {
		return fmt.Errorf("invalid difficulty")
	}

	gm.state.Difficulty = difficulty
	return nil
}

// CanStartGame checks if the game can be started
func (gm *GameManager) CanStartGame() (bool, string) {
	// Check if host is connected
	if !gm.playerManager.IsHostConnected() {
		return false, "Host must be connected to start the game"
	}

	// Get non-host players for game requirements
	connectedPlayers := gm.playerManager.GetConnectedNonHostPlayers()
	readyPlayers := gm.playerManager.GetReadyNonHostPlayers()

	if len(connectedPlayers) < constants.MinPlayers {
		return false, fmt.Sprintf("Need at least %d players (current: %d)", constants.MinPlayers, len(connectedPlayers))
	}

	if len(readyPlayers) < len(connectedPlayers) {
		return false, fmt.Sprintf("All players must be ready (%d/%d ready)", len(readyPlayers), len(connectedPlayers))
	}

	// Check if all non-host players have selected roles and specialties
	for _, player := range connectedPlayers {
		player.mu.RLock()
		hasRole := player.Role != ""
		hasSpecialties := len(player.Specialties) > 0
		player.mu.RUnlock()

		if !hasRole {
			return false, "All players must select a role"
		}
		if !hasSpecialties {
			return false, "All players must select specialties"
		}
	}

	return true, ""
}

// StartGame transitions from setup to resource gathering phase - Updated initialization
func (gm *GameManager) StartGame() error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	if gm.state.Phase != PhaseSetup {
		return fmt.Errorf("game already started")
	}

	// Initialize player analytics for NON-HOST players only
	nonHostPlayers := gm.playerManager.GetConnectedNonHostPlayers()
	for _, player := range nonHostPlayers {
		gm.state.PlayerAnalytics[player.ID] = &PlayerAnalytics{
			PlayerID:        player.ID,
			PlayerName:      player.Name,
			TokenCollection: make(map[string]int),
			TriviaPerformance: TriviaPerformance{
				AccuracyByCategory: make(map[string]float64),
			},
			PuzzleMetrics: PuzzleSolvingMetrics{},
		}

		// Initialize question history
		gm.state.QuestionHistory[player.ID] = make(map[string]bool)
	}

	// Transition to resource gathering
	gm.state.Phase = PhaseResourceGathering
	gm.state.CurrentRound = 1
	gm.state.RoundStartTime = time.Now()

	// Start resource gathering phase
	go gm.runResourceGatheringPhase()

	return nil
}

// runResourceGatheringPhase manages the resource gathering phase
func (gm *GameManager) runResourceGatheringPhase() {
	// Send resource phase start message
	gm.broadcastChan <- BroadcastMessage{
		Type: MsgResourcePhaseStart,
		Payload: map[string]interface{}{
			"resourceHashes": constants.ResourceStationHashes,
		},
	}

	// Apply difficulty modifiers
	difficultyMod := gm.getDifficultyModifiers()

	// Run rounds
	totalRounds := int(float64(constants.ResourceGatheringRounds) * difficultyMod.TimeLimitModifier)
	for round := 1; round <= totalRounds; round++ {
		gm.mu.Lock()
		gm.state.CurrentRound = round
		gm.state.RoundStartTime = time.Now()
		gm.mu.Unlock()

		// Send round start to host
		gm.sendHostUpdate()

		// Run trivia questions for this round
		gm.runTriviaRound()

		// Check if game was stopped
		select {
		case <-gm.stopChan:
			return
		default:
		}
	}

	// Transition to puzzle phase
	gm.startPuzzlePhase()
}

// runTriviaRound manages a single trivia round - Updated to only send to non-host players
func (gm *GameManager) runTriviaRound() {
	difficultyMod := gm.getDifficultyModifiers()
	roundDuration := time.Duration(float64(constants.ResourceGatheringRoundDuration)*difficultyMod.TimeLimitModifier) * time.Second
	questionInterval := time.Duration(constants.TriviaQuestionInterval) * time.Second

	roundEnd := time.Now().Add(roundDuration)

	for time.Now().Before(roundEnd) {
		// Send trivia questions to all connected NON-HOST players only
		players := gm.playerManager.GetConnectedNonHostPlayers()

		for _, player := range players {
			go gm.sendTriviaQuestion(player)
		}

		// Send progress update
		gm.sendTeamProgressUpdate()

		// Wait for next question interval
		select {
		case <-time.After(questionInterval):
			continue
		case <-gm.stopChan:
			return
		}
	}
}

// sendTriviaQuestion sends a trivia question to a specific player
func (gm *GameManager) sendTriviaQuestion(player *Player) {
	// Get player's question history
	gm.mu.RLock()
	history := gm.state.QuestionHistory[player.ID]
	difficulty := gm.state.Difficulty
	gm.mu.RUnlock()

	// Get a question
	question, err := gm.triviaManager.GetQuestion(difficulty, player.Specialties, history)
	if err != nil {
		log.Printf("Error getting trivia question for player %s: %v", player.ID, err)
		return
	}

	// Mark question as asked
	gm.mu.Lock()
	gm.state.QuestionHistory[player.ID][question.ID] = true
	gm.state.CurrentQuestions[player.ID] = question
	gm.mu.Unlock()

	// Send question to player
	sendToPlayer(player, MsgTriviaQuestion, question)
}

// ProcessTriviaAnswer handles a player's trivia answer - Updated to exclude hosts
func (gm *GameManager) ProcessTriviaAnswer(playerID, questionID, answer string) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	// Validate game phase
	if gm.state.Phase != PhaseResourceGathering {
		return fmt.Errorf("not in resource gathering phase")
	}

	// Get player
	player, err := gm.playerManager.GetPlayer(playerID)
	if err != nil {
		return err
	}

	// Hosts don't participate in trivia
	player.mu.RLock()
	isHost := player.IsHost
	player.mu.RUnlock()

	if isHost {
		return fmt.Errorf("host does not participate in trivia questions")
	}

	// Check if player has a location
	if player.CurrentLocation == "" {
		return fmt.Errorf("player must be at a resource station")
	}

	// Get the current question for this player
	currentQuestion, exists := gm.state.CurrentQuestions[playerID]
	if !exists || currentQuestion.ID != questionID {
		return fmt.Errorf("invalid or expired question")
	}

	// Validate answer against correct answer
	correct := strings.EqualFold(strings.TrimSpace(answer), strings.TrimSpace(currentQuestion.CorrectAnswer))

	// Update analytics
	analytics := gm.state.PlayerAnalytics[playerID]
	analytics.TriviaPerformance.TotalQuestions++

	// Check if this is a specialty question
	isSpecialtyQuestion := currentQuestion.IsSpecialty
	if isSpecialtyQuestion {
		analytics.TriviaPerformance.SpecialtyTotal++
	}

	if correct {
		analytics.TriviaPerformance.CorrectAnswers++
		if isSpecialtyQuestion {
			analytics.TriviaPerformance.SpecialtyCorrect++
		}

		// Award tokens based on player location
		tokenType := gm.getTokenTypeForLocation(player.CurrentLocation)
		if tokenType != "" {
			tokensAwarded := constants.BaseTokensPerCorrectAnswer

			// Apply role bonus if applicable
			if bonusToken, ok := constants.RoleTokenBonuses[player.Role]; ok && bonusToken == tokenType {
				tokensAwarded = int(float64(tokensAwarded) * constants.RoleResourceMultiplier)
			}

			// Apply specialty point multiplier
			if isSpecialtyQuestion {
				tokensAwarded = int(float64(tokensAwarded) * constants.SpecialtyPointMultiplier)
				analytics.TriviaPerformance.SpecialtyBonus += tokensAwarded - constants.BaseTokensPerCorrectAnswer
			}

			// Apply difficulty modifiers
			difficultyMod := gm.getDifficultyModifiers()
			tokensAwarded = int(float64(tokensAwarded) * difficultyMod.TokenThresholdModifier)

			// Add tokens to team total
			switch tokenType {
			case constants.TokenAnchor:
				gm.state.TeamTokens.AnchorTokens += tokensAwarded
			case constants.TokenChronos:
				gm.state.TeamTokens.ChronosTokens += tokensAwarded
			case constants.TokenGuide:
				gm.state.TeamTokens.GuideTokens += tokensAwarded
			case constants.TokenClarity:
				gm.state.TeamTokens.ClarityTokens += tokensAwarded
			}

			// Track in player analytics
			analytics.TokenCollection[tokenType] += tokensAwarded
		}
	}

	// Remove current question
	delete(gm.state.CurrentQuestions, playerID)

	return nil
}

// getTokenTypeForLocation returns the token type for a given location hash
func (gm *GameManager) getTokenTypeForLocation(locationHash string) string {
	for tokenType, hash := range constants.ResourceStationHashes {
		if hash == locationHash {
			return tokenType
		}
	}
	return ""
}

// startPuzzlePhase transitions to the puzzle assembly phase - Updated for non-host players only
func (gm *GameManager) startPuzzlePhase() {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	gm.state.Phase = PhasePuzzleAssembly

	// Calculate grid size based on NON-HOST player count
	nonHostPlayers := gm.playerManager.GetConnectedNonHostPlayers()
	playerCount := len(nonHostPlayers)
	gridSize := gm.calculateGridSize(playerCount)
	gm.state.GridSize = gridSize

	// Select random puzzle image
	gm.state.PuzzleImageID = fmt.Sprintf("masterpiece_%03d", rand.Intn(constants.AvailablePuzzleImages)+1)

	// Calculate anchor token effects (pre-solved pieces)
	anchorThresholds := gm.state.TeamTokens.AnchorTokens / (constants.AnchorTokenThresholds * int(gm.getDifficultyModifiers().TokenThresholdModifier))
	maxPreSolved := min(anchorThresholds, constants.IndividualPuzzlePieces-4) // Leave at least 4 pieces to solve

	// Initialize puzzle fragments for NON-HOST players only
	gm.state.PuzzleFragments = make(map[string]*PuzzleFragment)

	for i, player := range nonHostPlayers {
		correctPos := gm.calculateCorrectPosition(i, gridSize)
		fragment := &PuzzleFragment{
			ID:              fmt.Sprintf("fragment_%s", player.ID),
			PlayerID:        player.ID,
			Position:        GridPos{X: i % gridSize, Y: i / gridSize}, // Start at distributed positions
			CorrectPosition: correctPos,
			Solved:          false,
			PreSolved:       i < maxPreSolved, // Pre-solve based on anchor tokens
		}

		if fragment.PreSolved {
			fragment.Solved = true
		}

		gm.state.PuzzleFragments[fragment.ID] = fragment
	}

	// Send clarity bonus (image preview)
	clarityThresholds := gm.state.TeamTokens.ClarityTokens / (constants.ClarityTokenThresholds * int(gm.getDifficultyModifiers().TokenThresholdModifier))
	previewDuration := clarityThresholds * constants.ClarityTimeBonus

	if previewDuration > 0 {
		gm.broadcastChan <- BroadcastMessage{
			Type: MsgImagePreview,
			Payload: map[string]interface{}{
				"imageId":  gm.state.PuzzleImageID,
				"duration": previewDuration,
			},
		}
	}

	// Send puzzle phase load message to NON-HOST players only
	for _, player := range nonHostPlayers {
		fragment := gm.state.PuzzleFragments[fmt.Sprintf("fragment_%s", player.ID)]
		segmentID := fmt.Sprintf("segment_%c%d", 'a'+fragment.CorrectPosition.Y, fragment.CorrectPosition.X+1)

		sendToPlayer(player, MsgPuzzlePhaseLoad, map[string]interface{}{
			"imageId":   gm.state.PuzzleImageID,
			"segmentId": segmentID,
			"gridSize":  gridSize,
			"preSolved": fragment.PreSolved,
		})
	}

	// Send a different message to the host
	host := gm.playerManager.GetHost()
	if host != nil {
		sendToPlayer(host, MsgPuzzlePhaseLoad, map[string]interface{}{
			"imageId":     gm.state.PuzzleImageID,
			"gridSize":    gridSize,
			"isHost":      true,
			"playerCount": len(nonHostPlayers),
			"message":     "Puzzle phase started - monitor player progress",
		})
	}
}

// calculateCorrectPosition determines the correct position for a fragment
func (gm *GameManager) calculateCorrectPosition(playerIndex, gridSize int) GridPos {
	return GridPos{
		X: playerIndex % gridSize,
		Y: playerIndex / gridSize,
	}
}

// StartPuzzle begins the puzzle solving timer - IMPLEMENTED CHRONOS TOKEN EFFECTS
func (gm *GameManager) StartPuzzle() error {
	gm.mu.Lock()

	if gm.state.Phase != PhasePuzzleAssembly {
		gm.mu.Unlock()
		return fmt.Errorf("not in puzzle assembly phase")
	}

	gm.state.PuzzleStartTime = time.Now()
	gm.mu.Unlock()

	// IMPLEMENTED: Calculate total time with chronos bonuses and difficulty modifiers
	difficultyMod := gm.getDifficultyModifiers()
	baseTime := int(float64(constants.PuzzleAssemblyBaseTime) * difficultyMod.TimeLimitModifier)

	chronosThresholds := gm.state.TeamTokens.ChronosTokens / (constants.ChronosTokenThresholds * int(difficultyMod.TokenThresholdModifier))
	chronosBonus := chronosThresholds * constants.ChronosTimeBonus

	totalTime := baseTime + chronosBonus

	// Send puzzle phase start
	gm.broadcastChan <- BroadcastMessage{
		Type: MsgPuzzlePhaseStart,
		Payload: map[string]interface{}{
			"startTimestamp": time.Now().Unix(),
			"totalTime":      totalTime,
		},
	}

	// Start puzzle timer
	go gm.runPuzzleTimer(time.Duration(totalTime) * time.Second)

	return nil
}

// runPuzzleTimer manages the puzzle phase timer
func (gm *GameManager) runPuzzleTimer(duration time.Duration) {
	timer := time.NewTimer(duration)
	defer timer.Stop()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timer.C:
			// Time's up!
			gm.endGame(false)
			return
		case <-ticker.C:
			// Send progress updates
			gm.sendPuzzleProgress()
			gm.sendHostUpdate()
		case <-gm.stopChan:
			return
		}
	}
}

// ProcessSegmentCompleted handles when a player completes their puzzle segment - Updated to exclude hosts
func (gm *GameManager) ProcessSegmentCompleted(playerID, segmentID string) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	if gm.state.Phase != PhasePuzzleAssembly {
		return fmt.Errorf("not in puzzle assembly phase")
	}

	// Get player and verify they're not the host
	player, err := gm.playerManager.GetPlayer(playerID)
	if err != nil {
		return err
	}

	player.mu.RLock()
	isHost := player.IsHost
	player.mu.RUnlock()

	if isHost {
		return fmt.Errorf("host does not have puzzle segments to complete")
	}

	// Find the fragment
	fragment := gm.state.PuzzleFragments[fmt.Sprintf("fragment_%s", playerID)]
	if fragment == nil {
		return fmt.Errorf("fragment not found")
	}

	if fragment.PreSolved {
		return fmt.Errorf("fragment was pre-solved by anchor tokens")
	}

	// Mark as solved
	fragment.Solved = true

	// Update analytics
	if analytics, ok := gm.state.PlayerAnalytics[playerID]; ok {
		analytics.PuzzleMetrics.FragmentSolveTime = int(time.Since(gm.state.PuzzleStartTime).Seconds())
	}

	// Send acknowledgment
	sendToPlayer(player, MsgSegmentCompletionAck, map[string]interface{}{
		"status":       "acknowledged",
		"segmentId":    segmentID,
		"gridPosition": fragment.Position,
	})

	// Send guide token hints if available
	gm.sendGuideHints(playerID)

	// Check if all fragments are solved and positioned correctly
	if gm.checkPuzzleComplete() {
		gm.endGame(true)
	}

	return nil
}

// IMPLEMENTED: Send guide token hints for piece placement
func (gm *GameManager) sendGuideHints(playerID string) {
	guideThresholds := gm.state.TeamTokens.GuideTokens / (constants.GuideTokenThresholds * int(gm.getDifficultyModifiers().TokenThresholdModifier))

	if guideThresholds > 0 {
		fragment := gm.state.PuzzleFragments[fmt.Sprintf("fragment_%s", playerID)]
		if fragment != nil {
			// Provide hints based on guide token level
			hintLevel := min(guideThresholds, 3) // Max 3 levels of hints

			hints := make([]string, 0)
			switch hintLevel {
			case 3:
				hints = append(hints, fmt.Sprintf("Exact position: (%d, %d)", fragment.CorrectPosition.X, fragment.CorrectPosition.Y))
			case 2:
				hints = append(hints, fmt.Sprintf("Correct row: %d", fragment.CorrectPosition.Y))
				hints = append(hints, fmt.Sprintf("Correct column: %d", fragment.CorrectPosition.X))
			case 1:
				if fragment.Position.X == fragment.CorrectPosition.X {
					hints = append(hints, "Column is correct!")
				} else if fragment.Position.Y == fragment.CorrectPosition.Y {
					hints = append(hints, "Row is correct!")
				} else {
					hints = append(hints, "Position needs adjustment")
				}
			}

			if player, _ := gm.playerManager.GetPlayer(playerID); player != nil {
				sendToPlayer(player, MsgPieceRecommendation, map[string]interface{}{
					"type":  "guide_hint",
					"hints": hints,
				})
			}
		}
	}
}

// ProcessFragmentMove handles a fragment move request
func (gm *GameManager) ProcessFragmentMove(playerID, fragmentID string, newPos GridPos) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	if gm.state.Phase != PhasePuzzleAssembly {
		return fmt.Errorf("not in puzzle assembly phase")
	}

	fragment, exists := gm.state.PuzzleFragments[fragmentID]
	if !exists {
		return fmt.Errorf("fragment not found")
	}

	// Check cooldown
	cooldownDuration := time.Duration(constants.FragmentMovementCooldown) * time.Millisecond
	if time.Since(fragment.LastMoved) < cooldownDuration {
		player, _ := gm.playerManager.GetPlayer(playerID)
		if player != nil {
			sendToPlayer(player, MsgFragmentMoveResponse, map[string]interface{}{
				"status":            "ignored",
				"reason":            "cooldown",
				"nextMoveAvailable": fragment.LastMoved.Add(cooldownDuration).Unix(),
			})
		}
		return nil
	}

	// Validate new position
	if newPos.X < 0 || newPos.X >= gm.state.GridSize || newPos.Y < 0 || newPos.Y >= gm.state.GridSize {
		return fmt.Errorf("position out of bounds")
	}

	// Find fragment at target position and swap
	var targetFragment *PuzzleFragment
	for _, f := range gm.state.PuzzleFragments {
		if f.Position.X == newPos.X && f.Position.Y == newPos.Y {
			targetFragment = f
			break
		}
	}

	oldPos := fragment.Position
	if targetFragment != nil {
		// Swap positions
		fragment.Position = newPos
		targetFragment.Position = oldPos
		targetFragment.LastMoved = time.Now()
	} else {
		// Just move to empty position
		fragment.Position = newPos
	}

	fragment.LastMoved = time.Now()

	// Record move
	gm.state.FragmentMoveHistory = append(gm.state.FragmentMoveHistory, FragmentMove{
		FragmentID: fragmentID,
		FromPos:    oldPos,
		ToPos:      newPos,
		PlayerID:   playerID,
		Timestamp:  time.Now(),
	})

	// Update analytics
	if analytics, ok := gm.state.PlayerAnalytics[playerID]; ok {
		analytics.PuzzleMetrics.MovesContributed++
		analytics.PuzzleMetrics.SuccessfulMoves++
	}

	// Send response
	player, _ := gm.playerManager.GetPlayer(playerID)
	if player != nil {
		sendToPlayer(player, MsgFragmentMoveResponse, map[string]interface{}{
			"status":   "success",
			"fragment": fragment,
		})
	}

	// Broadcast updated puzzle state
	gm.broadcastPuzzleState()

	// Check if puzzle is complete after move
	if gm.checkPuzzleComplete() {
		gm.endGame(true)
	}

	return nil
}

// IMPLEMENTED: Comprehensive puzzle completion check
func (gm *GameManager) checkPuzzleComplete() bool {
	// Check if all fragments are solved
	for _, fragment := range gm.state.PuzzleFragments {
		if !fragment.Solved {
			return false
		}
	}

	// IMPLEMENTED: Check if fragments are in correct positions
	for _, fragment := range gm.state.PuzzleFragments {
		if fragment.Position.X != fragment.CorrectPosition.X || fragment.Position.Y != fragment.CorrectPosition.Y {
			return false
		}
	}

	return true
}

// IMPLEMENTED: Piece recommendation system
func (gm *GameManager) ProcessPieceRecommendation(fromPlayerID, toPlayerID, message string, fromFragmentID, toFragmentID string, suggestedFromPos, suggestedToPos GridPos) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	if gm.state.Phase != PhasePuzzleAssembly {
		return fmt.Errorf("not in puzzle assembly phase")
	}

	// Create recommendation
	recommendation := &PieceRecommendation{
		ID:               uuid.New().String(),
		FromPlayerID:     fromPlayerID,
		ToPlayerID:       toPlayerID,
		FromFragmentID:   fromFragmentID,
		ToFragmentID:     toFragmentID,
		SuggestedFromPos: suggestedFromPos,
		SuggestedToPos:   suggestedToPos,
		Message:          message,
		Timestamp:        time.Now(),
	}

	gm.state.PieceRecommendations[recommendation.ID] = recommendation

	// Update analytics
	if analytics, ok := gm.state.PlayerAnalytics[fromPlayerID]; ok {
		analytics.PuzzleMetrics.RecommendationsSent++
	}
	if analytics, ok := gm.state.PlayerAnalytics[toPlayerID]; ok {
		analytics.PuzzleMetrics.RecommendationsReceived++
	}

	// Send recommendation to target player
	if toPlayer, err := gm.playerManager.GetPlayer(toPlayerID); err == nil {
		sendToPlayer(toPlayer, MsgPieceRecommendation, recommendation)
	}

	return nil
}

// Process piece recommendation response
func (gm *GameManager) ProcessPieceRecommendationResponse(playerID, recommendationID string, accepted bool) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	recommendation, exists := gm.state.PieceRecommendations[recommendationID]
	if !exists {
		return fmt.Errorf("recommendation not found")
	}

	if recommendation.ToPlayerID != playerID {
		return fmt.Errorf("not authorized to respond to this recommendation")
	}

	if accepted {
		// Execute the recommended moves
		if fromFragment, exists := gm.state.PuzzleFragments[recommendation.FromFragmentID]; exists {
			fromFragment.Position = recommendation.SuggestedFromPos
			fromFragment.LastMoved = time.Now()
		}
		if toFragment, exists := gm.state.PuzzleFragments[recommendation.ToFragmentID]; exists {
			toFragment.Position = recommendation.SuggestedToPos
			toFragment.LastMoved = time.Now()
		}

		// Update analytics
		if analytics, ok := gm.state.PlayerAnalytics[playerID]; ok {
			analytics.PuzzleMetrics.RecommendationsAccepted++
		}

		// Broadcast updated puzzle state
		gm.broadcastPuzzleState()

		// Check if puzzle is complete
		if gm.checkPuzzleComplete() {
			gm.endGame(true)
		}
	}

	// Remove recommendation
	delete(gm.state.PieceRecommendations, recommendationID)

	return nil
}

// endGame handles game completion
func (gm *GameManager) endGame(success bool) {
	gm.mu.Lock()
	gm.state.Phase = PhasePostGame
	gm.mu.Unlock()

	// Calculate final analytics
	analytics := gm.calculateFinalAnalytics(success)

	// Send analytics to all players
	gm.broadcastChan <- BroadcastMessage{
		Type:    MsgGameAnalytics,
		Payload: analytics,
	}

	// Send special analytics to host
	gm.sendHostUpdate()

	// Start reset timer
	go func() {
		time.Sleep(time.Duration(constants.PostGameAnalyticsDuration) * time.Second)
		gm.resetGame()
	}()
}

// IMPLEMENTED: Enhanced analytics calculations
func (gm *GameManager) calculateFinalAnalytics(success bool) map[string]interface{} {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	// Individual analytics
	personalAnalytics := make([]PlayerAnalytics, 0)
	totalRecommendations := 0
	acceptedRecommendations := 0

	for _, analytics := range gm.state.PlayerAnalytics {
		// Calculate category accuracies
		if analytics.TriviaPerformance.TotalQuestions > 0 {
			accuracy := float64(analytics.TriviaPerformance.CorrectAnswers) / float64(analytics.TriviaPerformance.TotalQuestions)
			for _, cat := range constants.TriviaCategories {
				analytics.TriviaPerformance.AccuracyByCategory[cat] = accuracy
			}
		}

		totalRecommendations += analytics.PuzzleMetrics.RecommendationsSent
		acceptedRecommendations += analytics.PuzzleMetrics.RecommendationsAccepted

		personalAnalytics = append(personalAnalytics, *analytics)
	}

	// Team analytics
	totalTime := 0
	if !gm.state.PuzzleStartTime.IsZero() {
		totalTime = int(time.Since(gm.state.PuzzleStartTime).Seconds())
	}

	// Calculate collaboration score based on recommendations
	collaborationScore := 0.5 // Base score
	if totalRecommendations > 0 {
		acceptanceRate := float64(acceptedRecommendations) / float64(totalRecommendations)
		collaborationScore = 0.3 + (acceptanceRate * 0.7) // 0.3-1.0 range
	}

	teamAnalytics := TeamAnalytics{
		OverallPerformance: TeamPerformance{
			TotalTime:      totalTime,
			CompletionRate: 0.0,
			TotalScore:     0,
		},
		CollaborationScores: CollaborationMetrics{
			AverageResponseTime:     15.0, // Placeholder - could calculate from move history
			CommunicationScore:      collaborationScore,
			CoordinationScore:       collaborationScore * 0.9, // Slightly lower than communication
			TotalRecommendations:    totalRecommendations,
			AcceptedRecommendations: acceptedRecommendations,
		},
		ResourceEfficiency: ResourceMetrics{
			TokensPerRound: float64(gm.getTotalTokens()) / float64(constants.ResourceGatheringRounds),
			TokenDistribution: map[string]float64{
				constants.TokenAnchor:  float64(gm.state.TeamTokens.AnchorTokens),
				constants.TokenChronos: float64(gm.state.TeamTokens.ChronosTokens),
				constants.TokenGuide:   float64(gm.state.TeamTokens.GuideTokens),
				constants.TokenClarity: float64(gm.state.TeamTokens.ClarityTokens),
			},
			ThresholdsReached: gm.calculateThresholdsReached(),
		},
	}

	if success {
		teamAnalytics.OverallPerformance.CompletionRate = 1.0
		teamAnalytics.OverallPerformance.TotalScore = 1000 - totalTime*5 // Simple scoring
	}

	// Enhanced leaderboard with multiple scoring factors
	leaderboard := make([]LeaderboardEntry, 0)
	for i, analytics := range personalAnalytics {
		score := analytics.TriviaPerformance.CorrectAnswers * 10
		score += analytics.TriviaPerformance.SpecialtyBonus * 2 // Bonus for specialty questions
		if analytics.PuzzleMetrics.FragmentSolveTime > 0 {
			score += 100 // Completion bonus
			// Time bonus (faster = better)
			if analytics.PuzzleMetrics.FragmentSolveTime < 300 { // Under 5 minutes
				score += (300 - analytics.PuzzleMetrics.FragmentSolveTime)
			}
		}
		score += analytics.PuzzleMetrics.SuccessfulMoves * 5
		score += analytics.PuzzleMetrics.RecommendationsSent * 3
		score += analytics.PuzzleMetrics.RecommendationsAccepted * 8 // Collaboration bonus

		leaderboard = append(leaderboard, LeaderboardEntry{
			PlayerID:   analytics.PlayerID,
			PlayerName: analytics.PlayerName,
			TotalScore: score,
			Rank:       i + 1,
		})
	}

	return map[string]interface{}{
		"personalAnalytics": personalAnalytics,
		"teamAnalytics":     teamAnalytics,
		"globalLeaderboard": leaderboard,
		"gameSuccess":       success,
	}
}

// IMPLEMENTED: Get difficulty modifiers
func (gm *GameManager) getDifficultyModifiers() constants.DifficultyModifiers {
	switch gm.state.Difficulty {
	case "easy":
		return constants.EasyMode
	case "hard":
		return constants.HardMode
	default:
		return constants.MediumMode
	}
}

// Helper functions

func (gm *GameManager) calculateGridSize(playerCount int) int {
	for _, breakpoint := range constants.GridSizeBreakpoints {
		if playerCount >= breakpoint.MinPlayers && playerCount <= breakpoint.MaxPlayers {
			return breakpoint.GridSize
		}
	}
	// Default to square root if not in breakpoints
	return int(math.Ceil(math.Sqrt(float64(playerCount))))
}

func (gm *GameManager) getTotalTokens() int {
	return gm.state.TeamTokens.AnchorTokens +
		gm.state.TeamTokens.ChronosTokens +
		gm.state.TeamTokens.GuideTokens +
		gm.state.TeamTokens.ClarityTokens
}

func (gm *GameManager) calculateThresholdsReached() map[string]int {
	difficultyMod := gm.getDifficultyModifiers()
	return map[string]int{
		constants.TokenAnchor:  gm.state.TeamTokens.AnchorTokens / (constants.AnchorTokenThresholds * int(difficultyMod.TokenThresholdModifier)),
		constants.TokenChronos: gm.state.TeamTokens.ChronosTokens / (constants.ChronosTokenThresholds * int(difficultyMod.TokenThresholdModifier)),
		constants.TokenGuide:   gm.state.TeamTokens.GuideTokens / (constants.GuideTokenThresholds * int(difficultyMod.TokenThresholdModifier)),
		constants.TokenClarity: gm.state.TeamTokens.ClarityTokens / (constants.ClarityTokenThresholds * int(difficultyMod.TokenThresholdModifier)),
	}
}

// sendTeamProgressUpdate sends progress updates - Updated calculation for non-host players
func (gm *GameManager) sendTeamProgressUpdate() {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	totalQuestions := 0
	// Only count questions from non-host players
	for playerID, analytics := range gm.state.PlayerAnalytics {
		// Verify this player is not a host
		if player, err := gm.playerManager.GetPlayer(playerID); err == nil {
			player.mu.RLock()
			isHost := player.IsHost
			player.mu.RUnlock()

			if !isHost {
				totalQuestions += analytics.TriviaPerformance.TotalQuestions
			}
		}
	}

	nonHostPlayerCount := len(gm.playerManager.GetConnectedNonHostPlayers())

	gm.broadcastChan <- BroadcastMessage{
		Type: MsgTeamProgressUpdate,
		Payload: map[string]interface{}{
			"questionsAnswered": totalQuestions,
			"totalQuestions":    constants.ResourceGatheringRounds * nonHostPlayerCount,
			"teamTokens":        gm.state.TeamTokens,
		},
	}
}

func (gm *GameManager) sendPuzzleProgress() {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	solvedCount := 0
	for _, fragment := range gm.state.PuzzleFragments {
		if fragment.Solved {
			solvedCount++
		}
	}

	// This is primarily for the host
	gm.sendHostUpdate()
}

func (gm *GameManager) broadcastPuzzleState() {
	gm.mu.RLock()
	fragments := make([]*PuzzleFragment, 0, len(gm.state.PuzzleFragments))
	for _, f := range gm.state.PuzzleFragments {
		fragments = append(fragments, f)
	}
	gm.mu.RUnlock()

	gm.broadcastChan <- BroadcastMessage{
		Type: MsgCentralPuzzleState,
		Payload: map[string]interface{}{
			"fragments": fragments,
			"gridSize":  gm.state.GridSize,
		},
	}
}

// sendHostUpdate updates host with current game status - Enhanced for new host system
func (gm *GameManager) sendHostUpdate() {
	gm.mu.RLock()

	playerStatuses := make(map[string]PlayerStatus)
	allPlayers := gm.playerManager.GetAllPlayers()

	for _, player := range allPlayers {
		player.mu.RLock()

		// Don't include the host in player statuses
		if !player.IsHost {
			playerStatuses[player.ID] = PlayerStatus{
				Name:      player.Name,
				Role:      player.Role,
				Connected: player.State == StateConnected,
				Ready:     player.Ready,
				Location:  player.CurrentLocation,
			}
		}
		player.mu.RUnlock()
	}

	// Calculate progress based on non-host players
	var progress float64
	if gm.state.Phase == PhasePuzzleAssembly {
		solvedCount := 0
		for _, fragment := range gm.state.PuzzleFragments {
			if fragment.Solved {
				solvedCount++
			}
		}
		if len(gm.state.PuzzleFragments) > 0 {
			progress = float64(solvedCount) / float64(len(gm.state.PuzzleFragments))
		}
	}

	// Calculate time remaining
	var timeRemaining int
	if gm.state.Phase == PhaseResourceGathering {
		elapsed := time.Since(gm.state.RoundStartTime)
		remaining := time.Duration(constants.ResourceGatheringRoundDuration)*time.Second - elapsed
		if remaining > 0 {
			timeRemaining = int(remaining.Seconds())
		}
	}

	nonHostPlayers := gm.playerManager.GetConnectedNonHostPlayers()
	readyNonHostPlayers := gm.playerManager.GetReadyNonHostPlayers()

	update := HostUpdate{
		Phase:            gm.state.Phase.String(),
		ConnectedPlayers: len(nonHostPlayers),
		ReadyPlayers:     len(readyNonHostPlayers),
		CurrentRound:     gm.state.CurrentRound,
		TimeRemaining:    timeRemaining,
		TeamTokens:       gm.state.TeamTokens,
		PlayerStatuses:   playerStatuses,
		PuzzleProgress:   progress,
	}

	gm.mu.RUnlock()

	// Send only to host
	host := gm.playerManager.GetHost()
	if host != nil {
		sendToPlayer(host, MsgHostUpdate, update)
	}
}

func (gm *GameManager) resetGame() {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	// Send reset message
	gm.broadcastChan <- BroadcastMessage{
		Type: MsgGameReset,
		Payload: map[string]interface{}{
			"message":           "Game resetting. Please rejoin to start a new game.",
			"reconnectRequired": true,
		},
	}

	// Reset state
	gm.state = &GameState{
		Phase:                PhaseSetup,
		Difficulty:           "medium",
		Players:              make(map[string]*Player),
		TeamTokens:           TeamTokens{},
		QuestionHistory:      make(map[string]map[string]bool),
		PlayerAnalytics:      make(map[string]*PlayerAnalytics),
		PieceRecommendations: make(map[string]*PieceRecommendation),
		CurrentQuestions:     make(map[string]*TriviaQuestion),
	}
}

// String methods for GamePhase
func (p GamePhase) String() string {
	switch p {
	case PhaseSetup:
		return "setup"
	case PhaseResourceGathering:
		return "resource_gathering"
	case PhasePuzzleAssembly:
		return "puzzle_assembly"
	case PhasePostGame:
		return "post_game"
	default:
		return "unknown"
	}
}

// Utility function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
