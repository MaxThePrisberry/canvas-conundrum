// Game Phases
export const GamePhase = {
  SETUP: 'SETUP',
  RESOURCE_GATHERING: 'RESOURCE_GATHERING',
  PUZZLE_ASSEMBLY: 'PUZZLE_ASSEMBLY',
  POST_GAME: 'POST_GAME'
};

// WebSocket Message Types - Server to Client
export const MessageType = {
  AVAILABLE_ROLES: 'available_roles',
  GAME_LOBBY_STATUS: 'game_lobby_status',
  RESOURCE_PHASE_START: 'resource_phase_start',
  TRIVIA_QUESTION: 'trivia_question',
  TEAM_PROGRESS_UPDATE: 'team_progress_update',
  PUZZLE_PHASE_LOAD: 'puzzle_phase_load',
  PUZZLE_PHASE_START: 'puzzle_phase_start',
  SEGMENT_COMPLETION_ACK: 'segment_completion_ack',
  FRAGMENT_MOVE_RESPONSE: 'fragment_move_response',
  CENTRAL_PUZZLE_STATE: 'central_puzzle_state',
  GAME_ANALYTICS: 'game_analytics',
  GAME_RESET: 'game_reset',
  ERROR: 'error',
  HOST_UPDATE: 'host_update',
  COUNTDOWN: 'countdown',
  
  // Client to Server
  PLAYER_JOIN: 'player_join',
  ROLE_SELECTION: 'role_selection',
  TRIVIA_SPECIALTY_SELECTION: 'trivia_specialty_selection',
  RESOURCE_LOCATION_VERIFIED: 'resource_location_verified',
  TRIVIA_ANSWER: 'trivia_answer',
  SEGMENT_COMPLETED: 'segment_completed',
  FRAGMENT_MOVE_REQUEST: 'fragment_move_request',
  PLAYER_READY: 'player_ready',
  HOST_START_GAME: 'host_start_game',
  HOST_START_PUZZLE: 'host_start_puzzle'
};

// Token Types
export const TokenType = {
  ANCHOR: 'anchor',
  CHRONOS: 'chronos',
  GUIDE: 'guide',
  CLARITY: 'clarity'
};

// Role Types
export const RoleType = {
  ART_ENTHUSIAST: 'art_enthusiast',
  DETECTIVE: 'detective',
  TOURIST: 'tourist',
  JANITOR: 'janitor'
};

// Colors
export const Colors = {
  primary: '#2DD4BF', // Turquoise/Teal
  secondary: '#14B8A6',
  tertiary: '#0D9488',
  accent: '#5EEAD4',
  background: '#FFFFFF',
  surface: '#F0FDFA',
  text: {
    primary: '#134E4A',
    secondary: '#0F766E',
    light: '#5EEAD4'
  },
  token: {
    anchor: '#7C3AED', // Purple
    chronos: '#2563EB', // Blue
    guide: '#10B981', // Green
    clarity: '#F59E0B' // Amber
  },
  success: '#10B981',
  error: '#EF4444',
  warning: '#F59E0B'
};

// Animation Durations
export const AnimationDuration = {
  SHORT: 0.3,
  MEDIUM: 0.6,
  LONG: 1.0,
  PHASE_TRANSITION: 1.5,
  CELEBRATION: 2.0
};

// Token Thresholds (matching server constants)
export const TokenThresholds = {
  ANCHOR: 5,
  CHRONOS: 5,
  GUIDE: 5,
  CLARITY: 5
};

// Swap Request Timeout
export const SWAP_REQUEST_TIMEOUT = 10000; // 10 seconds

// WebSocket Configuration
export const WS_URL = process.env.REACT_APP_WS_URL || 'ws://localhost:8080/ws';

// Puzzle Configuration
export const PUZZLE_GRID_SIZE = 4; // 4x4 grid for individual puzzle
export const PUZZLE_PIECES = 16;