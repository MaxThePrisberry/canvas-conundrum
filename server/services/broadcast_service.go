package services

import (
	"canvas-conundrum/config"
	"canvas-conundrum/constants"
	"canvas-conundrum/models"
	"canvas-conundrum/utils"
	"log"
	"sync"
	"time"
)

// BroadcastService manages WebSocket message broadcasting
type BroadcastService struct {
	mu              sync.RWMutex
	gridUpdateTimer *time.Timer
	lastGridState   []byte // Cache of last grid state for efficiency
}

// NewBroadcastService creates a new broadcast service
func NewBroadcastService() *BroadcastService {
	return &BroadcastService{}
}

// SendToPlayer sends a message to a specific player
func (bs *BroadcastService) SendToPlayer(player *models.Player, event string, payload interface{}) {
	if player == nil || !player.IsActive {
		return
	}

	msg := utils.NewServerMessage(event, payload)
	data, err := msg.Marshal()
	if err != nil {
		log.Printf("Error marshaling message for player %s: %v", player.ID, err)
		return
	}

	select {
	case player.Send <- data:
	default:
		log.Printf("Player %s send channel full, dropping message", player.ID)
	}
}

// SendToHost sends a message to the host
func (bs *BroadcastService) SendToHost(host *models.Host, event string, payload interface{}) {
	if host == nil || host.Connection == nil {
		return
	}

	msg := utils.NewServerMessage(event, payload)
	data, err := msg.Marshal()
	if err != nil {
		log.Printf("Error marshaling message for host: %v", err)
		return
	}

	select {
	case host.Send <- data:
	default:
		log.Printf("Host send channel full, dropping message")
	}
}

// BroadcastToAllPlayers sends a message to all connected players
func (bs *BroadcastService) BroadcastToAllPlayers(event string, payload interface{}) {
	gameManager := GetGameInstance()
	players := gameManager.GetAllPlayers()

	msg := utils.NewServerMessage(event, payload)
	data, err := msg.Marshal()
	if err != nil {
		log.Printf("Error marshaling broadcast message: %v", err)
		return
	}

	for _, player := range players {
		if player.IsActive {
			select {
			case player.Send <- data:
			default:
				log.Printf("Player %s send channel full, dropping broadcast", player.ID)
			}
		}
	}
}

// BroadcastToAll sends a message to all participants (players and host)
func (bs *BroadcastService) BroadcastToAll(event string, payload interface{}) {
	bs.BroadcastToAllPlayers(event, payload)

	gameManager := GetGameInstance()
	if host := gameManager.GetHost(); host != nil {
		bs.SendToHost(host, event, payload)
	}
}

// BroadcastLobbyStatus broadcasts the current lobby status
func (bs *BroadcastService) BroadcastLobbyStatus() {
	gameManager := GetGameInstance()
	players := gameManager.GetAllPlayers()

	// Count ready players
	readyCount := 0
	connectedCount := 0
	for _, player := range players {
		if player.IsActive {
			connectedCount++
			if player.IsReady {
				readyCount++
			}
		}
	}

	// Get role distribution
	roleDistribution := gameManager.GetRoleDistribution()

	// Create lobby status payload
	payload := map[string]interface{}{
		"currentPlayers": connectedCount + 1, // Include host in total
		"nonHostPlayers": connectedCount,
		"minPlayers":     constants.MinPlayers,
		"maxPlayers":     constants.MaxPlayers,
		"playerRoles": map[string]int{
			"art_enthusiast": roleDistribution[models.RoleArtEnthusiast],
			"detective":      roleDistribution[models.RoleDetective],
			"tourist":        roleDistribution[models.RoleTourist],
			"janitor":        roleDistribution[models.RoleJanitor],
		},
		"hasHost":           gameManager.IsHostConnected(),
		"allPlayersReady":   readyCount >= constants.MinPlayers,
		"readyPlayers":      readyCount,
		"gameStartEligible": gameManager.CanStartGame(),
		"waitingMessage":    bs.getWaitingMessage(readyCount, connectedCount),
	}

	bs.BroadcastToAll(constants.EventSetupToClientLobbyStatus, payload)

	// Send detailed roster to host
	bs.sendHostPlayerRoster()
}

// getWaitingMessage generates appropriate waiting message
func (bs *BroadcastService) getWaitingMessage(readyCount, connectedCount int) string {
	needed := constants.MinPlayers - readyCount
	if needed > 0 {
		return "Waiting for " + string(rune(needed)) + " more player(s) to be ready"
	}
	if !GetGameInstance().IsHostConnected() {
		return "Waiting for host to connect"
	}
	return "Ready to start!"
}

// sendHostPlayerRoster sends detailed player roster to host
func (bs *BroadcastService) sendHostPlayerRoster() {
	gameManager := GetGameInstance()
	host := gameManager.GetHost()
	if host == nil {
		return
	}

	players := gameManager.GetAllPlayers()
	playerStatuses := make(map[string]interface{})

	for id, player := range players {
		playerStatuses[id] = map[string]interface{}{
			"playerName":   player.Name,
			"role":         player.Role,
			"specialties":  player.Specialties,
			"connected":    player.IsActive,
			"ready":        player.IsReady,
			"lastActivity": player.LastSeen.Format(time.RFC3339),
		}
	}

	payload := map[string]interface{}{
		"phase":             gameManager.GetCurrentPhase(),
		"connectedPlayers":  gameManager.GetPlayerCount(),
		"readyPlayers":      bs.countReadyPlayers(),
		"gameStartEligible": gameManager.CanStartGame(),
		"playerStatuses":    playerStatuses,
		"roleDistribution":  gameManager.GetRoleDistribution(),
	}

	bs.SendToHost(host, constants.EventSetupToHostPlayerRoster, payload)
}

// countReadyPlayers counts ready players
func (bs *BroadcastService) countReadyPlayers() int {
	count := 0
	for _, player := range GetGameInstance().GetAllPlayers() {
		if player.IsReady && player.IsActive {
			count++
		}
	}
	return count
}

// BroadcastGameStart broadcasts game start event
func (bs *BroadcastService) BroadcastGameStart() {
	// To players
	playerPayload := map[string]interface{}{
		"nextPhase":           "resource_gathering",
		"transitionCountdown": 5,
		"message":             "Game starting! Prepare for resource gathering phase.",
		"instructions":        "Make your way to the resource gathering stations.",
	}
	bs.BroadcastToAllPlayers(constants.EventSetupToClientGameStarted, playerPayload)

	// To host
	gameManager := GetGameInstance()
	game := gameManager.GetGame()
	hostPayload := map[string]interface{}{
		"phase":        "resource_gathering",
		"gameStarted":  true,
		"totalPlayers": gameManager.GetPlayerCount(),
		"initialTeamTokens": map[string]int{
			"anchorTokens":  game.TeamTokens.AnchorTokens,
			"chronosTokens": game.TeamTokens.ChronosTokens,
			"guideTokens":   game.TeamTokens.GuideTokens,
			"clarityTokens": game.TeamTokens.ClarityTokens,
		},
		"monitoringActive": true,
	}

	if host := gameManager.GetHost(); host != nil {
		bs.SendToHost(host, constants.EventSetupToHostGameStarted, hostPayload)
	}
}

// BroadcastPhaseTransition broadcasts phase transition
func (bs *BroadcastService) BroadcastPhaseTransition(fromPhase, toPhase models.GamePhase) {
	// Common payload
	payload := map[string]interface{}{
		"fromPhase":        fromPhase,
		"toPhase":          toPhase,
		"transitionReason": bs.getTransitionReason(fromPhase, toPhase),
		"countdown":        30,
		"message":          bs.getTransitionMessage(fromPhase, toPhase),
	}

	// Add phase-specific instructions
	if toPhase == models.PhasePuzzleAssembly {
		payload["preparationInstructions"] = []string{
			"Return to the main room",
			"Prepare for collaborative puzzle solving",
			"Individual puzzle segments will be assigned",
		}
		payload["phaseInfo"] = map[string]interface{}{
			"nextPhaseName":        "Puzzle Assembly",
			"nextPhaseDescription": "Solve individual puzzles then collaborate on master assembly",
			"estimatedDuration":    "6-8 minutes",
		}
	}

	bs.BroadcastToAll(constants.EventSystemToClientPhaseTransition, payload)

	// Send host-specific transition info
	bs.sendHostPhaseTransition(fromPhase, toPhase)
}

// getTransitionReason returns reason for phase transition
func (bs *BroadcastService) getTransitionReason(fromPhase, toPhase models.GamePhase) string {
	switch {
	case fromPhase == models.PhaseSetup && toPhase == models.PhaseResourceGathering:
		return "game_started"
	case fromPhase == models.PhaseResourceGathering && toPhase == models.PhasePuzzleAssembly:
		return "resource_phase_completed"
	case fromPhase == models.PhasePuzzleAssembly && toPhase == models.PhaseAnalytics:
		return "puzzle_completed"
	default:
		return "phase_transition"
	}
}

// getTransitionMessage returns message for phase transition
func (bs *BroadcastService) getTransitionMessage(fromPhase, toPhase models.GamePhase) string {
	switch toPhase {
	case models.PhaseResourceGathering:
		return "Starting resource gathering phase!"
	case models.PhasePuzzleAssembly:
		return "Transitioning to puzzle assembly phase in 30 seconds"
	case models.PhaseAnalytics:
		return "Game complete! Viewing analytics..."
	default:
		return "Phase transition in progress..."
	}
}

// sendHostPhaseTransition sends phase transition details to host
func (bs *BroadcastService) sendHostPhaseTransition(fromPhase, toPhase models.GamePhase) {
	gameManager := GetGameInstance()
	host := gameManager.GetHost()
	if host == nil {
		return
	}

	payload := map[string]interface{}{
		"fromPhase":        fromPhase,
		"toPhase":          toPhase,
		"transitionStatus": "confirmed",
		"transitionTime":   30,
		"hostControls": map[string]interface{}{
			"availableActions":      bs.getHostActions(toPhase),
			"monitoringFeatures":    bs.getHostMonitoringFeatures(toPhase),
			"phaseSpecificControls": bs.getHostPhaseControls(toPhase),
		},
		"playerTransitionStatus": map[string]interface{}{
			"playersReady":         gameManager.GetPlayerCount(),
			"playersTransitioning": 0,
			"transitionComplete":   true,
		},
	}

	bs.SendToHost(host, constants.EventSystemToHostPhaseTransition, payload)
}

// getHostActions returns available host actions for a phase
func (bs *BroadcastService) getHostActions(phase models.GamePhase) []string {
	switch phase {
	case models.PhaseSetup:
		return []string{"start_game", "monitor_players"}
	case models.PhaseResourceGathering:
		return []string{"monitor_progress", "view_analytics"}
	case models.PhasePuzzleAssembly:
		return []string{"start_puzzle_timer", "monitor_progress"}
	case models.PhaseAnalytics:
		return []string{"view_reports", "reset_game"}
	default:
		return []string{"monitor"}
	}
}

// getHostMonitoringFeatures returns monitoring features for a phase
func (bs *BroadcastService) getHostMonitoringFeatures(phase models.GamePhase) []string {
	switch phase {
	case models.PhasePuzzleAssembly:
		return []string{"individual_progress", "grid_state", "collaboration_metrics"}
	case models.PhaseResourceGathering:
		return []string{"trivia_performance", "token_collection", "location_tracking"}
	default:
		return []string{"player_status", "game_state"}
	}
}

// getHostPhaseControls returns phase-specific controls
func (bs *BroadcastService) getHostPhaseControls(phase models.GamePhase) []string {
	switch phase {
	case models.PhasePuzzleAssembly:
		return []string{"timer_management", "completion_tracking"}
	case models.PhaseAnalytics:
		return []string{"export_data", "reset_game"}
	default:
		return []string{}
	}
}

// BroadcastResourcePhaseStart broadcasts resource gathering phase start
func (bs *BroadcastService) BroadcastResourcePhaseStart() {
	gameManager := GetGameInstance()
	game := gameManager.GetGame()

	// To players
	playerPayload := map[string]interface{}{
		"phase":        "resource_gathering",
		"totalRounds":  constants.ResourceGatheringRounds,
		"roundDuration": constants.ResourceGatheringRoundDuration,
		"answerTime":    constants.TriviaAnswerTime,
		"graceTime":     constants.TriviaGraceTime,
		"resourceStationHashes": map[string]string{
			"anchor":  config.HashAnchorStation,
			"chronos": config.HashChronosStation,
			"guide":   config.HashGuideStation,
			"clarity": config.HashClarityStation,
		},
		"tokenThresholds": map[string]int{
			"anchor":  constants.AnchorTokenThreshold,
			"chronos": constants.ChronosTokenThreshold,
			"guide":   constants.GuideTokenThreshold,
			"clarity": constants.ClarityTokenThreshold,
		},
		"difficultySettings": map[string]interface{}{
			"mode":                 game.Difficulty,
			"specialtyProbability":  bs.getSpecialtyProbability(game.Difficulty),
			"timeMultiplier":       bs.getTimeMultiplier(game.Difficulty),
			"thresholdMultiplier":  bs.getThresholdMultiplier(game.Difficulty),
		},
	}
	bs.BroadcastToAllPlayers(constants.EventResourceToClientPhaseStart, playerPayload)

	// To host
	hostPayload := map[string]interface{}{
		"phase": "resource_gathering",
		"monitoringDashboard": map[string]interface{}{
			"totalRounds":      constants.ResourceGatheringRounds,
			"currentRound":     0,
			"roundDuration":    constants.ResourceGatheringRoundDuration,
			"playerDistribution": bs.getPlayerDistribution(),
		},
		"analyticsTracking": map[string]interface{}{
			"questionDelivery":   true,
			"answerTracking":     true,
			"locationTracking":   true,
			"performanceMetrics": true,
		},
	}

	if host := gameManager.GetHost(); host != nil {
		bs.SendToHost(host, constants.EventResourceToHostPhaseStart, hostPayload)
	}
}

// getSpecialtyProbability returns specialty probability for difficulty
func (bs *BroadcastService) getSpecialtyProbability(difficulty models.DifficultyMode) float64 {
	switch difficulty {
	case models.DifficultyEasy:
		return constants.EasySpecialtyProbability
	case models.DifficultyHard:
		return constants.HardSpecialtyProbability
	default:
		return constants.MediumSpecialtyProbability
	}
}

// getTimeMultiplier returns time multiplier for difficulty
func (bs *BroadcastService) getTimeMultiplier(difficulty models.DifficultyMode) float64 {
	switch difficulty {
	case models.DifficultyEasy:
		return constants.EasyTimeMultiplier
	case models.DifficultyHard:
		return constants.HardTimeMultiplier
	default:
		return constants.MediumTimeMultiplier
	}
}

// getThresholdMultiplier returns threshold multiplier for difficulty
func (bs *BroadcastService) getThresholdMultiplier(difficulty models.DifficultyMode) float64 {
	switch difficulty {
	case models.DifficultyEasy:
		return constants.EasyThresholdMultiplier
	case models.DifficultyHard:
		return constants.HardThresholdMultiplier
	default:
		return constants.MediumThresholdMultiplier
	}
}

// getPlayerDistribution returns current player distribution by station
func (bs *BroadcastService) getPlayerDistribution() map[string]int {
	dist := map[string]int{
		"anchor":  0,
		"chronos": 0,
		"guide":   0,
		"clarity": 0,
		"unknown": 0,
	}

	for _, player := range GetGameInstance().GetAllPlayers() {
		if player.IsActive {
			if player.CurrentStation == "" {
				dist["unknown"]++
			} else {
				dist[player.CurrentStation]++
			}
		}
	}
	return dist
}

// BroadcastTriviaQuestions sends trivia questions to players
func (bs *BroadcastService) BroadcastTriviaQuestions(questions map[string]*models.TriviaQuestion) {
	gameManager := GetGameInstance()
	game := gameManager.GetGame()

	for playerID, question := range questions {
		player, exists := gameManager.GetPlayer(playerID)
		if !exists || !player.IsActive {
			continue
		}

		payload := map[string]interface{}{
			"questionId":     question.ID,
			"questionText":   question.Question,
			"category":       question.Category,
			"difficulty":     question.Difficulty,
			"isSpecialty":    question.IsSpecialty,
			"specialtyBonus": question.SpecialtyBonus,
			"timeLimit":      constants.TriviaAnswerTime,
			"options":        question.Options,
			"roundNumber":    game.CurrentRound,
			"totalRounds":    constants.ResourceGatheringRounds,
			"answerDeadline": time.Now().Add(time.Duration(constants.TriviaAnswerTime) * time.Second).Format(time.RFC3339),
		}

		bs.SendToPlayer(player, constants.EventResourceToPlayerTriviaQuestion, payload)
	}

	// Update host with round analytics
	bs.sendHostRoundAnalytics()
}

// sendHostRoundAnalytics sends round analytics to host
func (bs *BroadcastService) sendHostRoundAnalytics() {
	gameManager := GetGameInstance()
	host := gameManager.GetHost()
	game := gameManager.GetGame()

	if host == nil {
		return
	}

	playerPerformance := make(map[string]interface{})
	for id, player := range gameManager.GetAllPlayers() {
		if player.IsActive {
			playerPerformance[id] = map[string]interface{}{
				"location":        player.CurrentStation,
				"runningAccuracy": player.GetAccuracy(),
			}
		}
	}

	payload := map[string]interface{}{
		"currentRound":      game.CurrentRound,
		"totalRounds":       constants.ResourceGatheringRounds,
		"playerPerformance": playerPerformance,
		"teamTokens": map[string]int{
			"anchorTokens":  game.TeamTokens.AnchorTokens,
			"chronosTokens": game.TeamTokens.ChronosTokens,
			"guideTokens":   game.TeamTokens.GuideTokens,
			"clarityTokens": game.TeamTokens.ClarityTokens,
		},
	}

	bs.SendToHost(host, constants.EventResourceToHostRoundAnalytics, payload)
}

// BroadcastPuzzlePhaseStart broadcasts puzzle assembly phase start
func (bs *BroadcastService) BroadcastPuzzlePhaseStart() {
	gameManager := GetGameInstance()
	game := gameManager.GetGame()

	// Get grid size and segment assignments
	gridSize := game.PuzzleGrid.Size
	playerSegments := make(map[string]string)
	allSegmentIds := []string{}
	
	for _, player := range gameManager.GetAllPlayers() {
		if player.IsActive && player.AssignedSegment != "" {
			playerSegments[player.ID] = player.AssignedSegment
			allSegmentIds = append(allSegmentIds, player.AssignedSegment)
		}
	}

	// Generate all possible segment IDs for the grid
	for i := 0; i < gridSize*gridSize; i++ {
		row := i / gridSize
		col := i % gridSize
		segmentID := string(rune('A'+row)) + string(rune('1'+col))
		found := false
		for _, id := range allSegmentIds {
			if id == segmentID {
				found = true
				break
			}
		}
		if !found {
			allSegmentIds = append(allSegmentIds, segmentID)
		}
	}

	// To players
	for _, player := range gameManager.GetAllPlayers() {
		if player.IsActive {
			playerPayload := map[string]interface{}{
				"phase":                   "puzzle_assembly",
				"imageId":                 "masterpiece_001",
				"assignedSegmentId":       player.AssignedSegment,
				"individualPuzzleSize":    constants.IndividualPuzzlePieces,
				"anchorPreSolvedPieces":   game.TeamTokens.GetThreshold(models.TokenAnchor) * 2,
				"centralGridSize":         gridSize,
				"totalCentralFragments":   gridSize * gridSize,
				"clarityPreviewDuration":  constants.ClarityBasePreviewTime + game.TeamTokens.GetThreshold(models.TokenClarity),
				"guideHighlightCount":     bs.calculateGuideHighlights(gridSize, game.TeamTokens.GetThreshold(models.TokenGuide)),
				"allSegmentIds":           allSegmentIds,
				"loadInstructions":        "Load your assigned segment and prepare individual puzzle",
				"waitingForHost":          true,
			}
			bs.SendToPlayer(player, constants.EventPuzzleToClientPhaseLoad, playerPayload)
		}
	}

	// To host
	unassignedSegments := []string{}
	for _, segmentID := range allSegmentIds {
		assigned := false
		for _, playerSegment := range playerSegments {
			if playerSegment == segmentID {
				assigned = true
				break
			}
		}
		if !assigned {
			unassignedSegments = append(unassignedSegments, segmentID)
		}
	}

	hostPayload := map[string]interface{}{
		"phase":                      "puzzle_assembly",
		"imageId":                    "masterpiece_001",
		"centralGridSize":            gridSize,
		"totalFragments":             gridSize * gridSize,
		"isHost":                     true,
		"playerCount":                gameManager.GetPlayerCount(),
		"playerSegmentAssignments":   playerSegments,
		"unassignedSegments":         unassignedSegments,
		"bonusEffects": map[string]interface{}{
			"anchorPreSolved":  game.TeamTokens.GetThreshold(models.TokenAnchor) * 2,
			"chronosTimeBonus": game.TeamTokens.GetThreshold(models.TokenChronos) * 20,
			"guideHighlights":  bs.calculateGuideHighlights(gridSize, game.TeamTokens.GetThreshold(models.TokenGuide)),
			"clarityPreview":   constants.ClarityBasePreviewTime + game.TeamTokens.GetThreshold(models.TokenClarity),
		},
		"monitoringActive": true,
		"canStartTimer":    true,
	}

	if host := gameManager.GetHost(); host != nil {
		bs.SendToHost(host, constants.EventPuzzleToHostPhaseLoad, hostPayload)
	}
}

// calculateGuideHighlights calculates number of guide highlights based on threshold
func (bs *BroadcastService) calculateGuideHighlights(gridSize, threshold int) int {
	total := gridSize * gridSize
	removed := 0
	for i := 0; i < threshold; i++ {
		removed += total / 7
	}
	return total - removed
}

// BroadcastPuzzleTimerStart broadcasts puzzle timer start
func (bs *BroadcastService) BroadcastPuzzleTimerStart(totalTime int, previewTime int) {
	gameManager := GetGameInstance()

	// To players
	playerPayload := map[string]interface{}{
		"startTimestamp":       time.Now().Unix(),
		"totalTime":            totalTime,
		"baseTime":             constants.PuzzleBaseTime,
		"chronosBonus":         totalTime - constants.PuzzleBaseTime,
		"clarityPreviewActive": true,
		"previewDuration":      previewTime,
		"instructions":         "Begin solving your individual puzzle segments",
	}
	bs.BroadcastToAllPlayers(constants.EventPuzzleToClientTimerStart, playerPayload)

	// To host
	hostPayload := map[string]interface{}{
		"timerActive":     true,
		"startTimestamp":  time.Now().Unix(),
		"totalTime":       totalTime,
		"baseTime":        constants.PuzzleBaseTime,
		"bonusTime":       totalTime - constants.PuzzleBaseTime,
		"playersInPhase2": gameManager.GetPlayerCount(),
		"playersInPhase3": 0,
	}

	if host := gameManager.GetHost(); host != nil {
		bs.SendToHost(host, constants.EventPuzzleToHostTimerStart, hostPayload)
	}

	// Start periodic grid updates for players
	bs.startPeriodicGridUpdates()
}

// startPeriodicGridUpdates starts periodic grid state updates
func (bs *BroadcastService) startPeriodicGridUpdates() {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	// Cancel existing timer if any
	if bs.gridUpdateTimer != nil {
		bs.gridUpdateTimer.Stop()
	}

	// Create new timer
	bs.gridUpdateTimer = time.NewTimer(constants.GridUpdateIntervalDuration)

	go func() {
		for {
			<-bs.gridUpdateTimer.C
			bs.broadcastGridState()
			bs.gridUpdateTimer.Reset(constants.GridUpdateIntervalDuration)
		}
	}()
}

// broadcastGridState broadcasts current grid state
func (bs *BroadcastService) broadcastGridState() {
	gameManager := GetGameInstance()
	game := gameManager.GetGame()

	if game.PuzzleGrid == nil {
		return
	}

	// Prepare fragment data
	fragments := []map[string]interface{}{}
	for _, fragment := range game.PuzzleGrid.Fragments {
		if fragment.Visible {
			fragments = append(fragments, map[string]interface{}{
				"fragmentId": fragment.ID,
				"segmentId":  fragment.SegmentID,
				"position":   fragment.Position,
				"visible":    fragment.Visible,
			})
		}
	}

	payload := map[string]interface{}{
		"updateType":    "periodic",
		"gridSize":      game.PuzzleGrid.Size,
		"fragments":     fragments,
		"timeRemaining": game.GetPuzzleTimeRemaining(),
	}

	// Send to players
	bs.BroadcastToAllPlayers(constants.EventPuzzleToClientGridState, payload)

	// Send immediate update to host with more details
	bs.sendHostGridState()
}

// sendHostGridState sends detailed grid state to host
func (bs *BroadcastService) sendHostGridState() {
	gameManager := GetGameInstance()
	host := gameManager.GetHost()
	game := gameManager.GetGame()

	if host == nil || game.PuzzleGrid == nil {
		return
	}

	// Prepare detailed fragment data for host
	fragments := []map[string]interface{}{}
	for _, fragment := range game.PuzzleGrid.Fragments {
		fragmentData := map[string]interface{}{
			"fragmentId": fragment.ID,
			"segmentId":  fragment.SegmentID,
			"position":   fragment.Position,
			"visible":    fragment.Visible,
			"lastMoved":  fragment.LastMoved.Format(time.RFC3339),
			"moveCount":  fragment.MoveCount,
		}

		// Add player info if owned
		if fragment.PlayerID != "" {
			if player, exists := gameManager.GetPlayer(fragment.PlayerID); exists {
				fragmentData["playerId"] = fragment.PlayerID
				fragmentData["playerName"] = player.Name
			}
		}

		fragments = append(fragments, fragmentData)
	}

	// Prepare player metrics
	playerMetrics := make(map[string]interface{})
	for id, player := range gameManager.GetAllPlayers() {
		phase := 2
		if player.SegmentCompleted {
			phase = 3
		}

		playerMetrics[id] = map[string]interface{}{
			"phase":            phase,
			"fragmentsOwned":   1, // Each player owns one fragment
			"movesContributed": player.FragmentMoves,
			"lastActivity":     player.LastSeen.Format(time.RFC3339),
		}
	}

	payload := map[string]interface{}{
		"updateType":    "immediate",
		"gridSize":      game.PuzzleGrid.Size,
		"fragments":     fragments,
		"playerMetrics": playerMetrics,
		"timeRemaining": game.GetPuzzleTimeRemaining(),
	}

	bs.SendToHost(host, constants.EventPuzzleToHostGridState, payload)
}

// BroadcastResourcePhaseComplete broadcasts resource gathering phase completion
func (bs *BroadcastService) BroadcastResourcePhaseComplete() {
	gameManager := GetGameInstance()
	game := gameManager.GetGame()

	// Calculate bonus effects from tokens
	anchorThreshold := game.TeamTokens.GetThreshold(models.TokenAnchor)
	chronosThreshold := game.TeamTokens.GetThreshold(models.TokenChronos)
	guideThreshold := game.TeamTokens.GetThreshold(models.TokenGuide)
	clarityThreshold := game.TeamTokens.GetThreshold(models.TokenClarity)

	// To players
	playerPayload := map[string]interface{}{
		"phase":     "resource_gathering",
		"nextPhase": "puzzle_assembly",
		"finalTokenTotals": map[string]int{
			"anchorTokens":  game.TeamTokens.AnchorTokens,
			"chronosTokens": game.TeamTokens.ChronosTokens,
			"guideTokens":   game.TeamTokens.GuideTokens,
			"clarityTokens": game.TeamTokens.ClarityTokens,
		},
		"thresholdAchievements": map[string]int{
			"anchor":  anchorThreshold,
			"chronos": chronosThreshold,
			"guide":   guideThreshold,
			"clarity": clarityThreshold,
		},
		"bonusEffects": map[string]interface{}{
			"preSolvedPieces": anchorThreshold * 2,
			"extraTime":       chronosThreshold * 20,
			"guideHighlights": guideThreshold,
			"previewTime":     constants.ClarityBasePreviewTime + clarityThreshold,
		},
		"transitionInstructions": "Return to the main room for puzzle assembly",
		"transitionCountdown":    30,
	}
	bs.BroadcastToAllPlayers(constants.EventResourceToClientPhaseComplete, playerPayload)

	// To host with analytics
	if host := gameManager.GetHost(); host != nil {
		playerAnalytics := make(map[string]interface{})
		// TODO: Get analytics from analytics service when properly implemented

		hostPayload := map[string]interface{}{
			"phase":     "resource_gathering",
			"completed": true,
			"totalQuestionsAnswered": bs.getTotalQuestionsAnswered(),
			"teamPerformance": map[string]interface{}{
				"overallAccuracy":      bs.getTeamAccuracy(),
				"totalTokensEarned":    game.TeamTokens.AnchorTokens + game.TeamTokens.ChronosTokens + game.TeamTokens.GuideTokens + game.TeamTokens.ClarityTokens,
				"averageResponseTime":  bs.getAverageResponseTime(),
			},
			"finalTokenDistribution": map[string]int{
				"anchorTokens":  game.TeamTokens.AnchorTokens,
				"chronosTokens": game.TeamTokens.ChronosTokens,
				"guideTokens":   game.TeamTokens.GuideTokens,
				"clarityTokens": game.TeamTokens.ClarityTokens,
			},
			"playerAnalytics":     playerAnalytics,
			"readyForPuzzlePhase": true,
		}
		bs.SendToHost(host, constants.EventResourceToHostPhaseComplete, hostPayload)
	}
}

// Helper methods for analytics
func (bs *BroadcastService) getTotalQuestionsAnswered() int {
	total := 0
	for _, player := range GetGameInstance().GetAllPlayers() {
		total += player.QuestionsAnswered
	}
	return total
}

func (bs *BroadcastService) getTeamAccuracy() float64 {
	totalQuestions := 0
	totalCorrect := 0
	for _, player := range GetGameInstance().GetAllPlayers() {
		totalQuestions += player.QuestionsAnswered
		totalCorrect += player.CorrectAnswers
	}
	if totalQuestions == 0 {
		return 0
	}
	return float64(totalCorrect) / float64(totalQuestions)
}

func (bs *BroadcastService) getAverageResponseTime() float64 {
	// This would need to be tracked in analytics service
	return 15.0 // Placeholder
}

// BroadcastSegmentCompleted broadcasts segment completion
func (bs *BroadcastService) BroadcastSegmentCompleted(player *models.Player, fragment *models.Fragment) {
	// To the completing player
	playerPayload := map[string]interface{}{
		"segmentId":           player.AssignedSegment,
		"acknowledged":        true,
		"centralGridPosition": fragment.Position,
		"fragmentId":          fragment.ID,
	}
	bs.SendToPlayer(player, constants.EventPuzzleToPlayerSegmentAcknowledged, playerPayload)

	// To host
	gameManager := GetGameInstance()
	if host := gameManager.GetHost(); host != nil {
		hostPayload := map[string]interface{}{
			"playerId":            player.ID,
			"playerName":          player.Name,
			"segmentId":           player.AssignedSegment,
			"completionTime":      player.SegmentSolveTime,
			"centralGridPosition": fragment.Position,
			"fragmentId":          fragment.ID,
		}
		bs.SendToHost(host, constants.EventPuzzleToHostSegmentCompleted, hostPayload)
	}

	// Update grid state immediately
	bs.sendHostGridState()
}

// BroadcastPuzzleComplete broadcasts puzzle completion
func (bs *BroadcastService) BroadcastPuzzleComplete(success bool, completionTime float64) {
	// Stop grid updates
	bs.mu.Lock()
	if bs.gridUpdateTimer != nil {
		bs.gridUpdateTimer.Stop()
	}
	bs.mu.Unlock()

	gameManager := GetGameInstance()
	game := gameManager.GetGame()

	if success {
		// Success payload
		payload := map[string]interface{}{
			"success":             true,
			"completionTime":      int(completionTime),
			"totalTime":           game.GetTotalPuzzleTime(),
			"timeRemaining":       game.GetPuzzleTimeRemaining(),
			"message":             "Masterpiece restored! Well done!",
			"celebrationDuration": 5,
			"nextPhase":           "analytics",
		}
		bs.BroadcastToAll(constants.EventPuzzleToClientCompletedSuccess, payload)
	} else {
		// Timeout payload
		payload := map[string]interface{}{
			"success":     false,
			"reason":      "time_expired",
			"totalTime":   game.GetTotalPuzzleTime(),
			"timeExpired": true,
			"message":     "Time's up! The masterpiece remains incomplete.",
			"nextPhase":   "analytics",
		}
		bs.BroadcastToAll(constants.EventPuzzleToClientCompletedTimeout, payload)
	}
}

// BroadcastPlayerDisconnected broadcasts player disconnection
func (bs *BroadcastService) BroadcastPlayerDisconnected(playerID, playerName string, fragment *models.Fragment) {
	// To all players
	playerPayload := map[string]interface{}{
		"playerId":   playerID,
		"playerName": playerName,
		"message":    playerName + " has disconnected",
	}
	bs.BroadcastToAllPlayers(constants.EventSystemToClientDisconnectionWarning, playerPayload)

	// To host with more details
	gameManager := GetGameInstance()
	if host := gameManager.GetHost(); host != nil {
		hostPayload := map[string]interface{}{
			"playerId":          playerID,
			"playerName":        playerName,
			"disconnectionTime": time.Now().Format(time.RFC3339),
			"currentPhase":      gameManager.GetCurrentPhase(),
			"gameImpact": map[string]interface{}{
				"fragmentHandling": map[string]interface{}{
					"fragmentId":        fragment.ID,
					"action":            "auto_solved_and_unassigned",
					"newPosition":       fragment.Position,
					"ownershipTransfer": "unassigned",
				},
			},
			"updatedPlayerCount": gameManager.GetPlayerCount(),
		}
		bs.SendToHost(host, constants.EventSystemToHostPlayerDisconnected, hostPayload)
	}
}

// SendError sends an error message
func (bs *BroadcastService) SendError(recipient interface{}, errorCode, message, details string) {
	payload := utils.CreateErrorPayload("error", errorCode, message, details)

	switch r := recipient.(type) {
	case *models.Player:
		bs.SendToPlayer(r, constants.EventSystemToClientError, payload)
	case *models.Host:
		bs.SendToHost(r, constants.EventSystemToHostError, payload)
	}
}
