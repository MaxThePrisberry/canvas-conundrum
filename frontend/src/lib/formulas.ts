// Pure game formulas mirrored client-side for display (the server remains
// authoritative; see game-design.md).

// segmentIdFor renders the canonical segment id for a 0-based cell.
export function segmentIdFor(x: number, y: number): string {
  return `segment_${String.fromCharCode(97 + y)}${x + 1}`;
}

// correctPositionFor inverts segmentIdFor: the cell a segment belongs on.
export function correctPositionFor(segmentId: string): { x: number; y: number } | null {
  const match = /^segment_([a-z])(\d+)$/.exec(segmentId);
  if (!match) return null;
  return { x: parseInt(match[2], 10) - 1, y: match[1].charCodeAt(0) - 97 };
}

// gridSizeFor is the player-count → grid-size table from game-design.md.
export function gridSizeFor(players: number): number {
  if (players <= 9) return 3;
  if (players <= 16) return 4;
  if (players <= 25) return 5;
  if (players <= 36) return 6;
  if (players <= 49) return 7;
  return 8;
}

// formatSeconds renders a countdown as m:ss.
export function formatSeconds(total: number): string {
  const clamped = Math.max(0, Math.ceil(total));
  const minutes = Math.floor(clamped / 60);
  const seconds = clamped % 60;
  return `${minutes}:${String(seconds).padStart(2, '0')}`;
}
