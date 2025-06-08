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
- **Reconnection**: Can reconnect using same endpoint + assigned token
- **Management**: Only one host allowed per game instance

**Host Benefits:**
- Reliable game progression without dependency on player actions
- Comprehensive game monitoring and control
- Detailed real-time analytics for educational/team-building scenarios
- Ability to pace game according to group needs

### Player System
**Player Connection & Role:**
- **Frontend Endpoint**: `/`
- **WebSocket Endpoint**: `/ws`
- **Capabilities**:
  - Select character roles with resource bonuses
  - Choose trivia specialty categories
  - Answer trivia questions during resource gathering
  - Solve individual puzzle segments privately
  - Collaborate on master puzzle assembly through recommendations
  - Move fragments on shared puzzle grid
- **Requirements**: Host must be connected for game to start
- **Reconnection**: Can reconnect using assigned token

## Authentication System

### Security Model
All communication after initial connection requires authentication:

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

**Validation Features:**
- UUID v4 format validation for tokens
- Comprehensive input validation (8KB message limit)
- UTF-8 text validation with length limits
- Privilege verification (host vs player actions)
- Rate limiting on fragment movements (`constants.FragmentMoveCooldown`)

## Player Setup and Character Selection

### Role Selection (Players Only)
**4 Available Roles:**
1. **Art Enthusiast** → Clarity Token Bonus
2. **Detective** → Guide Token Bonus
3. **Tourist** → Chronos Token Bonus
4. **Janitor** → Anchor Token Bonus

**Role Mechanics:**
- Each role provides bonus collection for specific token type
- Bonus multiplier: `constants.RoleResourceMultiplier`
- Even distribution of roles enforced across players
- Host does not select a role

**Character Distribution Algorithm:**
- Calculates max per role: `(playerCount + 3) / 4`
- Ensures representation of all roles in larger groups
- As players join, more people are allowed to choose each role

### Trivia Specialty Selection (Players Only)
**Available Categories:**
- General Knowledge, Geography, History, Music, Science, Video Games

**Specialty Mechanics:**
- Players select 1 category as their specialty
- Specialty questions are harder difficulty (+1 level)
- Same time limits as regular questions (no extension)
- Specialty bonus: `constants.SpecialtyPointMultiplier`
- Players are immediately marked as ready upon successful specialty selection

**Specialty Question Frequency:**
- Easy Mode: `constants.SpecialtyQFreqEasy`
- Medium Mode: `constants.SpecialtyQFreqMedium`
- Hard Mode: `constants.SpecialtyQFreqHard`

## Game Phases

### Phase 1: Resource Gathering
**Duration**: Configurable rounds and duration per round
- Number of rounds: `constants.ResourceGatheringRounds`
- Round duration: `constants.ResourceGatheringRoundDuration` seconds per round
- Each gathering round = one trivia round = one trivia question sent to all players
- First part of round: Players select their answer from multiple choice options for `constants.TriviaAnswerTime` seconds
- Second part of round: Answers locked in, marked right/wrong, grace period of `constants.TriviaGraceTime` seconds for location changes and team discussion

**Location**: 4 QR code stations in physical spaces

**Participants**: Players only (host monitors)

#### Core Mechanics
**Physical Movement:**
- Players physically move between 4 QR code stations
- Each station corresponds to different token type
- Location verification only required when changing stations
- QR codes' text value is the hash sent to the server for validation
- Station hashes stored as constants: `constants.HashAnchorStation`, `constants.HashChronosStation`, `constants.HashGuideStation`, `constants.HashClarityStation`

**Trivia System:**
- One question delivered per gathering round (every `constants.ResourceGatheringRoundDuration` seconds)
- Questions presented as distinct multiple-choice options
- No fuzzy matching - clear right/wrong based on selected option
- Automatic question cycling prevents repetition
- All questions have same time limit regardless of specialty status

#### Enhanced Trivia Features
**Question Management:**
- 6 categories × 3 difficulties = 18 question pools
- Automatic pool cycling when exhausted
- Question history tracking prevents immediate repeats
- Support for HTML entity decoding and text normalization

**Answer Validation:**
- Multiple-choice selection with clear right/wrong determination
- No fuzzy matching or partial credit
- Answer selection locked and marked correct or incorrect after `constants.TriviaAnswerTime` seconds
- Comprehensive logging for debugging and analysis

#### Resource Token System
**Token Types & Effects:**

1. **Anchor Tokens** → Pre-solved Individual Puzzle Pieces
   - 6 thresholds: `teamAnchorTokens / constants.AnchorTokenThreshold`
   - Effect: Each threshold pre-solves 2 pieces of the 16-piece individual puzzle
   - Maximum 12 pieces pre-solved (6 thresholds × 2 pieces)
   - Pre-solved pieces are visually locked and unmovable
   - Only affects individual puzzle solving, NOT the central grid
   - Leaves minimum 4 pieces for manual solving

2. **Chronos Tokens** → Extended Puzzle Time
   - 6 thresholds: `teamChronosTokens / constants.ChronosTokenThreshold`
   - Effect: +20 seconds per threshold to puzzle assembly time
   - Maximum +120 seconds (6 thresholds × 20 seconds)
   - Base time: `constants.PuzzleBaseTime` seconds
   - Team-wide benefit applied to entire puzzle phase

3. **Guide Tokens** → Fragment Placement Guidance on Central Grid
   - 6 thresholds: `teamGuideTokens / constants.GuideTokenThreshold`
   - Effect: Highlights possible positions for player's fragment on central grid on player's personal device
   - Each threshold removes (gridSize × gridSize) / 7 highlighted squares
   - Progression from many possible positions to precise guidance
   - Individual hints visible only to each player for their own fragment
   - Only applies after individual puzzle completion

4. **Clarity Tokens** → Complete Image Preview
   - 6 thresholds: `teamClarityTokens / constants.ClarityTokenThreshold`
   - Effect: +1 second per threshold of complete image display
   - Maximum 6 seconds additional preview time
   - Base preview time: `constants.ClarityBasePreviewTime` seconds
   - Shown automatically at puzzle phase start
   - Helps with spatial understanding and planning

#### Scoring System
**Base Scoring:**
- Correct Answer: `constants.BaseTokensPerCorrectAnswer` tokens
- Role Bonus: `constants.RoleResourceMultiplier` when at matching station
- Specialty Bonus: `constants.SpecialtyPointMultiplier` for specialty questions
- Difficulty Modifier: Applied to final token awards

**Token Distribution:**
- Anchor Station → Anchor Tokens (Janitor role bonus)
- Chronos Station → Chronos Tokens (Tourist role bonus)
- Guide Station → Guide Tokens (Detective role bonus)
- Clarity Station → Clarity Tokens (Art Enthusiast role bonus)

### Phase 2: Puzzle Assembly
**Location**: Large central room (gymnasium recommended)

**Duration**: `constants.PuzzleBaseTime` seconds + chronos bonuses + difficulty modifiers

**Participants**: Players solve and collaborate (host monitors + shows big central grid for phase 3)

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
- **Assignment**: Each player receives exactly one unique segment ID (e.g., `segment_a5`, `segment_b2`)
- **Client Responsibilities**: 
  - Load segment image using provided ID
  - Split segment into 16 individual jigsaw pieces
  - Shuffle pieces randomly for puzzle challenge
  - Choose which pieces to pre-solve based on anchor token count
  - Mark pre-solved pieces as locked and unmovable
  - Handle all piece movement and swapping logic
  - Validate when puzzle is correctly assembled
- **Solving Process**: Players arrange these 16 pieces into the correct configuration privately
- **Pre-solving Effects**: Anchor tokens provide count of pieces to pre-solve (up to 12), client chooses which pieces
- **No Interaction**: Other players cannot see, help with, or influence individual puzzle progress
- **Host Blindness**: Host cannot monitor or view individual puzzle progress in real-time


### System 2: Central Shared Puzzle Grid (Public & Collaborative)

**Collaborative Space Characteristics:**
- **Full Visibility**: All activities visible to all players and host in real-time
- **Fragment-Based**: Operates with completed puzzle fragments of central puzzle, not individual pieces of players' segments
- **Post-Completion Only**: Only becomes populated after individual puzzle completions
- **Shared Control**: Players can move fragments collaboratively within ownership rules
- **Real-Time Updates**: All movements are broadcast to all participants periodically

**Central Grid Mechanics:**
- **Dynamic Scaling**: Grid size automatically scales with player count
- **Fragment Creation**: Each completed individual puzzle becomes one movable fragment
- **Unassigned Fragments**: If more fragments in central puzzle than players, unassigned fragments appear gradually as players finish individual fragments
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
   - Server places fragment at random unoccupied position on central shared grid
   - Fragment becomes visible to all players and host immediately
   - Fragment becomes movable according to ownership rules
   - Player transitions from Phase 2 to Phase 3

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
- Supports position swapping between any fragments
- As segments enter play they are placed on random open tiles of the grid

#### Fragment Ownership and Movement System

**Ownership Categories:**
1. **Player-Owned Fragments**: Created when player completes individual puzzle
   - Only the creating player can move their own fragment
   - Clearly identified with player ID in fragment data
   - Maintains ownership until game completion or disconnection

2. **Unassigned Fragments**: Fragments never assigned to a player or fragments assigned to a player that disconnects
   - Any player can move these fragments
   - No specific ownership restrictions
   - Marked as `playerId: null` in system

**Movement Mechanics (Switches/Swaps):**
- **Movement Cooldown**: `constants.FragmentMoveCooldown` ms enforced consistently for swapped pieces
- **Terminology**: Also called fragment move requests, piece recommendations, or switch requests
- **Position Validation**: All swaps validated against grid boundaries (0 to gridSize-1)
- **Collision Resolution**: Fragments swap positions or one fragment moves to open grid space
- **Permission Checking**: Server validates ownership before allowing movement
- **State Synchronization**: 
  - Host: Immediate updates on all movements
  - Players: Updates every `constants.GridUpdateInterval` seconds

#### Fragment Visibility and State Management

**Visibility Rules:**
- **Invisible Until Completion**: Fragments only become visible after individual puzzle completion
- **Immediate Visibility**: Once visible, fragments remain visible to all players and host
- **Pre-Solved Visibility**: Anchor token pre-solved fragments are immediately visible at game start
- **Personal View Consistency**: Each player sees identical central grid state

**State Broadcasting:**
- **Central Puzzle State**: Complete grid state sent to all players every `constants.GridUpdateInterval` seconds
- **Host Updates**: Receives immediate updates on all fragment movements and state changes
- **Personal Puzzle State**: Individual view with guide highlighting (from guide tokens)
- **Update Frequency**: 
  - Players: Periodic updates every `constants.GridUpdateInterval` seconds (default 3s)
  - Host: Immediate updates on all changes
- **Phase Tracking**: Server implicitly tracks each player as Phase 2 (individual) or Phase 3 (collaborative)

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
- **Strategic Communication**: Players can suggest optimal fragment switches between a segment they control (their own or unassigned) and any other fragment
- **Accept/Reject Mechanism**: If the other fragment is controlled by another player (not unassigned) the other player chooses whether to allow/reject the suggested switch
- **Analytics Tracking**: All recommendations tracked for collaboration scoring
- **No Auto-Execution**: Recommendations require explicit acceptance to take effect
- **Verbal Coordination**: Players encouraged to communicate during collaboration

#### Token Effects in Puzzle Phase

**Guide Token Implementation:**
- **Central Grid Highlighting**: Shows highlighted squares on central grid where player's fragment should go
- **Progressive Precision**: Each threshold removes (gridSize²) / 7 possible positions
- **Individual View**: Each player sees highlights only for their own fragment
- **Always Active**: Highlights visible throughout puzzle phase after individual completion (phase 3)
- **Public vs Private**: Fragment positions public on host screen, highlights private to player

**Anchor Token Pre-Solving:**
- **Individual Puzzle Only**: Pre-solves pieces in 16-piece individual puzzles
- **Visual Lock**: Pre-solved pieces marked as locked and unmovable
- **No Central Grid Effect**: Does NOT pre-place fragments on central grid
- **Progressive Unlock**: 2 pieces pre-solved per threshold (max 12 pieces)
- **Balanced Challenge**: Minimum 4 pieces always require manual solving

**Chronos Token Time Extension:**
- **Base Time**: `constants.PuzzleBaseTime` seconds for puzzle assembly phase
- **Threshold Bonuses**: +20 seconds per threshold level achieved
- **Total Time Calculation**: Base + (thresholds × 20) + difficulty modifiers
- **Team Benefit**: Extended time applies to entire team collaboration period

**Clarity Token Preview:**
- **Automatic Display**: Shows complete image automatically at puzzle phase start
- **Duration Calculation**: `constants.ClarityBasePreviewTime` + (thresholds × 1) seconds
- **Maximum Preview**: Up to 6 seconds additional preview time
- **Strategic Value**: Helps players understand spatial relationships before solving

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
- **Ownership Transfer**: Fragment becomes movable by any remaining player (becomes an unassigned fragment)
- **No Reconnection**: No reconnection permitted during puzzle assembly phase

**Player Disconnection During Collaboration:**
- **Fragment Status Change**: Player's fragment becomes unassigned immediately
- **Movement Permission**: Any player can now move the disconnected player's fragment
- **State Broadcasting**: Disconnection status broadcast to all remaining players
- **No Reconnection**: No reconnection permitted during puzzle assembly phase

**Host Disconnection:**
- **During Setup/Resource Phases**: Game pauses until host reconnects
- **During Puzzle Phase**: Game continues without interruption
- **Player Notification**: All players notified of host disconnection
- **State Preservation**: Complete game state maintained for host reconnection
- **Automatic Recovery**: Host can reconnect to resume monitoring

### Phase 3: Post-Game Analytics
**Duration**: Game result and all analytics display until the host manually resets the game
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
**Easy Mode:**
- Easier trivia question selection
- Time multiplier: `constants.EasyTimeMultiplier`
- Token threshold multiplier: `constants.EasyThresholdMultiplier`
- Specialty question probability: `constants.EasySpecialtyProbability`

**Medium Mode:**
- Baseline difficulty for all aspects
- Time multiplier: `constants.MediumTimeMultiplier`
- Token threshold multiplier: `constants.MediumThresholdMultiplier`
- Specialty question probability: `constants.MediumSpecialtyProbability`

**Hard Mode:**
- Harder trivia questions prioritized
- Time multiplier: `constants.HardTimeMultiplier`
- Token threshold multiplier: `constants.HardThresholdMultiplier`
- Specialty question probability: `constants.HardSpecialtyProbability`

### Dynamic Scaling Applications
- Trivia question difficulty selection
- Time limits for resource gathering and puzzle phases
- Token threshold calculations for all bonus effects
- Specialty question probability adjustments

## WebSocket Events Outline

### Phase 0
1. **
### Phase 1


**Individual Puzzle Workflow:**
1. **Phase Start**: Server sends host initial grid configuration showing all empty squares
2. **Segment Assignment**: Player receives `puzzle_phase_load` with their unique `segmentId`, total number of central puzzle segments, and number of pieces to pre-solve due to accrued anchor tokens.
3. **Client Processing**: Client loads their segment, splits into 16 pieces, shuffles, and applies pre-solving
4. **Private Solving**: Player works on 16-piece puzzle completely invisibly (Phase 2)
5. **Completion Trigger**: Player completes arrangement and sends `segment_completed` message
6. **Server Processing**: Server places completed segment in random unoccupied grid square of the master puzzle
7. **Phase Transition**: Player receives updated grid state and transitions to Phase 3 (collaborative solving)
8. **Periodic Updates**: Players receive grid updates every `constants.GridUpdateInterval` seconds

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
- Resource Gathering Rounds: `constants.ResourceGatheringRounds`
- Resource Gathering Round Duration: `constants.ResourceGatheringRoundDuration` seconds
- One trivia question per gathering round
- Puzzle Base Time: `constants.PuzzleBaseTime` seconds
- Post-Game Analytics: `constants.PostGameDuration` seconds

**Token Economics:**
- Base Tokens Per Answer: `constants.BaseTokensPerCorrectAnswer`
- Role Multiplier: `constants.RoleResourceMultiplier`
- Specialty Multiplier: `constants.SpecialtyPointMultiplier`
- Anchor Token Threshold: `constants.AnchorTokenThreshold`
- Chronos Token Threshold: `constants.ChronosTokenThreshold`
- Guide Token Threshold: `constants.GuideTokenThreshold`
- Clarity Token Threshold: `constants.ClarityTokenThreshold`

**Game Balance:**
- Fragment Movement Cooldown: `constants.FragmentMoveCooldown` ms
- Individual Puzzle Pieces: `constants.IndividualPuzzlePieces` per player
- Answer Selection Time: `constants.TriviaAnswerTime`
- Grace Period Time: `constants.TriviaGraceTime`
- Max Specialties Per Player: `constants.MaxSpecialtiesPerPlayer`
- Grid Update Interval: `constants.GridUpdateInterval` seconds (default 3s)

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
