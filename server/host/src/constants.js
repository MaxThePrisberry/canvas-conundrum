// Host Constants - Same as player but host-specific additions

// Game Phases
export const GamePhase = {
  SETUP: 'SETUP',
  RESOURCE_GATHERING: 'RESOURCE_GATHERING',
  PUZZLE_ASSEMBLY: 'PUZZLE_ASSEMBLY',
  POST_GAME: 'POST_GAME'
};

// WebSocket Message Types - Matching server protocol
export const MessageType = {
  // Server to Client (Host receives these)
  AVAILABLE_ROLES: 'available_roles',
  HOST_UPDATE: 'host_update',
  GAME_LOBBY_STATUS: 'game_lobby_status', // Added missing message type
  GAME_ANALYTICS: 'game_analytics',
  GAME_RESET: 'game_reset',
  ERROR: 'error',
  
  // Client to Server (Host sends these)
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
  
  // Phase colors
  phase: {
    setup: '#87CEEB',
    resource: '#34D399',
    puzzle: '#F59E0B',
    postgame: '#8B5CF6'
  }
};

// Host-specific Configuration
export const HOST_CONFIG = {
  RECONNECT_ATTEMPTS: 5,
  HEARTBEAT_INTERVAL: 30000, // 30 seconds
  UPDATE_INTERVAL: 1000,     // 1 second
  MIN_PLAYERS: 4,
  MAX_PLAYERS: 64
};

// Animation Durations
export const AnimationDuration = {
  SHORT: 0.3,
  MEDIUM: 0.6,
  LONG: 1.0,
  PHASE_TRANSITION: 1.5,
  CELEBRATION: 2.0
};

// WebSocket Configuration
export const WS_CONFIG = {
  BASE_URL: process.env.REACT_APP_WS_URL || 'ws://localhost:8080',
  RECONNECT_DELAY: 1000,
  MAX_RECONNECT_ATTEMPTS: 5
};

// Host Dashboard Layout
export const DASHBOARD_CONFIG = {
  GRID_BREAKPOINTS: {
    MOBILE: 768,
    TABLET: 1024,
    DESKTOP: 1200
  },
  PANEL_SIZES: {
    SMALL: 'span 1',
    MEDIUM: 'span 2',
    LARGE: 'span 3',
    FULL: 'span 4'
  }
};

// Status Types for Host UI
export const StatusType = {
  CONNECTED: 'connected',
  DISCONNECTED: 'disconnected',
  READY: 'ready',
  WAITING: 'waiting',
  ACTIVE: 'active',
  COMPLETED: 'completed'
};

// Host Error Messages
export const HOST_ERRORS = {
  CONNECTION_FAILED: 'Failed to connect to game server',
  INVALID_HASH: 'Invalid host code provided',
  ALREADY_CONNECTED: 'Another host is already connected',
  GAME_IN_PROGRESS: 'Cannot join - game already in progress',
  INSUFFICIENT_PLAYERS: 'Not enough players to start game',
  WEBSOCKET_ERROR: 'WebSocket connection error'
};

// Host Success Messages
export const HOST_SUCCESS = {
  CONNECTED: 'Successfully connected as host',
  GAME_STARTED: 'Game started successfully',
  PUZZLE_STARTED: 'Puzzle phase started',
  PHASE_TRANSITION: 'Phase transition completed'
};

// Chart Configuration for Analytics
export const CHART_CONFIG = {
  COLORS: {
    PRIMARY: '#87CEEB',
    SECONDARY: '#B0E0E6',
    SUCCESS: '#34D399',
    WARNING: '#F59E0B',
    ERROR: '#EF4444'
  },
  ANIMATION: {
    DURATION: 1000,
    DELAY: 100
  }
};

// Export default configuration object
export default {
  GamePhase,
  MessageType,
  TokenType,
  RoleType,
  Colors,
  HOST_CONFIG,
  AnimationDuration,
  WS_CONFIG,
  DASHBOARD_CONFIG,
  StatusType,
  HOST_ERRORS,
  HOST_SUCCESS,
  CHART_CONFIG
};
