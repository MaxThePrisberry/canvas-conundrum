// Package app wires the server together. Integration tests construct the
// exact same App as production main(), differing only in Options.
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/config"
	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/game"
	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/server"
	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/trivia"
	"github.com/google/uuid"
)

// Options configures an App. The timing fields default to the values fixed
// by websocket-events.md; integration tests shorten them so full games play
// out in real time in a few seconds.
type Options struct {
	Environment      string
	Port             string
	ConfigPath       string
	TriviaDir        string
	PuzzleSourcesDir string

	// HostUUID overrides the generated host token (tests only).
	HostUUID string

	// ConnectDeadline is how long a player socket may sit without sending
	// its SETUP_TO_SERVER_PLAYER_CONNECT frame before being closed (spec: 10s).
	ConnectDeadline time.Duration
	// DisconnectAfter is the heartbeat silence window after which a client
	// is treated as disconnected (spec: 90s = 3 missed 30s pings).
	DisconnectAfter time.Duration

	Logger *slog.Logger
}

func (o *Options) applyDefaults() {
	if o.Port == "" {
		o.Port = "8080"
	}
	if o.HostUUID == "" {
		o.HostUUID = uuid.NewString()
	}
	if o.ConnectDeadline == 0 {
		o.ConnectDeadline = 10 * time.Second
	}
	if o.DisconnectAfter == 0 {
		o.DisconnectAfter = 90 * time.Second
	}
	if o.Logger == nil {
		o.Logger = slog.Default()
	}
}

// App is a fully constructed, not-yet-listening server.
type App struct {
	opts   Options
	log    *slog.Logger
	engine *game.Engine
	srv    *server.Server
}

// New loads and validates all startup inputs (config, trivia, puzzle image)
// and constructs the engine and transport. Per game-design.md the server
// refuses to start on any validation failure.
func New(opts Options) (*App, error) {
	opts.applyDefaults()

	cfg, err := config.Load(opts.ConfigPath)
	if err != nil {
		return nil, err
	}
	if err := cfg.Validate(opts.PuzzleSourcesDir); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	bank, err := trivia.Load(opts.TriviaDir)
	if err != nil {
		return nil, err
	}

	engine := game.New(cfg, bank, game.Options{
		HostUUID:         opts.HostUUID,
		DisconnectAfter:  opts.DisconnectAfter,
		PuzzleSourcesDir: opts.PuzzleSourcesDir,
		Logger:           opts.Logger,
	})

	srv := server.New(engine, server.Options{
		HostUUID:        opts.HostUUID,
		ConnectDeadline: opts.ConnectDeadline,
		Logger:          opts.Logger,
	})

	return &App{opts: opts, log: opts.Logger, engine: engine, srv: srv}, nil
}

// HostUUID returns the host authentication token for this server instance.
func (a *App) HostUUID() string { return a.opts.HostUUID }

// Run listens on the configured port and serves until ctx is cancelled.
func (a *App) Run(ctx context.Context) error {
	l, err := net.Listen("tcp", ":"+a.opts.Port)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	return a.Serve(ctx, l)
}

// Serve starts the engine and serves HTTP on l until ctx is cancelled.
// Tests pass an ephemeral-port listener.
func (a *App) Serve(ctx context.Context, l net.Listener) error {
	// The host connects via this UUID; it is generated fresh per server
	// start and shared only through this log line.
	a.log.Info("host connection URL generated", "path", "/ws/host/"+a.opts.HostUUID)
	a.log.Info("server listening", "addr", l.Addr().String(), "environment", a.opts.Environment)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go a.engine.Run(ctx)

	srv := &http.Server{Handler: a.srv.Handler()}
	errCh := make(chan error, 1)
	go func() { errCh <- srv.Serve(l) }()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
