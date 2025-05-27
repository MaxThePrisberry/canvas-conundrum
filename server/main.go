package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MaxThePrisberry/canvas-conundrum/server/constants"
)

var (
	port     = flag.String("port", "8080", "Server port")
	host     = flag.String("host", "0.0.0.0", "Server host")
	certFile = flag.String("cert", "", "TLS certificate file (optional)")
	keyFile  = flag.String("key", "", "TLS key file (optional)")
)

func main() {
	flag.Parse()

	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.Printf("Canvas Conundrum Server starting on %s:%s", *host, *port)

	// Initialize components
	broadcastChan := make(chan BroadcastMessage, constants.BroadcastChannelBuffer)
	playerManager := NewPlayerManager()
	triviaManager := NewTriviaManager()
	gameManager := NewGameManager(playerManager, triviaManager, broadcastChan)
	eventHandlers := NewEventHandlers(gameManager, playerManager, broadcastChan)
	wsHandler := NewWebSocketHandler(playerManager, gameManager, eventHandlers, broadcastChan)

	// Start broadcaster
	wsHandler.StartBroadcaster()

	// Log trivia statistics
	stats := triviaManager.GetCategoryStats()
	log.Printf("Loaded trivia questions:")
	for category, difficulties := range stats {
		total := 0
		for _, count := range difficulties {
			total += count
		}
		log.Printf("  %s: %d questions", category, total)
	}

	// Set up HTTP routes
	mux := http.NewServeMux()

	// WebSocket endpoint
	mux.HandleFunc("/ws", wsHandler.HandleConnection)

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "healthy",
			"phase":  gameManager.GetPhase().String(),
			"players": map[string]int{
				"total":     playerManager.GetPlayerCount(),
				"connected": playerManager.GetConnectedCount(),
				"ready":     playerManager.GetReadyCount(),
			},
		})
	})

	// Game statistics endpoint
	mux.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		gameManager.mu.RLock()
		stats := map[string]interface{}{
			"phase":       gameManager.state.Phase.String(),
			"difficulty":  gameManager.state.Difficulty,
			"round":       gameManager.state.CurrentRound,
			"teamTokens":  gameManager.state.TeamTokens,
			"gridSize":    gameManager.state.GridSize,
			"playerCount": playerManager.GetPlayerCount(),
		}
		gameManager.mu.RUnlock()

		json.NewEncoder(w).Encode(stats)
	})

	// CORS middleware (for development)
	handler := corsMiddleware(mux)

	// Create server
	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", *host, *port),
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		var err error
		if *certFile != "" && *keyFile != "" {
			log.Printf("Starting HTTPS server on %s", srv.Addr)
			err = srv.ListenAndServeTLS(*certFile, *keyFile)
		} else {
			log.Printf("Starting HTTP server on %s", srv.Addr)
			err = srv.ListenAndServe()
		}

		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Close broadcast channel
	close(broadcastChan)

	// Notify all players of shutdown
	for _, player := range playerManager.GetAllPlayers() {
		sendToPlayer(player, MsgError, map[string]string{
			"error": "Server shutting down",
		})
	}

	// Shutdown HTTP server
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}

// corsMiddleware adds CORS headers for development
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// In production, replace "*" with specific allowed origins
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
