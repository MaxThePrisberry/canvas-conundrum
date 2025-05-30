import React from 'react';
import { motion } from 'framer-motion';
import { GamePhase } from '../constants';
import './HostHeader.css';

const HostHeader = ({ hashCode, gameState, onDisconnect }) => {
  const { phase, connectedPlayers, teamTokens } = gameState;

  const getPhaseDisplay = (currentPhase) => {
    switch (currentPhase) {
      case GamePhase.SETUP:
        return { name: 'Setup Phase', icon: '⚙️', color: '#87CEEB' };
      case GamePhase.RESOURCE_GATHERING:
        return { name: 'Resource Gathering', icon: '🎯', color: '#34D399' };
      case GamePhase.PUZZLE_ASSEMBLY:
        return { name: 'Puzzle Assembly', icon: '🧩', color: '#F59E0B' };
      case GamePhase.POST_GAME:
        return { name: 'Analytics', icon: '📊', color: '#8B5CF6' };
      default:
        return { name: 'Unknown', icon: '❓', color: '#6B7280' };
    }
  };

  const phaseInfo = getPhaseDisplay(phase);
  const totalTokens = Object.values(teamTokens).reduce((sum, tokens) => sum + tokens, 0);

  return (
    <motion.header
      className="host-header"
      initial={{ y: -100 }}
      animate={{ y: 0 }}
      transition={{ type: "spring", stiffness: 200 }}
    >
      <div className="header-content">
        <div className="header-left">
          <div className="host-badge">
            <span className="badge-icon">👑</span>
            <span className="badge-text">HOST</span>
          </div>
          
          <div className="game-info">
            <h1 className="game-title">Canvas Conundrum</h1>
            <div className="hash-display">
              <span className="hash-label">Code:</span>
              <span className="hash-value">{hashCode}</span>
            </div>
          </div>
        </div>

        <div className="header-center">
          <motion.div
            className="phase-indicator"
            style={{ backgroundColor: phaseInfo.color }}
            animate={{ scale: [1, 1.05, 1] }}
            transition={{ duration: 2, repeat: Infinity }}
          >
            <span className="phase-icon">{phaseInfo.icon}</span>
            <span className="phase-name">{phaseInfo.name}</span>
          </motion.div>
        </div>

        <div className="header-right">
          <div className="quick-stats">
            <div className="stat-item">
              <span className="stat-icon">👥</span>
              <span className="stat-value">{connectedPlayers}</span>
              <span className="stat-label">Players</span>
            </div>
            
            <div className="stat-item">
              <span className="stat-icon">🎯</span>
              <span className="stat-value">{totalTokens}</span>
              <span className="stat-label">Tokens</span>
            </div>
          </div>

          <button
            className="disconnect-button"
            onClick={onDisconnect}
            title="Disconnect and return to landing"
          >
            <span className="disconnect-icon">🚪</span>
            <span className="disconnect-text">Exit</span>
          </button>
        </div>
      </div>

      {/* Progress bar for non-setup phases */}
      {phase !== GamePhase.SETUP && phase !== GamePhase.POST_GAME && (
        <div className="phase-progress">
          <div 
            className="progress-fill"
            style={{
              width: phase === GamePhase.RESOURCE_GATHERING ? '33%' : 
                     phase === GamePhase.PUZZLE_ASSEMBLY ? '66%' : '100%',
              backgroundColor: phaseInfo.color
            }}
          />
        </div>
      )}
    </motion.header>
  );
};

export default HostHeader;
