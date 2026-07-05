package game

import (
	"fmt"
	"strings"

	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/protocol"
)

// roleDef is the static catalog of the four roles (game-design.md § Role
// Selection). Order matches the spec's SETUP_TO_PLAYER_ROLES_AVAILABLE
// example.
type roleDef struct {
	Type           string
	DisplayName    string
	BonusTokenType string
}

var roleDefs = []roleDef{
	{"art_enthusiast", "Art Enthusiast", "clarity"},
	{"detective", "Detective", "guide"},
	{"tourist", "Tourist", "chronos"},
	{"janitor", "Janitor", "anchor"},
}

func roleByType(roleType string) (roleDef, bool) {
	for _, r := range roleDefs {
		if r.Type == roleType {
			return r, true
		}
	}
	return roleDef{}, false
}

// roleCapacity is the per-role slot count: max(1, ceil(connected/4)).
func (e *Engine) roleCapacity() int {
	return max(1, (e.connectedPlayerCount()+3)/4)
}

// roleOccupancy counts connected, configured players per role.
func (e *Engine) roleOccupancy() map[string]int {
	occ := map[string]int{}
	for _, p := range e.players {
		if p.Connected && p.Ready && p.Role != "" {
			occ[p.Role]++
		}
	}
	return occ
}

// roleDistribution is roleOccupancy with every role present (zeros included),
// the shape used in roster and disconnect payloads.
func (e *Engine) roleDistribution() map[string]int {
	dist := map[string]int{}
	occ := e.roleOccupancy()
	for _, r := range roleDefs {
		dist[r.Type] = occ[r.Type]
	}
	return dist
}

// ── Configuration ──────────────────────────────────────────────────────────

func (e *Engine) handleConfiguration(p *Player, cfg protocol.PlayerConfiguration) {
	if e.phase != protocol.PhaseSetup {
		e.sendPlayerError(p, protocol.ErrorTypeGameState, protocol.ErrForbiddenPhase,
			"configuration not accepted now", "configuration is only accepted during setup")
		return
	}
	if p.Ready {
		e.sendPlayerError(p, protocol.ErrorTypeValidation, protocol.ErrConfigurationLocked,
			"configuration is locked once ready", "resubmission after ready is not allowed")
		return
	}

	if _, ok := roleByType(cfg.SelectedRole); !ok {
		e.sendPlayerError(p, protocol.ErrorTypeValidation, protocol.ErrInvalidRoleSelection,
			"Role selection validation failed",
			fmt.Sprintf("'%s' is not a recognized role", cfg.SelectedRole))
		return
	}
	if err := e.validateSpecialties(cfg.SelectedSpecialties); err != nil {
		e.sendPlayerError(p, protocol.ErrorTypeValidation, protocol.ErrInvalidSpecialtySelection,
			"Specialty selection validation failed", err.Error())
		return
	}

	// Serial processing makes this the race-resolution point: the first
	// configuration to land takes the slot, later ones see it occupied.
	if e.roleOccupancy()[cfg.SelectedRole] >= e.roleCapacity() {
		e.sendPlayerError(p, protocol.ErrorTypeValidation, protocol.ErrRoleFull,
			"All slots for the requested role are taken",
			fmt.Sprintf("role '%s' is at capacity; choose another role and resubmit", cfg.SelectedRole))
		return
	}

	p.Role = cfg.SelectedRole
	p.Specialties = append([]string(nil), cfg.SelectedSpecialties...)
	p.Name = cfg.PlayerName
	p.Ready = true

	e.afterLobbyChange()
	e.notifyHostRosterOnly()
}

func (e *Engine) validateSpecialties(specialties []string) error {
	if len(specialties) < 1 || len(specialties) > e.cfg.MaxSpecialtiesPerPlayer {
		return fmt.Errorf("selectedSpecialties must contain 1-%d categories, got %d",
			e.cfg.MaxSpecialtiesPerPlayer, len(specialties))
	}
	known := map[string]bool{}
	for _, c := range e.bank.Categories() {
		known[c] = true
	}
	seen := map[string]bool{}
	for _, s := range specialties {
		if !known[s] {
			return fmt.Errorf("'%s' is not a known trivia category", s)
		}
		if seen[s] {
			return fmt.Errorf("duplicate specialty '%s'", s)
		}
		seen[s] = true
	}
	return nil
}

// restoreSetupRole re-applies a reconnecting player's preserved role if a
// slot is still free; otherwise the role (and ready state) is dropped and
// the player must reselect (game-design.md § Race Resolution).
func (e *Engine) restoreSetupRole(p *Player) {
	if p.Role == "" {
		return
	}
	// Occupancy counts everyone but the reconnecting player: they were
	// removed from the distribution on disconnect and are reclaiming a slot.
	occupied := 0
	for _, other := range e.players {
		if other != p && other.Connected && other.Ready && other.Role == p.Role {
			occupied++
		}
	}
	if occupied >= e.roleCapacity() {
		p.Role = ""
		p.Ready = false
	}
}

func (e *Engine) buildExistingConfiguration(p *Player) *protocol.ExistingConfiguration {
	if p.Name == "" && p.Role == "" && len(p.Specialties) == 0 {
		return nil // never configured
	}
	cfg := &protocol.ExistingConfiguration{
		SelectedSpecialties: append([]string(nil), p.Specialties...),
		PlayerName:          p.Name,
		Ready:               p.Ready,
	}
	if p.Role != "" {
		role := p.Role
		cfg.SelectedRole = &role
	}
	return cfg
}

// ── Lobby broadcasts ───────────────────────────────────────────────────────

// afterLobbyChange re-broadcasts setup state after any join/leave/config
// change: role availability (only if it changed) and lobby status.
func (e *Engine) afterLobbyChange() {
	if e.phase != protocol.PhaseSetup {
		return
	}
	e.maybeBroadcastRoles()
	e.broadcastPlayers(protocol.SetupToClientLobbyStatus, e.buildLobbyStatus())
}

func (e *Engine) notifyHostRosterOnly() {
	if e.phase == protocol.PhaseSetup {
		e.sendHost(protocol.SetupToHostPlayerRoster, e.buildRoster())
	}
}

func (e *Engine) buildRolesAvailable() protocol.RolesAvailable {
	capacity := e.roleCapacity()
	occ := e.roleOccupancy()
	roles := make([]protocol.RoleInfo, 0, len(roleDefs))
	for _, r := range roleDefs {
		roles = append(roles, protocol.RoleInfo{
			RoleType:       r.Type,
			DisplayName:    r.DisplayName,
			ResourceBonus:  e.cfg.RoleResourceMultiplier,
			BonusTokenType: r.BonusTokenType,
			Description:    "Excels at " + r.BonusTokenType + " token collection",
			Available:      occ[r.Type] < capacity,
		})
	}
	return protocol.RolesAvailable{
		Roles:            roles,
		TriviaCategories: e.bank.Categories(),
		MaxSpecialties:   e.cfg.MaxSpecialtiesPerPlayer,
	}
}

func rolesSignature(ra protocol.RolesAvailable) string {
	var b strings.Builder
	for _, r := range ra.Roles {
		fmt.Fprintf(&b, "%s=%t;", r.RoleType, r.Available)
	}
	return b.String()
}

// maybeBroadcastRoles sends SETUP_TO_PLAYER_ROLES_AVAILABLE to every unready
// connected player, but only when availability actually changed since the
// last broadcast (the spec's trigger condition).
func (e *Engine) maybeBroadcastRoles() {
	ra := e.buildRolesAvailable()
	sig := rolesSignature(ra)
	if sig == e.lastRolesSig {
		return
	}
	e.lastRolesSig = sig
	for _, p := range e.players {
		if p.Connected && !p.Ready {
			p.send(protocol.SetupToPlayerRolesAvailable, ra)
		}
	}
}

// sendRolesAvailable sends the current availability to one player
// (immediately after their handshake).
func (e *Engine) sendRolesAvailable(p *Player) {
	ra := e.buildRolesAvailable()
	e.lastRolesSig = rolesSignature(ra)
	p.send(protocol.SetupToPlayerRolesAvailable, ra)
}

func (e *Engine) gameStartEligible() bool {
	connected := e.connectedPlayerCount()
	ready := e.readyPlayerCount()
	return connected > 0 && ready == connected && ready >= e.cfg.MinPlayers
}

func (e *Engine) buildLobbyStatus() protocol.LobbyStatus {
	connected := e.connectedPlayerCount()
	ready := e.readyPlayerCount()

	// PlayerRoles counts configured players only, so it sums to ready.
	playerRoles := map[string]int{}
	for role, n := range e.roleOccupancy() {
		playerRoles[role] = n
	}

	return protocol.LobbyStatus{
		CurrentPlayers:    connected,
		MinPlayers:        e.cfg.MinPlayers,
		MaxPlayers:        e.cfg.MaxPlayers,
		PlayerRoles:       playerRoles,
		HasHost:           e.host != nil,
		AllPlayersReady:   connected > 0 && ready == connected,
		ReadyPlayers:      ready,
		GameStartEligible: e.gameStartEligible(),
		WaitingMessage:    e.waitingMessage(connected, ready),
	}
}

func (e *Engine) waitingMessage(connected, ready int) string {
	switch {
	case e.host == nil:
		return "Waiting for the host to connect"
	case connected < e.cfg.MinPlayers:
		return fmt.Sprintf("Waiting for %d more %s to join",
			e.cfg.MinPlayers-connected, plural(e.cfg.MinPlayers-connected, "player"))
	case ready < connected:
		return fmt.Sprintf("Waiting for %d more %s to be ready",
			connected-ready, plural(connected-ready, "player"))
	default:
		return "Ready to start"
	}
}

func plural(n int, word string) string {
	if n == 1 {
		return word
	}
	return word + "s"
}

func (e *Engine) buildRoster() protocol.PlayerRoster {
	statuses := map[string]protocol.PlayerStatus{}
	for id, p := range e.players {
		st := protocol.PlayerStatus{
			PlayerName:   p.Name,
			Specialties:  append([]string{}, p.Specialties...),
			Connected:    p.Connected,
			Ready:        p.Connected && p.Ready,
			LastActivity: protocol.Timestamp(p.LastActivity),
		}
		if p.Role != "" {
			role := p.Role
			st.Role = &role
		}
		statuses[id] = st
	}
	return protocol.PlayerRoster{
		Phase:             e.phase,
		ConnectedPlayers:  e.connectedPlayerCount(),
		ReadyPlayers:      e.readyPlayerCount(),
		GameStartEligible: e.gameStartEligible(),
		PlayerStatuses:    statuses,
		RoleDistribution:  e.roleDistribution(),
	}
}

// ── Game start ─────────────────────────────────────────────────────────────

func (e *Engine) handleStartGame() {
	if e.phase != protocol.PhaseSetup {
		e.sendHostError(protocol.ErrorTypeGameState, protocol.ErrForbiddenPhase,
			"game already started", "SETUP_TO_SERVER_START_GAME is only valid during setup")
		return
	}
	if !e.gameStartEligible() {
		connected := e.connectedPlayerCount()
		ready := e.readyPlayerCount()
		e.sendHost(protocol.SystemToHostError, protocol.ErrorPayload{
			ErrorType: protocol.ErrorTypeGameState,
			ErrorCode: protocol.ErrInsufficientPlayers,
			Message:   "Cannot start game with insufficient players",
			Details: fmt.Sprintf("Need at least %d ready players with every connected player ready; currently %d of %d connected are ready",
				e.cfg.MinPlayers, ready, connected),
			Context: map[string]any{
				"requestedAction":  "start_game",
				"currentPlayers":   connected,
				"requiredPlayers":  e.cfg.MinPlayers,
				"connectedPlayers": connected,
				"readyPlayers":     ready,
			},
			SuggestedActions: []string{
				"Wait for more players to join",
				"Verify all players are properly connected",
			},
		})
		return
	}

	e.sendHost(protocol.SetupToHostGameStarted, protocol.GameStarted{
		Phase:             protocol.PhaseResourceGathering,
		TotalPlayers:      e.connectedPlayerCount(),
		InitialTeamTokens: e.tokens,
	})
	e.enterResourceGathering()
}

// playerIDsInJoinOrder returns the IDs of players still in the game, in the
// order they first joined (used for deterministic segment assignment).
func (e *Engine) playerIDsInJoinOrder() []string {
	ids := make([]string, 0, len(e.players))
	for _, id := range e.joinOrder {
		if _, ok := e.players[id]; ok {
			ids = append(ids, id)
		}
	}
	return ids
}
