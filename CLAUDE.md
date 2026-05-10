# Project Overview

Canvas Conundrum is a collaborative multiplayer puzzle game with educational
trivia elements. Players answer trivia questions to earn resources, then work
together to assemble puzzle pieces on a shared canvas. The GitHub repository
is under the owner "MaxThePrisberry" and is called "canvas-conundrum".

# Repo layout

- `backend/` — Go WebSocket server (Dockerized)
- `frontend/` — SPA + nginx reverse proxy (Dockerized)
- `assets/puzzle-sources/` — committed source images; bind-mounted read-only
  into the backend at runtime
- `trivia/` — Open Trivia DB content, mounted into backend at runtime
- `game-config.json` — tunable game balance values, mounted at runtime
- `game-design.md`, `websocket-events.md` — gameplay and protocol specs

# Development commands

```bash
# Run the full stack with hot reload (uses docker-compose.override.yml)
docker compose up --build

# Run production-style images (no overrides)
docker compose -f docker-compose.yml up --build

# Pre-commit
pre-commit install
```

# Design

Refer to `game-design.md` and `websocket-events.md` for any question about game
design or what the code should look like. They are the source of truth — code
should be derived from these specs, not the other way around. Puzzle tiles are
generated at runtime per game; see `game-design.md` § *Asset Delivery (Puzzle
Images)* and `websocket-events.md` § *Asset Delivery (HTTP)* for the contract.
