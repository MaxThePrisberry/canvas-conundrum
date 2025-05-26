# Canvas Conundrum - WebSocket Communication Specification

## Connection Lifecycle

### Initial Connection
- Establish secure WebSocket connection
- Client authentication
- Game session initialization

## Game Phases and Events

### 1. Setup Phase

#### Client to Server Events
- `player_join`:
  - Unique player identifier
- `role_selection`:
  ```json
  {
    "playerId": "unique_id",
    "role": "art_enthusiast",
    "resourceBonus": 1.5
  }
  ```
- `trivia_specialty_selection`:
  ```json
  {
    "playerId": "unique_id",
    "specialties": ["tech", "art"]
  }
  ```

#### Server to Client Events
- `available_roles`:
  - Current role distribution
- `game_setup_status`:
  - Player count
  - Role availability
  - Game readiness

### 2. Resource Gathering Phase

#### Client to Server Events
- `trivia_answer`:
  ```json
  {
    "questionId": "q123",
    "answer": "selected_answer",
    "timestamp": 1234567890
  }
  ```
- `resource_location`:
  ```json
  {
    "playerId": "unique_id",
    "qrCodeLocation": "station_a"
  }
  ```

#### Server to Client Events
- `team_token_update`:
  ```json
  {
    "anchorTokens": 3,
    "chronosTokens": 2,
    "guideTokens": 1,
    "clarityTokens": 1
  }
  ```
- `trivia_question`:
  ```json
  {
    "questionId": "q123",
    "text": "Question text",
    "category": "tech",
    "difficulty": "medium",
    "timeLimit": 60
  }
  ```

### 3. Puzzle Assembly Phase

#### Grid Configuration Management
- Dynamic grid sizing algorithm:
  ```
  Grid Size = Ceil(Sqrt(Total Players))
  Total Fragments = Grid Size²
  ```

#### Fragment Movement Protocol
- Movement Cooldown: 1 second
- Prevents race conditions
- Ignores rapid successive move requests

#### Client to Server Events
- `fragment_move_request`:
  ```json
  {
    "fragmentId": "unique_identifier",
    "newPosition": {"x": 0, "y": 0},
    "timestamp": 1234567890
  }
  ```

#### Server to Client Events
- `fragment_move_response`:
  ```json
  {
    "status": "success" | "ignored" | "invalid",
    "reason": "cooldown" | "out_of_bounds" | null,
    "fragment": { /* fragment details */ },
    "nextMoveAvailable": 1234567890
  }
  ```
- `central_puzzle_state`:
  - Complete puzzle configuration
  - Fragment locations
  - Player progress

### Disconnection Handling
- Immediate fragment auto-solve
- Random grid placement
- Broadcast disconnection to all players

## Post-Game Events

#### Server to Client
- `personal_analytics`:
  ```json
  {
    "playerId": "unique_id",
    "tokenCollection": { /* token stats */ },
    "triviaPerformance": { /* trivia stats */ }
  }
  ```
- `team_analytics`:
  - Overall team performance
  - Detailed game statistics
- `global_leaderboard`:
  - Player rankings

## Error Handling and Edge Cases
- Robust error detection
- Graceful degradation
- Comprehensive logging

## Performance Considerations
- Efficient data serialization
- Minimal payload size
- Rate limiting
- Connection state management

## Security Measures
- Secure WebSocket connection
- Input validation
- Anti-cheat mechanisms# Canvas Conundrum - WebSocket Communication Specification

## Game Phases and WebSocket Events

### 1. Setup Phase Events

#### Client to Server
- `player_join`: 
  - Send player's unique identifier
  - Request available character roles
- `role_selection`: 
  - Selected character role
- `trivia_specialty_selection`:
  - 1-2 trivia categories player is proficient in

#### Server to Client
- `available_roles`: 
  - List of currently available roles
- `game_setup_status`:
  - Current player count
  - Role distribution
  - Waiting for game to start

### 2. Resource Gathering Phase Events

#### Client to Server
- `trivia_answer`:
  - Question ID
  - Selected answer
- `resource_location`:
  - Current QR code/resource station
- `player_token_update`:
  - Resources collected

#### Server to Client
- `team_token_update`:
  - Current team token counts for each resource type
- `trivia_question`:
  - Question text
  - Question difficulty
  - Question category
- `player_score_update`:
  - Individual player's contribution
  - Team progress

### 3. Puzzle Assembly Phase Events

#### Grid Configuration Management
- Dynamic grid sizing based on player count
- Grid size calculation:
  ```
  Grid Size = Ceil(Sqrt(Total Players))
  Total Fragments = Grid Size²
  ```

#### Fragment Movement Protocol
- Movement Cooldown Mechanism:
  ```javascript
  class FragmentMovementController {
    constructor() {
      this.lastMoveTimestamp = 0;
      this.MOVE_COOLDOWN = 1000; // 1 second
    }

    canMove(currentTimestamp) {
      return currentTimestamp - this.lastMoveTimestamp >= this.MOVE_COOLDOWN;
    }

    registerMove(currentTimestamp) {
      this.lastMoveTimestamp = currentTimestamp;
    }
  }
  ```

#### Client to Server Events
- `fragment_move_request`:
  ```json
  {
    "fragmentId": "unique_identifier",
    "newPosition": {"x": 0, "y": 0},
    "timestamp": 1234567890
  }
  ```
- Server validates:
  1. Fragment exists
  2. Move is within cooldown period
  3. Move is within grid boundaries

#### Server to Client Events
- `fragment_move_response`:
  ```json
  {
    "status": "success" | "ignored" | "invalid",
    "reason": "cooldown" | "out_of_bounds" | null,
    "fragment": { /* fragment details */ },
    "nextMoveAvailable": 1234567890
  }
  ```

#### Disconnection Handling
- Immediate fragment auto-solve
- Random grid placement
- Broadcast disconnection to all players

## Technical Specifications
- Minimum Grid: 3x3 (1-9 players)
- Maximum Dynamic Grid Scaling
- Consistent 1-second movement cooldown
- Robust error handling
- Prevent race conditions

### Grid Size Progression
- 1-9 players: 3x3 (9 fragments)
- 10-16 players: 4x4 (16 fragments)
- 17-25 players: 5x5 (25 fragments)
- Continues with n² grid sizes

### Disconnection Workflow
1. Detect WebSocket communication failure
2. Mark player as disconnected
3. Auto-solve player's fragment
4. Randomly place fragment
5. Notify all players
6. Continue game progression

### 4. Post-Game Events

#### Server to Client
- `personal_analytics`:
  - Individual player performance
  - Token collection stats
  - Trivia performance
- `team_analytics`:
  - Overall team performance
  - Detailed game statistics
- `global_leaderboard`:
  - Player rankings across different metrics

## Disconnection Handling
- Persistent session management using cookies/caching
- Ability to rejoin game in progress
- Restore player's previous state upon reconnection

## Technical Considerations
- WebSocket connection maintained throughout game
- Fallback to polling if WebSocket connection fails
- Secure authentication for each game session
- Encryption of sensitive game state information

## Performance Optimization
- Minimal payload size
- Efficient data serialization
- Rate limiting on event frequency
- Batch updates where possible

## Error Handling
- Graceful degradation of features
- Clear error messages
- Automatic reconnection attempts
- Fallback mechanisms for critical game events
