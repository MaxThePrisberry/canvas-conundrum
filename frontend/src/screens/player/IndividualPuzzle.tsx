// Phase 2A: the private k×k mini-puzzle over the player's segment image.
// Tap two tiles to swap them; solving sends PUZZLE_TO_SERVER_SEGMENT_COMPLETED
// (the client is authoritative for its own segment; the ack is idempotent so
// retries are safe).

import { useEffect, useMemo, useRef, useState } from 'react';
import { segmentURL } from '../../lib/assets';
import { isSolved, preSolvedCells, shuffle, swapCells, type Permutation } from '../../lib/minipuzzle';
import { wireTimestamp } from '../../protocol/envelope';
import { Events } from '../../protocol/events';
import type { PlayerState } from '../../state/playerReducer';
import type { GameSocket } from '../../ws/client';
import { Shell } from './PlayerApp';

interface Props {
  state: PlayerState;
  socket: GameSocket;
}

export function IndividualPuzzle({ state, socket }: Props) {
  const load = state.phaseLoad;
  const pieces = load?.individualPuzzleSize ?? 16;
  const k = Math.round(Math.sqrt(pieces));
  const preSolved = useMemo(
    () => preSolvedCells(pieces, load?.anchorPreSolvedPieces ?? 0),
    [pieces, load?.anchorPreSolvedPieces],
  );

  const [imageURL, setImageURL] = useState<string | null>(null);
  const [perm, setPerm] = useState<Permutation>(() => shuffle(pieces, preSolved));
  const [selected, setSelected] = useState<number | null>(null);
  const [sent, setSent] = useState(false);
  const startedAt = useRef(Date.now());

  useEffect(() => {
    if (!load) return;
    segmentURL(load.assignedSegmentId, socket.authToken)
      .then(setImageURL)
      .catch(() => setImageURL(null));
  }, [load, socket]);

  useEffect(() => {
    if (!load || sent || !isSolved(perm)) return;
    setSent(true);
    socket.send(Events.PuzzleToServerSegmentCompleted, {
      segmentId: load.assignedSegmentId,
      completionTimestamp: wireTimestamp(),
      solveTime: (Date.now() - startedAt.current) / 1000,
      manualPiecesSolved: pieces - preSolved.size,
      preSolvedPieces: preSolved.size,
    });
  }, [perm, sent, load, pieces, preSolved, socket]);

  if (!load) {
    return (
      <Shell title="Loading puzzle…">
        <p className="muted">Waiting for your segment assignment.</p>
      </Shell>
    );
  }

  const tap = (cell: number) => {
    if (preSolved.has(cell) || sent) return;
    if (selected === null) {
      setSelected(cell);
      return;
    }
    if (selected !== cell) {
      setPerm((p) => swapCells(p, selected, cell));
    }
    setSelected(null);
  };

  const tilePercent = 100 / k;

  return (
    <Shell title="Solve your piece">
      <p className="muted">Tap two tiles to swap them. Locked tiles came pre-solved (anchor tokens).</p>
      <div className="mini-puzzle" style={{ gridTemplateColumns: `repeat(${k}, 1fr)` }}>
        {perm.map((tile, cell) => {
          const tx = (tile % k) * tilePercent;
          const ty = Math.floor(tile / k) * tilePercent;
          const classes = ['mini-tile'];
          if (preSolved.has(cell)) classes.push('locked');
          if (selected === cell) classes.push('selected');
          return (
            <div
              key={cell}
              className={classes.join(' ')}
              onClick={() => tap(cell)}
              style={
                imageURL
                  ? {
                      backgroundImage: `url(${imageURL})`,
                      backgroundSize: `${k * 100}% ${k * 100}%`,
                      backgroundPosition: `${(tx / (100 - tilePercent)) * 100}% ${(ty / (100 - tilePercent)) * 100}%`,
                    }
                  : undefined
              }
            />
          );
        })}
      </div>
      {sent && <p>Segment complete — joining the central canvas…</p>}
    </Shell>
  );
}
