// Game Constants - Sky Blue Theme

// Game Phases
export const GamePhase = {
  SETUP: 'SETUP',
  RESOURCE_GATHERING: 'RESOURCE_GATHERING',
  PUZZLE_ASSEMBLY: 'PUZZLE_ASSEMBLY',
  POST_GAME: 'POST_GAME'
};

// WebSocket Message Types - Matching server protocol
export const MessageType = {
  // Server to Client
  AVAILABLE_ROLES: 'available_roles',
  GAME_LOBBY_STATUS: 'game_lobby_status',
  RESOURCE_PHASE_START: 'resource_phase_start',
  TRIVIA_QUESTION: 'trivia_question',
  TEAM_PROGRESS_UPDATE: 'team_progress_update',
  IMAGE_PREVIEW: 'image_preview',
  PUZZLE_PHASE_LOAD: 'puzzle_phase_load',
  PUZZLE_PHASE_START: 'puzzle_phase_start',
  SEGMENT_COMPLETION_ACK: 'segment_completion_ack',
  PERSONAL_PUZZLE_STATE: 'personal_puzzle_state',
  CENTRAL_PUZZLE_STATE: 'central_puzzle_state',
  FRAGMENT_MOVE_RESPONSE: 'fragment_move_response',
  PIECE_RECOMMENDATION: 'piece_recommendation',
  GAME_ANALYTICS: 'game_analytics',
  GAME_RESET: 'game_reset',
  ERROR: 'error',
  HOST_UPDATE: 'host_update',
  
  // Client to Server
  ROLE_SELECTION: 'role_selection',
  TRIVIA_SPECIALTY_SELECTION: 'trivia_specialty_selection',
  RESOURCE_LOCATION_VERIFIED: 'resource_location_verified',
  TRIVIA_ANSWER: 'trivia_answer',
  SEGMENT_COMPLETED: 'segment_completed',
  FRAGMENT_MOVE_REQUEST: 'fragment_move_request',
  PIECE_RECOMMENDATION_REQUEST: 'piece_recommendation_request',
  PIECE_RECOMMENDATION_RESPONSE: 'piece_recommendation_response',
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

// Sky Blue Color Palette
export const Colors = {
  // Primary colors
  primary: '#87CEEB',      // Sky blue
  secondary: '#B0E0E6',    // Powder blue
  tertiary: '#E0F2FE',     // Light sky
  accent: '#4A9FD5',       // Bright blue accent
  
  // Backgrounds
  background: '#FFFFFF',
  surface: '#F8FBFF',      // Very light blue
  surfaceAlt: '#F0F7FF',   // Slightly darker light blue
  
  // Text colors
  text: {
    primary: '#1A365D',    // Dark blue
    secondary: '#2C5282',  // Medium blue
    light: '#718096'       // Gray blue
  },
  
  // Token colors - Pastel versions
  token: {
    anchor: '#C4B5FD',     // Pastel purple
    chronos: '#93C5FD',    // Pastel blue
    guide: '#86EFAC',      // Pastel green
    clarity: '#FDE68A'     // Pastel yellow
  },
  
  // Status colors
  success: '#34D399',      // Mint green
  error: '#EF4444',        // Coral red
  warning: '#F59E0B',      // Amber
  info: '#60A5FA',         // Light blue
  
  // Gradient colors
  gradient: {
    start: '#FFFFFF',
    middle: '#F0F9FF',
    end: '#E0F2FE'
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

// Resource Station Hashes (from server)
export const RESOURCE_HASHES = {
  ANCHOR: 'HASH_ANCHOR_STATION_2025',
  CHRONOS: 'HASH_CHRONOS_STATION_2025',
  GUIDE: 'HASH_GUIDE_STATION_2025',
  CLARITY: 'HASH_CLARITY_STATION_2025'
};

// Token Thresholds
export const TokenThresholds = {
  ANCHOR: 5,
  CHRONOS: 5,
  GUIDE: 5,
  CLARITY: 5
};

// Game Configuration
export const GAME_CONFIG = {
  MIN_PLAYERS: 4,
  MAX_PLAYERS: 64,
  INDIVIDUAL_PUZZLE_PIECES: 16,
  MOVEMENT_COOLDOWN: 1000, // 1 second
  BASE_PUZZLE_TIME: 300,   // 5 minutes
  CHRONOS_TIME_BONUS: 20,  // seconds per threshold
  CLARITY_PREVIEW_BASE: 3, // base seconds
  CLARITY_PREVIEW_BONUS: 1 // seconds per threshold
};

// WebSocket Configuration
export const WS_URL = process.env.REACT_APP_WS_URL || 'ws://localhost:8080/ws';

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
    description: 'Pre-solve puzzle pieces',
    color: Colors.token.anchor
  },
  [TokenType.CHRONOS]: {
    name: 'Time Station',
    icon: '⏰',
    description: 'Extend puzzle time',
    color: Colors.token.chronos
  },
  [TokenType.GUIDE]: {
    name: 'Guide Station',
    icon: '🧭',
    description: 'Get placement hints',
    color: Colors.token.guide
  },
  [TokenType.CLARITY]: {
    name: 'Clarity Station',
    icon: '💎',
    description: 'Preview complete image',
    color: Colors.token.clarity
  }
};

// Trivia Configuration
export const TRIVIA_CATEGORIES = [
  'general',
  'geography',
  'history',
  'music',
  'science',
  'video_games'
];

// Error Messages
export const ERROR_MESSAGES = {
  CONNECTION_FAILED: 'Unable to connect to game server',
  CAMERA_PERMISSION_DENIED: 'Camera permission required for QR scanning',
  QR_SCAN_FAILED: 'QR scan failed - try manual entry',
  INVALID_CODE: 'Invalid station code',
  WEBSOCKET_ERROR: 'Connection error - reconnecting...',
  ROLE_UNAVAILABLE: 'Role unavailable - please choose another',
  MOVEMENT_COOLDOWN: 'Please wait before moving again',
  OWNERSHIP_ERROR: 'You can only move your own fragment'
};

// Success Messages
export const SUCCESS_MESSAGES = {
  CONNECTED: 'Connected to game!',
  LOCATION_VERIFIED: 'Location verified!',
  ANSWER_CORRECT: 'Correct! Tokens earned',
  PUZZLE_COMPLETE: 'Puzzle complete!',
  FRAGMENT_MOVED: 'Fragment moved successfully'
};

// Haptic Patterns
export const HAPTIC_PATTERNS = {
  LIGHT: 20,
  MEDIUM: 30,
  STRONG: 50,
  SUCCESS: [50, 30, 50],
  ERROR: [100, 50, 100],
  VICTORY: [100, 50, 100, 50, 200, 100, 300]
};
