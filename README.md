# Canvas Conundrum

Collaborative multiplayer puzzle game with educational trivia. Design docs:
`game-design.md` (gameplay) and `websocket-events.md` (protocol) — the source
of truth. Tunables live in `game-config.json`, mounted at runtime.

Repo layout: `backend/` (Go WebSocket server), `frontend/` (React SPA behind
nginx), `assets/puzzle-sources/` (puzzle images), `trivia/` (question pools).

## Run

Requires Docker with Compose v2.

```bash
# Development (hot reload)
docker compose up --build

# Production
docker compose -f docker-compose.yml up --build
```

| URL | Purpose |
|---|---|
| `http://localhost:5173/` (dev) / `http://localhost:8080/` (prod) | Players |
| same origin + `/host` | Host console |
| `http://localhost:8081/` (dev only) | Backend directly, for debugging |

The host console asks for the host UUID — copy it from the backend log line
`host connection URL generated` (a fresh UUID is printed on every server
start).

## Tests

```bash
cd backend && go test ./...    # unit + full-game integration tests
cd frontend && npm test        # unit tests for the pure client logic
```

## Attribution

Trivia content from [Open Trivia Database](https://opentdb.com/) under
[CC-BY-SA 4.0](https://creativecommons.org/licenses/by-sa/4.0/), unmodified.
