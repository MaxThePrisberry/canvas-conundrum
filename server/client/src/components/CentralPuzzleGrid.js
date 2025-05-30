import React, { useState, useEffect } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { GAME_CONFIG } from '../constants';
import './CentralPuzzleGrid.css';

const CentralPuzzleGrid = ({ 
  centralPuzzleState,
  personalPuzzleState,
  playerId,
  onFragmentMove,
  onRecommendationRequest,
  onRecommendationResponse,
  incomingRecommendation
}) => {
  const [selectedCell, setSelectedCell] = useState(null);
  const [lastMoveTime, setLastMoveTime] = useState(0);
  const [showRecommendation, setShowRecommendation] = useState(false);

  const gridSize = centralPuzzleState?.gridSize || 4;
  const fragments = centralPuzzleState?.fragments || [];

  useEffect(() => {
    if (incomingRecommendation) {
      setShowRecommendation(true);
    }
  }, [incomingRecommendation]);

  const handleCellClick = (position) => {
    const now = Date.now();
    if (now - lastMoveTime < GAME_CONFIG.MOVEMENT_COOLDOWN) {
      console.log('Movement on cooldown');
      return;
    }

    const fragment = fragments.find(f => 
      f.position.x === Math.floor(position / gridSize) && 
      f.position.y === position % gridSize
    );

    if (!fragment) return;

    // Check if player can move this fragment
    const canMove = fragment.playerId === playerId || !fragment.playerId;
    if (!canMove) {
      console.log('Cannot move this fragment');
      return;
    }

    if (selectedCell === null) {
      setSelectedCell(position);
      if (window.navigator && window.navigator.vibrate) {
        window.navigator.vibrate(20);
      }
    } else if (selectedCell === position) {
      setSelectedCell(null);
    } else {
      // Request fragment move
      const fromPos = {
        x: Math.floor(selectedCell / gridSize),
        y: selectedCell % gridSize
      };
      const toPos = {
        x: Math.floor(position / gridSize),
        y: position % gridSize
      };

      const movingFragment = fragments.find(f => 
        f.position.x === fromPos.x && f.position.y === fromPos.y
      );

      if (movingFragment) {
        onFragmentMove(movingFragment.id, toPos);
        setLastMoveTime(now);
      }

      setSelectedCell(null);
    }
  };

  const getFragmentAtPosition = (x, y) => {
    return fragments.find(f => f.position.x === x && f.position.y === y);
  };

  const isHighlighted = (position) => {
    if (!personalPuzzleState?.guideHighlight) return false;
    
    const x = Math.floor(position / gridSize);
    const y = position % gridSize;
    
    return personalPuzzleState.guideHighlight.positions.some(
      pos => pos.x === x && pos.y === y
    );
  };

  const handleRecommendationResponse = (accepted) => {
    if (incomingRecommendation) {
      onRecommendationResponse(incomingRecommendation.id, accepted);
      setShowRecommendation(false);
    }
  };

  return (
    <div className="central-puzzle-grid">
      <div className="grid-header">
        <h3>Team Puzzle Grid</h3>
        <p>Work together to arrange the fragments!</p>
      </div>

      <div 
        className="master-grid"
        style={{ 
          gridTemplateColumns: `repeat(${gridSize}, 1fr)`,
          gridTemplateRows: `repeat(${gridSize}, 1fr)`
        }}
      >
        {Array.from({ length: gridSize * gridSize }).map((_, index) => {
          const x = Math.floor(index / gridSize);
          const y = index % gridSize;
          const fragment = getFragmentAtPosition(x, y);
          const isSelected = selectedCell === index;
          const isGuideHighlighted = isHighlighted(index);
          
          return (
            <motion.div
              key={index}
              className={`grid-cell ${
                isSelected ? 'selected' : ''
              } ${
                isGuideHighlighted ? 'highlighted' : ''
              } ${
                fragment ? 'has-fragment' : 'empty'
              }`}
              onClick={() => handleCellClick(index)}
              whileHover={{ scale: 1.05 }}
              whileTap={{ scale: 0.95 }}
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              transition={{ delay: index * 0.02 }}
            >
              {fragment ? (
                <div 
                  className={`fragment ${
                    fragment.playerId === playerId ? 'owned' : ''
                  } ${
                    !fragment.playerId ? 'unassigned' : ''
                  }`}
                  style={{
                    backgroundImage: `url(/images/puzzles/${centralPuzzleState.imageId}/fragment_${fragment.id}.png)`
                  }}
                >
                  {fragment.playerId === playerId && (
                    <div className="ownership-indicator">You</div>
                  )}
                </div>
              ) : (
                <div className="cell-label">
                  {String.fromCharCode(65 + x)}{y + 1}
                </div>
              )}
            </motion.div>
          );
        })}
      </div>

      <AnimatePresence>
        {showRecommendation && incomingRecommendation && (
          <motion.div
            className="recommendation-popup"
            initial={{ opacity: 0, y: 50 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: 50 }}
          >
            <div className="recommendation-content">
              <h4>Swap Suggestion</h4>
              <p>Another player suggests moving:</p>
              <div className="recommendation-details">
                <span className="from-pos">
                  {String.fromCharCode(65 + incomingRecommendation.suggestedFromPos.x)}
                  {incomingRecommendation.suggestedFromPos.y + 1}
                </span>
                <span className="arrow">→</span>
                <span className="to-pos">
                  {String.fromCharCode(65 + incomingRecommendation.suggestedToPos.x)}
                  {incomingRecommendation.suggestedToPos.y + 1}
                </span>
              </div>
              <div className="recommendation-actions">
                <button 
                  className="btn-secondary"
                  onClick={() => handleRecommendationResponse(false)}
                >
                  Decline
                </button>
                <button 
                  className="btn-primary"
                  onClick={() => handleRecommendationResponse(true)}
                >
                  Accept
                </button>
              </div>
            </div>
          </motion.div>
        )}
      </AnimatePresence>

      <div className="grid-info">
        <p>💡 Tap two cells to swap fragments</p>
        {personalPuzzleState?.guideHighlight && (
          <p className="guide-hint">✨ Highlighted areas show optimal placement</p>
        )}
      </div>
    </div>
  );
};

export default CentralPuzzleGrid;
