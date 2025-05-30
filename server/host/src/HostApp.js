import React, { useState, useEffect, useCallback } from 'react';
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
  const [hasAttemptedConnection, setHasAttemptedConnection] = useState(false); // Track if user tried to connect
  const [hashCode, setHashCode] = useState('');
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
    lobbyStatus: null // Add lobby status to game state
  });

  const { 
    isConnected: wsConnected, 
    isReconnecting, 
    sendMessage, 
    lastMessage,
    connect: connectWebSocket,
    disconnect: disconnectWebSocket
  } = useHostWebSocket();

  // Handle WebSocket connection status - only set isConnected if we've attempted to connect
  useEffect(() => {
    if (hasAttemptedConnection) {
      setIsConnected(wsConnected);
    }
  }, [wsConnected, hasAttemptedConnection]);

  // Handle incoming WebSocket messages
  useEffect(() => {
    if (!lastMessage) return;

    const { type, payload } = lastMessage;
    console.log('Processing host message:', type, payload);

    switch (type) {
      case MessageType.AVAILABLE_ROLES:
        // Host connection confirmation
        if (payload.isHost) {
          setIsConnected(true);
          console.log('Host connected successfully:', payload);
        }
        break;

      case MessageType.GAME_LOBBY_STATUS:
        // Handle lobby status updates
        console.log('Received lobby status:', payload);
        setGameState(prev => ({
          ...prev,
          connectedPlayers: payload.currentPlayers || prev.connectedPlayers,
          lobbyStatus: payload
        }));
        break;

      case MessageType.HOST_UPDATE:
        console.log('Received host update:', payload);
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
        console.log('Received game analytics:', payload);
        setGameState(prev => ({
          ...prev,
          phase: GamePhase.POST_GAME,
          analyticsData: payload
        }));
        break;

      case MessageType.GAME_RESET:
        console.log('Game reset received');
        // Reset to initial state
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
        break;

      case MessageType.ERROR:
        console.error('Host error received:', payload);
        // Handle specific error types
        if (payload.type === 'host_disconnected') {
          // Don't auto-disconnect on host_disconnected errors
          console.log('Host disconnection error - maintaining connection');
        }
        break;

      default:
        console.log('Unhandled host message type:', type, payload);
    }
  }, [lastMessage]);

  // Host action handlers
  const handleStartGame = useCallback(() => {
    if (!isConnected || !hashCode) {
      console.error('Cannot start game: not connected or no hash code');
      return;
    }

    console.log('Starting game as host');
    sendMessage({
      type: MessageType.HOST_START_GAME,
      auth: { playerId: hashCode }, // Using hashCode as playerId for host
      payload: {}
    });
  }, [sendMessage, hashCode, isConnected]);

  const handleStartPuzzle = useCallback(() => {
    if (!isConnected || !hashCode) {
      console.error('Cannot start puzzle: not connected or no hash code');
      return;
    }

    console.log('Starting puzzle as host');
    sendMessage({
      type: MessageType.HOST_START_PUZZLE,
      auth: { playerId: hashCode },
      payload: {}
    });
  }, [sendMessage, hashCode, isConnected]);

  const handleConnect = useCallback((code) => {
    console.log('Attempting to connect with code:', code);
    setHashCode(code);
    setHasAttemptedConnection(true); // Mark that user attempted to connect
    connectWebSocket(code);
  }, [connectWebSocket]);

  const handleDisconnect = useCallback(() => {
    console.log('Host disconnecting');
    if (disconnectWebSocket) {
      disconnectWebSocket();
    }
    setIsConnected(false);
    setHasAttemptedConnection(false); // Reset connection attempt state
    setHashCode('');
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
  }, [disconnectWebSocket]);

  // Debug logging
  useEffect(() => {
    console.log('Host state update:', {
      isConnected,
      wsConnected,
      isReconnecting,
      hasAttemptedConnection,
      hashCode,
      gamePhase: gameState.phase,
      connectedPlayers: gameState.connectedPlayers
    });
  }, [isConnected, wsConnected, isReconnecting, hasAttemptedConnection, hashCode, gameState.phase, gameState.connectedPlayers]);

  return (
    <div className="HostApp">
      {/* Only show connection overlay if user has attempted to connect */}
      {hasAttemptedConnection && (
        <ConnectionOverlay 
          isConnected={wsConnected} 
          isReconnecting={isReconnecting} 
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
