import React, { useState, useEffect } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { 
  LineChart, Line, BarChart, Bar, RadarChart, Radar,
  XAxis, YAxis, CartesianGrid, Tooltip, Legend,
  ResponsiveContainer, PolarGrid, PolarAngleAxis, PolarRadiusAxis
} from 'recharts';
import { CircularProgressbar, buildStyles } from 'react-circular-progressbar';
import Confetti from 'react-confetti';
import { Colors } from '../constants';
import PhaseTransition from './PhaseTransition';
import './PostGamePhase.css';

const PostGamePhase = ({ analyticsData }) => {
  const [showTransition, setShowTransition] = useState(true);
  const [currentView, setCurrentView] = useState('overview');
  const [showConfetti, setShowConfetti] = useState(false);
  const { personalAnalytics, teamAnalytics, globalLeaderboard, gameSuccess } = analyticsData || {};

  useEffect(() => {
    const timer = setTimeout(() => {
      setShowTransition(false);
      if (gameSuccess) {
        setShowConfetti(true);
        playVictorySound();
        
        // Haptic celebration
        if (window.navigator && window.navigator.vibrate) {
          window.navigator.vibrate([100, 50, 100, 50, 200, 100, 300]);
        }
      }
    }, 2000);
    return () => clearTimeout(timer);
  }, [gameSuccess]);

  const playVictorySound = () => {
    const audioContext = new (window.AudioContext || window.webkitAudioContext)();
    const notes = [523.25, 659.25, 783.99, 1046.50]; // C5, E5, G5, C6
    
    notes.forEach((frequency, index) => {
      const oscillator = audioContext.createOscillator();
      const gainNode = audioContext.createGain();
      
      oscillator.connect(gainNode);
      gainNode.connect(audioContext.destination);
      
      oscillator.frequency.setValueAtTime(frequency, audioContext.currentTime + index * 0.15);
      gainNode.gain.setValueAtTime(0.3, audioContext.currentTime + index * 0.15);
      gainNode.gain.exponentialRampToValueAtTime(0.01, audioContext.currentTime + index * 0.15 + 0.5);
      
      oscillator.start(audioContext.currentTime + index * 0.15);
      oscillator.stop(audioContext.currentTime + index * 0.15 + 0.5);
    });
  };

  const formatTime = (seconds) => {
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    return `${mins}:${secs.toString().padStart(2, '0')}`;
  };

  const getPersonalData = () => {
    if (!personalAnalytics || personalAnalytics.length === 0) return null;
    return personalAnalytics[0]; // Assuming first entry is current player
  };

  const renderOverview = () => {
    const personalData = getPersonalData();
    if (!personalData || !teamAnalytics) return null;

    return (
      <motion.div
        className="overview-container"
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ delay: 0.5 }}
      >
        <div className="result-header">
          <motion.div
            className="result-icon"
            initial={{ scale: 0 }}
            animate={{ scale: 1 }}
            transition={{ type: "spring", stiffness: 200 }}
          >
            {gameSuccess ? '🏆' : '💪'}
          </motion.div>
          <h1>{gameSuccess ? 'Victory!' : 'Game Complete'}</h1>
          <p className="result-subtitle">
            {gameSuccess ? 'Masterpiece restored!' : 'Better luck next time!'}
          </p>
        </div>

        <div className="stats-grid">
          <motion.div
            className="stat-card"
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.6 }}
          >
            <h3>Team Performance</h3>
            <div className="stat-value">{formatTime(teamAnalytics.overallPerformance.totalTime)}</div>
            <p className="stat-label">Total Time</p>
            <div className="progress-container">
              <CircularProgressbar
                value={teamAnalytics.overallPerformance.completionRate * 100}
                text={`${Math.round(teamAnalytics.overallPerformance.completionRate * 100)}%`}
                styles={buildStyles({
                  pathColor: Colors.primary,
                  textColor: Colors.text.primary,
                  trailColor: Colors.surface
                })}
              />
            </div>
          </motion.div>

          <motion.div
            className="stat-card"
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.7 }}
          >
            <h3>Your Performance</h3>
            <div className="stat-value">
              {personalData.triviaPerformance.correctAnswers}/{personalData.triviaPerformance.totalQuestions}
            </div>
            <p className="stat-label">Trivia Accuracy</p>
            <div className="token-stats">
              {Object.entries(personalData.tokenCollection).map(([token, count]) => (
                <div key={token} className="token-stat">
                  <span className="token-icon" style={{ color: Colors.token[token] }}>
                    {token === 'anchor' ? '⚓' : token === 'chronos' ? '⏰' : token === 'guide' ? '🧭' : '💎'}
                  </span>
                  <span className="token-count">{count}</span>
                </div>
              ))}
            </div>
          </motion.div>

          <motion.div
            className="stat-card"
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.8 }}
          >
            <h3>Team Score</h3>
            <div className="stat-value">{teamAnalytics.overallPerformance.totalScore}</div>
            <p className="stat-label">Points</p>
            <button 
              className="btn-secondary view-details"
              onClick={() => setCurrentView('detailed')}
            >
              View Details
            </button>
          </motion.div>
        </div>

        <motion.div
          className="leaderboard-preview"
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.9 }}
        >
          <h3>Top Players</h3>
          <div className="leaderboard-list">
            {globalLeaderboard.slice(0, 3).map((player, index) => (
              <motion.div
                key={player.playerId}
                className="leaderboard-item"
                initial={{ opacity: 0, x: -20 }}
                animate={{ opacity: 1, x: 0 }}
                transition={{ delay: 1 + index * 0.1 }}
              >
                <span className="rank">#{player.rank}</span>
                <span className="name">{player.playerName}</span>
                <span className="score">{player.totalScore}</span>
              </motion.div>
            ))}
          </div>
        </motion.div>
      </motion.div>
    );
  };

  const renderDetailedStats = () => {
    const personalData = getPersonalData();
    if (!personalData || !teamAnalytics) return null;

    // Prepare data for charts
    const categoryData = Object.entries(personalData.triviaPerformance.accuracyByCategory).map(
      ([category, accuracy]) => ({
        category: category.replace('_', ' '),
        accuracy: Math.round(accuracy * 100)
      })
    );

    const tokenData = Object.entries(teamAnalytics.resourceEfficiency.tokenDistribution).map(
      ([token, count]) => ({
        token: token.charAt(0).toUpperCase() + token.slice(1),
        count: Math.round(count)
      })
    );

    return (
      <motion.div
        className="detailed-stats"
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
      >
        <div className="stats-header">
          <button 
            className="back-button"
            onClick={() => setCurrentView('overview')}
          >
            ← Back
          </button>
          <h2>Detailed Analytics</h2>
        </div>

        <div className="charts-grid">
          <motion.div
            className="chart-card"
            initial={{ opacity: 0, scale: 0.9 }}
            animate={{ opacity: 1, scale: 1 }}
            transition={{ delay: 0.2 }}
          >
            <h3>Trivia Performance by Category</h3>
            <ResponsiveContainer width="100%" height={200}>
              <RadarChart data={categoryData}>
                <PolarGrid stroke="#E0F2FE" />
                <PolarAngleAxis dataKey="category" tick={{ fontSize: 12 }} />
                <PolarRadiusAxis angle={90} domain={[0, 100]} />
                <Radar
                  name="Accuracy"
                  dataKey="accuracy"
                  stroke={Colors.primary}
                  fill={Colors.primary}
                  fillOpacity={0.6}
                />
                <Tooltip />
              </RadarChart>
            </ResponsiveContainer>
          </motion.div>

          <motion.div
            className="chart-card"
            initial={{ opacity: 0, scale: 0.9 }}
            animate={{ opacity: 1, scale: 1 }}
            transition={{ delay: 0.3 }}
          >
            <h3>Token Collection</h3>
            <ResponsiveContainer width="100%" height={200}>
              <BarChart data={tokenData}>
                <CartesianGrid strokeDasharray="3 3" stroke="#E0F2FE" />
                <XAxis dataKey="token" />
                <YAxis />
                <Tooltip />
                <Bar dataKey="count" fill={Colors.primary} radius={[8, 8, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </motion.div>

          <motion.div
            className="chart-card"
            initial={{ opacity: 0, scale: 0.9 }}
            animate={{ opacity: 1, scale: 1 }}
            transition={{ delay: 0.4 }}
          >
            <h3>Collaboration Metrics</h3>
            <div className="collaboration-stats">
              <div className="collab-item">
                <div className="collab-label">Communication</div>
                <div className="collab-bar">
                  <motion.div
                    className="collab-fill"
                    initial={{ width: 0 }}
                    animate={{ width: `${teamAnalytics.collaborationScores.communicationScore * 100}%` }}
                    transition={{ delay: 0.6, duration: 1 }}
                    style={{ backgroundColor: Colors.token.guide }}
                  />
                </div>
              </div>
              <div className="collab-item">
                <div className="collab-label">Coordination</div>
                <div className="collab-bar">
                  <motion.div
                    className="collab-fill"
                    initial={{ width: 0 }}
                    animate={{ width: `${teamAnalytics.collaborationScores.coordinationScore * 100}%` }}
                    transition={{ delay: 0.8, duration: 1 }}
                    style={{ backgroundColor: Colors.token.clarity }}
                  />
                </div>
              </div>
              <div className="collab-item">
                <div className="collab-label">Response Time</div>
                <div className="collab-bar">
                  <motion.div
                    className="collab-fill"
                    initial={{ width: 0 }}
                    animate={{ width: `${(15 - teamAnalytics.collaborationScores.averageResponseTime) / 15 * 100}%` }}
                    transition={{ delay: 1, duration: 1 }}
                    style={{ backgroundColor: Colors.token.chronos }}
                  />
                </div>
              </div>
            </div>
          </motion.div>
        </div>

        <motion.div
          className="full-leaderboard"
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.5 }}
        >
          <h3>Complete Leaderboard</h3>
          <div className="leaderboard-table">
            {globalLeaderboard.map((player, index) => (
              <motion.div
                key={player.playerId}
                className={`leaderboard-row ${player.playerId === personalData.playerId ? 'current-player' : ''}`}
                initial={{ opacity: 0, x: -20 }}
                animate={{ opacity: 1, x: 0 }}
                transition={{ delay: 0.6 + index * 0.05 }}
              >
                <span className="rank">#{player.rank}</span>
                <span className="name">{player.playerName}</span>
                <span className="score">{player.totalScore} pts</span>
              </motion.div>
            ))}
          </div>
        </motion.div>
      </motion.div>
    );
  };

  return (
    <div className="post-game-phase">
      {showConfetti && (
        <Confetti
          width={window.innerWidth}
          height={window.innerHeight}
          numberOfPieces={200}
          gravity={0.1}
          colors={[Colors.primary, Colors.secondary, Colors.tertiary, Colors.accent]}
          recycle={false}
        />
      )}

      <AnimatePresence>
        {showTransition && (
          <PhaseTransition 
            title={gameSuccess ? "Victory!" : "Game Complete"}
            subtitle="Let's see how you did"
            celebration={gameSuccess}
          />
        )}
      </AnimatePresence>

      {!showTransition && (
        <>
          {currentView === 'overview' && renderOverview()}
          {currentView === 'detailed' && renderDetailedStats()}
        </>
      )}
    </div>
  );
};

export default PostGamePhase;