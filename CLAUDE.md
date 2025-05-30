# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Canvas Conundrum is a collaborative multiplayer puzzle game with educational trivia elements. Players answer trivia questions to earn resources, then work together to assemble puzzle pieces on a shared canvas.

## Development Commands

```bash
# Install dependencies
go mod download

# Run the server
go run .

# Run with custom configuration
go run . -env=development -port=8080

# Build server binary
go build -o canvas-conundrum-server

# Setup pre-commit hooks
pre-commit install

# Format code (via pre-commit)
pre-commit run go-fmt --all-files
```

## Architecture Overview

### Server Structure (Go WebSocket Backend)

The server uses an event-driven architecture with three core managers:

1. **GameManager** (`game_manager.go`): Controls game flow through phases (Setup → Resource → Puzzle → PostGame). Manages trivia rounds, token distribution, and puzzle assembly logic.

2. **PlayerManager** (`player_manager.go`): Handles player connections, roles, specialties, and distinguishes between hosts (non-playing monitors) and regular players.

3. **TriviaManager** (`trivia_manager.go`): Loads questions from JSON files, validates answers with fuzzy matching, and manages question cycling when pools are exhausted.

### Key Patterns

- **WebSocket Communication**: All real-time updates flow through WebSocket connections with type-based message routing
- **Dual Endpoint System**: Hosts connect via `/ws/host/{uuid}`, players via `/ws`
- **Broadcast Channel**: Centralized message distribution system for game state updates
- **Mutex Protection**: Thread-safe state management across concurrent connections
- **Validation Layer**: All inputs validated before processing (`validation.go`)

### Game Phases

1. **Setup Phase**: Host configuration, player joining
2. **Resource Phase**: 5 or more rounds of synchronized trivia depending on constants/difficulty (60s each)
3. **Puzzle Phase**: Collaborative puzzle assembly with token usage
4. **PostGame Phase**: Analytics display and game cleanup

### Token System

Four token types with specific effects:
- Freeze: Lock fragments
- Rotate: Turn fragments
- Swap: Exchange positions
- Hint: Reveal connections

## Testing

Currently no tests exist. When implementing tests:
```bash
go test ./...                    # Run all tests
go test -cover ./...            # With coverage
go test -v -run TestSpecific    # Run specific test
```

## Important Notes

- The client directory exists but is not yet implemented
- Trivia questions are stored in JSON files under `server/trivia/`
- Game supports 4-64 players with automatic grid scaling
- All game constants are centralized in `constants/game_balance.go`
- WebSocket messages follow strict type definitions in `types.go`
