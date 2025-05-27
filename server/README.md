# Canvas Conundrum Server

A Go-based WebSocket server for the Canvas Conundrum collaborative puzzle game. This server manages real-time multiplayer gameplay including trivia rounds, resource collection, and collaborative puzzle solving.

## Prerequisites

- Go 1.21 or higher
- Git

## Quick Start

1. **Clone and setup**:
```bash
git clone <repository-url>
cd canvas-conundrum/server
go mod download
```

2. **Create trivia directory structure**:
```bash
mkdir -p trivia/{general,geography,history,music,science,video_games}
```

3. **Add trivia questions**: Place JSON files with trivia questions in each category directory. See [Trivia Setup](#trivia-setup) for format details.

4. **Start the server**:
```bash
go run .
```

5. **Note the endpoints**:
```
🎮 HOST ENDPOINT: /ws/host/a1b2c3d4-e5f6-7890-abcd-ef1234567890
👥 PLAYER ENDPOINT: /ws
🔗 Host URL: ws://localhost:8080/ws/host/a1b2c3d4-e5f6-7890-abcd-ef1234567890
🔗 Player URL: ws://localhost:8080/ws
```

## Host vs Player System

Canvas Conundrum uses a dedicated host system for reliable game management:

### Host Connection
- **Endpoint**: `/ws/host/{unique-uuid}` (generated fresh each server start)
- **Role**: Game moderator and controller
- **Capabilities**: Start games, monitor progress, control game flow
- **Limitations**: Cannot participate in trivia or puzzle solving
- **Reconnection**: Can reconnect using same endpoint + player ID

### Player Connection
- **Endpoint**: `/ws`
- **Role**: Game participants
- **Capabilities**: Answer trivia, collect tokens, solve puzzles, select roles
- **Requirements**: Must have host present to start games
- **Reconnection**: Can reconnect using player ID

## Configuration

### Command Line Options
```bash
go run . [options]

Options:
  -env string
        Environment: development, staging, production (default "development")
  -port string
        Server port (default "8080")
  -host string
        Server host (default "0.0.0.0")
  -origins string
        Comma-separated allowed CORS origins (auto-configured for development)
  -cert string
        TLS certificate file for HTTPS
  -key string
        TLS private key file for HTTPS
```

### Environment Variables
```bash
# CORS configuration (required for production)
ALLOWED_ORIGINS="https://yourdomain.com,https://www.yourdomain.com"

# Admin authentication (optional - enables admin endpoints)
ADMIN_TOKEN="your-secure-random-token"
```

### Game Balance Configuration

Edit `constants/game_balance.go` to adjust:

- **Player Limits**: Min/max players (default: 4-64)
- **Time Settings**: Round durations, puzzle time limits
- **Token Economics**: Resource collection rates, threshold requirements
- **Trivia Settings**: Question intervals, specialty bonuses
- **Puzzle Mechanics**: Grid scaling, fragment movement cooldowns

## Trivia Setup

### Directory Structure
```
trivia/
├── general/
│   ├── easy.json
│   ├── medium.json
│   └── hard.json
├── geography/
│   ├── easy.json
│   ├── medium.json
│   └── hard.json
└── ... (other categories)
```

### Question Format
```json
{
  "response_code": 0,
  "results": [
    {
      "type": "multiple",
      "difficulty": "medium",
      "category": "General Knowledge",
      "question": "What is the capital of France?",
      "correct_answer": "Paris",
      "incorrect_answers": ["London", "Berlin", "Madrid"]
    }
  ]
}
```

### Supported Categories
- `general` - General knowledge questions
- `geography` - Geography and places
- `history` - Historical events and figures
- `music` - Music and entertainment
- `science` - Science and nature
- `video_games` - Gaming and technology

## API Endpoints

### WebSocket Endpoints
- `GET /ws` - Player connections
- `GET /ws/host/{uuid}` - Host connection (UUID shown in server logs)

### HTTP Endpoints
- `GET /health` - Server health check and status
- `GET /stats` - Current game statistics
- `POST /admin/reload-trivia` - Reload trivia questions (requires admin token)
- `GET /admin/host-endpoint` - Get current host endpoint (requires admin token)

### Health Check Response
```json
{
  "status": "healthy",
  "timestamp": 1640995200,
  "environment": "development",
  "endpoints": {
    "players": "/ws",
    "host": "/ws/host/a1b2c3d4-e5f6-7890-abcd-ef1234567890"
  },
  "game": {
    "phase": "setup",
    "hasHost": true,
    "players": {
      "total": 5,
      "connected": 4,
      "ready": 3
    }
  },
  "trivia": {
    "totalQuestions": 1500,
    "categories": 6
  }
}
```

## Game Flow

### 1. Setup Phase
- Host connects to host endpoint
- Players connect to player endpoint
- Players select roles and trivia specialties
- Host starts game when all players ready

### 2. Resource Gathering Phase
- Players move between QR code stations physically
- Trivia questions asked at timed intervals
- Correct answers earn tokens based on location and player role
- Tokens unlock bonuses for puzzle phase

### 3. Puzzle Assembly Phase
- Players receive individual puzzle segments to solve
- Collaborative grid assembly of completed segments
- Real-time piece movement and position suggestions
- Host can monitor progress and provide guidance

### 4. Analytics Phase
- Individual and team performance metrics
- Detailed breakdown of contributions and collaboration
- Leaderboard and achievement summaries

## Deployment

### Development
```bash
# Auto-configures CORS for localhost
go run . -env=development
```

### Production
```bash
# Build binary
go build -o canvas-conundrum-server

# Set environment
export ALLOWED_ORIGINS="https://yourdomain.com"
export ADMIN_TOKEN="$(openssl rand -base64 32)"

# Run with HTTPS
./canvas-conundrum-server \
  -env=production \
  -cert=cert.pem \
  -key=key.pem \
  -port=8080 \
  -origins="https://yourdomain.com"
```

### Docker Deployment
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o canvas-conundrum-server

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/canvas-conundrum-server .
COPY --from=builder /app/trivia ./trivia
EXPOSE 8080
CMD ["./canvas-conundrum-server", "-env=production"]
```

## Monitoring and Management

### Server Logs
```bash
# Startup information
INFO: Canvas Conundrum Server starting on 0.0.0.0:8080 (env: development)
🎮 HOST ENDPOINT: /ws/host/a1b2c3d4-e5f6-7890-abcd-ef1234567890
INFO: Loaded 1500 trivia questions across 6 categories

# Connection events
INFO: New host connected: abc123
INFO: New player connected: def456
INFO: Player def456 disconnected

# Game events
INFO: Game started with 8 players
INFO: Puzzle phase started - 4x4 grid
INFO: Game completed successfully in 1200 seconds
```

### Performance Metrics
- WebSocket connection count and stability
- Message throughput and latency
- Game completion rates and duration
- Trivia question distribution and cycling
- Memory usage and garbage collection

## Troubleshooting

### Common Issues

**"No trivia questions loaded"**
- Ensure JSON files exist in all `trivia/{category}/` directories
- Verify JSON format matches the expected structure
- Check file permissions and server file access

**"WebSocket connection failed"**
- Verify CORS origins are configured correctly
- Check firewall settings and port accessibility
- Ensure WebSocket upgrade headers are present

**"A host is already connected"**
- Only one host can connect per game instance
- Previous host may need to disconnect first
- Check for stale connections in server logs

**"Cannot join game at this time"**
- Game may have progressed beyond setup phase
- Player limit (64) may have been reached
- Host may not be connected

### Development Tips

- Use `-env=development` for automatic localhost CORS handling
- Monitor `/health` endpoint for real-time server status
- Check server logs for detailed error messages and game events
- Use `/stats` endpoint to monitor active game state
- Test reconnection scenarios with network interruptions

### Production Considerations

- Always use HTTPS in production environments
- Set up proper firewall rules for WebSocket connections
- Monitor server resources and connection limits
- Implement log rotation and monitoring systems
- Use environment variables for sensitive configuration
- Set up health checks for load balancer integration

## Game Features

### Dynamic Scaling
- Supports 4-64 players with automatic puzzle grid sizing
- Role distribution ensures balanced team composition
- Difficulty scaling affects time limits and token requirements

### Real-time Collaboration
- Live puzzle state synchronization across all clients
- Piece recommendation system for strategic coordination
- In-person communication combined with digital assistance

### Robust Reconnection
- Players and hosts can reconnect during active games
- Game state restoration based on current phase
- Automatic fragment handling for disconnected players

### Comprehensive Analytics
- Individual performance tracking across all game phases
- Team collaboration metrics and efficiency scores
- Detailed breakdowns for post-game analysis and improvement

## Support

For technical issues:
1. Check server logs for specific error messages
2. Verify configuration matches deployment environment
3. Test with minimal setup to isolate issues
4. Review firewall and network connectivity

For game balance adjustments:
1. Modify values in `constants/game_balance.go`
2. Restart server for changes to take effect
3. Test with different player counts and skill levels
4. Monitor completion rates and player feedback
