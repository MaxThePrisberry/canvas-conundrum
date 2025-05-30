import React from 'react';
import { motion } from 'framer-motion';
import { TokenType, Colors } from '../constants';
import './HostResourcePhase.css';

const HostResourcePhase = ({ gameState }) => {
  const { 
    teamTokens, 
    playerStatuses, 
    questionsAnswered, 
    totalQuestions, 
    roundProgress 
  } = gameState;

  const tokenConfig = [
    { key: 'anchorTokens', type: TokenType.ANCHOR, label: 'Anchor', icon: '⚓', color: Colors.token.anchor },
    { key: 'chronosTokens', type: TokenType.CHRONOS, label: 'Time', icon: '⏰', color: Colors.token.chronos },
    { key: 'guideTokens', type: TokenType.GUIDE, label: 'Guide', icon: '🧭', color: Colors.token.guide },
    { key: 'clarityTokens', type: TokenType.CLARITY, label: 'Clarity', icon: '💎', color: Colors.token.clarity }
  ];

  const totalTokens = Object.values(teamTokens).reduce((sum, tokens) => sum + tokens, 0);
  const progressPercentage = totalQuestions > 0 ? (questionsAnswered / totalQuestions) * 100 : 0;

  const getPlayerLocationStats = () => {
    const locationStats = {
      anchor: 0,
      chronos: 0,
      guide: 0,
      clarity: 0,
      unknown: 0
    };

    Object.values(playerStatuses).forEach(player => {
      if (player.connected) {
        const location = player.location || 'unknown';
        if (locationStats.hasOwnProperty(location)) {
          locationStats[location]++;
        } else {
          locationStats.unknown++;
        }
      }
    });

    return locationStats;
  };

  const locationStats = getPlayerLocationStats();

  return (
    <motion.div
      className="host-resource-phase"
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, y: -20 }}
      transition={{ duration: 0.5 }}
    >
      <div className="resource-header">
        <motion.h2
          initial={{ opacity: 0, y: -20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.2 }}
        >
          🎯 Resource Gathering in Progress
        </motion.h2>
        <motion.p
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ delay: 0.3 }}
        >
          Monitor team progress as players collect tokens through trivia challenges
        </motion.p>
      </div>

      <div className="resource-content">
        <div className="resource-grid">
          {/* Progress Overview */}
          <motion.div
            className="progress-panel host-panel"
            initial={{ opacity: 0, x: -20 }}
            animate={{ opacity: 1, x: 0 }}
            transition={{ delay: 0.4 }}
          >
            <h3>📊 Progress Overview</h3>
            
            <div className="progress-stats">
              <div className="progress-circle">
                <svg viewBox="0 0 100 100" className="circular-progress">
                  <circle
                    cx="50"
                    cy="50"
                    r="45"
                    stroke="#E0F2FE"
                    strokeWidth="8"
                    fill="none"
                  />
                  <motion.circle
                    cx="50"
                    cy="50"
                    r="45"
                    stroke="#87CEEB"
                    strokeWidth="8"
                    fill="none"
                    strokeLinecap="round"
                    strokeDasharray={`${2 * Math.PI * 45}`}
                    strokeDashoffset={`${2 * Math.PI * 45 * (1 - progressPercentage / 100)}`}
                    transform="rotate(-90 50 50)"
                    initial={{ strokeDashoffset: 2 * Math.PI * 45 }}
                    animate={{ strokeDashoffset: 2 * Math.PI * 45 * (1 - progressPercentage / 100) }}
                    transition={{ duration: 1, delay: 0.6 }}
                  />
                </svg>
                <div className="progress-text">
                  <span className="progress-value">{Math.round(progressPercentage)}%</span>
                  <span className="progress-label">Complete</span>
                </div>
              </div>

              <div className="progress-details">
                <div className="detail-item">
                  <span className="detail-icon">📝</span>
                  <span className="detail-text">
                    <strong>{questionsAnswered}</strong> of <strong>{totalQuestions}</strong> questions
                  </span>
                </div>
                
                <div className="detail-item">
                  <span className="detail-icon">🎯</span>
                  <span className="detail-text">
                    <strong>{totalTokens}</strong> total tokens collected
                  </span>
                </div>
                
                {roundProgress && (
                  <div className="detail-item">
                    <span className="detail-icon">🔄</span>
                    <span className="detail-text">
                      Round <strong>{roundProgress.current}</strong> of <strong>{roundProgress.total}</strong>
                    </span>
                  </div>
                )}
              </div>
            </div>
          </motion.div>

          {/* Token Distribution */}
          <motion.div
            className="tokens-panel host-panel"
            initial={{ opacity: 0, x: 20 }}
            animate={{ opacity: 1, x: 0 }}
            transition={{ delay: 0.5 }}
          >
            <h3>🎯 Token Distribution</h3>
            
            <div className="tokens-grid">
              {tokenConfig.map((token, index) => (
                <motion.div
                  key={token.key}
                  className="token-card"
                  style={{ '--token-color': token.color }}
                  initial={{ opacity: 0, scale: 0.8 }}
                  animate={{ opacity: 1, scale: 1 }}
                  transition={{ delay: 0.6 + index * 0.1 }}
                >
                  <div className="token-icon-wrapper">
                    <span className="token-icon">{token.icon}</span>
                  </div>
                  
                  <div className="token-info">
                    <h4>{token.label}</h4>
                    <div className="token-count">{teamTokens[token.key] || 0}</div>
                  </div>
                  
                  <div className="token-bar">
                    <motion.div
                      className="token-fill"
                      initial={{ width: 0 }}
                      animate={{ width: `${Math.min(100, ((teamTokens[token.key] || 0) / 50) * 100)}%` }}
                      transition={{ duration: 1, delay: 0.8 + index * 0.1 }}
                    />
                  </div>
                </motion.div>
              ))}
            </div>
          </motion.div>

          {/* Player Locations */}
          <motion.div
            className="locations-panel host-panel"
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.6 }}
          >
            <h3>📍 Player Locations</h3>
            
            <div className="location-stats">
              {Object.entries(locationStats).map(([location, count]) => {
                const tokenInfo = tokenConfig.find(t => t.type === location) || 
                  { label: 'Unknown', icon: '❓', color: '#6B7280' };
                
                return (
                  <motion.div
                    key={location}
                    className="location-item"
                    style={{ '--location-color': tokenInfo.color }}
                    initial={{ opacity: 0, x: -20 }}
                    animate={{ opacity: 1, x: 0 }}
                    transition={{ delay: 0.8 + Object.keys(locationStats).indexOf(location) * 0.1 }}
                  >
                    <div className="location-icon">{tokenInfo.icon}</div>
                    <div className="location-label">{tokenInfo.label} Station</div>
                    <div className="location-count">{count} players</div>
                  </motion.div>
                );
              })}
            </div>
          </motion.div>

          {/* Active Players */}
          <motion.div
            className="players-panel host-panel"
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.7 }}
          >
            <h3>👥 Active Players</h3>
            
            <div className="players-list">
              {Object.entries(playerStatuses).map(([playerId, player], index) => (
                <motion.div
                  key={playerId}
                  className={`player-status-item ${player.connected ? 'connected' : 'disconnected'}`}
                  initial={{ opacity: 0, x: -20 }}
                  animate={{ opacity: 1, x: 0 }}
                  transition={{ delay: 0.9 + index * 0.05 }}
                >
                  <div className="player-avatar">
                    <div className={`status-dot ${player.connected ? 'online' : 'offline'}`}></div>
                    <span className="player-initial">
                      {(player.name || `P${index + 1}`).charAt(0).toUpperCase()}
                    </span>
                  </div>
                  
                  <div className="player-details">
                    <div className="player-name">{player.name || `Player ${index + 1}`}</div>
                    <div className="player-location">
                      {player.location ? (
                        <span style={{ color: tokenConfig.find(t => t.type === player.location)?.color }}>
                          {tokenConfig.find(t => t.type === player.location)?.icon} {' '}
                          {tokenConfig.find(t => t.type === player.location)?.label} Station
                        </span>
                      ) : (
                        <span className="no-location">Location unknown</span>
                      )}
                    </div>
                  </div>
                  
                  <div className="player-status-indicator">
                    {player.connected ? (
                      <span className="status-active">🟢 Active</span>
                    ) : (
                      <span className="status-inactive">🔴 Offline</span>
                    )}
                  </div>
                </motion.div>
              ))}
            </div>
          </motion.div>
        </div>
      </div>
    </motion.div>
  );
};

export default HostResourcePhase;
