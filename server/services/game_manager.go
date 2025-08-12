package services

import (
	"canvas-conundrum/constants"
	"canvas-conundrum/models"
	"canvas-conundrum/utils"
	"fmt"
	"log"
	"sync"
	"time"
)

// GameManager manages the single game instance
type GameManager struct {
	mu      sync.RWMutex
	game    *models.Game
	players map[string]*models.Player
	host    *models.Host

	// Services
	broadcastSvc *BroadcastService
	triviaSvc    *TriviaService
	puzzleSvc    *PuzzleService
	analyticsSvc *AnalyticsService

	// Timers
	roundTimer  *utils.Timer
	puzzleTimer *utils.Timer

	// Recommendations
	recommendations map[string]*models.MoveRecommendation
}

// Singleton instance
var (
	gameInstance *GameManager
	once         sync.Once
)

// GetGameInstance returns the singleton game manager instance
func GetGameInstance() *GameManager {
	once.Do(func() {
		gameInstance = &GameManager{
			game:            models.NewGame(),
			players:         make(map[string]*models.Player),
			recommendations: make(map[string]*models.MoveRecommendation),
		}
	})
	return gameInstance
}

// SetTriviaService sets the trivia service
func (gm *GameManager) SetTriviaService(service *TriviaService) {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	gm.triviaSvc = service
}

// SetPuzzleService sets the puzzle service
func (gm *GameManager) SetPuzzleService(service *PuzzleService) {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	gm.puzzleSvc = service
}

// SetBroadcastService sets the broadcast service
func (gm *GameManager) SetBroadcastService(service *BroadcastService) {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	gm.broadcastSvc = service
}

// SetAnalyticsService sets the analytics service
func (gm *GameManager) SetAnalyticsService(service *AnalyticsService) {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	gm.analyticsSvc = service
}

// GetCurrentPhase returns the current game phase
func (gm *GameManager) GetCurrentPhase() string {
	gm.mu.RLock()
	defer gm.mu.RUnlock()
	return string(gm.game.CurrentPhase)
}

// GetPlayerCount returns the number of connected players
func (gm *GameManager) GetPlayerCount() int {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	count := 0
	for _, player := range gm.players {
		if player.IsActive {
			count++
		}
	}
	return count
}

// IsHostConnected returns whether a host is connected
func (gm *GameManager) IsHostConnected() bool {
	gm.mu.RLock()
	defer gm.mu.RUnlock()
	return gm.host != nil && gm.host.Connection != nil
}

// AddPlayer adds a new player to the game or reconnects an existing player
func (gm *GameManager) AddPlayer(player *models.Player) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	// Check if this is a reconnection
	if existingPlayer, exists := gm.players[player.ID]; exists {
		// Player reconnection
		if gm.game.CurrentPhase == models.PhasePuzzleAssembly && !existingPlayer.SegmentCompleted {
			// Cannot reconnect during puzzle phase if segment not completed
			return fmt.Errorf("cannot reconnect during puzzle phase")
		}

		// Update connection and mark as active
		existingPlayer.Connection = player.Connection
		existingPlayer.IsActive = true
		existingPlayer.LastSeen = time.Now()
		log.Printf("Player %s reconnected", player.ID)
		return nil
	}

	// New player joining
	// Check if game is in progress
	if gm.game.CurrentPhase != models.PhaseSetup {
		return fmt.Errorf("cannot join game in progress")
	}

	// Check max players
	if len(gm.players) >= gm.game.MaxPlayers {
		return fmt.Errorf("maximum players reached")
	}

	gm.players[player.ID] = player
	gm.game.PlayerCount = len(gm.players)

	// Initialize analytics for player
	if gm.analyticsSvc != nil {
		gm.analyticsSvc.InitializePlayer(player.ID, player.Name)
	}

	log.Printf("Player %s joined the game", player.ID)
	return nil
}

// RemovePlayer removes a player from the game
func (gm *GameManager) RemovePlayer(playerID string) {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	player, exists := gm.players[playerID]
	if !exists {
		return
	}

	player.IsActive = false

	// Handle disconnection based on phase
	switch gm.game.CurrentPhase {
	case models.PhasePuzzleAssembly:
		// Auto-solve puzzle and make fragment unassigned
		if !player.SegmentCompleted && player.AssignedSegment != "" {
			gm.handlePuzzleDisconnection(player)
		}
	}

	log.Printf("Player %s disconnected", playerID)
}

// handlePuzzleDisconnection handles player disconnection during puzzle phase
func (gm *GameManager) handlePuzzleDisconnection(player *models.Player) {
	// Mark segment as completed
	player.SegmentCompleted = true

	// Add fragment as unassigned
	if gm.game.PuzzleGrid != nil {
		fragment := gm.game.PuzzleGrid.AddFragment(player.AssignedSegment, "")
		player.FragmentID = fragment.ID

		// Broadcast update
		if gm.broadcastSvc != nil {
			gm.broadcastSvc.BroadcastPlayerDisconnected(player.ID, player.Name, fragment)
		}
	}
}

// GetPlayer returns a player by ID
func (gm *GameManager) GetPlayer(playerID string) (*models.Player, bool) {
	gm.mu.RLock()
	defer gm.mu.RUnlock()
	player, exists := gm.players[playerID]
	return player, exists
}

// GetAllPlayers returns all players
func (gm *GameManager) GetAllPlayers() map[string]*models.Player {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	// Return a copy to avoid race conditions
	playersCopy := make(map[string]*models.Player)
	for id, player := range gm.players {
		playersCopy[id] = player
	}
	return playersCopy
}

// SetHost sets the game host or reconnects an existing host
func (gm *GameManager) SetHost(host *models.Host) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	// If we have an existing host with an active connection, reject
	if gm.host != nil && gm.host.Connection != nil {
		return fmt.Errorf("host already connected")
	}

	// If this is a reconnection (same host ID), update the connection
	if gm.host != nil && gm.host.ID == host.ID {
		gm.host.Connection = host.Connection
		gm.host.ConnectedAt = time.Now()
		log.Printf("Host %s reconnected", host.ID)
	} else {
		// New host or replacing disconnected host
		gm.host = host
		log.Printf("Host %s connected", host.ID)
	}

	return nil
}

// RemoveHost disconnects the host (but keeps the host object for reconnection)
func (gm *GameManager) RemoveHost() {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	if gm.host != nil {
		gm.host.Connection = nil
		log.Printf("Host %s disconnected", gm.host.ID)
	}
}

// GetHost returns the current host
func (gm *GameManager) GetHost() *models.Host {
	gm.mu.RLock()
	defer gm.mu.RUnlock()
	return gm.host
}

// UpdatePlayerConfiguration updates a player's role and specialties
func (gm *GameManager) UpdatePlayerConfiguration(playerID string, name string, role models.Role, specialties []string) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	player, exists := gm.players[playerID]
	if !exists {
		return fmt.Errorf("player not found")
	}

	// Validate role availability
	if !gm.isRoleAvailable(role, playerID) {
		return fmt.Errorf("role not available")
	}

	// Validate specialties
	if len(specialties) > constants.MaxSpecialtiesPerPlayer {
		return fmt.Errorf("too many specialties selected")
	}

	// Convert string specialties to TriviaCategory
	categorySpecialties := make([]models.TriviaCategory, len(specialties))
	for i, s := range specialties {
		categorySpecialties[i] = models.TriviaCategory(s)
	}

	player.Name = name
	player.Role = role
	player.Specialties = categorySpecialties
	player.IsReady = true

	log.Printf("Player %s configured: role=%s, specialties=%v", playerID, role, specialties)
	return nil
}

// isRoleAvailable checks if a role is available for selection
func (gm *GameManager) isRoleAvailable(role models.Role, excludePlayerID string) bool {
	maxPerRole := (len(gm.players) + 3) / 4

	count := 0
	for id, player := range gm.players {
		if id != excludePlayerID && player.Role == role {
			count++
		}
	}

	return count < maxPerRole
}

// GetRoleDistribution returns the current role distribution
func (gm *GameManager) GetRoleDistribution() map[models.Role]int {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	distribution := map[models.Role]int{
		models.RoleArtEnthusiast: 0,
		models.RoleDetective:     0,
		models.RoleTourist:       0,
		models.RoleJanitor:       0,
	}

	for _, player := range gm.players {
		if player.Role != models.RoleNone {
			distribution[player.Role]++
		}
	}

	return distribution
}

// CanStartGame checks if the game can be started
func (gm *GameManager) CanStartGame() bool {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	// Check minimum players
	readyCount := 0
	for _, player := range gm.players {
		if player.IsReady && player.IsActive {
			readyCount++
		}
	}

	return readyCount >= gm.game.MinPlayers && gm.host != nil && gm.host.Connection != nil
}

// StartGame starts the game
func (gm *GameManager) StartGame() error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	if gm.game.CurrentPhase != models.PhaseSetup {
		return fmt.Errorf("game already started")
	}

	// Check if we can start the game
	readyCount := 0
	for _, player := range gm.players {
		if player.IsReady && player.IsActive {
			readyCount++
		}
	}

	if readyCount < gm.game.MinPlayers {
		return fmt.Errorf("cannot start game: need minimum %d ready players, have %d", gm.game.MinPlayers, readyCount)
	}

	if gm.host == nil || gm.host.Connection == nil {
		return fmt.Errorf("cannot start game: no host connected")
	}

	// Initialize analytics
	if gm.analyticsSvc != nil {
		gm.analyticsSvc.StartGame(gm.game.ID)
	}

	// Transition to resource gathering
	gm.game.StartResourceGathering()

	log.Println("Game started - transitioning to resource gathering phase")

	// Start first round after transition
	go func() {
		time.Sleep(5 * time.Second) // Give players time to transition
		gm.StartResourceRound()
	}()

	return nil
}

// StartResourceRound starts a new resource gathering round
func (gm *GameManager) StartResourceRound() {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	if gm.game.CurrentPhase != models.PhaseResourceGathering {
		return
	}

	gm.game.StartNextRound()

	log.Printf("Starting resource round %d/%d", gm.game.CurrentRound, constants.ResourceGatheringRounds)

	// Deliver trivia questions
	if gm.triviaSvc != nil && gm.broadcastSvc != nil {
		questions := gm.triviaSvc.GetQuestionsForRound(gm.players)
		gm.broadcastSvc.BroadcastTriviaQuestions(questions)
	}

	// Set timer for next round
	if gm.game.CurrentRound < constants.ResourceGatheringRounds {
		gm.roundTimer = utils.NewTimer(
			time.Duration(constants.ResourceGatheringRoundDuration)*time.Second,
			func() {
				gm.StartResourceRound()
			},
		)
		gm.roundTimer.Start()
	} else {
		// End resource gathering phase
		go func() {
			time.Sleep(time.Duration(constants.ResourceGatheringRoundDuration) * time.Second)
			gm.EndResourceGathering()
		}()
	}
}

// EndResourceGathering ends the resource gathering phase
func (gm *GameManager) EndResourceGathering() {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	log.Println("Resource gathering phase complete")

	// Start puzzle phase
	playerCount := gm.GetPlayerCount()
	gm.game.StartPuzzlePhase(playerCount)

	// Assign puzzle segments
	if gm.puzzleSvc != nil {
		gm.puzzleSvc.AssignSegments(gm.players, gm.game.PuzzleGrid.Size)
	}

	// Broadcast phase transition
	if gm.broadcastSvc != nil {
		gm.broadcastSvc.BroadcastPhaseTransition(models.PhaseResourceGathering, models.PhasePuzzleAssembly)
	}
}

// StartPuzzleTimer starts the puzzle phase timer
func (gm *GameManager) StartPuzzleTimer() error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	if gm.game.CurrentPhase != models.PhasePuzzleAssembly {
		return fmt.Errorf("not in puzzle phase")
	}

	if gm.game.PuzzleTimerStarted {
		return fmt.Errorf("timer already started")
	}

	gm.game.StartPuzzleTimer()

	totalTime := gm.game.GetTotalPuzzleTime()
	gm.puzzleTimer = utils.NewTimer(
		time.Duration(totalTime)*time.Second,
		func() {
			gm.PuzzleTimeout()
		},
	)
	gm.puzzleTimer.Start()

	log.Printf("Puzzle timer started - %d seconds", totalTime)

	// Broadcast timer start
	if gm.broadcastSvc != nil {
		gm.broadcastSvc.BroadcastPuzzleTimerStart(totalTime, gm.game.GetClarityPreviewTime())
	}

	return nil
}

// CompleteSegment marks a player's segment as completed
func (gm *GameManager) CompleteSegment(playerID string, segmentID string) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	player, exists := gm.players[playerID]
	if !exists {
		return fmt.Errorf("player not found")
	}

	if player.SegmentCompleted {
		return fmt.Errorf("segment already completed")
	}

	// Mark as completed
	player.SegmentCompleted = true
	player.SegmentSolveTime = time.Since(gm.game.PuzzleStartTime).Seconds()

	// Add fragment to grid
	fragment := gm.game.PuzzleGrid.AddFragment(segmentID, playerID)
	player.FragmentID = fragment.ID

	log.Printf("Player %s completed segment %s", playerID, segmentID)

	// Check if all players completed
	allCompleted := true
	for _, p := range gm.players {
		if p.IsActive && !p.SegmentCompleted {
			allCompleted = false
			break
		}
	}

	// Broadcast updates
	if gm.broadcastSvc != nil {
		gm.broadcastSvc.BroadcastSegmentCompleted(player, fragment)

		if allCompleted {
			// Check for immediate victory
			if gm.game.PuzzleGrid.CheckCompletion() {
				gm.PuzzleComplete(true)
			}
		}
	}

	return nil
}

// MoveFragment handles a fragment move request
func (gm *GameManager) MoveFragment(playerID string, fragmentID string, targetPos models.Position) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	player, exists := gm.players[playerID]
	if !exists {
		return fmt.Errorf("player not found")
	}

	// Check cooldown
	if !player.CanMoveFragment(constants.FragmentMoveCooldown) {
		return fmt.Errorf("move on cooldown")
	}

	// Check ownership
	fragment, exists := gm.game.PuzzleGrid.Fragments[fragmentID]
	if !exists {
		return fmt.Errorf("fragment not found")
	}

	if fragment.IsOwned() && fragment.PlayerID != playerID {
		return fmt.Errorf("cannot move another player's fragment")
	}

	// Check if position is occupied
	targetFragment := gm.game.PuzzleGrid.GetFragmentAt(targetPos)

	var err error
	if targetFragment != nil {
		// Swap fragments
		err = gm.game.PuzzleGrid.SwapFragments(fragmentID, targetFragment.ID)
	} else {
		// Move to empty position
		err = gm.game.PuzzleGrid.MoveFragment(fragmentID, targetPos)
	}

	if err != nil {
		return err
	}

	player.UpdateLastMove()

	// Check for completion
	if gm.game.PuzzleGrid.CheckCompletion() {
		gm.PuzzleComplete(true)
	}

	return nil
}

// PuzzleComplete handles puzzle completion
func (gm *GameManager) PuzzleComplete(success bool) {
	gm.game.CompleteGame(success)

	if gm.puzzleTimer != nil {
		gm.puzzleTimer.Stop()
	}

	log.Printf("Puzzle phase complete - success: %v", success)

	// Calculate analytics
	if gm.analyticsSvc != nil {
		gm.analyticsSvc.FinalizeGame(gm.game, gm.players, success)
	}

	// Broadcast completion
	if gm.broadcastSvc != nil {
		gm.broadcastSvc.BroadcastPuzzleComplete(success, gm.game.CompletionTime)
	}
}

// PuzzleTimeout handles puzzle timeout
func (gm *GameManager) PuzzleTimeout() {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	if gm.game.PuzzleCompleted {
		return
	}

	gm.PuzzleComplete(false)
}

// ResetGame resets the game to initial state
func (gm *GameManager) ResetGame() {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	// Stop timers
	if gm.roundTimer != nil {
		gm.roundTimer.Stop()
	}
	if gm.puzzleTimer != nil {
		gm.puzzleTimer.Stop()
	}

	// Reset game state
	gm.game.Reset()

	// Clear players
	for _, player := range gm.players {
		if player.Done != nil {
			close(player.Done)
		}
	}
	gm.players = make(map[string]*models.Player)

	// Clear recommendations
	gm.recommendations = make(map[string]*models.MoveRecommendation)

	// Reset analytics
	if gm.analyticsSvc != nil {
		gm.analyticsSvc.Reset()
	}

	log.Println("Game reset to initial state")
}

// GetGame returns the current game state
func (gm *GameManager) GetGame() *models.Game {
	gm.mu.RLock()
	defer gm.mu.RUnlock()
	return gm.game
}

// GetCurrentRound returns the current round number
func (gm *GameManager) GetCurrentRound() int {
	gm.mu.RLock()
	defer gm.mu.RUnlock()
	return gm.game.CurrentRound
}

// NextRound advances to the next round
func (gm *GameManager) NextRound() {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	gm.game.StartNextRound()
}

// TransitionToPhase transitions to a specific phase
func (gm *GameManager) TransitionToPhase(phase models.GamePhase) {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	gm.game.CurrentPhase = phase
	gm.game.PhaseStartTime = time.Now()
}

// IsGameStarted checks if the game has started
func (gm *GameManager) IsGameStarted() bool {
	gm.mu.RLock()
	defer gm.mu.RUnlock()
	return gm.game.GameStarted
}

// AddTeamTokens adds tokens to the team's total
func (gm *GameManager) AddTeamTokens(tokenType models.TokenType, amount int) {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	gm.game.TeamTokens.AddTokens(tokenType, amount)
}

// GetTeamTokens returns the team's current token counts
func (gm *GameManager) GetTeamTokens() *models.TeamTokens {
	gm.mu.RLock()
	defer gm.mu.RUnlock()
	return gm.game.TeamTokens
}

// GetBroadcastService returns the broadcast service
func (gm *GameManager) GetBroadcastService() *BroadcastService {
	gm.mu.RLock()
	defer gm.mu.RUnlock()
	return gm.broadcastSvc
}

// GetTriviaService returns the trivia service
func (gm *GameManager) GetTriviaService() *TriviaService {
	gm.mu.RLock()
	defer gm.mu.RUnlock()
	return gm.triviaSvc
}

// GetPuzzleService returns the puzzle service
func (gm *GameManager) GetPuzzleService() *PuzzleService {
	gm.mu.RLock()
	defer gm.mu.RUnlock()
	return gm.puzzleSvc
}

// GetAnalyticsService returns the analytics service
func (gm *GameManager) GetAnalyticsService() *AnalyticsService {
	gm.mu.RLock()
	defer gm.mu.RUnlock()
	return gm.analyticsSvc
}
