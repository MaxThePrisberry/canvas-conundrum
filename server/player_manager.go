package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/MaxThePrisberry/canvas-conundrum/server/constants"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// PlayerManager handles all player-related operations
type PlayerManager struct {
	players map[string]*Player
	mu      sync.RWMutex
}

// NewPlayerManager creates a new player manager instance
func NewPlayerManager() *PlayerManager {
	return &PlayerManager{
		players: make(map[string]*Player),
	}
}

// CreatePlayer creates a new player with a unique ID
func (pm *PlayerManager) CreatePlayer(conn *websocket.Conn) *Player {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	player := &Player{
		ID:         uuid.New().String(),
		Connection: conn,
		State:      StateConnected,
		LastSeen:   time.Now(),
	}

	// First player becomes the host
	if len(pm.players) == 0 {
		player.IsHost = true
		player.Name = "Host"
	} else {
		player.Name = fmt.Sprintf("Player%d", len(pm.players)+1)
	}

	pm.players[player.ID] = player
	return player
}

// GetPlayer retrieves a player by ID
func (pm *PlayerManager) GetPlayer(playerID string) (*Player, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	player, exists := pm.players[playerID]
	if !exists {
		return nil, fmt.Errorf("player not found")
	}

	return player, nil
}

// GetAllPlayers returns all players
func (pm *PlayerManager) GetAllPlayers() []*Player {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	players := make([]*Player, 0, len(pm.players))
	for _, p := range pm.players {
		players = append(players, p)
	}

	return players
}

// GetConnectedPlayers returns only connected players
func (pm *PlayerManager) GetConnectedPlayers() []*Player {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	players := make([]*Player, 0)
	for _, p := range pm.players {
		p.mu.RLock()
		if p.State == StateConnected {
			players = append(players, p)
		}
		p.mu.RUnlock()
	}

	return players
}

// GetReadyPlayers returns players marked as ready
func (pm *PlayerManager) GetReadyPlayers() []*Player {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	players := make([]*Player, 0)
	for _, p := range pm.players {
		p.mu.RLock()
		if p.State == StateConnected && p.Ready {
			players = append(players, p)
		}
		p.mu.RUnlock()
	}

	return players
}

// DisconnectPlayer marks a player as disconnected
func (pm *PlayerManager) DisconnectPlayer(playerID string) error {
	pm.mu.RLock()
	player, exists := pm.players[playerID]
	pm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("player not found")
	}

	player.mu.Lock()
	player.State = StateDisconnected
	player.Connection = nil
	player.mu.Unlock()

	return nil
}

// ReconnectPlayer handles player reconnection
func (pm *PlayerManager) ReconnectPlayer(playerID string, conn *websocket.Conn) error {
	pm.mu.RLock()
	player, exists := pm.players[playerID]
	pm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("player not found")
	}

	player.mu.Lock()
	player.State = StateConnected
	player.Connection = conn
	player.LastSeen = time.Now()
	player.mu.Unlock()

	return nil
}

// GetAvailableRoles returns roles that haven't been selected yet
func (pm *PlayerManager) GetAvailableRoles() []RoleInfo {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// Count role selections
	roleCounts := make(map[string]int)
	for _, p := range pm.players {
		p.mu.RLock()
		if p.Role != "" {
			roleCounts[p.Role]++
		}
		p.mu.RUnlock()
	}

	// Calculate max allowed per role
	totalPlayers := len(pm.players)
	maxPerRole := (totalPlayers + 3) / 4 // Ensures even distribution

	// Build available roles list
	roles := []RoleInfo{
		{Role: constants.RoleArtEnthusiast, ResourceBonus: constants.RoleResourceMultiplier},
		{Role: constants.RoleDetective, ResourceBonus: constants.RoleResourceMultiplier},
		{Role: constants.RoleTourist, ResourceBonus: constants.RoleResourceMultiplier},
		{Role: constants.RoleJanitor, ResourceBonus: constants.RoleResourceMultiplier},
	}

	for i := range roles {
		roles[i].Available = roleCounts[roles[i].Role] < maxPerRole
	}

	return roles
}

// SetPlayerRole assigns a role to a player
func (pm *PlayerManager) SetPlayerRole(playerID, role string) error {
	pm.mu.RLock()
	player, exists := pm.players[playerID]
	pm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("player not found")
	}

	// Validate role
	validRoles := map[string]bool{
		constants.RoleArtEnthusiast: true,
		constants.RoleDetective:     true,
		constants.RoleTourist:       true,
		constants.RoleJanitor:       true,
	}

	if !validRoles[role] {
		return fmt.Errorf("invalid role")
	}

	// Check if role is available
	availableRoles := pm.GetAvailableRoles()
	roleAvailable := false
	for _, r := range availableRoles {
		if r.Role == role && r.Available {
			roleAvailable = true
			break
		}
	}

	if !roleAvailable {
		return fmt.Errorf("role not available")
	}

	player.mu.Lock()
	player.Role = role
	player.mu.Unlock()

	return nil
}

// SetPlayerSpecialties assigns trivia specialties to a player
func (pm *PlayerManager) SetPlayerSpecialties(playerID string, specialties []string) error {
	pm.mu.RLock()
	player, exists := pm.players[playerID]
	pm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("player not found")
	}

	if len(specialties) > constants.MaxSpecialtiesPerPlayer {
		return fmt.Errorf("too many specialties selected")
	}

	// Validate specialties
	validCategories := make(map[string]bool)
	for _, cat := range constants.TriviaCategories {
		validCategories[cat] = true
	}

	for _, specialty := range specialties {
		if !validCategories[specialty] {
			return fmt.Errorf("invalid specialty: %s", specialty)
		}
	}

	player.mu.Lock()
	player.Specialties = specialties
	player.mu.Unlock()

	return nil
}

// SetPlayerReady marks a player as ready to start
func (pm *PlayerManager) SetPlayerReady(playerID string, ready bool) error {
	pm.mu.RLock()
	player, exists := pm.players[playerID]
	pm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("player not found")
	}

	player.mu.Lock()
	player.Ready = ready
	player.mu.Unlock()

	return nil
}

// GetPlayerCount returns the total number of players
func (pm *PlayerManager) GetPlayerCount() int {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	return len(pm.players)
}

// GetConnectedCount returns the number of connected players
func (pm *PlayerManager) GetConnectedCount() int {
	return len(pm.GetConnectedPlayers())
}

// GetReadyCount returns the number of ready players
func (pm *PlayerManager) GetReadyCount() int {
	return len(pm.GetReadyPlayers())
}

// GetHost returns the host player
func (pm *PlayerManager) GetHost() *Player {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	for _, p := range pm.players {
		p.mu.RLock()
		isHost := p.IsHost
		p.mu.RUnlock()

		if isHost {
			return p
		}
	}

	return nil
}

// GetRoleDistribution returns the current role distribution
func (pm *PlayerManager) GetRoleDistribution() map[string]int {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	distribution := make(map[string]int)

	for _, p := range pm.players {
		p.mu.RLock()
		if p.Role != "" {
			distribution[p.Role]++
		}
		p.mu.RUnlock()
	}

	return distribution
}

// UpdatePlayerLocation updates a player's current resource station
func (pm *PlayerManager) UpdatePlayerLocation(playerID string, locationHash string) error {
	pm.mu.RLock()
	player, exists := pm.players[playerID]
	pm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("player not found")
	}

	// Validate location hash
	validLocation := false
	for _, hash := range constants.ResourceStationHashes {
		if hash == locationHash {
			validLocation = true
			break
		}
	}

	if !validLocation {
		return fmt.Errorf("invalid location hash")
	}

	player.mu.Lock()
	player.CurrentLocation = locationHash
	player.mu.Unlock()

	return nil
}

// GetPlayerLocation returns a player's current location
func (pm *PlayerManager) GetPlayerLocation(playerID string) (string, error) {
	pm.mu.RLock()
	player, exists := pm.players[playerID]
	pm.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("player not found")
	}

	player.mu.RLock()
	location := player.CurrentLocation
	player.mu.RUnlock()

	return location, nil
}
