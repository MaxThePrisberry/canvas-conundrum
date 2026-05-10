# Canvas Conundrum - Game Design Document

## Game Concept
A collaborative puzzle-solving game where players recover a stolen masterpiece by gathering resources, answering trivia, and assembling a fragmented artwork. The game uses a dedicated host system for reliable game management and real-time coordination.

## Terminology
- **Piece**: One of the 16 sub-tiles a player manipulates inside their *own* individual puzzle (Phase 2A). Pieces are private and never appear on the central grid.
- **Segment**: The image one player is assigned to solve. A segment, once assembled by its owner, becomes a single **fragment** on the central grid.
- **Fragment**: A completed segment occupying one cell of the central shared grid (Phase 2B). Fragments are public and can be moved within ownership rules.
- **Central grid**: The N×N shared board where fragments are arranged to reconstruct the full image.

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

### Player System
**Player Connection & Role:**
- **Frontend Endpoint**: `/`
- **WebSocket Endpoint**: `/ws`
- **Capabilities**:
  - Select character roles with resource bonuses
  - Choose trivia specialty category
  - Answer trivia questions during resource gathering
  - Solve individual puzzle segments privately
  - Collaborate on master puzzle assembly through recommendations
  - Move fragments on shared puzzle grid
- **Requirements**: Host must be connected for game to start
- **Reconnection**: Can reconnect using assigned token

## Authentication System

### Security Model
All communication after initial connection requires authentication using the standardized message format:

```json
{
  "event": "EVENT_NAME",
  "auth": {
    "token": "uuid-generated-by-server"
  },
  "payload": {
    // Event-specific data
  },
  "timestamp": "2025-01-XX:XX:XX.XXXZ"
}
```

**Validation Features:**
- UUID v4 format validation for tokens
- Comprehensive input validation (8KB message limit)
- UTF-8 text validation with length limits
- Privilege verification (host vs player actions)
- Rate limiting on fragment movements (`gameConfig.fragmentMoveCooldown`)

## Player Setup and Character Selection

### Role Selection (Players Only)
**4 Available Roles:**
1. **Art Enthusiast** → Clarity Token Bonus
2. **Detective** → Guide Token Bonus
3. **Tourist** → Chronos Token Bonus
4. **Janitor** → Anchor Token Bonus

**Role Mechanics:**
- Each role provides bonus collection for specific token type
- Bonus multiplier: `gameConfig.roleResourceMultiplier`
- Even distribution of roles enforced across players
- Host does not select a role

**Character Distribution Algorithm:**
- Calculates max per role: `max(1, (playerCount + 3) / 4)`
- Ensures representation of all roles in larger groups
- As players join, more people are allowed to choose each role
- Minimum of 1 player per role to ensure all roles are always available with small player counts

### Trivia Specialty Selection (Players Only)
**Available Categories:**
- General Knowledge, Geography, History, Music, Science, Video Games

**Specialty Mechanics:**
- Players select 1 category as their specialty
- Specialty questions are harder difficulty (+1 level)
- Same time limits as regular questions (no extension)
- Specialty bonus: `gameConfig.specialtyPointMultiplier`
- Players are immediately marked as ready upon successful specialty selection

**Specialty Question Frequency:**
- Easy Mode: `gameConfig.easySpecialtyProbability`
- Medium Mode: `gameConfig.mediumSpecialtyProbability`
- Hard Mode: `gameConfig.hardSpecialtyProbability`

## Game Phases

### Phase 0: Setup and Connection
Players connect, select roles and specialties, host monitors readiness and starts the game when minimum players are ready.

### Phase 1: Resource Gathering
**Duration**: Configurable rounds and duration per round
- Number of rounds: `gameConfig.resourceGatheringRounds`
- Round duration: `gameConfig.resourceGatheringRoundDuration` seconds per round
- Each gathering round = one trivia round = one trivia question sent to all players
- First part of round: Players select their answer from multiple choice options for `gameConfig.triviaAnswerTime` seconds
- Second part of round: Answers locked in, marked right/wrong, grace period of `gameConfig.triviaGraceTime` seconds for location changes and team discussion

**Location**: 4 QR code stations in physical spaces

**Participants**: Players only (host monitors)

#### Core Mechanics
**Physical Movement:**
- Players physically move between 4 QR code stations
- Each station corresponds to different token type
- Location verification only required when changing stations
- QR codes' text value is the hash sent to the server for validation
- Station hashes stored as constants: `gameConfig.stationHashes.anchor`, `gameConfig.stationHashes.chronos`, `gameConfig.stationHashes.guide`, `gameConfig.stationHashes.clarity`

**Trivia System:**
- One question delivered per gathering round (every `gameConfig.resourceGatheringRoundDuration` seconds)
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
- Answer selection locked and marked correct or incorrect after `gameConfig.triviaAnswerTime` seconds
- Comprehensive logging for debugging and analysis

#### Resource Token System
**Token Types & Effects:**

1. **Anchor Tokens** → Pre-solved Individual Puzzle Pieces
   - 6 thresholds: `teamAnchorTokens / gameConfig.anchorTokenThreshold`
   - Effect: Each threshold pre-solves 2 pieces of the 16-piece individual puzzle
   - Maximum 12 pieces pre-solved (6 thresholds × 2 pieces)
   - Pre-solved pieces are visually locked and unmovable
   - Only affects individual puzzle solving, NOT the central grid
   - Leaves minimum 4 pieces for manual solving

2. **Chronos Tokens** → Extended Puzzle Time
   - 6 thresholds: `teamChronosTokens / gameConfig.chronosTokenThreshold`
   - Effect: +20 seconds per threshold to puzzle assembly time
   - Maximum +120 seconds (6 thresholds × 20 seconds)
   - Base time: `gameConfig.puzzleBaseTime` seconds
   - Team-wide benefit applied to entire puzzle phase

3. **Guide Tokens** → Fragment Placement Guidance on Central Grid
   - 6 thresholds: `teamGuideTokens / gameConfig.guideTokenThreshold`
   - Effect: Highlights possible positions for player's fragment on central grid on player's personal device
   - Each threshold removes (gridSize × gridSize) / 7 highlighted squares
   - Progression from many possible positions to precise guidance
   - Individual hints visible only to each player for their own fragment
   - Only applies after individual puzzle completion

4. **Clarity Tokens** → Complete Image Preview
   - 6 thresholds: `teamClarityTokens / gameConfig.clarityTokenThreshold`
   - Effect: +1 second per threshold of complete image display
   - Maximum 6 seconds additional preview time
   - Base preview time: `gameConfig.clarityBasePreviewTime` seconds
   - Shown automatically at puzzle phase start
   - Helps with spatial understanding and planning

#### Scoring System
**Base Scoring:**
- Correct Answer: `gameConfig.baseTokensPerCorrectAnswer` tokens
- Role Bonus: `gameConfig.roleResourceMultiplier` when at matching station
- Specialty Bonus: `gameConfig.specialtyPointMultiplier` for specialty questions
- Difficulty Modifier: Applied to final token awards

**Token Distribution:**
- Anchor Station → Anchor Tokens (Janitor role bonus)
- Chronos Station → Chronos Tokens (Tourist role bonus)
- Guide Station → Guide Tokens (Detective role bonus)
- Clarity Station → Clarity Tokens (Art Enthusiast role bonus)

### Phase 2: Puzzle Assembly
**Location**: Large central room (gymnasium recommended)

**Duration**: `gameConfig.puzzleBaseTime` seconds + chronos bonuses + difficulty modifiers

**Participants**: Players solve and collaborate (host monitors + shows big central grid for phase 2B)

## Dual-Puzzle System Architecture

Canvas Conundrum operates with two completely independent puzzle systems that remain entirely separate until a specific transition moment. The separation is the core of the game's design.

### System 1: Individual Player Puzzles (Private & Invisible) - Phase 2A

**Isolation Rules:**
- Each player solves their assigned segment in a private workspace. Their progress is invisible to other players, the host, and any shared display.
- No fragment, position, or placeholder exists on the central grid for an in-progress segment. Fragments only appear there after their owner completes the individual puzzle.

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

### System 2: Central Shared Puzzle Grid (Public & Collaborative) - Phase 2B

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
   - Player sends `PUZZLE_TO_SERVER_SEGMENT_COMPLETED` message with `segmentId` and timestamp
   - Server validates completion and assigns grid position

3. **Instant Transformation**:
   - Individual 16-piece puzzle instantly becomes one single fragment
   - Server places fragment at random unoccupied position on central shared grid
   - Fragment becomes visible to all players and host immediately
   - Fragment becomes movable according to ownership rules
   - Player transitions from Phase 2A to Phase 2B

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
- **Movement Cooldown**: `gameConfig.fragmentMoveCooldown` ms enforced consistently for swapped pieces
- **Terminology**: Also called fragment move requests, piece recommendations, or switch requests
- **Position Validation**: All swaps validated against grid boundaries (0 to gridSize-1)
- **Collision Resolution**: Fragments swap positions or one fragment moves to open grid space
- **Permission Checking**: Server validates ownership before allowing movement
- **State Synchronization**:
  - Host: Immediate updates on all movements
  - Players: Updates every `gameConfig.gridUpdateInterval` seconds

#### Fragment Visibility and State Management

**Visibility Rules:**
- **Invisible Until Completion**: Fragments only become visible after individual puzzle completion
- **Immediate Visibility**: Once visible, fragments remain visible to all players and host
- **Pre-Solved Visibility**: Anchor token pre-solved fragments are immediately visible at game start
- **Personal View Consistency**: Each player sees identical central grid state

**State Broadcasting:**
- **Central Puzzle State**: Complete grid state sent to all players every `gameConfig.gridUpdateInterval` seconds
- **Host Updates**: Receives immediate updates on all fragment movements and state changes
- **Personal Puzzle State**: Individual view with guide highlighting (from guide tokens)
- **Update Frequency**:
  - Players: Periodic updates every `gameConfig.gridUpdateInterval` seconds (default 3s)
  - Host: Immediate updates on all changes
- **Phase Tracking**: Server implicitly tracks each player as Phase 2A (individual) or Phase 2B (collaborative)

#### Strategic Collaboration System

**Piece Recommendation Protocol:**
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
- **Always Active**: Highlights visible throughout puzzle phase after individual completion (phase 2B)
- **Public vs Private**: Fragment positions public on host screen, highlights private to player

**Anchor Token Pre-Solving:**
- **Individual Puzzle Only**: Pre-solves pieces in 16-piece individual puzzles
- **Visual Lock**: Pre-solved pieces marked as locked and unmovable
- **No Central Grid Effect**: Does NOT pre-place fragments on central grid
- **Progressive Unlock**: 2 pieces pre-solved per threshold (max 12 pieces)
- **Balanced Challenge**: Minimum 4 pieces always require manual solving

**Chronos Token Time Extension:**
- **Base Time**: `gameConfig.puzzleBaseTime` seconds for puzzle assembly phase
- **Threshold Bonuses**: +20 seconds per threshold level achieved
- **Total Time Calculation**: Base + (thresholds × 20) + difficulty modifiers
- **Team Benefit**: Extended time applies to entire team collaboration period

**Clarity Token Preview:**
- **Automatic Display**: Shows complete image automatically at puzzle phase start
- **Duration Calculation**: `gameConfig.clarityBasePreviewTime` + (thresholds × 1) seconds
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

## Phase-Specific Disconnection Rules

Canvas Conundrum handles disconnections differently based on the game phase to balance game integrity with player experience.

### Setup Phase (Phase 0) Disconnections

**Player Disconnection:**
- **Immediate Removal**: Player removed from all counts (connected, ready, role distribution)
- **State Preservation**: Player's selection data (role, specialty, name) preserved for potential reconnection
- **No Game Impact**: Disconnection does not affect other players or game progression
- **Reconnection Behavior**:
  - Player can reconnect and restore previous selections if role still available
  - If selected role has filled up since disconnection, player must reselect role
  - All other selections (specialty, name) restored automatically
  - Player automatically marked ready if role available and they were ready before disconnection

**Host Disconnection:**
- **Game Pause**: Game cannot progress until host reconnects
- **State Preservation**: Complete setup state maintained
- **Player Notification**: All players notified of host disconnection
- **Automatic Recovery**: Host can reconnect to resume control

### Post-Setup Phases (Phases 1-3) Disconnections

**Player Disconnection:**
- **Maintained Presence**: Player remains in game counts and their contributions continue to matter
- **Resource Phase**: Collected tokens remain in team totals
- **Puzzle Phase**: Individual puzzle auto-solved, fragment becomes unassigned for team use
- **Analytics Phase**: Personal analytics preserved for viewing upon reconnection
- **Limited Reconnection**: Cannot reconnect during puzzle assembly phase; can reconnect during resource gathering and analytics phases

**Host Disconnection:**
- **Game Continuation**: Game continues without interruption during puzzle phase
- **Monitoring Loss**: Real-time host monitoring temporarily unavailable
- **State Preservation**: Complete game state maintained for host reconnection
- **Automatic Recovery**: Host can reconnect to resume monitoring at any time

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

### Phase 3: Post-Game Analytics
**Duration**: Game result and all analytics display until the host manually resets the game
**Comprehensive Performance Tracking:**

#### Individual Player Analytics
- **Token Collection**: Total tokens by type, role bonuses earned
- **Trivia Performance**: Accuracy by category, specialty bonus points
- **Puzzle Solving Metrics**: Individual solve time, fragment moves, recommendations sent/received/accepted

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
- Time multiplier: `gameConfig.easyTimeMultiplier`
- Token threshold multiplier: `gameConfig.easyThresholdMultiplier`
- Specialty question probability: `gameConfig.easySpecialtyProbability`

**Medium Mode:**
- Baseline difficulty for all aspects
- Time multiplier: `gameConfig.mediumTimeMultiplier`
- Token threshold multiplier: `gameConfig.mediumThresholdMultiplier`
- Specialty question probability: `gameConfig.mediumSpecialtyProbability`

**Hard Mode:**
- Harder trivia questions prioritized
- Time multiplier: `gameConfig.hardTimeMultiplier`
- Token threshold multiplier: `gameConfig.hardThresholdMultiplier`
- Specialty question probability: `gameConfig.hardSpecialtyProbability`

### Dynamic Scaling Applications
- Trivia question difficulty selection
- Time limits for resource gathering and puzzle phases
- Token threshold calculations for all bonus effects
- Specialty question probability adjustments

## Technical Architecture

### Deployment
- **Containerization**: Backend and frontend ship as separate Docker images, orchestrated by `docker-compose`.
- **Single-Origin Topology**: Browser only ever talks to the frontend container; nginx reverse-proxies `/ws` and `/api` to the backend over an internal compose network. Eliminates CORS and means QR-code links and dev/prod URLs are identical.
- **Runtime Configuration**: `game-config.json` and `trivia/` are bind-mounted into the backend at runtime (not baked into the image), so tuning values and refreshing trivia content does not require a rebuild.

### Backend Infrastructure
- **Language**: Go (with a WebSocket library — Gorilla WebSocket is a reasonable default).
- **Concurrency**: Thread-safe operations with RWMutex.
- **Performance**: Connection pooling, efficient broadcasting.
- **Scalability**: Support for 4-64 players dynamically.

### Communication Protocol
- **WebSocket**: Full-duplex real-time communication for game events.
- **Authentication**: UUID-based session management with structured event format.
- **Validation**: Comprehensive input sanitization.
- **Error Handling**: Detailed error responses with context.

### Asset Delivery (Puzzle Images)
- **Source Images at Runtime**: Source images live under `assets/puzzle-sources/` and are bind-mounted into the backend container read-only at `/app/puzzle-sources/`. They are *not* baked into the image — adding a puzzle requires only a file drop and container restart.
- **Startup Validation**: On boot, the server verifies that `puzzle-sources/` is non-empty and that every file decodes as a usable image (square or square-croppable). The server refuses to start if validation fails, so missing or corrupt sources are caught at deploy time rather than mid-game.
- **Deferred Tile Generation**: Tiles are *not* generated at build time, on every connection, or on demand. Generation is triggered exactly once per game, at the resource-gathering → puzzle-assembly transition (see *Phase 1 → 2 Preparation* below), using the Go standard library's `image` package. Only the segments needed for the current player count and grid size are produced (e.g. 16 crops for a 16-player 4×4 game), not all six grid sizes.
- **In-Memory Tile Cache**: Generated segment images live in an in-memory `map[segmentId][]byte` held by the game manager. They are never written to disk and are released when the game resets.
- **Server-Gated Fetches**: The Go server is the sole holder of tile bytes. It exposes authenticated HTTP endpoints (under `/api/...`) that validate the requesting session token and check that the player is the assigned owner of the requested segment before streaming the bytes. Players never receive segment images they do not own.
- **Time-Windowed Full-Image Preview**: The clarity-token preview is enforced server-side. The full assembled image is only available through an authenticated endpoint during the calculated preview window (`gameConfig.clarityBasePreviewTime` + threshold bonuses); requests outside the window return 403 regardless of token validity.
- **Central Grid Fragments**: Once a player completes their individual puzzle, their fragment becomes visible to all participants. The server then permits all authenticated session tokens to fetch that specific fragment's image — visibility expansion is a server-controlled state transition, not a client decision.

### Phase 1 → 2 Preparation
When resource gathering ends, the server enters a brief "preparing puzzle" state during which it crops the source image into per-segment tiles. This pause maps onto the natural physical-world delay while players gather in the puzzle room. The host cannot start the puzzle phase until preparation completes; the host UI surfaces a "preparing puzzle…" indicator while it waits.

For the wire-level event sequence and rejection behavior, see *Resource Gathering → Puzzle Assembly* in `websocket-events.md`.

### State Management
- **Game State**: Atomic transitions between phases.
- **Player State**: Individual progress and analytics tracking.
- **Puzzle State**: Real-time grid synchronization with dual-system architecture.
- **Analytics**: Persistent tracking across reconnections.

### Security Features
- **Input Validation**: Size limits, format checking, UTF-8 validation.
- **Rate Limiting**: Fragment movement cooldown enforcement.
- **Privilege Checking**: Host vs player action authorization.
- **Asset Authorization**: All puzzle imagery served through authenticated, server-gated endpoints (see *Asset Delivery* above).

## Advanced Features

### Reconnection System

#### Authentication Requirements
**Both host and players reconnect using their original authentication tokens:**
- **Host**: Uses the same UUID provided at server startup
- **Players**: Use the UUID generated when they first connected

#### Player Reconnection
**Permitted Phases:**
- ✅ **Setup Phase**: Full reconnection with state restoration and role revalidation
- ✅ **Resource Gathering Phase**: Rejoin current round with preserved progress
- ❌ **Puzzle Assembly Phase**: Reconnection explicitly forbidden
- ✅ **Analytics Phase**: View personal performance report

**Phase-Specific Reconnection Behavior:**

**Setup Phase Reconnection:**
1. **Authentication**: Player reconnects with original UUID token
2. **Phase Detection**: Server identifies current phase as setup
3. **State Restoration**: Player removed from all counts during disconnection, now re-added
4. **Role Revalidation**: Check if previously selected role is still available
   - If role available: Restore all previous selections (role, specialty, name)
   - If role full: Force new role selection, preserve specialty and name
5. **Ready State**: Automatically marked ready if they were ready before disconnection
6. **Count Updates**: Player re-added to connected count, ready count, and role distribution

**Post-Setup Phase Reconnection:**
1. **Authentication**: Player reconnects with original UUID token
2. **Phase Detection**: Server identifies current game phase
3. **Configuration Recovery**: Restore previously selected role and specialty (no revalidation needed)
4. **Context Restoration**: Receive phase-appropriate game state and progress
5. **Maintained Presence**: Player was never removed from game counts during disconnection

#### Host Reconnection
**Permitted Phases:**
- ✅ **All Phases**: Host can reconnect during any phase

**State Restoration Process:**
1. **Authentication**: Host reconnects with original UUID
2. **Phase Context**: Receive current phase and appropriate monitoring interface
3. **Control Recovery**: Regain access to phase-specific host controls
4. **State Synchronization**: Get complete game state for monitoring dashboard
5. **Player Notification**: All players notified of host reconnection

#### Technical Implementation
- **Connection Events**: Both `SETUP_TO_HOST_CONNECTION_CONFIRMED` and `SETUP_TO_PLAYER_ROLES_AVAILABLE` include current phase and reconnection status
- **Token Validation**: Strict authentication prevents unauthorized reconnections
- **Phase Enforcement**: Puzzle phase reconnection block maintained for game integrity
- **State Synchronization**: Complete context restoration ensures seamless experience
- **Error Handling**: Clear feedback when reconnection is not permitted

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

### Server Configuration (`game-config.json`)
All values below come from `game-config.json`, mounted into the backend container at runtime so they can be tuned without a rebuild.

**Player Limits:**
- Minimum Players: `gameConfig.minPlayers` (excluding host)
- Maximum Players: `gameConfig.maxPlayers` (excluding host)

**Phase Timing:**
- Resource Gathering Rounds: `gameConfig.resourceGatheringRounds`
- Resource Gathering Round Duration: `gameConfig.resourceGatheringRoundDuration` seconds
- One trivia question per gathering round
- Puzzle Base Time: `gameConfig.puzzleBaseTime` seconds
- Post-Game Analytics: `gameConfig.postGameDuration` seconds

**Token Economics:**
- Base Tokens Per Answer: `gameConfig.baseTokensPerCorrectAnswer`
- Role Multiplier: `gameConfig.roleResourceMultiplier`
- Specialty Multiplier: `gameConfig.specialtyPointMultiplier`
- Anchor Token Threshold: `gameConfig.anchorTokenThreshold`
- Chronos Token Threshold: `gameConfig.chronosTokenThreshold`
- Guide Token Threshold: `gameConfig.guideTokenThreshold`
- Clarity Token Threshold: `gameConfig.clarityTokenThreshold`

**Game Balance:**
- Fragment Movement Cooldown: `gameConfig.fragmentMoveCooldown` ms
- Individual Puzzle Pieces: `gameConfig.individualPuzzlePieces` per player
- Answer Selection Time: `gameConfig.triviaAnswerTime`
- Grace Period Time: `gameConfig.triviaGraceTime`
- Max Specialties Per Player: `gameConfig.maxSpecialtiesPerPlayer` (set to 1)
- Grid Update Interval: `gameConfig.gridUpdateInterval` seconds (default 3s)

---

## WebSocket Event Reference

See `websocket-events.md` for the protocol specification — message formats,
event sequencing, HTTP asset endpoints, and authorization rules. This
document is the source of truth for game mechanics; `websocket-events.md`
is the source of truth for the wire format.
