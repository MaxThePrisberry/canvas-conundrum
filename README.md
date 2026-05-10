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

## Prerequisites

Requires Docker with Compose v2 (`docker compose ...`, not the legacy
`docker-compose`).

```bash
sudo apt install pre-commit -y
pre-commit install
```

## Running

```bash
# Development (hot reload)
docker compose up --build

# Production
docker compose -f docker-compose.yml up --build
```

Frontend on `http://localhost:8080` (prod) or `http://localhost:5173` (dev).
Backend exposed directly on `:8081` in dev for debugging.

## Attribution

Trivia content from [Open Trivia Database](https://opentdb.com/) under
[CC-BY-SA 4.0](https://creativecommons.org/licenses/by-sa/4.0/), unmodified.
