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
  playerSpecialties,
  lobbyStatus
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
      image: '/images/roles/art_enthusiast.png',
      color: Colors.token.clarity
    },
    [RoleType.DETECTIVE]: {
      title: 'Detective',
      description: 'Bonus to Guide tokens',
      icon: '🔍',
      image: '/images/roles/detective.png',
      color: Colors.token.guide
    },
    [RoleType.TOURIST]: {
      title: 'Tourist',
      description: 'Bonus to Time tokens',
      icon: '📸',
      image: '/images/roles/tourist.png',
      color: Colors.token.chronos
    },
    [RoleType.JANITOR]: {
      title: 'Janitor',
      description: 'Bonus to Anchor tokens',
      icon: '🧹',
      image: '/images/roles/janitor.png',
      color: Colors.token.anchor
    }
  };

  const handleRoleSelect = (role) => {
    setSelectedRole(role);
    onRoleSelect(role);
    setCurrentStep('specialties');

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
      
      if (window.navigator && window.navigator.vibrate) {
        window.navigator.vibrate(20);
      }
    }
  };

  const handleSpecialtyConfirm = () => {
    if (selectedSpecialties.length > 0) {
      onSpecialtySelect(selectedSpecialties);
      setIsWaiting(true);

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
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            transition={{ duration: 0.5 }}
            className="setup-content"
          >
            {currentStep === 'role' ? (
              <>
                <h1 className="setup-title slide-down">Choose Your Role</h1>
                <p className="setup-subtitle fade-in">Each role provides unique bonuses</p>

                <div className="role-grid">
                  {availableRoles.map((role, index) => {
                    const info = roleInfo[role.role];
                    return (
                      <button
                        key={role.role}
                        className={`role-card ${!role.available ? 'disabled' : ''}`}
                        onClick={() => role.available && handleRoleSelect(role.role)}
                        disabled={!role.available}
                        style={{
                          '--role-color': info.color,
                          '--animation-delay': `${index * 0.1}s`
                        }}
                      >
                        <div className="role-image-container">
                          <img 
                            src={info.image} 
                            alt={info.title}
                            className="role-image"
                            onError={(e) => {
                              e.target.style.display = 'none';
                              e.target.nextSibling.style.display = 'flex';
                            }}
                          />
                          <div className="role-icon-fallback" style={{ display: 'none' }}>
                            {info.icon}
                          </div>
                        </div>
                        <h3>{info.title}</h3>
                        <p>{info.description}</p>
                        {!role.available && (
                          <div className="role-taken">Taken</div>
                        )}
                      </button>
                    );
                  })}
                </div>
              </>
            ) : (
              <>
                <h1 className="setup-title slide-down">Choose Your Specialties</h1>
                <p className="setup-subtitle fade-in">Select 1-2 trivia categories for bonus points</p>

                <div className="specialty-grid">
                  {triviaCategories.map((category, index) => (
                    <button
                      key={category}
                      className={`specialty-card ${
                        selectedSpecialties.includes(category) ? 'selected' : ''
                      }`}
                      onClick={() => handleSpecialtyToggle(category)}
                      style={{ '--animation-delay': `${index * 0.05}s` }}
                    >
                      <div className="specialty-icon">
                        {category === 'general' && '🌍'}
                        {category === 'geography' && '🗺️'}
                        {category === 'history' && '📚'}
                        {category === 'music' && '🎵'}
                        {category === 'science' && '🔬'}
                        {category === 'video_games' && '🎮'}
                      </div>
                      <span>{category.replace('_', ' ')}</span>
                      {selectedSpecialties.includes(category) && (
                        <div className="specialty-check">✓</div>
                      )}
                    </button>
                  ))}
                </div>

                <div className="selection-info">
                  {selectedSpecialties.length}/2 selected
                </div>

                <button
                  className="btn-primary confirm-button"
                  onClick={handleSpecialtyConfirm}
                  disabled={selectedSpecialties.length === 0}
                >
                  Confirm Selection
                </button>
              </>
            )}
          </motion.div>
        ) : (
          <motion.div
            key="waiting"
            initial={{ opacity: 0, scale: 0.9 }}
            animate={{ opacity: 1, scale: 1 }}
            exit={{ opacity: 0, scale: 0.9 }}
            className="waiting-screen"
          >
            <div className="waiting-animation">
              <div className="waiting-circles">
                <div className="circle circle-1"></div>
                <div className="circle circle-2"></div>
                <div className="circle circle-3"></div>
              </div>
              
              <motion.div
                className="center-role"
                animate={{ rotate: [0, 10, -10, 0] }}
                transition={{ duration: 4, repeat: Infinity }}
              >
                <img 
                  src={roleInfo[selectedRole]?.image} 
                  alt={roleInfo[selectedRole]?.title}
                  className="role-image"
                  onError={(e) => {
                    e.target.style.display = 'none';
                    e.target.nextSibling.style.display = 'flex';
                  }}
                />
                <div className="role-icon-fallback" style={{ display: 'none' }}>
                  {roleInfo[selectedRole]?.icon}
                </div>
              </motion.div>
            </div>

            <h2>Waiting for Game to Start</h2>

            {lobbyStatus && (
              <div className="lobby-info">
                <p>{lobbyStatus.waitingMessage || `${lobbyStatus.currentPlayers} players connected`}</p>
                {lobbyStatus.gameStarting && (
                  <div className="starting-soon">Game starting soon!</div>
                )}
              </div>
            )}

            <div className="player-info card">
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
            </div>

            <div className="waiting-dots">
              <span></span>
              <span></span>
              <span></span>
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
};

export default SetupPhase;
