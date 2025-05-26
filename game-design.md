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
- Bonus multiplier: 1.5x resource collection rate

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
  - Trivia questions every minute
  - Resource collection based on character roles

#### Trivia Mechanics
- Question Sources:
  - Large dataset of pre-loaded questions
  - Categorized by difficulty and topic
- Specialty Bonus:
  - Questions in player's specialty trigger:
    * Harder question difficulty
    * Double points if answered correctly
- Scoring:
  - Correct answer: Earn resource tokens
  - Incorrect answer: No resource tokens
  - Potential role-based multiplier

### Resource Tokens
1. **Anchor Tokens**
   - 5 thresholds
   - Each threshold adds 1 permanent piece to individual puzzle
   - 0 thresholds: All 16 pieces must be solved
   - 5 thresholds: 5 pieces permanently placed

2. **Chronos Tokens**
   - 5 thresholds
   - Each threshold adds 20 seconds to total puzzle-solving time
   - Time limit configurable in server settings

3. **Guide Tokens**
   - 5 thresholds
   - Provides narrowing location options for puzzle pieces
   - Options increase with more players

4. **Clarity Tokens**
   - 5 thresholds
   - Each threshold adds 1 second to initial image display
   - Default: No initial image display

### Phase 2: Puzzle Assembly
- **Location**: Large central room
- **Core Mechanics**:
  - Central screen displays puzzle assembly
  - Players solve individual 16-piece puzzle fragments
  - Collaborative piece placement

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
- Cooldown Period:
  - 1-second movement restriction after each fragment move
  - Prevents race conditions
  - Incoming move requests during cooldown ignored
- Movement Constraints:
  - Fragments can move freely within grid
  - 1-second cooldown applies to all fragments

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
- Individual player performance tracking
- Team performance metrics
- Resource collection statistics
- Trivia performance analysis

## Technical Infrastructure
- Frontend: React
- Backend: Go
- Communication: WebSocket
- Adaptive grid management
- Dynamic game scaling# Canvas Conundrum - Game Design Specification

## Game Concept
A collaborative puzzle-solving game where players recover a stolen masterpiece by gathering resources, answering trivia, and assembling a fragmented artwork.

## Game Phases

### Phase 1: Resource Gathering
- **Location**: Multiple rooms with QR code stations
- **Duration**: Configurable number of rounds
- **Core Mechanics**:
  - Players physically move between 4 QR code stations
  - Trivia questions every minute
  - Resource collection based on character roles

### Phase 2: Puzzle Assembly
- **Location**: Large central room (e.g., gym)
- **Core Mechanics**:
  - Central screen displays puzzle assembly
  - Players solve individual 16-piece puzzle fragments
  - Collaborative piece placement

## Character Selection

### Roles
1. Art Enthusiast
2. Detective
3. Tourist
4. Janitor

**Role Selection Rules**:
- Ensures even distribution of roles
- First players choose freely
- Subsequent players choose from remaining roles

### Trivia Specialties
- Players choose 1-2 trivia types
- Specialty questions are harder but offer bonus points

## Resource Tokens

### Anchor Tokens
- 5 thresholds
- Each threshold adds 1 permanent piece to individual puzzle
- At 0 thresholds: All 16 pieces must be solved
- At 5 thresholds: 5 pieces permanently placed

### Chronos Tokens
- 5 thresholds
- Each threshold adds 20 seconds to total puzzle-solving time
- Time limit configurable in server settings

### Guide Tokens
- 5 thresholds
- Provides narrowing location options for puzzle pieces
- Options increase with more players
- Helps reduce possible placement locations

### Clarity Tokens
- 5 thresholds
- Each threshold adds 1 second to initial image display
- Default: No initial image display

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
- Cooldown Period:
  - 1-second movement restriction after each fragment move
  - Prevents race conditions
  - Incoming move requests during cooldown ignored
- Movement Constraints:
  - Fragments can move freely within grid
  - 1-second cooldown applies to all fragments
    * Player-solved fragments
    * Pre-solved fragments
    * Disconnected players' fragments

### Grid Scaling Example
- Player Count | Grid Size | Total Fragments
- 1-9 players  | 3x3       | 9
- 10-16 players| 4x4       | 16
- 17-25 players| 5x5       | 25

### Disconnection Handling
- Disconnection defined as WebSocket communication failure
- Disconnected player's fragment:
  - Automatically solved
  - Randomly placed on grid
  - Becomes movable by remaining players
  - Subject to same movement cooldown rules

## Technical Design Principles
- Dynamic grid management
- Consistent fragment movement rules
- Robust handling of player disconnections
- Preventing potential race conditions

## Technology Stack
- Frontend: React
- Backend: Go
- Communication: WebSocket

## Difficulty Levels
- Easy
- Medium
- Hard
- Affects:
  - Trivia question complexity
  - Puzzle image complexity
  - Time limits
  - Token thresholds

## Winning Condition
- Complete the entire puzzle before time runs out

## Player Interaction during Puzzle Assembly
- In-person communication
- Digital piece recommendation system
  - Players can suggest piece swaps
  - Receiving player can accept or reject

## Configuration Considerations
- Configurable in server settings:
  - Number of rounds
  - Time per round
  - Trivia question pools
  - Token threshold times
  - Difficulty level settings
