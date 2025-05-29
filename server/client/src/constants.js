// Game Constants - Ocean/Sky Theme

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

// Ocean/Sky Color Palette
export const Colors = {
  // Primary ocean colors
  primary: '#0EA5E9',      // Sky blue
  secondary: '#0284C7',    // Ocean blue
  tertiary: '#0369A1',     // Deep ocean
  accent: '#38BDF8',       // Light sky
  
  // Backgrounds
  background: '#FFFFFF',
  surface: '#F0F9FF',      // Light ocean
  surfaceAlt: '#E0F2FE',   // Lighter ocean
  
  // Text colors
  text: {
    primary: '#0F172A',    // Dark navy
    secondary: '#334155',  // Medium gray
    light: '#64748B'       // Light gray
  },
  
  // Token colors - Ocean themed
  token: {
    anchor: '#7C3AED',     // Purple (mystic ocean)
    chronos: '#2563EB',    // Blue (time ocean)
    guide: '#10B981',      // Green (sea green)
    clarity: '#F59E0B'     // Amber (sunset)
  },
  
  // Status colors
  success: '#10B981',      // Sea green
  error: '#EF4444',        // Coral red
  warning: '#F59E0B',      // Sunset orange
  info: '#0EA5E9',         // Sky blue
  
  // Ocean gradient stops
  ocean: {
    50: '#F0F9FF',
    100: '#E0F2FE',
    200: '#BAE6FD',
    300: '#7DD3FC',
    400: '#38BDF8',
    500: '#0EA5E9',
    600: '#0284C7',
    700: '#0369A1',
    800: '#075985',
    900: '#0C4A6E'
  }
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

// Base tokens per correct answer
export const BASE_TOKENS_PER_ANSWER = 10;

// Swap Request Timeout
export const SWAP_REQUEST_TIMEOUT = 10000; // 10 seconds

// WebSocket Configuration
export const WS_URL = process.env.REACT_APP_WS_URL || 'ws://localhost:8080/ws';

// Puzzle Configuration
export const PUZZLE_GRID_SIZE = 4; // 4x4 grid for individual puzzle
export const PUZZLE_PIECES = 16;

// Trivia Configuration
export const TRIVIA_TIME_WARNING = 5; // Show warning when 5 seconds remain
export const TRIVIA_CATEGORIES = [
  'general',
  'geography',
  'history',
  'music',
  'science',
  'video_games'
];

// QR Scanner Configuration
export const QR_SCANNER_CONFIG = {
  fps: 10,
  qrbox: { width: 250, height: 250 },
  aspectRatio: 1.0,
  showTorchButtonIfSupported: true,
  showZoomSliderIfSupported: true,
  defaultZoomValueIfSupported: 1.5
};

// Resource Station Configuration
export const RESOURCE_STATIONS = {
  [TokenType.ANCHOR]: {
    name: 'Anchor Station',
    icon: '⚓',
    description: 'Stability tokens'
  },
  [TokenType.CHRONOS]: {
    name: 'Time Station',
    icon: '⏰',
    description: 'Time extension tokens'
  },
  [TokenType.GUIDE]: {
    name: 'Guide Station',
    icon: '🧭',
    description: 'Hint tokens'
  },
  [TokenType.CLARITY]: {
    name: 'Clarity Station',
    icon: '💎',
    description: 'Preview time tokens'
  }
};

// Phase Icons
export const PHASE_ICONS = {
  [GamePhase.SETUP]: '🎯',
  [GamePhase.RESOURCE_GATHERING]: '🏃‍♂️',
  [GamePhase.PUZZLE_ASSEMBLY]: '🧩',
  [GamePhase.POST_GAME]: '🏆'
};

// Error Messages
export const ERROR_MESSAGES = {
  CONNECTION_FAILED: 'Unable to connect to game server. Please check your connection.',
  CAMERA_PERMISSION_DENIED: 'Camera permission denied. Please enable camera access to scan QR codes.',
  QR_SCAN_FAILED: 'Failed to scan QR code. Please try again or enter the code manually.',
  INVALID_CODE: 'Invalid station code. Please check and try again.',
  WEBSOCKET_ERROR: 'Connection error. Attempting to reconnect...',
  ROLE_UNAVAILABLE: 'This role is no longer available. Please choose another.',
  GAME_FULL: 'The game is full. Please wait for the next game.',
  PHASE_ERROR: 'Unable to proceed to the next phase. Please wait.'
};

// Success Messages
export const SUCCESS_MESSAGES = {
  CONNECTED: 'Connected to game server!',
  LOCATION_VERIFIED: 'Location verified successfully!',
  ANSWER_CORRECT: 'Correct! +10 tokens earned',
  SEGMENT_COMPLETED: 'Puzzle segment completed!',
  GAME_WON: 'Victory! Masterpiece restored!'
};

// Haptic Patterns (in milliseconds)
export const HAPTIC_PATTERNS = {
  LIGHT: 20,
  MEDIUM: 30,
  STRONG: 50,
  SUCCESS: [50, 30, 50],
  ERROR: [100, 50, 100],
  VICTORY: [100, 50, 100, 50, 200]
};

// Local Storage Keys
export const STORAGE_KEYS = {
  PLAYER_ID: 'canvas_conundrum_player_id',
  SOUND_ENABLED: 'canvas_conundrum_sound_enabled',
  HAPTIC_ENABLED: 'canvas_conundrum_haptic_enabled'
};

// Game Configuration
export const GAME_CONFIG = {
  MIN_PLAYERS: 4,
  MAX_PLAYERS: 64,
  RECONNECT_ATTEMPTS: 5,
  RECONNECT_DELAY: 1000, // Base delay, exponential backoff
  LOBBY_COUNTDOWN: 30,
  RESOURCE_ROUND_DURATION: 180,
  PUZZLE_BASE_TIME: 300,
  POST_GAME_DURATION: 60
};
