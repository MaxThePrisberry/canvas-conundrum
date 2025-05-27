# Canvas Conundrum - WebSocket Communication Specification

## Connection Lifecycle

### Initial Connection
- Establish secure WebSocket connection
- Server generates unique player identifier
- Client authentication via generated UUID
- Game session initialization

## Authentication Format
All client-to-server events after initial connection use wrapper structure:
```json
{
  "auth": {
    "playerId": "uuid-generated-by-server"
  },
  "payload": {
    // Event-specific data
  }
}
```

## Game Phases and Events

### 1. Setup Phase

#### Client to Server Events
- `player_join`:
  - Initial connection request (no auth wrapper needed)

#### Server to Client Events
- `available_roles`:
  ```json
  {
    "playerId": "uuid-generated-by-server",
    "roles": [
      {
        "role": "art_enthusiast",
        "resourceBonus": 1.5,
        "available": true
      },
      {
        "role": "detective",
        "resourceBonus": 1.5,
        "available": false
      }
    ],
    "triviaCategories": ["general", "geography", "history", "music", "science", "video_games"]
  }
  ```

- `role_selection`:
  ```json
  {
    "auth": {
      "playerId": "uuid-generated-by-server"
    },
    "payload": {
      "role": "art_enthusiast"
    }
  }
  ```

- `trivia_specialty_selection`:
  ```json
  {
    "auth": {
      "playerId": "uuid-generated-by-server"
    },
    "payload": {
      "specialties": ["science", "history"]
    }
  }
  ```

- `game_lobby_status`:
  ```json
  {
    "currentPlayers": 6,
    "playerRoles": {
      "art_enthusiast": 2,
      "detective": 1,
      "tourist": 2,
      "janitor": 1
    },
    "gameStarting": false,
    "waitingMessage": "Waiting for more players..."
  }
  ```

- `countdown`:
  ```json
  {
    "seconds": 25,
    "message": "Game starting in 25 seconds...",
    "canAbort": true
  }
  ```

### 2. Resource Gathering Phase

#### Server to Client Events (Phase Start)
- `resource_phase_start`:
  ```json
  {
    "resourceHashes": {
      "anchor": "HASH_ANCHOR_STATION_2025",
      "chronos": "HASH_CHRONOS_STATION_2025",
      "guide": "HASH_GUIDE_STATION_2025",
      "clarity": "HASH_CLARITY_STATION_2025"
    }
  }
  ```

- `trivia_question`:
  ```json
  {
    "questionId": "general_medium_42",
    "text": "What is the capital of France?",
    "category": "geography",
    "difficulty": "medium",
    "timeLimit": 30,
    "options": ["Paris", "London", "Berlin", "Madrid"],
    "isSpecialty": false
  }
  ```

  **Specialty Question Example:**
  ```json
  {
    "questionId": "science_hard_15",
    "text": "What is the speed of light in vacuum?",
    "category": "science (Specialty)",
    "difficulty": "hard",
    "timeLimit": 45,
    "options": ["299,792,458 m/s", "300,000,000 m/s", "186,000 mi/s", "3.0 × 10^8 m/s"],
    "isSpecialty": true
  }
  ```

- `team_progress_update`:
  ```json
  {
    "questionsAnswered": 28,
    "totalQuestions": 40,
    "teamTokens": {
      "anchorTokens": 45,
      "chronosTokens": 32,
      "guideTokens": 28,
      "clarityTokens": 38
    }
  }
  ```

#### Client to Server Events
- `resource_location_verified`:
  ```json
  {
    "auth": {
      "playerId": "uuid-generated-by-server"
    },
    "payload": {
      "verifiedHash": "HASH_ANCHOR_STATION_2025"
    }
  }
  ```

  **Note**: Clients only need to send this event when changing locations. If staying at the same location between rounds, no reverification is required.

- `trivia_answer`:
  ```json
  {
    "auth": {
      "playerId": "uuid-generated-by-server"
    },
    "payload": {
      "questionId": "general_medium_42",
      "answer": "Paris",
      "timestamp": 1234567890
    }
  }
  ```

### 3. Puzzle Assembly Phase

#### Server to Client Events (Phase Start)

- `image_preview`:
  ```json
  {
    "imageId": "masterpiece_001",
    "duration": 3
  }
  ```

- `puzzle_phase_load`:
  ```json
  {
    "imageId": "masterpiece_001",
    "segmentId": "segment_a5",
    "gridSize": 4,
    "preSolved": false
  }
  ```

- `puzzle_phase_start`:
  ```json
  {
    "startTimestamp": 1234567890,
    "totalTime": 340
  }
  ```

#### Grid Configuration Management
- Dynamic grid sizing algorithm:
  ```
  Grid Size = Based on player count breakpoints
  1-9 players: 3x3, 10-16 players: 4x4, etc.
  ```

#### Fragment Movement Protocol
- Movement Cooldown: 1000ms
- Prevents race conditions
- Ignores rapid successive move requests

#### Client to Server Events
- `segment_completed`:
  ```json
  {
    "auth": {
      "playerId": "uuid-generated-by-server"
    },
    "payload": {
      "segmentId": "segment_a5",
      "completionTimestamp": 1234567890
    }
  }
  ```

- `fragment_move_request`:
  ```json
  {
    "auth": {
      "playerId": "uuid-generated-by-server"
    },
    "payload": {
      "fragmentId": "fragment_player-uuid",
      "newPosition": {"x": 2, "y": 1},
      "timestamp": 1234567890
    }
  }
  ```

- `piece_recommendation_request`:
  ```json
  {
    "auth": {
      "playerId": "uuid-generated-by-server"
    },
    "payload": {
      "toPlayerId": "target-player-uuid",
      "fromFragmentId": "fragment_sender-uuid",
      "toFragmentId": "fragment_target-uuid",
      "suggestedFromPos": {"x": 1, "y": 2},
      "suggestedToPos": {"x": 3, "y": 0},
      "message": "I think your piece goes in the top right corner"
    }
  }
  ```

- `piece_recommendation_response`:
  ```json
  {
    "auth": {
      "playerId": "uuid-generated-by-server"
    },
    "payload": {
      "recommendationId": "recommendation-uuid",
      "accepted": true
    }
  }
  ```

#### Server to Client Events
- `segment_completion_ack`:
  ```json
  {
    "status": "acknowledged",
    "segmentId": "segment_a5",
    "gridPosition": {"x": 2, "y": 3}
  }
  ```

- `fragment_move_response`:
  ```json
  {
    "status": "success",
    "fragment": {
      "id": "fragment_player-uuid",
      "playerId": "player-uuid",
      "position": {"x": 2, "y": 1},
      "solved": true,
      "correctPosition": {"x": 2, "y": 1},
      "preSolved": false
    },
    "nextMoveAvailable": 1234567891
  }
  ```

- `piece_recommendation`:
  ```json
  {
    "id": "recommendation-uuid",
    "fromPlayerId": "sender-uuid",
    "toPlayerId": "receiver-uuid",
    "fromFragmentId": "fragment_sender-uuid",
    "toFragmentId": "fragment_receiver-uuid",
    "suggestedFromPos": {"x": 1, "y": 2},
    "suggestedToPos": {"x": 3, "y": 0},
    "message": "I think your piece goes in the top right corner",
    "timestamp": "2025-05-26T10:30:00Z"
  }
  ```

  **Guide Hints (Guide Token Effect):**
  ```json
  {
    "type": "guide_hint",
    "hints": [
      "Exact position: (2, 3)",
      "Column is correct!",
      "Row needs adjustment"
    ]
  }
  ```

- `central_puzzle_state`:
  ```json
  {
    "fragments": [
      {
        "id": "fragment_player1-uuid",
        "playerId": "player1-uuid",
        "position": {"x": 0, "y": 0},
        "solved": true,
        "correctPosition": {"x": 0, "y": 0},
        "preSolved": false
      }
    ],
    "gridSize": 4,
    "playerDisconnected": "disconnected-player-uuid"
  }
  ```

#### Disconnection Handling
- Immediate fragment auto-solve
- Random grid placement
- Broadcast disconnection to all players

### 4. Post-Game Events

#### Server to Client Events
- `game_analytics`:
  ```json
  {
    "personalAnalytics": [
      {
        "playerId": "uuid",
        "playerName": "Player1",
        "tokenCollection": {
          "anchor": 12,
          "chronos": 8,
          "guide": 15,
          "clarity": 10
        },
        "triviaPerformance": {
          "totalQuestions": 20,
          "correctAnswers": 16,
          "accuracyByCategory": {
            "general": 0.85,
            "science": 0.90
          },
          "specialtyBonus": 40,
          "specialtyCorrect": 4,
          "specialtyTotal": 5
        },
        "puzzleSolvingMetrics": {
          "fragmentSolveTime": 180,
          "movesContributed": 8,
          "successfulMoves": 7,
          "recommendationsSent": 3,
          "recommendationsReceived": 2,
          "recommendationsAccepted": 1
        }
      }
    ],
    "teamAnalytics": {
      "overallPerformance": {
        "totalTime": 1200,
        "completionRate": 1.0,
        "totalScore": 2500
      },
      "collaborationScores": {
        "averageResponseTime": 12.5,
        "communicationScore": 0.85,
        "coordinationScore": 0.80,
        "totalRecommendations": 15,
        "acceptedRecommendations": 8
      },
      "resourceEfficiency": {
        "tokensPerRound": 25.6,
        "tokenDistribution": {
          "anchor": 45.0,
          "chronos": 32.0,
          "guide": 28.0,
          "clarity": 38.0
        },
        "thresholdsReached": {
          "anchor": 2,
          "chronos": 1,
          "guide": 1,
          "clarity": 2
        }
      }
    },
    "globalLeaderboard": [
      {
        "playerId": "uuid",
        "playerName": "Player1",
        "totalScore": 1850,
        "rank": 1
      }
    ],
    "gameSuccess": true
  }
  ```

- `game_reset`:
  ```json
  {
    "message": "Game resetting. Please rejoin to start a new game.",
    "reconnectRequired": true
  }
  ```

## Host-Specific Events

- `host_update`:
  ```json
  {
    "phase": "puzzle_assembly",
    "connectedPlayers": 8,
    "readyPlayers": 8,
    "currentRound": 3,
    "timeRemaining": 120,
    "teamTokens": {
      "anchorTokens": 45,
      "chronosTokens": 32,
      "guideTokens": 28,
      "clarityTokens": 38
    },
    "playerStatuses": {
      "player1-uuid": {
        "name": "Alice",
        "role": "detective",
        "connected": true,
        "ready": true,
        "location": "HASH_GUIDE_STATION_2025"
      }
    },
    "puzzleProgress": 0.625
  }
  ```

## Token Effect Implementations

### Anchor Tokens
- **Effect**: Pre-solve puzzle pieces to reduce individual workload
- **Calculation**: `thresholds = tokens / (5 * difficultyModifier)`
- **Maximum**: Up to 12 pieces pre-solved (leaving minimum 4 to solve)

### Chronos Tokens
- **Effect**: Extend puzzle assembly time
- **Calculation**: `bonus = (tokens / 5) * 20 seconds * difficultyModifier`
- **Applied**: Added to base 300-second puzzle timer

### Guide Tokens
- **Effect**: Provide piece placement hints after segment completion
- **Levels**:
  - Level 1: General positioning feedback
  - Level 2: Exact row or column hints
  - Level 3: Exact coordinates
- **Sent via**: `piece_recommendation` event with `type: "guide_hint"`

### Clarity Tokens
- **Effect**: Show complete puzzle image preview at start of puzzle phase
- **Duration**: `seconds = (tokens / 5) * 1 second * difficultyModifier`
- **Sent via**: `image_preview` event before puzzle phase starts

## Difficulty Scaling

### Easy Mode (0.7x / 1.3x / 0.8x)
- Easier trivia questions
- 30% more time for all phases
- 20% fewer tokens needed for thresholds
- 20% specialty question chance

### Medium Mode (1.0x / 1.0x / 1.0x)
- Normal difficulty baseline
- Standard time limits
- Standard token requirements
- 30% specialty question chance

### Hard Mode (1.4x / 0.7x / 1.3x)
- Harder trivia questions
- 30% less time for all phases
- 30% more tokens needed for thresholds
- 40% specialty question chance

## Error Handling and Edge Cases
- Robust error detection with specific error messages
- Graceful degradation when features unavailable
- Comprehensive logging for debugging
- Invalid authentication handled with clear messages
- Question pool exhaustion handled with reset mechanism

## Performance Considerations
- Efficient data serialization with minimal payload sizes
- Rate limiting on fragment movements (1000ms cooldown)
- Connection state management with ping/pong heartbeats
- Batched progress updates every 5 seconds during puzzle phase

## Security Measures
- Secure WebSocket connections in production
- UUID-based player authentication for all events
- Input validation on all client messages
- Anti-cheat mechanisms for trivia answers and fragment movements
- Host privilege verification for administrative actions

## Technical Specifications
- Dynamic Grid Scaling: 3×3 to 8×8 based on player count
- Fragment Movement Cooldown: 1000ms consistently applied
- Question Pool Management: Auto-reset when exhausted
- Specialty Question Handling: Increased difficulty and time limits
- Collaboration Tracking: Full recommendation system with accept/reject

## Real-Time Features
- Live puzzle state synchronization
- Instant piece recommendation delivery
- Real-time progress tracking for host
- Immediate feedback on all player actions
- Collaborative hint system through guide tokens

## Reconnection Support
- Automatic state restoration based on current game phase
- Fragment ownership maintained across disconnections
- Auto-solve and relocation for disconnected players during puzzle phase
- Host transfer mechanism when host disconnects
- Session continuity with proper state synchronization
