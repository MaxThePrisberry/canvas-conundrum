import React, { useState, useEffect } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import './TriviaQuestion.css';

const TriviaQuestion = ({ question, onAnswer }) => {
  const [selectedAnswer, setSelectedAnswer] = useState(null);
  const [showResult, setShowResult] = useState(false);
  const [timeRemaining, setTimeRemaining] = useState(question.timeLimit);
  const [isCorrect, setIsCorrect] = useState(false);

  useEffect(() => {
    const timer = setInterval(() => {
      setTimeRemaining(prev => {
        if (prev <= 1) {
          handleTimeout();
          return 0;
        }
        return prev - 1;
      });
    }, 1000);

    return () => clearInterval(timer);
  }, []);

  const handleTimeout = () => {
    if (!selectedAnswer && !showResult) {
      setShowResult(true);
      onAnswer(null, false);
    }
  };

  const handleAnswerSelect = (answer) => {
    if (showResult) return;

    setSelectedAnswer(answer);
    setShowResult(true);
    
    // Check if answer is correct (this would normally come from server)
    const correct = answer === question.options[0]; // Assuming first option is correct
    setIsCorrect(correct);
    
    // Haptic feedback
    if (window.navigator && window.navigator.vibrate) {
      window.navigator.vibrate(correct ? [50, 30, 50] : [100, 50, 100]);
    }

    // Play sound effect
    if (correct) {
      playSuccessSound();
    }

    onAnswer(answer, correct);
  };

  const playSuccessSound = () => {
    // Create a simple success chime using Web Audio API
    const audioContext = new (window.AudioContext || window.webkitAudioContext)();
    const oscillator = audioContext.createOscillator();
    const gainNode = audioContext.createGain();

    oscillator.connect(gainNode);
    gainNode.connect(audioContext.destination);

    oscillator.frequency.setValueAtTime(523.25, audioContext.currentTime); // C5
    oscillator.frequency.setValueAtTime(659.25, audioContext.currentTime + 0.1); // E5
    oscillator.frequency.setValueAtTime(783.99, audioContext.currentTime + 0.2); // G5

    gainNode.gain.setValueAtTime(0.3, audioContext.currentTime);
    gainNode.gain.exponentialRampToValueAtTime(0.01, audioContext.currentTime + 0.5);

    oscillator.start(audioContext.currentTime);
    oscillator.stop(audioContext.currentTime + 0.5);
  };

  const getTimeColor = () => {
    if (timeRemaining > 20) return '#10B981';
    if (timeRemaining > 10) return '#F59E0B';
    return '#EF4444';
  };

  const getTimerRadius = () => {
    const radius = 45;
    const circumference = 2 * Math.PI * radius;
    const progress = timeRemaining / question.timeLimit;
    return circumference * (1 - progress);
  };

  return (
    <motion.div
      className="trivia-question"
      initial={{ opacity: 0, scale: 0.9 }}
      animate={{ opacity: 1, scale: 1 }}
      exit={{ opacity: 0, scale: 0.9 }}
      transition={{ duration: 0.5 }}
    >
      <div className="question-header">
        <motion.div
          className="timer-container"
          animate={timeRemaining <= 5 ? { scale: [1, 1.1, 1] } : {}}
          transition={{ duration: 1, repeat: timeRemaining <= 5 ? Infinity : 0 }}
        >
          <svg className="timer-svg" viewBox="0 0 100 100">
            <circle
              className="timer-background"
              cx="50"
              cy="50"
              r="45"
              fill="none"
              stroke="#F0FDFA"
              strokeWidth="8"
            />
            <motion.circle
              className="timer-progress"
              cx="50"
              cy="50"
              r="45"
              fill="none"
              stroke={getTimeColor()}
              strokeWidth="8"
              strokeLinecap="round"
              strokeDasharray={`${2 * Math.PI * 45}`}
              strokeDashoffset={getTimerRadius()}
              transform="rotate(-90 50 50)"
              transition={{ duration: 1, ease: "linear" }}
            />
          </svg>
          <div className="timer-text" style={{ color: getTimeColor() }}>
            {timeRemaining}
          </div>
        </motion.div>

        <div className="category-badge">
          {question.category.replace('_', ' ')}
        </div>
      </div>

      <motion.h2
        className="question-text"
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ delay: 0.2 }}
      >
        {question.text}
      </motion.h2>

      <div className="options-grid">
        {question.options.map((option, index) => (
          <motion.button
            key={option}
            className={`option-button ${
              showResult && selectedAnswer === option
                ? isCorrect ? 'correct' : 'incorrect'
                : ''
            } ${showResult && !selectedAnswer ? 'disabled' : ''}`}
            onClick={() => handleAnswerSelect(option)}
            disabled={showResult}
            initial={{ opacity: 0, x: -20 }}
            animate={{ opacity: 1, x: 0 }}
            transition={{ delay: 0.3 + index * 0.1 }}
            whileHover={!showResult ? { scale: 1.03 } : {}}
            whileTap={!showResult ? { scale: 0.97 } : {}}
          >
            <span className="option-letter">
              {String.fromCharCode(65 + index)}
            </span>
            <span className="option-text">{option}</span>
            
            {showResult && selectedAnswer === option && (
              <motion.div
                className="option-result"
                initial={{ scale: 0 }}
                animate={{ scale: 1 }}
                transition={{ type: "spring", stiffness: 300 }}
              >
                {isCorrect ? '✓' : '✗'}
              </motion.div>
            )}

            {showResult && selectedAnswer === option && (
              <AnimatePresence>
                <motion.div
                  className="spinner-container"
                  initial={{ opacity: 0 }}
                  animate={{ opacity: 1 }}
                  exit={{ opacity: 0 }}
                >
                  <div className="spinner" />
                </motion.div>
              </AnimatePresence>
            )}
          </motion.button>
        ))}
      </div>

      <AnimatePresence>
        {showResult && (
          <motion.div
            className={`result-message ${isCorrect ? 'success' : 'failure'}`}
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            transition={{ delay: 0.5 }}
          >
            {isCorrect ? (
              <>
                <span className="result-icon">🎉</span>
                <span>Correct! +10 tokens earned</span>
              </>
            ) : (
              <>
                <span className="result-icon">💭</span>
                <span>Not quite! Keep trying</span>
              </>
            )}
          </motion.div>
        )}
      </AnimatePresence>
    </motion.div>
  );
};

export default TriviaQuestion;