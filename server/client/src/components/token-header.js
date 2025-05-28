import React, { useEffect, useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { TokenType, TokenThresholds, Colors } from '../constants';
import './TokenHeader.css';

const TokenHeader = ({ tokens }) => {
  const [prevTokens, setPrevTokens] = useState(tokens);
  const [animatingTokens, setAnimatingTokens] = useState({});
  const [thresholdReached, setThresholdReached] = useState({});

  useEffect(() => {
    // Check for token increases
    const newAnimatingTokens = {};
    const newThresholdReached = {};

    Object.entries(tokens).forEach(([key, value]) => {
      const prevValue = prevTokens[key] || 0;
      
      // Check if token count increased
      if (value > prevValue) {
        newAnimatingTokens[key] = true;
        setTimeout(() => {
          setAnimatingTokens(prev => ({ ...prev, [key]: false }));
        }, 600);

        // Haptic feedback on token increase
        if (window.navigator && window.navigator.vibrate) {
          window.navigator.vibrate(30);
        }
      }

      // Check if threshold reached
      const tokenType = key.replace('Tokens', '').toUpperCase();
      const threshold = TokenThresholds[tokenType];
      const prevThresholdLevel = Math.floor(prevValue / threshold);
      const currentThresholdLevel = Math.floor(value / threshold);
      
      if (currentThresholdLevel > prevThresholdLevel) {
        newThresholdReached[key] = true;
        setTimeout(() => {
          setThresholdReached(prev => ({ ...prev, [key]: false }));
        }, 2000);

        // Stronger haptic feedback for threshold
        if (window.navigator && window.navigator.vibrate) {
          window.navigator.vibrate([50, 30, 50]);
        }
      }
    });

    setAnimatingTokens(newAnimatingTokens);
    setThresholdReached(newThresholdReached);
    setPrevTokens(tokens);
  }, [tokens, prevTokens]);

  const getTokenProgress = (tokenKey, tokenValue) => {
    const tokenType = tokenKey.replace('Tokens', '').toUpperCase();
    const threshold = TokenThresholds[tokenType];
    const progress = (tokenValue % threshold) / threshold * 100;
    const level = Math.floor(tokenValue / threshold);
    return { progress, level, threshold };
  };

  const tokenConfig = [
    { key: 'anchorTokens', type: TokenType.ANCHOR, label: 'Anchor', icon: '⚓' },
    { key: 'chronosTokens', type: TokenType.CHRONOS, label: 'Time', icon: '⏰' },
    { key: 'guideTokens', type: TokenType.GUIDE, label: 'Guide', icon: '🧭' },
    { key: 'clarityTokens', type: TokenType.CLARITY, label: 'Clarity', icon: '💎' }
  ];

  return (
    <motion.div 
      className="token-header"
      initial={{ y: -100 }}
      animate={{ y: 0 }}
      transition={{ type: "spring", stiffness: 200 }}
    >
      <div className="token-grid">
        {tokenConfig.map(({ key, type, label, icon }) => {
          const value = tokens[key] || 0;
          const { progress, level, threshold } = getTokenProgress(key, value);
          const isAnimating = animatingTokens[key];
          const hasReachedThreshold = thresholdReached[key];

          return (
            <motion.div
              key={key}
              className={`token-item ${isAnimating ? 'animating' : ''}`}
              animate={isAnimating ? { scale: [1, 1.2, 1] } : {}}
              transition={{ duration: 0.6 }}
            >
              <AnimatePresence>
                {hasReachedThreshold && (
                  <motion.div
                    className="threshold-celebration"
                    initial={{ scale: 0, opacity: 0 }}
                    animate={{ scale: 1, opacity: 1 }}
                    exit={{ scale: 0, opacity: 0 }}
                    transition={{ duration: 0.5 }}
                  >
                    <span>Level {level}!</span>
                  </motion.div>
                )}
              </AnimatePresence>

              <div 
                className="token-icon"
                style={{ backgroundColor: Colors.token[type] }}
              >
                <span>{icon}</span>
              </div>

              <div className="token-info">
                <div className="token-label">{label}</div>
                <motion.div 
                  className="token-value"
                  key={value}
                  initial={isAnimating ? { scale: 1.5, opacity: 0 } : false}
                  animate={{ scale: 1, opacity: 1 }}
                  transition={{ duration: 0.3 }}
                >
                  {value}
                </motion.div>
              </div>

              <div className="token-progress">
                <div className="progress-bar">
                  <motion.div
                    className="progress-fill"
                    style={{ backgroundColor: Colors.token[type] }}
                    animate={{ width: `${progress}%` }}
                    transition={{ duration: 0.5, ease: "easeOut" }}
                  />
                </div>
                <div className="progress-text">
                  Lvl {level}
                </div>
              </div>

              {isAnimating && (
                <motion.div
                  className="token-plus"
                  initial={{ y: 0, opacity: 1 }}
                  animate={{ y: -30, opacity: 0 }}
                  transition={{ duration: 1 }}
                >
                  +{value - (prevTokens[key] || 0)}
                </motion.div>
              )}
            </motion.div>
          );
        })}
      </div>
    </motion.div>
  );
};

export default TokenHeader;