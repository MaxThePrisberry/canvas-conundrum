import React, { useState, useEffect, useRef } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import PhaseTransition from './PhaseTransition';
import SwapRequestList from './SwapRequestList';
import { MessageType } from '../constants';
import './PuzzleAssemblyPhase.css';

const PuzzleAssemblyPhase = ({ 
  puzzleData, 
  playerId,
  onSegmentCompleted,
  onFragmentMoveRequest,
  sendMessage
}) => {
  const [showTransition, setShowTransition] = useState(true);
  const [showPrompt, setShowPrompt] = useState(true);
  const [puzzleStarted, setPuzzleStarted] = useState(false);
  const [showFullImage, setShowFullImage] = useState(false);
  const [pieces, setPieces] = useState([]);
  const [selectedPiece, setSelectedPiece] = useState(null);
  const [isSolved, setIsSolved] = useState(false);
  const [showMasterGrid, setShowMasterGrid] = useState(false);
  const [masterFragments, setMasterFragments] = useState([]);
  const [selectedGridCells, setSelectedGridCells] = useState([]);
  const [swapRequests, setSwapRequests] = useState([]);
  const [imageDisplayTime, setImageDisplayTime] = useState(0);
  
  const canvasRef = useRef(null);
  const imageRef = useRef(null);

  // Hide transition after animation
  useEffect(() => {
    const timer = setTimeout(() => setShowTransition(false), 2000);
    return () => clearTimeout(timer);
  }, []);

  // Handle puzzle phase start message
  useEffect(() => {
    if (!puzzleData) return;

    const handleMessage = (message) => {
      switch (message.type) {
        case MessageType.PUZZLE_PHASE_START:
          setPuzzleStarted(true);
          setShowPrompt(false);
          // Calculate display time based on clarity tokens (would come from server)
          const displayTime = message.payload.totalTime || 3;
          setImageDisplayTime(displayTime);
          startPuzzle(displayTime);
          break;

        case MessageType.CENTRAL_PUZZLE_STATE:
          setMasterFragments(message.payload.fragments || []);
          break;

        case MessageType.FRAGMENT_MOVE_RESPONSE:
          handleFragmentMoveResponse(message.payload);
          break;

        default:
          break;
      }
    };

    // In a real app, this would be handled by the WebSocket connection
    // For now, we'll simulate starting the puzzle after a delay
    if (!puzzleStarted && !showTransition) {
      setTimeout(() => {
        setPuzzleStarted(true);
        setShowPrompt(false);
        startPuzzle(3); // 3 seconds default
      }, 3000);
    }
  }, [puzzleData, puzzleStarted, showTransition]);

  const startPuzzle = (displayTime) => {
    // Show full image briefly
    setShowFullImage(true);
    
    setTimeout(() => {
      setShowFullImage(false);
      initializePuzzle();
    }, displayTime * 1000);
  };

  const initializePuzzle = () => {
    const img = new Image();
    img.onload = () => {
      imageRef.current = img;
      createPuzzlePieces();
    };
    // Placeholder URL - in real app, use: `/images/puzzles/${puzzleData.imageId}/${puzzleData.segmentId}.png`
    img.src = `/images/puzzles/${puzzleData.imageId}/${puzzleData.segmentId}.png`;
  };

  const createPuzzlePieces = () => {
    const pieceWidth = imageRef.current.width / 4;
    const pieceHeight = imageRef.current.height / 4;
    const newPieces = [];

    // Create 16 pieces in correct order
    for (let y = 0; y < 4; y++) {
      for (let x = 0; x < 4; x++) {
        newPieces.push({
          id: y * 4 + x,
          correctPosition: y * 4 + x,
          currentPosition: y * 4 + x,
          x: x * pieceWidth,
          y: y * pieceHeight,
          width: pieceWidth,
          height: pieceHeight
        });
      }
    }

    // Shuffle pieces
    const shuffled = [...newPieces];
    for (let i = shuffled.length - 1; i > 0; i--) {
      const j = Math.floor(Math.random() * (i + 1));
      const tempPos = shuffled[i].currentPosition;
      shuffled[i].currentPosition = shuffled[j].currentPosition;
      shuffled[j].currentPosition = tempPos;
    }

    setPieces(shuffled);
  };

  const handlePieceClick = (piece) => {
    if (isSolved) return;

    if (selectedPiece === null) {
      setSelectedPiece(piece.id);
      // Haptic feedback
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
      
      if (piece1 && piece2) {
        const tempPos = piece1.currentPosition;
        piece1.currentPosition = piece2.currentPosition;
        piece2.currentPosition = tempPos;
      }

      // Check if puzzle is solved
      const solved = checkIfSolved(newPieces);
      if (solved && !isSolved) {
        handlePuzzleSolved();
      }

      return newPieces;
    });

    // Haptic feedback
    if (window.navigator && window.navigator.vibrate) {
      window.navigator.vibrate(30);
    }
  };

  const checkIfSolved = (piecesToCheck) => {
    return piecesToCheck.every(piece => piece.currentPosition === piece.correctPosition);
  };

  const handlePuzzleSolved = () => {
    setIsSolved(true);
    onSegmentCompleted(puzzleData.segmentId);
    
    // Victory haptic feedback
    if (window.navigator && window.navigator.vibrate) {
      window.navigator.vibrate([100, 50, 100, 50, 200]);
    }

    // Transition to master grid after celebration
    setTimeout(() => {
      setShowMasterGrid(true);
    }, 2000);
  };

  const handleGridCellClick = (position) => {
    if (selectedGridCells.length === 0) {
      setSelectedGridCells([position]);
    } else if (selectedGridCells.length === 1) {
      if (selectedGridCells[0] !== position) {
        // Send swap request
        const fragmentId1 = `fragment_${selectedGridCells[0]}`;
        const fragmentId2 = `fragment_${position}`;
        
        onFragmentMoveRequest(fragmentId1, position);
        
        // Add to local swap requests for UI feedback
        const newRequest = {
          id: Date.now(),
          from: selectedGridCells[0],
          to: position,
          status: 'pending'
        };
        setSwapRequests(prev => [...prev, newRequest]);
      }
      setSelectedGridCells([]);
    }

    // Haptic feedback
    if (window.navigator && window.navigator.vibrate) {
      window.navigator.vibrate(20);
    }
  };

  const handleSwapRequestAction = (requestId, action) => {
    const request = swapRequests.find(r => r.id === requestId);
    if (!request) return;

    if (action === 'accept') {
      // Send accept message to server
      // This would be implemented with proper WebSocket messaging
      console.log('Accepting swap request', request);
    }

    // Remove from list
    setSwapRequests(prev => prev.filter(r => r.id !== requestId));
  };

  const handleFragmentMoveResponse = (response) => {
    if (response.status === 'success') {
      // Update local state if needed
      console.log('Fragment move successful');
    }
  };

  const renderPuzzle = () => {
    if (!pieces.length || !imageRef.current) return null;

    return (
      <div className="puzzle-container">
        <motion.div 
          className="puzzle-grid"
          initial={{ opacity: 0, scale: 0.8 }}
          animate={{ opacity: 1, scale: 1 }}
          transition={{ delay: 0.5 }}
        >
          {pieces.sort((a, b) => a.currentPosition - b.currentPosition).map((piece, index) => {
            const row = Math.floor(index / 4);
            const col = index % 4;
            
            return (
              <motion.div
                key={piece.id}
                className={`puzzle-piece ${selectedPiece === piece.id ? 'selected' : ''}`}
                onClick={() => handlePieceClick(piece)}
                whileHover={{ scale: 1.05 }}
                whileTap={{ scale: 0.95 }}
                initial={{ opacity: 0, scale: 0 }}
                animate={{ opacity: 1, scale: 1 }}
                transition={{ delay: index * 0.05 }}
                style={{
                  backgroundImage: `url(/images/puzzles/${puzzleData.imageId}/${puzzleData.segmentId}.png)`,
                  backgroundPosition: `-${piece.x}px -${piece.y}px`,
                  backgroundSize: '400% 400%',
                  gridRow: row + 1,
                  gridColumn: col + 1
                }}
              />
            );
          })}
        </motion.div>

        {isSolved && !showMasterGrid && (
          <motion.div
            className="solved-overlay"
            initial={{ opacity: 0, scale: 0 }}
            animate={{ opacity: 1, scale: 1 }}
            transition={{ type: "spring", stiffness: 200 }}
          >
            <div className="solved-content">
              <motion.div
                className="solved-icon"
                animate={{ rotate: [0, 360] }}
                transition={{ duration: 1 }}
              >
                ✨
              </motion.div>
              <h2>Puzzle Solved!</h2>
              <p>Great work! Now help solve the master puzzle...</p>
            </div>
          </motion.div>
        )}
      </div>
    );
  };

  const renderMasterGrid = () => {
    const gridSize = puzzleData.gridSize || 4;
    
    return (
      <div className="master-grid-container">
        <h2>Master Puzzle Grid</h2>
        <p>Tap two squares to suggest a swap</p>
        
        <motion.div 
          className="master-grid"
          style={{ gridTemplateColumns: `repeat(${gridSize}, 1fr)` }}
          initial={{ opacity: 0, scale: 0.8 }}
          animate={{ opacity: 1, scale: 1 }}
        >
          {Array.from({ length: gridSize * gridSize }).map((_, index) => (
            <motion.button
              key={index}
              className={`grid-cell ${
                selectedGridCells.includes(index) ? 'selected' : ''
              }`}
              onClick={() => handleGridCellClick(index)}
              whileHover={{ scale: 1.1 }}
              whileTap={{ scale: 0.9 }}
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              transition={{ delay: index * 0.02 }}
            >
              {String.fromCharCode(65 + Math.floor(index / gridSize))}{(index % gridSize) + 1}
            </motion.button>
          ))}
        </motion.div>

        <SwapRequestList
          requests={swapRequests}
          onAction={handleSwapRequestAction}
        />
      </div>
    );
  };

  return (
    <div className="puzzle-assembly-phase">
      <AnimatePresence>
        {showTransition && (
          <PhaseTransition 
            title="Puzzle Assembly"
            subtitle="Work together to restore the masterpiece"
          />
        )}
      </AnimatePresence>

      <AnimatePresence>
        {showPrompt && (
          <motion.div
            className="return-prompt"
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
          >
            <motion.div
              className="prompt-icon"
              animate={{ scale: [1, 1.1, 1] }}
              transition={{ duration: 2, repeat: Infinity }}
            >
              🏃‍♂️
            </motion.div>
            <h2>Return to the Gym!</h2>
            <p>The puzzle phase is about to begin</p>
          </motion.div>
        )}
      </AnimatePresence>

      <AnimatePresence>
        {showFullImage && (
          <motion.div
            className="full-image-preview"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
          >
            <img 
              src={`/images/puzzles/${puzzleData.imageId}/${puzzleData.segmentId}.png`}
              alt="Puzzle preview"
            />
            <motion.div
              className="preview-timer"
              initial={{ scaleX: 1 }}
              animate={{ scaleX: 0 }}
              transition={{ duration: imageDisplayTime, ease: "linear" }}
            />
          </motion.div>
        )}
      </AnimatePresence>

      {!showPrompt && !showFullImage && !showMasterGrid && renderPuzzle()}
      
      {showMasterGrid && renderMasterGrid()}
    </div>
  );
};

export default PuzzleAssemblyPhase;