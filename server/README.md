# Running Server

The server will start on port 8080 by default and provide:
- WebSocket endpoints at `/ws` (players) and `/ws/host/{uuid}` (host)
- Health check at `/health`
- Puzzle images at `/images/puzzle/`
- CORS configured for localhost:3000 and localhost:3001

## Direct Go Execution (Simplest)
```bash
# Install dependencies first
go mod download

# Run the server with default settings
go run .

# Or with custom configuration
go run . -env=development -port=8080
```

## Using Make (Recommended)
```bash
# Build and run the server
make run

# Or just build the binary
make build
./canvas-conundrum-server
```

## Docker Container
```bash
# Build the Docker image
docker build -t canvas-conundrum-server .

# Run the container
docker run -p 8080:8080 canvas-conundrum-server

# Or run with custom environment
docker run -p 8080:8080 -e ENV=development canvas-conundrum-server
```

The Dockerfile is already configured to:
- Use Go 1.21 Alpine image
- Download dependencies
- Build the binary
- Expose port 8080
- Run in production mode by default
