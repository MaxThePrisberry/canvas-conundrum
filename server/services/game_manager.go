package services

import (
	"canvas-conundrum/config"
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
	singletonMu  sync.Mutex // Protects singleton reset operations
)

// GetGameInstance returns the singleton game manager instance
func GetGameInstance() *GameManager {
	singletonMu.Lock()
	defer singletonMu.Unlock()

	once.Do(func() {
		gameInstance = &GameManager{
			game:            models.NewGame(),
			players:         make(map[string]*models.Player),
			recommendations: make(map[string]*models.MoveRecommendation),
		}
	})
	return gameInstance
}

// ResetGameManagerInstance resets the singleton instance (for testing)
func ResetGameManagerInstance() {
	singletonMu.Lock()
	defer singletonMu.Unlock()

	if gameInstance != nil {
		gameInstance.Cleanup()
	}
	gameInstance = nil
	once = sync.Once{}
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
func (gm *GameManager) GetCurrentPhase() models.GamePhase {
	gm.mu.RLock()
	defer gm.mu.RUnlock()
	return gm.game.CurrentPhase
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
func (gm *GameManager) AddPlayer(player *models.Player) (bool, error) {
	var shouldBroadcast bool
	var shouldBroadcastRoles bool
	var broadcastSvc *BroadcastService
	var beforeAvailability map[models.Role]bool
	var isReconnection bool

	gm.mu.Lock()

	// Capture role availability before adding player (within lock to avoid deadlock)
	beforeAvailability = gm.getRoleAvailabilityMapInline()
	// Check if this is a reconnection
	if existingPlayer, exists := gm.players[player.ID]; exists {
		// Player reconnection
		// Note: Puzzle phase connections are blocked at HTTP level in HandlePlayerWebSocket
		// so we don't need to check that here anymore

		// Update connection and mark as active
		existingPlayer.Connection = player.Connection
		existingPlayer.IsActive = true
		existingPlayer.LastSeen = time.Now()
		existingPlayer.Send = player.Send
		existingPlayer.Done = player.Done
		isReconnection = true
		log.Printf("Player %s reconnected", player.ID)

		// Phase-specific reconnection handling
		if gm.game.CurrentPhase == models.PhaseSetup {
			// Setup Phase: Check role revalidation
			if existingPlayer.Role != "" {
				// Check if previously selected role is still available
				if !gm.isRoleAvailable(existingPlayer.Role, existingPlayer.ID) {
					log.Printf("Player %s role %s no longer available, will need to reselect",
						player.ID, existingPlayer.Role)
					// Clear role selection - player will need to select a new role
					existingPlayer.Role = ""
					existingPlayer.IsReady = false
				} else {
					log.Printf("Player %s role %s still available, restoring ready state",
						player.ID, existingPlayer.Role)
					// Role is available, player can be ready if they were ready before
					// Note: Player.IsReady is preserved from before disconnection
				}
			}
		}

		// Set up broadcast for reconnection
		if gm.broadcastSvc != nil && gm.host != nil && gm.host.Connection != nil {
			shouldBroadcast = true
			broadcastSvc = gm.broadcastSvc
		}
		gm.mu.Unlock()

		// Send roster update outside of lock for reconnection
		if shouldBroadcast {
			log.Printf("Sending roster update to host after player %s reconnected", player.ID)
			broadcastSvc.BroadcastLobbyStatus()
		}
		return isReconnection, nil
	}

	// New player joining
	// Check if game is in progress
	if gm.game.CurrentPhase != models.PhaseSetup {
		gm.mu.Unlock()
		return false, fmt.Errorf("cannot join game in progress")
	}

	// Check max players
	if len(gm.players) >= gm.game.MaxPlayers {
		gm.mu.Unlock()
		return false, fmt.Errorf("maximum players reached")
	}

	gm.players[player.ID] = player
	gm.game.PlayerCount = len(gm.players)

	// Initialize analytics for player
	if gm.analyticsSvc != nil {
		gm.analyticsSvc.InitializePlayer(player.ID, player.Name)
	}

	log.Printf("Player %s joined the game", player.ID)

	// Set up broadcast for new player
	if gm.broadcastSvc != nil && gm.host != nil && gm.host.Connection != nil {
		shouldBroadcast = true
		broadcastSvc = gm.broadcastSvc
	}

	// Check if role availability changed after adding player (within lock to avoid deadlock)
	afterAvailability := gm.getRoleAvailabilityMapInline()
	shouldBroadcastRoles = gm.CheckRoleAvailabilityChanged(beforeAvailability, afterAvailability)

	gm.mu.Unlock()

	// Send updated roster to host outside of lock to avoid deadlock
	if shouldBroadcast {
		log.Printf("Sending roster update to host after player %s joined", player.ID)
		broadcastSvc.BroadcastLobbyStatus()
	}

	// Broadcast role availability changes if any occurred
	if shouldBroadcastRoles && broadcastSvc != nil {
		log.Printf("Broadcasting role availability changes after player %s joined", player.ID)
		broadcastSvc.BroadcastRoleAvailability()
	}

	return false, nil
}

// RemovePlayer removes a player from the game
func (gm *GameManager) RemovePlayer(playerID string) {
	var shouldBroadcast bool
	var shouldBroadcastRoles bool
	var broadcastSvc *BroadcastService
	var beforeAvailability map[models.Role]bool
	var currentPhase models.GamePhase

	gm.mu.Lock()
	player, exists := gm.players[playerID]
	if !exists {
		gm.mu.Unlock()
		log.Printf("RemovePlayer: Player %s not found", playerID)
		return
	}

	// Capture role availability before removing player (within lock to avoid deadlock)
	beforeAvailability = gm.getRoleAvailabilityMapInline()

	log.Printf("RemovePlayer: Removing player %s (name: %s)", playerID, player.Name)

	// Phase-specific disconnection handling
	var updatedPlayerCount int
	var updatedCounts map[string]interface{}

	if gm.game.CurrentPhase == models.PhaseSetup {
		// Setup Phase: Remove from all counts, preserve data for reconnection
		player.IsActive = false

		// Calculate updated counts for setup phase
		connectedCount := 0
		readyCount := 0
		roleDistribution := make(map[string]int)

		for _, p := range gm.players {
			if p.IsActive {
				connectedCount++
				if p.IsReady {
					readyCount++
				}
				if p.Role != "" {
					roleDistribution[string(p.Role)]++
				}
			}
		}

		updatedPlayerCount = connectedCount
		updatedCounts = map[string]interface{}{
			"connectedPlayers": connectedCount,
			"readyPlayers":     readyCount,
			"roleDistribution": roleDistribution,
		}

		log.Printf("RemovePlayer: Setup phase - player removed from counts. Connected: %d, Ready: %d",
			connectedCount, readyCount)
	} else {
		// Post-Setup Phases: Keep in counts, mark as inactive
		player.IsActive = false

		// For post-setup phases, player count includes all players that joined the game
		// They remain "in the game" even when disconnected
		updatedPlayerCount = len(gm.players)

		log.Printf("RemovePlayer: Post-setup phase - player maintained in game. Total players: %d", updatedPlayerCount)
	}

	// Notify host of disconnection (in all phases)
	if gm.broadcastSvc != nil && gm.host != nil {
		log.Printf("RemovePlayer: Sending disconnection notification to host for player %s", playerID)

		// Wrap in defer/recover to catch any panics
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("RemovePlayer: Panic while sending notification: %v", r)
				}
			}()

			// Send disconnection notification to host with phase-appropriate payload
			hostPayload := map[string]interface{}{
				"playerId":          playerID,
				"playerName":        player.Name,
				"disconnectionTime": time.Now().Format(time.RFC3339),
				"currentPhase":      string(gm.game.CurrentPhase),
			}

			// Add phase-specific information
			switch gm.game.CurrentPhase {
			case models.PhaseSetup:
				if updatedCounts != nil {
					hostPayload["updatedCounts"] = updatedCounts
				}
			case models.PhasePuzzleAssembly:
				if player.FragmentID != "" {
					hostPayload["fragmentHandling"] = map[string]interface{}{
						"fragmentId":    player.FragmentID,
						"newPosition":   map[string]interface{}{"x": 0, "y": 0}, // Would be set by grid logic
						"nowUnassigned": true,
					}
				}
				hostPayload["updatedPlayerCount"] = updatedPlayerCount
			default:
				// Resource gathering, Analytics phases
				hostPayload["updatedPlayerCount"] = updatedPlayerCount
			}

			gm.broadcastSvc.SendToHost(gm.host, config.EventSystemToHostPlayerDisconnected, hostPayload)
			log.Printf("RemovePlayer: Disconnection notification sent")
		}()
	} else {
		log.Printf("RemovePlayer: Cannot send disconnection notification - broadcastSvc=%v, host=%v",
			gm.broadcastSvc != nil, gm.host != nil)
	}

	// Capture variables for puzzle disconnection broadcast outside lock
	var puzzleFragment *models.Fragment
	var shouldBroadcastPuzzleDisconnection bool

	// Handle phase-specific disconnection logic
	switch gm.game.CurrentPhase {
	case models.PhasePuzzleAssembly:
		// Auto-solve puzzle and make fragment unassigned
		if !player.SegmentCompleted && player.AssignedSegment != "" {
			// Mark segment as completed
			player.SegmentCompleted = true

			// Add fragment as unassigned
			if gm.game.PuzzleGrid != nil {
				fragment := gm.game.PuzzleGrid.AddFragment(player.AssignedSegment, "")
				player.FragmentID = fragment.ID
				puzzleFragment = fragment
				shouldBroadcastPuzzleDisconnection = true
			}
		}
	}

	// Capture current phase for use after mutex unlock
	currentPhase = gm.game.CurrentPhase

	// Check if we should broadcast roster update (only during setup phase)
	if currentPhase == models.PhaseSetup && gm.broadcastSvc != nil && gm.host != nil && gm.host.Connection != nil {
		shouldBroadcast = true
		broadcastSvc = gm.broadcastSvc
	}

	// Capture role availability after removing player (within lock to avoid deadlock)
	afterAvailability := gm.getRoleAvailabilityMapInline()
	shouldBroadcastRoles = gm.CheckRoleAvailabilityChanged(beforeAvailability, afterAvailability)

	log.Printf("Player %s disconnected", playerID)
	gm.mu.Unlock()

	// Send updated roster to host outside of lock to avoid deadlock
	if shouldBroadcast {
		log.Printf("Sending roster update to host after player %s disconnected", playerID)
		broadcastSvc.BroadcastLobbyStatus()
	}

	// Broadcast puzzle disconnection outside of lock to avoid deadlock
	if shouldBroadcastPuzzleDisconnection && gm.broadcastSvc != nil && puzzleFragment != nil {
		gm.broadcastSvc.BroadcastPlayerDisconnected(player.ID, player.Name, puzzleFragment)
	}

	// Broadcast role availability changes if any occurred (only during setup phase)
	if shouldBroadcastRoles && currentPhase == models.PhaseSetup && broadcastSvc != nil {
		log.Printf("Broadcasting role availability changes after player %s disconnected", playerID)
		broadcastSvc.BroadcastRoleAvailability()
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

	// Return deep copies of players to avoid race conditions
	playersCopy := make(map[string]*models.Player)
	for id, player := range gm.players {
		// Create a deep copy of the player
		playerCopy := *player
		playersCopy[id] = &playerCopy
	}
	return playersCopy
}

// SetHost sets the game host or reconnects an existing host
func (gm *GameManager) SetHost(host *models.Host) (bool, error) {
	var wasReconnection bool
	var broadcastSvc *BroadcastService

	gm.mu.Lock()
	// If we have an existing host with an active connection, reject
	if gm.host != nil && gm.host.Connection != nil {
		gm.mu.Unlock()
		return false, fmt.Errorf("host already connected")
	}

	// If this is a reconnection (same host ID), update the connection
	if gm.host != nil && gm.host.ID == host.ID {
		gm.host.Connection = host.Connection
		gm.host.ConnectedAt = time.Now()
		wasReconnection = true
		log.Printf("Host %s reconnected", host.ID)
	} else {
		// New host or replacing disconnected host
		gm.host = host
		log.Printf("Host %s connected", host.ID)
	}

	// Store reference to broadcast service before unlocking
	broadcastSvc = gm.broadcastSvc
	gm.mu.Unlock()

	// Broadcast host reconnection outside the lock to avoid deadlock
	if wasReconnection && broadcastSvc != nil {
		broadcastSvc.BroadcastHostReconnected()
	}

	return wasReconnection, nil
}

// RemoveHost disconnects the host (but keeps the host object for reconnection)
func (gm *GameManager) RemoveHost() {
	// Store what we need before unlocking
	var broadcastSvc *BroadcastService
	var hostID string

	gm.mu.Lock()
	if gm.host != nil {
		gm.host.SetConnection(nil)
		hostID = gm.host.ID
		broadcastSvc = gm.broadcastSvc
	}
	gm.mu.Unlock()

	if hostID != "" {
		log.Printf("Host %s disconnected", hostID)

		// Broadcast host disconnection to all players (outside lock)
		if broadcastSvc != nil {
			broadcastSvc.BroadcastHostDisconnected()
		}
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

	// Capture role availability before making changes (within lock to avoid deadlock)
	beforeAvailability := gm.getRoleAvailabilityMapInline()
	player, exists := gm.players[playerID]
	if !exists {
		gm.mu.Unlock()
		return fmt.Errorf("player not found")
	}

	// Validate role availability
	if !gm.isRoleAvailable(role, playerID) {
		gm.mu.Unlock()
		return fmt.Errorf("role not available")
	}

	// Validate specialties
	if len(specialties) > config.MaxSpecialtiesPerPlayer {
		gm.mu.Unlock()
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

	// Check if role availability changed after the update (within lock to avoid deadlock)
	afterAvailability := gm.getRoleAvailabilityMapInline()
	shouldBroadcast := gm.CheckRoleAvailabilityChanged(beforeAvailability, afterAvailability)

	broadcastSvc := gm.broadcastSvc
	gm.mu.Unlock()

	// Broadcast role availability changes if any occurred
	if shouldBroadcast && broadcastSvc != nil {
		broadcastSvc.BroadcastRoleAvailability()
	}

	return nil
}

// isRoleAvailable checks if a role is available for selection
func (gm *GameManager) isRoleAvailable(role models.Role, excludePlayerID string) bool {
	// Calculate maxPerRole based on active players during setup, all players otherwise
	var playerCount int
	if gm.game.CurrentPhase == models.PhaseSetup {
		playerCount = 0
		for _, player := range gm.players {
			if player.IsActive {
				playerCount++
			}
		}
	} else {
		playerCount = len(gm.players)
	}

	maxPerRole := (playerCount + 3) / 4
	if maxPerRole < 1 {
		maxPerRole = 1 // Ensure at least 1 player can select each role
	}

	count := 0
	for id, player := range gm.players {
		if id != excludePlayerID && player.Role == role {
			// During setup phase, only count active players
			// During other phases, count all players (including disconnected ones)
			if gm.game.CurrentPhase == models.PhaseSetup {
				if player.IsActive {
					count++
				}
			} else {
				count++
			}
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
			// During setup phase, only count active players
			// During other phases, count all players (including disconnected ones)
			if gm.game.CurrentPhase == models.PhaseSetup {
				if player.IsActive {
					distribution[player.Role]++
				}
			} else {
				distribution[player.Role]++
			}
		}
	}

	return distribution
}

// GetRoleAvailabilityMap returns the availability status of all roles
func (gm *GameManager) GetRoleAvailabilityMap() map[models.Role]bool {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	// Calculate distribution inline to avoid double locking
	distribution := map[models.Role]int{
		models.RoleArtEnthusiast: 0,
		models.RoleDetective:     0,
		models.RoleTourist:       0,
		models.RoleJanitor:       0,
	}

	for _, player := range gm.players {
		if player.Role != models.RoleNone {
			// During setup phase, only count active players
			// During other phases, count all players (including disconnected ones)
			if gm.game.CurrentPhase == models.PhaseSetup {
				if player.IsActive {
					distribution[player.Role]++
				}
			} else {
				distribution[player.Role]++
			}
		}
	}

	// Calculate maxPerRole based on active players during setup, all players otherwise
	var playerCount int
	if gm.game.CurrentPhase == models.PhaseSetup {
		playerCount = 0
		for _, player := range gm.players {
			if player.IsActive {
				playerCount++
			}
		}
	} else {
		playerCount = len(gm.players)
	}

	maxPerRole := (playerCount + 3) / 4
	if maxPerRole < 1 {
		maxPerRole = 1 // Ensure at least 1 player can select each role
	}

	return map[models.Role]bool{
		models.RoleArtEnthusiast: distribution[models.RoleArtEnthusiast] < maxPerRole,
		models.RoleDetective:     distribution[models.RoleDetective] < maxPerRole,
		models.RoleTourist:       distribution[models.RoleTourist] < maxPerRole,
		models.RoleJanitor:       distribution[models.RoleJanitor] < maxPerRole,
	}
}

// getRoleAvailabilityMapInline calculates role availability without acquiring additional locks
// This assumes the caller already holds the appropriate lock
func (gm *GameManager) getRoleAvailabilityMapInline() map[models.Role]bool {
	// Calculate distribution inline to avoid double locking
	distribution := map[models.Role]int{
		models.RoleArtEnthusiast: 0,
		models.RoleDetective:     0,
		models.RoleTourist:       0,
		models.RoleJanitor:       0,
	}

	for _, player := range gm.players {
		if player.Role != models.RoleNone {
			// During setup phase, only count active players
			// During other phases, count all players (including disconnected ones)
			if gm.game.CurrentPhase == models.PhaseSetup {
				if player.IsActive {
					distribution[player.Role]++
				}
			} else {
				distribution[player.Role]++
			}
		}
	}

	// Calculate maxPerRole based on active players during setup, all players otherwise
	var playerCount int
	if gm.game.CurrentPhase == models.PhaseSetup {
		playerCount = 0
		for _, player := range gm.players {
			if player.IsActive {
				playerCount++
			}
		}
	} else {
		playerCount = len(gm.players)
	}

	maxPerRole := (playerCount + 3) / 4
	if maxPerRole < 1 {
		maxPerRole = 1 // Ensure at least 1 player can select each role
	}

	return map[models.Role]bool{
		models.RoleArtEnthusiast: distribution[models.RoleArtEnthusiast] < maxPerRole,
		models.RoleDetective:     distribution[models.RoleDetective] < maxPerRole,
		models.RoleTourist:       distribution[models.RoleTourist] < maxPerRole,
		models.RoleJanitor:       distribution[models.RoleJanitor] < maxPerRole,
	}
}

// CheckRoleAvailabilityChanged compares two role availability maps and returns true if any changed
func (gm *GameManager) CheckRoleAvailabilityChanged(before, after map[models.Role]bool) bool {
	roles := []models.Role{
		models.RoleArtEnthusiast,
		models.RoleDetective,
		models.RoleTourist,
		models.RoleJanitor,
	}

	for _, role := range roles {
		if before[role] != after[role] {
			return true
		}
	}
	return false
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
func (gm *GameManager) StartGame(difficulty string) error {
	// Keep track of what to broadcast after unlocking
	var shouldBroadcast bool

	// Do all state changes under lock
	err := func() error {
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
			// Initialize analytics for all existing players
			for _, player := range gm.players {
				if player.IsActive {
					gm.analyticsSvc.InitializePlayer(player.ID, player.Name)
				}
			}
		}

		// Check if we can broadcast
		shouldBroadcast = gm.broadcastSvc != nil

		// Set the game difficulty
		switch difficulty {
		case "easy":
			gm.game.Difficulty = models.DifficultyEasy
		case "hard":
			gm.game.Difficulty = models.DifficultyHard
		default:
			gm.game.Difficulty = models.DifficultyMedium
		}

		// Transition to resource gathering
		gm.game.StartResourceGathering()

		log.Println("Game started - transitioning to resource gathering phase")
		return nil
	}()

	if err != nil {
		return err
	}

	// Send game started confirmation to host first
	if shouldBroadcast {
		gm.sendGameStartedToHost()
	}

	// Broadcast phase transition after releasing lock
	if shouldBroadcast {
		gm.broadcastSvc.BroadcastResourcePhaseStart()
	}

	// Start first round after transition
	go func() {
		// Wait one full round duration to give players time to move to resource stations and scan QR codes
		time.Sleep(time.Duration(config.ResourceGatheringRoundDuration) * time.Second)
		gm.StartResourceRound()
	}()

	return nil
}

// sendGameStartedToHost sends SETUP_TO_HOST_GAME_STARTED event
func (gm *GameManager) sendGameStartedToHost() {
	gm.mu.RLock()
	host := gm.host

	// Count active players
	totalPlayers := 0
	for _, player := range gm.players {
		if player.IsActive {
			totalPlayers++
		}
	}
	gm.mu.RUnlock()

	if host != nil && host.Connection != nil {
		payload := map[string]interface{}{
			"phase":        "resource_gathering",
			"totalPlayers": totalPlayers,
			"initialTeamTokens": map[string]int{
				"anchorTokens":  0,
				"chronosTokens": 0,
				"guideTokens":   0,
				"clarityTokens": 0,
			},
		}

		if gm.broadcastSvc != nil {
			gm.broadcastSvc.SendToHost(host, config.EventSetupToHostGameStarted, payload)
		}
	}
}

// StartResourceRound starts a new resource gathering round
func (gm *GameManager) StartResourceRound() {
	// Add a small delay to reduce lock contention during rapid phase transitions
	time.Sleep(50 * time.Millisecond)

	gm.mu.Lock()

	if gm.game.CurrentPhase != models.PhaseResourceGathering {
		gm.mu.Unlock()
		return
	}

	gm.game.StartNextRound()

	log.Printf("Starting resource round %d/%d", gm.game.CurrentRound, config.ResourceGatheringRounds)

	// Store references for calls outside lock
	// CRITICAL: Create a defensive copy of the players map to avoid race conditions
	// when BroadcastTriviaQuestions iterates over it without mutex protection
	players := make(map[string]*models.Player)
	for id, player := range gm.players {
		players[id] = player
	}
	triviaSvc := gm.triviaSvc
	broadcastSvc := gm.broadcastSvc

	// Release lock before making external calls
	gm.mu.Unlock()

	// Deliver trivia questions
	if triviaSvc != nil && broadcastSvc != nil {
		questions := triviaSvc.GetQuestionsForRound(players)
		broadcastSvc.BroadcastTriviaQuestions(questions)
	}

	// Reacquire lock for timer setup
	gm.mu.Lock()

	// Set timer for next round or complete phase
	if gm.game.CurrentRound < config.ResourceGatheringRounds {
		gm.roundTimer = utils.NewTimer(
			time.Duration(config.ResourceGatheringRoundDuration)*time.Second,
			func() {
				go gm.StartResourceRound()
			},
		)
		gm.roundTimer.Start()
	} else {
		// Resource gathering complete
		go func() {
			time.Sleep(time.Duration(config.ResourceGatheringRoundDuration) * time.Second)
			gm.CompleteResourceGathering()
		}()
	}

	gm.mu.Unlock()
}

// CompleteResourceGathering completes the resource gathering phase
func (gm *GameManager) CompleteResourceGathering() {
	// Broadcast completion outside of lock
	func() {
		gm.mu.Lock()
		defer gm.mu.Unlock()
		// Just verify we're in the right phase
		if gm.game.CurrentPhase != models.PhaseResourceGathering {
			return
		}
	}()

	// Broadcast resource phase completion
	if gm.broadcastSvc != nil {
		gm.broadcastSvc.BroadcastResourcePhaseComplete()
	}

	// Wait a bit for transition
	go func() {
		time.Sleep(5 * time.Second)

		// Keep track of what to broadcast after unlocking
		var shouldBroadcast bool
		var broadcastSvc *BroadcastService

		// Do state changes under lock
		func() {
			gm.mu.Lock()
			defer gm.mu.Unlock()

			// Check if we can broadcast
			shouldBroadcast = gm.broadcastSvc != nil

			// Store service reference for use outside lock
			broadcastSvc = gm.broadcastSvc

			// Transition to puzzle phase
			playerCount := len(gm.players)
			gm.game.StartPuzzlePhase(playerCount)
			gridSize := gm.game.PuzzleGrid.Size

			// Assign puzzle segments while under lock
			if gm.puzzleSvc != nil {
				gm.puzzleSvc.AssignSegments(gm.players, gridSize, gm.game)
			}
		}()

		// Broadcast puzzle phase load outside of lock
		if shouldBroadcast {
			broadcastSvc.BroadcastPuzzlePhaseLoad()
		}

		log.Println("Transitioned to puzzle assembly phase")
	}()

	log.Println("Resource gathering phase complete")
}

// StartPuzzleTimer starts the puzzle phase timer
func (gm *GameManager) StartPuzzleTimer() error {
	var totalTime int
	var previewTime int
	var shouldBroadcast bool
	var broadcastSvc *BroadcastService

	// Do state changes under lock
	err := func() error {
		gm.mu.Lock()
		defer gm.mu.Unlock()

		if gm.game.CurrentPhase != models.PhasePuzzleAssembly {
			return fmt.Errorf("not in puzzle phase")
		}

		if gm.game.PuzzleTimerStarted {
			return fmt.Errorf("timer already started")
		}

		gm.game.StartPuzzleTimer()

		totalTime = gm.game.GetTotalPuzzleTime()
		previewTime = gm.game.GetClarityPreviewTime()
		gm.puzzleTimer = utils.NewTimer(
			time.Duration(totalTime)*time.Second,
			func() {
				go gm.PuzzleTimeout()
			},
		)
		gm.puzzleTimer.Start()

		log.Printf("Puzzle timer started - %d seconds", totalTime)

		// Check if we can broadcast
		shouldBroadcast = gm.broadcastSvc != nil
		if shouldBroadcast {
			broadcastSvc = gm.broadcastSvc
		}

		return nil
	}()

	if err != nil {
		return err
	}

	// Broadcast timer start outside of lock to avoid deadlock
	if shouldBroadcast {
		broadcastSvc.BroadcastPuzzlePhaseStart(totalTime, previewTime)
	}

	return nil
}

// CompleteSegment marks a player's segment as completed
func (gm *GameManager) CompleteSegment(playerID string, segmentID string) error {
	// Hold data to broadcast after releasing lock
	var playerCopy *models.Player
	var fragmentCopy *models.Fragment
	var shouldBroadcast bool
	var checkCompletion bool

	// Update state under lock
	err := func() error {
		gm.mu.Lock()
		defer gm.mu.Unlock()

		player, exists := gm.players[playerID]
		if !exists {
			return fmt.Errorf("player not found")
		}

		// Check if player is still in individual puzzle phase (2A)
		if player.PuzzlePhase != "2A" {
			return fmt.Errorf("player not in individual puzzle phase")
		}

		if player.SegmentCompleted {
			return fmt.Errorf("segment already completed")
		}

		// Complete the individual puzzle
		if player.IndividualPuzzle != nil {
			player.IndividualPuzzle.Complete()
			player.SegmentSolveTime = player.IndividualPuzzle.SolveTimeSeconds
		} else {
			player.SegmentSolveTime = time.Since(gm.game.PuzzleStartTime).Seconds()
		}

		// Mark as completed
		player.SegmentCompleted = true

		// Transition player to collaborative phase (2B)
		player.PuzzlePhase = "2B"

		// Add fragment to central grid (instant transformation from individual to collaborative)
		fragment := gm.game.PuzzleGrid.AddFragment(segmentID, playerID)
		player.FragmentID = fragment.ID

		// Record in analytics
		if gm.analyticsSvc != nil {
			gm.analyticsSvc.RecordSegmentCompletion(playerID, player.SegmentSolveTime)
		}

		log.Printf("Player %s completed individual puzzle for segment %s, transitioning to Phase 2B", playerID, segmentID)

		// Check if all players completed
		allCompleted := true
		for _, p := range gm.players {
			if p.IsActive && !p.SegmentCompleted {
				allCompleted = false
				break
			}
		}

		// Make copies for broadcasting
		playerCopy = &models.Player{
			ID:               player.ID,
			Name:             player.Name,
			AssignedSegment:  player.AssignedSegment,
			SegmentSolveTime: player.SegmentSolveTime,
			FragmentID:       player.FragmentID,
			Connection:       player.Connection,
			Send:             player.Send,
		}
		fragmentCopy = &models.Fragment{
			ID:       fragment.ID,
			Position: fragment.Position,
		}
		shouldBroadcast = gm.broadcastSvc != nil
		checkCompletion = allCompleted && gm.game.PuzzleGrid != nil

		return nil
	}()

	if err != nil {
		return err
	}

	// Broadcast updates outside of lock
	if shouldBroadcast {
		gm.broadcastSvc.BroadcastSegmentCompleted(playerCopy, fragmentCopy)

		if checkCompletion {
			// Check for immediate victory with a fresh lock
			gm.mu.RLock()
			isComplete := gm.game.PuzzleGrid != nil && gm.game.PuzzleGrid.CheckCompletion()
			gm.mu.RUnlock()

			if isComplete {
				gm.PuzzleComplete(true)
			}
		}
	}

	return nil
}

// MoveFragment handles a fragment move request
func (gm *GameManager) MoveFragment(playerID string, fragmentID string, targetPos models.Position) error {
	gm.mu.Lock()

	player, exists := gm.players[playerID]
	if !exists {
		gm.mu.Unlock()
		return fmt.Errorf("player not found")
	}

	// Check cooldown
	if !player.CanMoveFragment(config.FragmentMoveCooldown) {
		gm.mu.Unlock()
		return fmt.Errorf("move on cooldown")
	}

	// Store references for validation outside lock
	puzzleGrid := gm.game.PuzzleGrid
	puzzleSvc := gm.puzzleSvc

	// Release lock before validation to avoid deadlock
	gm.mu.Unlock()

	// Validate move using puzzle service
	if puzzleSvc != nil {
		if err := puzzleSvc.ValidateFragmentMove(puzzleGrid, playerID, fragmentID, targetPos); err != nil {
			return err
		}
	}

	// Reacquire lock for move execution
	gm.mu.Lock()
	defer gm.mu.Unlock()

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

	// Invalidate recommendations involving moved fragments
	if puzzleSvc != nil {
		puzzleSvc.InvalidateRecommendationsForFragment(fragmentID)
		if targetFragment != nil {
			// Also invalidate recommendations for the swapped fragment
			puzzleSvc.InvalidateRecommendationsForFragment(targetFragment.ID)
		}
	}

	// Check for completion
	if gm.game.PuzzleGrid.CheckCompletion() {
		gm.PuzzleComplete(true)
	}

	return nil
}

// PuzzleComplete handles puzzle completion
func (gm *GameManager) PuzzleComplete(success bool) {
	gm.mu.Lock()

	// Check current state

	// Check if already completed
	if gm.game.PuzzleCompleted {
		gm.mu.Unlock()
		return
	}

	gm.game.CompleteGame(success)

	if gm.puzzleTimer != nil {
		gm.puzzleTimer.Stop()
	}

	log.Printf("Puzzle phase complete - success: %v", success)

	// Calculate analytics
	if gm.analyticsSvc != nil {
		gm.analyticsSvc.FinalizeGame(gm.game, gm.players, success)
	}

	// Store references for use after unlock
	broadcastSvc := gm.broadcastSvc
	analyticsSvc := gm.analyticsSvc
	completionTime := gm.game.CompletionTime

	gm.mu.Unlock()

	// Broadcast completion and analytics (outside of lock)
	if broadcastSvc != nil {
		broadcastSvc.BroadcastPuzzleComplete(success, completionTime)

		// Send analytics data to host
		if analyticsSvc != nil {
			if analytics := analyticsSvc.GetFullAnalytics(); analytics != nil {
				broadcastSvc.BroadcastAnalytics(analytics)
			}
		}
	}
}

// PuzzleTimeout handles puzzle timeout
func (gm *GameManager) PuzzleTimeout() {
	// Check if already completed without holding the lock for too long
	gm.mu.RLock()
	completed := gm.game.PuzzleCompleted
	gm.mu.RUnlock()

	if completed {
		return
	}

	// Call PuzzleComplete which will acquire its own lock
	gm.PuzzleComplete(false)
}

// ResetGame resets the game to initial state
func (gm *GameManager) ResetGame() {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	// Stop timers
	if gm.roundTimer != nil {
		gm.roundTimer.Stop()
		gm.roundTimer = nil
	}
	if gm.puzzleTimer != nil {
		gm.puzzleTimer.Stop()
		gm.puzzleTimer = nil
	}

	// Reset game state
	gm.game.Reset()

	// Clear players
	for _, player := range gm.players {
		if player.Done != nil {
			// Safely close channel using recover to handle already closed channels
			func() {
				defer func() {
					if r := recover(); r != nil {
						// Channel was already closed, ignore
					}
				}()
				close(player.Done)
			}()
		}
	}
	gm.players = make(map[string]*models.Player)

	// Clear host
	if gm.host != nil && gm.host.Done != nil {
		// Safely close channel using recover to handle already closed channels
		func() {
			defer func() {
				if r := recover(); r != nil {
					// Channel was already closed, ignore
				}
			}()
			close(gm.host.Done)
		}()
	}
	gm.host = nil

	// Clear recommendations
	gm.recommendations = make(map[string]*models.MoveRecommendation)

	// Reset analytics
	if gm.analyticsSvc != nil {
		gm.analyticsSvc.Reset()
	}

	// Clear service references to allow new ones to be set
	gm.broadcastSvc = nil
	gm.triviaSvc = nil
	gm.puzzleSvc = nil
	gm.analyticsSvc = nil

	log.Println("Game reset to initial state")
}

// Cleanup properly shuts down the game manager
func (gm *GameManager) Cleanup() {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	// Stop all timers
	if gm.roundTimer != nil {
		gm.roundTimer.Stop()
		gm.roundTimer = nil
	}
	if gm.puzzleTimer != nil {
		gm.puzzleTimer.Stop()
		gm.puzzleTimer = nil
	}

	// Stop puzzle service expiration monitor
	if gm.puzzleSvc != nil {
		// Try to send stop signal
		select {
		case gm.puzzleSvc.stopExpiration <- true:
		default:
			// Channel might be closed or full, ignore
		}
		// The ticker will be stopped by the goroutine when it receives the stop signal
		// Don't manually stop or nil the ticker here to avoid race conditions
		// Clear the service reference
		gm.puzzleSvc = nil
	}

	// Close all player channels
	for _, player := range gm.players {
		if player.Done != nil {
			func() {
				defer func() {
					if r := recover(); r != nil {
						// Channel was already closed, ignore
					}
				}()
				close(player.Done)
			}()
		}
	}

	// Close host channel if exists
	if gm.host != nil && gm.host.Done != nil {
		func() {
			defer func() {
				if r := recover(); r != nil {
					// Channel was already closed, ignore
				}
			}()
			close(gm.host.Done)
		}()
	}

	log.Println("Game manager cleaned up")
}

// GetGame returns the current game state
func (gm *GameManager) GetGame() *models.Game {
	gm.mu.RLock()
	defer gm.mu.RUnlock()
	return gm.game
}

// GetBroadcastService returns the broadcast service
// Note: This method doesn't use locking as the broadcast service is set once during initialization
// and never changes during runtime, making it safe for concurrent access
func (gm *GameManager) GetBroadcastService() *BroadcastService {
	return gm.broadcastSvc
}

// GetTriviaService returns the trivia service
// Note: This method doesn't use locking as the trivia service is set once during initialization
// and never changes during runtime, making it safe for concurrent access
func (gm *GameManager) GetTriviaService() *TriviaService {
	return gm.triviaSvc
}

// GetPuzzleService returns the puzzle service
// Note: This method doesn't use locking as the puzzle service is set once during initialization
// and never changes during runtime, making it safe for concurrent access
func (gm *GameManager) GetPuzzleService() *PuzzleService {
	return gm.puzzleSvc
}

// GetAnalyticsService returns the analytics service
// Note: This method doesn't use locking as the analytics service is set once during initialization
// and never changes during runtime, making it safe for concurrent access
func (gm *GameManager) GetAnalyticsService() *AnalyticsService {
	return gm.analyticsSvc
}
