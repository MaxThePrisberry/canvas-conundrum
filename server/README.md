# Canvas Conundrum Server

A Go-based WebSocket server for the Canvas Conundrum collaborative puzzle game.

## Prerequisites

- Go 1.21 or higher
- Git

## Installation

1. Clone the repository and navigate to the server directory:
```bash
cd server
```

2. Install dependencies:
```bash
go mod download
```

3. Create the trivia questions directory structure:
```bash
mkdir -p trivia/{general,geography,history,music,science,video_games}/{easy,medium,hard}
```

4. Add trivia questions JSON files to each directory. The format should match the provided `medium.json` example.

## Running the Server

### Development
```bash
# Basic development server (allows localhost origins)
go run .

# Custom port
go run . -port=3000
```

### Production
```bash
# With HTTPS and custom origins
go run . -env=production -cert=cert.pem -key=key.pem -origins="https://yourdomain.com"

# Build and run binary
go build -o canvas-conundrum-server
./canvas-conundrum-server -env=production
```

## Configuration

### Command Line Options
```bash
  -env string
        Environment: development, staging, production (default "development")
  -port string
        Server port (default "8080")
  -host string
        Server host (default "0.0.0.0")
  -origins string
        Comma-separated allowed CORS origins (auto-configured for development)
  -cert string
        TLS certificate file
  -key string
        TLS key file
```

### Environment Variables
```bash
# CORS origins (required for production)
ALLOWED_ORIGINS="https://yourdomain.com,https://www.yourdomain.com"

# Admin token (optional, enables admin endpoints)
ADMIN_TOKEN="your-secure-token"
```

## Game Configuration

Adjust game balance in `constants/game_balance.go`:
- Player limits, time durations, token thresholds
- Trivia question intervals and scoring
- Puzzle mechanics and grid scaling

## API Endpoints

- `GET /health` - Server health check
- `GET /stats` - Game statistics
- `GET /ws` - WebSocket connection (with optional `?playerId=<id>` for reconnection)
- `POST /admin/reload-trivia` - Reload trivia questions (requires admin token)

## Trivia Questions

The server automatically cycles through trivia questions:
- Questions are shuffled into pools per category/difficulty
- When a pool is exhausted, it automatically resets and reshuffles
- No manual intervention required - questions will never run out

### Question File Format
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

## Deployment

### Development
```bash
go run . -env=development
```
- Automatically allows localhost origins
- HTTP only
- Verbose logging

### Production
```bash
export ALLOWED_ORIGINS="https://yourdomain.com"
export ADMIN_TOKEN="$(openssl rand -base64 32)"

./canvas-conundrum-server \
  -env=production \
  -cert=cert.pem \
  -key=key.pem \
  -port=8080
```
- Requires explicit CORS origins
- HTTPS recommended
- Security headers enabled

## Game Features

- **Multi-phase gameplay**: Setup → Resource Gathering → Puzzle Assembly → Analytics
- **Dynamic scaling**: Supports 4-64 players with auto-adjusting puzzle grids
- **Reconnection support**: Players can rejoin during active games
- **Real-time updates**: All game state synchronized via WebSocket
- **Comprehensive analytics**: Individual and team performance tracking

## Troubleshooting

### Common Issues
- **"Cannot load trivia questions"**: Check that JSON files exist in all `trivia/{category}/{difficulty}/` directories
- **"WebSocket connection failed"**: Verify CORS origins are configured correctly
- **"CORS origin rejected"**: Add your frontend domain to `ALLOWED_ORIGINS`

### Development Tips
- Use `-env=development` for automatic localhost CORS handling
- Check `/health` endpoint for server status and trivia question counts
- Monitor server logs for detailed error messages

## Project Structure
```
server/
├── main.go                    # Server entry point
├── websocket_handlers.go      # WebSocket connection handling
├── event_handlers.go          # Game event processing
├── game_manager.go            # Core game logic
├── trivia_manager.go          # Question loading and cycling
├── player_manager.go          # Player state management
├── validation.go              # Input validation
├── constants/game_balance.go  # Game configuration
└── trivia/                    # Question files
    ├── general/{easy,medium,hard}.json
    ├── geography/{easy,medium,hard}.json
    └── ...
```

## License

See main project LICENSE file.
