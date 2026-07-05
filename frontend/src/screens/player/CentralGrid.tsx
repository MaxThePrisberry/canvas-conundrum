// Phase 2B: the shared canvas. Tap-to-select then tap-to-move/swap; swaps
// that would displace another player's fragment open the recommendation
// composer instead. Incoming recommendations arrive as a prompt; guide
// highlights overlay privately; the clarity preview is a one-shot overlay.

import { useEffect, useRef, useState } from 'react';
import { Countdown } from '../../components/Countdown';
import { GridView } from '../../components/GridView';
import { previewURL } from '../../lib/assets';
import { Events, type GridFragment, type Position } from '../../protocol/events';
import type { PlayerAction, PlayerState } from '../../state/playerReducer';
import type { GameSocket } from '../../ws/client';
import { Shell } from './PlayerApp';

interface Props {
  state: PlayerState;
  socket: GameSocket;
  dispatch: React.Dispatch<PlayerAction>;
}

export function CentralGrid({ state, socket, dispatch }: Props) {
  const [selected, setSelected] = useState<GridFragment | null>(null);
  const [composing, setComposing] = useState<GridFragment | null>(null); // recommendation target
  const [reasoning, setReasoning] = useState('');
  const [preview, setPreview] = useState<string | null>(null);
  const gridAnchoredAt = useRef(Date.now());

  const grid = state.grid;
  useEffect(() => {
    gridAnchoredAt.current = Date.now();
  }, [grid]);

  // Dismiss the preview overlay when the server says the window closed.
  useEffect(() => {
    if (state.previewExpired) setPreview(null);
  }, [state.previewExpired]);

  const gridSize = state.phaseLoad?.centralGridSize ?? 3;
  const controls = (f: GridFragment) => f.playerId === null || f.playerId === state.playerId;

  const cellClick = (pos: Position, fragment: GridFragment | null) => {
    if (state.paused) return;
    if (!selected) {
      if (fragment && controls(fragment)) setSelected(fragment);
      return;
    }
    if (fragment?.segmentId === selected.segmentId) {
      setSelected(null);
      return;
    }
    if (!fragment) {
      socket.send(Events.PuzzleToServerFragmentMove, {
        segmentId: selected.segmentId,
        targetPosition: pos,
        swapWithSegmentId: null,
      });
      setSelected(null);
      return;
    }
    if (controls(fragment)) {
      socket.send(Events.PuzzleToServerFragmentMove, {
        segmentId: selected.segmentId,
        targetPosition: pos,
        swapWithSegmentId: fragment.segmentId,
      });
      setSelected(null);
      return;
    }
    // Another player's fragment: propose the swap instead.
    setComposing(fragment);
  };

  const sendRecommendation = () => {
    if (!selected || !composing?.playerId) return;
    socket.send(Events.PuzzleToServerRecommendMove, {
      targetPlayerId: composing.playerId,
      fromSegmentId: selected.segmentId,
      toSegmentId: composing.segmentId,
      reasoning: reasoning.slice(0, 200),
    });
    dispatch({ type: 'recommendationSent' });
    setComposing(null);
    setSelected(null);
    setReasoning('');
  };

  const respond = (accept: boolean) => {
    const rec = state.incomingRecommendation;
    if (!rec) return;
    socket.send(Events.PuzzleToServerRecommendationResponse, {
      moveId: rec.moveId,
      response: accept ? 'accept' : 'reject',
    });
    dispatch({ type: 'recommendationResolved' });
  };

  const openPreview = () => {
    previewURL(socket.authToken)
      .then(setPreview)
      .catch(() => setPreview(null));
  };

  const previewAvailable =
    state.puzzleStart?.clarityPreviewActive && !state.previewExpired && !preview;

  return (
    <Shell title="Assemble the canvas">
      <div className="grid-header">
        {grid && (
          <Countdown
            secondsRemaining={grid.timeRemaining}
            anchoredAt={gridAnchoredAt.current}
            paused={state.paused}
          />
        )}
        {previewAvailable && <button onClick={openPreview}>Peek at the full image</button>}
      </div>

      <GridView
        gridSize={gridSize}
        fragments={grid?.fragments ?? []}
        token={socket.authToken}
        highlights={state.personal?.guideHighlights ?? []}
        selected={selected?.segmentId ?? null}
        myPlayerId={state.playerId}
        onCellClick={cellClick}
      />
      <p className="muted">
        {selected
          ? `Moving ${selected.segmentId}: tap an empty cell, a fragment you control to swap, or another player's fragment to propose a swap.`
          : 'Tap your fragment or an unassigned one to start a move. Highlighted cells are guide hints for your fragment.'}
      </p>

      {state.lastMoveResult?.status === 'rejected' && (
        <p className="move-rejected">
          Move rejected ({state.lastMoveResult.reason}
          {state.lastMoveResult.cooldownInfo &&
            `, ready in ${state.lastMoveResult.cooldownInfo.cooldownRemaining.toFixed(1)}s`}
          )
        </p>
      )}

      {composing && selected && (
        <div className="modal">
          <h2>Propose swap to {composing.playerName}</h2>
          <p>
            Your {selected.segmentId} ⇄ their {composing.segmentId}
          </p>
          <textarea
            value={reasoning}
            onChange={(e) => setReasoning(e.target.value)}
            maxLength={200}
            placeholder="Why should they accept? (optional)"
          />
          <div className="modal-actions">
            <button className="primary" onClick={sendRecommendation} disabled={state.outgoingPending}>
              {state.outgoingPending ? 'One already pending…' : 'Send proposal'}
            </button>
            <button onClick={() => setComposing(null)}>Cancel</button>
          </div>
        </div>
      )}

      {state.incomingRecommendation && (
        <div className="modal">
          <h2>{state.incomingRecommendation.fromPlayerName} proposes a swap</h2>
          <p>
            Their {state.incomingRecommendation.fromSegmentId} ⇄ your{' '}
            {state.incomingRecommendation.toSegmentId}
          </p>
          {state.incomingRecommendation.reasoning && (
            <blockquote>{state.incomingRecommendation.reasoning}</blockquote>
          )}
          <div className="modal-actions">
            <button className="primary" onClick={() => respond(true)}>
              Accept
            </button>
            <button onClick={() => respond(false)}>Reject</button>
          </div>
        </div>
      )}

      {preview && (
        <div className="modal preview-modal" onClick={() => setPreview(null)}>
          <img src={preview} alt="full puzzle preview" />
          <p className="muted">Memorize it! Tap to dismiss.</p>
        </div>
      )}
    </Shell>
  );
}
