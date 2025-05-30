import React, { useState, useEffect, useRef } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { GAME_CONFIG } from '../constants';
import './IndividualPuzzle.css';

const IndividualPuzzle = ({ 
  puzzleData, 
  preSolvedPieces, 
  onComplete,
  timeRemaining 
}) => {
  const [pieces, setPieces] = useState([]);
  const [selectedPiece, setSelectedPiece] = useState(null);
  const [isSolved, setIsSolved] = useState(false);
  const imageRef = useRef(null);

  useEffect(() => {
    // Initialize puzzle pieces
    const initializePuzzle = () => {
      const newPieces = [];
      const totalPieces = GAME_CONFIG.INDIVIDUAL_PUZZLE_PIECES;
      
      // Create pieces in correct positions
      for (let i = 0; i < totalPieces; i++) {
        const row = Math.floor(i / 4);
        const col = i % 4;
        
        newPieces.push({
          id: i,
          correctPosition: i,
          currentPosition: i,
          row,
          col,
          isPreSolved: preSolvedPieces > i
        });
      }

      // Shuffle non-presolved pieces
      const shuffleablePieces = newPieces.filter(p => !p.isPreSolved);
      const shuffledPositions = shuffleablePieces.map(p => p.currentPosition).sort(() => Math.random() - 0.5);
      
      shuffleablePieces.forEach((piece, index) => {
        const targetPiece = newPieces.find(p => p.id === piece.id);
        targetPiece.currentPosition = shuffledPositions[index];
      });

      setPieces(newPieces);
    };

    initializePuzzle();
  }, [preSolvedPieces]);

  const handlePieceClick = (piece) => {
    if (isSolved || piece.isPreSolved) return;

    if (selectedPiece === null) {
      setSelectedPiece(piece.id);
      if (window.navigator && window.navigator.vibrate) {
        window.navigator.vibrate(20);
      }
    } else if (selectedPiece === piece.id) {
      setSelectedPiece(null);
    } else {
      // Swap pieces
      swapPieces(selectedPiece, piece.id);
      setSelectedPiece(null);
    }
  };

  const swapPieces = (id1, id2) => {
    setPieces(prevPieces => {
      const newPieces = [...prevPieces];
      const piece1 = newPieces.find(p => p.id === id1);
      const piece2 = newPieces.find(p => p.id === id2);
      
      if (piece1 && piece2 && !piece1.isPreSolved && !piece2.isPreSolved) {
        const tempPos = piece1.currentPosition;
        piece1.currentPosition = piece2.currentPosition;
        piece2.currentPosition = tempPos;

        // Check if solved
        const solved = checkIfSolved(newPieces);
        if (solved && !isSolved) {
          handlePuzzleSolved();
        }
      }

      return newPieces;
    });

    if (window.navigator && window.navigator.vibrate) {
      window.navigator.vibrate(30);
    }
  };

  const checkIfSolved = (piecesToCheck) => {
    return piecesToCheck.every(piece => piece.currentPosition === piece.correctPosition);
  };

  const handlePuzzleSolved = () => {
    setIsSolved(true);
    onComplete();
    
    if (window.navigator && window.navigator.vibrate) {
      window.navigator.vibrate([100, 50, 100, 50, 200]);
    }
  };

  return (
    <div className="individual-puzzle">
      <div className="puzzle-header">
        <h3>Solve Your Puzzle Segment</h3>
        <div className="time-remaining">
          <span className="time-icon">⏱️</span>
          <span>{Math.floor(timeRemaining / 60)}:{(timeRemaining % 60).toString().padStart(2, '0')}</span>
        </div>
      </div>

      <div className="puzzle-container">
        <div className="puzzle-grid">
          {pieces.sort((a, b) => a.currentPosition - b.currentPosition).map((piece, index) => {
            const row = Math.floor(index / 4);
            const col = index % 4;
            
            return (
              <motion.div
                key={piece.id}
                className={`puzzle-piece ${
                  selectedPiece === piece.id ? 'selected' : ''
                } ${piece.isPreSolved ? 'pre-solved' : ''}`}
                onClick={() => handlePieceClick(piece)}
                whileHover={!piece.isPreSolved ? { scale: 1.05 } : {}}
                whileTap={!piece.isPreSolved ? { scale: 0.95 } : {}}
                style={{
                  gridRow: row + 1,
                  gridColumn: col + 1,
                  backgroundImage: `url(/images/puzzles/${puzzleData.imageId}/${puzzleData.segmentId}.png)`,
                  backgroundPosition: `${-(piece.col * 25)}% ${-(piece.row * 25)}%`,
                  backgroundSize: '400% 400%'
                }}
                initial={{ opacity: 0, scale: 0 }}
                animate={{ opacity: 1, scale: 1 }}
                transition={{ delay: index * 0.03 }}
              >
                {piece.isPreSolved && (
                  <div className="pre-solved-indicator">
                    <span>✓</span>
                  </div>
                )}
              </motion.div>
            );
          })}
        </div>

        <AnimatePresence>
          {isSolved && (
            <motion.div
              className="solved-overlay"
              initial={{ opacity: 0, scale: 0.8 }}
              animate={{ opacity: 1, scale: 1 }}
              exit={{ opacity: 0, scale: 0.8 }}
            >
              <div className="solved-content">
                <div className="solved-icon">✨</div>
                <h2>Segment Complete!</h2>
                <p>Your fragment is being added to the team puzzle...</p>
              </div>
            </motion.div>
          )}
        </AnimatePresence>
      </div>

      <div className="puzzle-info">
        <p>
          {preSolvedPieces > 0 && (
            <span className="pre-solved-info">
              {preSolvedPieces} pieces pre-solved by Anchor tokens!
            </span>
          )}
        </p>
      </div>
    </div>
  );
};

export default IndividualPuzzle;
