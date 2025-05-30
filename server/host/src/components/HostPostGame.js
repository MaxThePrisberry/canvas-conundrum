import React, { useState, useEffect } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
  PieChart,
  Pie,
  Cell,
  LineChart,
  Line,
  RadarChart,
  PolarGrid,
  PolarAngleAxis,
  PolarRadiusAxis,
  Radar
} from 'recharts';
import { Colors } from '../constants';
import './HostPostGame.css';

const HostPostGame = ({ analyticsData }) => {
  const [viewMode, setViewMode] = useState('overview'); // 'overview', 'team', 'individual'
  const [selectedPlayer, setSelectedPlayer] = useState(null);
  const [showCelebration, setShowCelebration] = useState(false);

  const { personalAnalytics, teamAnalytics, globalLeaderboard, gameSuccess } = analyticsData || {};

  // Show celebration effect on mount if game was successful
  useEffect(() => {
    if (gameSuccess) {
      setShowCelebration(true);
      const timer = setTimeout(() => setShowCelebration(false), 3000);
      return () => clearTimeout(timer);
    }
  }, [gameSuccess]);

  // Prepare data for charts
  const prepareTokenDistributionData = () => {
    if (!teamAnalytics?.resourceEfficiency?.tokenDistribution) return [];
    
    const tokens = teamAnalytics.resourceEfficiency.tokenDistribution;
    return [
      { name: 'Anchor', value: tokens.anchor, color: Colors.token.anchor, icon: '⚓' },
      { name: 'Chronos', value: tokens.chronos, color: Colors.token.chronos, icon: '⏰' },
      { name: 'Guide', value: tokens.guide, color: Colors.token.guide, icon: '🧭' },
      { name: 'Clarity', value: tokens.clarity, color: Colors.token.clarity, icon: '💎' }
    ];
  };

  const preparePlayerPerformanceData = () => {
    if (!personalAnalytics) return [];
    
    return personalAnalytics.map((player, index) => ({
      name: player.playerName || `Player ${index + 1}`,
      tokens: Object.values(player.tokenCollection || {}).reduce((sum, val) => sum + val, 0),
      triviaScore: ((player.triviaPerformance?.correctAnswers || 0) / (player.triviaPerformance?.totalQuestions || 1)) * 100,
      puzzleTime: player.puzzleSolvingMetrics?.fragmentSolveTime || 0,
      moves: player.puzzleSolvingMetrics?.movesContributed || 0
    }));
  };

  const prepareAccuracyByCategoryData = () => {
    if (!personalAnalytics?.[0]?.triviaPerformance?.accuracyByCategory) return [];
    
    // Aggregate accuracy across all players
    const categoryTotals = {};
    personalAnalytics.forEach(player => {
      const accuracies = player.triviaPerformance?.accuracyByCategory || {};
      Object.entries(accuracies).forEach(([category, accuracy]) => {
        if (!categoryTotals[category]) {
          categoryTotals[category] = { total: 0, count: 0 };
        }
        categoryTotals[category].total += accuracy;
        categoryTotals[category].count += 1;
      });
    });
    
    return Object.entries(categoryTotals).map(([category, data]) => ({
      category: category.charAt(0).toUpperCase() + category.slice(1),
      accuracy: Math.round((data.total / data.count) * 100)
    }));
  };

  const prepareTeamRadarData = () => {
    if (!teamAnalytics) return [];
    
    return [
      {
        subject: 'Speed',
        value: Math.max(0, 100 - ((teamAnalytics.overallPerformance?.totalTime || 1200) / 1200) * 100),
        fullMark: 100
      },
      {
        subject: 'Collaboration',
        value: (teamAnalytics.collaborationScores?.collaborationScore || 0) * 100,
        fullMark: 100
      },
      {
        subject: 'Communication',
        value: (teamAnalytics.collaborationScores?.communicationScore || 0) * 100,
        fullMark: 100
      },
      {
        subject: 'Coordination',
        value: (teamAnalytics.collaborationScores?.coordinationScore || 0) * 100,
        fullMark: 100
      },
      {
        subject: 'Efficiency',
        value: Math.min(100, (teamAnalytics.resourceEfficiency?.tokensPerRound || 0) * 4),
        fullMark: 100
      }
    ];
  };

  const CustomTooltip = ({ active, payload, label }) => {
    if (active && payload && payload.length) {
      return (
        <div className="custom-tooltip">
          <p className="tooltip-label">{label}</p>
          {payload.map((pld, index) => (
            <p key={index} className="tooltip-value" style={{ color: pld.color }}>
              {`${pld.dataKey}: ${pld.value}${pld.dataKey === 'triviaScore' ? '%' : ''}`}
            </p>
          ))}
        </div>
      );
    }
    return null;
  };

  const CustomPieLabel = ({ cx, cy, midAngle, innerRadius, outerRadius, value, name, icon }) => {
    const RADIAN = Math.PI / 180;
    const radius = innerRadius + (outerRadius - innerRadius) * 0.5;
    const x = cx + radius * Math.cos(-midAngle * RADIAN);
    const y = cy + radius * Math.sin(-midAngle * RADIAN);

    return (
      <text
        x={x}
        y={y}
        fill="white"
        textAnchor={x > cx ? 'start' : 'end'}
        dominantBaseline="central"
        fontSize="12"
        fontWeight="600"
      >
        {`${icon} ${value}`}
      </text>
    );
  };

  const OverviewView = () => (
    <motion.div
      className="analytics-view overview-view"
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, y: -20 }}
      transition={{ duration: 0.5 }}
    >
      <div className="analytics-grid">
        {/* Game Summary */}
        <div className="summary-card">
          <h3>🎯 Game Summary</h3>
          <div className="summary-stats">
            <div className="summary-item">
              <div className={`status-badge ${gameSuccess ? 'success' : 'failure'}`}>
                {gameSuccess ? '🏆 Success' : '❌ Incomplete'}
              </div>
            </div>
            <div className="summary-item">
              <span className="stat-label">Total Time</span>
              <span className="stat-value">
                {Math.floor((teamAnalytics?.overallPerformance?.totalTime || 0) / 60)}:
                {String((teamAnalytics?.overallPerformance?.totalTime || 0) % 60).padStart(2, '0')}
              </span>
            </div>
            <div className="summary-item">
              <span className="stat-label">Team Score</span>
              <span className="stat-value">{teamAnalytics?.overallPerformance?.totalScore || 0}</span>
            </div>
            <div className="summary-item">
              <span className="stat-label">Completion Rate</span>
              <span className="stat-value">
                {Math.round((teamAnalytics?.overallPerformance?.completionRate || 0) * 100)}%
              </span>
            </div>
          </div>
        </div>

        {/* Token Distribution Pie Chart */}
        <div className="chart-card">
          <h3>🎯 Token Distribution</h3>
          <ResponsiveContainer width="100%" height={300}>
            <PieChart>
              <Pie
                data={prepareTokenDistributionData()}
                cx="50%"
                cy="50%"
                labelLine={false}
                label={CustomPieLabel}
                outerRadius={80}
                fill="#8884d8"
                dataKey="value"
                animationBegin={0}
                animationDuration={1000}
              >
                {prepareTokenDistributionData().map((entry, index) => (
                  <Cell key={`cell-${index}`} fill={entry.color} />
                ))}
              </Pie>
              <Tooltip />
            </PieChart>
          </ResponsiveContainer>
        </div>

        {/* Team Performance Radar */}
        <div className="chart-card">
          <h3>📊 Team Performance</h3>
          <ResponsiveContainer width="100%" height={300}>
            <RadarChart data={prepareTeamRadarData()}>
              <PolarGrid />
              <PolarAngleAxis dataKey="subject" tick={{ fontSize: 12 }} />
              <PolarRadiusAxis
                angle={0}
                domain={[0, 100]}
                tickCount={6}
                tick={{ fontSize: 10 }}
              />
              <Radar
                name="Performance"
                dataKey="value"
                stroke={Colors.primary}
                fill={Colors.primary}
                fillOpacity={0.3}
                strokeWidth={2}
                animationBegin={200}
                animationDuration={1000}
              />
            </RadarChart>
          </ResponsiveContainer>
        </div>

        {/* Player Rankings */}
        <div className="leaderboard-card">
          <h3>🏆 Player Rankings</h3>
          <div className="leaderboard-list">
            {globalLeaderboard?.slice(0, 8).map((player, index) => (
              <motion.div
                key={player.playerId}
                className={`leaderboard-item rank-${index + 1}`}
                initial={{ opacity: 0, x: -20 }}
                animate={{ opacity: 1, x: 0 }}
                transition={{ delay: index * 0.1 }}
              >
                <div className="rank-badge">#{player.rank}</div>
                <div className="player-info">
                  <span className="player-name">{player.playerName}</span>
                  <span className="player-score">{player.totalScore} pts</span>
                </div>
                {index < 3 && (
                  <div className="trophy-icon">
                    {index === 0 ? '🥇' : index === 1 ? '🥈' : '🥉'}
                  </div>
                )}
              </motion.div>
            ))}
          </div>
        </div>
      </div>
    </motion.div>
  );

  const TeamView = () => (
    <motion.div
      className="analytics-view team-view"
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, y: -20 }}
      transition={{ duration: 0.5 }}
    >
      <div className="analytics-grid">
        {/* Player Performance Bar Chart */}
        <div className="chart-card large">
          <h3>👥 Player Performance Comparison</h3>
          <ResponsiveContainer width="100%" height={400}>
            <BarChart data={preparePlayerPerformanceData()}>
              <CartesianGrid strokeDasharray="3 3" stroke="#E0F2FE" />
              <XAxis 
                dataKey="name" 
                tick={{ fontSize: 12 }}
                angle={-45}
                textAnchor="end"
                height={80}
              />
              <YAxis tick={{ fontSize: 12 }} />
              <Tooltip content={<CustomTooltip />} />
              <Legend />
              <Bar 
                dataKey="tokens" 
                fill={Colors.token.clarity} 
                name="Total Tokens"
                animationBegin={0}
                animationDuration={1000}
              />
              <Bar 
                dataKey="triviaScore" 
                fill={Colors.primary} 
                name="Trivia Score (%)"
                animationBegin={200}
                animationDuration={1000}
              />
              <Bar 
                dataKey="moves" 
                fill={Colors.token.guide} 
                name="Puzzle Moves"
                animationBegin={400}
                animationDuration={1000}
              />
            </BarChart>
          </ResponsiveContainer>
        </div>

        {/* Trivia Category Accuracy */}
        <div className="chart-card">
          <h3>📚 Category Performance</h3>
          <ResponsiveContainer width="100%" height={300}>
            <BarChart data={prepareAccuracyByCategoryData()} layout="horizontal">
              <CartesianGrid strokeDasharray="3 3" stroke="#E0F2FE" />
              <XAxis type="number" domain={[0, 100]} tick={{ fontSize: 12 }} />
              <YAxis dataKey="category" type="category" tick={{ fontSize: 12 }} width={80} />
              <Tooltip />
              <Bar 
                dataKey="accuracy" 
                fill={Colors.token.anchor}
                animationBegin={600}
                animationDuration={1000}
              />
            </BarChart>
          </ResponsiveContainer>
        </div>

        {/* Collaboration Metrics */}
        <div className="metrics-card">
          <h3>🤝 Collaboration Metrics</h3>
          <div className="collaboration-stats">
            <div className="collab-stat">
              <div className="stat-icon">💬</div>
              <div className="stat-content">
                <span className="stat-number">
                  {Math.round((teamAnalytics?.collaborationScores?.communicationScore || 0) * 100)}%
                </span>
                <span className="stat-label">Communication</span>
              </div>
            </div>
            <div className="collab-stat">
              <div className="stat-icon">🎯</div>
              <div className="stat-content">
                <span className="stat-number">
                  {Math.round((teamAnalytics?.collaborationScores?.coordinationScore || 0) * 100)}%
                </span>
                <span className="stat-label">Coordination</span>
              </div>
            </div>
            <div className="collab-stat">
              <div className="stat-icon">⚡</div>
              <div className="stat-content">
                <span className="stat-number">
                  {teamAnalytics?.collaborationScores?.averageResponseTime?.toFixed(1)}s
                </span>
                <span className="stat-label">Avg Response</span>
              </div>
            </div>
            <div className="collab-stat">
              <div className="stat-icon">🔄</div>
              <div className="stat-content">
                <span className="stat-number">
                  {teamAnalytics?.collaborationScores?.acceptedRecommendations || 0}/
                  {teamAnalytics?.collaborationScores?.totalRecommendations || 0}
                </span>
                <span className="stat-label">Recommendations</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </motion.div>
  );

  const IndividualView = () => (
    <motion.div
      className="analytics-view individual-view"
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, y: -20 }}
      transition={{ duration: 0.5 }}
    >
      <div className="player-selector">
        <h3>👤 Select Player for Detailed Analytics</h3>
        <div className="player-buttons">
          {personalAnalytics?.map((player, index) => (
            <button
              key={player.playerId}
              className={`player-btn ${selectedPlayer === index ? 'active' : ''}`}
              onClick={() => setSelectedPlayer(index)}
            >
              <span className="player-initial">
                {(player.playerName || `P${index + 1}`).charAt(0)}
              </span>
              <span className="player-name">{player.playerName || `Player ${index + 1}`}</span>
            </button>
          ))}
        </div>
      </div>

      {selectedPlayer !== null && personalAnalytics?.[selectedPlayer] && (
        <motion.div
          className="player-details"
          initial={{ opacity: 0, scale: 0.95 }}
          animate={{ opacity: 1, scale: 1 }}
          transition={{ duration: 0.3 }}
        >
          {/* Player Header */}
          <div className="player-header">
            <div className="player-avatar-large">
              {(personalAnalytics[selectedPlayer].playerName || `P${selectedPlayer + 1}`).charAt(0)}
            </div>
            <div className="player-title">
              <h2>{personalAnalytics[selectedPlayer].playerName || `Player ${selectedPlayer + 1}`}</h2>
              <p>Detailed Performance Analysis</p>
            </div>
          </div>

          {/* Individual Metrics Grid */}
          <div className="individual-grid">
            {/* Token Collection */}
            <div className="metric-card">
              <h4>🎯 Token Collection</h4>
              <div className="token-breakdown">
                {Object.entries(personalAnalytics[selectedPlayer].tokenCollection || {}).map(([tokenType, count]) => (
                  <div key={tokenType} className="token-item">
                    <div className="token-icon-small" style={{ background: Colors.token[tokenType] }}>
                      {tokenType === 'anchor' ? '⚓' : tokenType === 'chronos' ? '⏰' : 
                       tokenType === 'guide' ? '🧭' : '💎'}
                    </div>
                    <span className="token-count-small">{count}</span>
                    <span className="token-name">{tokenType}</span>
                  </div>
                ))}
              </div>
            </div>

            {/* Trivia Performance */}
            <div className="metric-card">
              <h4>🧠 Trivia Performance</h4>
              <div className="trivia-stats">
                <div className="trivia-item">
                  <span className="trivia-label">Accuracy</span>
                  <span className="trivia-value">
                    {Math.round(((personalAnalytics[selectedPlayer].triviaPerformance?.correctAnswers || 0) / 
                    (personalAnalytics[selectedPlayer].triviaPerformance?.totalQuestions || 1)) * 100)}%
                  </span>
                </div>
                <div className="trivia-item">
                  <span className="trivia-label">Specialty Bonus</span>
                  <span className="trivia-value">
                    {personalAnalytics[selectedPlayer].triviaPerformance?.specialtyBonus || 0}
                  </span>
                </div>
                <div className="trivia-item">
                  <span className="trivia-label">Questions Answered</span>
                  <span className="trivia-value">
                    {personalAnalytics[selectedPlayer].triviaPerformance?.correctAnswers || 0}/
                    {personalAnalytics[selectedPlayer].triviaPerformance?.totalQuestions || 0}
                  </span>
                </div>
              </div>
            </div>

            {/* Puzzle Metrics */}
            <div className="metric-card">
              <h4>🧩 Puzzle Performance</h4>
              <div className="puzzle-stats">
                <div className="puzzle-item">
                  <span className="puzzle-label">Solve Time</span>
                  <span className="puzzle-value">
                    {Math.floor((personalAnalytics[selectedPlayer].puzzleSolvingMetrics?.fragmentSolveTime || 0) / 60)}:
                    {String((personalAnalytics[selectedPlayer].puzzleSolvingMetrics?.fragmentSolveTime || 0) % 60).padStart(2, '0')}
                  </span>
                </div>
                <div className="puzzle-item">
                  <span className="puzzle-label">Moves Made</span>
                  <span className="puzzle-value">
                    {personalAnalytics[selectedPlayer].puzzleSolvingMetrics?.movesContributed || 0}
                  </span>
                </div>
                <div className="puzzle-item">
                  <span className="puzzle-label">Success Rate</span>
                  <span className="puzzle-value">
                    {Math.round(((personalAnalytics[selectedPlayer].puzzleSolvingMetrics?.successfulMoves || 0) / 
                    Math.max(1, personalAnalytics[selectedPlayer].puzzleSolvingMetrics?.movesContributed || 1)) * 100)}%
                  </span>
                </div>
              </div>
            </div>

            {/* Collaboration */}
            <div className="metric-card">
              <h4>🤝 Collaboration</h4>
              <div className="collaboration-stats-individual">
                <div className="collab-item">
                  <span className="collab-label">Recommendations Sent</span>
                  <span className="collab-value">
                    {personalAnalytics[selectedPlayer].puzzleSolvingMetrics?.recommendationsSent || 0}
                  </span>
                </div>
                <div className="collab-item">
                  <span className="collab-label">Recommendations Received</span>
                  <span className="collab-value">
                    {personalAnalytics[selectedPlayer].puzzleSolvingMetrics?.recommendationsReceived || 0}
                  </span>
                </div>
                <div className="collab-item">
                  <span className="collab-label">Accepted</span>
                  <span className="collab-value">
                    {personalAnalytics[selectedPlayer].puzzleSolvingMetrics?.recommendationsAccepted || 0}
                  </span>
                </div>
              </div>
            </div>
          </div>
        </motion.div>
      )}

      {selectedPlayer === null && personalAnalytics?.length > 0 && (
        <div className="no-selection">
          <div className="no-selection-icon">👆</div>
          <p>Select a player above to view detailed analytics</p>
        </div>
      )}
    </motion.div>
  );

  return (
    <motion.div
      className="host-post-game"
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, y: -20 }}
      transition={{ duration: 0.5 }}
    >
      {/* Celebration Overlay */}
      <AnimatePresence>
        {showCelebration && (
          <motion.div
            className="celebration-overlay"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={{ duration: 0.5 }}
          >
            <motion.div
              className="celebration-content"
              initial={{ scale: 0, rotate: -180 }}
              animate={{ scale: 1, rotate: 0 }}
              exit={{ scale: 0, rotate: 180 }}
              transition={{ type: "spring", stiffness: 200 }}
            >
              <div className="celebration-icon">🎉</div>
              <h2>Congratulations!</h2>
              <p>Team successfully completed the puzzle!</p>
            </motion.div>
          </motion.div>
        )}
      </AnimatePresence>

      <div className="post-game-header">
        <motion.h2
          initial={{ opacity: 0, y: -20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.2 }}
        >
          📊 Game Analytics & Results
        </motion.h2>
        <motion.p
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ delay: 0.3 }}
        >
          Comprehensive analysis of team and individual performance
        </motion.p>
      </div>

      <div className="analytics-controls">
        <div className="view-tabs">
          <button
            className={`tab-btn ${viewMode === 'overview' ? 'active' : ''}`}
            onClick={() => setViewMode('overview')}
          >
            <span className="tab-icon">🎯</span>
            Overview
          </button>
          <button
            className={`tab-btn ${viewMode === 'team' ? 'active' : ''}`}
            onClick={() => setViewMode('team')}
          >
            <span className="tab-icon">👥</span>
            Team Analysis
          </button>
          <button
            className={`tab-btn ${viewMode === 'individual' ? 'active' : ''}`}
            onClick={() => setViewMode('individual')}
          >
            <span className="tab-icon">👤</span>
            Individual
          </button>
        </div>
      </div>

      <div className="analytics-content">
        <AnimatePresence mode="wait">
          {viewMode === 'overview' && <OverviewView key="overview" />}
          {viewMode === 'team' && <TeamView key="team" />}
          {viewMode === 'individual' && <IndividualView key="individual" />}
        </AnimatePresence>
      </div>
    </motion.div>
  );
};

export default HostPostGame;
