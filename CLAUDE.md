# Project Overview

Canvas Conundrum is a collaborative multiplayer puzzle game with educational
trivia elements. Players answer trivia questions to earn resources, then work
together to assemble puzzle pieces on a shared canvas. The GitHub repository
is under the owner "MaxThePrisberry" and is called "canvas-conundrum".

# Repo layout

- `backend/` — Go WebSocket server (Dockerized)
- `frontend/` — SPA + nginx reverse proxy (Dockerized)
- `assets/puzzle-sources/` — committed source images; split into tiles at
  backend build time and baked into the backend image
- `trivia/` — Open Trivia DB content, mounted into backend at runtime
- `game-config.json` — tunable game balance values, mounted at runtime
- `game-design.md`, `websocket-events.md` — gameplay and protocol specs
- `preprocess_puzzle_images.py` — splits a source image into 3×3–8×8 tile grids

# Development commands

```bash
# Run the full stack with hot reload (source bind-mounted)
docker compose up --build

# Run production-style images
docker compose -f docker-compose.yml up --build

# Re-split a puzzle source image locally without rebuilding the backend
python preprocess_puzzle_images.py assets/puzzle-sources/nature_image.png

# Pre-commit
pre-commit install
```

# Design

Refer to `game-design.md` and `websocket-events.md` for any question about game
design or what the code should look like. They are the source of truth — code
should be derived from these specs, not the other way around.

# Image strategy

Puzzle tiles are generated at backend build time and baked into the backend
image. They are never committed to git and never served as static files — the
Go server gates all tile fetches behind session auth so each client only ever
receives its own assigned segment. The clarity-token full-image preview is
similarly server-gated to a limited time window.
