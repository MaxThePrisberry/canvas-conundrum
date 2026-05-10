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

**Distribution Goals:**
- Roles should be evenly distributed across players, scaling with player count.
- All four roles must remain selectable until at least one player has filled each, so that small-player-count games still see every role's bonus token type collected.

**Race Resolution:**
- Two players may submit `SETUP_TO_SERVER_PLAYER_CONFIGURATION` for the last available slot of a role at almost the same time. The server processes configuration messages serially: the first to land claims the slot.
- Losers receive `SYSTEM_TO_CLIENT_ERROR` with code `ROLE_FULL`. Their previously selected specialty and player name are preserved server-side; they must reselect a role and resubmit. The next `SETUP_TO_PLAYER_ROLES_AVAILABLE` broadcast reflects the updated availability so loser clients can immediately offer remaining roles.

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
   - Threshold count: `gameConfig.maxThresholds` (default 6)
   - Threshold spacing: `teamAnchorTokens / gameConfig.anchorTokenThreshold`
   - **Derived limits** (computed from `gameConfig.individualPuzzlePieces` and `gameConfig.maxThresholds`):
     - `maxPreSolvedPieces = floor(individualPuzzlePieces × 0.75)` — caps total pieces an anchor effect can pre-solve
     - `piecesPreSolvedPerThreshold = ceil(maxPreSolvedPieces / maxThresholds)` — pieces unlocked at each threshold (capped so the cumulative count never exceeds `maxPreSolvedPieces`)
     - At default 16 pieces: 12 max pre-solved, 2 per threshold, minimum 4 pieces always require manual solving
   - Pre-solved pieces are visually locked and unmovable
   - Only affects individual puzzle solving, NOT the central grid

2. **Chronos Tokens** → Extended Puzzle Time
   - Threshold count: `gameConfig.maxThresholds`
   - Threshold spacing: `teamChronosTokens / gameConfig.chronosTokenThreshold`
   - Effect: `gameConfig.timeExtensionPerThreshold` seconds added per threshold to puzzle assembly time
   - Maximum total bonus: `gameConfig.maxThresholds × gameConfig.timeExtensionPerThreshold` seconds
   - Base time: `gameConfig.puzzleBaseTime` seconds
   - Team-wide benefit applied to entire puzzle phase

3. **Guide Tokens** → Fragment Placement Guidance on Central Grid
   - Threshold count: `gameConfig.maxThresholds`
   - Threshold spacing: `teamGuideTokens / gameConfig.guideTokenThreshold`
   - Effect: Highlights possible positions for the player's fragment on the central grid on the player's personal device
   - **Highlight count formula**: When threshold *N* of `gameConfig.maxThresholds` has been achieved:
     - If `N == 0`: no cells are highlighted (no guidance earned yet — showing every cell would be visual noise carrying no information).
     - If `N ≥ 1`: `max(1, ceil(gridSize² × (1 − N / maxThresholds)))` cells are highlighted.
     - At full thresholds the count converges to exactly one cell — the correct destination.
   - Individual hints visible only to each player for their own fragment
   - Only applies after individual puzzle completion

4. **Clarity Tokens** → Complete Image Preview
   - Threshold count: `gameConfig.maxThresholds`
   - Threshold spacing: `teamClarityTokens / gameConfig.clarityTokenThreshold`
   - **Preview duration formula** (gated on threshold count, consistent with other token types):
     - If `N == 0`: no preview window — the full-image overlay is never shown and `/api/preview/full` returns `403 FORBIDDEN_PREVIEW_WINDOW_CLOSED` for the entire puzzle phase.
     - If `N ≥ 1`: preview window is `gameConfig.clarityBasePreviewTime + (N × gameConfig.previewTimePerThreshold)` seconds, starting from `PUZZLE_TO_CLIENT_PHASE_START`.
   - Maximum window: `gameConfig.clarityBasePreviewTime + (gameConfig.maxThresholds × gameConfig.previewTimePerThreshold)` seconds (when all thresholds earned)
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

Canvas Conundrum operates with two independent puzzle systems that remain separate until a specific transition moment.

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
- **Dynamic Scaling**: Grid size automatically scales with player count.
- **Fragment Creation**: Each completed individual puzzle becomes one movable fragment.
- **Proportional Unassigned Fragment Reveal**: When more grid cells exist than players (i.e. `gridSize² > playerCount`), unassigned fragments appear gradually in lockstep with player completions. After *k* of *N* players have finished their individual puzzles, the grid shows a total of `ceil((k / N) × gridSize²)` fragments — *k* of which are the completed player-owned fragments, with the remainder filled in by randomly-selected unassigned fragments at random unoccupied positions. When `gridSize² == playerCount`, no unassigned fragments exist and the formula naturally yields zero.
- **Movement Rules**: Players can move their own fragments and any unassigned fragments.
- **Collaboration Features**: Recommendation system for strategic coordination.

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
- **Movement Cooldown**: `gameConfig.fragmentMoveCooldown` ms is enforced **per fragment**, not per player. Each fragment has its own cooldown clock that starts when the fragment last moved (whether moved by its owner, an unassigned mover, or as the swap target of another move). A single player can chain rapid moves across different fragments. Two players who both want to move the same unassigned fragment serialize on its cooldown.
- **Terminology**: Also called fragment move requests, fragment recommendations, or switch requests
- **Position Validation**: All swaps validated against grid boundaries (0 to gridSize-1)
- **Collision Resolution**: Fragments swap positions or one fragment moves to open grid space
- **Permission Checking**: Server validates ownership before allowing movement
- **State Synchronization**:
  - Host: Immediate updates on all movements
  - Players: Updates every `gameConfig.gridUpdateInterval` seconds

#### Fragment Visibility and State Management

**Visibility Rules:**
- **Invisible Until Completion**: Fragments only become visible on the central grid after their owner completes their individual puzzle. Anchor tokens pre-solve *pieces inside individual puzzles* — they do not place fragments on the central grid (see *Anchor Token Pre-Solving*).
- **Persistent Once Visible**: Once a fragment appears on the grid it remains visible to all players and the host for the rest of the puzzle phase.
- **Personal View Consistency**: Each player sees identical central grid state.
- **Public vs Private**: Fragment positions are fully public — both the host's big-screen display and every player's device show the same grid. Guide-token highlights are strictly private; each player sees only the highlights for their own fragment, never another player's.

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

The four token types and their puzzle-phase effects are fully defined under *Resource Token System* (Phase 1). One additional point not covered there: total puzzle-phase time is computed as `gameConfig.puzzleBaseTime + chronosBonus` and then scaled by the active difficulty's time multiplier (see *Difficulty Levels and Modifiers*).

#### Puzzle Completion Logic

**Victory Conditions (Both Required):**
1. **All Fragments Present**: Every player's individual puzzle completed and converted to fragment
2. **Correct Positioning**: All fragments positioned at their designated correct grid coordinates

**Completion Validation:**
- **Continuous Checking**: Server validates the victory conditions after every fragment movement.
- **Immediate Resolution on Success**: The game ends instantly when both conditions are satisfied; the server emits `PUZZLE_TO_CLIENT_COMPLETED_SUCCESS`.
- **Timer Expiry = Loss**: If the puzzle phase timer reaches zero before both victory conditions are met — for any reason — the team loses. This applies regardless of how many players are still in Phase 2A solving privately, how many fragments are unrevealed, or how many fragments are in incorrect positions. The server emits `PUZZLE_TO_CLIENT_COMPLETED_TIMEOUT` and transitions directly to the Analytics phase. There is no auto-solve fallback.
- **Success Analytics**: Comprehensive performance tracking for the successful-completion case.

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

The general rule: a host disconnect should never block phases that don't require host input or a shared host-driven display. Where the host's role is purely monitoring, the game proceeds; where the host's screen is part of the gameplay artifact (the shared central grid display) or where host input is required to advance state, the game pauses.

| Phase | Effect of host disconnect |
|---|---|
| Setup | **Pause.** Game cannot start without `SETUP_TO_SERVER_START_GAME` from the host. |
| Resource Gathering | **No effect.** Trivia rounds proceed on schedule; tokens accumulate. Host loses real-time monitoring until reconnect. |
| Puzzle Preparation (Phase 1 → 2 transition) | **Tile generation completes**, but the puzzle phase timer cannot start until the host reconnects and sends `PUZZLE_TO_SERVER_PHASE_START`. Game waits in a "ready, awaiting host" state. |
| Puzzle Assembly | **Timer pauses, resumes on reconnect.** Players still solving their individual puzzles privately (Phase 2A) continue normally — they don't depend on the host's display. Phase 2B players retain access to the central grid on their own devices and may continue moving fragments, but the shared big-screen experience is unavailable until host reconnects. The puzzle phase deadline is extended by the disconnect duration. |
| Analytics | **No effect.** Reports remain available on each player's device; reset still requires the host but there is no time pressure. |

In all cases: complete game state is preserved for host reconnection, and players are notified of host disconnection and reconnection events. If the host never reconnects, the affected phase remains stalled — recovery requires a deployment-level restart.

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
  (Correct Answers × gameConfig.pointsPerCorrectAnswer) +
  (Specialty Bonus × gameConfig.specialtyBonusPoints) +
  (gameConfig.completionBonus, if puzzle completed) +
  (Speed Bonus: scaled 0..gameConfig.maxSpeedBonus by completion time) +
  (Successful Moves × gameConfig.pointsPerSuccessfulMove) +
  (Recommendations Sent × gameConfig.pointsPerRecommendationSent) +
  (Recommendations Accepted × gameConfig.pointsPerRecommendationAccepted)
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
- Post-Game Analytics: displays until the host sends `ANALYTICS_TO_SERVER_RESET_GAME`; no auto-expiry

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
- Individual Puzzle Pieces: `gameConfig.individualPuzzlePieces` per player — must be a perfect square (4, 9, 16, 25, 36, 49, 64) since it's manipulated as an N×N sub-grid; default 16. The server rejects startup if this value is not a perfect square.
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
