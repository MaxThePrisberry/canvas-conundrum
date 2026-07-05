// Canvas Conundrum game server.
//
// Environment (set by docker-compose):
//
//	ENVIRONMENT        production | development
//	PORT               listen port (default 8080)
//	PUZZLE_SOURCES_DIR directory of source puzzle images
//	TRIVIA_DIR         directory of trivia pools ({category}/{difficulty}.json)
//	CONFIG_PATH        path to game-config.json
//
// The -health-check flag makes the binary probe a running instance and exit
// 0/1; docker-compose uses it as the container healthcheck.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/app"
)

func main() {
	healthCheck := flag.Bool("health-check", false, "probe the running server's /healthz and exit")
	flag.Parse()

	port := envOr("PORT", "8080")

	if *healthCheck {
		os.Exit(runHealthCheck(port))
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	opts := app.Options{
		Environment:      envOr("ENVIRONMENT", "development"),
		Port:             port,
		ConfigPath:       envOr("CONFIG_PATH", "game-config.json"),
		TriviaDir:        envOr("TRIVIA_DIR", "trivia"),
		PuzzleSourcesDir: envOr("PUZZLE_SOURCES_DIR", "assets/puzzle-sources"),
		Logger:           logger,
	}

	a, err := app.New(opts)
	if err != nil {
		logger.Error("startup failed", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := a.Run(ctx); err != nil {
		logger.Error("server exited", "error", err)
		os.Exit(1)
	}
}

// runHealthCheck probes the local server over HTTP rather than just dialing
// TCP so a passing check proves the mux is actually serving requests.
func runHealthCheck(port string) int {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%s/healthz", port))
	if err != nil {
		fmt.Fprintln(os.Stderr, "health check failed:", err)
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintln(os.Stderr, "health check failed: status", resp.StatusCode)
		return 1
	}
	return 0
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
