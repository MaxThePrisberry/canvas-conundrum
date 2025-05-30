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
      
      if (window.navigator && window.navigator.vibrate) {
        window.navigator.vibrate([50, 30, 50]);
      }
    }
  };

  const handleInputChange = (e) => {
    setCode(e.target.value.toUpperCase());
    setError(false);
  };

  return (
    <motion.div
      className="manual-code-entry"
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
      transition={{ duration: 0.3 }}
    >
      <motion.div
        className="manual-entry-content"
        initial={{ scale: 0.9, y: 50 }}
        animate={{ scale: 1, y: 0 }}
        exit={{ scale: 0.9, y: 50 }}
        transition={{ type: "spring", stiffness: 200 }}
      >
        <button
          type="button"
          className="close-button"
          onClick={onCancel}
          aria-label="Close"
        >
          ×
        </button>

        <h3>Enter Station Code</h3>
        <p>Can't scan? Enter the code shown below the QR code</p>
        
        <form onSubmit={handleSubmit}>
          <div className="input-wrapper">
            <input
              type="text"
              value={code}
              onChange={handleInputChange}
              placeholder="Enter code here"
              className={`code-input ${error ? 'error' : ''}`}
              autoFocus
              autoComplete="off"
              autoCorrect="off"
              autoCapitalize="characters"
              maxLength="30"
            />
            {error && (
              <motion.div
                className="error-shake"
                animate={{ x: [-10, 10, -10, 10, 0] }}
                transition={{ duration: 0.5 }}
              />
            )}
          </div>
          
          <div className="manual-entry-buttons">
            <button
              type="button"
              className="btn-secondary"
              onClick={onCancel}
            >
              Cancel
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
          Station codes are typically 20-30 characters long
        </p>
      </motion.div>
    </motion.div>
  );
};

export default ManualCodeEntry;
