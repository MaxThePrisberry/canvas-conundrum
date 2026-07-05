import { describe, expect, it } from 'vitest';
import { correctPositionFor, formatSeconds, gridSizeFor, segmentIdFor } from './formulas';

describe('segment ids', () => {
  it('round-trips the spec mapping', () => {
    expect(segmentIdFor(0, 0)).toBe('segment_a1');
    expect(segmentIdFor(2, 2)).toBe('segment_c3');
    expect(correctPositionFor('segment_a1')).toEqual({ x: 0, y: 0 });
    expect(correctPositionFor('segment_h8')).toEqual({ x: 7, y: 7 });
    expect(correctPositionFor('nonsense')).toBeNull();
  });
});

describe('gridSizeFor', () => {
  it('matches the game-design table', () => {
    const cases: Array<[number, number]> = [
      [1, 3], [9, 3], [10, 4], [16, 4], [17, 5], [25, 5],
      [26, 6], [36, 6], [37, 7], [49, 7], [50, 8], [64, 8],
    ];
    for (const [players, size] of cases) {
      expect(gridSizeFor(players)).toBe(size);
    }
  });
});

describe('formatSeconds', () => {
  it('renders m:ss and clamps at zero', () => {
    expect(formatSeconds(65)).toBe('1:05');
    expect(formatSeconds(0)).toBe('0:00');
    expect(formatSeconds(-3)).toBe('0:00');
    expect(formatSeconds(359.2)).toBe('6:00');
  });
});
