import React, { useState, useEffect, useRef } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { Html5QrcodeScanner } from 'html5-qrcode';
import { TokenType, Colors, QR_SCANNER_CONFIG, RESOURCE_STATIONS, ERROR_MESSAGES, SUCCESS_MESSAGES, HAPTIC_PATTERNS } from '../constants';
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
  const [scanError, setScanError] = useState(null);
  const scannerRef = useRef(null);
  const scannerInstanceRef = useRef(null);

  useEffect(() => {
    // Hide transition after animation
    const timer = setTimeout(() => setShowTransition(false), 2000);
    return () => clearTimeout(timer);
  }, []);

  useEffect(() => {
    // Handle question arrival
    if (currentQuestion && currentView === 'waiting') {
      setCurrentView('question');
      setLastAnswerCorrect(null);
    }
  }, [currentQuestion, currentView]);

  const handleResourceSelect = (resourceType) => {
    setSelectedResource(resourceType);
    setCurrentView('scanner');
    setScanError(null);

    // Haptic feedback
    if (window.navigator && window.navigator.vibrate) {
      window.navigator.vibrate(HAPTIC_PATTERNS.MEDIUM);
    }
  };

  const handleScanSuccess = (decodedText, decodedResult) => {
    console.log('QR Code scanned:', decodedText);
    verifyCode(decodedText);
  };

  const handleScanError = (errorMessage) => {
    // Ignore continuous scan errors, they're normal
    if (errorMessage?.includes('NotFoundException')) {
      return;
    }
    console.log('QR scan error:', errorMessage);
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
        scannerInstanceRef.current.clear().catch(console.error);
        scannerInstanceRef.current = null;
      }

      // Verify location
      setVerifiedLocation(resourceType);
      onLocationVerified(code);
      setCurrentView('waiting');
      setShowManualEntry(false);
      setScanError(null);

      // Success haptic feedback
      if (window.navigator && window.navigator.vibrate) {
        window.navigator.vibrate(HAPTIC_PATTERNS.SUCCESS);
      }
    } else {
      // Invalid QR code - show error feedback
      setScanError(ERROR_MESSAGES.INVALID_CODE);
      if (window.navigator && window.navigator.vibrate) {
        window.navigator.vibrate(HAPTIC_PATTERNS.ERROR);
      }
      
      // Show manual entry option after error
      setTimeout(() => {
        setShowManualEntry(true);
      }, 1000);
    }
  };

  const handleBackToMenu = () => {
    // Stop scanner if active
    if (scannerInstanceRef.current) {
      scannerInstanceRef.current.clear().catch(console.error);
      scannerInstanceRef.current = null;
    }
    
    setCurrentView('menu');
    setSelectedResource(null);
    setScanError(null);
    setShowManualEntry(false);
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
      try {
        const scanner = new Html5QrcodeScanner(
          "qr-reader",
          QR_SCANNER_CONFIG,
          false // verbose
        );

        scannerInstanceRef.current = scanner;
        scanner.render(handleScanSuccess, handleScanError);
      } catch (error) {
        console.error('Failed to initialize QR scanner:', error);
        setScanError(ERROR_MESSAGES.QR_SCAN_FAILED);
      }
    }

    return () => {
      if (scannerInstanceRef.current && currentView !== 'scanner') {
        scannerInstanceRef.current.clear().catch(console.error);
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
              {Object.entries(RESOURCE_STATIONS).map(([type, config], index) => (
                <motion.button
                  key={type}
                  className="resource-card"
                  onClick={() => handleResourceSelect(type)}
                  initial={{ opacity: 0, scale: 0.8 }}
                  animate={{ opacity: 1, scale: 1 }}
                  transition={{ delay: index * 0.1 }}
                  whileHover={{ scale: 1.05 }}
                  whileTap={{ scale: 0.95 }}
                  style={{ '--resource-color': Colors.token[type] }}
                >
                  <div className="resource-icon">{config.icon}</div>
                  <h3>{config.name}</h3>
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
                <strong>{RESOURCE_STATIONS[verifiedLocation].name}</strong>
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
              <h2>Scan {RESOURCE_STATIONS[selectedResource]?.name}</h2>
            </div>

            <div className="scanner-container">
              <div id="qr-reader" ref={scannerRef}></div>
              
              <motion.div
                className="scan-frame"
                animate={{ opacity: [0.5, 1, 0.5] }}
                transition={{ duration: 2, repeat: Infinity }}
              >
                <span></span>
                <span></span>
              </motion.div>
            </div>

            {scanError && (
              <motion.p
                className="scanner-error"
                initial={{ opacity: 0, y: -10 }}
                animate={{ opacity: 1, y: 0 }}
                style={{ color: Colors.error, textAlign: 'center', marginTop: '1rem' }}
              >
                {scanError}
              </motion.p>
            )}

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
                style={{ backgroundColor: Colors.token[verifiedLocation] }}
                animate={{ rotate: [0, 10, -10, 0] }}
                transition={{ duration: 2, repeat: Infinity }}
              >
                {RESOURCE_STATIONS[verifiedLocation]?.icon}
              </motion.div>
              
              <motion.div
                className="verified-ring"
                animate={{ scale: [1, 1.2, 1] }}
                transition={{ duration: 2, repeat: Infinity }}
                style={{ borderColor: Colors.token[verifiedLocation] }}
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
                {lastAnswerCorrect ? SUCCESS_MESSAGES.ANSWER_CORRECT : '✗ Incorrect'}
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
            onCancel={() => {
              setShowManualEntry(false);
              setScanError(null);
            }}
          />
        )}
      </AnimatePresence>
    </div>
  );
};

export default ResourceGatheringPhase;
