import React, { useState } from 'react';
import { motion } from 'framer-motion';
import './HostLanding.css';

const HostLanding = ({ onConnect }) => {
  const [hashCode, setHashCode] = useState('');
  const [error, setError] = useState('');
  const [isConnecting, setIsConnecting] = useState(false);

  const handleSubmit = (e) => {
    e.preventDefault();
    
    if (!hashCode.trim()) {
      setError('Please enter a hash code');
      return;
    }

    if (hashCode.length < 8) {
      setError('Hash code must be at least 8 characters');
      return;
    }

    setError('');
    setIsConnecting(true);
    
    // Add a small delay for UX
    setTimeout(() => {
      onConnect(hashCode.trim());
      setIsConnecting(false);
    }, 500);
  };

  const handleInputChange = (e) => {
    setHashCode(e.target.value);
    setError('');
  };

  return (
    <div className="host-landing">
      <div className="landing-background">
        <div className="floating-shape shape-1"></div>
        <div className="floating-shape shape-2"></div>
        <div className="floating-shape shape-3"></div>
      </div>

      <motion.div
        className="landing-content"
        initial={{ opacity: 0, y: 30 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.8 }}
      >
        <motion.div
          className="host-logo"
          initial={{ scale: 0 }}
          animate={{ scale: 1 }}
          transition={{ type: "spring", stiffness: 200, delay: 0.2 }}
        >
          <div className="logo-icon">🎮</div>
          <div className="logo-rings">
            <div className="ring ring-1"></div>
            <div className="ring ring-2"></div>
            <div className="ring ring-3"></div>
          </div>
        </motion.div>

        <motion.h1
          className="landing-title"
          initial={{ opacity: 0, y: -20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.4 }}
        >
          Canvas Conundrum
        </motion.h1>

        <motion.h2
          className="landing-subtitle"
          initial={{ opacity: 0, y: -20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.5 }}
        >
          Host Dashboard
        </motion.h2>

        <motion.p
          className="landing-description"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ delay: 0.6 }}
        >
          Enter your unique host code to start managing the game
        </motion.p>

        <motion.form
          className="hash-form"
          onSubmit={handleSubmit}
          initial={{ opacity: 0, scale: 0.9 }}
          animate={{ opacity: 1, scale: 1 }}
          transition={{ delay: 0.7 }}
        >
          <div className="input-group">
            <label htmlFor="hashCode" className="input-label">
              Host Code
            </label>
            <input
              type="text"
              id="hashCode"
              value={hashCode}
              onChange={handleInputChange}
              placeholder="Enter your host code..."
              className={`hash-input ${error ? 'error' : ''}`}
              disabled={isConnecting}
              autoFocus
              autoComplete="off"
              spellCheck="false"
            />
            {error && (
              <motion.div
                className="error-message"
                initial={{ opacity: 0, y: -10 }}
                animate={{ opacity: 1, y: 0 }}
              >
                {error}
              </motion.div>
            )}
          </div>

          <motion.button
            type="submit"
            className="connect-button"
            disabled={isConnecting || !hashCode.trim()}
            whileHover={!isConnecting ? { y: -2 } : {}}
            whileTap={!isConnecting ? { scale: 0.98 } : {}}
          >
            {isConnecting ? (
              <>
                <div className="spinner"></div>
                Connecting...
              </>
            ) : (
              <>
                <span className="button-icon">🚀</span>
                Connect as Host
              </>
            )}
          </motion.button>
        </motion.form>

        <motion.div
          className="landing-info"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ delay: 0.9 }}
        >
          <div className="info-card">
            <h3>Host Capabilities</h3>
            <ul>
              <li>Start and control game phases</li>
              <li>Monitor player progress in real-time</li>
              <li>Access comprehensive analytics</li>
              <li>Manage puzzle timing</li>
            </ul>
          </div>
        </motion.div>
      </motion.div>
    </div>
  );
};

export default HostLanding;
