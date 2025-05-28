import React, { useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { RoleType, Colors } from '../constants';
import './SetupPhase.css';

const SetupPhase = ({ 
  availableRoles, 
  triviaCategories, 
  onRoleSelect, 
  onSpecialtySelect,
  playerRole,
  playerSpecialties
}) => {
  const [currentStep, setCurrentStep] = useState(playerRole ? 'specialties' : 'role');
  const [selectedRole, setSelectedRole] = useState(playerRole);
  const [selectedSpecialties, setSelectedSpecialties] = useState(playerSpecialties || []);
  const [isWaiting, setIsWaiting] = useState(playerSpecialties.length > 0);

  const roleInfo = {
    [RoleType.ART_ENTHUSIAST]: {
      title: 'Art Enthusiast',
      description: 'Bonus to Clarity tokens',
      icon: '🎨',
      color: Colors.token.clarity
    },
    [RoleType.DETECTIVE]: {
      title: 'Detective',
      description: 'Bonus to Guide tokens',
      icon: '🔍',
      color: Colors.token.guide
    },
    [RoleType.TOURIST]: {
      title: 'Tourist',
      description: 'Bonus to Time tokens',
      icon: '📸',
      color: Colors.token.chronos
    },
    [RoleType.JANITOR]: {
      title: 'Janitor',
      description: 'Bonus to Anchor tokens',
      icon: '🧹',
      color: Colors.token.anchor
    }
  };

  const handleRoleSelect = (role) => {
    setSelectedRole(role);
    onRoleSelect(role);
    setCurrentStep('specialties');

    // Haptic feedback
    if (window.navigator && window.navigator.vibrate) {
      window.navigator.vibrate(30);
    }
  };

  const handleSpecialtyToggle = (category) => {
    const newSpecialties = selectedSpecialties.includes(category)
      ? selectedSpecialties.filter(s => s !== category)
      : [...selectedSpecialties, category];

    if (newSpecialties.length <= 2) {
      setSelectedSpecialties(newSpecialties);
      
      // Haptic feedback
      if (window.navigator && window.navigator.vibrate) {
        window.navigator.vibrate(20);
      }
    }
  };

  const handleSpecialtyConfirm = () => {
    if (selectedSpecialties.length > 0) {
      onSpecialtySelect(selectedSpecialties);
      setIsWaiting(true);

      // Haptic feedback
      if (window.navigator && window.navigator.vibrate) {
        window.navigator.vibrate(50);
      }
    }
  };

  return (
    <div className="setup-phase">
      <AnimatePresence mode="wait">
        {!isWaiting ? (
          <motion.div
            key={currentStep}
            initial={{ opacity: 0, x: 100 }}
            animate={{ opacity: 1, x: 0 }}
            exit={{ opacity: 0, x: -100 }}
            transition={{ duration: 0.5, ease: "easeInOut" }}
            className="setup-content"
          >
            {currentStep === 'role' ? (
              <>
                <motion.h1
                  initial={{ opacity: 0, y: -20 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{ delay: 0.2 }}
                  className="setup-title"
                >
                  Choose Your Role
                </motion.h1>
                
                <motion.p
                  initial={{ opacity: 0 }}
                  animate={{ opacity: 1 }}
                  transition={{ delay: 0.3 }}
                  className="setup-subtitle"
                >
                  Each role provides unique bonuses
                </motion.p>

                <div className="role-grid">
                  {availableRoles.map((role, index) => {
                    const info = roleInfo[role.role];
                    return (
                      <motion.button
                        key={role.role}
                        className={`role-card ${!role.available ? 'disabled' : ''}`}
                        onClick={() => role.available && handleRoleSelect(role.role)}
                        disabled={!role.available}
                        initial={{ opacity: 0, scale: 0.8 }}
                        animate={{ opacity: 1, scale: 1 }}
                        transition={{ delay: index * 0.1 + 0.4 }}
                        whileHover={role.available ? { scale: 1.05 } : {}}
                        whileTap={role.available ? { scale: 0.95 } : {}}
                        style={{
                          '--role-color': info.color
                        }}
                      >
                        <div className="role-icon">{info.icon}</div>
                        <h3>{info.title}</h3>
                        <p>{info.description}</p>
                        {!role.available && (
                          <div className="role-taken">Taken</div>
                        )}
                      </motion.button>
                    );
                  })}
                </div>
              </>
            ) : (
              <>
                <motion.h1
                  initial={{ opacity: 0, y: -20 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{ delay: 0.2 }}
                  className="setup-title"
                >
                  Choose Your Specialties
                </motion.h1>
                
                <motion.p
                  initial={{ opacity: 0 }}
                  animate={{ opacity: 1 }}
                  transition={{ delay: 0.3 }}
                  className="setup-subtitle"
                >
                  Select 1-2 trivia categories for bonus points
                </motion.p>

                <div className="specialty-grid">
                  {triviaCategories.map((category, index) => (
                    <motion.button
                      key={category}
                      className={`specialty-card ${
                        selectedSpecialties.includes(category) ? 'selected' : ''
                      }`}
                      onClick={() => handleSpecialtyToggle(category)}
                      initial={{ opacity: 0, scale: 0.8 }}
                      animate={{ opacity: 1, scale: 1 }}
                      transition={{ delay: index * 0.05 + 0.4 }}
                      whileHover={{ scale: 1.05 }}
                      whileTap={{ scale: 0.95 }}
                    >
                      <motion.div
                        className="specialty-check"
                        initial={false}
                        animate={selectedSpecialties.includes(category) ? 
                          { scale: 1, opacity: 1 } : 
                          { scale: 0, opacity: 0 }
                        }
                      >
                        ✓
                      </motion.div>
                      <span>{category.replace('_', ' ')}</span>
                    </motion.button>
                  ))}
                </div>

                <motion.div
                  initial={{ opacity: 0 }}
                  animate={{ opacity: 1 }}
                  transition={{ delay: 0.6 }}
                  className="selection-info"
                >
                  {selectedSpecialties.length}/2 selected
                </motion.div>

                <motion.button
                  className="btn-primary confirm-button"
                  onClick={handleSpecialtyConfirm}
                  disabled={selectedSpecialties.length === 0}
                  initial={{ opacity: 0, y: 20 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{ delay: 0.7 }}
                  whileHover={{ scale: 1.05 }}
                  whileTap={{ scale: 0.95 }}
                >
                  Confirm Selection
                </motion.button>
              </>
            )}
          </motion.div>
        ) : (
          <motion.div
            key="waiting"
            initial={{ opacity: 0, scale: 0.8 }}
            animate={{ opacity: 1, scale: 1 }}
            exit={{ opacity: 0, scale: 0.8 }}
            className="waiting-screen"
          >
            <div className="waiting-animation">
              <motion.div
                className="orb-container"
                animate={{ rotate: 360 }}
                transition={{ duration: 20, repeat: Infinity, ease: "linear" }}
              >
                {[0, 1, 2, 3].map((i) => (
                  <motion.div
                    key={i}
                    className="orb"
                    style={{
                      '--orb-color': Object.values(Colors.token)[i],
                      '--orb-delay': `${i * 0.2}s`
                    }}
                    animate={{
                      scale: [1, 1.2, 1],
                      opacity: [0.6, 1, 0.6]
                    }}
                    transition={{
                      duration: 2,
                      delay: i * 0.2,
                      repeat: Infinity,
                      ease: "easeInOut"
                    }}
                  />
                ))}
              </motion.div>
              
              <motion.div
                className="center-icon"
                animate={{ 
                  scale: [1, 1.1, 1],
                  rotate: [0, 5, -5, 0]
                }}
                transition={{
                  duration: 4,
                  repeat: Infinity,
                  ease: "easeInOut"
                }}
              >
                {roleInfo[selectedRole]?.icon}
              </motion.div>
            </div>

            <motion.h2
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 0.5 }}
            >
              Waiting for Game to Start
            </motion.h2>

            <motion.div
              className="player-info"
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              transition={{ delay: 0.7 }}
            >
              <div className="info-item">
                <span className="label">Role:</span>
                <span className="value">{roleInfo[selectedRole]?.title}</span>
              </div>
              <div className="info-item">
                <span className="label">Specialties:</span>
                <span className="value">
                  {selectedSpecialties.map(s => s.replace('_', ' ')).join(', ')}
                </span>
              </div>
            </motion.div>

            <motion.div
              className="waiting-dots"
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              transition={{ delay: 1 }}
            >
              {[0, 1, 2].map((i) => (
                <motion.span
                  key={i}
                  animate={{ opacity: [0.3, 1, 0.3] }}
                  transition={{
                    duration: 1.5,
                    delay: i * 0.2,
                    repeat: Infinity,
                    ease: "easeInOut"
                  }}
                >
                  •
                </motion.span>
              ))}
            </motion.div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
};

export default SetupPhase;