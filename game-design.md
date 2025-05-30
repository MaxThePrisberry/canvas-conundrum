# Canvas Conundrum - Comprehensive Game Design

## Game Concept
A collaborative puzzle-solving game where players recover a stolen masterpiece by gathering resources, answering trivia, and assembling a fragmented artwork. The game uses a dedicated host system for reliable game management and real-time coordination.

## Host vs Player System

### Host System
Canvas Conundrum uses a dedicated host model for reliable game management:

**Host Connection & Role:**
- **Frontend Endpoint**: `/host` (web interface for hosts to enter UUID and connect)
- **WebSocket Endpoint**: `/ws/host/{unique-uuid}` (UUID generated fresh each server start)
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
  - Solve individual puzzle segments privately
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

## CRITICAL: Dual-Puzzle System Architecture

**FUNDAMENTAL DESIGN PRINCIPLE**: Canvas Conundrum operates with two completely independent puzzle systems that remain entirely separate until a specific transition moment. Understanding this separation is crucial for proper implementation.

### System 1: Individual Player Puzzles (Private & Invisible)

**Complete Isolation Characteristics:**
- **Zero Visibility**: Individual puzzle work is completely invisible to all other players, the host, and any shared displays
- **No Grid Connection**: Individual puzzles have absolutely no relationship to the central shared puzzle grid
- **No Space Reservation**: No position, placeholder, or reservation exists on the central grid during individual solving
- **Isolated Processing**: Individual puzzle state is processed completely separately from shared game state
- **Private Workspace**: Each player works in their own private puzzle-solving environment

**Individual Puzzle Mechanics:**
- **Assignment**: Each player receives exactly one unique 16-piece puzzle segment (e.g., `segment_a5`, `segment_b2`)
- **Content**: Each segment contains 16 individual jigsaw pieces that form part of the larger artwork
- **Solving Process**: Players arrange these 16 pieces into the correct configuration privately
- **Pre-solving Effects**: Anchor tokens can pre-solve up to 12 of these 16 pieces, leaving minimum 4 for manual solving
- **No Interaction**: Other players cannot see, help with, or influence individual puzzle progress
- **Host Blindness**: Host cannot monitor or view individual puzzle progress in real-time

**Individual Puzzle Workflow:**
1. **Phase Start**: Player receives `puzzle_phase_load` with their unique `segmentId`
2. **Private Solving**: Player works on 16-piece puzzle completely invisibly
3. **No Broadcasting**: No progress updates sent to other players or host
4. **Completion Trigger**: Player completes arrangement and sends `segment_completed` message
5. **Transformation**: Individual puzzle immediately converts to central grid fragment

### System 2: Central Shared Puzzle Grid (Public & Collaborative)

**Collaborative Space Characteristics:**
- **Full Visibility**: All activities visible to all players and host in real-time
- **Fragment-Based**: Operates with completed puzzle fragments, not individual pieces
- **Post-Completion Only**: Only becomes populated after individual puzzle completions
- **Shared Control**: Players can move fragments collaboratively within ownership rules
- **Real-Time Updates**: All movements immediately broadcast to all participants

**Central Grid Mechanics:**
- **Dynamic Scaling**: Grid size automatically scales with player count
- **Fragment Creation**: Each completed individual puzzle becomes one movable fragment
- **Position Assignment**: Fragments appear at predetermined grid coordinates
- **Movement Rules**: Players can move their own fragments and unassigned fragments
- **Collaboration Features**: Recommendation system for strategic coordination

### The Critical Transformation Moment

**Individual Completion → Central Fragment Activation:**

This is the single most important transition in the entire game system:

1. **Pre-Completion State**:
   - Individual puzzle exists only in player's private space
   - Central grid shows no trace of this puzzle
   - Other players see no indication of progress
   - Host displays show no fragment for this player

2. **Completion Trigger**:
   - Player arranges final pieces of their 16-piece puzzle
   - Player sends `segment_completed` message with `segmentId` and timestamp
   - Server validates completion and assigns grid position

3. **Instant Transformation**:
   - Individual 16-piece puzzle instantly becomes one single fragment
   - Fragment appears at designated position on central shared grid
   - Fragment becomes visible to all players and host immediately
   - Fragment becomes movable according to ownership rules

4. **Post-Completion State**:
   - Individual puzzle workspace no longer exists for that player
   - Player now participates only in central grid collaboration
   - Fragment participates in shared puzzle assembly process

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
- Each player's completed individual puzzle becomes exactly one fragment
- Grid positions calculated deterministically
- Supports position swapping between any fragments

#### Fragment Ownership and Movement System

**Ownership Categories:**
1. **Player-Owned Fragments**: Created when player completes individual puzzle
   - Only the creating player can move their own fragment
   - Clearly identified with player ID in fragment data
   - Maintains ownership until game completion or disconnection

2. **Unassigned Fragments**: Pre-solved by anchor tokens or from disconnected players
   - Any player can move these fragments
   - No specific ownership restrictions
   - Marked as `playerId: null` in system

3. **Disconnected Player Fragments**: Auto-solved and become unassigned
   - Immediately converted to unassigned status
   - Can be moved by any remaining player
   - Maintains correct solution but loses ownership

**Movement Mechanics:**
- **Movement Cooldown**: 1000ms enforced consistently across all fragment types
- **Position Validation**: All moves validated against grid boundaries (0 to gridSize-1)
- **Collision Resolution**: Fragments swap positions when movement causes collision
- **Permission Checking**: Server validates ownership before allowing movement
- **State Synchronization**: All movements immediately broadcast to all participants

#### Fragment Visibility and State Management

**Visibility Rules:**
- **Invisible Until Completion**: Fragments only become visible after individual puzzle completion
- **Immediate Visibility**: Once visible, fragments remain visible to all players and host
- **Pre-Solved Visibility**: Anchor token pre-solved fragments are immediately visible at game start
- **Personal View Consistency**: Each player sees identical central grid state

**State Broadcasting:**
- **Central Puzzle State**: Complete grid state sent to all players and host
- **Personal Puzzle State**: Individual view with guide highlighting (guide tokens only)
- **Host Monitoring**: Comprehensive view including fragment ownership and movement history
- **Real-Time Updates**: State changes broadcast immediately upon fragment movement

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
- **Strategic Communication**: Players can suggest optimal fragment placements
- **Accept/Reject Mechanism**: Target player chooses whether to follow suggestions
- **Analytics Tracking**: All recommendations tracked for collaboration scoring
- **No Auto-Execution**: Recommendations require explicit acceptance to take effect
- **Verbal Coordination**: Players encouraged to communicate during collaboration

#### Token Effects in Puzzle Phase

**Guide Token Implementation:**
- **Personal Highlighting**: Shows highlighted area on player's personal puzzle view
- **Progressive Precision**: Linear progression from large area to 2-position precision
- **Own Fragment Only**: Guidance applies only to player's own fragment positioning
- **Threshold Levels**: Multiple thresholds provide increasingly precise guidance
- **Visual Integration**: Highlighting overlays on personal puzzle grid view

**Anchor Token Pre-Solving:**
- **Individual Puzzle Pre-Solving**: Up to 12 of 16 pieces in individual puzzles pre-solved
- **Central Grid Pre-Population**: Some fragments appear immediately as unassigned
- **Reduced Workload**: Players solve fewer individual pieces before grid participation
- **Balanced Challenge**: Minimum 4 pieces always require manual solving

**Chronos Token Time Extension:**
- **Base Time**: 300 seconds for puzzle assembly phase
- **Threshold Bonuses**: +20 seconds per threshold level achieved
- **Total Time Calculation**: Base + (thresholds × 20) + difficulty modifiers
- **Shared Benefit**: Extended time applies to entire team collaboration period

**Clarity Token Preview:**
- **Complete Image Display**: Shows full assembled artwork before puzzle phase
- **Duration Calculation**: Base 3 seconds + 1 second per threshold level
- **Strategic Value**: Helps players understand spatial relationships and planning
- **Timing**: Displayed immediately before puzzle phase begins

#### Puzzle Completion Logic

**Victory Conditions (Both Required):**
1. **All Fragments Present**: Every player's individual puzzle completed and converted to fragment
2. **Correct Positioning**: All fragments positioned at their designated correct grid coordinates

**Completion Validation:**
- **Continuous Checking**: Server validates completion after every fragment movement
- **Immediate Resolution**: Game ends instantly when both conditions satisfied
- **Success Analytics**: Comprehensive performance tracking for successful completion
- **Failure Handling**: Time-based failure if puzzle timer expires before completion

#### Disconnection and Error Handling

**Player Disconnection During Individual Solving:**
- **Auto-Solve Trigger**: Disconnected player's individual puzzle immediately auto-solved
- **Fragment Creation**: Auto-solved puzzle converts to unassigned fragment on central grid
- **Random Placement**: Fragment placed at random grid position to maintain puzzle integrity
- **Ownership Transfer**: Fragment becomes movable by any remaining player

**Player Disconnection During Collaboration:**
- **Fragment Status Change**: Player's fragment becomes unassigned immediately
- **Movement Permission**: Any player can now move the disconnected player's fragment
- **State Broadcasting**: Disconnection status broadcast to all remaining players
- **No Reconnection**: No reconnection permitted during puzzle assembly phase

**Host Disconnection:**
- **Game Pause**: Puzzle timer pauses until host reconnects
- **Player Notification**: All players notified of host disconnection
- **State Preservation**: Complete game state maintained for host reconnection
- **Automatic Recovery**: Game resumes when host reconnects to host endpoint

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
- **Puzzle State**: Real-time grid synchronization with dual-system architecture
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

---

## Implementation Summary: Individual vs Central Puzzle System

**Critical Points for Development:**

### Complete System Separation
1. **Individual Puzzles**: Completely private, invisible, no grid interaction
2. **Central Grid**: Collaborative space for completed fragments only
3. **Zero Overlap**: No shared state between systems until completion trigger
4. **Instant Transformation**: Individual completion immediately creates central fragment

### State Management Requirements
1. **Dual State Tracking**: Separate tracking systems for individual and central puzzles
2. **Visibility Control**: Strict enforcement of individual puzzle invisibility
3. **Transition Handling**: Reliable conversion from individual to central fragment
4. **Broadcasting Logic**: Different message types for individual vs collaborative phases

### User Experience Design
1. **Clear Phase Distinction**: Players understand when they're in individual vs collaborative mode
2. **Smooth Transition**: Seamless experience when individual puzzle becomes collaborative fragment
3. **Visual Feedback**: Clear indicators of individual progress vs central grid participation
4. **Collaborative Focus**: Central grid emphasizes teamwork and strategic coordination

This dual-system architecture is fundamental to Canvas Conundrum's unique gameplay experience, ensuring both individual contribution and collaborative problem-solving while maintaining clear separation between private work and shared coordination.
