# Canvas Conundrum - Comprehensive Game Design

## Game Concept
A collaborative puzzle-solving game where players recover a stolen masterpiece by gathering resources, answering trivia, and assembling a fragmented artwork. The game uses a dedicated host system for reliable game management and real-time coordination.

## Host vs Player System

### Host System
Canvas Conundrum uses a dedicated host model for reliable game management:

**Host Connection & Role:**
- **Endpoint**: `/ws/host/{unique-uuid}` (UUID generated fresh each server start)
- **Capabilities**:
  - Start and control game flow
  - Monitor all player progress in real-time
  - Access comprehensive analytics and statistics
  - Control puzzle phase timing
  - View detailed player performance metrics
- **Limitations**:
  - Cannot participate in trivia questions
  - Cannot solve puzzle segments
  - Cannot select roles or specialties
  - Does not count toward minimum player requirements
- **Reconnection**: Can reconnect using same endpoint + assigned player ID
- **Management**: Only one host allowed per game instance

**Host Benefits:**
- Reliable game progression without dependency on player actions
- Comprehensive game monitoring and control
- Detailed real-time analytics for educational/team-building scenarios
- Ability to pace game according to group needs

### Player System
**Player Connection & Role:**
- **Endpoint**: `/ws`
- **Capabilities**:
  - Select character roles with resource bonuses
  - Choose trivia specialty categories
  - Answer trivia questions during resource gathering
  - Solve individual puzzle segments
  - Collaborate on puzzle assembly through recommendations
  - Move fragments on shared puzzle grid
- **Requirements**: Host must be connected for game to start
- **Reconnection**: Can reconnect using assigned player ID

## Authentication System

### Security Model
All communication after initial connection requires authentication:

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

**Validation Features:**
- UUID v4 format validation for player IDs
- Comprehensive input validation (8KB message limit)
- UTF-8 text validation with length limits
- Privilege verification (host vs player actions)
- Rate limiting on fragment movements (1000ms cooldown)
- CORS origin validation for security

## Player Setup and Character Selection

### Role Selection (Players Only)
**4 Available Roles:**
1. **Art Enthusiast** → Clarity Token Bonus (1.5x multiplier)
2. **Detective** → Guide Token Bonus (1.5x multiplier)
3. **Tourist** → Chronos Token Bonus (1.5x multiplier)
4. **Janitor** → Anchor Token Bonus (1.5x multiplier)

**Role Mechanics:**
- Each role provides bonus collection for specific token type
- Bonus multiplier: `constants.RoleResourceMultiplier` (1.5x)
- Even distribution enforced across players
- Host does not select a role

**Character Distribution Algorithm:**
- Calculates max per role: `(playerCount + 3) / 4`
- Ensures representation of all roles in larger groups
- As players join, more people are allowed to choose each role

### Trivia Specialty Selection (Players Only)
**Available Categories:**
- General Knowledge, Geography, History, Music, Science, Video Games

**Specialty Mechanics:**
- Players select 1-2 categories as specialties
- Specialty questions are harder difficulty (+1 level)
- Same time limits as regular questions (no extension)
- Specialty bonus: `constants.SpecialtyPointMultiplier` (2.0x points)
- Players auto-marked ready after specialty selection

**Specialty Question Frequency:**
- Easy Mode: 20% chance per question
- Medium Mode: 30% chance per question
- Hard Mode: 40% chance per question

## Game Phases

### Phase 1: Resource Gathering
**Duration**: Configurable rounds and duration per round
- Default: `constants.ResourceGatheringRounds` rounds (5)
- Default: `constants.ResourceGatheringRoundDuration` seconds per round (60)
- Each round = one trivia question sent to all players
**Location**: Multiple QR code stations in physical spaces
**Participants**: Players only (host monitors)

#### Core Mechanics
**Physical Movement:**
- Players physically move between 4 QR code stations
- Each station corresponds to different token type
- Location verification only required when changing stations
- QR codes contain cryptographic hashes for validation

**Trivia System:**
- One question delivered per round (every `constants.ResourceGatheringRoundDuration` seconds)
- Questions sourced from comprehensive categorized database
- Automatic question cycling prevents repetition
- Enhanced answer validation with fuzzy matching
- All questions have same time limit regardless of specialty status

#### Enhanced Trivia Features
**Question Management:**
- 6 categories × 3 difficulties = 18 question pools
- Automatic pool cycling when exhausted
- Question history tracking prevents immediate repeats
- Support for HTML entity decoding and text normalization

**Answer Validation:**
- Exact match after normalization (case-insensitive, punctuation-removed)
- Abbreviation recognition (USA/United States, UK/United Kingdom)
- Partial matching for complex answers
- Comprehensive logging for debugging and analysis

#### Resource Token System
**Token Types & Effects:**

1. **Anchor Tokens** → Pre-solved Puzzle Pieces
   - Thresholds: `teamTokens / (5 × difficultyModifier)`
   - Effect: Up to 12 of 16 individual puzzle pieces pre-solved
   - Leaves minimum 4 pieces for player to solve
   - Reduces individual workload significantly

2. **Chronos Tokens** → Extended Puzzle Time
   - Thresholds: `teamTokens / (5 × difficultyModifier)`
   - Effect: +20 seconds per threshold to puzzle assembly time
   - Base time: 300 seconds (adjustable by difficulty)
   - Critical for complex puzzles or larger groups

3. **Guide Tokens** → Linear Placement Guidance
   - Multiple thresholds with linear progression
   - Effect: Highlighted area on personal puzzle view
   - First threshold: Large area guidance (quarter of grid)
   - Middle thresholds: Progressively smaller highlighted areas
   - Final threshold: Precise guidance (2 possible positions)
   - Personal puzzle view shows all visible fragments and movement
   - Only applies to player's own fragment positioning

4. **Clarity Tokens** → Image Preview
   - Thresholds: `teamTokens / (5 × difficultyModifier)`
   - Effect: +1 second per threshold of complete image display
   - Shown before puzzle phase begins
   - Helps with spatial understanding and planning

#### Scoring System
**Base Scoring:**
- Correct Answer: `constants.BaseTokensPerCorrectAnswer` (10 tokens)
- Role Bonus: 1.5x multiplier when at matching station
- Specialty Bonus: 2.0x multiplier for specialty questions
- Difficulty Modifier: Applied to final token awards

**Token Distribution:**
- Anchor Station → Anchor Tokens (Janitor role bonus)
- Chronos Station → Chronos Tokens (Tourist role bonus)
- Guide Station → Guide Tokens (Detective role bonus)
- Clarity Station → Clarity Tokens (Art Enthusiast role bonus)

### Phase 2: Puzzle Assembly
**Location**: Large central room (gymnasium recommended)
**Duration**: Base 300 seconds + chronos bonuses + difficulty modifiers
**Participants**: Players solve and collaborate (host monitors)

#### Dynamic Grid System
**Grid Scaling Algorithm:**
```
Player Count → Grid Size → Total Fragments
1-9 players  → 3×3 grid  → 9 fragments
10-16 players → 4×4 grid  → 16 fragments
17-25 players → 5×5 grid  → 25 fragments
26-36 players → 6×6 grid  → 36 fragments
37-49 players → 7×7 grid  → 49 fragments
50-64 players → 8×8 grid  → 64 fragments
```

**Grid Properties:**
- Always maintains perfect square shape
- Each player receives one puzzle segment (16 pieces each)
- Grid positions calculated deterministically
- Supports swapping between any positions

#### Individual Puzzle Solving
**Segment Assignment:**
- Each player receives unique 16-piece puzzle segment
- Segments named: `segment_a1`, `segment_b2`, etc.
- Segment difficulty configurable per game settings
- Pre-solved segments (from anchor tokens) marked as complete

**Solving Process:**
1. Players solve their individual 16-piece puzzle
2. Completion triggers server acknowledgment
3. **Fragment becomes visible, movable, and fills a space on central grid**
4. Guide token hints provided if available
5. Fragment can now be moved by any player on shared grid

**Fragment Visibility:**
- Fragments are invisible until individual puzzle completion
- Only completed fragments appear on player screens and host display
- Pre-solved fragments (from anchor tokens) are immediately visible
- **Personal Puzzle View**: Each player sees all visible fragments with guide highlighting for their own piece

#### Collaborative Grid Assembly
**Fragment Ownership and Movement:**
- **Individual Ownership**: Players can only move their own completed fragment
- **Unassigned Fragments**: All players can move fragments not owned by any player. These fragments are gradually added as players finish their individual fragments
- **Movement Restrictions**: Cannot move other players' active fragments
- **Disconnected Fragments**: Become unassigned and movable by anyone

**Personal Puzzle Display:**
- Each player sees a personal view of the complete puzzle grid
- Shows all currently visible fragments and their real-time positions
- Highlights suggested area for their own fragment (guide token effect)
- Smaller version of what the host displays on main screen
- Guide highlighting becomes more precise with additional token thresholds

**Fragment Movement:**
- **Movement Cooldown**: 1000ms enforced consistently
- **Position Swapping**: Fragments swap positions when collision occurs
- **Ownership Validation**: Server validates movement permissions
- **State Synchronization**: Real-time updates to all players and host
- **Host Display**: Complete puzzle state sent to host for projector/main screen display
- **Host Monitoring**: Live progress tracking with completion percentage and ownership status

**Movement Validation:**
- Grid boundary checking (0 to gridSize-1)
- Cooldown enforcement prevents race conditions
- Position conflict resolution through swapping
- Comprehensive logging for move history

#### Strategic Collaboration System
**Piece Recommendation Protocol:**
```json
{
  "toPlayerId": "target-uuid",
  "fromFragmentId": "sender-fragment",
  "toFragmentId": "target-fragment",
  "suggestedFromPos": {"x": 1, "y": 2},
  "suggestedToPos": {"x": 3, "y": 0}
}
```

**Recommendation Features:**
- Real-time delivery to target player for strategic discussion
- Accept/reject mechanism for coordination
- Only applies to unassigned fragments or strategic suggestions
- Analytics tracking for collaboration metrics
- No custom messaging to maintain game flow focus
- Players must coordinate verbally for their own fragment movements

#### Puzzle Completion Logic
**Victory Conditions (Both Required):**
1. All fragments marked as solved (individual puzzles complete)
2. All fragments positioned at correct grid coordinates

**Completion Checking:**
- Continuous validation after each move
- Immediate game end when conditions met
- Success/failure analytics based on completion
- Time-based failure if puzzle timer expires

### Phase 3: Post-Game Analytics
**Duration**: 60 seconds display time before reset
**Comprehensive Performance Tracking:**

#### Individual Player Analytics
```json
{
  "tokenCollection": {"anchor": 12, "chronos": 8, "guide": 15, "clarity": 10},
  "triviaPerformance": {
    "totalQuestions": 20,
    "correctAnswers": 16,
    "accuracyByCategory": {"general": 0.85, "science": 0.90},
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
```

#### Team Performance Metrics
- **Overall Performance**: Completion rate, total time, team score
- **Collaboration Analysis**: Communication effectiveness, coordination scores
- **Resource Efficiency**: Token distribution, threshold achievements
- **Strategic Analysis**: Recommendation acceptance rates, move efficiency

#### Advanced Scoring Algorithm
```
Individual Score =
  (Correct Answers × 10) +
  (Specialty Bonus × 2) +
  (Completion Bonus: 100) +
  (Speed Bonus: max 300) +
  (Successful Moves × 5) +
  (Recommendations Sent × 3) +
  (Recommendations Accepted × 8)
```

## Difficulty Levels and Modifiers

### Difficulty Settings Impact
**Easy Mode (0.7× / 1.3× / 0.8×):**
- Easier trivia question selection
- 30% more time for all phases
- 20% fewer tokens required for thresholds
- 20% specialty question probability

**Medium Mode (1.0× / 1.0× / 1.0×):**
- Baseline difficulty for all aspects
- Standard time limits and token requirements
- 30% specialty question probability

**Hard Mode (1.4× / 0.7× / 1.3×):**
- Harder trivia questions prioritized
- 30% less time for all phases
- 30% more tokens required for thresholds
- 40% specialty question probability

### Dynamic Scaling Applications
- Trivia question difficulty selection
- Time limits for resource gathering and puzzle phases
- Token threshold calculations for all bonus effects
- Specialty question probability adjustments

## Technical Architecture

### Backend Infrastructure
- **Language**: Go with Gorilla WebSocket
- **Concurrency**: Thread-safe operations with RWMutex
- **Performance**: Connection pooling, efficient broadcasting
- **Scalability**: Support for 4-64 players dynamically

### Communication Protocol
- **WebSocket**: Full-duplex real-time communication
- **Authentication**: UUID-based session management
- **Validation**: Comprehensive input sanitization
- **Error Handling**: Detailed error responses with context

### State Management
- **Game State**: Atomic transitions between phases
- **Player State**: Individual progress and analytics tracking
- **Puzzle State**: Real-time grid synchronization
- **Analytics**: Persistent tracking across reconnections

### Security Features
- **CORS Validation**: Configurable allowed origins
- **Input Validation**: Size limits, format checking, UTF-8 validation
- **Rate Limiting**: Fragment movement cooldown enforcement
- **Privilege Checking**: Host vs player action authorization

## Advanced Features

### Reconnection System
**Player Reconnection:**
- Maintains game state across disconnections during setup and resource gathering
- Restores current phase context (lobby, trivia progress)
- Preserves analytics and progress data
- **No reconnection permitted during puzzle assembly phase**
- Seamless reintegration into active gameplay (when allowed)

**Host Reconnection:**
- Reconnect to same host endpoint with player ID
- Full game state restoration for monitoring
- Continued access to host-specific controls
- No automatic host transfer system

### Question Management System
**Automatic Cycling:**
- Question pools reset when exhausted
- Randomized order prevents predictable patterns
- History tracking prevents immediate repetition
- Support for thousands of questions per category

**Content Management:**
- JSON-based question format with validation
- HTML entity decoding for special characters
- Comprehensive answer normalization
- Category and difficulty organization

### Performance Optimizations
- **Broadcasting**: Efficient message distribution with filtering
- **Memory Management**: Cleanup routines for stale data
- **Connection Monitoring**: Ping/pong heartbeat system
- **Resource Usage**: Optimized data structures and algorithms

## Configuration and Customization

### Server Configuration (constants/game_balance.go)
**Player Limits:**
- Minimum Players: 4 (excluding host)
- Maximum Players: 64 (excluding host)

**Phase Timing:**
- Resource Gathering Rounds: `constants.ResourceGatheringRounds` (default: 5)
- Resource Gathering Round Duration: `constants.ResourceGatheringRoundDuration` (default: 60 seconds)
- One trivia question per round
- Puzzle Base Time: 300 seconds
- Post-Game Analytics: 60 seconds

**Token Economics:**
- Base Tokens Per Answer: 10
- Role Multiplier: 1.5x
- Specialty Multiplier: 2.0x
- Threshold Calculations: 5 tokens per threshold level

**Game Balance:**
- Fragment Movement Cooldown: 1000ms
- Individual Puzzle Pieces: 16 per player
- Answer Timeout: 30 seconds
- Max Specialties Per Player: 2

### Deployment Configuration
**Environment Variables:**
- `ALLOWED_ORIGINS`: CORS configuration
- `ADMIN_TOKEN`: Administrative endpoint access

**Command Line Options:**
- `-env`: Environment (development/staging/production)
- `-port`: Server port (default: 8080)
- `-cert/-key`: HTTPS certificate files
- `-origins`: Allowed CORS origins override
