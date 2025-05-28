import React, { useState, useEffect, useRef, useCallback } from 'react';
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
    puzzleData: null,
    analyticsData: null,
    playerRole: null,
    playerSpecialties: []
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
        setGameState(prev => ({
          ...prev,
          availableRoles: payload.roles,
          triviaCategories: payload.triviaCategories
        }));
        break;

      case MessageType.GAME_LOBBY_STATUS:
        // Update lobby status if needed
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
          teamTokens: payload.teamTokens
        }));
        break;

      case MessageType.PUZZLE_PHASE_LOAD:
        setPhase(GamePhase.PUZZLE_ASSEMBLY);
        setGameState(prev => ({
          ...prev,
          puzzleData: payload
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
          puzzleData: null,
          analyticsData: null,
          playerRole: null,
          playerSpecialties: []
        });
        break;

      default:
        console.log('Unhandled message type:', type);
    }
  }, [lastMessage]);

  // Callback for sending authenticated messages
  const sendAuthenticatedMessage = useCallback((type, payload) => {
    if (!playerId) return;
    
    sendMessage({
      type,
      payload: {
        auth: { playerId },
        payload
      }
    });
  }, [playerId, sendMessage]);

  // Update game state when player selects role
  const handleRoleSelection = (role) => {
    setGameState(prev => ({ ...prev, playerRole: role }));
    sendAuthenticatedMessage(MessageType.ROLE_SELECTION, { role });
  };

  // Update game state when player selects specialties
  const handleSpecialtySelection = (specialties) => {
    setGameState(prev => ({ ...prev, playerSpecialties: specialties }));
    sendAuthenticatedMessage(MessageType.TRIVIA_SPECIALTY_SELECTION, { specialties });
  };

  return (
    <div className="App">
      {/* Connection overlay for disconnections/reconnections */}
      <ConnectionOverlay 
        isConnected={isConnected} 
        isReconnecting={isReconnecting} 
      />

      {/* Token header visible during resource gathering and puzzle assembly */}
      {(phase === GamePhase.RESOURCE_GATHERING || phase === GamePhase.PUZZLE_ASSEMBLY) && (
        <TokenHeader tokens={gameState.teamTokens} />
      )}

      {/* Main game content */}
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
          />
        )}

        {phase === GamePhase.RESOURCE_GATHERING && (
          <ResourceGatheringPhase
            key="resource"
            resourceHashes={gameState.resourceHashes}
            currentQuestion={gameState.currentQuestion}
            onLocationVerified={(hash) => 
              sendAuthenticatedMessage(MessageType.RESOURCE_LOCATION_VERIFIED, { verifiedHash: hash })
            }
            onAnswerSubmit={(questionId, answer) => 
              sendAuthenticatedMessage(MessageType.TRIVIA_ANSWER, { 
                questionId, 
                answer, 
                timestamp: Date.now() 
              })
            }
          />
        )}

        {phase === GamePhase.PUZZLE_ASSEMBLY && (
          <PuzzleAssemblyPhase
            key="puzzle"
            puzzleData={gameState.puzzleData}
            playerId={playerId}
            onSegmentCompleted={(segmentId) =>
              sendAuthenticatedMessage(MessageType.SEGMENT_COMPLETED, {
                segmentId,
                completionTimestamp: Date.now()
              })
            }
            onFragmentMoveRequest={(fragmentId, newPosition) =>
              sendAuthenticatedMessage(MessageType.FRAGMENT_MOVE_REQUEST, {
                fragmentId,
                newPosition,
                timestamp: Date.now()
              })
            }
            sendMessage={sendAuthenticatedMessage}
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