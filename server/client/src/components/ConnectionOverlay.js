import React from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import './ConnectionOverlay.css';

const ConnectionOverlay = ({ isConnected, isReconnecting }) => {
  const showOverlay = !isConnected || isReconnecting;

  return (
    <AnimatePresence>
      {showOverlay && (
        <motion.div
          className="connection-overlay"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          transition={{ duration: 0.3 }}
        >
          <div className="connection-content">
            {isReconnecting ? (
              <>
                <motion.div
                  className="reconnecting-spinner"
                  animate={{ rotate: 360 }}
                  transition={{ duration: 2, repeat: Infinity, ease: "linear" }}
                >
                  <div className="spinner-ring">
                    <div className="spinner-dot" />
                    <div className="spinner-dot" />
                    <div className="spinner-dot" />
                    <div className="spinner-dot" />
                  </div>
                </motion.div>
                <motion.h2
                  initial={{ opacity: 0, y: 20 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{ delay: 0.2 }}
                >
                  Syncing...
                </motion.h2>
                <motion.p
                  initial={{ opacity: 0 }}
                  animate={{ opacity: 1 }}
                  transition={{ delay: 0.4 }}
                >
                  Reconnecting to game server
                </motion.p>
              </>
            ) : (
              <>
                <motion.div
                  className="disconnected-icon"
                  initial={{ scale: 0 }}
                  animate={{ scale: 1 }}
                  transition={{ type: "spring", stiffness: 200 }}
                >
                  <svg width="80" height="80" viewBox="0 0 24 24" fill="none">
                    <path
                      d="M3 7L21 7"
                      stroke="currentColor"
                      strokeWidth="2"
                      strokeLinecap="round"
                    />
                    <path
                      d="M8 12L16 12"
                      stroke="currentColor"
                      strokeWidth="2"
                      strokeLinecap="round"
                    />
                    <path
                      d="M12 17L12 17.01"
                      stroke="currentColor"
                      strokeWidth="2"
                      strokeLinecap="round"
                    />
                    <motion.path
                      d="M2 2L22 22"
                      stroke="#EF4444"
                      strokeWidth="3"
                      strokeLinecap="round"
                      initial={{ pathLength: 0 }}
                      animate={{ pathLength: 1 }}
                      transition={{ duration: 0.5 }}
                    />
                  </svg>
                </motion.div>
                <motion.h2
                  initial={{ opacity: 0, y: 20 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{ delay: 0.2 }}
                >
                  Connection Lost
                </motion.h2>
                <motion.p
                  initial={{ opacity: 0 }}
                  animate={{ opacity: 1 }}
                  transition={{ delay: 0.4 }}
                >
                  Attempting to reconnect...
                </motion.p>
              </>
            )}
          </div>
        </motion.div>
      )}
    </AnimatePresence>
  );
};

export default ConnectionOverlay;