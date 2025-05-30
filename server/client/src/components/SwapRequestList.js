import React, { useEffect, useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { SWAP_REQUEST_TIMEOUT, HAPTIC_PATTERNS, Colors } from '../constants';
import './SwapRequestList.css';

const SwapRequestList = ({ requests, onAction }) => {
  const [timedOutRequests, setTimedOutRequests] = useState(new Set());

  useEffect(() => {
    // Set up timers for each request
    const timers = {};

    requests.forEach(request => {
      if (!timedOutRequests.has(request.id)) {
        const timeElapsed = Date.now() - new Date(request.timestamp).getTime();
        const remainingTime = Math.max(0, SWAP_REQUEST_TIMEOUT - timeElapsed);
        
        if (remainingTime > 0) {
          timers[request.id] = setTimeout(() => {
            setTimedOutRequests(prev => new Set([...prev, request.id]));
            onAction(request.id, 'timeout');
          }, remainingTime);
        } else {
          // Request has already timed out
          setTimedOutRequests(prev => new Set([...prev, request.id]));
          onAction(request.id, 'timeout');
        }
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
      const pattern = action === 'accept' ? HAPTIC_PATTERNS.SUCCESS : HAPTIC_PATTERNS.MEDIUM;
      window.navigator.vibrate(pattern);
    }
  };

  const getPositionLabel = (position) => {
    // Convert grid position to readable notation (A1, B2, etc.)
    return `${String.fromCharCode(65 + position.x)}${position.y + 1}`;
  };

  const getTimeProgress = (request) => {
    const timeElapsed = Date.now() - new Date(request.timestamp).getTime();
    const progress = Math.max(0, 1 - (timeElapsed / SWAP_REQUEST_TIMEOUT));
    return progress;
  };

  const getTimeColor = (progress) => {
    if (progress > 0.5) return Colors.success;
    if (progress > 0.2) return Colors.warning;
    return Colors.error;
  };

  if (!requests || requests.length === 0) {
    return null;
  }

  return (
    <motion.div
      className="swap-request-list"
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, y: 20 }}
      transition={{ duration: 0.3 }}
    >
      <div className="request-header">
        <h3>Swap Suggestions</h3>
        <div className="request-count">
          {requests.length}
        </div>
      </div>

      <div className="request-scroll-container">
        <AnimatePresence>
          {requests.map((request, index) => {
            const timeProgress = getTimeProgress(request);
            const timeColor = getTimeColor(timeProgress);
            const isTimedOut = timedOutRequests.has(request.id);

            if (isTimedOut) return null;

            return (
              <motion.div
                key={request.id}
                className="swap-request-item"
                initial={{ opacity: 0, x: -50, height: 0 }}
                animate={{ opacity: 1, x: 0, height: 'auto' }}
                exit={{ opacity: 0, x: 50, height: 0 }}
                transition={{ 
                  duration: 0.4,
                  type: "spring",
                  stiffness: 200
                }}
                style={{ 
                  marginBottom: index < requests.length - 1 ? '1rem' : 0,
                  '--time-color': timeColor
                }}
              >
                <div className="request-content">
                  <div className="request-info">
                    <div className="swap-positions">
                      <span className="position from-position">
                        {getPositionLabel(request.suggestedFromPos)}
                      </span>
                      <motion.div 
                        className="swap-arrow"
                        animate={{ x: [0, 5, 0] }}
                        transition={{ duration: 1.5, repeat: Infinity }}
                      >
                        ↔
                      </motion.div>
                      <span className="position to-position">
                        {getPositionLabel(request.suggestedToPos)}
                      </span>
                    </div>
                    <p className="request-description">
                      Team suggestion for optimal placement
                    </p>
                  </div>

                  <div className="request-actions">
                    <motion.button
                      className="action-button reject"
                      onClick={() => handleAction(request.id, 'reject')}
                      whileHover={{ scale: 1.05, y: -2 }}
                      whileTap={{ scale: 0.95 }}
                    >
                      <span className="action-icon">✗</span>
                      <span className="action-label">Decline</span>
                    </motion.button>
                    
                    <motion.button
                      className="action-button accept"
                      onClick={() => handleAction(request.id, 'accept')}
                      whileHover={{ scale: 1.05, y: -2 }}
                      whileTap={{ scale: 0.95 }}
                    >
                      <span className="action-icon">✓</span>
                      <span className="action-label">Accept</span>
                    </motion.button>
                  </div>
                </div>

                <motion.div
                  className="timeout-bar"
                  initial={{ scaleX: 1 }}
                  animate={{ scaleX: timeProgress }}
                  transition={{ duration: 0.5, ease: "linear" }}
                  style={{ backgroundColor: timeColor }}
                />

                {timeProgress < 0.3 && (
                  <motion.div
                    className="urgency-indicator"
                    animate={{ opacity: [0.5, 1, 0.5] }}
                    transition={{ duration: 1, repeat: Infinity }}
                  >
                    ⚡ Expires soon!
                  </motion.div>
                )}
              </motion.div>
            );
          })}
        </AnimatePresence>
      </div>

      {requests.length > 3 && (
        <div className="scroll-hint">
          <motion.div
            animate={{ y: [0, 5, 0] }}
            transition={{ duration: 2, repeat: Infinity }}
          >
            ↓ Scroll for more
          </motion.div>
        </div>
      )}
    </motion.div>
  );
};

export default SwapRequestList;
