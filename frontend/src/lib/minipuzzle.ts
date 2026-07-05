// Phase 2A individual-puzzle model: the player's segment image is cut into
// k×k tiles; a shuffled permutation is solved by tap-tap swapping. Any
// permutation is reachable by swaps, so every shuffle is solvable. The
// client is authoritative for its own completion (game-design.md § Phase 2).

// permutation[cellIndex] = tileIndex currently displayed in that cell.
export type Permutation = number[];

export function identity(n: number): Permutation {
  return Array.from({ length: n }, (_, i) => i);
}

export function isSolved(p: Permutation): boolean {
  return p.every((tile, cell) => tile === cell);
}

// shuffle produces a uniformly random non-identity permutation of size n,
// leaving the given pre-solved cells fixed in place (anchor tokens).
// rand defaults to Math.random; tests inject a deterministic source.
export function shuffle(
  n: number,
  preSolved: Set<number> = new Set(),
  rand: () => number = Math.random,
): Permutation {
  const movable = identity(n).filter((i) => !preSolved.has(i));
  if (movable.length < 2) return identity(n);

  const result = identity(n);
  for (let attempt = 0; attempt < 100; attempt++) {
    // Fisher–Yates over the movable subset only.
    const values = [...movable];
    for (let i = values.length - 1; i > 0; i--) {
      const j = Math.floor(rand() * (i + 1));
      [values[i], values[j]] = [values[j], values[i]];
    }
    movable.forEach((cell, i) => {
      result[cell] = values[i];
    });
    if (!isSolved(result)) return result;
  }
  // Degenerate rand: force a single swap.
  const [a, b] = movable;
  const forced = identity(n);
  [forced[a], forced[b]] = [forced[b], forced[a]];
  return forced;
}

// swapCells exchanges the tiles shown in two cells (returns a new array).
export function swapCells(p: Permutation, a: number, b: number): Permutation {
  const next = [...p];
  [next[a], next[b]] = [next[b], next[a]];
  return next;
}

// preSolvedCells picks which cells arrive already solved: the first `count`
// cells in reading order. Deterministic so a reconnecting client rebuilds
// the same board shape (contents are local-only either way).
export function preSolvedCells(n: number, count: number): Set<number> {
  const cells = new Set<number>();
  for (let i = 0; i < Math.min(count, n); i++) {
    cells.add(i);
  }
  return cells;
}
