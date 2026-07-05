import { describe, expect, it } from 'vitest';
import { Events } from '../protocol/events';
import { initialPlayerState, playerReducer, type PlayerState } from './playerReducer';

function feed(state: PlayerState, event: string, payload: unknown): PlayerState {
  return playerReducer(state, {
    type: 'frame',
    frame: { event: event as never, payload, timestamp: '2025-06-15T14:23:05.000Z' },
  });
}

describe('playerReducer', () => {
  it('runs the setup handshake', () => {
    let s = initialPlayerState;
    s = feed(s, Events.SetupToPlayerConnectionConfirmed, {
      playerId: 'p1',
      currentPhase: 'setup',
      isReconnection: false,
      existingConfiguration: null,
    });
    expect(s.playerId).toBe('p1');
    expect(s.phase).toBe('setup');

    s = feed(s, Events.SetupToPlayerRolesAvailable, {
      roles: [{ roleType: 'detective', available: true }],
      triviaCategories: ['science'],
      maxSpecialties: 1,
    });
    expect(s.rolesAvailable?.roles[0].roleType).toBe('detective');
  });

  it('restores existing configuration on reconnect', () => {
    const s = feed(initialPlayerState, Events.SetupToPlayerConnectionConfirmed, {
      playerId: 'p1',
      currentPhase: 'resource_gathering',
      isReconnection: true,
      existingConfiguration: {
        selectedRole: 'detective',
        selectedSpecialties: ['science'],
        playerName: 'Alice',
        ready: true,
      },
    });
    expect(s.myRole).toBe('detective');
    expect(s.myName).toBe('Alice');
    expect(s.ready).toBe(true);
    expect(s.phase).toBe('resource_gathering');
  });

  it('rolls back an optimistic configuration on ROLE_FULL', () => {
    let s = playerReducer(initialPlayerState, {
      type: 'configured',
      role: 'detective',
      specialties: ['science'],
      name: 'Alice',
    });
    expect(s.ready).toBe(true);

    s = feed(s, Events.SystemToClientError, {
      errorType: 'validation_error',
      errorCode: 'ROLE_FULL',
      message: 'taken',
    });
    expect(s.ready).toBe(false);
    expect(s.myRole).toBeNull();
    expect(s.lastError?.errorCode).toBe('ROLE_FULL');
  });

  it('tracks a full trivia round', () => {
    let s = feed(initialPlayerState, Events.ResourceToClientPhaseStart, {
      phase: 'resource_gathering',
      totalRounds: 5,
    });
    expect(s.phase).toBe('resource_gathering');

    s = feed(s, Events.ResourceToPlayerTriviaQuestion, {
      questionId: 'q1',
      options: ['a', 'b'],
      roundNumber: 1,
    });
    expect(s.question?.questionId).toBe('q1');

    s = playerReducer(s, { type: 'answered', index: 1 });
    expect(s.answeredIndex).toBe(1);

    s = feed(s, Events.ResourceToPlayerAnswerResult, {
      questionId: 'q1',
      correct: true,
      currentLocation: 'guide',
      tokensEarned: 30,
    });
    expect(s.question).toBeNull();
    expect(s.answerResult?.correct).toBe(true);
    expect(s.location).toBe('guide');
  });

  it('moves through the puzzle phases', () => {
    let s = feed(initialPlayerState, Events.PuzzleToClientPhaseLoad, {
      phase: 'puzzle_preparation',
      assignedSegmentId: 'segment_a1',
      centralGridSize: 3,
    });
    expect(s.phase).toBe('puzzle_preparation');

    s = feed(s, Events.PuzzleToClientPhaseStart, {
      startTimestamp: 't',
      totalTime: 360,
      clarityPreviewActive: true,
    });
    expect(s.phase).toBe('puzzle_assembly');

    s = feed(s, Events.PuzzleToClientPreviewExpired, {});
    expect(s.previewExpired).toBe(true);

    s = feed(s, Events.PuzzleToClientGridState, { fragments: [], timeRemaining: 100 });
    expect(s.grid?.timeRemaining).toBe(100);

    s = feed(s, Events.PuzzleToClientCompletedSuccess, {
      success: true,
      completionTime: 285,
    });
    expect(s.completion?.success).toBe(true);
  });

  it('clears an incoming recommendation when it expires', () => {
    let s = feed(initialPlayerState, Events.PuzzleToPlayerMoveRecommendation, {
      moveId: 'rec-1',
      fromPlayerName: 'Diana',
    });
    expect(s.incomingRecommendation?.moveId).toBe('rec-1');

    s = feed(s, Events.PuzzleToPlayerRecommendationExpired, { moveId: 'rec-1', reason: 'timeout' });
    expect(s.incomingRecommendation).toBeNull();
  });

  it('pauses on a host disconnect during assembly and resumes', () => {
    let s = feed(initialPlayerState, Events.SystemToClientHostDisconnected, {
      hostStatus: 'disconnected',
      currentPhase: 'puzzle_assembly',
      gameImpact: { canContinue: false, affectedFeatures: ['puzzle_timer'] },
      timerPausedAt: 't',
    });
    expect(s.paused).toBe(true);
    expect(s.hostConnected).toBe(false);

    s = feed(s, Events.SystemToClientHostReconnected, {
      hostStatus: 'reconnected',
      currentPhase: 'puzzle_assembly',
      restoredFeatures: [],
      timeRemaining: 215,
    });
    expect(s.paused).toBe(false);
    expect(s.hostConnected).toBe(true);
  });

  it('lands on analytics reports and reset', () => {
    let s = feed(initialPlayerState, Events.AnalyticsToPlayerPersonalReport, {
      playerId: 'p1',
      personalScore: 196,
      gameSuccess: true,
    });
    expect(s.phase).toBe('analytics');
    expect(s.personalReport?.personalScore).toBe(196);

    s = feed(s, Events.AnalyticsToClientTeamSummary, { totalScore: 932, leaderboard: [] });
    expect(s.teamSummary?.totalScore).toBe(932);

    s = feed(s, Events.AnalyticsToClientGameReset, {
      reason: 'host_initiated_reset',
      reconnectRequired: true,
    });
    expect(s.gameReset?.reconnectRequired).toBe(true);
  });
});
