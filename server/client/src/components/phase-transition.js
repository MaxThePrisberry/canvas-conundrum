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
          <motion.div
            className="celebration-burst"
            initial={{ scale: 0 }}
            animate={{ scale: [0, 1.5, 1.2] }}
            transition={{ duration: 0.8, times: [0, 0.6, 1] }}
          >
            {[...Array(8)].map((_, i) => (
              <motion.div
                key={i}
                className="burst-line"
                style={{ '--rotation': `${i * 45}deg` }}
                initial={{ scaleY: 0 }}
                animate={{ scaleY: [0, 1, 0] }}
                transition={{ duration: 1, delay: 0.2 }}
              />
            ))}
          </motion.div>
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
            {celebration ? '🎉' : '🎯'}
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
          className="transition-dots"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ delay: 1 }}
        >
          {[0, 1, 2].map((i) => (
            <motion.span
              key={i}
              className="dot"
              animate={{ 
                scale: [1, 1.5, 1],
                opacity: [0.3, 1, 0.3]
              }}
              transition={{
                duration: 1.5,
                delay: i * 0.2,
                repeat: Infinity,
                ease: "easeInOut"
              }}
            />
          ))}
        </motion.div>
      </div>

      <motion.div
        className="transition-wave"
        initial={{ x: '-100%' }}
        animate={{ x: '100%' }}
        transition={{ duration: 1.5, ease: "easeInOut" }}
      />
    </motion.div>
  );
};

export default PhaseTransition;