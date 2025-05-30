import React, { useState, useEffect, useCallback } from 'react';
import './App.css';
import { AnimatePresence } from 'framer-motion';
import {
  SetupPhase,
  ResourceGatheringPhase,
  PuzzleAssemblyPhase,
  PostGamePhase,
  ConnectionOverlay,
  TokenHeader
} from './components';
import { useWebSocket } from './hooks/useWebSocket';
import { GamePhase, MessageType } from './constants';

function App() {
  const [phase, setPhase] = useState(GamePhase.SETUP);
  const [playerId, setPlayerId] = useState(null);
  const [gameState, setGameState] = useState({
    availableRoles: [],
    triviaCategories: [],
    teamTokens: {
      anchorTokens: 0,
      chronosTokens: 0,
      guideTokens: 0,
      clarityTokens: 0
    },
    resourceHashes: {},
    currentQuestion: null,
    imagePreview: null,
    puzzleData: null,
    analyticsData: null,
    playerRole: null,
    playerSpecialties: [],
    puzzleTimer: null,
    individualPuzzleComplete: false,
    centralPuzzleState: null,
    personalPuzzleState: null
  });

  const { 
    isConnected, 
    isReconnecting, 
    sendMessage, 
    lastMessage 
  } = useWebSocket();

  // Handle incoming WebSocket messages
  useEffect(() => {
    if (!lastMessage) return;

    const { type, payload } = lastMessage;

    switch (type) {
      case MessageType.AVAILABLE_ROLES:
        setPlayerId(payload.playerId);
        // Only set roles if not a host connection
        if (!payload.isHost) {
          setGameState(prev => ({
            ...prev,
            availableRoles: payload.roles,
            triviaCategories: payload.triviaCategories
          }));
        }
        break;

      case MessageType.GAME_LOBBY_STATUS:
        // Update lobby status
        setGameState(prev => ({
          ...prev,
          lobbyStatus: payload
        }));
        break;

      case MessageType.RESOURCE_PHASE_START:
        setPhase(GamePhase.RESOURCE_GATHERING);
        setGameState(prev => ({
          ...prev,
          resourceHashes: payload.resourceHashes
        }));
        break;

      case MessageType.TRIVIA_QUESTION:
        setGameState(prev => ({
          ...prev,
          currentQuestion: payload
        }));
        break;

      case MessageType.TEAM_PROGRESS_UPDATE:
        setGameState(prev => ({
          ...prev,
          teamTokens: payload.teamTokens,
          questionsAnswered: payload.questionsAnswered,
          totalQuestions: payload.totalQuestions
        }));
        break;

      case MessageType.IMAGE_PREVIEW:
        setGameState(prev => ({
          ...prev,
          imagePreview: payload
        }));
        break;

      case MessageType.PUZZLE_PHASE_LOAD:
        if (!payload.isHost) {
          setPhase(GamePhase.PUZZLE_ASSEMBLY);
          setGameState(prev => ({
            ...prev,
            puzzleData: payload,
            individualPuzzleComplete: false
          }));
        }
        break;

      case MessageType.PUZZLE_PHASE_START:
        setGameState(prev => ({
          ...prev,
          puzzleTimer: {
            startTime: payload.startTimestamp,
            totalTime: payload.totalTime
          }
        }));
        break;

      case MessageType.SEGMENT_COMPLETION_ACK:
        setGameState(prev => ({
          ...prev,
          individualPuzzleComplete: true,
          gridPosition: payload.gridPosition
        }));
        break;

      case MessageType.PERSONAL_PUZZLE_STATE:
        setGameState(prev => ({
          ...prev,
          personalPuzzleState: payload.personalView
        }));
        break;

      case MessageType.CENTRAL_PUZZLE_STATE:
        setGameState(prev => ({
          ...prev,
          centralPuzzleState: payload
        }));
        break;

      case MessageType.FRAGMENT_MOVE_RESPONSE:
        // Handle move response if needed
        break;

      case MessageType.PIECE_RECOMMENDATION:
        // Handle incoming recommendation
        setGameState(prev => ({
          ...prev,
          incomingRecommendation: payload
        }));
        break;

      case MessageType.GAME_ANALYTICS:
        setPhase(GamePhase.POST_GAME);
        setGameState(prev => ({
          ...prev,
          analyticsData: payload
        }));
        break;

      case MessageType.GAME_RESET:
        // Reset to initial state
        setPhase(GamePhase.SETUP);
        setPlayerId(null);
        setGameState({
          availableRoles: [],
          triviaCategories: [],
          teamTokens: {
            anchorTokens: 0,
            chronosTokens: 0,
            guideTokens: 0,
            clarityTokens: 0
          },
          resourceHashes: {},
          currentQuestion: null,
          imagePreview: null,
          puzzleData: null,
          analyticsData: null,
          playerRole: null,
          playerSpecialties: [],
          puzzleTimer: null,
          individualPuzzleComplete: false,
          centralPuzzleState: null,
          personalPuzzleState: null
        });
        break;

      case MessageType.ERROR:
        console.error('Game error:', payload);
        // Could show error toast here
        break;

      default:
        console.log('Unhandled message type:', type);
    }
  }, [lastMessage]);

  // Callback for sending authenticated messages
  const sendAuthenticatedMessage = useCallback((type, payload) => {
    if (!playerId) {
      console.warn('Cannot send message: playerId not set');
      return;
    }
    
    sendMessage({
      type,
      auth: { playerId },
      payload
    });
  }, [playerId, sendMessage]);

  // Event handlers
  const handleRoleSelection = useCallback((role) => {
    setGameState(prev => ({ ...prev, playerRole: role }));
    sendAuthenticatedMessage(MessageType.ROLE_SELECTION, { role });
  }, [sendAuthenticatedMessage]);

  const handleSpecialtySelection = useCallback((specialties) => {
    setGameState(prev => ({ ...prev, playerSpecialties: specialties }));
    sendAuthenticatedMessage(MessageType.TRIVIA_SPECIALTY_SELECTION, { specialties });
  }, [sendAuthenticatedMessage]);

  const handleLocationVerified = useCallback((hash) => {
    sendAuthenticatedMessage(MessageType.RESOURCE_LOCATION_VERIFIED, { verifiedHash: hash });
  }, [sendAuthenticatedMessage]);

  const handleAnswerSubmit = useCallback((questionId, answer) => {
    sendAuthenticatedMessage(MessageType.TRIVIA_ANSWER, { 
      questionId, 
      answer, 
      timestamp: Date.now() 
    });
  }, [sendAuthenticatedMessage]);

  const handleSegmentCompleted = useCallback((segmentId) => {
    sendAuthenticatedMessage(MessageType.SEGMENT_COMPLETED, {
      segmentId,
      completionTimestamp: Date.now()
    });
  }, [sendAuthenticatedMessage]);

  const handleFragmentMoveRequest = useCallback((fragmentId, newPosition) => {
    sendAuthenticatedMessage(MessageType.FRAGMENT_MOVE_REQUEST, {
      fragmentId,
      newPosition,
      timestamp: Date.now()
    });
  }, [sendAuthenticatedMessage]);

  const handleRecommendationRequest = useCallback((data) => {
    sendAuthenticatedMessage(MessageType.PIECE_RECOMMENDATION_REQUEST, data);
  }, [sendAuthenticatedMessage]);

  const handleRecommendationResponse = useCallback((recommendationId, accepted) => {
    sendAuthenticatedMessage(MessageType.PIECE_RECOMMENDATION_RESPONSE, {
      recommendationId,
      accepted
    });
  }, [sendAuthenticatedMessage]);

  return (
    <div className="App">
      <ConnectionOverlay 
        isConnected={isConnected} 
        isReconnecting={isReconnecting} 
      />

      {(phase === GamePhase.RESOURCE_GATHERING || phase === GamePhase.PUZZLE_ASSEMBLY) && (
        <TokenHeader tokens={gameState.teamTokens} />
      )}

      <AnimatePresence mode="wait">
        {phase === GamePhase.SETUP && (
          <SetupPhase
            key="setup"
            availableRoles={gameState.availableRoles}
            triviaCategories={gameState.triviaCategories}
            onRoleSelect={handleRoleSelection}
            onSpecialtySelect={handleSpecialtySelection}
            playerRole={gameState.playerRole}
            playerSpecialties={gameState.playerSpecialties}
            lobbyStatus={gameState.lobbyStatus}
          />
        )}

        {phase === GamePhase.RESOURCE_GATHERING && (
          <ResourceGatheringPhase
            key="resource"
            resourceHashes={gameState.resourceHashes}
            currentQuestion={gameState.currentQuestion}
            onLocationVerified={handleLocationVerified}
            onAnswerSubmit={handleAnswerSubmit}
            teamTokens={gameState.teamTokens}
            questionsAnswered={gameState.questionsAnswered}
            totalQuestions={gameState.totalQuestions}
          />
        )}

        {phase === GamePhase.PUZZLE_ASSEMBLY && (
          <PuzzleAssemblyPhase
            key="puzzle"
            puzzleData={gameState.puzzleData}
            imagePreview={gameState.imagePreview}
            puzzleTimer={gameState.puzzleTimer}
            playerId={playerId}
            individualPuzzleComplete={gameState.individualPuzzleComplete}
            centralPuzzleState={gameState.centralPuzzleState}
            personalPuzzleState={gameState.personalPuzzleState}
            incomingRecommendation={gameState.incomingRecommendation}
            onSegmentCompleted={handleSegmentCompleted}
            onFragmentMoveRequest={handleFragmentMoveRequest}
            onRecommendationRequest={handleRecommendationRequest}
            onRecommendationResponse={handleRecommendationResponse}
          />
        )}

        {phase === GamePhase.POST_GAME && (
          <PostGamePhase
            key="postgame"
            analyticsData={gameState.analyticsData}
          />
        )}
      </AnimatePresence>
    </div>
  );
}

export default App;
