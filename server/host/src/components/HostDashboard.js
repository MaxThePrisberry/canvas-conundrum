import React from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { GamePhase } from '../constants';
import {
  HostHeader,
  HostSetupPhase,
  HostResourcePhase,
  HostPuzzlePhase,
  HostPostGame
} from './';
import './HostDashboard.css';

const HostDashboard = ({
  gameState,
  hashCode,
  onStartGame,
  onStartPuzzle,
  onDisconnect
}) => {
  const { phase } = gameState;

  return (
    <div className="host-dashboard">
      <HostHeader
        hashCode={hashCode}
        gameState={gameState}
        onDisconnect={onDisconnect}
      />

      <div className="dashboard-content">
        <AnimatePresence mode="wait">
          {phase === GamePhase.SETUP && (
            <HostSetupPhase
              key="setup"
              gameState={gameState}
              onStartGame={onStartGame}
            />
          )}

          {phase === GamePhase.RESOURCE_GATHERING && (
            <HostResourcePhase
              key="resource"
              gameState={gameState}
            />
          )}

          {phase === GamePhase.PUZZLE_ASSEMBLY && (
            <HostPuzzlePhase
              key="puzzle"
              gameState={gameState}
              onStartPuzzle={onStartPuzzle}
            />
          )}

          {phase === GamePhase.POST_GAME && (
            <HostPostGame
              key="postgame"
              analyticsData={gameState.analyticsData}
            />
          )}
        </AnimatePresence>
      </div>
    </div>
  );
};

export default HostDashboard;
