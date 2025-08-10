import React, { useState, useEffect } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { MessageType, Colors } from '../constants';
import './HostPuzzlePhase.css';

const HostPuzzlePhase = ({ gameState, onStartPuzzle }) => {
  const [viewMode, setViewMode] = useState('grid'); // 'grid' or 'metrics'
  const [timeRemaining, setTimeRemaining] = useState(0);
  const [puzzleStarted, setPuzzleStarted] = useState(false);

  const { 
    puzzleData, 
    centralPuzzleState, 
    playerStatuses,
    teamTokens 
  } = gameState;

  // Calculate time remaining
  useEffect(() => {
    if (puzzleData?.startTimestamp && puzzleData?.totalTime) {
      const updateTimer = () => {
        const now = Date.now();
        const elapsed = (now - puzzleData.startTimestamp) / 1000;
        const remaining = Math.max(0, puzzleData.totalTime - elapsed);
        setTimeRemaining(Math.floor(remaining));
        setPuzzleStarted(true);
      };

      updateTimer();
      const interval = setInterval(updateTimer, 1000);
      return () => clearInterval(interval);
    }
  }, [puzzleData]);

  const formatTime = (seconds) => {
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    return `${mins}:${secs.toString().padStart(2, '0')}`;
  };

  const getCompletionStats = () => {
    if (!centralPuzzleState?.fragments) return { completed: 0, total: 0, percentage: 0 };
    
    const fragments = centralPuzzleState.fragments;
    const total = fragments.length;
    const completed = fragments.filter(f => 
      f.position.x === f.correctPosition.x && f.position.y === f.correctPosition.y
    ).length;
    
    return { completed, total, percentage: total > 0 ? (completed / total) * 100 : 0 };
  };

  const getPlayerActivity = () => {
    if (!playerStatuses) return [];
    
    return Object.entries(playerStatuses).map(([playerId, player]) => ({
      id: playerId,
      name: player.name || 'Unknown',
      connected: player.connected,
      hasFragment: centralPuzzleState?.fragments?.some(f => f.playerId === playerId) || false,
      movesCount: player.movesCount || 0,
      lastActivity: player.lastActivity || null
    }));
  };

  const getFragmentOwnership = () => {
    if (!centralPuzzleState?.fragments) return { owned: 0, unassigned: 0 };
    
    const fragments = centralPuzzleState.fragments;
    const owned = fragments.filter(f => f.playerId).length;
    const unassigned = fragments.filter(f => !f.playerId).length;
    
    return { owned, unassigned };
  };

  const completionStats = getCompletionStats();
  const playerActivity = getPlayerActivity();
  const fragmentOwnership = getFragmentOwnership();

  const GridView = () => (
    <motion.div
      className="puzzle-grid-view"
      initial={{ opacity: 0, scale: 0.95 }}
      animate={{ opacity: 1, scale: 1 }}
      exit={{ opacity: 0, scale: 0.95 }}
      transition={{ duration: 0.3 }}
    >
      <div className="grid-header">
        <h3>🧩 Live Puzzle Grid</h3>
        <div className="grid-stats">
          <span className="stat">
            <strong>{completionStats.completed}</strong>/{completionStats.total} correct
          </span>
          <span className="stat">
            <strong>{Math.round(completionStats.percentage)}%</strong> complete
          </span>
        </div>
      </div>

      <div className="puzzle-grid-container">
        {centralPuzzleState?.fragments ? (
          <div 
            className="puzzle-grid"
            style={{
              gridTemplateColumns: `repeat(${centralPuzzleState.gridSize}, 1fr)`,
              gridTemplateRows: `repeat(${centralPuzzleState.gridSize}, 1fr)`
            }}
          >
            {Array.from({ length: centralPuzzleState.gridSize * centralPuzzleState.gridSize }).map((_, index) => {
              const x = Math.floor(index / centralPuzzleState.gridSize);
              const y = index % centralPuzzleState.gridSize;
              const fragment = centralPuzzleState.fragments.find(f => 
                f.position.x === x && f.position.y === y
              );

              return (
                <motion.div
                  key={index}
                  className={`grid-cell ${fragment ? 'has-fragment' : 'empty'} ${
                    fragment && fragment.position.x === fragment.correctPosition.x && 
                    fragment.position.y === fragment.correctPosition.y ? 'correct' : ''
                  }`}
                  layout
                  transition={{ duration: 0.3 }}
                >
                  {fragment ? (
                    <motion.div
                      className={`fragment ${fragment.playerId ? 'owned' : 'unassigned'}`}
                      style={{
                        backgroundImage: `url(/images/puzzles/${puzzleData?.imageId}/fragment_${fragment.id}.png)`,
                        '--owner-color': fragment.playerId ? Colors.primary : Colors.text.light
                      }}
                      initial={{ scale: 0, rotate: -180 }}
                      animate={{ scale: 1, rotate: 0 }}
                      transition={{ type: "spring", stiffness: 200 }}
                    >
                      {fragment.playerId && (
                        <div className="fragment-owner">
                          {playerStatuses[fragment.playerId]?.name?.charAt(0) || 'P'}
                        </div>
                      )}
                      {fragment.position.x === fragment.correctPosition.x && 
                       fragment.position.y === fragment.correctPosition.y && (
                        <div className="correct-indicator">✓</div>
                      )}
                    </motion.div>
                  ) : (
                    <div className="cell-label">
                      {String.fromCharCode(65 + x)}{y + 1}
                    </div>
                  )}
                </motion.div>
              );
            })}
          </div>
        ) : (
          <div className="grid-placeholder">
            <div className="placeholder-icon">🧩</div>
            <p>Waiting for puzzle data...</p>
          </div>
        )}
      </div>
    </motion.div>
  );

  const MetricsView = () => (
    <motion.div
      className="puzzle-metrics-view"
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, y: -20 }}
      transition={{ duration: 0.3 }}
    >
      <div className="metrics-grid">
        {/* Completion Progress */}
        <div className="metric-card completion-card">
          <h4>🎯 Completion Progress</h4>
          <div className="completion-circle">
            <svg viewBox="0 0 120 120" className="progress-ring">
              <circle
                cx="60"
                cy="60"
                r="54"
                stroke="#E0F2FE"
                strokeWidth="8"
                fill="none"
              />
              <motion.circle
                cx="60"
                cy="60"
                r="54"
                stroke="#34D399"
                strokeWidth="8"
                fill="none"
                strokeLinecap="round"
                strokeDasharray={`${2 * Math.PI * 54}`}
                strokeDashoffset={`${2 * Math.PI * 54 * (1 - completionStats.percentage / 100)}`}
                transform="rotate(-90 60 60)"
                initial={{ strokeDashoffset: 2 * Math.PI * 54 }}
                animate={{ strokeDashoffset: 2 * Math.PI * 54 * (1 - completionStats.percentage / 100) }}
                transition={{ duration: 1 }}
              />
            </svg>
            <div className="progress-text">
              <span className="progress-percentage">{Math.round(completionStats.percentage)}%</span>
              <span className="progress-detail">{completionStats.completed}/{completionStats.total}</span>
            </div>
          </div>
        </div>

        {/* Player Activity */}
        <div className="metric-card activity-card">
          <h4>👥 Player Activity</h4>
          <div className="activity-list">
            {playerActivity.map((player, index) => (
              <motion.div
                key={player.id}
                className={`activity-item ${player.connected ? 'active' : 'inactive'}`}
                initial={{ opacity: 0, x: -20 }}
                animate={{ opacity: 1, x: 0 }}
                transition={{ delay: index * 0.1 }}
              >
                <div className="player-avatar">
                  <div className={`status-dot ${player.connected ? 'online' : 'offline'}`}></div>
                  {player.name.charAt(0)}
                </div>
                <div className="player-info">
                  <div className="player-name">{player.name}</div>
                  <div className="player-stats">
                    {player.hasFragment ? '🧩 Has fragment' : '⏳ Solving'} • 
                    {player.movesCount} moves
                  </div>
                </div>
              </motion.div>
            ))}
          </div>
        </div>

        {/* Fragment Status */}
        <div className="metric-card fragments-card">
          <h4>🔧 Fragment Status</h4>
          <div className="fragment-stats">
            <div className="stat-item">
              <div className="stat-icon owned">👤</div>
              <div className="stat-info">
                <span className="stat-number">{fragmentOwnership.owned}</span>
                <span className="stat-label">Owned Fragments</span>
              </div>
            </div>
            <div className="stat-item">
              <div className="stat-icon unassigned">❓</div>
              <div className="stat-info">
                <span className="stat-number">{fragmentOwnership.unassigned}</span>
                <span className="stat-label">Unassigned</span>
              </div>
            </div>
          </div>
        </div>

        {/* Time & Tokens */}
        <div className="metric-card time-tokens-card">
          <h4>⏱️ Time & Bonuses</h4>
          <div className="time-display">
            <div className="time-remaining">
              <span className="time-value">{formatTime(timeRemaining)}</span>
              <span className="time-label">Remaining</span>
            </div>
            <div className="token-bonuses">
              <div className="bonus-item">
                <span className="bonus-icon">⏰</span>
                <span className="bonus-text">+{Math.floor((teamTokens.chronosTokens || 0) / 5) * 20}s</span>
              </div>
              <div className="bonus-item">
                <span className="bonus-icon">🧭</span>
                <span className="bonus-text">{teamTokens.guideTokens || 0} guide</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </motion.div>
  );

  return (
    <motion.div
      className="host-puzzle-phase"
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, y: -20 }}
      transition={{ duration: 0.5 }}
    >
      <div className="puzzle-header">
        <motion.h2
          initial={{ opacity: 0, y: -20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.2 }}
        >
          🧩 Puzzle Assembly Monitor
        </motion.h2>
        <motion.p
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ delay: 0.3 }}
        >
          Monitor team collaboration and puzzle progress in real-time
        </motion.p>
      </div>

      <div className="puzzle-controls">
        <div className="view-toggle">
          <button
            className={`toggle-btn ${viewMode === 'grid' ? 'active' : ''}`}
            onClick={() => setViewMode('grid')}
          >
            <span className="btn-icon">🔍</span>
            Grid View
          </button>
          <button
            className={`toggle-btn ${viewMode === 'metrics' ? 'active' : ''}`}
            onClick={() => setViewMode('metrics')}
          >
            <span className="btn-icon">📊</span>
            Metrics
          </button>
        </div>

        <div className="time-status">
          <div className={`timer-display ${timeRemaining < 60 ? 'warning' : ''}`}>
            <span className="timer-icon">⏱️</span>
            <span className="timer-value">{formatTime(timeRemaining)}</span>
          </div>
          
          {!puzzleStarted && (
            <motion.button
              className="start-puzzle-btn host-btn-primary"
              onClick={onStartPuzzle}
              whileHover={{ scale: 1.02, y: -2 }}
              whileTap={{ scale: 0.98 }}
            >
              <span className="btn-icon">🚀</span>
              Start Puzzle Timer
            </motion.button>
          )}
        </div>
      </div>

      <div className="puzzle-content">
        <AnimatePresence mode="wait">
          {viewMode === 'grid' ? (
            <GridView key="grid" />
          ) : (
            <MetricsView key="metrics" />
          )}
        </AnimatePresence>
      </div>

      {puzzleStarted && (
        <motion.div
          className="live-indicator"
          initial={{ opacity: 0, scale: 0 }}
          animate={{ opacity: 1, scale: 1 }}
          transition={{ delay: 0.5 }}
        >
          <div className="live-dot"></div>
          <span>LIVE</span>
        </motion.div>
      )}
    </motion.div>
  );
};

export default HostPuzzlePhase;
