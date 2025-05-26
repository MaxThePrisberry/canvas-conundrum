# Canvas Conundrum - Comprehensive Game Design

## Game Concept
A collaborative puzzle-solving game where players recover a stolen masterpiece by gathering resources, answering trivia, and assembling a fragmented artwork.

## Player Setup and Character Selection

### Role Selection
- 4 Available Roles:
  1. Art Enthusiast
  2. Detective
  3. Tourist
  4. Janitor

#### Role Mechanics
- Each role provides a resource collection bonus
- Bonus applies to specific resource type
- Bonus multiplier: `constants.RoleResourceMultiplier` (see constants/game_balance.go)

### Character Distribution
- Ensures even distribution of roles
- Selection process:
  1. First players choose freely from all roles
  2. Subsequent players choose from remaining roles
  3. Guarantees representation of all roles

### Trivia Specialty Selection
- Players choose 1-2 trivia categories
- Specialties determine bonus points and question difficulty

## Game Phases

### Phase 1: Resource Gathering
- **Location**: Multiple rooms with QR code stations
- **Duration**: Configurable number of rounds
- **Core Mechanics**:
  - Players physically move between 4 QR code stations
  - Trivia questions every `constants.TriviaQuestionInterval` seconds (see constants/game_balance.go)
  - Resource collection based on character roles

#### Trivia Mechanics
- Question Sources:
  - Large dataset of pre-loaded questions
  - Categorized by difficulty and topic
- Specialty Bonus:
  - Questions in player's specialty trigger:
    * Harder question difficulty
    * `constants.SpecialtyPointMultiplier` points if answered correctly (see constants/game_balance.go)
- Scoring:
  - Correct answer: Earn resource tokens
  - Incorrect answer: No resource tokens
  - Potential role-based multiplier

### Resource Tokens
1. **Anchor Tokens**
   - `constants.AnchorTokenThresholds` thresholds (see constants/game_balance.go)
   - Each threshold adds 1 permanent piece to individual puzzle
   - 0 thresholds: All 16 pieces must be solved
   - Max thresholds: `constants.AnchorTokenThresholds` pieces permanently placed

2. **Chronos Tokens**
   - `constants.ChronosTokenThresholds` thresholds (see constants/game_balance.go)
   - Each threshold adds `constants.ChronosTimeBonus` seconds to total puzzle-solving time (see constants/game_balance.go)
   - Time limit configurable in server settings

3. **Guide Tokens**
   - `constants.GuideTokenThresholds` thresholds (see constants/game_balance.go)
   - Provides narrowing location options for puzzle pieces
   - Options increase with more players

4. **Clarity Tokens**
   - `constants.ClarityTokenThresholds` thresholds (see constants/game_balance.go)
   - Each threshold adds `constants.ClarityTimeBonus` second to initial image display (see constants/game_balance.go)
   - Default: No initial image display

### Phase 2: Puzzle Assembly
- **Location**: Large central room (e.g., gym)
- **Core Mechanics**:
  - Central screen displays puzzle assembly
  - Players solve individual 16-piece puzzle fragments
  - Collaborative piece placement
  - In-person communication
  - Digital piece recommendation system

## Player Interaction during Puzzle Assembly
- **In-person communication**: Players can communicate verbally and physically coordinate
- **Digital piece recommendation system**:
  - Players can suggest piece swaps to other players
  - Receiving player can accept or reject suggestions
  - Facilitates strategic coordination

## Puzzle Mechanics: Grid Scaling

### Grid Size Determination
- Grid always maintains a perfect square shape
- Grid size scales based on total number of players
- Grid size progression:
  - 1-9 players: 3x3 grid (9 fragments)
  - 10-16 players: 4x4 grid (16 fragments)
  - 17-25 players: 5x5 grid (25 fragments)
  - Continues with increasing square grid sizes

### Fragment Movement Rules
- **Cooldown Period**:
  - `constants.FragmentMovementCooldown` millisecond movement restriction after each fragment move (see constants/game_balance.go)
  - Prevents race conditions
  - Incoming move requests during cooldown ignored
- **Movement Constraints**:
  - Fragments can move freely within grid
  - Cooldown applies to all fragments:
    * Player-solved fragments
    * Pre-solved fragments
    * Disconnected players' fragments

### Player Disconnection Handling
- Disconnection defined as WebSocket communication failure
- Disconnected player's fragment:
  - Automatically solved
  - Randomly placed on grid
  - Becomes movable by remaining players
  - Subject to same movement cooldown rules

## Winning Condition
- Complete the entire puzzle before time runs out

## Difficulty Levels
- Easy
- Medium
- Hard
- Affects:
  - Trivia question complexity
  - Puzzle image complexity
  - Time limits
  - Token thresholds

## Post-Game Analytics
- **Individual player performance tracking**:
  - Trivia accuracy rates by category
  - Resource collection efficiency
  - Puzzle-solving contribution metrics
- **Team performance metrics**:
  - Overall completion time
  - Collaboration effectiveness scores
  - Resource allocation efficiency
- **Resource collection statistics**:
  - Token distribution analysis
  - Role-based performance comparison
- **Trivia performance analysis**:
  - Question difficulty vs success rates
  - Specialty bonus utilization
  - Knowledge gap identification

## Technical Infrastructure
- **Frontend**: React
- **Backend**: Go
- **Communication**: WebSocket
- **Features**:
  - Adaptive grid management
  - Dynamic game scaling
  - Robust disconnection handling
  - Real-time collaborative puzzle interface

## Configuration Considerations
Configurable in server settings:
- Number of rounds in Phase 1
- Time per round
- Trivia question pools and categories
- Token threshold requirements
- Difficulty level settings
- Base time limits for puzzle assembly
- Grid scaling parameters

## Technical Design Principles
- Dynamic grid management based on player count
- Consistent fragment movement rules across all scenarios
- Robust handling of player disconnections
- Prevention of race conditions in collaborative interactions
- Scalable architecture supporting variable player counts
