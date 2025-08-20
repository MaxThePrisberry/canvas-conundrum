# Canvas Conundrum - Complete WebSocket Event Specifications

## Authentication Wrapper Format

All client-to-server messages (except initial connection) use this format:

```json
{
  "event": "EVENT_NAME",
  "auth": {
    "token": "uuid-generated-by-server"
  },
  "payload": {
    // Event-specific data shown below
  },
  "timestamp": "2025-01-XX:XX:XX.XXXZ"
}
```

All server-to-client messages use this format:

```json
{
  "event": "EVENT_NAME",
  "payload": {
    // Event-specific data shown below
  },
  "timestamp": "2025-01-XX:XX:XX.XXXZ"
}
```

---

## Reconnection Behavior

### Host Reconnection
When a host reconnects using the same UUID and token:

1. **Connection Confirmation**: Always receives `SETUP_TO_HOST_CONNECTION_CONFIRMED` with:
   - `isReconnection: true`
   - `currentPhase`: Current game phase
   - Current game state appropriate to the phase

2. **Phase-Specific State Restoration Events**:
   - **Setup Phase**:
     - `SETUP_TO_HOST_PLAYER_ROSTER`
   - **Resource Gathering Phase**:
     - `RESOURCE_TO_HOST_PHASE_START`
     - `RESOURCE_TO_HOST_ROUND_ANALYTICS` (if rounds are in progress)
   - **Puzzle Assembly Phase**:
     - `PUZZLE_TO_HOST_PHASE_LOAD`
     - `PUZZLE_TO_HOST_GRID_STATE`
     - `PUZZLE_TO_HOST_TIMER_START` (if timer is active)
   - **Analytics Phase**:
     - `ANALYTICS_TO_HOST_COMPLETE_REPORT`

3. **Reconnection Notification**: All players receive `SYSTEM_TO_CLIENT_HOST_RECONNECTED`

### Player Reconnection
When a player reconnects using the same token:

1. **Phase-Specific Behavior**:

   **Setup Phase:**
   - Receives `SETUP_TO_PLAYER_ROLES_AVAILABLE` with `isReconnection: true`
   - If already configured, automatically marked as ready
   - If already configured, all players receive `SETUP_TO_CLIENT_LOBBY_STATUS` with updated lobby state
   - Host receives `SETUP_TO_HOST_PLAYER_ROSTER` update

   **Resource Gathering Phase:**
   - Receives `SETUP_TO_PLAYER_ROLES_AVAILABLE` with `isReconnection: true`
   - Followed by `RESOURCE_TO_CLIENT_PHASE_START`
   - Followed by `RESOURCE_TO_CLIENT_TEAM_PROGRESS`
   - If mid-round, receives current `RESOURCE_TO_PLAYER_TRIVIA_QUESTION`

   **Puzzle Assembly Phase:**
   - **HTTP 403 Forbidden** returned during WebSocket upgrade for ALL player connections
   - Connection refused at the HTTP level before WebSocket establishment
   - No distinction between new players and reconnecting players - all blocked

   **Analytics Phase:**
   - Receives `SETUP_TO_PLAYER_ROLES_AVAILABLE` with `isReconnection: true`
   - Followed by `ANALYTICS_TO_PLAYER_PERSONAL_REPORT`
   - Followed by `ANALYTICS_TO_CLIENT_TEAM_SUMMARY`

2. **Important Notes**:
   - Player retains their authentication token and previous game state
   - Host receives updated player roster showing reconnection (except during puzzle phase)
   - All reconnection state restoration happens automatically after initial connection
   - During puzzle assembly phase, NO player WebSocket connections are permitted regardless of reconnection status

---

## Phase 0: Connection and Setup

### Initial Connection Events

#### `SETUP_TO_HOST_CONNECTION_CONFIRMED`
**Direction**: Server → Host
**Trigger**: Host connects to `/ws/host/{uuid}` (initial connection or reconnection)

```json
{
  "event": "SETUP_TO_HOST_CONNECTION_CONFIRMED",
  "payload": {
    "playerId": "uuid-generated-by-server",
    "message": "Connected as game host",
    "currentPhase": "setup",
    "isReconnection": false,
    "gameConfig": {
      "minPlayers": 4,
      "maxPlayers": 64,
      "resourceGatheringRounds": 5,
      "resourceGatheringRoundDuration": 60,
      "puzzleBaseTime": 300,
      "difficultyMode": "medium"
    }
  },
  "timestamp": "2025-01-XX:XX:XX.XXXZ"
}
```

#### `SETUP_TO_PLAYER_ROLES_AVAILABLE`
**Direction**: Server → Player
**Trigger**: Player connects to `/ws` (initial connection or reconnection) or when role availability changes due to more players joining

```json
{
  "event": "SETUP_TO_PLAYER_ROLES_AVAILABLE",
  "payload": {
    "playerId": "uuid-generated-by-server",
    "currentPhase": "setup",
    "isReconnection": false,
    "existingConfiguration": null,
    "roles": [
      {
        "roleType": "art_enthusiast",
        "displayName": "Art Enthusiast",
        "resourceBonus": 1.5,
        "bonusTokenType": "clarity",
        "description": "Excels at clarity token collection",
        "available": true
      },
      {
        "roleType": "detective",
        "displayName": "Detective",
        "resourceBonus": 1.5,
        "bonusTokenType": "guide",
        "description": "Excels at guide token collection",
        "available": false
      },
      {
        "roleType": "tourist",
        "displayName": "Tourist",
        "resourceBonus": 1.5,
        "bonusTokenType": "chronos",
        "description": "Excels at chronos token collection",
        "available": true
      },
      {
        "roleType": "janitor",
        "displayName": "Janitor",
        "resourceBonus": 1.5,
        "bonusTokenType": "anchor",
        "description": "Excels at anchor token collection",
        "available": true
      }
    ],
    "triviaCategories": [
      "general",
      "geography",
      "history",
      "music",
      "science",
      "video_games"
    ],
    "maxSpecialties": 1
  },
  "timestamp": "2025-01-XX:XX:XX.XXXZ"
}
```

### Role and Specialty Selection

#### `SETUP_TO_SERVER_PLAYER_CONFIGURATION`
**Direction**: Player → Server
**Trigger**: Player completes role and specialty selection

```json
{
  "event": "SETUP_TO_SERVER_PLAYER_CONFIGURATION",
  "auth": {
    "token": "player-uuid"
  },
  "payload": {
    "selectedRole": "art_enthusiast",
    "selectedSpecialties": ["science"],
    "playerName": "Player Display Name"
  },
  "timestamp": "2025-01-XX:XX:XX.XXXZ"
}
```

#### `SETUP_TO_CLIENT_LOBBY_STATUS`
**Direction**: Server → All Players
**Trigger**: Any player joins/leaves or changes configuration

```json
{
  "event": "SETUP_TO_CLIENT_LOBBY_STATUS",
  "payload": {
    "currentPlayers": 5,
    "minPlayers": 4,
    "maxPlayers": 64,
    "playerRoles": {
      "art_enthusiast": 2,
      "detective": 1,
      "tourist": 1,
      "janitor": 1
    },
    "hasHost": true,
    "allPlayersReady": false,
    "readyPlayers": 4,
    "gameStartEligible": false,
    "waitingMessage": "Waiting for 1 more player to be ready"
  },
  "timestamp": "2025-01-XX:XX:XX.XXXZ"
}
```

#### `SETUP_TO_HOST_PLAYER_ROSTER`
**Direction**: Server → Host
**Trigger**: Any change to player status

```json
{
  "event": "SETUP_TO_HOST_PLAYER_ROSTER",
  "payload": {
    "phase": "setup",
    "connectedPlayers": 5,
    "readyPlayers": 4,
    "gameStartEligible": false,
    "playerStatuses": {
      "player1-uuid": {
        "playerName": "Alice",
        "role": "detective",
        "specialties": ["science"],
        "connected": true,
        "ready": true,
        "lastActivity": "2025-01-XX:XX:XX.XXXZ"
      },
      "player2-uuid": {
        "playerName": "Bob",
        "role": "art_enthusiast",
        "specialties": ["music"],
        "connected": true,
        "ready": false,
        "lastActivity": "2025-01-XX:XX:XX.XXXZ"
      }
    },
    "roleDistribution": {
      "art_enthusiast": 2,
      "detective": 1,
      "tourist": 1,
      "janitor": 1
    }
  },
  "timestamp": "2025-01-XX:XX:XX.XXXZ"
}
```

### Game Start

#### `SETUP_TO_SERVER_START_GAME`
**Direction**: Host → Server
**Trigger**: Host clicks start game button

```json
{
  "event": "SETUP_TO_SERVER_START_GAME",
  "auth": {
    "token": "host-uuid"
  },
  "payload": {},
  "timestamp": "2025-01-XX:XX:XX.XXXZ"
}
```

#### `SETUP_TO_CLIENT_GAME_STARTED`
**Direction**: Server → All Players
**Trigger**: Host starts game successfully

```json
{
  "event": "SETUP_TO_CLIENT_GAME_STARTED",
  "payload": {
    "nextPhase": "resource_gathering",
    "transitionCountdown": 5,
    "message": "Game starting! Prepare for resource gathering phase.",
    "instructions": "Make your way to the resource gathering stations."
  },
  "timestamp": "2025-01-XX:XX:XX.XXXZ"
}
```

#### `SETUP_TO_HOST_GAME_STARTED`
**Direction**: Server → Host
**Trigger**: Game successfully started

```json
{
  "event": "SETUP_TO_HOST_GAME_STARTED",
  "payload": {
    "phase": "resource_gathering",
    "gameStarted": true,
    "totalPlayers": 5,
    "initialTeamTokens": {
      "anchorTokens": 0,
      "chronosTokens": 0,
      "guideTokens": 0,
      "clarityTokens": 0
    },
    "monitoringActive": true
  },
  "timestamp": "2025-01-XX:XX:XX.XXXZ"
}
```

---

## Phase 1: Resource Gathering

### Phase Initialization

#### `RESOURCE_TO_CLIENT_PHASE_START`
**Direction**: Server → All Players
**Trigger**: Resource gathering phase begins

```json
{
  "event": "RESOURCE_TO_CLIENT_PHASE_START",
  "payload": {
    "phase": "resource_gathering",
    "totalRounds": 5,
    "roundDuration": 60,
    "answerTime": 30,
    "graceTime": 30,
    "resourceStationHashes": {
      "anchor": "hash_anchor_station_constant",
      "chronos": "hash_chronos_station_constant",
      "guide": "hash_guide_station_constant",
      "clarity": "hash_clarity_station_constant"
    },
    "tokenThresholds": {
      "anchor": 25,
      "chronos": 20,
      "guide": 15,
      "clarity": 30
    },
    "difficultySettings": {
      "mode": "medium",
      "specialtyProbability": 0.3,
      "timeMultiplier": 1.0,
      "thresholdMultiplier": 1.0
    }
  },
  "timestamp": "2025-01-XX:XX:XX.XXXZ"
}
```

#### `RESOURCE_TO_HOST_PHASE_START`
**Direction**: Server → Host
**Trigger**: Resource gathering phase begins

```json
{
  "event": "RESOURCE_TO_HOST_PHASE_START",
  "payload": {
    "phase": "resource_gathering",
    "monitoringDashboard": {
      "totalRounds": 5,
      "currentRound": 0,
      "roundDuration": 60,
      "playerDistribution": {
        "anchor": 0,
        "chronos": 0,
        "guide": 0,
        "clarity": 0,
        "unknown": 5
      }
    },
    "analyticsTracking": {
      "questionDelivery": true,
      "answerTracking": true,
      "locationTracking": true,
      "performanceMetrics": true
    }
  },
  "timestamp": "2025-01-XX:XX:XX.XXXZ"
}
```

### Location Verification

#### `RESOURCE_TO_SERVER_LOCATION_VERIFIED`
**Direction**: Player → Server
**Trigger**: Player scans QR code at new station

```json
{
  "event": "RESOURCE_TO_SERVER_LOCATION_VERIFIED",
  "auth": {
    "token": "player-uuid"
  },
  "payload": {
    "stationHash": "hash_anchor_station_constant",
    "previousLocation": "clarity",
    "scanTimestamp": "2025-01-XX:XX:XX.XXXZ"
  },
  "timestamp": "2025-01-XX:XX:XX.XXXZ"
}
```

### Trivia Questions and Answers

#### `RESOURCE_TO_PLAYER_TRIVIA_QUESTION`
**Direction**: Server → Individual Player
**Trigger**: Start of each resource gathering round

```json
{
  "event": "RESOURCE_TO_PLAYER_TRIVIA_QUESTION",
  "payload": {
    "questionId": "general_medium_42_1234567",
    "questionText": "What is the capital of France?",
    "category": "geography",
    "difficulty": "medium",
    "isSpecialty": false,
    "specialtyBonus": false,
    "timeLimit": 30,
    "options": [
      "Paris",
      "London",
      "Berlin",
      "Madrid"
    ],
    "roundNumber": 3,
    "totalRounds": 5,
    "answerDeadline": "2025-01-XX:XX:XX.XXXZ"
  },
  "timestamp": "2025-01-XX:XX:XX.XXXZ"
}
```

#### `RESOURCE_TO_SERVER_TRIVIA_ANSWER`
**Direction**: Player → Server
**Trigger**: Player selects answer or time expires

```json
{
  "event": "RESOURCE_TO_SERVER_TRIVIA_ANSWER",
  "auth": {
    "token": "player-uuid"
  },
  "payload": {
    "questionId": "general_medium_42_1234567",
    "selectedAnswer": "Paris",
    "answerIndex": 0,
    "timeElapsed": 18.5,
    "currentLocation": "clarity"
  },
  "timestamp": "2025-01-XX:XX:XX.XXXZ"
}
```

#### `RESOURCE_TO_PLAYER_ANSWER_RESULT`
**Direction**: Server → Individual Player
**Trigger**: Answer period ends for question

```json
{
  "event": "RESOURCE_TO_PLAYER_ANSWER_RESULT",
  "payload": {
    "questionId": "general_medium_42_1234567",
    "correct": true,
    "selectedAnswer": "Paris",
    "correctAnswer": "Paris",
    "tokensEarned": 30,
    "baseTokens": 20,
    "bonuses": {
      "roleBonus": true,
      "roleBonusTokens": 10,
      "specialtyBonus": false,
      "specialtyBonusTokens": 0,
      "difficultyMultiplier": 1.0
    },
    "currentLocation": "clarity"
  },
  "timestamp": "2025-01-XX:XX:XX.XXXZ"
}
```

### Progress Updates

#### `RESOURCE_TO_CLIENT_TEAM_PROGRESS`
**Direction**: Server → All Players
**Trigger**: After each round or significant progress

```json
{
  "event": "RESOURCE_TO_CLIENT_TEAM_PROGRESS",
  "payload": {
    "currentRound": 3,
    "totalRounds": 5,
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
        "tokensPerThreshold": 25,
        "effectDescription": "2 pieces pre-solved per threshold"
      },
      "chronos": {
        "currentThreshold": 1,
        "maxThresholds": 6,
        "tokensPerThreshold": 20,
        "effectDescription": "+20 seconds per threshold"
      },
      "guide": {
        "currentThreshold": 1,
        "maxThresholds": 6,
        "tokensPerThreshold": 15,
        "effectDescription": "Remove (gridSize²)/7 squares per threshold"
      },
      "clarity": {
        "currentThreshold": 1,
        "maxThresholds": 6,
        "tokensPerThreshold": 30,
        "effectDescription": "+1 second preview per threshold"
      }
    },
    "teamPerformance": {
      "averageAccuracy": 0.78,
      "roundTimeRemaining": 42
    }
  },
  "timestamp": "2025-01-XX:XX:XX.XXXZ"
}
```

#### `RESOURCE_TO_HOST_ROUND_ANALYTICS`
**Direction**: Server → Host
**Trigger**: After each question round

```json
{
  "event": "RESOURCE_TO_HOST_ROUND_ANALYTICS",
  "payload": {
    "currentRound": 3,
    "totalRounds": 5,
    "roundResults": {
      "questionsDelivered": 5,
      "answersReceived": 5,
      "correctAnswers": 4,
      "averageResponseTime": 22.3,
      "tokensAwarded": 95
    },
    "playerPerformance": {
      "player1-uuid": {
        "location": "anchor",
        "answerCorrect": true,
        "responseTime": 18.2,
        "tokensEarned": 30,
        "runningAccuracy": 0.85
      },
      "player2-uuid": {
        "location": "clarity",
        "answerCorrect": false,
        "responseTime": 28.7,
        "tokensEarned": 0,
        "runningAccuracy": 0.72
      }
    },
    "stationDistribution": {
      "anchor": 2,
      "chronos": 1,
      "guide": 1,
      "clarity": 1
    },
    "teamTokens": {
      "anchorTokens": 45,
      "chronosTokens": 32,
      "guideTokens": 28,
      "clarityTokens": 38
    }
  },
  "timestamp": "2025-01-XX:XX:XX.XXXZ"
}
```

### Phase Completion

#### `RESOURCE_TO_CLIENT_PHASE_COMPLETE`
**Direction**: Server → All Players
**Trigger**: All resource gathering rounds completed

```json
{
  "event": "RESOURCE_TO_CLIENT_PHASE_COMPLETE",
  "payload": {
    "phase": "resource_gathering",
    "nextPhase": "puzzle_assembly",
    "finalTokenTotals": {
      "anchorTokens": 87,
      "chronosTokens": 64,
      "guideTokens": 52,
      "clarityTokens": 78
    },
    "thresholdAchievements": {
      "anchor": 3,
      "chronos": 3,
      "guide": 3,
      "clarity": 2
    },
    "bonusEffects": {
      "preSolvedPieces": 6,
      "extraTime": 60,
      "guideHighlights": 3,
      "previewTime": 5
    },
    "transitionInstructions": "Return to the main room for puzzle assembly",
    "transitionCountdown": 30
  },
  "timestamp": "2025-01-XX:XX:XX.XXXZ"
}
```

#### `RESOURCE_TO_HOST_PHASE_COMPLETE`
**Direction**: Server → Host
**Trigger**: Resource gathering phase ends

```json
{
  "event": "RESOURCE_TO_HOST_PHASE_COMPLETE",
  "payload": {
    "phase": "resource_gathering",
    "completed": true,
    "totalQuestionsAnswered": 40,
    "teamPerformance": {
      "overallAccuracy": 0.78,
      "totalTokensEarned": 281,
      "averageResponseTime": 23.5
    },
    "finalTokenDistribution": {
      "anchorTokens": 87,
      "chronosTokens": 64,
      "guideTokens": 52,
      "clarityTokens": 78
    },
    "playerAnalytics": {
      "player1-uuid": {
        "questionsAnswered": 8,
        "correctAnswers": 7,
        "accuracy": 0.875,
        "tokensEarned": 165,
        "specialtyPerformance": {
          "questionsReceived": 3,
          "correctAnswers": 3,
          "bonusTokens": 45
        }
      }
    },
    "readyForPuzzlePhase": true
  },
  "timestamp": "2025-01-XX:XX:XX.XXXZ"
}
```

---

## Phase 2: Puzzle Assembly

### Phase Initialization

#### `PUZZLE_TO_CLIENT_PHASE_LOAD`
**Direction**: Server → All Players
**Trigger**: Puzzle phase begins

```json
{
  "event": "PUZZLE_TO_CLIENT_PHASE_LOAD",
  "payload": {
    "phase": "puzzle_assembly",
    "imageId": "masterpiece_001",
    "assignedSegmentId": "segment_01",
    "individualPuzzleSize": 16,
    "anchorPreSolvedPieces": 6,
    "centralGridSize": 4,
    "totalCentralFragments": 16,
    "clarityPreviewDuration": 5,
    "guideHighlightCount": 9,
    "allSegmentIds": [
      "segment_01", "segment_02", "segment_03", "segment_04",
      "segment_05", "segment_06", "segment_07", "segment_08",
      "segment_09", "segment_10", "segment_11", "segment_12",
      "segment_13", "segment_14", "segment_15", "segment_16"
    ],
    "loadInstructions": "Load your assigned segment and prepare individual puzzle",
    "waitingForHost": true
  },
  "timestamp": "2025-01-XX:XX:XX.XXXZ"
}
```

#### `PUZZLE_TO_HOST_PHASE_LOAD`
**Direction**: Server → Host
**Trigger**: Puzzle phase begins

```json
{
  "event": "PUZZLE_TO_HOST_PHASE_LOAD",
  "payload": {
    "phase": "puzzle_assembly",
    "imageId": "masterpiece_001",
    "centralGridSize": 4,
    "totalFragments": 16,
    "playerCount": 5,
    "playerSegmentAssignments": {
      "player1-uuid": "segment_a1",
      "player2-uuid": "segment_a2",
      "player3-uuid": "segment_a3",
      "player4-uuid": "segment_a4",
      "player5-uuid": "segment_b1"
    },
    "unassignedSegments": [
      "segment_b2", "segment_b3", "segment_b4",
      "segment_c1", "segment_c2", "segment_c3", "segment_c4",
      "segment_d1", "segment_d2", "segment_d3", "segment_d4"
    ],
    "bonusEffects": {
      "anchorPreSolved": 6,
      "chronosTimeBonus": 60,
      "guideHighlights": 9,
      "clarityPreview": 5
    },
    "monitoringActive": true,
    "canStartTimer": true
  },
  "timestamp": "2025-01-XX:XX:XX.XXXZ"
}
```

### Timer Management

#### `PUZZLE_TO_SERVER_START_TIMER`
**Direction**: Host → Server
**Trigger**: Host starts puzzle timer

```json
{
  "event": "PUZZLE_TO_SERVER_START_TIMER",
  "auth": {
    "token": "host-uuid"
  },
  "payload": {},
  "timestamp": "2025-01-XX:XX:XX.XXXZ"
}
```

#### `PUZZLE_TO_CLIENT_TIMER_START`
**Direction**: Server → All Players
**Trigger**: Puzzle timer started by host

```json
{
  "event": "PUZZLE_TO_CLIENT_TIMER_START",
  "payload": {
    "startTimestamp": 1640995200000,
    "totalTime": 360,
    "baseTime": 300,
    "chronosBonus": 60,
    "clarityPreviewActive": true,
    "previewDuration": 5,
    "playerPhases": {
      "phase2": ["player1-uuid", "player2-uuid", "player3-uuid", "player4-uuid", "player5-uuid"],
      "phase3": []
    },
    "instructions": "Begin solving your individual puzzle segments"
  },
  "timestamp": "2025-01-XX:XX:XX.XXXZ"
}
```

#### `PUZZLE_TO_HOST_TIMER_START`
**Direction**: Server → Host
**Trigger**: Puzzle timer started

```json
{
  "event": "PUZZLE_TO_HOST_TIMER_START",
  "payload": {
    "timerActive": true,
    "startTimestamp": 1640995200000,
    "totalTime": 360,
    "baseTime": 300,
    "bonusTime": 60,
    "playersInPhase2": 5,
    "playersInPhase3": 0
  },
  "timestamp": "2025-01-XX:XX:XX.XXXZ"
}
```

### Individual Puzzle Solving (Phase 2A)

#### `PUZZLE_TO_SERVER_SEGMENT_COMPLETED`
**Direction**: Player → Server
**Trigger**: Player completes their 16-piece individual puzzle

```json
{
  "event": "PUZZLE_TO_SERVER_SEGMENT_COMPLETED",
  "auth": {
    "token": "player-uuid"
  },
  "payload": {
    "segmentId": "segment_a5",
    "completionTimestamp": 1640995200000,
    "solveTime": 180,
    "manualPiecesSolved": 10,
    "preSolvedPieces": 6
  },
  "timestamp": "2025-01-XX:XX:XX.XXXZ"
}
```

#### `PUZZLE_TO_PLAYER_SEGMENT_ACKNOWLEDGED`
**Direction**: Server → Individual Player
**Trigger**: Server processes segment completion

```json
{
  "event": "PUZZLE_TO_PLAYER_SEGMENT_ACKNOWLEDGED",
  "payload": {
    "segmentId": "segment_01",
    "acknowledged": true,
    "centralGridPosition": {"x": 2, "y": 3},
    "fragmentId": "fragment_01",
    "allSegmentCompletions": {
      "segment_01": true,
      "segment_02": false,
      "segment_03": true,
      "segment_04": true,
      "segment_05": true,
      "segment_06": false,
      "segment_07": false,
      "segment_08": false,
      "segment_09": false,
      "segment_10": false,
      "segment_11": false,
      "segment_12": false,
      "segment_13": false,
      "segment_14": false,
      "segment_15": false,
      "segment_16": false
    }
  },
  "timestamp": "2025-01-XX:XX:XX.XXXZ"
}
```

#### `PUZZLE_TO_HOST_SEGMENT_COMPLETED`
**Direction**: Server → Host
**Trigger**: Player completes individual segment

```json
{
  "event": "PUZZLE_TO_HOST_SEGMENT_COMPLETED",
  "payload": {
    "playerId": "player1-uuid",
    "playerName": "Alice",
    "segmentId": "segment_01",
    "completionTime": 180,
    "centralGridPosition": {"x": 2, "y": 3},
    "fragmentId": "fragment_01",
    "phaseTransition": {
      "playersInPhase2": 4,
      "playersInPhase3": 1
    },
    "completionStats": {
      "totalCompleted": 1,
      "totalRequired": 5,
      "unassignedFragments": 11
    }
  },
  "timestamp": "2025-01-XX:XX:XX.XXXZ"
}
```

### Collaborative Fragment Movement (Phase 2B)

#### `PUZZLE_TO_SERVER_FRAGMENT_MOVE`
**Direction**: Player → Server
**Trigger**: Player attempts to swap fragments on central grid

```json
{
  "event": "PUZZLE_TO_SERVER_FRAGMENT_MOVE",
  "auth": {
    "token": "player-uuid"
  },
  "payload": {
    "fragmentId": "fragment_01",
    "currentPosition": {"x": 2, "y": 3},
    "targetPosition": {"x": 1, "y": 1},
    "swapWithFragmentId": "fragment_02"
  },
  "timestamp": "2025-01-XX:XX:XX.XXXZ"
}
```

#### `PUZZLE_TO_PLAYER_MOVE_RESULT`
**Direction**: Server → Individual Player
**Trigger**: Server processes movement request

```json
{
  "event": "PUZZLE_TO_PLAYER_MOVE_RESULT",
  "payload": {
    "moveId": "move-uuid-12345",
    "status": "success",
    "fragmentId": "fragment_01",
    "newPosition": {"x": 1, "y": 1},
    "swappedFragmentId": "fragment_02",
    "swappedFragmentNewPosition": {"x": 2, "y": 3},
    "cooldownInfo": {
      "nextMoveAvailable": 1640995203000,
      "cooldownRemaining": 2.5
    }
  },
  "timestamp": "2025-01-XX:XX:XX.XXXZ"
}
```

### Grid State Updates

#### `PUZZLE_TO_CLIENT_GRID_STATE`
**Direction**: Server → All Players
**Trigger**: Periodic updates (every 3 seconds) or after movements

```json
{
  "event": "PUZZLE_TO_CLIENT_GRID_STATE",
  "payload": {
    "updateType": "periodic",
    "gridSize": 4,
    "fragments": [
      {
        "fragmentId": "fragment_01",
        "segmentId": "segment_01",
        "position": {"x": 1, "y": 1},
        "visible": true
      },
      {
        "fragmentId": "fragment_02",
        "segmentId": "segment_02",
        "position": {"x": 2, "y": 3},
        "visible": true
      }
    ],
    "timeRemaining": 285
  },
  "timestamp": "2025-01-XX:XX:XX.XXXZ"
}
```

#### `PUZZLE_TO_HOST_GRID_STATE`
**Direction**: Server → Host
**Trigger**: Immediate updates on any change

```json
{
  "event": "PUZZLE_TO_HOST_GRID_STATE",
  "payload": {
    "updateType": "immediate",
    "gridSize": 4,
    "fragments": [
      {
        "fragmentId": "fragment_01",
        "playerId": "player1-uuid",
        "playerName": "Alice",
        "segmentId": "segment_01",
        "position": {"x": 1, "y": 1},
        "visible": true,
        "lastMoved": "2025-01-XX:XX:XX.XXXZ",
        "moveCount": 3
      }
    ],
    "playerMetrics": {
      "player1-uuid": {
        "phase": 3,
        "fragmentsOwned": 1,
        "movesContributed": 3,
        "successfulMoves": 2,
        "lastActivity": "2025-01-XX:XX:XX.XXXZ"
      }
    },
    "collaborationActivity": {
      "recentMoves": 5,
      "activeRecommendations": 2,
      "moveFrequency": 1.2
    },
    "timeRemaining": 285
  },
  "timestamp": "2025-01-XX:XX:XX.XXXZ"
}
```

### Strategic Collaboration

#### `PUZZLE_TO_SERVER_RECOMMEND_MOVE`
**Direction**: Player → Server
**Trigger**: Player sends recommendation to another player

```json
{
  "event": "PUZZLE_TO_SERVER_RECOMMEND_MOVE",
  "auth": {
    "token": "player-uuid"
  },
  "payload": {
    "targetPlayerId": "player2-uuid",
    "fromFragmentId": "fragment_01",
    "toFragmentId": "fragment_02",
    "reasoning": "This swap would place both fragments closer to correct positions"
  },
  "timestamp": "2025-01-XX:XX:XX.XXXZ"
}
```

#### `PUZZLE_TO_PLAYER_MOVE_RECOMMENDATION`
**Direction**: Server → Target Player
**Trigger**: Another player sends recommendation

```json
{
  "event": "PUZZLE_TO_PLAYER_MOVE_RECOMMENDATION",
  "payload": {
    "moveId": "rec-uuid-67890",
    "fromPlayerId": "player1-uuid",
    "fromPlayerName": "Alice",
    "toPlayerId": "player2-uuid",
    "fromFragmentId": "fragment_01",
    "toFragmentId": "fragment_02",
    "reasoning": "This swap would place both fragments closer to correct positions",
    "expiresAt": "2025-01-XX:XX:XX.XXXZ",
    "currentPositions": {
      "fromFragment": {"x": 1, "y": 1},
      "toFragment": {"x": 3, "y": 3}
    }
  },
  "timestamp": "2025-01-XX:XX:XX.XXXZ"
}
```

#### `PUZZLE_TO_SERVER_RECOMMENDATION_RESPONSE`
**Direction**: Player → Server
**Trigger**: Player responds to recommendation

```json
{
  "event": "PUZZLE_TO_SERVER_RECOMMENDATION_RESPONSE",
  "auth": {
    "token": "player-uuid"
  },
  "payload": {
    "moveId": "rec-uuid-67890",
    "response": "accept",
    "responseReason": "Good strategic move"
  },
  "timestamp": "2025-01-XX:XX:XX.XXXZ"
}
```

#### `PUZZLE_TO_PLAYER_RECOMMENDATION_RESULT`
**Direction**: Server → Original Player
**Trigger**: Target player responds to recommendation

```json
{
  "event": "PUZZLE_TO_PLAYER_RECOMMENDATION_RESULT",
  "payload": {
    "moveId": "rec-uuid-67890",
    "targetPlayerId": "player2-uuid",
    "targetPlayerName": "Bob",
    "response": "accept",
    "responseReason": "Good strategic move",
    "executionStatus": "success",
    "swapExecuted": {
      "fragment1Id": "fragment_01",
      "fragment1OldPosition": {"x": 1, "y": 1},
      "fragment1NewPosition": {"x": 3, "y": 3},
      "fragment2Id": "fragment_02",
      "fragment2OldPosition": {"x": 3, "y": 3},
      "fragment2NewPosition": {"x": 1, "y": 1}
    }
  },
  "timestamp": "2025-01-XX:XX:XX.XXXZ"
}
```

#### `PUZZLE_TO_PLAYER_RECOMMENDATION_EXPIRED`
**Direction**: Server → Target Player
**Trigger**: Recommendation becomes invalid due to grid changes

```json
{
  "event": "PUZZLE_TO_PLAYER_RECOMMENDATION_EXPIRED",
  "payload": {
    "moveId": "rec-uuid-67890",
    "reason": "grid_state_changed",
    "details": "One or more fragments involved in recommendation have moved"
  },
  "timestamp": "2025-01-XX:XX:XX.XXXZ"
}
```

### Puzzle Completion

#### `PUZZLE_TO_CLIENT_COMPLETED_SUCCESS`
**Direction**: Server → All Participants
**Trigger**: Puzzle completed successfully

```json
{
  "event": "PUZZLE_TO_CLIENT_COMPLETED_SUCCESS",
  "payload": {
    "success": true,
    "completionTime": 285,
    "totalTime": 360,
    "timeRemaining": 75,
    "message": "Masterpiece restored! Well done!",
    "celebrationDuration": 5,
    "finalGridState": {
      "allFragmentsCorrect": true,
      "totalFragments": 16,
      "correctFragments": 16
    },
    "teamAchievements": [
      "Perfect Collaboration",
      "Strategic Masters",
      "Time Efficient"
    ],
    "nextPhase": "analytics"
  },
  "timestamp": "2025-01-XX:XX:XX.XXXZ"
}
```

#### `PUZZLE_TO_CLIENT_COMPLETED_TIMEOUT`
**Direction**: Server → All Participants
**Trigger**: Time expires before completion

```json
{
  "event": "PUZZLE_TO_CLIENT_COMPLETED_TIMEOUT",
  "payload": {
    "success": false,
    "reason": "time_expired",
    "totalTime": 360,
    "timeExpired": true,
    "finalStats": {
      "fragmentsPlaced": 14,
      "totalFragments": 16,
      "correctlyPlaced": 10,
      "completionPercentage": 62.5
    },
    "message": "Time's up! The masterpiece remains incomplete.",
    "partialAchievements": [
      "Team Effort",
      "Good Communication"
    ],
    "nextPhase": "analytics"
  },
  "timestamp": "2025-01-XX:XX:XX.XXXZ"
}
```

#### `PUZZLE_TO_HOST_COMPLETION_ANALYTICS`
**Direction**: Server → Host
**Trigger**: Puzzle phase ends (success or timeout)

```json
{
  "event": "PUZZLE_TO_HOST_COMPLETION_ANALYTICS",
  "payload": {
    "puzzleSuccess": true,
    "completionTime": 285,
    "totalTime": 360,
    "efficiency": 0.79,
    "playerContributions": {
      "player1-uuid": {
        "individualSolveTime": 180,
        "fragmentMoves": 8,
        "successfulMoves": 7,
        "recommendationsSent": 3,
        "recommendationsReceived": 2,
        "recommendationsAccepted": 1,
        "finalFragmentCorrect": true
      }
    },
    "collaborationMetrics": {
      "totalMoves": 32,
      "successfulMoves": 28,
      "totalRecommendations": 12,
      "acceptedRecommendations": 7,
      "averageResponseTime": 15.2,
      "communicationScore": 0.85
    },
    "phaseTransitions": {
      "playersCompletedIndividual": 5,
      "averageIndividualTime": 165,
      "fastestIndividual": 142,
      "slowestIndividual": 198
    }
  },
  "timestamp": "2025-01-XX:XX:XX.XXXZ"
}
```

---

## Phase 3: Post-Game Analytics

### Analytics Distribution

#### `ANALYTICS_TO_PLAYER_PERSONAL_REPORT`
**Direction**: Server → Individual Player
**Trigger**: Game completion

```json
{
  "event": "ANALYTICS_TO_PLAYER_PERSONAL_REPORT",
  "payload": {
    "playerId": "player1-uuid",
    "playerName": "Alice",
    "gameSuccess": true,
    "personalScore": 1850,
    "rank": 1,
    "totalPlayers": 5,
    "tokenCollection": {
      "anchorTokens": 12,
      "chronosTokens": 8,
      "guideTokens": 15,
      "clarityTokens": 10,
      "totalTokens": 45
    },
    "triviaPerformance": {
      "totalQuestions": 20,
      "correctAnswers": 16,
      "accuracy": 0.80,
      "accuracyByCategory": {
        "general": 0.85,
        "science": 0.90,
        "history": 0.75
      },
      "specialtyPerformance": {
        "specialtyQuestions": 5,
        "specialtyCorrect": 4,
        "specialtyAccuracy": 0.80,
        "specialtyBonus": 40
      },
      "averageResponseTime": 18.5
    },
    "puzzleSolvingMetrics": {
      "individualSolveTime": 180,
      "individualRank": 2,
      "fragmentMoves": 8,
      "successfulMoves": 7,
      "moveAccuracy": 0.875,
      "recommendationsSent": 3,
      "recommendationsReceived": 2,
      "recommendationsAccepted": 1,
      "collaborationScore": 0.82
    },
    "achievements": [
      "Trivia Master",
      "Strategic Thinker",
      "Team Player"
    ],
    "scoreBreakdown": {
      "triviaPoints": 960,
      "specialtyBonus": 240,
      "puzzlePoints": 350,
      "collaborationBonus": 180,
      "speedBonus": 120,
      "totalScore": 1850
    }
  },
  "timestamp": "2025-01-XX:XX:XX.XXXZ"
}
```

#### `ANALYTICS_TO_CLIENT_TEAM_SUMMARY`
**Direction**: Server → All Players
**Trigger**: Game completion

```json
{
  "event": "ANALYTICS_TO_CLIENT_TEAM_SUMMARY",
  "payload": {
    "gameSuccess": true,
    "totalScore": 8250,
    "totalPlayers": 5,
    "gameTime": 1200,
    "teamPerformance": {
      "overallAccuracy": 0.78,
      "totalTokensCollected": 281,
      "thresholdAchievements": {
        "anchor": 3,
        "chronos": 3,
        "guide": 3,
        "clarity": 2
      },
      "puzzleCompletionTime": 285,
      "collaborationEfficiency": 0.85
    },
    "globalLeaderboard": [
      {
        "playerId": "player1-uuid",
        "playerName": "Alice",
        "totalScore": 1850,
        "rank": 1,
        "role": "detective"
      },
      {
        "playerId": "player2-uuid",
        "playerName": "Bob",
        "totalScore": 1720,
        "rank": 2,
        "role": "art_enthusiast"
      },
      {
        "playerId": "player3-uuid",
        "playerName": "Charlie",
        "totalScore": 1680,
        "rank": 3,
        "role": "tourist"
      },
      {
        "playerId": "player4-uuid",
        "playerName": "Diana",
        "totalScore": 1575,
        "rank": 4,
        "role": "janitor"
      },
      {
        "playerId": "player5-uuid",
        "playerName": "Eve",
        "totalScore": 1425,
        "rank": 5,
        "role": "art_enthusiast"
      }
    ],
    "teamAchievements": [
      "Perfect Collaboration",
      "Token Masters",
      "Puzzle Champions"
    ],
    "notableStats": {
      "fastestAnswerer": "Alice",
      "mostTokens": "Charlie",
      "bestCollaborator": "Bob",
      "puzzleMVP": "Diana"
    }
  },
  "timestamp": "2025-01-XX:XX:XX.XXXZ"
}
```

#### `ANALYTICS_TO_HOST_COMPLETE_REPORT`
**Direction**: Server → Host
**Trigger**: Game completion

```json
{
  "event": "ANALYTICS_TO_HOST_COMPLETE_REPORT",
  "payload": {
    "gameId": "game-uuid-12345",
    "gameSuccess": true,
    "totalGameTime": 1200,
    "totalPlayers": 5,
    "difficultyMode": "medium",
    "overallPerformance": {
      "totalScore": 8250,
      "averageScore": 1650,
      "completionRate": 1.0,
      "efficiency": 0.82
    },
    "resourceGatheringAnalytics": {
      "totalRounds": 5,
      "questionsAnswered": 40,
      "overallAccuracy": 0.78,
      "tokenDistribution": {
        "anchorTokens": 87,
        "chronosTokens": 64,
        "guideTokens": 52,
        "clarityTokens": 78
      },
      "playerPerformance": {
        "player1-uuid": {
          "questionsAnswered": 8,
          "correctAnswers": 7,
          "accuracy": 0.875,
          "tokensEarned": 165,
          "averageResponseTime": 18.5,
          "specialtyPerformance": {
            "questionsReceived": 3,
            "correctAnswers": 3,
            "bonusTokens": 45
          },
          "stationPreferences": {
            "anchor": 2,
            "chronos": 1,
            "guide": 3,
            "clarity": 2
          }
        }
      }
    },
    "puzzleAssemblyAnalytics": {
      "totalTime": 360,
      "completionTime": 285,
      "timeUtilization": 0.79,
      "individualPhaseMetrics": {
        "averageSolveTime": 165,
        "fastestCompletion": 142,
        "slowestCompletion": 198,
        "preSolvedPiecesUsed": 30
      },
      "collaborativePhaseMetrics": {
        "totalMoves": 32,
        "successfulMoves": 28,
        "moveAccuracy": 0.875,
        "totalRecommendations": 12,
        "acceptedRecommendations": 7,
        "recommendationAcceptanceRate": 0.583
      },
      "playerContributions": {
        "player1-uuid": {
          "individualSolveTime": 180,
          "fragmentMoves": 8,
          "successfulMoves": 7,
          "recommendationsSent": 3,
          "recommendationsReceived": 2,
          "recommendationsAccepted": 1,
          "collaborationScore": 0.82
        }
      }
    },
    "collaborationAnalysis": {
      "communicationScore": 0.85,
      "coordinationScore": 0.80,
      "averageResponseTime": 15.2,
      "recommendationNetwork": {
        "totalRecommendations": 12,
        "acceptedRecommendations": 7,
        "mostActiveRecommender": "player1-uuid",
        "bestCollaborator": "player2-uuid"
      },
      "teamworkMetrics": {
        "mutualAssistanceScore": 0.78,
        "strategicCoordination": 0.85,
        "conflictResolution": 0.92
      }
    },
    "categoryPerformance": {
      "general": {
        "questionsAsked": 12,
        "correctAnswers": 9,
        "accuracy": 0.75
      },
      "science": {
        "questionsAsked": 8,
        "correctAnswers": 7,
        "accuracy": 0.875
      },
      "history": {
        "questionsAsked": 6,
        "correctAnswers": 5,
        "accuracy": 0.833
      },
      "geography": {
        "questionsAsked": 7,
        "correctAnswers": 5,
        "accuracy": 0.714
      },
      "music": {
        "questionsAsked": 4,
        "correctAnswers": 3,
        "accuracy": 0.75
      },
      "video_games": {
        "questionsAsked": 3,
        "correctAnswers": 2,
        "accuracy": 0.667
      }
    },
    "timelineAnalysis": {
      "setupPhase": 120,
      "resourcePhase": 300,
      "puzzlePhase": 285,
      "analyticsPhase": 0,
      "phaseTransitions": [
        {
          "fromPhase": "setup",
          "toPhase": "resource_gathering",
          "timestamp": "2025-01-XX:XX:XX.XXXZ",
          "duration": 5
        },
        {
          "fromPhase": "resource_gathering",
          "toPhase": "puzzle_assembly",
          "timestamp": "2025-01-XX:XX:XX.XXXZ",
          "duration": 30
        }
      ]
    },
    "recommendationsForImprovement": [
      "Consider increasing difficulty for geography category",
      "Player communication was excellent",
      "Puzzle phase could benefit from additional time",
      "Team showed strong strategic coordination"
    ]
  },
  "timestamp": "2025-01-XX:XX:XX.XXXZ"
}
```

### Game Reset

#### `ANALYTICS_TO_SERVER_RESET_GAME`
**Direction**: Host → Server
**Trigger**: Host initiates new game

```json
{
  "event": "ANALYTICS_TO_SERVER_RESET_GAME",
  "auth": {
    "token": "host-uuid"
  },
  "payload": {
    "confirmReset": true,
    "saveAnalytics": true
  },
  "timestamp": "2025-01-XX:XX:XX.XXXZ"
}
```

#### `ANALYTICS_TO_CLIENT_GAME_RESET`
**Direction**: Server → All Participants
**Trigger**: Host resets game

```json
{
  "event": "ANALYTICS_TO_CLIENT_GAME_RESET",
  "payload": {
    "reason": "host_initiated_reset",
    "message": "Game resetting. Please rejoin to start a new game.",
    "reconnectRequired": true,
    "reconnectInstructions": "Refresh your browser and reconnect to join the next game",
    "gracePeriod": 30,
    "newGameAvailable": true
  },
  "timestamp": "2025-01-XX:XX:XX.XXXZ"
}
```

---

## System-Wide Events (Cross-Phase)

### Error Handling

#### `SYSTEM_TO_CLIENT_ERROR`
**Direction**: Server → Client
**Trigger**: Various error conditions

```json
{
  "event": "SYSTEM_TO_CLIENT_ERROR",
  "payload": {
    "errorType": "validation_error",
    "errorCode": "INVALID_ROLE_SELECTION",
    "message": "Role selection validation failed",
    "details": "Selected role 'invalid_role' is not available",
    "context": {
      "requestedAction": "role_selection",
      "currentPhase": "setup",
      "playerId": "player-uuid"
    },
    "suggestedActions": [
      "Select from available roles: art_enthusiast, detective, tourist, janitor",
      "Refresh the page if role list appears outdated"
    ],
    "retryable": true,
    "severity": "warning"
  },
  "timestamp": "2025-01-XX:XX:XX.XXXZ"
}
```

#### `SYSTEM_TO_HOST_ERROR`
**Direction**: Server → Host
**Trigger**: Host-specific errors

```json
{
  "event": "SYSTEM_TO_HOST_ERROR",
  "payload": {
    "errorType": "game_state_error",
    "errorCode": "INSUFFICIENT_PLAYERS",
    "message": "Cannot start game with insufficient players",
    "details": "Need at least 4 players, currently have 3",
    "context": {
      "requestedAction": "start_game",
      "currentPlayers": 3,
      "requiredPlayers": 4,
      "connectedPlayers": 3,
      "readyPlayers": 3
    },
    "suggestedActions": [
      "Wait for more players to join",
      "Verify all players are properly connected"
    ],
    "retryable": true,
    "severity": "error"
  },
  "timestamp": "2025-01-XX:XX:XX.XXXZ"
}
```

### Connection Management

#### `SYSTEM_TO_CLIENT_DISCONNECTION_WARNING`
**Direction**: Server → Client
**Trigger**: Connection issues detected

```json
{
  "event": "SYSTEM_TO_CLIENT_DISCONNECTION_WARNING",
  "payload": {
    "warning": "connection_unstable",
    "message": "Connection quality degraded, monitoring for disconnection",
    "reconnectInstructions": "If disconnected, reconnect using the same browser session",
    "statePreservation": {
      "phase": "resource_gathering",
      "progress": "saved",
      "canReconnect": true,
      "reconnectTimeLimit": 300
    },
    "connectionMetrics": {
      "latency": 250,
      "packetLoss": 0.05,
      "lastPong": "2025-01-XX:XX:XX.XXXZ"
    }
  },
  "timestamp": "2025-01-XX:XX:XX.XXXZ"
}
```

#### `SYSTEM_TO_CLIENT_HOST_DISCONNECTED`
**Direction**: Server → All Players
**Trigger**: Host disconnects

```json
{
  "event": "SYSTEM_TO_CLIENT_HOST_DISCONNECTED",
  "payload": {
    "hostStatus": "disconnected",
    "currentPhase": "puzzle_assembly",
    "gameImpact": {
      "gamePaused": false,
      "canContinue": true,
      "affectedFeatures": ["host_monitoring", "phase_transitions"]
    },
    "message": "Host disconnected. Game continuing without host monitoring.",
    "reconnectInfo": {
      "hostCanReconnect": true,
      "reconnectTimeLimit": 600,
      "gameWillContinue": true
    }
  },
  "timestamp": "2025-01-XX:XX:XX.XXXZ"
}
```

#### `SYSTEM_TO_CLIENT_HOST_RECONNECTED`
**Direction**: Server → All Players
**Trigger**: Host reconnects

```json
{
  "event": "SYSTEM_TO_CLIENT_HOST_RECONNECTED",
  "payload": {
    "hostStatus": "reconnected",
    "currentPhase": "puzzle_assembly",
    "restoredFeatures": ["host_monitoring", "phase_transitions", "analytics_tracking"],
    "message": "Host reconnected. Full monitoring resumed.",
    "gameImpact": {
      "gameResumed": false,
      "monitoringRestored": true,
      "noInterruption": true
    }
  },
  "timestamp": "2025-01-XX:XX:XX.XXXZ"
}
```

#### `SYSTEM_TO_HOST_PLAYER_DISCONNECTED`
**Direction**: Server → Host
**Trigger**: Player disconnects

```json
{
  "event": "SYSTEM_TO_HOST_PLAYER_DISCONNECTED",
  "payload": {
    "playerId": "player3-uuid",
    "playerName": "Charlie",
    "disconnectionTime": "2025-01-XX:XX:XX.XXXZ",
    "currentPhase": "puzzle_assembly",
    "gameImpact": {
      "fragmentHandling": {
        "fragmentId": "fragment_player3-uuid",
        "action": "auto_solved_and_unassigned",
        "newPosition": {"x": 1, "y": 2},
        "ownershipTransfer": "unassigned"
      },
      "phaseImpact": {
        "individualPuzzle": "auto_completed",
        "collaborativeAccess": "revoked"
      }
    },
    "reconnectionPolicy": {
      "canReconnectInCurrentPhase": false,
      "canReconnectInNextPhase": true,
      "statePreservation": "analytics_only"
    },
    "updatedPlayerCount": 4
  },
  "timestamp": "2025-01-XX:XX:XX.XXXZ"
}
```

### Heartbeat and Health

#### `SYSTEM_PING`
**Direction**: Client → Server
**Trigger**: Periodic heartbeat (every 30 seconds)

```json
{
  "event": "SYSTEM_PING",
  "payload": {
    "clientTimestamp": "2025-01-XX:XX:XX.XXXZ",
    "sequenceNumber": 42,
    "connectionQuality": {
      "latency": 45,
      "messagesReceived": 156,
      "messagesSent": 23
    }
  },
  "timestamp": "2025-01-XX:XX:XX.XXXZ"
}
```

#### `SYSTEM_PONG`
**Direction**: Server → Client
**Trigger**: Response to ping

```json
{
  "event": "SYSTEM_PONG",
  "payload": {
    "serverTimestamp": "2025-01-XX:XX:XX.XXXZ",
    "clientTimestamp": "2025-01-XX:XX:XX.XXXZ",
    "sequenceNumber": 42,
    "serverHealth": {
      "activeConnections": 6,
      "serverLoad": 0.15,
      "gamePhase": "puzzle_assembly"
    },
    "roundTripTime": 47
  },
  "timestamp": "2025-01-XX:XX:XX.XXXZ"
}
```

### Phase Transitions

#### `SYSTEM_TO_CLIENT_PHASE_TRANSITION`
**Direction**: Server → All Participants
**Trigger**: Game phase changes

```json
{
  "event": "SYSTEM_TO_CLIENT_PHASE_TRANSITION",
  "payload": {
    "fromPhase": "resource_gathering",
    "toPhase": "puzzle_assembly",
    "transitionReason": "resource_phase_completed",
    "countdown": 30,
    "message": "Transitioning to puzzle assembly phase in 30 seconds",
    "preparationInstructions": [
      "Return to the main room",
      "Prepare for collaborative puzzle solving",
      "Individual puzzle segments will be assigned"
    ],
    "phaseInfo": {
      "nextPhaseName": "Puzzle Assembly",
      "nextPhaseDescription": "Solve individual puzzles then collaborate on master assembly",
      "estimatedDuration": "6-8 minutes"
    }
  },
  "timestamp": "2025-01-XX:XX:XX.XXXZ"
}
```

#### `SYSTEM_TO_HOST_PHASE_TRANSITION`
**Direction**: Server → Host
**Trigger**: Game phase changes

```json
{
  "event": "SYSTEM_TO_HOST_PHASE_TRANSITION",
  "payload": {
    "fromPhase": "resource_gathering",
    "toPhase": "puzzle_assembly",
    "transitionStatus": "confirmed",
    "transitionTime": 30,
    "hostControls": {
      "availableActions": ["start_puzzle_timer", "monitor_progress"],
      "monitoringFeatures": ["individual_progress", "grid_state", "collaboration_metrics"],
      "phaseSpecificControls": ["timer_management", "completion_tracking"]
    },
    "playerTransitionStatus": {
      "playersReady": 5,
      "playersTransitioning": 0,
      "transitionComplete": true
    },
    "nextPhaseSetup": {
      "segmentAssignments": "completed",
      "gridInitialization": "ready",
      "bonusEffectsApplied": true
    }
  },
  "timestamp": "2025-01-XX:XX:XX.XXXZ"
}
```

---

## Event Summary

**Total Events**: 53
- **Setup Phase**: 8 events
- **Resource Gathering Phase**: 10 events
- **Puzzle Assembly Phase**: 20 events
- **Post-Game Analytics**: 5 events
- **System-Wide Events**: 10 events

**By Direction**:
- **Client to Server**: 8 events
- **Host to Server**: 3 events
- **Server to All Clients**: 15 events
- **Server to Host Only**: 15 events
- **Server to Individual Players**: 12 events

This specification provides exact JSON structures for every WebSocket event in Canvas Conundrum, eliminating ambiguity about payload field names, data types, and message formats. Each event includes comprehensive context, making implementation straightforward for both client and server developers.
