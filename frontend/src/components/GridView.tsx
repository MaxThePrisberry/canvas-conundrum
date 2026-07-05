// GridView renders the central puzzle grid: visible fragments as tile
// images, guide highlights as private overlays, and cell taps for the
// move/swap interaction.

import { useEffect, useState } from 'react';
import { segmentURL } from '../lib/assets';
import type { GridFragment, Position } from '../protocol/events';

interface Props {
  gridSize: number;
  fragments: GridFragment[];
  token: string;
  highlights?: Position[];
  selected?: string | null; // selected fragment's segmentId
  myPlayerId?: string;
  onCellClick?: (pos: Position, fragment: GridFragment | null) => void;
}

export function GridView({
  gridSize,
  fragments,
  token,
  highlights = [],
  selected,
  myPlayerId,
  onCellClick,
}: Props) {
  const [tileURLs, setTileURLs] = useState<Record<string, string>>({});

  useEffect(() => {
    let cancelled = false;
    for (const f of fragments) {
      if (tileURLs[f.segmentId]) continue;
      segmentURL(f.segmentId, token)
        .then((url) => {
          if (!cancelled) setTileURLs((prev) => ({ ...prev, [f.segmentId]: url }));
        })
        .catch(() => {
          /* not fetchable yet; retried on next state change */
        });
    }
    return () => {
      cancelled = true;
    };
  }, [fragments, token, tileURLs]);

  const byPos = new Map<string, GridFragment>();
  for (const f of fragments) {
    byPos.set(`${f.position.x},${f.position.y}`, f);
  }
  const highlightSet = new Set(highlights.map((h) => `${h.x},${h.y}`));

  const cells = [];
  for (let y = 0; y < gridSize; y++) {
    for (let x = 0; x < gridSize; x++) {
      const key = `${x},${y}`;
      const fragment = byPos.get(key) ?? null;
      const classes = ['grid-cell'];
      if (highlightSet.has(key)) classes.push('grid-cell-highlight');
      if (fragment && fragment.segmentId === selected) classes.push('grid-cell-selected');
      if (fragment && fragment.playerId === null) classes.push('grid-cell-unassigned');
      if (fragment && myPlayerId && fragment.playerId === myPlayerId) classes.push('grid-cell-mine');

      cells.push(
        <div
          key={key}
          className={classes.join(' ')}
          onClick={() => onCellClick?.({ x, y }, fragment)}
        >
          {fragment && tileURLs[fragment.segmentId] && (
            <img src={tileURLs[fragment.segmentId]} alt={fragment.segmentId} draggable={false} />
          )}
          {fragment && (
            <span className="grid-cell-owner">{fragment.playerName ?? 'unassigned'}</span>
          )}
        </div>,
      );
    }
  }

  return (
    <div className="grid-view" style={{ gridTemplateColumns: `repeat(${gridSize}, 1fr)` }}>
      {cells}
    </div>
  );
}
