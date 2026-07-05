import { describe, expect, it } from 'vitest';
import { identity, isSolved, preSolvedCells, shuffle, swapCells } from './minipuzzle';

describe('minipuzzle', () => {
  it('never shuffles into the solved state', () => {
    for (let i = 0; i < 200; i++) {
      expect(isSolved(shuffle(16))).toBe(false);
    }
  });

  it('keeps pre-solved cells fixed', () => {
    const pre = preSolvedCells(16, 6);
    for (let i = 0; i < 100; i++) {
      const p = shuffle(16, pre);
      for (const cell of pre) {
        expect(p[cell]).toBe(cell);
      }
    }
  });

  it('is a permutation', () => {
    const p = shuffle(16, preSolvedCells(16, 6));
    expect([...p].sort((a, b) => a - b)).toEqual(identity(16));
  });

  it('solves by swaps', () => {
    let p = shuffle(4);
    // Selection sort by swaps: any permutation is reachable.
    for (let cell = 0; cell < p.length; cell++) {
      if (p[cell] !== cell) {
        p = swapCells(p, cell, p.indexOf(cell));
      }
    }
    expect(isSolved(p)).toBe(true);
  });

  it('handles a fully pre-solved board', () => {
    const p = shuffle(4, preSolvedCells(4, 4));
    expect(isSolved(p)).toBe(true); // nothing movable → identity
  });

  it('escapes a degenerate random source with a forced swap', () => {
    const p = shuffle(4, new Set(), () => 0);
    expect(isSolved(p)).toBe(false);
  });
});
