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
    analyticsData: null
  });

  const { 
    isConnected: wsConnected, 
    isReconnecting, 
    sendMessage, 
    lastMessage,
    connect: connectWebSocket
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

    switch (type) {
      case MessageType.AVAILABLE_ROLES:
        // Host connection confirmation
        if (payload.isHost) {
          setIsConnected(true);
          console.log('Host connected successfully');
        }
        break;

      case MessageType.HOST_UPDATE:
        setGameState(prev => ({
          ...prev,
          phase: payload.phase || prev.phase,
          connectedPlayers: payload.connectedPlayers || 0,
          readyPlayers: payload.readyPlayers || 0,
          teamTokens: payload.teamTokens || prev.teamTokens,
          playerStatuses: payload.playerStatuses || {},
          questionsAnswered: payload.questionsAnswered,
          totalQuestions: payload.totalQuestions,
          puzzleData: payload.puzzleData,
          roundProgress: payload.roundProgress
        }));
        break;

      case MessageType.GAME_ANALYTICS:
        setGameState(prev => ({
          ...prev,
          phase: GamePhase.POST_GAME,
          analyticsData: payload
        }));
        break;

      case MessageType.GAME_RESET:
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
          analyticsData: null
        });
        break;

      case MessageType.ERROR:
        console.error('Host error:', payload);
        break;

      default:
        console.log('Unhandled host message type:', type);
    }
  }, [lastMessage]);

  // Host action handlers
  const handleStartGame = useCallback(() => {
    sendMessage({
      type: MessageType.HOST_START_GAME,
      auth: { playerId: hashCode }, // Using hashCode as playerId for host
      payload: {}
    });
  }, [sendMessage, hashCode]);

  const handleStartPuzzle = useCallback(() => {
    sendMessage({
      type: MessageType.HOST_START_PUZZLE,
      auth: { playerId: hashCode },
      payload: {}
    });
  }, [sendMessage, hashCode]);

  const handleConnect = useCallback((code) => {
    setHashCode(code);
    setHasAttemptedConnection(true); // Mark that user attempted to connect
    connectWebSocket(code);
  }, [connectWebSocket]);

  const handleDisconnect = useCallback(() => {
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
      analyticsData: null
    });
  }, []);

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
