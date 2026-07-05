import { describe, expect, it } from 'vitest';
import { nextBackoff, shouldReconnect } from './backoff';

describe('shouldReconnect', () => {
  it('never retries the terminal codes', () => {
    // 1000 (reset / supersede) and every application 4xxx are permanent.
    for (const code of [1000, 4001, 4002, 4003, 4999]) {
      expect(shouldReconnect(code)).toBe(false);
    }
  });

  it('retries transient closes', () => {
    for (const code of [1001, 1006, 1011]) {
      expect(shouldReconnect(code)).toBe(true);
    }
  });
});

describe('nextBackoff', () => {
  it('doubles from 1s and caps at 30s', () => {
    expect(nextBackoff(0)).toBe(1000);
    expect(nextBackoff(1)).toBe(2000);
    expect(nextBackoff(4)).toBe(16000);
    expect(nextBackoff(5)).toBe(30000);
    expect(nextBackoff(20)).toBe(30000);
  });
});
