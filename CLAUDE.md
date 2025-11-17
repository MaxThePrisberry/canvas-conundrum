# Project Overview

Canvas Conundrum is a collaborative multiplayer puzzle game with educational trivia elements. Players answer trivia questions to earn resources, then work together to assemble puzzle pieces on a shared canvas. The GitHub repository is under the owner "MaxThePrisberry" and is called "canvas-conundrum".
# Development Commands

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

# Design
Refer to `game-design.md` and `websocket-events.md` whenever there is a question about game design or what the code should look like.
