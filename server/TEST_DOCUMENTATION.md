# Canvas Conundrum Test Suite Documentation

## Overview
A comprehensive test suite has been created for the Canvas Conundrum Go backend server. The test suite ensures full compliance with the specifications in `websocket-events.md` and `game-design.md`, covering all 59 WebSocket events and game mechanics.

## Test Structure

### 1. Test Helpers (`test_helpers/`)
- **test_utils.go**: Common test utilities and assertions
  - Mock player/host creation
  - Test message creation
  - Fluent assertion helpers for game state and tokens
  
- **websocket_client.go**: WebSocket test client implementation
  - `TestWebSocketClient`: Generic WebSocket client for testing
  - `TestHostClient`: Specialized host client with game control methods
  - `TestPlayerClient`: Specialized player client with game action methods
  - Connection management and message tracking
  
- **fixtures.go**: Test data factories and scenarios
  - Pre-configured test players, hosts, and games
  - Mock trivia question generation
  - Game scenario definitions (small/medium/large games)
  - Temporary file creation for trivia JSON testing

### 2. Unit Tests

#### Models Package (`models/*_test.go`)
- **game_test.go**: Tests for game state management
  - Phase transitions
  - Difficulty settings and grid size calculation
  - Token threshold calculations
  - Pre-solved pieces based on tokens
  - Puzzle time calculations with chronos bonuses
  
- **player_test.go**: Tests for player and host models
  - Player initialization and state management
  - Role and specialty assignment
  - Token and trivia statistics tracking
  - Puzzle segment assignment
  - Connection status management
  
- **trivia_test.go**: Tests for trivia question handling
  - Question parsing and HTML decoding
  - Category and difficulty mapping
  - Question pool management
  - Options shuffling
  - JSON loading from API response format

#### Services Package (`services/*_test.go`)
- **game_manager_test.go**: Tests for singleton game manager
  - Player/host management
  - Game flow control
  - Token management
  - Role distribution
  - Recommendation system
  - Concurrent operations safety
  
- **trivia_service_test.go**: Tests for trivia service
  - Question loading and storage
  - Specialty question selection
  - Answer validation
  - Current question tracking

#### Utils Package (`utils/*_test.go`)
- **json_test.go**: Tests for WebSocket message handling
  - Message marshaling/unmarshaling
  - Error payload creation
  - Complex nested payload handling
  - Timestamp management

### 3. Integration Tests (`integration_tests/`)
- **websocket_integration_test.go**: WebSocket handler integration tests
  - Player and host connection flows
  - Authentication validation
  - Multiple player joining
  - Game start sequence
  - Ping/pong heartbeat
  - Concurrent connections (20+ players)
  - Disconnection handling

### 4. End-to-End Tests (`e2e_tests/`)
- **complete_game_flow_test.go**: Full game flow testing
  - Complete 4-phase game progression
  - Token collection and threshold effects
  - Puzzle assembly simulation
  - Analytics generation
  - Game reset functionality
  
- **websocket_events_coverage_test.go**: Event coverage verification
  - Validates all 59 WebSocket events are defined
  - Checks event naming conventions
  - Verifies event directionality (to/from server/client/player/host)
  - Groups events by phase for comprehensive coverage

## Test Execution

### Using Make Commands
```bash
# Run all tests
make test

# Run unit tests only
make unit

# Run integration tests
make integration

# Run end-to-end tests
make e2e

# Run with coverage report
make coverage

# Generate HTML coverage report
make coverage-html

# Run with race detector
make race

# Quick unit tests (no coverage)
make quick

# Run specific test by name
make test-name NAME=TestCompleteGameFlow

# Run tests for specific package
make test-pkg PKG=models

# Continuous integration suite
make ci
```

### Test Coverage Goals
- **Target**: 80% overall code coverage
- **Critical Paths**: 100% coverage for:
  - Game state management
  - Score calculation
  - WebSocket event handling
  - Token threshold mechanics

## Key Test Scenarios

### 1. Setup Phase
- Host connection with valid/invalid UUID
- Player connections and role selection
- Specialty selection (max 1 per player)
- Lobby status updates
- Game start validation (4-64 players)

### 2. Resource Gathering Phase
- QR code verification at stations
- Trivia question delivery
- Answer validation and token rewards
- Round transitions
- Team progress broadcasting

### 3. Puzzle Assembly Phase
- Individual puzzle solving
- Fragment movement validation
- Recommendation system
- Grid state synchronization
- Completion detection

### 4. Analytics Phase
- Score calculation
- Achievement determination
- Team metrics aggregation
- Report generation for players and host

### 5. System Events
- Error handling
- Disconnection/reconnection
- Phase transitions
- Real-time broadcasting

## WebSocket Events Coverage

All 59 events from the specification are tested:
- **Setup Phase**: 8 events
- **Resource Gathering**: 10 events  
- **Puzzle Assembly**: 20 events
- **Analytics**: 5 events
- **System Events**: 10 events
- **Special Events**: 6 events (PING/PONG, phase transitions)

## Performance Testing

The test suite includes:
- Concurrent connection tests (20+ simultaneous players)
- Large-scale game simulation (32-64 players)
- Race condition detection
- Memory leak prevention

## Best Practices Implemented

1. **Test Isolation**: Each test is independent and can run in any order
2. **Mock Dependencies**: WebSocket connections are mocked for unit tests
3. **Fixtures and Factories**: Reusable test data creation
4. **Fluent Assertions**: Readable test assertions
5. **Comprehensive Coverage**: All game specifications validated
6. **CI/CD Ready**: Makefile targets for automated testing

## Running Tests in CI/CD

```yaml
# Example GitHub Actions workflow
- name: Run Tests
  run: |
    make deps
    make ci
    make coverage
```

## Notes

- Tests do not modify server code to make testing easier
- All tests validate against specifications in websocket-events.md and game-design.md
- Import cycles have been avoided by careful package structuring
- Tests use standard Go testing package with testify for assertions