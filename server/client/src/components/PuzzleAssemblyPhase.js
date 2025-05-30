import React, { useState, useEffect } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import PhaseTransition from './PhaseTransition';
import IndividualPuzzle from './IndividualPuzzle';
import CentralPuzzleGrid from './CentralPuzzleGrid';
import { MessageType, GAME_CONFIG } from '../constants';
import './PuzzleAssemblyPhase.css';

const PuzzleAssemblyPhase = ({ 
  puzzleData, 
  imagePreview,
  puzzleTimer,
  playerId,
  individualPuzzleComplete,
  centralPuzzleState,
  personalPuzzleState,
  incomingRecommendation,
  onSegmentCompleted,
  onFragmentMoveRequest,
  onRecommendationRequest,
  onRecommendationResponse
}) => {
  const [showTransition, setShowTransition] = useState(true);
  const [showImagePreview, setShowImagePreview] = useState(false);
  const [timeRemaining, setTimeRemaining] = useState(puzzleTimer?.totalTime || GAME_CONFIG.BASE_PUZZLE_TIME);
  const [preSolvedPieces, setPreSolvedPieces] = useState(0);

  // Calculate pre-solved pieces from puzzle data
  useEffect(() => {
    if (puzzleData?.preSolved) {
      // Anchor tokens pre-solve up to 12 of 16 pieces
      setPreSolvedPieces(Math.min(12, Math.floor(Math.random() * 8) + 5));
    }
  }, [puzzleData]);

  // Handle phase transition
  useEffect(() => {
    const timer = setTimeout(() => {
      setShowTransition(false);
      if (imagePreview) {
        setShowImagePreview(true);
        setTimeout(() => {
          setShowImagePreview(false);
        }, (imagePreview.duration || 3) * 1000);
      }
    }, 2000);
    return () => clearTimeout(timer);
  }, [imagePreview]);

  // Timer countdown
  useEffect(() => {
    if (!puzzleTimer || showTransition || showImagePreview) return;

    const interval = setInterval(() => {
      setTimeRemaining(prev => {
        if (prev <= 0) return 0;
        return prev - 1;
      });
    }, 1000);

    return () => clearInterval(interval);
  }, [puzzleTimer, showTransition, showImagePreview]);

  const handleIndividualComplete = () => {
    onSegmentCompleted(puzzleData.segmentId);
  };

  return (
    <div className="puzzle-assembly-phase">
      <AnimatePresence>
        {showTransition && (
          <PhaseTransition 
            title="Puzzle Assembly"
            subtitle="Restore the masterpiece together"
          />
        )}
      </AnimatePresence>

      <AnimatePresence>
        {showImagePreview && imagePreview && (
          <motion.div
            className="image-preview-overlay"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
          >
            <div className="preview-container">
              <img 
                src={`/images/puzzles/${imagePreview.imageId}/complete.png`}
                alt="Complete puzzle"
                className="preview-image"
              />
              <div className="preview-timer">
                <div 
                  className="timer-bar"
                  style={{ animationDuration: `${imagePreview.duration}s` }}
                />
              </div>
              <p className="preview-hint">Memorize this image!</p>
            </div>
          </motion.div>
        )}
      </AnimatePresence>

      {!showTransition && !showImagePreview && (
        <div className="puzzle-content">
          <div className="timer-display">
            <span className="timer-label">Time Remaining</span>
            <span className={`timer-value ${timeRemaining < 60 ? 'warning' : ''}`}>
              {Math.floor(timeRemaining / 60)}:{(timeRemaining % 60).toString().padStart(2, '0')}
            </span>
          </div>

          <AnimatePresence mode="wait">
            {!individualPuzzleComplete ? (
              <motion.div
                key="individual"
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0, y: -20 }}
              >
                <IndividualPuzzle
                  puzzleData={puzzleData}
                  preSolvedPieces={preSolvedPieces}
                  onComplete={handleIndividualComplete}
                  timeRemaining={timeRemaining}
                />
              </motion.div>
            ) : (
              <motion.div
                key="central"
                initial={{ opacity: 0, scale: 0.9 }}
                animate={{ opacity: 1, scale: 1 }}
                exit={{ opacity: 0, scale: 0.9 }}
              >
                <CentralPuzzleGrid
                  centralPuzzleState={centralPuzzleState}
                  personalPuzzleState={personalPuzzleState}
                  playerId={playerId}
                  onFragmentMove={onFragmentMoveRequest}
                  onRecommendationRequest={onRecommendationRequest}
                  onRecommendationResponse={onRecommendationResponse}
                  incomingRecommendation={incomingRecommendation}
                />
              </motion.div>
            )}
          </AnimatePresence>

          {individualPuzzleComplete && (
            <div className="collaboration-hint">
              <p>💡 Communicate with your team to solve the master puzzle!</p>
            </div>
          )}
        </div>
      )}
    </div>
  );
};

export default PuzzleAssemblyPhase;
