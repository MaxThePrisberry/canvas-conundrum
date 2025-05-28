import React, { useState, useEffect, useRef } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { Html5QrcodeScanner } from 'html5-qrcode';
import { TokenType, Colors } from '../constants';
import PhaseTransition from './PhaseTransition';
import TriviaQuestion from './TriviaQuestion';
import ManualCodeEntry from './ManualCodeEntry';
import './ResourceGatheringPhase.css';

const ResourceGatheringPhase = ({ 
  resourceHashes, 
  currentQuestion, 
  onLocationVerified, 
  onAnswerSubmit 
}) => {
  const [currentView, setCurrentView] = useState('menu'); // menu, scanner, waiting, question
  const [selectedResource, setSelectedResource] = useState(null);
  const [verifiedLocation, setVerifiedLocation] = useState(null);
  const [showTransition, setShowTransition] = useState(true);
  const [lastAnswerCorrect, setLastAnswerCorrect] = useState(null);
  const [showManualEntry, setShowManualEntry] = useState(false);
  const scannerRef = useRef(null);
  const scannerInstanceRef = useRef(null);

  useEffect(() => {
    // Hide transition after animation
    const timer = setTimeout(() => setShowTransition(false), 2000);
    return () => clearTimeout(timer);
  }, []);

  useEffect(() => {
    // Handle question arrival
    if (currentQuestion) {
      setCurrentView('question');
      setLastAnswerCorrect(null);
    }
  }, [currentQuestion]);

  const resourceConfig = {
    [TokenType.ANCHOR]: {
      title: 'Anchor Station',
      icon: '⚓',
      color: Colors.token.anchor,
      description: 'Stability tokens'
    },
    [TokenType.CHRONOS]: {
      title: 'Time Station',
      icon: '⏰',
      color: Colors.token.chronos,
      description: 'Time extension tokens'
    },
    [TokenType.GUIDE]: {
      title: 'Guide Station',
      icon: '🧭',
      color: Colors.token.guide,
      description: 'Hint tokens'
    },
    [TokenType.CLARITY]: {
      title: 'Clarity Station',
      icon: '💎',
      color: Colors.token.clarity,
      description: 'Preview time tokens'
    }
  };

  const handleResourceSelect = (resourceType) => {
    setSelectedResource(resourceType);
    setCurrentView('scanner');

    // Haptic feedback
    if (window.navigator && window.navigator.vibrate) {
      window.navigator.vibrate(30);
    }
  };

  const handleScanSuccess = (decodedText) => {
    verifyCode(decodedText);
  };

  const verifyCode = (code) => {
    // Check if scanned QR matches any resource hash
    const matchedResource = Object.entries(resourceHashes).find(
      ([_, hash]) => hash === code
    );

    if (matchedResource) {
      const [resourceType] = matchedResource;
      
      // Stop scanner if active
      if (scannerInstanceRef.current) {
        scannerInstanceRef.current.clear();
        scannerInstanceRef.current = null;
      }

      // Verify location
      setVerifiedLocation(resourceType);
      onLocationVerified(code);
      setCurrentView('waiting');
      setShowManualEntry(false);

      // Success haptic feedback
      if (window.navigator && window.navigator.vibrate) {
        window.navigator.vibrate([50, 30, 50]);
      }
    } else {
      // Invalid QR code - show error feedback
      if (window.navigator && window.navigator.vibrate) {
        window.navigator.vibrate([100, 50, 100]);
      }
      
      // Show error message or manual entry option
      setShowManualEntry(true);
    }
  };

  const handleBackToMenu = () => {
    // Stop scanner if active
    if (scannerInstanceRef.current) {
      scannerInstanceRef.current.clear();
      scannerInstanceRef.current = null;
    }
    
    setCurrentView('menu');
    setSelectedResource(null);
  };

  const handleAnswerSubmit = (answer, isCorrect) => {
    onAnswerSubmit(currentQuestion.questionId, answer);
    setLastAnswerCorrect(isCorrect);
    
    // After showing result, go back to waiting
    setTimeout(() => {
      setCurrentView('waiting');
    }, 2000);
  };

  // Initialize QR scanner when scanner view is active
  useEffect(() => {
    if (currentView === 'scanner' && scannerRef.current && !scannerInstanceRef.current) {
      const scanner = new Html5QrcodeScanner("qr-reader", {
        fps: 10,
        qrbox: { width: 250, height: 250 },
        aspectRatio: 1.0,
        showTorchButtonIfSupported: true,
        showZoomSliderIfSupported: true,
        defaultZoomValueIfSupported: 1.5
      });

      scannerInstanceRef.current = scanner;

      scanner.render(handleScanSuccess, (error) => {
        // Ignore errors - scanner is still active
      });
    }

    return () => {
      if (scannerInstanceRef.current) {
        scannerInstanceRef.current.clear();
        scannerInstanceRef.current = null;
      }
    };
  }, [currentView]);

  return (
    <div className="resource-gathering-phase">
      <AnimatePresence>
        {showTransition && (
          <PhaseTransition 
            title="Resource Gathering"
            subtitle="Collect tokens by answering trivia at QR stations"
          />
        )}
      </AnimatePresence>

      <AnimatePresence mode="wait">
        {currentView === 'menu' && (
          <motion.div
            key="menu"
            className="resource-menu"
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            transition={{ duration: 0.5 }}
          >
            <h2 className="menu-title">Select a Resource Station</h2>
            <p className="menu-subtitle">Scan the QR code at your chosen location</p>

            <div className="resource-grid">
              {Object.entries(resourceConfig).map(([type, config], index) => (
                <motion.button
                  key={type}
                  className="resource-card"
                  onClick={() => handleResourceSelect(type)}
                  initial={{ opacity: 0, scale: 0.8 }}
                  animate={{ opacity: 1, scale: 1 }}
                  transition={{ delay: index * 0.1 }}
                  whileHover={{ scale: 1.05 }}
                  whileTap={{ scale: 0.95 }}
                  style={{ '--resource-color': config.color }}
                >
                  <div className="resource-icon">{config.icon}</div>
                  <h3>{config.title}</h3>
                  <p>{config.description}</p>
                </motion.button>
              ))}
            </div>

            {verifiedLocation && (
              <motion.div
                className="current-location"
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
              >
                <span>Current Location:</span>
                <strong>{resourceConfig[verifiedLocation].title}</strong>
              </motion.div>
            )}
          </motion.div>
        )}

        {currentView === 'scanner' && (
          <motion.div
            key="scanner"
            className="scanner-view"
            initial={{ opacity: 0, scale: 0.9 }}
            animate={{ opacity: 1, scale: 1 }}
            exit={{ opacity: 0, scale: 0.9 }}
            transition={{ duration: 0.3 }}
          >
            <div className="scanner-header">
              <button className="back-button" onClick={handleBackToMenu}>
                ← Back
              </button>
              <h2>Scan {resourceConfig[selectedResource].title}</h2>
            </div>

            <div className="scanner-container">
              <div id="qr-reader" ref={scannerRef}></div>
              
              <motion.div
                className="scan-frame"
                animate={{ opacity: [0.5, 1, 0.5] }}
                transition={{ duration: 2, repeat: Infinity }}
              />
            </div>

            <p className="scanner-hint">
              Point your camera at the QR code
            </p>
            
            <button 
              className="btn-secondary manual-entry-button"
              onClick={() => setShowManualEntry(true)}
            >
              Enter Code Manually
            </button>
          </motion.div>
        )}

        {currentView === 'waiting' && (
          <motion.div
            key="waiting"
            className="waiting-view"
            initial={{ opacity: 0, scale: 0.8 }}
            animate={{ opacity: 1, scale: 1 }}
            exit={{ opacity: 0, scale: 0.8 }}
            transition={{ duration: 0.5 }}
          >
            <motion.div
              className="verified-animation"
              initial={{ scale: 0 }}
              animate={{ scale: 1 }}
              transition={{ type: "spring", stiffness: 200 }}
            >
              <motion.div
                className="verified-icon"
                style={{ backgroundColor: resourceConfig[verifiedLocation]?.color }}
                animate={{ rotate: [0, 10, -10, 0] }}
                transition={{ duration: 2, repeat: Infinity }}
              >
                {resourceConfig[verifiedLocation]?.icon}
              </motion.div>
              
              <motion.div
                className="verified-ring"
                animate={{ scale: [1, 1.2, 1] }}
                transition={{ duration: 2, repeat: Infinity }}
                style={{ borderColor: resourceConfig[verifiedLocation]?.color }}
              />
            </motion.div>

            <h2>Location Verified!</h2>
            <p>Waiting for trivia question...</p>

            {lastAnswerCorrect !== null && (
              <motion.div
                className={`answer-result ${lastAnswerCorrect ? 'correct' : 'incorrect'}`}
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0, y: -20 }}
              >
                {lastAnswerCorrect ? '✓ Correct! +10 tokens' : '✗ Incorrect'}
              </motion.div>
            )}

            <button 
              className="btn-secondary change-location"
              onClick={handleBackToMenu}
            >
              Change Location
            </button>
          </motion.div>
        )}

        {currentView === 'question' && currentQuestion && (
          <TriviaQuestion
            key={currentQuestion.questionId}
            question={currentQuestion}
            onAnswer={handleAnswerSubmit}
          />
        )}
      </AnimatePresence>
      
      <AnimatePresence>
        {showManualEntry && (
          <ManualCodeEntry
            onCodeSubmit={verifyCode}
            onCancel={() => setShowManualEntry(false)}
          />
        )}
      </AnimatePresence>
    </div>
  );
};

export default ResourceGatheringPhase;