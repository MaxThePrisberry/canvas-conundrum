# Canvas Conundrum - WebSocket Communication Specification

## Connection System Overview

Canvas Conundrum uses a dual-connection system with dedicated host and player endpoints for reliable game management.

### Host vs Player System

**Host Connection:**
- **Frontend Endpoint**: `/host` (web interface for hosts to enter UUID and connect)
- **WebSocket Endpoint**: `/ws/host/{unique-uuid}` (UUID generated fresh each server start)
- **Role**: Game moderator and controller
- **Capabilities**: Start games, monitor progress, control game flow, view analytics
- **Limitations**: Cannot participate in trivia or puzzle solving
- **Reconnection**: Can reconnect using same endpoint + player ID

**Player Connection:**
- **Endpoint**: `/ws`
- **Role**: Game participants
- **Capabilities**: Answer trivia, collect tokens, solve puzzles, select roles/specialties
- **Requirements**: Host must be present to start games
- **Reconnection**: Can reconnect using player ID

## Authentication System

### Initial Connection
- Establish secure WebSocket connection
- Server generates unique player identifier (UUID v4)
- All subsequent messages require authentication wrapper

### Packet Distribution Principles
- **Host-Only Packets**: Game management, monitoring, and control events
- **Player-Only Packets**: Role selection, trivia, resource gathering, puzzle solving
- **Shared Packets**: Game state updates, phase transitions (when applicable to both)
- **Important**: Hosts do not receive player participation packets (available_roles, trivia_question, image_preview, etc.)

### Authentication Format
All client-to-server events after initial connection use this wrapper:
```json
{
  "auth": {
    "token": "uuid-generated-by-server"
  },
  "payload": {
    // Event-specific data
  }
}
```

**Validation Rules:**
- Token must be valid UUID v4 format
- Token must match the connection's assigned ID
- Payload must be valid JSON
- Message size limited to 8KB
- Comprehensive input validation on all fields

## Game Phases and Events

### 1. Setup Phase

#### Initial Connection Flow

**Player Connection:**
1. Client connects to `/ws`
2. Server generates UUID and creates player
3. Server sends `available_roles` with player ID and options

**Host Connection:**
1. Client navigates to `/host` frontend endpoint
2. Host enters UUID in web form interface
3. Client connects to `/ws/host/{uuid}` (UUID from server logs/API)
4. Server validates no existing host
5. Server creates host player and sends `host_connection_confirmed`

**IMPORTANT**: Hosts do NOT receive the `available_roles` packet since they cannot select roles or specialties. They receive only the `host_connection_confirmed` packet.

#### Server to Client Events

**Available Roles (Sent to Players):**
```json
{
  "playerId": "uuid-generated-by-server",
  "isHost": false,
  "roles": [
    {
      "role": "art_enthusiast",
      "resourceBonus": "constants.RoleResourceMultiplier",
      "bonusToken": "clarity",
      "available": true
    },
    {
      "role": "detective",
      "resourceBonus": "constants.RoleResourceMultiplier",
      "bonusToken": "guide",
      "available": false
    },
    {
      "role": "tourist",
      "resourceBonus": "constants.RoleResourceMultiplier",
      "bonusToken": "chronos",
      "available": true
    },
    {
      "role": "janitor",
      "resourceBonus": "constants.RoleResourceMultiplier",
      "bonusToken": "anchor",
      "available": true
    }
  ],
  "triviaCategories": ["general", "geography", "history", "music", "science", "video_games"]
}
```

**Host Connection Confirmed (Sent to Host Only):**
```json
{
  "playerId": "uuid-generated-by-server",
  "isHost": true,
  "message": "Connected as game host"
}
```

**Game Lobby Status:**
```json
{
  "currentPlayers": 6,
  "nonHostPlayers": 5,
  "playerRoles": {
    "art_enthusiast": 2,
    "detective": 1,
    "tourist": 1,
    "janitor": 1
  },
  "hasHost": true,
  "gameStarting": false,
  "waitingMessage": "Ready to start! (Host can begin the game)"
}
```

**Host Update (Host Only):**
```json
{
  "phase": "setup",
  "connectedPlayers": 5,
  "readyPlayers": 4,
  "teamTokens": {
    "anchorTokens": 0,
    "chronosTokens": 0,
    "guideTokens": 0,
    "clarityTokens": 0
  },
  "playerStatuses": {
    "player1-uuid": {
      "name": "Player1",
      "role": "detective",
      "connected": true,
      "ready": true,
      "location": ""
    }
  }
}
```

#### Client to Server Events

**Role Selection (Players Only):**
```json
{
  "auth": {
    "token": "uuid-generated-by-server"
  },
  "payload": {
    "role": "art_enthusiast"
  }
}
```

**Trivia Specialty Selection (Players Only):**
```json
{
  "auth": {
    "token": "uuid-generated-by-server"
  },
  "payload": {
    "specialties": ["science", "history"]
  }
}
```
*Note: Players are immediately marked as ready upon successful specialty selection*

**Host Start Game (Host Only):**
```json
{
  "auth": {
    "token": "host-uuid"
  },
  "payload": {}
}
```

### 2. Resource Gathering Phase

#### Phase Start
**Resource Phase Start (Players Only):**
```json
{
  "resourceHashes": {
    "anchor": "constants.HashAnchorStation",
    "chronos": "constants.HashChronosStation",
    "guide": "constants.HashGuideStation",
    "clarity": "constants.HashClarityStation"
  }
}
```
**Note**: Hosts do not receive this packet since they don't participate in resource gathering. Hosts receive `host_update` with phase information instead.

**Trivia Round Structure:**
- Each gathering round = one trivia round = one trivia question
- Round duration: `constants.ResourceGatheringRoundDuration` seconds
- First 30 seconds: Answer selection period (multiple choice)
- Last 30 seconds: Answers locked, marked right/wrong, grace period for location changes

**Round Timer Event (All):**
```json
{
  "type": "round_timer",
  "currentRound": 3,
  "totalRounds": 5,
  "timeRemaining": 45,
  "phase": "answering",
  "answerLockTime": 30
}
```
*Phase can be "answering" or "grace_period"*

#### Trivia System

**Trivia Question (Players Only):**
```json
{
  "questionId": "general_medium_42_1234567",
  "text": "What is the capital of France?",
  "category": "geography",
  "difficulty": "medium",
  "timeLimit": 30,
  "options": ["Paris", "London", "Berlin", "Madrid"],
  "isSpecialty": false
}
```

**Specialty Question Example (Players Only):**
```json
{
  "questionId": "science_hard_15_7654321",
  "text": "What is the speed of light in vacuum?",
  "category": "science (Specialty)",
  "difficulty": "hard",
  "timeLimit": 30,
  "options": ["299,792,458 m/s", "300,000,000 m/s", "186,000 mi/s", "3.0 × 10^8 m/s"],
  "isSpecialty": true
}
```
*Note: Specialty questions have same time limit as regular questions. All answers are selected from multiple choice options with no fuzzy matching.*

**Team Progress Update (All):**
```json
{
  "questionsAnswered": 28,
  "totalQuestions": 40,
  "teamTokens": {
    "anchorTokens": 45,
    "chronosTokens": 32,
    "guideTokens": 28,
    "clarityTokens": 38
  },
  "tokenThresholds": {
    "anchor": {
      "currentThreshold": 2,
      "maxThresholds": 6,
      "tokensPerThreshold": "constants.AnchorTokenThreshold",
      "effectPerThreshold": "2 pieces pre-solved"
    },
    "chronos": {
      "currentThreshold": 1,
      "maxThresholds": 6,
      "tokensPerThreshold": "constants.ChronosTokenThreshold",
      "effectPerThreshold": "+20 seconds"
    },
    "guide": {
      "currentThreshold": 1,
      "maxThresholds": 6,
      "tokensPerThreshold": "constants.GuideTokenThreshold",
      "effectPerThreshold": "Remove (gridSize²)/7 squares"
    },
    "clarity": {
      "currentThreshold": 2,
      "maxThresholds": 6,
      "tokensPerThreshold": "constants.ClarityTokenThreshold",
      "effectPerThreshold": "+1 second preview"
    }
  }
}
```

#### Client to Server Events

**Resource Location Verified (Players Only):**
```json
{
  "auth": {
    "token": "uuid-generated-by-server"
  },
  "payload": {
    "verifiedHash": "constants.HashAnchorStation"
  }
}
```
*Note: Only required when changing locations between rounds. The hash must match one of the station constants.*

**Trivia Answer (Players Only):**
```json
{
  "auth": {
    "token": "uuid-generated-by-server"
  },
  "payload": {
    "questionId": "general_medium_42_1234567",
    "answer": "Paris",
    "timestamp": 1640995200
  }
}
```

**Answer Result (Players Only):**
```json
{
  "type": "answer_result",
  "questionId": "general_medium_42_1234567",
  "correct": true,
  "correctAnswer": "Paris",
  "tokensEarned": 20,
  "bonuses": {
    "roleBonus": true,
    "specialtyBonus": false
  },
  "currentLocation": "clarity"
}
```
*Sent after 30-second answer period ends*

### 3. Puzzle Assembly Phase

#### Phase Initialization

**Image Preview (Players Only):**
```json
{
  "imageId": "masterpiece_001",
  "duration": 3
}
```
*Duration = `constants.ClarityBasePreviewTime` + (clarity thresholds × 1) seconds. Hosts do not receive this packet since clarity tokens are a player benefit. Preview shown automatically at puzzle phase start.*

**Puzzle Phase Load (Players):**
```json
{
  "imageId": "masterpiece_001",
  "segmentId": "segment_a5",
  "gridSize": 4,
  "preSolved": false,
  "anchorPreSolvedPieces": 4,
  "totalPieces": 16,
  "allSegmentIds": [
    "segment_a1", "segment_a2", "segment_a3", "segment_a4",
    "segment_b1", "segment_b2", "segment_b3", "segment_b4",
    "segment_c1", "segment_c2", "segment_c3", "segment_c4",
    "segment_d1", "segment_d2", "segment_d3", "segment_d4"
  ],
  "guideHighlightCount": 9
}
```
**CRITICAL**: This loads the player's individual 16-piece puzzle segment that they must solve privately. The `allSegmentIds` allows clients to preload all segments in the background for smooth transition to central grid. The `guideHighlightCount` indicates how many squares should be highlighted on the central grid based on guide token thresholds: total squares - (threshold × gridSize² / 7).

**Puzzle Phase Load (Host):**
```json
{
  "imageId": "masterpiece_001",
  "gridSize": 4,
  "isHost": true,
  "playerCount": 8,
  "message": "Puzzle phase started - monitor player progress"
}
```

**Puzzle Phase Start (All):**
```json
{
  "startTimestamp": 1640995200,
  "totalTime": 340,
  "baseTime": 300,
  "chronosBonus": 40,
  "playerPhases": {
    "phase2": ["player1-uuid", "player2-uuid", "player3-uuid", "player4-uuid"],
    "phase3": []
  }
}
```
*Total time = base time + chronos token bonuses. All players start in phase 2 (individual solving)*

**Puzzle Timer Update (All):**
```json
{
  "type": "puzzle_timer",
  "timeRemaining": 285,
  "totalTime": 340
}
```
*Sent periodically during puzzle phase*

**Central Puzzle State Update (Periodic for Players, Immediate for Host):**
```json
{
  "type": "central_puzzle_state_update",
  "updateInterval": "constants.GridUpdateInterval",
  "note": "Players receive this every 3 seconds, host receives immediately on any change"
}
```

#### Individual vs Central Puzzle System

**CRITICAL DISTINCTION**: Canvas Conundrum operates with two completely separate puzzle systems:

1. **Individual Player Puzzles** (Private, Invisible to Others - Phase 2):
   - Each player receives a unique segment ID to load and solve
   - Client responsibilities:
     - Load segment using provided ID
     - Split into 16 pieces and shuffle
     - Choose which pieces to pre-solve based on anchor token count
     - Handle all piece movement and validation
   - These individual puzzles are completely separate from the central grid
   - No visibility on shared screens until completion
   - No space reserved on central grid until completion
   - Players work on these individually without affecting the shared game state
   - Clients display small grids showing completion status of all other players' segments

2. **Central Shared Puzzle Grid** (Public, Collaborative - Phase 3):
   - Only activated when players complete their individual puzzles
   - Server places completed segments at random unoccupied positions
   - Each completed individual puzzle becomes one movable fragment on the shared grid
   - Collaborative space where players swap fragments to correct positions
   - Visible to all players and host for coordination
   - Players transition from Phase 2 to Phase 3 upon individual completion

#### Grid and Fragment Management

**Dynamic Grid Sizing:**
- 1-9 players: 3×3 grid (9 fragments)
- 10-16 players: 4×4 grid (16 fragments)
- 17-25 players: 5×5 grid (25 fragments)
- Continues scaling to 8×8 maximum

**Fragment Lifecycle:**
1. **Invisible Phase**: Player works on individual 16-piece puzzle (not visible to others)
2. **Completion Trigger**: Player completes individual puzzle and sends completion message
3. **Activation Phase**: Individual puzzle becomes one fragment on central shared grid
4. **Collaborative Phase**: Fragment becomes visible and movable on shared puzzle grid

**Fragment Movement Protocol:**
- All movements are direct swaps between two fragments
- Movement cooldown: `constants.FragmentMoveCooldown` ms consistently applied  
- Also called: switch requests, swap requests, or move requests
- Prevents race conditions and rapid successive moves
- Host receives immediate updates, players receive updates every `constants.GridUpdateInterval` seconds

#### Client to Server Events

**Segment Completed (Players Only):**
```json
{
  "auth": {
    "token": "uuid-generated-by-server"
  },
  "payload": {
    "segmentId": "segment_a5",
    "completionTimestamp": 1640995200
  }
}
```
**CRITICAL**: Sent when player completes their individual puzzle. Server will respond with the full central puzzle state, placing this segment at a random unoccupied position and transitioning the player to Phase 3.

**Fragment Swap Request (Players in Phase 3 Only):**
```json
{
  "auth": {
    "token": "uuid-generated-by-server"
  },
  "payload": {
    "fragment1Id": "fragment_player1-uuid",
    "position1": {"x": 2, "y": 1},
    "fragment2Id": "fragment_player2-uuid",
    "position2": {"x": 3, "y": 0},
    "timestamp": 1640995200
  }
}
```
**Note**: All movements are swaps between two fragments. Only players in Phase 3 can make swap requests.

**Host Start Puzzle Timer (Host Only):**
```json
{
  "auth": {
    "token": "host-uuid"
  },
  "payload": {}
}
```

#### Server to Client Events

**Central Puzzle State (Sent immediately to completing player, periodically to others):**
```json
{
  "status": "acknowledged",
  "segmentId": "segment_a5",
  "gridPosition": {"x": 2, "y": 3},
  "allSegmentCompletions": {
    "segment_a1": true,
    "segment_a2": false,
    "segment_a3": true,
    "segment_a4": true,
    "segment_a5": true
  }
}
```
**CRITICAL**: This confirms that the player's individual puzzle has been converted to a fragment on the central shared grid at the specified position. The `allSegmentCompletions` allows clients to update their small grid displays.

**Fragment Move Response:**
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
  "nextMoveAvailable": 1640995891
}
```

**Central Puzzle State (All):**
```json
{
  "fragments": [
    {
      "id": "fragment_player1-uuid",
      "playerId": "player1-uuid",
      "position": {"x": 0, "y": 0},
      "solved": true,
      "correctPosition": {"x": 0, "y": 0},
      "preSolved": false,
      "visible": true,
      "movableBy": "player1-uuid",
      "segmentId": "segment_a5"
    },
    {
      "id": "fragment_unassigned-1",
      "playerId": null,
      "position": {"x": 1, "y": 1},
      "solved": true,
      "correctPosition": {"x": 1, "y": 1},
      "preSolved": false,
      "visible": true,
      "movableBy": "anyone",
      "segmentId": "segment_b2"
    }
  ],
  "gridSize": 4,
  "playerDisconnected": "disconnected-player-uuid",
  "segmentCompletions": {
    "segment_a1": true,
    "segment_a2": false,
    "segment_a3": true,
    "segment_a4": true,
    "segment_a5": true,
    "segment_b1": false,
    "segment_b2": true
  }
}
```
**CRITICAL**: This shows only the central shared puzzle grid. Individual puzzles in progress are NOT included here and remain completely invisible until completion.

#### Collaboration System

**Piece Recommendation Request:**
```json
{
  "auth": {
    "token": "uuid-generated-by-server"
  },
  "payload": {
    "toPlayerId": "target-player-uuid",
    "fromFragmentId": "fragment_sender-uuid",
    "toFragmentId": "fragment_target-uuid",
    "suggestedFromPos": {"x": 1, "y": 2},
    "suggestedToPos": {"x": 3, "y": 0}
  }
}
```

**Piece Recommendation (Sent to Target Player):**
```json
{
  "id": "recommendation-uuid",
  "fromPlayerId": "sender-uuid",
  "toPlayerId": "receiver-uuid",
  "fromFragmentId": "fragment_sender-uuid",
  "toFragmentId": "fragment_receiver-uuid",
  "suggestedFromPos": {"x": 1, "y": 2},
  "suggestedToPos": {"x": 3, "y": 0},
  "timestamp": "2025-05-26T10:30:00Z"
}
```

**Piece Recommendation Response:**
```json
{
  "auth": {
    "token": "uuid-generated-by-server"
  },
  "payload": {
    "recommendationId": "recommendation-uuid",
    "accepted": true
  }
}
```

#### Token Effects Implementation

**Guide Token Guidance:**
```json
{
  "type": "guide_highlight",
  "playerId": "player-uuid",
  "highlightedArea": {
    "positions": [
      {"x": 2, "y": 2},
      {"x": 2, "y": 3},
      {"x": 3, "y": 2},
      {"x": 3, "y": 3}
    ]
  },
  "thresholdLevel": 2,
  "maxThresholds": 6
}
```
*Note: Guide highlights show possible positions on central grid for player's own fragment. Each threshold removes (gridSize²)/7 positions. Highlights are private to each player.*

**Personal Puzzle State (Players Only):**
```json
{
  "personalView": {
    "fragments": [
      {
        "id": "fragment_player1-uuid",
        "playerId": "player1-uuid",
        "position": {"x": 0, "y": 0},
        "solved": true,
        "correctPosition": {"x": 0, "y": 0},
        "preSolved": false,
        "visible": true,
        "ownedByPlayer": true
      }
    ],
    "gridSize": 4,
    "playerFragmentId": "fragment_player1-uuid",
    "guideHighlight": {
      "positions": [{"x": 2, "y": 2}, {"x": 2, "y": 3}]
    }
  }
}
```
**CRITICAL**: This personal view shows only the central shared puzzle grid, not the player's individual puzzle work in progress.

**Token Effects Summary:**
- **Anchor Tokens**: Pre-solve individual puzzle pieces only (2 pieces per threshold, max 12 of 16 pieces, visually locked)
- **Chronos Tokens**: Extend puzzle time (+20 seconds per threshold, max +120 seconds)
- **Guide Tokens**: Highlight possible positions on central grid for player's fragment (6 thresholds, removing (gridSize²)/7 squares per threshold)
- **Clarity Tokens**: Show complete image preview automatically at puzzle start (+1 second per threshold, max +6 seconds)

#### Disconnection Handling
- Immediate fragment auto-solve for disconnected players
- Random grid placement maintains puzzle integrity
- Host disconnection notifications (no automatic transfer)
- No reconnection support during puzzle assembly phase
- Reconnection support with state restoration in other phases

### 4. Post-Game Analytics

**Game Analytics (All Players):**
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

**Game Reset (All Players):**
```json
{
  "message": "Game resetting. Please rejoin to start a new game.",
  "reconnectRequired": true
}
```

## Error Handling and Validation

### Error Response Format
```json
{
  "error": "Validation failed",
  "details": "Role selection validation: invalid role selection",
  "type": "validation_error"
}
```

### Common Error Types
- `validation_error`: Input validation failures
- `authentication_error`: Auth wrapper or player ID issues
- `game_state_error`: Action not allowed in current phase
- `host_error`: Host-specific operation failures
- `server_shutdown`: Server maintenance notifications

### Validation Rules
- **Token**: Must be valid UUID v4 format matching connection's assigned ID
- **Role Selection**: Must be available role from valid set
- **Specialties**: 1-2 categories from supported list, no duplicates
- **Grid Positions**: Within bounds (0 to gridSize-1)
- **Message Size**: Maximum 8KB payload
- **Hash Validation**: Resource station hashes must match constants
- **Timestamps**: Must be positive integers
- **Text Fields**: UTF-8 validation, length limits, no HTML injection

## Difficulty Scaling

### Difficulty Modifiers Applied
- **Easy Mode**: `constants.EasySpecialtyProbability` specialty questions, `constants.EasyTimeMultiplier` time, `constants.EasyThresholdMultiplier` token requirements
- **Medium Mode**: `constants.MediumSpecialtyProbability` specialty questions, `constants.MediumTimeMultiplier` time, `constants.MediumThresholdMultiplier` token requirements
- **Hard Mode**: `constants.HardSpecialtyProbability` specialty questions, `constants.HardTimeMultiplier` time, `constants.HardThresholdMultiplier` token requirements

### Specialty Question Mechanics
- Higher difficulty level than game setting
- Same time limits as regular questions (no extension)
- Point multiplier (`constants.SpecialtyPointMultiplier`) for correct answers
- Selected from player's chosen specialties

## Performance and Security

### Connection Management
- Ping/pong heartbeats every 30 seconds
- Connection timeout after 60 seconds without pong
- Graceful disconnection handling with state preservation
- Rate limiting on fragment moves (`constants.FragmentMoveCooldown` ms cooldown)

### Security Measures
- CORS validation for allowed origins
- Input sanitization and validation on all messages
- WebSocket upgrade origin checking
- Authentication token validation
- Message size and rate limiting
- Host privilege verification

### Scalability Features
- Dynamic grid scaling (3×3 to 8×8)
- Player limit enforcement (4-64 players)
- Efficient broadcast system with filtering
- Question pool management with automatic cycling
- Connection state monitoring and cleanup

## Technical Implementation Notes

### Message Broadcasting
- Targeted broadcasts using player filters
- Host-only messages for game management
- Player-only messages for game participation
- Efficient serialization with minimal payload sizes

### State Management
- Thread-safe operations with RWMutex
- Atomic state transitions between game phases
- Persistent analytics tracking across reconnections
- Question history management to prevent repeats

### Reconnection Support
- State restoration based on current game phase
- Fragment ownership maintained across disconnections
- Host reconnection to same endpoint with player ID
- Seamless gameplay continuation after reconnections

---

## Individual vs Central Puzzle Summary

**Key Points for Implementation:**

1. **Complete Separation**: Individual player puzzles and the central shared grid are entirely separate systems
2. **No Visibility**: Individual puzzle work is completely invisible to other players and the host
3. **No Reservation**: No space is reserved on the central grid until individual completion
4. **Activation Trigger**: Only upon individual puzzle completion does a fragment appear on the central grid
5. **State Isolation**: Individual puzzle state is never included in central puzzle state broadcasts
6. **Collaborative Focus**: Central grid is purely for collaboration between completed fragments

---

## Additional Critical Events for Complete Implementation

### Configuration and Game Constants

**Game Configuration (Sent on Connection):**
```json
{
  "type": "game_configuration",
  "config": {
    "resourceGatheringRounds": 5,
    "resourceGatheringRoundDuration": 60,
    "puzzleBaseTime": 300,
    "fragmentMoveCooldown": 1000,
    "gridUpdateInterval": 3000,
    "maxSpecialtiesPerPlayer": 2,
    "individualPuzzlePieces": 16,
    "difficultyMode": "medium"
  }
}
```
*Sent immediately after connection to provide game timing constants*

### Phase Transitions

**Phase Transition Event (All):**
```json
{
  "type": "phase_transition",
  "fromPhase": "setup",
  "toPhase": "resource_gathering",
  "message": "Get ready! Resource gathering begins in 5 seconds...",
  "countdown": 5
}
```

### Round Management During Resource Gathering

**Round Start Event (All):**
```json
{
  "type": "round_start",
  "roundNumber": 3,
  "totalRounds": 5,
  "duration": 60
}
```

**Answer Lock Event (Players Only):**
```json
{
  "type": "answer_lock",
  "questionId": "general_medium_42_1234567",
  "timeElapsed": 30,
  "gracePeriodRemaining": 30
}
```
*Sent after 30 seconds to indicate answers are locked*

### Individual Puzzle Progress (Players Only)

**Individual Puzzle Progress:**
```json
{
  "type": "individual_puzzle_progress",
  "segmentId": "segment_a5",
  "piecesPlaced": 12,
  "totalPieces": 16,
  "preSolvedPieces": 8,
  "remainingToSolve": 4
}
```
*Optional: Can be sent periodically to track individual puzzle progress*

### Puzzle Phase Completion

**Puzzle Success (All):**
```json
{
  "type": "puzzle_complete",
  "success": true,
  "completionTime": 285,
  "totalTime": 340,
  "message": "Masterpiece restored! Well done!",
  "acceptingSwaps": false
}
```
*Server rejects all further swap requests after puzzle completion*

**Puzzle Failure (All):**
```json
{
  "type": "puzzle_complete",
  "success": false,
  "reason": "time_expired",
  "fragmentsPlaced": 14,
  "totalFragments": 16,
  "correctlyPlaced": 10,
  "message": "Time's up! The masterpiece remains incomplete.",
  "acceptingSwaps": false
}
```
*Server stops accepting swap requests when timer expires*

### Host-Specific Monitoring Events

**Player Progress Update (Host Only):**
```json
{
  "type": "player_progress",
  "playersInPhase2": 2,
  "playersInPhase3": 2,
  "playerProgresses": {
    "player1-uuid": {
      "phase": 2,
      "segmentCompleted": false,
      "currentLocation": "anchor",
      "tokensEarned": 45,
      "questionsCorrect": 4
    },
    "player2-uuid": {
      "phase": 3,
      "segmentCompleted": true,
      "fragmentPosition": {"x": 2, "y": 1},
      "movesContributed": 3
    }
  },
  "gridSolveProgress": {
    "correctFragments": 8,
    "totalFragments": 16,
    "isComplete": false
  }
}
```
*Host shows count of players in each phase and overall puzzle progress*

### Disconnection Events

**Player Disconnected (All):**
```json
{
  "type": "player_disconnected",
  "playerId": "disconnected-uuid",
  "playerName": "Player3",
  "fragmentStatus": "auto_solved",
  "fragmentId": "fragment_disconnected-uuid",
  "newOwnership": "unassigned"
}
```

**Host Disconnected (Players Only):**
```json
{
  "type": "host_disconnected",
  "currentPhase": "puzzle_assembly",
  "gamePaused": false,
  "message": "Host disconnected. Game continuing..."
}
```
*Note: Game only pauses during setup/resource phases, not during puzzle phase*

**Host Reconnected (Players Only):**
```json
{
  "type": "host_reconnected",
  "currentPhase": "puzzle_assembly",
  "message": "Host reconnected and monitoring resumed."
}
```
