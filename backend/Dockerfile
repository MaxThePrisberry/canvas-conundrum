# Multi-stage build for the Canvas Conundrum backend.
#
# Puzzle source images are NOT baked into this image. They are bind-mounted
# at runtime from ./assets/puzzle-sources/ into /app/puzzle-sources/ so new
# puzzles can be added without a rebuild. The Go server splits sources into
# per-segment tiles in memory at the start of the puzzle preparation phase,
# and serves them only through authenticated /api/... endpoints.
#
# Stage 1 (deps):    Cache Go module downloads.
# Stage 2 (build):   Compile the Go binary.
# Stage 3 (dev):     Hot-reload variant used by docker-compose.override.yml.
# Stage 4 (runtime): Distroless final image — no shell, no package manager,
#                    just the binary.

# ──────────────────────────────────────────────────────────────────────────
# Stage 1: Go module cache
# ──────────────────────────────────────────────────────────────────────────
FROM golang:1.23-alpine AS deps
WORKDIR /src
COPY backend/go.mod backend/go.sum* ./
RUN go mod download

# ──────────────────────────────────────────────────────────────────────────
# Stage 2: build static binary
# ──────────────────────────────────────────────────────────────────────────
FROM deps AS build
COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -trimpath \
    -o /out/server ./cmd/server

# ──────────────────────────────────────────────────────────────────────────
# Stage 3: dev image with hot reload (used only by docker-compose.override)
# ──────────────────────────────────────────────────────────────────────────
FROM deps AS dev
RUN go install github.com/air-verse/air@latest
WORKDIR /src
EXPOSE 8080
# Source is bind-mounted at runtime; entrypoint comes from compose `command`.

# ──────────────────────────────────────────────────────────────────────────
# Stage 4: minimal runtime
# ──────────────────────────────────────────────────────────────────────────
FROM gcr.io/distroless/static-debian12:nonroot AS runtime
WORKDIR /app
COPY --from=build /out/server /app/server
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/server"]
