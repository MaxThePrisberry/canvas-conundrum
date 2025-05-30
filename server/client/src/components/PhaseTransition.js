import React from 'react';
import { motion } from 'framer-motion';
import './PhaseTransition.css';

const PhaseTransition = ({ title, subtitle, celebration = false }) => {
  return (
    <motion.div
      className="phase-transition"
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
      transition={{ duration: 0.5 }}
    >
      <div className="transition-content">
        {celebration && (
          <div className="celebration-burst">
            {[...Array(12)].map((_, i) => (
              <div
                key={i}
                className="burst-star"
                style={{ '--rotation': `${i * 30}deg` }}
              />
            ))}
          </div>
        )}

        <motion.div
          className="transition-icon-container"
          initial={{ scale: 0, rotate: -180 }}
          animate={{ scale: 1, rotate: 0 }}
          transition={{ 
            duration: 0.8, 
            type: "spring", 
            stiffness: 200,
            delay: 0.2 
          }}
        >
          <div className="transition-icon">
            {celebration ? '🎉' : '✨'}
          </div>
          <div className="icon-rings">
            <div className="ring ring-1"></div>
            <div className="ring ring-2"></div>
            <div className="ring ring-3"></div>
          </div>
        </motion.div>

        <motion.h1
          className="transition-title"
          initial={{ opacity: 0, y: 30 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.6, delay: 0.5 }}
        >
          {title}
        </motion.h1>

        {subtitle && (
          <motion.p
            className="transition-subtitle"
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.6, delay: 0.7 }}
          >
            {subtitle}
          </motion.p>
        )}

        <motion.div
          className="transition-loader"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ delay: 1 }}
        >
          <div className="loader-dots">
            <span></span>
            <span></span>
            <span></span>
          </div>
        </motion.div>
      </div>

      {/* Floating elements */}
      <div className="floating-elements">
        {[...Array(6)].map((_, i) => (
          <div
            key={i}
            className="floating-shape"
            style={{ '--delay': `${i * 0.5}s` }}
          />
        ))}
      </div>
    </motion.div>
  );
};

export default PhaseTransition;
