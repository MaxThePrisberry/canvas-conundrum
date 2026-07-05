package server

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/MaxThePrisberry/canvas-conundrum/backend/internal/game"
)

// Options configures the transport layer.
type Options struct {
	HostUUID string
	// ConnectDeadline bounds the wait for a player's first frame (spec: 10s).
	ConnectDeadline time.Duration
	Logger          *slog.Logger
}

// Server owns the HTTP mux: WebSocket endpoints, asset endpoints, healthz.
type Server struct {
	engine          *game.Engine
	hostUUID        string
	connectDeadline time.Duration
	log             *slog.Logger
	mux             *http.ServeMux
}

func New(engine *game.Engine, opts Options) *Server {
	s := &Server{
		engine:          engine,
		hostUUID:        opts.HostUUID,
		connectDeadline: opts.ConnectDeadline,
		log:             opts.Logger,
	}
	if s.log == nil {
		s.log = slog.Default()
	}

	s.mux = http.NewServeMux()
	s.mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	s.mux.HandleFunc("GET /ws", s.handlePlayer)
	s.mux.HandleFunc("GET /ws/host/{uuid}", s.handleHost)
	s.mux.HandleFunc("GET /api/segments/{segmentId}", s.handleSegment)
	s.mux.HandleFunc("GET /api/preview/full", s.handlePreview)
	return s
}

// Handler exposes the mux for http.Server / tests.
func (s *Server) Handler() http.Handler { return s.mux }
