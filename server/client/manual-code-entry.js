import React, { useState } from 'react';
import { motion } from 'framer-motion';
import './ManualCodeEntry.css';

const ManualCodeEntry = ({ onCodeSubmit, onCancel }) => {
  const [code, setCode] = useState('');
  const [error, setError] = useState(false);

  const handleSubmit = (e) => {
    e.preventDefault();
    if (code.trim()) {
      onCodeSubmit(code.trim());
    } else {
      setError(true);
      setTimeout(() => setError(false), 500);
      
      // Error haptic feedback
      if (window.navigator && window.navigator.vibrate) {
        window.navigator.vibrate([50, 30, 50]);
      }
    }
  };

  const handleInputChange = (e) => {
    setCode(e.target.value);
    setError(false);
  };

  return (
    <motion.div
      className="manual-code-entry"
      initial={{ opacity: 0, y: 50 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, y: 50 }}
      transition={{ duration: 0.3 }}
    >
      <div className="manual-entry-content">
        <h3>Enter Code Manually</h3>
        <p>Having trouble scanning? Enter the code below the QR code.</p>
        
        <form onSubmit={handleSubmit}>
          <motion.input
            type="text"
            value={code}
            onChange={handleInputChange}
            placeholder="Enter station code"
            className={`code-input ${error ? 'error' : ''}`}
            autoFocus
            autoComplete="off"
            autoCorrect="off"
            autoCapitalize="characters"
            animate={error ? { x: [-10, 10, -10, 10, 0] } : {}}
            transition={{ duration: 0.3 }}
          />
          
          <div className="manual-entry-buttons">
            <button
              type="button"
              className="btn-secondary"
              onClick={onCancel}
            >
              Back to Scanner
            </button>
            <button
              type="submit"
              className="btn-primary"
              disabled={!code.trim()}
            >
              Verify Code
            </button>
          </div>
        </form>
        
        <p className="code-hint">
          The code is usually 20+ characters long
        </p>
      </div>
    </motion.div>
  );
};

export default ManualCodeEntry;