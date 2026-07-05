// Pure host state machine, mirroring playerReducer for the host dashboard.

import type { ServerFrame } from '../protocol/envelope';
import {
  Events,
  type CompletionAnalytics,
  type ErrorPayload,
  type GameReset,
  type HostCompleteReport,
  type HostConnectionConfirmed,
  type HostGridState,
  type HostPuzzlePhaseLoad,
  type HostPuzzlePhaseStart,
  type HostResourcePhaseStart,
  type HostSegmentCompleted,
  type Phase,
  type PlayerDisconnected,
  type PlayerRoster,
  type RoundAnalytics,
} from '../protocol/events';
import type { SocketStatus } from '../ws/client';

export type PrepStatus = 'idle' | 'preparing' | 'ready' | 'started';

export interface HostState {
  status: SocketStatus;
  closeCode?: number;
  confirmed: HostConnectionConfirmed | null;
  phase: Phase;

  roster: PlayerRoster | null;
  resourceStart: HostResourcePhaseStart | null;
  roundAnalytics: RoundAnalytics | null;

  prepStatus: PrepStatus;
  phaseLoad: HostPuzzlePhaseLoad | null;
  puzzleStart: HostPuzzlePhaseStart | null;
  gridState: HostGridState | null;
  lastCompletion: HostSegmentCompleted | null;
  completionAnalytics: CompletionAnalytics | null;

  completeReport: HostCompleteReport | null;
  gameReset: GameReset | null;

  recentDisconnects: PlayerDisconnected[];
  lastError: ErrorPayload | null;
}

export const initialHostState: HostState = {
  status: 'connecting',
  confirmed: null,
  phase: 'setup',
  roster: null,
  resourceStart: null,
  roundAnalytics: null,
  prepStatus: 'idle',
  phaseLoad: null,
  puzzleStart: null,
  gridState: null,
  lastCompletion: null,
  completionAnalytics: null,
  completeReport: null,
  gameReset: null,
  recentDisconnects: [],
  lastError: null,
};

export type HostAction =
  | { type: 'frame'; frame: ServerFrame }
  | { type: 'status'; status: SocketStatus; closeCode?: number }
  | { type: 'clearError' };

export function hostReducer(state: HostState, action: HostAction): HostState {
  switch (action.type) {
    case 'status':
      return { ...state, status: action.status, closeCode: action.closeCode };
    case 'clearError':
      return { ...state, lastError: null };
    case 'frame':
      return reduceFrame(state, action.frame);
  }
}

function reduceFrame(state: HostState, frame: ServerFrame): HostState {
  const p = frame.payload;
  switch (frame.event) {
    case Events.SetupToHostConnectionConfirmed: {
      const c = p as HostConnectionConfirmed;
      return { ...state, confirmed: c, phase: c.currentPhase };
    }
    case Events.SetupToHostPlayerRoster:
      return { ...state, roster: p as PlayerRoster, phase: (p as PlayerRoster).phase };
    case Events.SetupToHostGameStarted:
      return { ...state, phase: 'resource_gathering' };

    case Events.ResourceToHostPhaseStart:
      return { ...state, phase: 'resource_gathering', resourceStart: p as HostResourcePhaseStart };
    case Events.ResourceToHostRoundAnalytics:
      return { ...state, roundAnalytics: p as RoundAnalytics };
    case Events.ResourceToHostPhaseComplete:
      return { ...state, phase: 'puzzle_preparation' };

    case Events.PuzzleToHostPreparing:
      return { ...state, phase: 'puzzle_preparation', prepStatus: 'preparing' };
    case Events.PuzzleToHostReady:
      return { ...state, prepStatus: 'ready' };
    case Events.PuzzleToHostPhaseLoad:
      return { ...state, phaseLoad: p as HostPuzzlePhaseLoad };
    case Events.PuzzleToHostPhaseStart:
      return {
        ...state,
        phase: 'puzzle_assembly',
        prepStatus: 'started',
        puzzleStart: p as HostPuzzlePhaseStart,
      };
    case Events.PuzzleToHostGridState:
      return { ...state, gridState: p as HostGridState };
    case Events.PuzzleToHostSegmentCompleted:
      return { ...state, lastCompletion: p as HostSegmentCompleted };
    case Events.PuzzleToHostCompletionAnalytics:
      return { ...state, completionAnalytics: p as CompletionAnalytics };

    case Events.AnalyticsToHostCompleteReport:
      return { ...state, phase: 'analytics', completeReport: p as HostCompleteReport };
    case Events.AnalyticsToClientGameReset:
      return { ...initialHostState, status: state.status, confirmed: state.confirmed };

    case Events.SystemToHostPlayerDisconnected: {
      const d = p as PlayerDisconnected;
      return { ...state, recentDisconnects: [...state.recentDisconnects.slice(-4), d] };
    }
    case Events.SystemToHostError:
      return { ...state, lastError: p as ErrorPayload };

    default:
      return state;
  }
}
