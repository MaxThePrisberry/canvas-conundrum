import { describe, expect, it } from 'vitest';
import { Events } from '../protocol/events';
import { hostReducer, initialHostState, type HostState } from './hostReducer';

function feed(state: HostState, event: string, payload: unknown): HostState {
  return hostReducer(state, {
    type: 'frame',
    frame: { event: event as never, payload, timestamp: '2025-06-15T14:23:05.000Z' },
  });
}

describe('hostReducer', () => {
  it('walks the full phase sequence', () => {
    let s = feed(initialHostState, Events.SetupToHostConnectionConfirmed, {
      hostId: 'h1',
      currentPhase: 'setup',
      isReconnection: false,
      gameConfig: { minPlayers: 4 },
    });
    expect(s.confirmed?.hostId).toBe('h1');

    s = feed(s, Events.SetupToHostGameStarted, { phase: 'resource_gathering' });
    expect(s.phase).toBe('resource_gathering');

    s = feed(s, Events.ResourceToHostRoundAnalytics, { currentRound: 2, totalRounds: 5 });
    expect(s.roundAnalytics?.currentRound).toBe(2);

    s = feed(s, Events.PuzzleToHostPreparing, {});
    expect(s.prepStatus).toBe('preparing');
    expect(s.phase).toBe('puzzle_preparation');

    s = feed(s, Events.PuzzleToHostReady, {});
    expect(s.prepStatus).toBe('ready');

    s = feed(s, Events.PuzzleToHostPhaseStart, { timerActive: true, totalTime: 360 });
    expect(s.phase).toBe('puzzle_assembly');
    expect(s.prepStatus).toBe('started');

    s = feed(s, Events.AnalyticsToHostCompleteReport, { gameSuccess: true });
    expect(s.phase).toBe('analytics');
    expect(s.completeReport?.gameSuccess).toBe(true);
  });

  it('reset returns to a fresh setup while keeping the connection', () => {
    let s = feed(initialHostState, Events.SetupToHostConnectionConfirmed, {
      hostId: 'h1',
      currentPhase: 'analytics',
      isReconnection: false,
      gameConfig: {},
    });
    s = feed(s, Events.AnalyticsToHostCompleteReport, { gameSuccess: false });
    s = feed(s, Events.AnalyticsToClientGameReset, { reason: 'host_initiated_reset' });

    expect(s.phase).toBe('setup');
    expect(s.completeReport).toBeNull();
    expect(s.confirmed?.hostId).toBe('h1'); // connection context survives
  });

  it('keeps a bounded disconnect log', () => {
    let s = initialHostState;
    for (let i = 0; i < 8; i++) {
      s = feed(s, Events.SystemToHostPlayerDisconnected, { playerId: `p${i}` });
    }
    expect(s.recentDisconnects.length).toBe(5);
    expect(s.recentDisconnects[4].playerId).toBe('p7');
  });
});
