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
    "playerId": "uuid-generated-by-server"
  },
  "payload": {
    // Event-specific data
  }
}
```

**Validation Rules:**
- Player ID must be valid UUID v4 format
- Player ID must match the connection's assigned ID
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
      "resourceBonus": 1.5,
      "available": true
    },
    {
      "role": "detective",
      "resourceBonus": 1.5,
      "available": false
    },
    {
      "role": "tourist",
      "resourceBonus": 1.5,
      "available": true
    },
    {
      "role": "janitor",
      "resourceBonus": 1.5,
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
    "playerId": "uuid-generated-by-server"
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
    "playerId": "uuid-generated-by-server"
  },
  "payload": {
    "specialties": ["science", "history"]
  }
}
```
*Note: Players are automatically marked ready after selecting specialties*

**Host Start Game (Host Only):**
```json
{
  "auth": {
    "playerId": "host-uuid"
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
    "anchor": "HASH_ANCHOR_STATION_2025",
    "chronos": "HASH_CHRONOS_STATION_2025",
    "guide": "HASH_GUIDE_STATION_2025",
    "clarity": "HASH_CLARITY_STATION_2025"
  }
}
```
**Note**: Hosts do not receive this packet since they don't participate in resource gathering. Hosts receive `host_update` with phase information instead.

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
*Note: Specialty questions have same time limit as regular questions*

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
  }
}
```

#### Client to Server Events

**Resource Location Verified (Players Only):**
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
*Note: Only required when changing locations between rounds*

**Trivia Answer (Players Only):**
```json
{
  "auth": {
    "playerId": "uuid-generated-by-server"
  },
  "payload": {
    "questionId": "general_medium_42_1234567",
    "answer": "Paris",
    "timestamp": 1640995200
  }
}
```

### 3. Puzzle Assembly Phase

#### Phase Initialization

**Image Preview (Players Only):**
```json
{
  "imageId": "masterpiece_001",
  "duration": 3
}
```
*Duration based on clarity tokens earned. Hosts do not receive this packet since clarity tokens are a player benefit.*

**Puzzle Phase Load (Players):**
```json
{
  "imageId": "masterpiece_001",
  "segmentId": "segment_a5",
  "gridSize": 4,
  "preSolved": false
}
```
**CRITICAL**: This loads the player's individual 16-piece puzzle segment that they must solve privately. This segment has NO connection to the central shared puzzle grid until completion.

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
  "totalTime": 340
}
```
*Total time includes chronos token bonuses*

#### Individual vs Central Puzzle System

**CRITICAL DISTINCTION**: Canvas Conundrum operates with two completely separate puzzle systems:

1. **Individual Player Puzzles** (Private, Invisible to Others):
   - Each player receives a unique 16-piece puzzle segment to solve privately
   - These individual puzzles are completely separate from the central grid
   - No visibility on shared screens until completion
   - No space reserved on central grid until completion
   - Players work on these individually without affecting the shared game state

2. **Central Shared Puzzle Grid** (Public, Collaborative):
   - Only activated when players complete their individual puzzles
   - Each completed individual puzzle becomes one movable fragment on the shared grid
   - Collaborative space where players move their completed fragments to correct positions
   - Visible to all players and host for coordination

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
- Movement cooldown: 1000ms consistently applied
- Prevents race conditions and rapid successive moves
- Cooldown applies to all fragments regardless of type

#### Client to Server Events

**Segment Completed (Players Only):**
```json
{
  "auth": {
    "playerId": "uuid-generated-by-server"
  },
  "payload": {
    "segmentId": "segment_a5",
    "completionTimestamp": 1640995200
  }
}
```
**CRITICAL**: This event transforms the player's individual puzzle into a fragment on the central shared grid. Before this event, the individual puzzle work is completely invisible to other players and the central grid.

**Fragment Move Request (All Players):**
```json
{
  "auth": {
    "playerId": "uuid-generated-by-server"
  },
  "payload": {
    "fragmentId": "fragment_player-uuid",
    "newPosition": {"x": 2, "y": 1},
    "timestamp": 1640995200
  }
}
```
**Note**: Only applies to fragments on the central shared grid, not individual puzzles

**Host Start Puzzle Timer (Host Only):**
```json
{
  "auth": {
    "playerId": "host-uuid"
  },
  "payload": {}
}
```

#### Server to Client Events

**Segment Completion Acknowledgment (Players):**
```json
{
  "status": "acknowledged",
  "segmentId": "segment_a5",
  "gridPosition": {"x": 2, "y": 3}
}
```
**CRITICAL**: This confirms that the player's individual puzzle has been converted to a fragment on the central shared grid at the specified position.

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
      "movableBy": "player1-uuid"
    },
    {
      "id": "fragment_unassigned-1",
      "playerId": null,
      "position": {"x": 1, "y": 1},
      "solved": true,
      "correctPosition": {"x": 1, "y": 1},
      "preSolved": false,
      "visible": true,
      "movableBy": "anyone"
    }
  ],
  "gridSize": 4,
  "playerDisconnected": "disconnected-player-uuid"
}
```
**CRITICAL**: This shows only the central shared puzzle grid. Individual puzzles in progress are NOT included here and remain completely invisible until completion.

#### Collaboration System

**Piece Recommendation Request:**
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
    "playerId": "uuid-generated-by-server"
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
  "maxThresholds": 5
}
```

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
- **Anchor Tokens**: Pre-solve puzzle pieces (max 12 of 16 pieces)
- **Chronos Tokens**: Extend puzzle time (+20 seconds per threshold)
- **Guide Tokens**: Provide placement guidance with highlighted areas (linear threshold progression from large area to 2-position precision)
- **Clarity Tokens**: Show complete image preview (+1 second per threshold)

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
- **Player ID**: Must be valid UUID v4 format
- **Role Selection**: Must be available role from valid set
- **Specialties**: 1-2 categories from supported list, no duplicates
- **Grid Positions**: Within bounds (0 to gridSize-1)
- **Message Size**: Maximum 8KB payload
- **Hash Validation**: Resource station hashes must match constants
- **Timestamps**: Must be positive integers
- **Text Fields**: UTF-8 validation, length limits, no HTML injection

## Difficulty Scaling

### Difficulty Modifiers Applied
- **Easy Mode**: 20% specialty questions, +30% time, -20% token requirements
- **Medium Mode**: 30% specialty questions, normal time/tokens
- **Hard Mode**: 40% specialty questions, -30% time, +30% token requirements

### Specialty Question Mechanics
- Higher difficulty level than game setting
- Extended time limits (1.5x base)
- Point multiplier (2.0x) for correct answers
- Selected from player's chosen specialties

## Performance and Security

### Connection Management
- Ping/pong heartbeats every 30 seconds
- Connection timeout after 60 seconds without pong
- Graceful disconnection handling with state preservation
- Rate limiting on fragment moves (1000ms cooldown)

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
