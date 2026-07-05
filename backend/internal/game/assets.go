package game

import (
	"fmt"
	"net/http"

	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/protocol"
)

// QueryAsset asks the engine (from any goroutine) whether an asset request
// may be served, returning the bytes when granted.
func (e *Engine) QueryAsset(token, segmentID string, preview bool) AssetVerdict {
	reply := make(chan AssetVerdict, 1)
	e.post(cmdAssetQuery{token: token, segmentID: segmentID, preview: preview, reply: reply})
	return <-reply
}

func (e *Engine) handleAssetQuery(c cmdAssetQuery) {
	if c.preview {
		c.reply <- e.previewVerdict(c.token)
	} else {
		c.reply <- e.segmentVerdict(c.token, c.segmentID)
	}
}

func deny(status int, code protocol.ErrorCode, message string) AssetVerdict {
	return AssetVerdict{Status: status, Code: code, Message: message}
}

// requesterKind resolves a bearer token: host, known player, or unknown.
func (e *Engine) requesterKind(token string) (isHost bool, p *Player, ok bool) {
	if token == e.opts.HostUUID {
		return true, nil, true
	}
	p, known := e.players[token]
	return false, p, known
}

// segmentVerdict enforces the /api/segments/{segmentId} rules in spec order
// (websocket-events.md § Asset Delivery): token, phase state, then ownership
// or grid visibility (host may fetch anything).
func (e *Engine) segmentVerdict(token, segmentID string) AssetVerdict {
	isHost, p, known := e.requesterKind(token)
	if !known {
		return deny(http.StatusUnauthorized, protocol.ErrUnauthorized, "Unknown or missing bearer token.")
	}

	if !e.assetsPhaseOpen() {
		if e.resetOccurred && e.puzzle.tiles == nil {
			// Post-reset: the tile cache is cleared, NOT_FOUND per lifecycle.
			return deny(http.StatusNotFound, protocol.ErrNotFound, "Tiles were cleared by the game reset.")
		}
		return deny(http.StatusForbidden, protocol.ErrForbiddenPhase,
			"Segment tiles are not available in the current phase state.")
	}

	tile, exists := e.puzzle.tiles[segmentID]
	if !exists {
		return deny(http.StatusNotFound, protocol.ErrNotFound,
			fmt.Sprintf("Segment %s does not exist.", segmentID))
	}

	if !isHost {
		if e.puzzle.assignments[p.ID] != segmentID && !e.segmentVisibleOnGrid(segmentID) {
			return deny(http.StatusForbidden, protocol.ErrForbiddenNotOwner,
				fmt.Sprintf("Segment %s is not assigned to this player.", segmentID))
		}
	}

	return AssetVerdict{Status: http.StatusOK, Bytes: tile}
}

// assetsPhaseOpen implements rule 2: preparation-with-tiles-ready, assembly,
// or analytics.
func (e *Engine) assetsPhaseOpen() bool {
	switch e.phase {
	case protocol.PhasePuzzlePreparation:
		return e.puzzle.tilesReady
	case protocol.PhasePuzzleAssembly, protocol.PhaseAnalytics:
		return e.puzzle.tiles != nil
	}
	return false
}

// previewVerdict enforces the /api/preview/full rules: token, assembly
// phase, at least one clarity threshold, and the active window.
func (e *Engine) previewVerdict(token string) AssetVerdict {
	if _, _, known := e.requesterKind(token); !known {
		return deny(http.StatusUnauthorized, protocol.ErrUnauthorized, "Unknown or missing bearer token.")
	}
	if e.phase != protocol.PhasePuzzleAssembly {
		return deny(http.StatusForbidden, protocol.ErrForbiddenPhase,
			"The full-image preview is only available during puzzle assembly.")
	}
	if !e.previewWindowOpen() {
		return deny(http.StatusForbidden, protocol.ErrForbiddenPreviewWindowClosed,
			"The clarity preview window is not open.")
	}
	return AssetVerdict{Status: http.StatusOK, Bytes: e.puzzle.preview}
}

// segmentVisibleOnGrid reports whether the fragment is currently visible on
// the central grid — completed player fragments and revealed unassigned
// fragments alike are fetchable by any player.
func (e *Engine) segmentVisibleOnGrid(segmentID string) bool {
	return e.puzzle.grid[segmentID] != nil
}
