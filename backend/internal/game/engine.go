package game

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"time"

	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/config"
	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/protocol"
	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/puzzle"
	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/trivia"
	"github.com/google/uuid"
)

const timerHeartbeatSweep = "heartbeat.sweep"

// Options are the engine's non-config dependencies and protocol constants.
type Options struct {
	HostUUID string
	// DisconnectAfter is the heartbeat silence window (spec: 90s).
	DisconnectAfter  time.Duration
	PuzzleSourcesDir string
	Logger           *slog.Logger
}

// Engine is the single-goroutine game actor. All fields below cmds are owned
// by the Run goroutine and must never be touched from outside it.
type Engine struct {
	cmds chan command

	cfg    *config.Config
	bank   *trivia.Bank
	deck   *trivia.Deck
	rng    *rand.Rand
	log    *slog.Logger
	opts   Options
	timers *timerService

	phase     string
	players   map[string]*Player
	joinOrder []string

	host              Client
	hostEverConnected bool
	hostLastActivity  time.Time

	tokens    protocol.TeamTokens
	resource  resourceState
	puzzle    puzzleState
	analytics analyticsState

	// resetOccurred distinguishes post-reset asset requests (NOT_FOUND)
	// from never-generated ones (FORBIDDEN_PHASE).
	resetOccurred bool

	// Phase boundary timestamps for the host report's timeline analysis.
	setupStartedAt    time.Time
	resourceStartedAt time.Time
	prepStartedAt     time.Time

	// lastRolesSig detects role-availability changes so
	// SETUP_TO_PLAYER_ROLES_AVAILABLE is only broadcast when it changed.
	lastRolesSig string
}

// gridSize is the central grid dimension for this game's player count.
func (e *Engine) gridSize() int {
	return puzzle.GridSize(len(e.players))
}

func New(cfg *config.Config, bank *trivia.Bank, opts Options) *Engine {
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	e := &Engine{
		cmds:           make(chan command, 256),
		cfg:            cfg,
		bank:           bank,
		deck:           trivia.NewDeck(bank),
		rng:            rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64())),
		log:            opts.Logger,
		opts:           opts,
		phase:          protocol.PhaseSetup,
		players:        map[string]*Player{},
		setupStartedAt: time.Now(),
	}
	e.timers = newTimerService(func(c cmdTimer) { e.post(c) })
	return e
}

// Run processes commands until ctx is cancelled. It must be running before
// the transport accepts connections.
func (e *Engine) Run(ctx context.Context) {
	e.timers.Schedule(timerHeartbeatSweep, e.sweepInterval(), false)
	for {
		select {
		case <-ctx.Done():
			return
		case cmd := <-e.cmds:
			e.handle(cmd)
		}
	}
}

func (e *Engine) post(cmd command) {
	e.cmds <- cmd
}

// ── Transport-facing API (safe from any goroutine) ─────────────────────────

// ConnectHost attaches a host socket whose URL token was already verified.
func (e *Engine) ConnectHost(client Client) {
	e.post(cmdHostConnect{client: client})
}

// ConnectPlayer processes a player connect frame and reports whether the
// connection was accepted. On rejection the engine has already closed the
// socket with the appropriate close code.
func (e *Engine) ConnectPlayer(token string, hasToken bool, client Client) PlayerConnectResult {
	reply := make(chan PlayerConnectResult, 1)
	e.post(cmdPlayerConnect{token: token, hasToken: hasToken, client: client, reply: reply})
	return <-reply
}

// PlayerFrame delivers a decoded post-handshake player frame.
func (e *Engine) PlayerFrame(playerID string, client Client, event protocol.EventType, payload any) {
	e.post(cmdPlayerFrame{playerID: playerID, client: client, event: event, payload: payload})
}

// HostFrame delivers a decoded post-handshake host frame.
func (e *Engine) HostFrame(client Client, event protocol.EventType, payload any) {
	e.post(cmdHostFrame{client: client, event: event, payload: payload})
}

// PlayerSocketClosed reports that a player connection's read pump exited.
func (e *Engine) PlayerSocketClosed(playerID string, client Client) {
	e.post(cmdPlayerClosed{playerID: playerID, client: client})
}

// HostSocketClosed reports that a host connection's read pump exited.
func (e *Engine) HostSocketClosed(client Client) {
	e.post(cmdHostClosed{client: client})
}

func (e *Engine) handle(cmd command) {
	switch c := cmd.(type) {
	case cmdHostConnect:
		e.handleHostConnect(c)
	case cmdPlayerConnect:
		e.handlePlayerConnect(c)
	case cmdPlayerFrame:
		e.handlePlayerFrame(c)
	case cmdHostFrame:
		e.handleHostFrame(c)
	case cmdPlayerClosed:
		e.handlePlayerClosed(c)
	case cmdHostClosed:
		e.handleHostClosed(c)
	case cmdTimer:
		if e.timers.consume(c) {
			e.handleTimer(c.name)
		}
	case cmdTilesReady:
		e.handleTilesReady(c)
	case cmdAssetQuery:
		e.handleAssetQuery(c)
	}
}

func (e *Engine) handleTimer(name string) {
	switch name {
	case timerHeartbeatSweep:
		e.sweepSilentClients()
		e.timers.Schedule(timerHeartbeatSweep, e.sweepInterval(), false)
	default:
		e.handlePhaseTimer(name)
	}
}

// handlePhaseTimer dispatches phase-specific timers (resource rounds, puzzle
// deadline, ...). Implemented per phase file.
func (e *Engine) handlePhaseTimer(name string) {
	switch e.phase {
	case protocol.PhaseResourceGathering:
		e.handleResourceTimer(name)
	case protocol.PhasePuzzleAssembly:
		e.handleAssemblyTimer(name)
	}
}

// ── Frame dispatch ─────────────────────────────────────────────────────────

func (e *Engine) handlePlayerFrame(c cmdPlayerFrame) {
	p, ok := e.players[c.playerID]
	if !ok || p.client != c.client {
		return // stale frame from a superseded/removed connection
	}
	p.LastActivity = time.Now()

	switch payload := c.payload.(type) {
	case protocol.Ping:
		p.send(protocol.SystemPong, protocol.Pong{
			ServerTimestamp: protocol.Timestamp(time.Now()),
			ClientTimestamp: payload.ClientTimestamp,
			SequenceNumber:  payload.SequenceNumber,
		})
	case protocol.PlayerConfiguration:
		e.handleConfiguration(p, payload)
	case protocol.LocationVerified:
		e.handleLocationVerified(p, payload)
	case protocol.TriviaAnswer:
		e.handleTriviaAnswer(p, payload)
	case protocol.SegmentCompleted:
		e.handleSegmentCompleted(p, payload)
	case protocol.FragmentMove:
		e.handleFragmentMove(p, payload)
	case protocol.RecommendMove:
		e.handleRecommendMove(p, payload)
	case protocol.RecommendationResponse:
		e.handleRecommendationResponse(p, payload)
	default:
		e.handlePhasePlayerFrame(p, c.event, c.payload)
	}
}

// handlePhasePlayerFrame handles the phase-scoped player events; the
// concrete handlers live in the per-phase files.
func (e *Engine) handlePhasePlayerFrame(p *Player, event protocol.EventType, payload any) {
	switch event {
	default:
		e.sendPlayerError(p, protocol.ErrorTypeGameState, protocol.ErrForbiddenPhase,
			"event not valid now", string(event)+" is not accepted in phase "+e.phase)
	}
}

func (e *Engine) handleHostFrame(c cmdHostFrame) {
	if e.host != c.client {
		return
	}
	e.hostLastActivity = time.Now()

	switch payload := c.payload.(type) {
	case protocol.Ping:
		e.sendHost(protocol.SystemPong, protocol.Pong{
			ServerTimestamp: protocol.Timestamp(time.Now()),
			ClientTimestamp: payload.ClientTimestamp,
			SequenceNumber:  payload.SequenceNumber,
		})
	case protocol.StartGame:
		e.handleStartGame()
	case protocol.PuzzleStart:
		e.handlePuzzlePhaseStart()
	case protocol.ResetGame:
		e.handleResetGame()
	default:
		e.sendHostError(protocol.ErrorTypeGameState, protocol.ErrForbiddenPhase,
			"event not valid now", string(c.event)+" is not accepted in phase "+e.phase)
	}
}

// ── Connection lifecycle ───────────────────────────────────────────────────

func (e *Engine) handleHostConnect(c cmdHostConnect) {
	if e.host != nil {
		// Supersede: 1000 is deliberately the one close code that stops the
		// old client from auto-reconnecting and fighting over the socket.
		e.host.CloseWithCode(protocol.CloseNormal)
	}
	isReconnection := e.hostEverConnected
	wasPaused := e.puzzle.paused
	e.host = c.client
	e.hostEverConnected = true
	e.hostLastActivity = time.Now()

	e.sendHost(protocol.SetupToHostConnectionConfirmed, protocol.HostConnectionConfirmed{
		HostID:         e.opts.HostUUID,
		CurrentPhase:   e.phase,
		IsReconnection: isReconnection,
		GameConfig: protocol.HostGameConfig{
			MinPlayers:              e.cfg.MinPlayers,
			MaxPlayers:              e.cfg.MaxPlayers,
			ResourceGatheringRounds: e.cfg.ResourceGatheringRounds,
			TriviaAnswerTime:        e.cfg.TriviaAnswerTime.Sec(),
			TriviaGraceTime:         e.cfg.TriviaGraceTime.Sec(),
			PuzzleBaseTime:          e.cfg.PuzzleBaseTime.Sec(),
			DifficultyMode:          e.cfg.DifficultyMode,
		},
	})

	// Resume anything the host's absence paused before replaying state, so
	// the replayed payloads carry post-resume clock values.
	if wasPaused {
		e.resumeAssembly()
	}

	e.replayHostState()

	if isReconnection {
		payload := e.hostReconnectedPayload()
		if wasPaused {
			remaining := round2(e.puzzleTimeRemaining().Seconds())
			payload.TimeRemaining = &remaining
		}
		e.broadcastPlayers(protocol.SystemToClientHostReconnected, payload)
	}
}

// replayHostState sends the phase-appropriate state-restoration sequence
// after a host handshake (websocket-events.md § Reconnection Behavior).
func (e *Engine) replayHostState() {
	switch e.phase {
	case protocol.PhaseSetup:
		e.sendHost(protocol.SetupToHostPlayerRoster, e.buildRoster())
	case protocol.PhaseResourceGathering:
		e.replayHostResourceState()
	case protocol.PhasePuzzlePreparation:
		e.replayHostPrepState()
	case protocol.PhasePuzzleAssembly:
		e.replayHostAssemblyState()
	case protocol.PhaseAnalytics:
		e.sendHost(protocol.AnalyticsToHostCompleteReport, e.analytics.hostReport)
	}
}

// hostReconnectedPayload builds SYSTEM_TO_CLIENT_HOST_RECONNECTED for the
// current phase. The puzzle-assembly resume extension lands in M6.
func (e *Engine) hostReconnectedPayload() protocol.HostReconnected {
	return protocol.HostReconnected{
		HostStatus:       "reconnected",
		CurrentPhase:     e.phase,
		RestoredFeatures: e.hostFeatures(),
	}
}

func (e *Engine) handlePlayerConnect(c cmdPlayerConnect) {
	if !c.hasToken {
		e.handleNewJoin(c)
		return
	}

	p, known := e.players[c.token]
	if !known {
		c.client.CloseWithCode(protocol.CloseUnauthorized)
		c.reply <- PlayerConnectResult{}
		return
	}
	if e.phase == protocol.PhasePuzzleAssembly {
		c.client.CloseWithCode(protocol.CloseReconnectForbidden)
		c.reply <- PlayerConnectResult{}
		return
	}

	// Supersede any still-open socket for the same identity.
	if p.client != nil {
		p.client.CloseWithCode(protocol.CloseNormal)
	}
	p.client = c.client
	p.Connected = true
	p.LastActivity = time.Now()

	if e.phase == protocol.PhaseSetup {
		e.restoreSetupRole(p)
	}

	p.send(protocol.SetupToPlayerConnectionConfirmed, protocol.PlayerConnectionConfirmed{
		PlayerID:              p.ID,
		CurrentPhase:          e.phase,
		IsReconnection:        true,
		ExistingConfiguration: e.buildExistingConfiguration(p),
	})
	c.reply <- PlayerConnectResult{OK: true, PlayerID: p.ID}

	e.replayPlayerState(p)
	e.afterLobbyChange()
	e.notifyHostRosterOnly()
}

func (e *Engine) handleNewJoin(c cmdPlayerConnect) {
	if e.phase != protocol.PhaseSetup || e.connectedPlayerCount() >= e.cfg.MaxPlayers {
		c.client.CloseWithCode(protocol.CloseJoinRejected)
		c.reply <- PlayerConnectResult{}
		return
	}

	p := &Player{
		ID:           uuid.NewString(),
		Connected:    true,
		LastActivity: time.Now(),
		client:       c.client,
	}
	e.players[p.ID] = p
	e.joinOrder = append(e.joinOrder, p.ID)

	p.send(protocol.SetupToPlayerConnectionConfirmed, protocol.PlayerConnectionConfirmed{
		PlayerID:       p.ID,
		CurrentPhase:   e.phase,
		IsReconnection: false,
	})
	c.reply <- PlayerConnectResult{OK: true, PlayerID: p.ID}

	e.sendRolesAvailable(p)
	e.afterLobbyChange()
	e.notifyHostRosterOnly()
}

// replayPlayerState sends the phase-appropriate state restoration after a
// player reconnection handshake.
func (e *Engine) replayPlayerState(p *Player) {
	switch e.phase {
	case protocol.PhaseSetup:
		if !p.Ready {
			e.sendRolesAvailable(p)
		}
	case protocol.PhaseResourceGathering:
		e.replayPlayerResourceState(p)
	case protocol.PhasePuzzlePreparation:
		e.replayPlayerPrepState(p)
	case protocol.PhaseAnalytics:
		e.replayPlayerAnalyticsState(p)
	}
}

func (e *Engine) handlePlayerClosed(c cmdPlayerClosed) {
	p, ok := e.players[c.playerID]
	if !ok || p.client != c.client {
		return // superseded connection going away, not a disconnect
	}
	e.disconnectPlayer(p)
}

func (e *Engine) handleHostClosed(c cmdHostClosed) {
	if e.host != c.client {
		return
	}
	e.disconnectHost()
}

func (e *Engine) disconnectPlayer(p *Player) {
	p.client = nil
	p.Connected = false
	now := time.Now()

	notice := protocol.PlayerDisconnected{
		PlayerID:          p.ID,
		PlayerName:        p.Name,
		DisconnectionTime: protocol.Timestamp(now),
		CurrentPhase:      e.phase,
	}

	switch e.phase {
	case protocol.PhaseSetup:
		notice.UpdatedCounts = &protocol.DisconnectCounts{
			ConnectedPlayers: e.connectedPlayerCount(),
			ReadyPlayers:     e.readyPlayerCount(),
			RoleDistribution: e.roleDistribution(),
		}
		e.afterLobbyChange()
	default:
		n := e.connectedPlayerCount()
		notice.UpdatedPlayerCount = &n
		e.onPlayerDisconnectedInPhase(p, &notice)
	}

	e.sendHost(protocol.SystemToHostPlayerDisconnected, notice)
}

// onPlayerDisconnectedInPhase applies post-setup phase semantics: during
// puzzle assembly the player's puzzle is auto-solved (2A) or their fragment
// becomes unassigned (2B), and their pending recommendations expire.
func (e *Engine) onPlayerDisconnectedInPhase(p *Player, notice *protocol.PlayerDisconnected) {
	if e.phase != protocol.PhasePuzzleAssembly || e.puzzle.finished {
		return
	}

	segment := e.puzzle.assignments[p.ID]
	if !e.puzzle.enteredGrid[p.ID] {
		// Phase 2A: auto-solve into an unassigned fragment at a random cell.
		f := e.autoSolve(p)
		if f != nil {
			pos := posOf(f.Pos)
			notice.FragmentHandling = &protocol.FragmentHandling{
				SegmentID:     segment,
				NewPosition:   pos,
				NowUnassigned: true,
			}
		}
		e.revealUnassigned()
	} else if f, ok := e.puzzle.grid[segment]; ok && f.OwnerID == p.ID {
		// Phase 2B: the fragment stays put but loses its owner.
		f.OwnerID = ""
		notice.FragmentHandling = &protocol.FragmentHandling{
			SegmentID:     segment,
			NewPosition:   posOf(f.Pos),
			NowUnassigned: true,
		}
	}

	e.expireRecommendationsInvolving(p.ID)
	e.touchGrid()
	e.checkVictory()
}

func (e *Engine) disconnectHost() {
	e.host = nil

	impact := protocol.GameImpact{
		CanContinue:      e.phase != protocol.PhasePuzzleAssembly,
		AffectedFeatures: e.hostFeatures(),
	}
	payload := protocol.HostDisconnected{
		HostStatus:   "disconnected",
		CurrentPhase: e.phase,
		GameImpact:   impact,
	}
	e.onHostLost(&payload)
	e.broadcastPlayers(protocol.SystemToClientHostDisconnected, payload)
}

// onHostLost applies phase-specific host-disconnect handling: during puzzle
// assembly, everything pauses (timer, cooldowns, recommendation timeouts,
// preview window) until the host returns.
func (e *Engine) onHostLost(payload *protocol.HostDisconnected) {
	if e.phase == protocol.PhasePuzzleAssembly && e.puzzle.timerRunning && !e.puzzle.paused {
		pausedAt := e.pauseAssembly()
		payload.TimerPausedAt = protocol.Timestamp(pausedAt)
	}
}

// hostFeatures lists what the host's presence provides in the current phase.
func (e *Engine) hostFeatures() []string {
	switch e.phase {
	case protocol.PhasePuzzleAssembly:
		return []string{"host_monitoring", "phase_transitions", "puzzle_timer"}
	case protocol.PhaseResourceGathering:
		return []string{"host_monitoring"}
	default:
		return []string{"host_monitoring", "phase_transitions"}
	}
}

// ── Heartbeat sweep ────────────────────────────────────────────────────────

func (e *Engine) sweepInterval() time.Duration {
	return max(e.opts.DisconnectAfter/3, 10*time.Millisecond)
}

// sweepSilentClients enforces the heartbeat rule: no traffic for the
// configured window (spec: 90s = three missed pings) means disconnected.
// 1001 (not 1000) so a live-but-stalled client is allowed to reconnect.
func (e *Engine) sweepSilentClients() {
	cutoff := time.Now().Add(-e.opts.DisconnectAfter)
	for _, p := range e.players {
		if p.Connected && p.LastActivity.Before(cutoff) {
			if p.client != nil {
				p.client.CloseWithCode(protocol.CloseGoingAway)
			}
			e.disconnectPlayer(p)
		}
	}
	if e.host != nil && e.hostLastActivity.Before(cutoff) {
		e.host.CloseWithCode(protocol.CloseGoingAway)
		e.disconnectHost()
	}
}

// ── Counting helpers ───────────────────────────────────────────────────────

func (e *Engine) connectedPlayerCount() int {
	n := 0
	for _, p := range e.players {
		if p.Connected {
			n++
		}
	}
	return n
}

func (e *Engine) readyPlayerCount() int {
	n := 0
	for _, p := range e.players {
		if p.Connected && p.Ready {
			n++
		}
	}
	return n
}

// ── Send helpers ───────────────────────────────────────────────────────────

func (e *Engine) sendHost(event protocol.EventType, payload any) {
	if e.host != nil {
		e.host.Send(event, payload)
	}
}

func (e *Engine) broadcastPlayers(event protocol.EventType, payload any) {
	for _, p := range e.players {
		if p.Connected {
			p.send(event, payload)
		}
	}
}

// broadcastAll sends to every connected client including the host.
func (e *Engine) broadcastAll(event protocol.EventType, payload any) {
	e.broadcastPlayers(event, payload)
	e.sendHost(event, payload)
}

func (e *Engine) sendPlayerError(p *Player, errorType string, code protocol.ErrorCode, message, details string) {
	p.send(protocol.SystemToClientError, protocol.ErrorPayload{
		ErrorType: errorType,
		ErrorCode: code,
		Message:   message,
		Details:   details,
	})
}

func (e *Engine) sendHostError(errorType string, code protocol.ErrorCode, message, details string) {
	e.sendHost(protocol.SystemToHostError, protocol.ErrorPayload{
		ErrorType: errorType,
		ErrorCode: code,
		Message:   message,
		Details:   details,
	})
}
