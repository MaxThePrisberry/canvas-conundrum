// Pure player state machine: (state, action) → state, where actions are
// server frames plus a few local/socket events. All server knowledge the
// player UI renders lives here.

import type { ServerFrame } from '../protocol/envelope';
import {
  Events,
  type AnswerResult,
  type CompletedSuccess,
  type CompletedTimeout,
  type ErrorPayload,
  type GameReset,
  type GridState,
  type HostDisconnected,
  type LobbyStatus,
  type MoveRecommendation,
  type MoveResult,
  type PersonalReport,
  type PersonalState,
  type Phase,
  type PlayerConnectionConfirmed,
  type PuzzlePhaseLoad,
  type PuzzlePhaseStart,
  type RecommendationResult,
  type ResourcePhaseComplete,
  type ResourcePhaseStart,
  type RolesAvailable,
  type SegmentAcknowledged,
  type TeamProgress,
  type TeamSummary,
  type TriviaQuestion,
} from '../protocol/events';
import type { SocketStatus } from '../ws/client';

export interface PlayerState {
  status: SocketStatus;
  closeCode?: number;
  playerId: string;
  phase: Phase;
  isReconnection: boolean;

  // Setup
  rolesAvailable: RolesAvailable | null;
  lobby: LobbyStatus | null;
  myName: string;
  myRole: string | null;
  mySpecialties: string[];
  ready: boolean;

  // Resource gathering
  resourceStart: ResourcePhaseStart | null;
  question: TriviaQuestion | null;
  answeredIndex: number | null;
  answerResult: AnswerResult | null;
  teamProgress: TeamProgress | null;
  location: string;
  resourceComplete: ResourcePhaseComplete | null;

  // Puzzle
  phaseLoad: PuzzlePhaseLoad | null;
  puzzleStart: PuzzlePhaseStart | null;
  segmentAck: SegmentAcknowledged | null;
  grid: GridState | null;
  personal: PersonalState | null;
  previewExpired: boolean;
  lastMoveResult: MoveResult | null;
  incomingRecommendation: MoveRecommendation | null;
  outgoingPending: boolean;
  lastRecommendationResult: RecommendationResult | null;
  completion: CompletedSuccess | CompletedTimeout | null;

  // Analytics
  personalReport: PersonalReport | null;
  teamSummary: TeamSummary | null;
  gameReset: GameReset | null;

  hostConnected: boolean;
  paused: boolean;
  lastError: ErrorPayload | null;
}

export const initialPlayerState: PlayerState = {
  status: 'connecting',
  playerId: '',
  phase: 'setup',
  isReconnection: false,
  rolesAvailable: null,
  lobby: null,
  myName: '',
  myRole: null,
  mySpecialties: [],
  ready: false,
  resourceStart: null,
  question: null,
  answeredIndex: null,
  answerResult: null,
  teamProgress: null,
  location: 'unknown',
  resourceComplete: null,
  phaseLoad: null,
  puzzleStart: null,
  segmentAck: null,
  grid: null,
  personal: null,
  previewExpired: false,
  lastMoveResult: null,
  incomingRecommendation: null,
  outgoingPending: false,
  lastRecommendationResult: null,
  completion: null,
  personalReport: null,
  teamSummary: null,
  gameReset: null,
  hostConnected: true,
  paused: false,
  lastError: null,
};

export type PlayerAction =
  | { type: 'frame'; frame: ServerFrame }
  | { type: 'status'; status: SocketStatus; closeCode?: number }
  | { type: 'configured'; role: string; specialties: string[]; name: string }
  | { type: 'answered'; index: number }
  | { type: 'recommendationSent' }
  | { type: 'recommendationResolved' }
  | { type: 'clearError' };

export function playerReducer(state: PlayerState, action: PlayerAction): PlayerState {
  switch (action.type) {
    case 'status':
      return { ...state, status: action.status, closeCode: action.closeCode };
    case 'configured':
      // Optimistic; CONFIGURATION errors roll visible state back via lobby.
      return {
        ...state,
        myRole: action.role,
        mySpecialties: action.specialties,
        myName: action.name,
        ready: true,
      };
    case 'answered':
      return { ...state, answeredIndex: action.index };
    case 'recommendationSent':
      return { ...state, outgoingPending: true };
    case 'recommendationResolved':
      return { ...state, incomingRecommendation: null };
    case 'clearError':
      return { ...state, lastError: null };
    case 'frame':
      return reduceFrame(state, action.frame);
  }
}

function reduceFrame(state: PlayerState, frame: ServerFrame): PlayerState {
  const p = frame.payload;
  switch (frame.event) {
    case Events.SetupToPlayerConnectionConfirmed: {
      const c = p as PlayerConnectionConfirmed;
      const existing = c.existingConfiguration;
      return {
        ...state,
        playerId: c.playerId,
        phase: c.currentPhase,
        isReconnection: c.isReconnection,
        myRole: existing ? existing.selectedRole : state.myRole,
        mySpecialties: existing ? existing.selectedSpecialties : state.mySpecialties,
        myName: existing ? existing.playerName : state.myName,
        ready: existing ? existing.ready : false,
      };
    }
    case Events.SetupToPlayerRolesAvailable:
      return { ...state, rolesAvailable: p as RolesAvailable };
    case Events.SetupToClientLobbyStatus: {
      const lobby = p as LobbyStatus;
      return { ...state, lobby, hostConnected: lobby.hasHost };
    }

    case Events.ResourceToClientPhaseStart:
      return { ...state, phase: 'resource_gathering', resourceStart: p as ResourcePhaseStart };
    case Events.ResourceToPlayerLocationConfirmed:
      return { ...state, location: (p as { newLocation: string }).newLocation };
    case Events.ResourceToPlayerTriviaQuestion:
      return { ...state, question: p as TriviaQuestion, answeredIndex: null, answerResult: null };
    case Events.ResourceToPlayerAnswerResult: {
      const result = p as AnswerResult;
      return { ...state, answerResult: result, question: null, location: result.currentLocation };
    }
    case Events.ResourceToClientTeamProgress:
      return { ...state, teamProgress: p as TeamProgress };
    case Events.ResourceToClientPhaseComplete:
      return {
        ...state,
        phase: 'puzzle_preparation',
        resourceComplete: p as ResourcePhaseComplete,
        question: null,
      };

    case Events.PuzzleToClientPhaseLoad:
      return { ...state, phase: 'puzzle_preparation', phaseLoad: p as PuzzlePhaseLoad };
    case Events.PuzzleToClientPhaseStart:
      return { ...state, phase: 'puzzle_assembly', puzzleStart: p as PuzzlePhaseStart };
    case Events.PuzzleToClientPreviewExpired:
      return { ...state, previewExpired: true };
    case Events.PuzzleToPlayerSegmentAcknowledged:
      return { ...state, segmentAck: p as SegmentAcknowledged };
    case Events.PuzzleToClientGridState:
      return { ...state, grid: p as GridState };
    case Events.PuzzleToPlayerPersonalState:
      return { ...state, personal: p as PersonalState };
    case Events.PuzzleToPlayerMoveResult:
      return { ...state, lastMoveResult: p as MoveResult };
    case Events.PuzzleToPlayerMoveRecommendation:
      return { ...state, incomingRecommendation: p as MoveRecommendation };
    case Events.PuzzleToPlayerRecommendationResult:
      return { ...state, lastRecommendationResult: p as RecommendationResult, outgoingPending: false };
    case Events.PuzzleToPlayerRecommendationExpired: {
      const expired = p as { moveId: string };
      const next = { ...state, outgoingPending: false };
      if (state.incomingRecommendation?.moveId === expired.moveId) {
        next.incomingRecommendation = null;
      }
      return next;
    }
    case Events.PuzzleToClientCompletedSuccess:
      return { ...state, completion: p as CompletedSuccess };
    case Events.PuzzleToClientCompletedTimeout:
      return { ...state, completion: p as CompletedTimeout };

    case Events.AnalyticsToPlayerPersonalReport:
      return { ...state, phase: 'analytics', personalReport: p as PersonalReport };
    case Events.AnalyticsToClientTeamSummary:
      return { ...state, phase: 'analytics', teamSummary: p as TeamSummary };
    case Events.AnalyticsToClientGameReset:
      return { ...state, gameReset: p as GameReset };

    case Events.SystemToClientHostDisconnected: {
      const d = p as HostDisconnected;
      return { ...state, hostConnected: false, paused: !d.gameImpact.canContinue };
    }
    case Events.SystemToClientHostReconnected:
      return { ...state, hostConnected: true, paused: false };
    case Events.SystemToClientError: {
      const err = p as ErrorPayload;
      const next = { ...state, lastError: err };
      // A rejected configuration rolls back the optimistic ready state so
      // the form reopens for a full resubmission.
      const configRejections = new Set([
        'ROLE_FULL',
        'INVALID_ROLE_SELECTION',
        'INVALID_SPECIALTY_SELECTION',
        'MALFORMED_PAYLOAD',
      ]);
      if (state.phase === 'setup' && state.ready && configRejections.has(err.errorCode)) {
        next.ready = false;
        next.myRole = null;
      }
      return next;
    }

    default:
      return state;
  }
}
