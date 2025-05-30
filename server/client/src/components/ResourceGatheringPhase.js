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
  onAnswerSubmit,
  teamTokens,
  questionsAnswered,
  totalQuestions
}) => {
  const [currentView, setCurrentView] = useState('menu');
  const [selectedResource, setSelectedResource] = useState(null);
  const [verifiedLocation, setVerifiedLocation] = useState(null);
  const [showTransition, setShowTransition] = useState(true);
  const [lastAnswerCorrect, setLastAnswerCorrect] = useState(null);
  const [showManualEntry, setShowManualEntry] = useState(false);
  const [scanError, setScanError] = useState(null);
  const scannerRef = useRef(null);
  const scannerInstanceRef = useRef(null);

  useEffect(() => {
    const timer = setTimeout(() => setShowTransition(false), 2000);
    return () => clearTimeout(timer);
  }, []);

  useEffect(() => {
    if (currentQuestion && currentView === 'waiting') {
      setCurrentView('question');
      setLastAnswerCorrect(null);
    }
  }, [currentQuestion, currentView]);

  const handleResourceSelect = (resourceType) => {
    setSelectedResource(resourceType);
    setCurrentView('scanner');
    setScanError(null);

    if (window.navigator && window.navigator.vibrate) {
      window.navigator.vibrate(HAPTIC_PATTERNS.MEDIUM);
    }
  };

  const handleScanSuccess = (decodedText, decodedResult) => {
    console.log('QR Code scanned:', decodedText);
    verifyCode(decodedText);
  };

  const handleScanError = (errorMessage) => {
    if (errorMessage?.includes('NotFoundException')) {
      return;
    }
    console.log('QR scan error:', errorMessage);
  };

  const verifyCode = (code) => {
    const matchedResource = Object.entries(resourceHashes).find(
      ([_, hash]) => hash === code
    );

    if (matchedResource) {
      const [resourceType] = matchedResource;
      
      if (scannerInstanceRef.current) {
        scannerInstanceRef.current.clear().catch(console.error);
        scannerInstanceRef.current = null;
      }

      setVerifiedLocation(resourceType);
      onLocationVerified(code);
      setCurrentView('waiting');
      setShowManualEntry(false);
      setScanError(null);

      if (window.navigator && window.navigator.vibrate) {
        window.navigator.vibrate(HAPTIC_PATTERNS.SUCCESS);
      }
    } else {
      setScanError(ERROR_MESSAGES.INVALID_CODE);
      if (window.navigator && window.navigator.vibrate) {
        window.navigator.vibrate(HAPTIC_PATTERNS.ERROR);
      }
      
      setTimeout(() => {
        setShowManualEntry(true);
      }, 1000);
    }
  };

  const handleBackToMenu = () => {
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
    
    setTimeout(() => {
      setCurrentView('waiting');
    }, 2000);
  };

  useEffect(() => {
    if (currentView === 'scanner' && scannerRef.current && !scannerInstanceRef.current) {
      try {
        const scanner = new Html5QrcodeScanner(
          "qr-reader",
          QR_SCANNER_CONFIG,
          false
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
            <div className="menu-header">
              <h2 className="menu-title">Select a Station</h2>
              <p className="menu-subtitle">Move to a physical location and scan its QR code</p>
              
              {questionsAnswered !== undefined && (
                <div className="progress-info">
                  <span>Round {questionsAnswered + 1} of {totalQuestions}</span>
                </div>
              )}
            </div>

            <div className="resource-grid">
              {Object.entries(RESOURCE_STATIONS).map(([type, config], index) => (
                <motion.button
                  key={type}
                  className="resource-card"
                  onClick={() => handleResourceSelect(type)}
                  initial={{ opacity: 0, y: 30 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{ delay: index * 0.1 }}
                  whileHover={{ y: -8 }}
                  whileTap={{ scale: 0.95 }}
                  style={{ '--resource-color': config.color }}
                >
                  <div className="resource-glow"></div>
                  <div className="resource-content">
                    <div className="resource-icon-container">
                      <img 
                        src={`/images/tokens/${type}.png`} 
                        alt={config.name}
                        className="resource-image"
                        onError={(e) => {
                          e.target.style.display = 'none';
                          e.target.nextSibling.style.display = 'flex';
                        }}
                      />
                      <div className="resource-icon-fallback" style={{ display: 'none' }}>
                        {config.icon}
                      </div>
                    </div>
                    <h3>{config.name}</h3>
                    <p>{config.description}</p>
                    
                    <div className="token-count">
                      <span className="count-value">{teamTokens[type + 'Tokens'] || 0}</span>
                      <span className="count-label">collected</span>
                    </div>
                  </div>
                </motion.button>
              ))}
            </div>

            {verifiedLocation && (
              <motion.div
                className="current-location card"
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
              >
                <span className="location-icon">📍</span>
                <span>Current Location: <strong>{RESOURCE_STATIONS[verifiedLocation].name}</strong></span>
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
              
              <div className="scan-frame">
                <div className="corner corner-tl"></div>
                <div className="corner corner-tr"></div>
                <div className="corner corner-bl"></div>
                <div className="corner corner-br"></div>
                <div className="scan-line"></div>
              </div>
            </div>

            {scanError && (
              <motion.p
                className="scanner-error"
                initial={{ opacity: 0, y: -10 }}
                animate={{ opacity: 1, y: 0 }}
              >
                {scanError}
              </motion.p>
            )}

            <p className="scanner-hint">
              Align the QR code within the frame
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
            <div className="verified-animation">
              <div className="verified-icon" style={{ backgroundColor: RESOURCE_STATIONS[verifiedLocation]?.color }}>
                <img 
                  src={`/images/tokens/${verifiedLocation}.png`} 
                  alt={RESOURCE_STATIONS[verifiedLocation]?.name}
                  className="token-image"
                  onError={(e) => {
                    e.target.style.display = 'none';
                    e.target.nextSibling.style.display = 'block';
                  }}
                />
                <span style={{ display: 'none' }}>{RESOURCE_STATIONS[verifiedLocation]?.icon}</span>
              </div>
              
              <div className="verified-rings">
                <div className="ring ring-1"></div>
                <div className="ring ring-2"></div>
                <div className="ring ring-3"></div>
              </div>
            </div>

            <h2>Location Verified!</h2>
            <p>Get ready for your trivia question...</p>

            {lastAnswerCorrect !== null && (
              <motion.div
                className={`answer-result ${lastAnswerCorrect ? 'correct' : 'incorrect'}`}
                initial={{ opacity: 0, scale: 0.8 }}
                animate={{ opacity: 1, scale: 1 }}
                exit={{ opacity: 0, scale: 0.8 }}
                transition={{ type: "spring", stiffness: 200 }}
              >
                {lastAnswerCorrect ? '✓ Correct! Tokens earned' : '✗ Incorrect - Keep trying!'}
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
