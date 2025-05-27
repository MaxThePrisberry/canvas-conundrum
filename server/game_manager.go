package main

import (
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"canvas-conundrum/server/constants"
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
			Phase:           PhaseSetup,
			Difficulty:      "medium",
			Players:         make(map[string]*Player),
			TeamTokens:      TeamTokens{},
			QuestionHistory: make(map[string]map[string]bool),
			PlayerAnalytics: make(map[string]*PlayerAnalytics),
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
	connectedPlayers := gm.playerManager.GetConnectedPlayers()
	readyPlayers := gm.playerManager.GetReadyPlayers()

	if len(connectedPlayers) < constants.MinPlayers {
		return false, fmt.Sprintf("Need at least %d players (current: %d)", constants.MinPlayers, len(connectedPlayers))
	}

	if len(readyPlayers) < len(connectedPlayers) {
		return false, fmt.Sprintf("All players must be ready (%d/%d ready)", len(readyPlayers), len(connectedPlayers))
	}

	// Check if all players have selected roles and specialties
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

// StartGame transitions from setup to resource gathering phase
func (gm *GameManager) StartGame() error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	if gm.state.Phase != PhaseSetup {
		return fmt.Errorf("game already started")
	}

	// Initialize player analytics
	for _, player := range gm.playerManager.GetAllPlayers() {
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

	// Run rounds
	for round := 1; round <= constants.ResourceGatheringRounds; round++ {
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

// runTriviaRound manages a single trivia round
func (gm *GameManager) runTriviaRound() {
	roundDuration := time.Duration(constants.ResourceGatheringRoundDuration) * time.Second
	questionInterval := time.Duration(constants.TriviaQuestionInterval) * time.Second

	roundEnd := time.Now().Add(roundDuration)

	for time.Now().Before(roundEnd) {
		// Send trivia questions to all connected players
		players := gm.playerManager.GetConnectedPlayers()

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
	gm.mu.Unlock()

	// Send question to player
	sendToPlayer(player, MsgTriviaQuestion, question)
}

// ProcessTriviaAnswer handles a player's trivia answer
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

	// Check if player has a location
	if player.CurrentLocation == "" {
		return fmt.Errorf("player must be at a resource station")
	}

	// TODO: Validate answer against question
	// For now, simulate 70% correct answer rate
	correct := rand.Float32() < 0.7

	// Update analytics
	analytics := gm.state.PlayerAnalytics[playerID]
	analytics.TriviaPerformance.TotalQuestions++

	if correct {
		analytics.TriviaPerformance.CorrectAnswers++

		// Award tokens based on player location
		tokenType := gm.getTokenTypeForLocation(player.CurrentLocation)
		if tokenType != "" {
			tokensAwarded := constants.BaseTokensPerCorrectAnswer

			// Apply role bonus if applicable
			if bonusToken, ok := constants.RoleTokenBonuses[player.Role]; ok && bonusToken == tokenType {
				tokensAwarded = int(float64(tokensAwarded) * constants.RoleResourceMultiplier)
			}

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

// startPuzzlePhase transitions to the puzzle assembly phase
func (gm *GameManager) startPuzzlePhase() {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	gm.state.Phase = PhasePuzzleAssembly

	// Calculate grid size based on player count
	playerCount := gm.playerManager.GetConnectedCount()
	gridSize := gm.calculateGridSize(playerCount)
	gm.state.GridSize = gridSize

	// Select random puzzle image
	gm.state.PuzzleImageID = fmt.Sprintf("masterpiece_%03d", rand.Intn(constants.AvailablePuzzleImages)+1)

	// Initialize puzzle fragments
	gm.state.PuzzleFragments = make(map[string]*PuzzleFragment)
	players := gm.playerManager.GetConnectedPlayers()

	for i, player := range players {
		fragment := &PuzzleFragment{
			ID:       fmt.Sprintf("fragment_%s", player.ID),
			PlayerID: player.ID,
			Position: GridPos{X: i % gridSize, Y: i / gridSize},
			Solved:   false,
		}
		gm.state.PuzzleFragments[fragment.ID] = fragment
	}

	// Send puzzle phase load message
	for _, player := range players {
		fragment := gm.state.PuzzleFragments[fmt.Sprintf("fragment_%s", player.ID)]
		segmentID := fmt.Sprintf("segment_%c%d", 'a'+fragment.Position.Y, fragment.Position.X+1)

		sendToPlayer(player, MsgPuzzlePhaseLoad, map[string]interface{}{
			"imageId":   gm.state.PuzzleImageID,
			"segmentId": segmentID,
			"gridSize":  gridSize,
		})
	}
}

// StartPuzzle begins the puzzle solving timer
func (gm *GameManager) StartPuzzle() error {
	gm.mu.Lock()

	if gm.state.Phase != PhasePuzzleAssembly {
		gm.mu.Unlock()
		return fmt.Errorf("not in puzzle assembly phase")
	}

	gm.state.PuzzleStartTime = time.Now()
	gm.mu.Unlock()

	// Calculate total time with bonuses
	baseTime := constants.PuzzleAssemblyBaseTime
	chronosBonus := (gm.state.TeamTokens.ChronosTokens / constants.ChronosTokenThresholds) * constants.ChronosTimeBonus
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

// ProcessSegmentCompleted handles when a player completes their puzzle segment
func (gm *GameManager) ProcessSegmentCompleted(playerID, segmentID string) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	if gm.state.Phase != PhasePuzzleAssembly {
		return fmt.Errorf("not in puzzle assembly phase")
	}

	// Find the fragment
	fragment := gm.state.PuzzleFragments[fmt.Sprintf("fragment_%s", playerID)]
	if fragment == nil {
		return fmt.Errorf("fragment not found")
	}

	// Mark as solved
	fragment.Solved = true

	// Update analytics
	if analytics, ok := gm.state.PlayerAnalytics[playerID]; ok {
		analytics.PuzzleMetrics.FragmentSolveTime = int(time.Since(gm.state.PuzzleStartTime).Seconds())
	}

	// Send acknowledgment
	player, _ := gm.playerManager.GetPlayer(playerID)
	if player != nil {
		sendToPlayer(player, MsgSegmentCompletionAck, map[string]interface{}{
			"status":       "acknowledged",
			"segmentId":    segmentID,
			"gridPosition": fragment.Position,
		})
	}

	// Check if all fragments are solved
	if gm.checkPuzzleComplete() {
		gm.endGame(true)
	}

	return nil
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
	if time.Since(fragment.LastMoved) < time.Duration(constants.FragmentMovementCooldown)*time.Millisecond {
		player, _ := gm.playerManager.GetPlayer(playerID)
		if player != nil {
			sendToPlayer(player, MsgFragmentMoveResponse, map[string]interface{}{
				"status":            "ignored",
				"reason":            "cooldown",
				"nextMoveAvailable": fragment.LastMoved.Add(time.Duration(constants.FragmentMovementCooldown) * time.Millisecond).Unix(),
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

	if targetFragment != nil {
		// Swap positions
		oldPos := fragment.Position
		fragment.Position = newPos
		targetFragment.Position = oldPos
	} else {
		// Just move to empty position
		fragment.Position = newPos
	}

	fragment.LastMoved = time.Now()

	// Record move
	gm.state.FragmentMoveHistory = append(gm.state.FragmentMoveHistory, FragmentMove{
		FragmentID: fragmentID,
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

	return nil
}

// checkPuzzleComplete checks if the puzzle is complete
func (gm *GameManager) checkPuzzleComplete() bool {
	// Check if all fragments are solved
	for _, fragment := range gm.state.PuzzleFragments {
		if !fragment.Solved {
			return false
		}
	}

	// Check if fragments are in correct positions
	// For now, we'll assume any complete configuration is valid
	// In a real implementation, you'd check against the correct solution

	return true
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

// calculateFinalAnalytics generates the final game analytics
func (gm *GameManager) calculateFinalAnalytics(success bool) map[string]interface{} {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	// Individual analytics
	personalAnalytics := make([]PlayerAnalytics, 0)
	for _, analytics := range gm.state.PlayerAnalytics {
		// Calculate category accuracies
		// This is simplified - in reality you'd track by actual categories
		if analytics.TriviaPerformance.TotalQuestions > 0 {
			accuracy := float64(analytics.TriviaPerformance.CorrectAnswers) / float64(analytics.TriviaPerformance.TotalQuestions)
			for _, cat := range constants.TriviaCategories {
				analytics.TriviaPerformance.AccuracyByCategory[cat] = accuracy
			}
		}
		personalAnalytics = append(personalAnalytics, *analytics)
	}

	// Team analytics
	totalTime := 0
	if !gm.state.PuzzleStartTime.IsZero() {
		totalTime = int(time.Since(gm.state.PuzzleStartTime).Seconds())
	}

	teamAnalytics := TeamAnalytics{
		OverallPerformance: TeamPerformance{
			TotalTime:      totalTime,
			CompletionRate: 0.0,
			TotalScore:     0,
		},
		CollaborationScores: CollaborationMetrics{
			AverageResponseTime: 15.0, // Placeholder
			CommunicationScore:  0.8,  // Placeholder
			CoordinationScore:   0.75, // Placeholder
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

	// Leaderboard
	leaderboard := make([]LeaderboardEntry, 0)
	for i, analytics := range personalAnalytics {
		score := analytics.TriviaPerformance.CorrectAnswers * 10
		if analytics.PuzzleMetrics.FragmentSolveTime > 0 {
			score += 100
		}
		score += analytics.PuzzleMetrics.SuccessfulMoves * 5

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
	return map[string]int{
		constants.TokenAnchor:  gm.state.TeamTokens.AnchorTokens / constants.AnchorTokenThresholds,
		constants.TokenChronos: gm.state.TeamTokens.ChronosTokens / constants.ChronosTokenThresholds,
		constants.TokenGuide:   gm.state.TeamTokens.GuideTokens / constants.GuideTokenThresholds,
		constants.TokenClarity: gm.state.TeamTokens.ClarityTokens / constants.ClarityTokenThresholds,
	}
}

func (gm *GameManager) sendTeamProgressUpdate() {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	totalQuestions := 0
	for _, analytics := range gm.state.PlayerAnalytics {
		totalQuestions += analytics.TriviaPerformance.TotalQuestions
	}

	gm.broadcastChan <- BroadcastMessage{
		Type: MsgTeamProgressUpdate,
		Payload: map[string]interface{}{
			"questionsAnswered": totalQuestions,
			"totalQuestions":    constants.ResourceGatheringRounds * len(gm.playerManager.GetAllPlayers()),
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

	progress := float64(solvedCount) / float64(len(gm.state.PuzzleFragments))

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

func (gm *GameManager) sendHostUpdate() {
	gm.mu.RLock()

	playerStatuses := make(map[string]PlayerStatus)
	for _, player := range gm.playerManager.GetAllPlayers() {
		player.mu.RLock()
		playerStatuses[player.ID] = PlayerStatus{
			Name:      player.Name,
			Role:      player.Role,
			Connected: player.State == StateConnected,
			Ready:     player.Ready,
			Location:  player.CurrentLocation,
		}
		player.mu.RUnlock()
	}

	// Calculate progress
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

	update := HostUpdate{
		Phase:            gm.state.Phase.String(),
		ConnectedPlayers: gm.playerManager.GetConnectedCount(),
		ReadyPlayers:     gm.playerManager.GetReadyCount(),
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
		Phase:           PhaseSetup,
		Difficulty:      "medium",
		Players:         make(map[string]*Player),
		TeamTokens:      TeamTokens{},
		QuestionHistory: make(map[string]map[string]bool),
		PlayerAnalytics: make(map[string]*PlayerAnalytics),
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
