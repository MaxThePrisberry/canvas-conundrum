# Canvas Conundrum

Collaborative multiplayer puzzle game with educational trivia.

## Repo layout

```
canvas-conundrum/
├── game-design.md            # Gameplay spec — the source of truth
├── websocket-events.md       # WebSocket event protocol
├── game-config.json          # Tunable game balance values (mounted at runtime)
│
├── backend/                  # Go WebSocket server (rebuild in progress)
│   └── Dockerfile
├── frontend/                 # SPA (rebuild in progress)
│   ├── Dockerfile
│   └── nginx.conf            # Reverse-proxies /ws and /api to backend
│
├── assets/
│   └── puzzle-sources/       # Source puzzle images (committed)
│       └── nature_image.png
│
├── trivia/                   # Open Trivia DB content (CC-BY-SA)
│   └── {category}/{difficulty}.json
│
├── docker-compose.yml            # Production
└── docker-compose.override.yml   # Development (hot reload + bind mounts)
```

## Image strategy

Puzzle tiles are **generated at runtime, exactly once per game**, when the
resource-gathering phase ends and the server begins preparing the puzzle
phase. The Go server crops the source image into per-segment tiles using
the standard library, holds them in memory, and serves them only through
authenticated `/api/...` endpoints. Players never receive segment images
they do not own; the full-image clarity preview is bounded to a
server-controlled time window.

Source images live under `assets/puzzle-sources/` and are bind-mounted into
the backend read-only at `/app/puzzle-sources/`. To add a new puzzle: drop
a file in there and restart the container — no rebuild needed. The server
validates the directory on startup and refuses to boot if no usable images
are present.

## Running

```bash
# Development (hot reload, source bind-mounted)
docker compose up --build

# Production-style (immutable images, no source mounts)
docker compose -f docker-compose.yml up --build
```

Frontend on `http://localhost:8080` (prod) or `http://localhost:5173` (dev).
Backend exposed directly on `:8081` in dev for debugging.

## Setup (host)

```bash
# Pre-commit
sudo apt install pre-commit -y
pre-commit install
```

## Attribution

Trivia content from [Open Trivia Database](https://opentdb.com/) under
[CC-BY-SA 4.0](https://creativecommons.org/licenses/by-sa/4.0/), unmodified.
