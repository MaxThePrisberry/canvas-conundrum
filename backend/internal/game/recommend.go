package game

import (
	"fmt"
	"time"

	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/protocol"
	"github.com/google/uuid"
)

// Recommendation is one pending swap proposal. It names fragments, not
// positions: it survives fragment moves and executes at current positions.
type Recommendation struct {
	ID          string
	FromID      string
	TargetID    string
	FromSegment string
	ToSegment   string
	Reasoning   string
	ExpiresAt   time.Time
	DeliveredAt time.Time
}

func (e *Engine) handleRecommendMove(p *Player, payload protocol.RecommendMove) {
	if !e.assemblyActive() || !e.puzzle.enteredGrid[p.ID] {
		e.sendPlayerError(p, protocol.ErrorTypeGameState, protocol.ErrForbiddenPhase,
			"recommendations are not accepted now",
			"the sender must be collaborating on the grid and the game must not be paused")
		return
	}

	target, ok := e.players[payload.TargetPlayerID]
	if !ok || !target.Connected {
		e.sendPlayerError(p, protocol.ErrorTypeValidation, protocol.ErrForbiddenNotOwner,
			"target player is not available", "")
		return
	}

	from, fromOK := e.puzzle.grid[payload.FromSegmentID]
	to, toOK := e.puzzle.grid[payload.ToSegmentID]
	if !fromOK || !toOK {
		e.sendPlayerError(p, protocol.ErrorTypeValidation, protocol.ErrForbiddenNotOwner,
			"both fragments must be visible on the grid", "")
		return
	}
	// The sender must control the from-fragment; the to-fragment must be
	// owned by the target (recommendations exist precisely for swaps the
	// sender cannot execute directly).
	if !e.controls(p, from) || to.OwnerID != payload.TargetPlayerID {
		e.sendPlayerError(p, protocol.ErrorTypeValidation, protocol.ErrForbiddenNotOwner,
			"fragment ownership does not match the recommendation",
			fmt.Sprintf("you must control %s and %s must own %s",
				payload.FromSegmentID, target.Name, payload.ToSegmentID))
		return
	}

	if e.fragmentOnCooldown(from) || e.fragmentOnCooldown(to) {
		e.sendPlayerError(p, protocol.ErrorTypeGameState, protocol.ErrCooldownActive,
			"an involved fragment is on its move cooldown", "")
		return
	}
	if _, pending := e.puzzle.pendingBySender[p.ID]; pending {
		e.sendPlayerError(p, protocol.ErrorTypeGameState, protocol.ErrRecommendationPending,
			"you already have an outgoing recommendation pending", "")
		return
	}

	rec := &Recommendation{
		ID:          uuid.NewString(),
		FromID:      p.ID,
		TargetID:    target.ID,
		FromSegment: payload.FromSegmentID,
		ToSegment:   payload.ToSegmentID,
		Reasoning:   payload.Reasoning,
		ExpiresAt:   time.Now().Add(e.cfg.RecommendationTimeout.Duration()),
		DeliveredAt: time.Now(),
	}
	e.puzzle.recommendations[rec.ID] = rec
	e.puzzle.pendingBySender[p.ID] = rec.ID
	e.timers.Schedule(timerRecPrefix+rec.ID, e.cfg.RecommendationTimeout.Duration(), true)

	p.Stats.RecommendationsSent++
	target.Stats.RecommendationsReceived++

	target.send(protocol.PuzzleToPlayerMoveRecommendation, protocol.MoveRecommendation{
		MoveID:         rec.ID,
		FromPlayerID:   p.ID,
		FromPlayerName: p.Name,
		TargetPlayerID: target.ID,
		FromSegmentID:  rec.FromSegment,
		ToSegmentID:    rec.ToSegment,
		Reasoning:      rec.Reasoning,
		ExpiresAt:      protocol.Timestamp(rec.ExpiresAt),
	})
	e.touchGrid() // activeRecommendations changed
}

func (e *Engine) handleRecommendationResponse(p *Player, payload protocol.RecommendationResponse) {
	if e.phase != protocol.PhasePuzzleAssembly || e.puzzle.paused || !e.puzzle.timerRunning {
		e.sendPlayerError(p, protocol.ErrorTypeGameState, protocol.ErrForbiddenPhase,
			"recommendation responses are not accepted now", "")
		return
	}

	rec, ok := e.puzzle.recommendations[payload.MoveID]
	if !ok {
		// Expired or already resolved: the responder has (or will get) the
		// EXPIRED event; a late response is not an error worth surfacing.
		return
	}
	if rec.TargetID != p.ID {
		e.sendPlayerError(p, protocol.ErrorTypeValidation, protocol.ErrForbiddenNotOwner,
			"this recommendation is not addressed to you", "")
		return
	}

	reason := ""
	if payload.ResponseReason != nil {
		reason = *payload.ResponseReason
	}
	sender := e.players[rec.FromID]

	if payload.Response == "reject" {
		e.recordRecResponse(p, rec)
		e.removeRecommendation(rec)
		if sender != nil {
			sender.send(protocol.PuzzleToPlayerRecommendationResult, protocol.RecommendationResult{
				MoveID:           rec.ID,
				TargetPlayerID:   p.ID,
				TargetPlayerName: p.Name,
				Response:         "reject",
				ResponseReason:   reason,
			})
		}
		e.touchGrid()
		return
	}

	// Accept: both fragments must still be off cooldown, else the
	// recommendation stays pending and can be accepted again later.
	from, fromOK := e.puzzle.grid[rec.FromSegment]
	to, toOK := e.puzzle.grid[rec.ToSegment]
	if !fromOK || !toOK {
		return // fragments vanished (cannot happen mid-game); drop silently
	}
	if e.fragmentOnCooldown(from) || e.fragmentOnCooldown(to) {
		e.sendPlayerError(p, protocol.ErrorTypeGameState, protocol.ErrCooldownActive,
			"an involved fragment is on its move cooldown",
			"the recommendation stays pending; accept again once the cooldown passes")
		return
	}

	e.recordRecResponse(p, rec)
	e.removeRecommendation(rec)

	swap := protocol.SwapExecuted{
		Segment1ID:          from.SegmentID,
		Segment1OldPosition: posOf(from.Pos),
		Segment2ID:          to.SegmentID,
		Segment2OldPosition: posOf(to.Pos),
	}
	from.Pos, to.Pos = to.Pos, from.Pos
	e.restartCooldown(from)
	e.restartCooldown(to)
	swap.Segment1NewPosition = posOf(from.Pos)
	swap.Segment2NewPosition = posOf(to.Pos)

	p.Stats.RecommendationsAccepted++

	if sender != nil {
		sender.send(protocol.PuzzleToPlayerRecommendationResult, protocol.RecommendationResult{
			MoveID:           rec.ID,
			TargetPlayerID:   p.ID,
			TargetPlayerName: p.Name,
			Response:         "accept",
			ResponseReason:   reason,
			SwapExecuted:     &swap,
		})
	}

	e.touchGrid()
	e.checkVictory()
}

func (e *Engine) recordRecResponse(p *Player, rec *Recommendation) {
	p.Stats.RecResponses++
	p.Stats.RecResponseTimeSum += time.Since(rec.DeliveredAt).Seconds()
}

func (e *Engine) removeRecommendation(rec *Recommendation) {
	delete(e.puzzle.recommendations, rec.ID)
	if e.puzzle.pendingBySender[rec.FromID] == rec.ID {
		delete(e.puzzle.pendingBySender, rec.FromID)
	}
	e.timers.Cancel(timerRecPrefix + rec.ID)
}

// expireRecommendation resolves a pending recommendation as expired and
// notifies both parties. reason is "timeout" or "player_disconnected".
func (e *Engine) expireRecommendation(id, reason string) {
	rec, ok := e.puzzle.recommendations[id]
	if !ok {
		return
	}
	e.removeRecommendation(rec)

	expired := protocol.RecommendationExpired{MoveID: rec.ID, Reason: reason}
	if sender, ok := e.players[rec.FromID]; ok {
		sender.send(protocol.PuzzleToPlayerRecommendationExpired, expired)
	}
	if target, ok := e.players[rec.TargetID]; ok {
		target.send(protocol.PuzzleToPlayerRecommendationExpired, expired)
	}
	e.touchGrid()
}

// expireRecommendationsInvolving clears every pending recommendation where
// the player is sender or target (disconnect handling).
func (e *Engine) expireRecommendationsInvolving(playerID string) {
	for id, rec := range e.puzzle.recommendations {
		if rec.FromID == playerID || rec.TargetID == playerID {
			e.expireRecommendation(id, "player_disconnected")
		}
	}
}
