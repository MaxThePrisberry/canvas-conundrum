# Multi-stage build for the Canvas Conundrum backend.
#
# Stage 1 (tiles):  Run preprocess_puzzle_images.py over every source image
#                   in assets/puzzle-sources/. Output is baked into the final
#                   image so the Go server can serve tiles through gated
#                   endpoints — clients never get static access to them.
#
# Stage 2 (deps):   Cache Go module downloads.
#
# Stage 3 (build):  Compile the Go binary.
#
# Stage 4 (dev):    Hot-reload variant used by docker-compose.override.yml.
#
# Stage 5 (runtime): Distroless final image — no shell, no package manager,
#                    just the binary, the tiles, and runtime config paths.

# ──────────────────────────────────────────────────────────────────────────
# Stage 1: generate puzzle tiles
# ──────────────────────────────────────────────────────────────────────────
FROM python:3.12-slim AS tiles
WORKDIR /work
RUN pip install --no-cache-dir Pillow==10.4.0
COPY preprocess_puzzle_images.py .
COPY assets/puzzle-sources/ ./puzzle-sources/
RUN mkdir -p /out && \
    for img in puzzle-sources/*; do \
      [ -f "$img" ] || continue; \
      python preprocess_puzzle_images.py "$img" -o /out; \
    done && \
    echo "Generated tiles:" && find /out -type f | wc -l

# ──────────────────────────────────────────────────────────────────────────
# Stage 2: Go module cache
# ──────────────────────────────────────────────────────────────────────────
FROM golang:1.23-alpine AS deps
WORKDIR /src
COPY backend/go.mod backend/go.sum* ./
RUN go mod download

# ──────────────────────────────────────────────────────────────────────────
# Stage 3: build static binary
# ──────────────────────────────────────────────────────────────────────────
FROM deps AS build
COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -trimpath \
    -o /out/server ./cmd/server

# ──────────────────────────────────────────────────────────────────────────
# Stage 4: dev image with hot reload (used only by docker-compose.override)
# ──────────────────────────────────────────────────────────────────────────
FROM deps AS dev
RUN go install github.com/air-verse/air@latest
WORKDIR /src
EXPOSE 8080
# Source is bind-mounted at runtime; entrypoint comes from compose `command`.

# ──────────────────────────────────────────────────────────────────────────
# Stage 5: minimal runtime
# ──────────────────────────────────────────────────────────────────────────
FROM gcr.io/distroless/static-debian12:nonroot AS runtime
WORKDIR /app
COPY --from=build  /out/server /app/server
COPY --from=tiles  /out        /app/puzzle-tiles
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/server"]
