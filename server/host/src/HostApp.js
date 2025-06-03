import React, { useState, useEffect, useCallback, useRef } from 'react';
import './HostApp.css';
import { AnimatePresence } from 'framer-motion';
import {
  HostLanding,
  HostDashboard,
  ConnectionOverlay
} from './components';
import { useHostWebSocket } from './hooks/useHostWebSocket';
import { GamePhase, MessageType } from './constants';

function HostApp() {
  const [isConnected, setIsConnected] = useState(false);
  const [hasAttemptedConnection, setHasAttemptedConnection] = useState(false);
  const [isConnecting, setIsConnecting] = useState(false); // Track initial connection attempts
  const [hashCode, setHashCode] = useState('');
  const [playerId, setPlayerId] = useState(''); // Store actual player ID
  const connectionTimeout = useRef(null);
  const [gameState, setGameState] = useState({
    phase: GamePhase.SETUP,
    connectedPlayers: 0,
    readyPlayers: 0,
    teamTokens: {
      anchorTokens: 0,
      chronosTokens: 0,
      guideTokens: 0,
      clarityTokens: 0
    },
    playerStatuses: {},
    questionsAnswered: 0,
    totalQuestions: 0,
    puzzleData: null,
    analyticsData: null,
    lobbyStatus: null
  });

  const { 
    isConnected: wsConnected, 
    isReconnecting, 
    sendMessage, 
    lastMessage,
    connect: connectWebSocket,
    disconnect: disconnectWebSocket
  } = useHostWebSocket();

  // Helper function to infer message type from content
  const inferMessageType = (message) => {
    // Check for explicit type field first
    if (message.type) {
      return message.type;
    }
    
    // Infer based on message content
    if (message.isHost !== undefined || (message.playerId && message.roles)) {
      return MessageType.AVAILABLE_ROLES;
    }
    if (message.phase !== undefined && message.connectedPlayers !== undefined) {
      return MessageType.HOST_UPDATE;
    }
    if (message.currentPlayers !== undefined && message.playerRoles !== undefined) {
      return MessageType.GAME_LOBBY_STATUS;
    }
    if (message.personalAnalytics || message.teamAnalytics || message.globalLeaderboard) {
      return MessageType.GAME_ANALYTICS;
    }
    if (message.reconnectRequired !== undefined) {
      return MessageType.GAME_RESET;
    }
    if (message.error !== undefined) {
      return MessageType.ERROR;
    }
    
    return 'unknown';
  };

  // Helper function to extract payload from message
  const extractPayload = (message, messageType) => {
    // If message has explicit payload, use it
    if (message.payload) {
      return message.payload;
    }
    
    // Otherwise, the entire message is the payload (except for type)
    const { type, ...payload } = message;
    return payload;
  };

  // Handle WebSocket connection status
  useEffect(() => {
    if (hasAttemptedConnection) {
      console.log('🔍 HOST CONNECTION STATUS ANALYSIS:');
      console.log('  - hasAttemptedConnection:', hasAttemptedConnection);
      console.log('  - wsConnected:', wsConnected);
      console.log('  - isConnected:', isConnected);
      console.log('  - isConnecting:', isConnecting);
      console.log('  - isReconnecting:', isReconnecting);
      
      if (wsConnected && !isConnected && !isConnecting) {
        console.log('📡 Setting isConnecting=true - WebSocket connected but no host confirmation yet');
        setIsConnecting(true);
      } else if (isConnected) {
        console.log('✅ Host confirmed - stopping connecting state');
        setIsConnecting(false);
      } else if (!wsConnected && !isReconnecting && isConnecting) {
        console.log('❌ Connection failed - WebSocket disconnected during connection attempt');
        setIsConnecting(false);
        if (connectionTimeout.current) {
          clearTimeout(connectionTimeout.current);
          connectionTimeout.current = null;
        }
      }
      
      // Determine what overlay state should be shown
      const shouldShowOverlay = !isConnected;
      const overlayState = isReconnecting ? 'RECONNECTING' : 
                          isConnecting ? 'CONNECTING' : 
                          'DISCONNECTED';
      
      console.log('🎭 OVERLAY STATE DETERMINATION:');
      console.log('  - shouldShowOverlay:', shouldShowOverlay);
      console.log('  - overlayState:', overlayState);
      
      if (shouldShowOverlay && overlayState === 'DISCONNECTED') {
        console.log('⚠️  WARNING: Showing DISCONNECTED overlay (this may appear as "Unknown")');
        console.log('  - This happens when hasAttemptedConnection=true but:');
        console.log('    - isConnected=false');
        console.log('    - isConnecting=false');
        console.log('    - isReconnecting=false');
      }
    }
  }, [wsConnected, isConnected, isReconnecting, hasAttemptedConnection, isConnecting]);

  // Handle incoming WebSocket messages
  useEffect(() => {
    if (!lastMessage) return;

    console.log('📨 RAW MESSAGE RECEIVED:', lastMessage);

    // Infer message type and extract payload
    const messageType = inferMessageType(lastMessage);
    const payload = extractPayload(lastMessage, messageType);
    
    console.log('🔍 MESSAGE PROCESSING:');
    console.log('  - Inferred type:', messageType);
    console.log('  - Extracted payload:', payload);
    console.log('  - Current state before processing:', {
      isConnected,
      isConnecting,
      hasAttemptedConnection,
      playerId
    });

    switch (messageType) {
      case MessageType.AVAILABLE_ROLES:
        console.log('🎯 Processing AVAILABLE_ROLES message');
        // Host connection confirmation
        if (payload.isHost) {
          console.log('✅ HOST CONFIRMATION RECEIVED!');
          console.log('  - Setting isConnected=true');
          console.log('  - Setting isConnecting=false');  
          console.log('  - Setting playerId=', payload.playerId);
          setIsConnected(true);
          setIsConnecting(false); // Stop connecting state
          setPlayerId(payload.playerId); // Store the actual player ID
          // Clear connection timeout
          if (connectionTimeout.current) {
            clearTimeout(connectionTimeout.current);
            connectionTimeout.current = null;
          }
        } else {
          console.log('❌ Received AVAILABLE_ROLES but isHost=false - this is unexpected for host client');
        }
        break;

      case MessageType.GAME_LOBBY_STATUS:
        console.log('🏠 Processing GAME_LOBBY_STATUS message');
        console.log('  - Current players:', payload.currentPlayers);
        setGameState(prev => ({
          ...prev,
          connectedPlayers: payload.currentPlayers || prev.connectedPlayers,
          lobbyStatus: payload
        }));
        break;

      case MessageType.HOST_UPDATE:
        console.log('🔄 Processing HOST_UPDATE message');
        console.log('  - Phase:', payload.phase);
        console.log('  - Connected players:', payload.connectedPlayers);
        console.log('  - Ready players:', payload.readyPlayers);
        setGameState(prev => ({
          ...prev,
          phase: payload.phase || prev.phase,
          connectedPlayers: payload.connectedPlayers ?? prev.connectedPlayers,
          readyPlayers: payload.readyPlayers ?? prev.readyPlayers,
          teamTokens: payload.teamTokens || prev.teamTokens,
          playerStatuses: payload.playerStatuses || prev.playerStatuses,
          questionsAnswered: payload.questionsAnswered ?? prev.questionsAnswered,
          totalQuestions: payload.totalQuestions ?? prev.totalQuestions,
          puzzleData: payload.puzzleData || prev.puzzleData,
          roundProgress: payload.roundProgress || prev.roundProgress,
          centralPuzzleState: payload.centralPuzzleState || prev.centralPuzzleState
        }));
        break;

      case MessageType.GAME_ANALYTICS:
        console.log('📊 Processing GAME_ANALYTICS message');
        setGameState(prev => ({
          ...prev,
          phase: GamePhase.POST_GAME,
          analyticsData: payload
        }));
        break;

      case MessageType.GAME_RESET:
        console.log('🔄 Processing GAME_RESET message');
        handleGameReset();
        break;

      case MessageType.ERROR:
        console.log('❌ Processing ERROR message:', payload);
        handleError(payload);
        break;

      default:
        console.log('❓ UNHANDLED MESSAGE TYPE:', messageType);
        console.log('  - Full message:', lastMessage);
        console.log('  - This could cause connection state issues!');
    }
    
    console.log('📊 STATE AFTER MESSAGE PROCESSING:', {
      isConnected,
      isConnecting,
      hasAttemptedConnection,
      playerId,
      messageType
    });
  }, [lastMessage, isConnected, isConnecting, hasAttemptedConnection, playerId, handleGameReset, handleError]);

  // Helper function to handle game reset
  const handleGameReset = useCallback(() => {
    console.log('🔄 HANDLING GAME RESET');
    console.log('  - Resetting all game state to initial values');
    setGameState({
      phase: GamePhase.SETUP,
      connectedPlayers: 0,
      readyPlayers: 0,
      teamTokens: {
        anchorTokens: 0,
        chronosTokens: 0,
        guideTokens: 0,
        clarityTokens: 0
      },
      playerStatuses: {},
      questionsAnswered: 0,
      totalQuestions: 0,
      puzzleData: null,
      analyticsData: null,
      lobbyStatus: null
    });
  }, []);

  // Helper function to handle errors
  const handleError = useCallback((errorPayload) => {
    console.log('❌ HANDLING ERROR:', errorPayload);
    
    // Clear connection timeout on any error
    if (connectionTimeout.current) {
      console.log('  - Clearing connection timeout due to error');
      clearTimeout(connectionTimeout.current);
      connectionTimeout.current = null;
    }
    
    if (errorPayload.type === 'host_disconnected') {
      console.log('  - Host disconnection error - maintaining connection');
    } else if (errorPayload.type === 'authentication_error') {
      console.log('  - Authentication error - setting isConnected=false, isConnecting=false');
      setIsConnected(false);
      setIsConnecting(false);
    } else if (errorPayload.type === 'validation_error') {
      console.log('  - Validation error - setting isConnecting=false');
      console.log('  - Details:', errorPayload.details);
      setIsConnecting(false);
    } else {
      console.log('  - General error - setting isConnecting=false');
      setIsConnecting(false);
    }
    
    console.log('⚠️  ERROR HANDLING COMPLETE - this may cause "Unknown" state if hasAttemptedConnection=true');
  }, []);

  // Host action handlers
  const handleStartGame = useCallback(() => {
    if (!isConnected || !playerId) {
      console.error('Cannot start game: not connected or no player ID');
      return;
    }

    console.log('Starting game as host');
    sendMessage({
      type: MessageType.HOST_START_GAME,
      auth: { playerId }, // Use the actual player ID from server
      payload: {}
    });
  }, [sendMessage, playerId, isConnected]);

  const handleStartPuzzle = useCallback(() => {
    if (!isConnected || !playerId) {
      console.error('Cannot start puzzle: not connected or no player ID');
      return;
    }

    console.log('Starting puzzle as host');
    sendMessage({
      type: MessageType.HOST_START_PUZZLE,
      auth: { playerId },
      payload: {}
    });
  }, [sendMessage, playerId, isConnected]);

  const handleConnect = useCallback((code) => {
    console.log('🔌 USER INITIATED CONNECTION');
    console.log('  - Host code:', code);
    console.log('  - Previous state:', {
      isConnected,
      isConnecting,
      hasAttemptedConnection,
      playerId
    });
    
    // Clear any existing timeout
    if (connectionTimeout.current) {
      console.log('  - Clearing existing connection timeout');
      clearTimeout(connectionTimeout.current);
    }
    
    setHashCode(code);
    setHasAttemptedConnection(true);
    setIsConnected(false); // Reset connection status
    setIsConnecting(true); // Start connecting
    setPlayerId(''); // Reset player ID
    
    console.log('  - Setting hasAttemptedConnection=true, isConnecting=true, isConnected=false');
    console.log('  - Starting 10-second connection timeout');
    
    connectWebSocket(code);
    
    // Set a timeout to stop connecting if it takes too long
    connectionTimeout.current = setTimeout(() => {
      console.log('⏰ CONNECTION TIMEOUT REACHED (10 seconds)');
      console.log('  - Setting isConnecting=false');
      console.log('  - This will likely cause "Unknown" state since hasAttemptedConnection=true but isConnecting=false');
      setIsConnecting(false);
    }, 10000); // 10 second timeout
  }, [connectWebSocket, isConnected, isConnecting, hasAttemptedConnection, playerId]);

  const handleDisconnect = useCallback(() => {
    console.log('🚪 USER INITIATED DISCONNECTION');
    console.log('  - Current state before disconnect:', {
      isConnected,
      isConnecting,
      hasAttemptedConnection,
      playerId
    });
    
    // Clear connection timeout
    if (connectionTimeout.current) {
      console.log('  - Clearing connection timeout');
      clearTimeout(connectionTimeout.current);
      connectionTimeout.current = null;
    }
    
    if (disconnectWebSocket) {
      console.log('  - Calling disconnectWebSocket()');
      disconnectWebSocket();
    }
    
    console.log('  - Resetting all connection states to false/empty');
    setIsConnected(false);
    setIsConnecting(false);
    setHasAttemptedConnection(false);
    setHashCode('');
    setPlayerId('');
    handleGameReset();
    
    console.log('  - Disconnect complete - should return to landing page');
  }, [disconnectWebSocket, handleGameReset, isConnected, isConnecting, hasAttemptedConnection, playerId]);

  // Debug logging
  useEffect(() => {
    console.log('🎛️  HOST DASHBOARD STATE UPDATE:', {
      isConnected,
      isConnecting,
      wsConnected,
      isReconnecting,
      hasAttemptedConnection,
      hashCode,
      playerId,
      gamePhase: gameState.phase,
      connectedPlayers: gameState.connectedPlayers
    });
    
    // Analyze overlay display logic
    const shouldShowOverlay = hasAttemptedConnection && !isConnected;
    console.log('🎭 OVERLAY ANALYSIS:');
    console.log('  - hasAttemptedConnection:', hasAttemptedConnection);
    console.log('  - isConnected:', isConnected);
    console.log('  - shouldShowOverlay:', shouldShowOverlay);
    
    if (shouldShowOverlay) {
      const overlayType = isReconnecting ? 'RECONNECTING' : 
                         isConnecting ? 'CONNECTING' : 
                         'DISCONNECTED (may show as Unknown)';
      console.log('  - overlayType will be:', overlayType);
      
      if (!isReconnecting && !isConnecting) {
        console.log('⚠️  POTENTIAL "UNKNOWN" STATE DETECTED!');
        console.log('  - Overlay will show because hasAttemptedConnection=true and isConnected=false');
        console.log('  - But neither isReconnecting nor isConnecting is true');
        console.log('  - This triggers the "Connection Lost" state which may appear as "Unknown"');
        console.log('  - Check if this happens after a failed connection attempt or error');
      }
    }
  }, [isConnected, isConnecting, wsConnected, isReconnecting, hasAttemptedConnection, hashCode, playerId, gameState.phase, gameState.connectedPlayers]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (connectionTimeout.current) {
        clearTimeout(connectionTimeout.current);
      }
    };
  }, []);

  return (
    <div className="HostApp">
      {/* Detailed logging for render decisions */}
      {(() => {
        const shouldShowOverlay = hasAttemptedConnection && !isConnected;
        const currentView = isConnected ? 'DASHBOARD' : 'LANDING';
        
        console.log('🎨 RENDER ANALYSIS:');
        console.log('  - Current view:', currentView);
        console.log('  - Should show overlay:', shouldShowOverlay);
        
        if (shouldShowOverlay) {
          const overlayProps = {
            isConnected: false,
            isReconnecting,
            isConnecting
          };
          console.log('  - Overlay props:', overlayProps);
          
          const overlayState = isReconnecting ? 'RECONNECTING' : 
                              isConnecting ? 'CONNECTING' : 
                              'DISCONNECTED';
          console.log('  - Overlay will show:', overlayState);
          
          if (overlayState === 'DISCONNECTED') {
            console.log('⚠️  OVERLAY SHOWING DISCONNECTED STATE - this may appear as "Unknown"!');
            console.log('  - This usually means connection failed or timed out');
            console.log('  - hasAttemptedConnection=true but no active connecting/reconnecting');
          }
        }
        
        return null; // This is just for logging
      })()}

      {/* Show connection overlay when attempting to connect or reconnecting */}
      {hasAttemptedConnection && !isConnected && (
        <ConnectionOverlay 
          isConnected={false} 
          isReconnecting={isReconnecting}
          isConnecting={isConnecting}
        />
      )}

      <AnimatePresence mode="wait">
        {!isConnected ? (
          <HostLanding
            key="landing"
            onConnect={handleConnect}
          />
        ) : (
          <HostDashboard
            key="dashboard"
            gameState={gameState}
            hashCode={hashCode}
            onStartGame={handleStartGame}
            onStartPuzzle={handleStartPuzzle}
            onDisconnect={handleDisconnect}
          />
        )}
      </AnimatePresence>
    </div>
  );
}

export default HostApp;
