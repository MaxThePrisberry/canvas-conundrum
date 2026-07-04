# Canvas Conundrum - Game Design Document

## Game Concept
A collaborative puzzle-solving game where players recover a stolen masterpiece by gathering resources, answering trivia, and assembling a fragmented artwork. The game uses a dedicated host system for reliable game management and real-time coordination.

The game runs through five phases: `setup` (Phase 0) → `resource_gathering` (Phase 1) → `puzzle_preparation` → `puzzle_assembly` (Phase 2) → `analytics` (Phase 3). Each has a section below, followed by the cross-cutting systems (difficulty, disconnections) and the technical architecture.

## Terminology
- **Piece**: One of the `gameConfig.individualPuzzlePieces` (default 16) sub-tiles a player manipulates inside their *own* individual puzzle (Phase 2A). Pieces are private and never appear on the central grid.
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
- Asset authorization: all puzzle imagery served through authenticated, server-gated HTTP endpoints (see *Asset Delivery (Puzzle Images)*)

## Phase 0: Setup and Connection

Players connect, select roles and specialties, host monitors readiness and starts the game when minimum players are ready.

### Role Selection (Players Only)
**4 Available Roles:**
1. **Art Enthusiast** → Clarity Token Bonus
2. **Detective** → Guide Token Bonus
3. **Tourist** → Chronos Token Bonus
4. **Janitor** → Anchor Token Bonus

**Role Mechanics:**
- Each role provides bonus collection for specific token type
- Bonus multiplier: `gameConfig.roleResourceMultiplier`
- Host does not select a role

**Slot Capacity:**
- Each role has capacity `max(1, ceil(connectedPlayers / 4))`; a role is selectable while its selected count is below capacity. This keeps roles evenly distributed, and because capacity is never below 1, a role only becomes unselectable after at least one player holds it — so small games still see every role's bonus token type collectible.
- Capacity is recomputed whenever a player connects or disconnects during setup; availability changes reach unready players via `SETUP_TO_PLAYER_ROLES_AVAILABLE`.
- If a disconnect shrinks capacity below a role's current count, existing selections are never revoked; the role is simply unselectable until capacity rises again.

**Race Resolution:**
- Two players may submit `SETUP_TO_SERVER_PLAYER_CONFIGURATION` for the last available slot of a role at almost the same time. The server processes configuration messages serially: the first to land claims the slot.
- Losers receive `SYSTEM_TO_CLIENT_ERROR` with code `ROLE_FULL`. Their previously selected specialty and player name are preserved server-side; they must reselect a role and resubmit. The next `SETUP_TO_PLAYER_ROLES_AVAILABLE` broadcast reflects the updated availability so loser clients can immediately offer remaining roles.

### Trivia Specialty Selection (Players Only)
**Available Categories:**
- General Knowledge, Geography, History, Music, Science, Video Games

**Specialty Mechanics:**
- Players select 1 category as their specialty
- Specialty questions are one difficulty level harder than the game's base difficulty (`gameConfig.difficultyMode`), capped at hard — in a hard game, specialty and regular questions are both drawn from the hard pool
- Same time limits as regular questions (no extension)
- Specialty bonus: `gameConfig.specialtyPointMultiplier`
- Frequency of specialty questions is difficulty-dependent — see *Difficulty Levels and Modifiers*
- Players are immediately marked as ready upon successful specialty selection

## Phase 1: Resource Gathering

**Duration**: Configurable rounds; each round lasts `gameConfig.triviaAnswerTime + gameConfig.triviaGraceTime` seconds (round duration is derived, not separately configurable)
- Number of rounds: `gameConfig.resourceGatheringRounds`
- Each gathering round = one trivia round = one trivia question sent to all players
- First part of round: Players select their answer from multiple choice options for `gameConfig.triviaAnswerTime` seconds
- Second part of round: Answers locked in, marked right/wrong, grace period of `gameConfig.triviaGraceTime` seconds for location changes and team discussion

**Location**: 4 QR code stations in physical spaces

**Participants**: Players only (host monitors)

### Core Mechanics
**Physical Movement:**
- Players physically move between 4 QR code stations
- Each station corresponds to different token type
- Location verification only required when changing stations
- Players start each game with no verified station (`unknown`); correct answers earn no tokens until the player's first successful QR scan
- QR codes' text value is the hash sent to the server for validation
- Station hashes stored as constants: `gameConfig.stationHashes.anchor`, `gameConfig.stationHashes.chronos`, `gameConfig.stationHashes.guide`, `gameConfig.stationHashes.clarity`

**Trivia System:**
- One question delivered per gathering round
- Distinct multiple-choice options; no fuzzy matching — clear right/wrong on the selected option, no partial credit
- Answers lock and are marked correct/incorrect after `gameConfig.triviaAnswerTime` seconds
- All questions have the same time limit regardless of specialty status

### Question Management
- 6 categories × 3 difficulties = 18 question pools (JSON files under `trivia/{category}/{difficulty}.json`)
- Automatic pool cycling when exhausted, with randomized order; history tracking prevents immediate repeats
- HTML entity decoding and text normalization

### Resource Token System
**Threshold formula (all four token types):**

```
thresholdsAchieved = min(gameConfig.maxThresholds,
                         floor(teamTokens / (tokenThreshold × thresholdMultiplier)))
```

where `tokenThreshold` is the per-type config value (`gameConfig.anchorTokenThreshold`, `gameConfig.chronosTokenThreshold`, `gameConfig.guideTokenThreshold`, `gameConfig.clarityTokenThreshold`) and `thresholdMultiplier` comes from the active difficulty mode (see *Difficulty Levels and Modifiers*). `gameConfig.maxThresholds` defaults to 6.

**Token Types & Effects:**

1. **Anchor Tokens** → Pre-solved Individual Puzzle Pieces
   - **Derived limits** (computed from `gameConfig.individualPuzzlePieces` and `gameConfig.maxThresholds`):
     - `maxPreSolvedPieces = floor(individualPuzzlePieces × 0.75)` — caps total pieces an anchor effect can pre-solve
     - `piecesPreSolvedPerThreshold = ceil(maxPreSolvedPieces / maxThresholds)` — pieces unlocked at each threshold (capped so the cumulative count never exceeds `maxPreSolvedPieces`)
     - At default 16 pieces: 12 max pre-solved, 2 per threshold, minimum 4 pieces always require manual solving
   - Pre-solved pieces are visually locked and unmovable
   - Only affects individual puzzle solving, NOT the central grid

2. **Chronos Tokens** → Extended Puzzle Time
   - Effect: `gameConfig.timeExtensionPerThreshold` seconds added per threshold to puzzle assembly time
   - Maximum total bonus: `gameConfig.maxThresholds × gameConfig.timeExtensionPerThreshold` seconds
   - Base time: `gameConfig.puzzleBaseTime` seconds
   - Team-wide benefit applied to entire puzzle phase

3. **Guide Tokens** → Fragment Placement Guidance on Central Grid
   - Effect: Highlights possible positions for the player's fragment on the central grid on the player's personal device
   - **Highlight count formula**: When threshold *N* of `gameConfig.maxThresholds` has been achieved:
     - If `N == 0`: no cells are highlighted (no guidance earned yet — showing every cell would be visual noise carrying no information).
     - If `N ≥ 1`: `max(1, ceil(gridSize² × (1 − N / maxThresholds)))` cells are highlighted.
     - At full thresholds the count converges to exactly one cell — the correct destination.
   - Individual hints visible only to each player for their own fragment
   - Only applies after individual puzzle completion

4. **Clarity Tokens** → Complete Image Preview
   - **Preview duration formula** (gated on threshold count, consistent with other token types):
     - If `N == 0`: no preview window — the full-image overlay is never shown and `/api/preview/full` returns `403 FORBIDDEN_PREVIEW_WINDOW_CLOSED` for the entire puzzle phase.
     - If `N ≥ 1`: preview window is `gameConfig.clarityBasePreviewTime + (N × gameConfig.previewTimePerThreshold)` seconds, starting from `PUZZLE_TO_CLIENT_PHASE_START`.
   - Maximum window: `gameConfig.clarityBasePreviewTime + (gameConfig.maxThresholds × gameConfig.previewTimePerThreshold)` seconds (when all thresholds earned)
   - Helps with spatial understanding and planning

### Token Scoring
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

## Puzzle Preparation Phase

When resource gathering ends, the game enters `puzzle_preparation` — a first-class phase with its own rules (player reconnection allowed; host-disconnect handling per *Disconnections and Reconnection* below). It covers both tile generation and the subsequent wait for the host to start the puzzle timer; `puzzle_assembly` begins only when the host sends `PUZZLE_TO_SERVER_PHASE_START`. The pause maps onto the natural physical-world delay while players gather in the puzzle room. The host cannot start the puzzle phase until tile generation completes; the host UI surfaces a "preparing puzzle…" indicator while it waits. Players still disconnected when the timer starts have their segments auto-solved into unassigned fragments at that moment.

For the wire-level event sequence and rejection behavior, see *Resource Gathering → Puzzle Assembly* in `websocket-events.md`.

## Phase 2: Puzzle Assembly

**Location**: Large central room (gymnasium recommended)

**Duration**: `gameConfig.puzzleBaseTime` seconds + chronos bonuses + difficulty modifiers

**Participants**: Players solve and collaborate (host monitors + shows big central grid for phase 2B)

Canvas Conundrum operates with two independent puzzle systems that remain separate until a specific transition moment.

### System 1: Individual Player Puzzles (Private & Invisible) - Phase 2A

**Isolation Rules:**
- Each player solves their assigned segment in a private workspace. Their progress is invisible to other players, the host, and any shared display.
- No fragment, position, or placeholder exists on the central grid for an in-progress segment. Fragments only appear there after their owner completes the individual puzzle.

**Individual Puzzle Mechanics:**
- **Assignment**: Each player receives exactly one unique segment ID (e.g., `segment_a5`, `segment_b2`)
- **Client Responsibilities**:
  - Load segment image using provided ID
  - Split segment into `gameConfig.individualPuzzlePieces` (default 16) jigsaw pieces
  - Shuffle pieces randomly for puzzle challenge
  - Choose which pieces to pre-solve based on anchor token count
  - Mark pre-solved pieces as locked and unmovable
  - Handle all piece movement and swapping logic
  - Validate when puzzle is correctly assembled
- **Solving Process**: Players arrange these pieces into the correct configuration privately
- **Pre-solving Effects**: Anchor tokens provide count of pieces to pre-solve (up to `maxPreSolvedPieces`), client chooses which pieces
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
- **Proportional Unassigned Fragment Reveal**: When more grid cells exist than players (i.e. `gridSize² > playerCount`), unassigned fragments appear gradually in lockstep with player completions. After *k* of *N* players have finished their individual puzzles, the grid shows a total of `ceil((k / N) × gridSize²)` fragments — *k* of which are the completed player-owned fragments, with the remainder filled in by randomly-selected unassigned fragments at random unoccupied positions. When `gridSize² == playerCount`, no unassigned fragments exist and the formula naturally yields zero. Auto-solved fragments from disconnected players count toward *k*; *N* is fixed when the puzzle phase starts and never shrinks.
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
   - Player arranges final pieces of their individual puzzle
   - Player sends `PUZZLE_TO_SERVER_SEGMENT_COMPLETED` message with `segmentId` and timestamp
   - Server accepts the completion claim — the client is authoritative for its own private puzzle — and assigns a grid position

3. **Instant Transformation**:
   - Individual puzzle instantly becomes one single fragment
   - Server places fragment at random unoccupied position on central shared grid
   - Fragment becomes visible to all players and host immediately
   - Fragment becomes movable according to ownership rules
   - Player transitions from Phase 2A to Phase 2B

4. **Post-Completion State**:
   - Individual puzzle workspace no longer exists for that player
   - Player now participates only in central grid collaboration
   - Fragment participates in shared puzzle assembly process

### Dynamic Grid System

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

### Fragment Ownership and Movement System

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
- **Position Validation**: All swaps validated against grid boundaries (0 to gridSize-1)
- **Collision Resolution**: Fragments swap positions or one fragment moves to open grid space
- **Permission Checking**: A direct move may only involve fragments the mover controls (their own fragment or unassigned ones). A swap that would displace another player's owned fragment is rejected with `not_owner` — propose it through the recommendation system instead.
- **Phase 2A Lockout**: Players still solving their individual puzzle cannot move fragments or use recommendations; they receive grid broadcasts read-only. Violations are rejected with `phase_invalid`.

### Fragment Visibility and State Management

**Visibility Rules:**
- **Invisible Until Completion**: Fragments only become visible on the central grid after their owner completes their individual puzzle. Anchor tokens pre-solve *pieces inside individual puzzles* — they do not place fragments on the central grid (see *Resource Token System*).
- **Persistent Once Visible**: Once a fragment appears on the grid it remains visible to all players and the host for the rest of the puzzle phase.
- **Personal View Consistency**: Each player sees identical central grid state.
- **Public vs Private**: Fragment positions are fully public — both the host's big-screen display and every player's device show the same grid. Guide-token highlights are strictly private; each player sees only the highlights for their own fragment, never another player's.

**State Broadcasting:**
- **Central Puzzle State**: Complete grid state sent to all players every `gameConfig.gridUpdateInterval` seconds (default 3s)
- **Host Updates**: Immediate updates on all fragment movements and state changes
- **Personal Puzzle State**: Individual view with guide highlighting (from guide tokens)
- **Phase Tracking**: Server implicitly tracks each player as Phase 2A (individual) or Phase 2B (collaborative)

### Strategic Collaboration System

**Piece Recommendation Protocol:**
- **Scope**: A recommendation proposes a swap between a fragment the sender controls (own or unassigned) and another player's owned fragment. The owner must explicitly accept before the swap executes. Swaps involving no other player's fragment never need a recommendation — they are executed directly as fragment moves.
- **Cooldown Gating**: Creating a recommendation and accepting one both require both named fragments to be off cooldown; otherwise the request is rejected with `COOLDOWN_ACTIVE`. A blocked acceptance leaves the recommendation pending — it can be retried once the cooldown passes.
- **Fragment-Based, Not Position-Based**: Pending recommendations are not invalidated when fragments move; an accepted swap exchanges the fragments' *current* positions. Both fragments' cooldowns restart when the swap executes.
- **Expiry**: A recommendation is cleared only by acceptance, rejection, timeout (`gameConfig.recommendationTimeout` seconds after creation), or either involved player disconnecting.
- **Analytics Tracking**: Recommendations sent/received/accepted are tracked for scoring.
- **Verbal Coordination**: Players encouraged to communicate during collaboration

### Token Effects in Puzzle Phase

The four token types and their puzzle-phase effects are fully defined under *Resource Token System* (Phase 1). One additional point not covered there: total puzzle-phase time is computed as `gameConfig.puzzleBaseTime + chronosBonus` and then scaled by the active difficulty's time multiplier (see *Difficulty Levels and Modifiers*).

### Puzzle Completion Logic

**Victory Conditions (Both Required):**
1. **All Fragments Present**: Every player's individual puzzle completed and converted to fragment
2. **Correct Positioning**: All fragments positioned at their designated correct grid coordinates

**Completion Validation:**
- **Authority Split**: Each client is authoritative for completing its own individual segment (see *The Critical Transformation Moment*); the server is authoritative for the central grid and the victory conditions.
- **Continuous Checking**: Server validates the victory conditions after every fragment movement.
- **Immediate Resolution on Success**: The game ends instantly when both conditions are satisfied; the server emits `PUZZLE_TO_CLIENT_COMPLETED_SUCCESS`.
- **Timer Expiry = Loss**: If the puzzle phase timer reaches zero before both victory conditions are met — for any reason — the team loses. This applies regardless of how many players are still in Phase 2A solving privately, how many fragments are unrevealed, or how many fragments are in incorrect positions. The server emits `PUZZLE_TO_CLIENT_COMPLETED_TIMEOUT` and transitions directly to the Analytics phase. There is no auto-solve fallback.

## Phase 3: Post-Game Analytics

**Duration**: Game result and analytics display until the host sends `ANALYTICS_TO_SERVER_RESET_GAME`; no auto-expiry

**Performance Tracking:**

### Individual Player Analytics
- **Token Collection**: Total tokens by type, role bonuses earned
- **Trivia Performance**: Accuracy by category, specialty accuracy and bonus tokens
- **Puzzle Solving Metrics**: Individual solve time, fragment moves and success rate, recommendations sent/received/accepted

### Team Performance Metrics
- **Overall Performance**: Puzzle outcome, completion time, team score
- **Resource Efficiency**: Token distribution, threshold achievements
- **Collaboration**: Move counts and success rate, recommendation acceptance rate

All reported metrics are direct counters or ratios of tracked events; the game does not compute synthetic scores (communication effectiveness, coordination ratings, etc.).

### Scoring Algorithm
```
Individual Score =
  (correctAnswers × gameConfig.pointsPerCorrectAnswer) +
  (specialtyCorrectAnswers × gameConfig.specialtyBonusPoints) +
  (gameConfig.completionBonus, if puzzle completed) +
  (successfulMoves × gameConfig.pointsPerSuccessfulMove) +
  (recommendationsSent × gameConfig.pointsPerRecommendationSent) +
  (recommendationsAccepted × gameConfig.pointsPerRecommendationAccepted)
```

## Difficulty Levels and Modifiers

The game difficulty is set by `gameConfig.difficultyMode` (`easy` | `medium` | `hard`), loaded at startup like every other config value. It has two kinds of effect:

**Trivia base difficulty** — regular questions are drawn from the pool matching the game difficulty. Specialty questions are one level harder, capped at hard (see *Trivia Specialty Selection*), so a hard game serves only hard questions.

**Gameplay modifiers** — each mode selects a multiplier set:

| Mode | Time multiplier | Threshold multiplier | Specialty probability |
|---|---|---|---|
| Easy | `gameConfig.easyTimeMultiplier` | `gameConfig.easyThresholdMultiplier` | `gameConfig.easySpecialtyProbability` |
| Medium | `gameConfig.mediumTimeMultiplier` | `gameConfig.mediumThresholdMultiplier` | `gameConfig.mediumSpecialtyProbability` |
| Hard | `gameConfig.hardTimeMultiplier` | `gameConfig.hardThresholdMultiplier` | `gameConfig.hardSpecialtyProbability` |

The time multiplier scales total puzzle-phase time (`(puzzleBaseTime + chronosBonus) × timeMultiplier`). The threshold multiplier scales the per-threshold token cost (see the threshold formula under *Resource Token System*). The specialty probability is the per-player, per-round chance that the round's question is drawn from that player's specialty category.

## Disconnections and Reconnection

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

### Post-Setup Phase Disconnections

**Player Disconnection:**
- **Maintained Presence**: Player remains in game counts and their contributions continue to matter
- **Resource Phase**: Collected tokens remain in team totals
- **Puzzle Phase**: If still solving (Phase 2A), the individual puzzle is immediately auto-solved into an unassigned fragment at a random unoccupied grid position; if already collaborating (Phase 2B), their existing fragment becomes unassigned. Either way, any remaining player can move it, and the disconnection is broadcast to all remaining players.
- **Analytics Phase**: Personal analytics preserved for viewing upon reconnection
- **Limited Reconnection**: Cannot reconnect during puzzle assembly phase; can reconnect during resource gathering, puzzle preparation, and analytics phases

### Host Disconnections (All Phases)

The general rule: setup and resource gathering proceed without a host — the host is only needed to *start* the game and to *start* the puzzle phase. Puzzle assembly, whose shared big-screen display is part of the gameplay artifact, pauses entirely.

| Phase | Effect of host disconnect |
|---|---|
| Setup | **Continues, but the game cannot start.** Players keep connecting, configuring, and readying up; `SETUP_TO_SERVER_START_GAME` requires the host. |
| Resource Gathering | **No effect.** Trivia rounds proceed on schedule; tokens accumulate. Host loses real-time monitoring until reconnect. |
| Puzzle Preparation | **Tile generation completes**, but the puzzle phase timer cannot start until the host reconnects and sends `PUZZLE_TO_SERVER_PHASE_START`. Game waits in a "ready, awaiting host" state. |
| Puzzle Assembly | **Full pause.** The phase timer stops and the server rejects every puzzle action — segment completions, fragment moves, recommendation creation and responses — with `FORBIDDEN_PHASE`/`phase_invalid` until the host reconnects. Pending recommendation timeout clocks pause as well. The puzzle deadline is extended by the disconnect duration. |
| Analytics | **No effect.** Reports remain available on each player's device; reset still requires the host but there is no time pressure. |

In all cases: complete game state is preserved for host reconnection, and players are notified of host disconnection and reconnection events. If the host never reconnects, the affected phase remains stalled — recovery requires a deployment-level restart.

### Reconnection

Both host and players reconnect with their original tokens (host: the server-startup UUID; players: the UUID issued at first connection). The host may reconnect during any phase. Players may reconnect during every phase except puzzle assembly; phase-specific effects are described above, and the wire-level handshake and state-restoration sequences are specified in `websocket-events.md` § *Reconnection Behavior*.

## Technical Architecture

### Deployment
- **Containerization**: Backend and frontend ship as separate Docker images, orchestrated by `docker-compose`.
- **Single-Origin Topology**: Browser only ever talks to the frontend container; nginx reverse-proxies `/ws` and `/api` to the backend over an internal compose network. Eliminates CORS and means QR-code links and dev/prod URLs are identical.
- **Runtime Configuration**: `game-config.json` and `trivia/` are bind-mounted into the backend at runtime (not baked into the image), so tuning values and refreshing trivia content does not require a rebuild.

### Backend Infrastructure
- **Language**: Go (with a WebSocket library — Gorilla WebSocket is a reasonable default).
- **Concurrency**: Thread-safe operations with RWMutex.

### Asset Delivery (Puzzle Images)
- **Source Images at Runtime**: Source images live under `assets/puzzle-sources/` and are bind-mounted into the backend container read-only at `/app/puzzle-sources/`. They are *not* baked into the image — adding a puzzle requires only a file drop and container restart.
- **Image Selection & Startup Validation**: `gameConfig.puzzleImage` names the source file (inside `puzzle-sources/`) used for every game until the config changes; the same filename is sent as `imageId` in the `PUZZLE_TO_*_PHASE_LOAD` events. On boot, the server verifies the configured file exists and decodes as a usable image (square or square-croppable), refusing to start otherwise — missing or corrupt sources are caught at deploy time rather than mid-game.
- **Deferred Tile Generation**: Tiles are *not* generated at build time, on every connection, or on demand. Generation is triggered exactly once per game, at the start of the puzzle preparation phase (see *Puzzle Preparation Phase*), using the Go standard library's `image` package. Only the segments needed for the current player count and grid size are produced (e.g. 16 crops for a 16-player 4×4 game), not all six grid sizes.
- **In-Memory Tile Cache**: Generated segment images live in an in-memory `map[segmentId][]byte` held by the game manager. They are never written to disk and are released when the game resets.
- **Server-Gated Fetches**: The Go server is the sole holder of tile bytes. It exposes authenticated HTTP endpoints (under `/api/...`) that validate the requesting session token and check that the player is the assigned owner of the requested segment before streaming the bytes. Players never receive segment images they do not own.
- **Time-Windowed Full-Image Preview**: The clarity-token preview is enforced server-side. The full assembled image is only available through an authenticated endpoint during the calculated preview window (`gameConfig.clarityBasePreviewTime` + threshold bonuses); requests outside the window return 403 regardless of token validity.
- **Central Grid Fragments**: Once a player completes their individual puzzle, their fragment becomes visible to all participants. The server then permits all authenticated session tokens to fetch that specific fragment's image — visibility expansion is a server-controlled state transition, not a client decision.

## Configuration Reference (`game-config.json`)

All values below come from `game-config.json`, mounted into the backend container at runtime so they can be tuned without a rebuild.

**Player Limits:**
- Minimum Players: `gameConfig.minPlayers` (excluding host)
- Maximum Players: `gameConfig.maxPlayers` (excluding host)

**Game Selection:**
- Difficulty Mode: `gameConfig.difficultyMode` (`easy` | `medium` | `hard`)
- Puzzle Image: `gameConfig.puzzleImage` (filename inside `assets/puzzle-sources/`)

**Phase Timing:**
- Resource Gathering Rounds: `gameConfig.resourceGatheringRounds`
- Round duration is derived: `gameConfig.triviaAnswerTime + gameConfig.triviaGraceTime` seconds
- One trivia question per gathering round
- Puzzle Base Time: `gameConfig.puzzleBaseTime` seconds

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
- Recommendation Timeout: `gameConfig.recommendationTimeout` seconds

---

## WebSocket Event Reference

See `websocket-events.md` for the protocol specification — message formats,
event sequencing, HTTP asset endpoints, and authorization rules. This
document is the source of truth for game mechanics; `websocket-events.md`
is the source of truth for the wire format.
