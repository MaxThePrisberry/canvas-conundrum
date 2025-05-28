import React, { useEffect, useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { SWAP_REQUEST_TIMEOUT } from '../constants';
import './SwapRequestList.css';

const SwapRequestList = ({ requests, onAction }) => {
  const [timedOutRequests, setTimedOutRequests] = useState(new Set());

  useEffect(() => {
    // Set up timers for each request
    const timers = {};

    requests.forEach(request => {
      if (!timedOutRequests.has(request.id)) {
        timers[request.id] = setTimeout(() => {
          setTimedOutRequests(prev => new Set([...prev, request.id]));
          onAction(request.id, 'timeout');
        }, SWAP_REQUEST_TIMEOUT);
      }
    });

    return () => {
      // Clean up timers
      Object.values(timers).forEach(timer => clearTimeout(timer));
    };
  }, [requests, onAction, timedOutRequests]);

  const handleAction = (requestId, action) => {
    onAction(requestId, action);
    
    // Haptic feedback
    if (window.navigator && window.navigator.vibrate) {
      window.navigator.vibrate(30);
    }
  };

  const getPositionLabel = (position) => {
    // Convert position to grid notation (A1, B2, etc.)
    const gridSize = Math.sqrt(64); // Assuming max 8x8 grid
    const row = Math.floor(position / gridSize);
    const col = position % gridSize;
    return `${String.fromCharCode(65 + row)}${col + 1}`;
  };

  if (requests.length === 0) {
    return null;
  }

  return (
    <div className="swap-request-list">
      <h3>Incoming Swap Requests</h3>
      <div className="request-scroll-container">
        <AnimatePresence>
          {requests.map((request, index) => {
            const timeRemaining = SWAP_REQUEST_TIMEOUT - (Date.now() - request.timestamp);
            const progress = Math.max(0, timeRemaining / SWAP_REQUEST_TIMEOUT);

            return (
              <motion.div
                key={request.id}
                className="swap-request-item"
                initial={{ opacity: 0, x: -50, height: 0 }}
                animate={{ opacity: 1, x: 0, height: 'auto' }}
                exit={{ opacity: 0, x: 50, height: 0 }}
                transition={{ duration: 0.3 }}
                style={{ marginBottom: index < requests.length - 1 ? '0.75rem' : 0 }}
              >
                <div className="request-info">
                  <span className="request-positions">
                    {getPositionLabel(request.from)} ↔ {getPositionLabel(request.to)}
                  </span>
                  <span className="request-player">
                    Player suggests swap
                  </span>
                </div>

                <div className="request-actions">
                  <motion.button
                    className="action-button accept"
                    onClick={() => handleAction(request.id, 'accept')}
                    whileHover={{ scale: 1.1 }}
                    whileTap={{ scale: 0.9 }}
                  >
                    ✓
                  </motion.button>
                  <motion.button
                    className="action-button reject"
                    onClick={() => handleAction(request.id, 'reject')}
                    whileHover={{ scale: 1.1 }}
                    whileTap={{ scale: 0.9 }}
                  >
                    ✗
                  </motion.button>
                </div>

                <motion.div
                  className="timeout-bar"
                  initial={{ scaleX: 1 }}
                  animate={{ scaleX: progress }}
                  transition={{ duration: 0.5, ease: "linear" }}
                  style={{
                    backgroundColor: progress > 0.3 ? '#10B981' : '#F59E0B'
                  }}
                />
              </motion.div>
            );
          })}
        </AnimatePresence>
      </div>
    </div>
  );
};

export default SwapRequestList;