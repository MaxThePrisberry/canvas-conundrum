// Wire protocol types mirroring websocket-events.md 1:1. Event names are
// string literals so reducers can switch exhaustively.

// ── Event names ────────────────────────────────────────────────────────────

export const Events = {
  // Setup
  SetupToHostConnectionConfirmed: 'SETUP_TO_HOST_CONNECTION_CONFIRMED',
  SetupToServerPlayerConnect: 'SETUP_TO_SERVER_PLAYER_CONNECT',
  SetupToPlayerConnectionConfirmed: 'SETUP_TO_PLAYER_CONNECTION_CONFIRMED',
  SetupToPlayerRolesAvailable: 'SETUP_TO_PLAYER_ROLES_AVAILABLE',
  SetupToServerPlayerConfiguration: 'SETUP_TO_SERVER_PLAYER_CONFIGURATION',
  SetupToClientLobbyStatus: 'SETUP_TO_CLIENT_LOBBY_STATUS',
  SetupToHostPlayerRoster: 'SETUP_TO_HOST_PLAYER_ROSTER',
  SetupToServerStartGame: 'SETUP_TO_SERVER_START_GAME',
  SetupToHostGameStarted: 'SETUP_TO_HOST_GAME_STARTED',
  // Resource gathering
  ResourceToClientPhaseStart: 'RESOURCE_TO_CLIENT_PHASE_START',
  ResourceToHostPhaseStart: 'RESOURCE_TO_HOST_PHASE_START',
  ResourceToServerLocationVerified: 'RESOURCE_TO_SERVER_LOCATION_VERIFIED',
  ResourceToPlayerLocationConfirmed: 'RESOURCE_TO_PLAYER_LOCATION_CONFIRMED',
  ResourceToPlayerTriviaQuestion: 'RESOURCE_TO_PLAYER_TRIVIA_QUESTION',
  ResourceToServerTriviaAnswer: 'RESOURCE_TO_SERVER_TRIVIA_ANSWER',
  ResourceToPlayerAnswerResult: 'RESOURCE_TO_PLAYER_ANSWER_RESULT',
  ResourceToClientTeamProgress: 'RESOURCE_TO_CLIENT_TEAM_PROGRESS',
  ResourceToHostRoundAnalytics: 'RESOURCE_TO_HOST_ROUND_ANALYTICS',
  ResourceToClientPhaseComplete: 'RESOURCE_TO_CLIENT_PHASE_COMPLETE',
  ResourceToHostPhaseComplete: 'RESOURCE_TO_HOST_PHASE_COMPLETE',
  // Puzzle preparation + assembly
  PuzzleToHostPreparing: 'PUZZLE_TO_HOST_PREPARING',
  PuzzleToHostReady: 'PUZZLE_TO_HOST_READY',
  PuzzleToClientPhaseLoad: 'PUZZLE_TO_CLIENT_PHASE_LOAD',
  PuzzleToHostPhaseLoad: 'PUZZLE_TO_HOST_PHASE_LOAD',
  PuzzleToServerPhaseStart: 'PUZZLE_TO_SERVER_PHASE_START',
  PuzzleToClientPhaseStart: 'PUZZLE_TO_CLIENT_PHASE_START',
  PuzzleToHostPhaseStart: 'PUZZLE_TO_HOST_PHASE_START',
  PuzzleToClientPreviewExpired: 'PUZZLE_TO_CLIENT_PREVIEW_EXPIRED',
  PuzzleToServerSegmentCompleted: 'PUZZLE_TO_SERVER_SEGMENT_COMPLETED',
  PuzzleToPlayerSegmentAcknowledged: 'PUZZLE_TO_PLAYER_SEGMENT_ACKNOWLEDGED',
  PuzzleToHostSegmentCompleted: 'PUZZLE_TO_HOST_SEGMENT_COMPLETED',
  PuzzleToServerFragmentMove: 'PUZZLE_TO_SERVER_FRAGMENT_MOVE',
  PuzzleToPlayerMoveResult: 'PUZZLE_TO_PLAYER_MOVE_RESULT',
  PuzzleToClientGridState: 'PUZZLE_TO_CLIENT_GRID_STATE',
  PuzzleToHostGridState: 'PUZZLE_TO_HOST_GRID_STATE',
  PuzzleToPlayerPersonalState: 'PUZZLE_TO_PLAYER_PERSONAL_STATE',
  PuzzleToServerRecommendMove: 'PUZZLE_TO_SERVER_RECOMMEND_MOVE',
  PuzzleToPlayerMoveRecommendation: 'PUZZLE_TO_PLAYER_MOVE_RECOMMENDATION',
  PuzzleToServerRecommendationResponse: 'PUZZLE_TO_SERVER_RECOMMENDATION_RESPONSE',
  PuzzleToPlayerRecommendationResult: 'PUZZLE_TO_PLAYER_RECOMMENDATION_RESULT',
  PuzzleToPlayerRecommendationExpired: 'PUZZLE_TO_PLAYER_RECOMMENDATION_EXPIRED',
  PuzzleToClientCompletedSuccess: 'PUZZLE_TO_CLIENT_COMPLETED_SUCCESS',
  PuzzleToClientCompletedTimeout: 'PUZZLE_TO_CLIENT_COMPLETED_TIMEOUT',
  PuzzleToHostCompletionAnalytics: 'PUZZLE_TO_HOST_COMPLETION_ANALYTICS',
  // Analytics
  AnalyticsToPlayerPersonalReport: 'ANALYTICS_TO_PLAYER_PERSONAL_REPORT',
  AnalyticsToClientTeamSummary: 'ANALYTICS_TO_CLIENT_TEAM_SUMMARY',
  AnalyticsToHostCompleteReport: 'ANALYTICS_TO_HOST_COMPLETE_REPORT',
  AnalyticsToServerResetGame: 'ANALYTICS_TO_SERVER_RESET_GAME',
  AnalyticsToClientGameReset: 'ANALYTICS_TO_CLIENT_GAME_RESET',
  // System
  SystemToClientError: 'SYSTEM_TO_CLIENT_ERROR',
  SystemToHostError: 'SYSTEM_TO_HOST_ERROR',
  SystemToClientHostDisconnected: 'SYSTEM_TO_CLIENT_HOST_DISCONNECTED',
  SystemToClientHostReconnected: 'SYSTEM_TO_CLIENT_HOST_RECONNECTED',
  SystemToHostPlayerDisconnected: 'SYSTEM_TO_HOST_PLAYER_DISCONNECTED',
  SystemPing: 'SYSTEM_PING',
  SystemPong: 'SYSTEM_PONG',
} as const;

export type EventName = (typeof Events)[keyof typeof Events];

export type Phase =
  | 'setup'
  | 'resource_gathering'
  | 'puzzle_preparation'
  | 'puzzle_assembly'
  | 'analytics';

export interface Position {
  x: number;
  y: number;
}

// ── Setup payloads ─────────────────────────────────────────────────────────

export interface HostGameConfig {
  minPlayers: number;
  maxPlayers: number;
  resourceGatheringRounds: number;
  triviaAnswerTime: number;
  triviaGraceTime: number;
  puzzleBaseTime: number;
  difficultyMode: string;
}

export interface HostConnectionConfirmed {
  hostId: string;
  currentPhase: Phase;
  isReconnection: boolean;
  gameConfig: HostGameConfig;
}

export interface ExistingConfiguration {
  selectedRole: string | null;
  selectedSpecialties: string[];
  playerName: string;
  ready: boolean;
}

export interface PlayerConnectionConfirmed {
  playerId: string;
  currentPhase: Phase;
  isReconnection: boolean;
  existingConfiguration: ExistingConfiguration | null;
}

export interface RoleInfo {
  roleType: string;
  displayName: string;
  resourceBonus: number;
  bonusTokenType: string;
  description: string;
  available: boolean;
}

export interface RolesAvailable {
  roles: RoleInfo[];
  triviaCategories: string[];
  maxSpecialties: number;
}

export interface PlayerConfiguration {
  selectedRole: string;
  selectedSpecialties: string[];
  playerName: string;
}

export interface LobbyStatus {
  currentPlayers: number;
  minPlayers: number;
  maxPlayers: number;
  playerRoles: Record<string, number>;
  hasHost: boolean;
  allPlayersReady: boolean;
  readyPlayers: number;
  gameStartEligible: boolean;
  waitingMessage: string;
}

export interface PlayerStatus {
  playerName: string;
  role: string | null;
  specialties: string[];
  connected: boolean;
  ready: boolean;
  lastActivity: string;
}

export interface PlayerRoster {
  phase: Phase;
  connectedPlayers: number;
  readyPlayers: number;
  gameStartEligible: boolean;
  playerStatuses: Record<string, PlayerStatus>;
  roleDistribution: Record<string, number>;
}

export interface TeamTokens {
  anchorTokens: number;
  chronosTokens: number;
  guideTokens: number;
  clarityTokens: number;
}

export interface GameStarted {
  phase: Phase;
  totalPlayers: number;
  initialTeamTokens: TeamTokens;
}

// ── Resource payloads ──────────────────────────────────────────────────────

export interface ThresholdSet {
  anchor: number;
  chronos: number;
  guide: number;
  clarity: number;
}

export interface ResourcePhaseStart {
  phase: Phase;
  totalRounds: number;
  roundDuration: number;
  answerTime: number;
  graceTime: number;
  tokenThresholds: ThresholdSet;
  difficultySettings: {
    mode: string;
    specialtyProbability: number;
    timeMultiplier: number;
    thresholdMultiplier: number;
  };
}

export interface HostResourcePhaseStart {
  phase: Phase;
  monitoringDashboard: {
    totalRounds: number;
    currentRound: number;
    roundDuration: number;
    playerDistribution: Record<string, number>;
  };
}

export interface LocationVerified {
  stationHash: string;
  previousLocation?: string | null;
  scanTimestamp: string;
}

export interface LocationConfirmed {
  newLocation: string;
}

export interface TriviaQuestion {
  questionId: string;
  questionText: string;
  category: string;
  difficulty: string;
  isSpecialty: boolean;
  options: string[];
  roundNumber: number;
  totalRounds: number;
  answerDeadline: string;
}

export interface TriviaAnswer {
  questionId: string;
  answerIndex: number;
  timeElapsed: number;
}

export interface AnswerResult {
  questionId: string;
  correct: boolean;
  selectedAnswer: string | null;
  correctAnswer: string;
  tokensEarned: number;
  baseTokens: number;
  bonuses: {
    roleBonus: boolean;
    roleBonusTokens: number;
    specialtyBonus: boolean;
    specialtyBonusTokens: number;
  };
  currentLocation: string;
  nextTriviaTimestamp: string;
}

export interface TeamProgress {
  currentRound: number;
  totalRounds: number;
  questionsAnswered: number;
  totalQuestions: number;
  teamTokens: TeamTokens;
  currentThresholds: ThresholdSet;
  teamPerformance: {
    averageAccuracy: number;
    roundTimeRemaining: number;
  };
}

export interface RoundAnalytics {
  currentRound: number;
  totalRounds: number;
  roundResults: {
    questionsDelivered: number;
    answersReceived: number;
    correctAnswers: number;
    averageResponseTime: number;
    tokensAwarded: number;
  };
  playerPerformance: Record<
    string,
    {
      location: string;
      answerCorrect: boolean;
      responseTime: number;
      tokensEarned: number;
      runningAccuracy: number;
    }
  >;
  stationDistribution: Record<string, number>;
  teamTokens: TeamTokens;
}

export interface BonusEffects {
  anchorPreSolved: number;
  chronosTimeBonus: number;
  guideHighlightCount: number;
  clarityPreviewDuration: number;
}

export interface ResourcePhaseComplete {
  phase: Phase;
  nextPhase: Phase;
  finalTokenTotals: TeamTokens;
  thresholdAchievements: ThresholdSet;
  bonusEffects: BonusEffects;
}

export interface HostResourcePhaseComplete {
  phase: Phase;
  totalQuestionsAnswered: number;
  teamPerformance: {
    overallAccuracy: number;
    totalTokensEarned: number;
    averageResponseTime: number;
  };
  finalTokenDistribution: TeamTokens;
  playerAnalytics: Record<
    string,
    {
      questionsAnswered: number;
      correctAnswers: number;
      accuracy: number;
      tokensEarned: number;
      specialtyPerformance: {
        questionsReceived: number;
        correctAnswers: number;
        bonusTokens: number;
      };
    }
  >;
  readyForPuzzlePhase: boolean;
}

// ── Puzzle payloads ────────────────────────────────────────────────────────

export interface PuzzlePhaseLoad {
  phase: Phase;
  imageId: string;
  assignedSegmentId: string;
  individualPuzzleSize: number;
  anchorPreSolvedPieces: number;
  centralGridSize: number;
  totalFragments: number;
  clarityPreviewDuration: number;
  guideHighlightCount: number;
}

export interface HostPuzzlePhaseLoad {
  phase: Phase;
  imageId: string;
  centralGridSize: number;
  totalFragments: number;
  playerCount: number;
  playerSegmentAssignments: Record<string, string>;
  bonusEffects: BonusEffects;
}

export interface PuzzlePhaseStart {
  startTimestamp: string;
  totalTime: number;
  baseTime: number;
  chronosBonus: number;
  clarityPreviewActive: boolean;
  clarityPreviewDuration: number;
  playerPhases: { phase2a: string[]; phase2b: string[] };
}

export interface HostPuzzlePhaseStart {
  timerActive: boolean;
  startTimestamp: string;
  totalTime: number;
  baseTime: number;
  chronosBonus: number;
  playersInPhase2a: number;
  playersInPhase2b: number;
}

export interface SegmentCompleted {
  segmentId: string;
  completionTimestamp: string;
  solveTime: number;
  manualPiecesSolved: number;
  preSolvedPieces: number;
}

export interface SegmentAcknowledged {
  segmentId: string;
  position: Position;
}

export interface HostSegmentCompleted {
  playerId: string;
  playerName: string;
  segmentId: string;
  completionTime: number;
  position: Position;
  phaseTransition: { playersInPhase2a: number; playersInPhase2b: number };
  completionStats: {
    totalCompleted: number;
    totalRequired: number;
    unassignedFragments: number;
  };
}

export interface FragmentMove {
  segmentId: string;
  targetPosition: Position;
  swapWithSegmentId?: string | null;
}

export type MoveRejectReason = 'cooldown' | 'not_owner' | 'target_invalid' | 'phase_invalid';

export interface CooldownInfo {
  nextMoveAvailable: string;
  cooldownRemaining: number;
}

export interface MoveResult {
  moveId: string;
  status: 'success' | 'rejected';
  segmentId: string;
  newPosition?: Position;
  swappedSegmentId?: string;
  swappedSegmentNewPosition?: Position;
  reason?: MoveRejectReason;
  cooldownInfo?: CooldownInfo;
}

export interface GridFragment {
  segmentId: string;
  playerId: string | null;
  playerName: string | null;
  position: Position;
}

export interface GridState {
  fragments: GridFragment[];
  timeRemaining: number;
}

export interface HostGridState {
  fragments: Array<{
    playerId: string | null;
    playerName: string | null;
    segmentId: string;
    position: Position;
    lastMoved: string;
    moveCount: number;
  }>;
  playerMetrics: Record<
    string,
    {
      phase: 'phase2a' | 'phase2b';
      fragmentsOwned: number;
      movesContributed: number;
      successfulMoves: number;
      lastActivity: string;
    }
  >;
  activeRecommendations: number;
  timeRemaining: number;
}

export interface PersonalState {
  guideHighlights: Position[];
}

export interface RecommendMove {
  targetPlayerId: string;
  fromSegmentId: string;
  toSegmentId: string;
  reasoning: string;
}

export interface MoveRecommendation {
  moveId: string;
  fromPlayerId: string;
  fromPlayerName: string;
  targetPlayerId: string;
  fromSegmentId: string;
  toSegmentId: string;
  reasoning: string;
  expiresAt: string;
}

export interface RecommendationResponse {
  moveId: string;
  response: 'accept' | 'reject';
  responseReason?: string | null;
}

export interface RecommendationResult {
  moveId: string;
  targetPlayerId: string;
  targetPlayerName: string;
  response: 'accept' | 'reject';
  responseReason?: string;
  swapExecuted?: {
    segment1Id: string;
    segment1OldPosition: Position;
    segment1NewPosition: Position;
    segment2Id: string;
    segment2OldPosition: Position;
    segment2NewPosition: Position;
  };
}

export interface RecommendationExpired {
  moveId: string;
  reason: 'timeout' | 'player_disconnected';
}

export interface CompletedSuccess {
  success: true;
  completionTime: number;
  totalTime: number;
  timeRemaining: number;
  finalGridState: {
    allFragmentsCorrect: boolean;
    totalFragments: number;
    correctFragments: number;
  };
}

export interface CompletedTimeout {
  success: false;
  reason: 'time_expired';
  totalTime: number;
  timeExpired: boolean;
  finalStats: {
    fragmentsPlaced: number;
    totalFragments: number;
    correctlyPlaced: number;
    completionPercentage: number;
  };
}

export interface CompletionAnalytics {
  puzzleSuccess: boolean;
  completionTime: number;
  totalTime: number;
  playerContributions: Record<
    string,
    {
      individualSolveTime: number;
      fragmentMoves: number;
      successfulMoves: number;
      recommendationsSent: number;
      recommendationsReceived: number;
      recommendationsAccepted: number;
      finalFragmentCorrect: boolean;
    }
  >;
  collaborationMetrics: {
    totalMoves: number;
    successfulMoves: number;
    totalRecommendations: number;
    acceptedRecommendations: number;
    averageResponseTime: number;
  };
  phaseTransitions: {
    playersCompletedIndividual: number;
    averageIndividualTime: number;
    fastestIndividual: number;
    slowestIndividual: number;
  };
}

// ── Analytics payloads ─────────────────────────────────────────────────────

export interface ScoreBreakdown {
  triviaPoints: number;
  specialtyPoints: number;
  completionBonus: number;
  movePoints: number;
  recommendationPoints: number;
  totalScore: number;
}

export interface PersonalReport {
  playerId: string;
  playerName: string;
  gameSuccess: boolean;
  personalScore: number;
  rank: number;
  totalPlayers: number;
  tokenCollection: TeamTokens & { totalTokens: number };
  triviaPerformance: {
    totalQuestions: number;
    correctAnswers: number;
    accuracy: number;
    accuracyByCategory: Record<string, number>;
    specialtyPerformance: {
      specialtyQuestions: number;
      specialtyCorrect: number;
      specialtyAccuracy: number;
      bonusTokens: number;
    };
    averageResponseTime: number;
  };
  puzzleSolvingMetrics: {
    individualSolveTime: number;
    individualRank: number;
    fragmentMoves: number;
    successfulMoves: number;
    moveAccuracy: number;
    recommendationsSent: number;
    recommendationsReceived: number;
    recommendationsAccepted: number;
  };
  scoreBreakdown: ScoreBreakdown;
}

export interface LeaderboardEntry {
  playerId: string;
  playerName: string;
  totalScore: number;
  rank: number;
  role: string;
}

export interface TeamSummary {
  gameSuccess: boolean;
  totalScore: number;
  totalPlayers: number;
  totalGameTime: number;
  teamPerformance: {
    overallAccuracy: number;
    totalTokensCollected: number;
    thresholdAchievements: ThresholdSet;
    puzzleCompletionTime: number;
  };
  leaderboard: LeaderboardEntry[];
}

export interface HostCompleteReport {
  gameSuccess: boolean;
  totalGameTime: number;
  totalPlayers: number;
  difficultyMode: string;
  overallPerformance: {
    totalScore: number;
    averageScore: number;
    completionRate: number;
  };
  resourceGatheringAnalytics: {
    totalRounds: number;
    questionsAnswered: number;
    overallAccuracy: number;
    tokenDistribution: TeamTokens;
    playerPerformance: Record<string, unknown>;
  };
  puzzleAssemblyAnalytics: {
    totalTime: number;
    completionTime: number;
    timeUtilization: number;
    individualPhaseMetrics: {
      averageSolveTime: number;
      fastestCompletion: number;
      slowestCompletion: number;
      preSolvedPiecesUsed: number;
    };
    collaborativePhaseMetrics: {
      totalMoves: number;
      successfulMoves: number;
      moveAccuracy: number;
      totalRecommendations: number;
      acceptedRecommendations: number;
      recommendationAcceptanceRate: number;
    };
    playerContributions: Record<string, unknown>;
  };
  categoryPerformance: Record<
    string,
    { questionsAsked: number; correctAnswers: number; accuracy: number }
  >;
  timelineAnalysis: {
    setupPhase: number;
    resourcePhase: number;
    preparationPhase: number;
    puzzlePhase: number;
  };
}

export interface GameReset {
  reason: string;
  reconnectRequired: boolean;
  reconnectInstructions: string;
  newGameAvailable: boolean;
}

// ── System payloads ────────────────────────────────────────────────────────

export interface ErrorPayload {
  errorType: 'auth_error' | 'validation_error' | 'game_state_error';
  errorCode: string;
  message: string;
  details?: string;
  context?: unknown;
  suggestedActions?: string[];
}

export interface HostDisconnected {
  hostStatus: string;
  currentPhase: Phase;
  gameImpact: { canContinue: boolean; affectedFeatures: string[] };
  timerPausedAt?: string;
}

export interface HostReconnected {
  hostStatus: string;
  currentPhase: Phase;
  restoredFeatures: string[];
  timeRemaining?: number;
}

export interface PlayerDisconnected {
  playerId: string;
  playerName: string;
  disconnectionTime: string;
  currentPhase: Phase;
  updatedCounts?: {
    connectedPlayers: number;
    readyPlayers: number;
    roleDistribution: Record<string, number>;
  };
  fragmentHandling?: {
    segmentId: string;
    newPosition: Position;
    nowUnassigned: boolean;
  };
  updatedPlayerCount?: number;
}

export interface Pong {
  serverTimestamp: string;
  clientTimestamp: string;
  sequenceNumber: number;
}
