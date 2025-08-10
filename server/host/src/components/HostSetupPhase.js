import React from 'react';
import { motion } from 'framer-motion';
import './HostSetupPhase.css';

const HostSetupPhase = ({ gameState, onStartGame }) => {
  const { connectedPlayers, readyPlayers, playerStatuses } = gameState;
  const minPlayers = 4;
  const canStartGame = connectedPlayers >= minPlayers && readyPlayers === connectedPlayers;

  const roleColors = {
    art_enthusiast: '#FDE68A', // Clarity token color
    detective: '#86EFAC',       // Guide token color
    tourist: '#93C5FD',         // Chronos token color
    janitor: '#C4B5FD'          // Anchor token color
  };

  const roleNames = {
    art_enthusiast: 'Art Enthusiast',
    detective: 'Detective',
    tourist: 'Tourist',
    janitor: 'Janitor'
  };

  const roleIcons = {
    art_enthusiast: '🎨',
    detective: '🔍',
    tourist: '📸',
    janitor: '🧹'
  };

  const getRoleCounts = () => {
    const counts = {};
    Object.values(playerStatuses).forEach(player => {
      if (player.role) {
        counts[player.role] = (counts[player.role] || 0) + 1;
      }
    });
    return counts;
  };

  const roleCounts = getRoleCounts();

  return (
    <motion.div
      className="host-setup-phase"
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, y: -20 }}
      transition={{ duration: 0.5 }}
    >
      <div className="setup-header">
        <motion.h2
          initial={{ opacity: 0, y: -20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.2 }}
        >
          Game Setup
        </motion.h2>
        <motion.p
          className="setup-description"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ delay: 0.3 }}
        >
          Monitor player connections and readiness. Start the game when ready.
        </motion.p>
      </div>

      <div className="setup-content">
        <div className="setup-grid">
          {/* Player Statistics */}
          <motion.div
            className="stats-panel"
            initial={{ opacity: 0, x: -20 }}
            animate={{ opacity: 1, x: 0 }}
            transition={{ delay: 0.4 }}
          >
            <h3>Player Statistics</h3>
            <div className="stats-grid">
              <div className="stat-card">
                <div className="stat-icon">👥</div>
                <div className="stat-number">{connectedPlayers}</div>
                <div className="stat-label">Connected</div>
              </div>
              
              <div className="stat-card">
                <div className="stat-icon">✅</div>
                <div className="stat-number">{readyPlayers}</div>
                <div className="stat-label">Ready</div>
              </div>
              
              <div className="stat-card">
                <div className="stat-icon">⏸️</div>
                <div className="stat-number">{connectedPlayers - readyPlayers}</div>
                <div className="stat-label">Waiting</div>
              </div>
              
              <div className="stat-card">
                <div className="stat-icon">🎯</div>
                <div className="stat-number">{minPlayers}</div>
                <div className="stat-label">Min Required</div>
              </div>
            </div>
          </motion.div>

          {/* Role Distribution */}
          <motion.div
            className="roles-panel"
            initial={{ opacity: 0, x: 20 }}
            animate={{ opacity: 1, x: 0 }}
            transition={{ delay: 0.5 }}
          >
            <h3>Role Distribution</h3>
            <div className="roles-grid">
              {Object.entries(roleNames).map(([roleKey, roleName]) => (
                <div 
                  key={roleKey}
                  className="role-stat"
                  style={{ '--role-color': roleColors[roleKey] }}
                >
                  <div className="role-icon">{roleIcons[roleKey]}</div>
                  <div className="role-info">
                    <div className="role-name">{roleName}</div>
                    <div className="role-count">{roleCounts[roleKey] || 0} players</div>
                  </div>
                </div>
              ))}
            </div>
          </motion.div>

          {/* Player List */}
          <motion.div
            className="players-panel"
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.6 }}
          >
            <h3>Connected Players</h3>
            <div className="players-list">
              {Object.entries(playerStatuses).length === 0 ? (
                <div className="empty-state">
                  <div className="empty-icon">👋</div>
                  <p>Waiting for players to join...</p>
                </div>
              ) : (
                Object.entries(playerStatuses).map(([playerId, player], index) => (
                  <motion.div
                    key={playerId}
                    className={`player-item ${player.ready ? 'ready' : 'waiting'}`}
                    initial={{ opacity: 0, x: -20 }}
                    animate={{ opacity: 1, x: 0 }}
                    transition={{ delay: 0.7 + index * 0.1 }}
                  >
                    <div className="player-status">
                      <div className={`status-indicator ${player.connected ? 'connected' : 'disconnected'}`}>
                        {player.connected ? '🟢' : '🔴'}
                      </div>
                    </div>
                    
                    <div className="player-info">
                      <div className="player-name">{player.name || `Player ${index + 1}`}</div>
                      <div className="player-details">
                        {player.role ? (
                          <span className="player-role" style={{ color: roleColors[player.role] }}>
                            {roleIcons[player.role]} {roleNames[player.role]}
                          </span>
                        ) : (
                          <span className="no-role">Selecting role...</span>
                        )}
                      </div>
                    </div>
                    
                    <div className="player-ready">
                      {player.ready ? (
                        <div className="ready-badge">✓ Ready</div>
                      ) : (
                        <div className="waiting-badge">⏳ Setting up</div>
                      )}
                    </div>
                  </motion.div>
                ))
              )}
            </div>
          </motion.div>

          {/* Game Controls */}
          <motion.div
            className="controls-panel"
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.8 }}
          >
            <h3>Game Controls</h3>
            
            <div className="readiness-indicator">
              {canStartGame ? (
                <div className="ready-to-start">
                  <div className="ready-icon">🚀</div>
                  <div className="ready-text">
                    <strong>Ready to Start!</strong>
                    <p>All players are connected and ready</p>
                  </div>
                </div>
              ) : (
                <div className="not-ready">
                  <div className="waiting-icon">⏳</div>
                  <div className="waiting-text">
                    <strong>Waiting for Players</strong>
                    <p>
                      {connectedPlayers < minPlayers 
                        ? `Need ${minPlayers - connectedPlayers} more players`
                        : `${connectedPlayers - readyPlayers} players still setting up`
                      }
                    </p>
                  </div>
                </div>
              )}
            </div>

            <motion.button
              className={`start-game-button ${canStartGame ? 'enabled' : 'disabled'}`}
              onClick={canStartGame ? onStartGame : undefined}
              disabled={!canStartGame}
              whileHover={canStartGame ? { scale: 1.02, y: -2 } : {}}
              whileTap={canStartGame ? { scale: 0.98 } : {}}
            >
              <span className="button-icon">🎮</span>
              <span className="button-text">Start Game</span>
              {canStartGame && (
                <div className="button-glow"></div>
              )}
            </motion.button>
          </motion.div>
        </div>
      </div>
    </motion.div>
  );
};

export default HostSetupPhase;
