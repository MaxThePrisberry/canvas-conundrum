package main

import (
	"bufio" // Added for Hijacker
	"context"
	"encoding/json"
	"errors" // Added for Hijacker error
	"flag"
	"fmt"
	"log"
	"net" // Added for Hijacker
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/MaxThePrisberry/canvas-conundrum/server/constants"
	"github.com/google/uuid"
)

var (
	port           = flag.String("port", "8080", "Server port")
	host           = flag.String("host", "0.0.0.0", "Server host")
	certFile       = flag.String("cert", "", "TLS certificate file (optional)")
	keyFile        = flag.String("key", "", "TLS key file (optional)")
	allowedOrigins = flag.String("origins", "", "Comma-separated list of allowed CORS origins (empty for development mode)")
	environment    = flag.String("env", "development", "Environment (development, staging, production)")
)

// Global host endpoint identifier - generated on server start
var hostEndpointID string

func main() {
	flag.Parse()

	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.Printf("Canvas Conundrum Server starting on %s:%s (env: %s)", *host, *port, *environment)

	// Generate unique host endpoint ID
	hostEndpointID = uuid.New().String()
	log.Printf("式 HOST ENDPOINT: /ws/host/%s", hostEndpointID)
	log.Printf("則 PLAYER ENDPOINT: /ws")

	if *environment == "development" {
		log.Printf("迫 Host URL: ws://localhost:%s/ws/host/%s", *port, hostEndpointID)
		log.Printf("迫 Player URL: ws://localhost:%s/ws", *port)
	}

	// Initialize CORS configuration
	initializeCORS()

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
	totalQuestions := 0
	for category, difficulties := range stats {
		categoryTotal := 0
		for _, count := range difficulties {
			categoryTotal += count
			totalQuestions += count
		}
		log.Printf("  %s: %d questions", category, categoryTotal)
	}
	log.Printf("Total questions loaded: %d", totalQuestions)

	// Validate trivia configuration
	if totalQuestions == 0 {
		log.Fatal("No trivia questions loaded. Please check trivia directory structure and files.")
	}

	// Set up HTTP routes
	mux := http.NewServeMux()

	// Player WebSocket endpoint (regular players)
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		wsHandler.HandleConnection(w, r, false) // false = not host
	})

	// Host WebSocket endpoint (host only)
	mux.HandleFunc("/ws/host/"+hostEndpointID, func(w http.ResponseWriter, r *http.Request) {
		wsHandler.HandleConnection(w, r, true) // true = is host
	})

	// Health check endpoint with detailed information including host endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Gather detailed health information
		connectedPlayers := playerManager.GetConnectedCount()
		readyPlayers := playerManager.GetReadyCount()
		phase := gameManager.GetPhase()
		hasHost := playerManager.GetHost() != nil

		healthStatus := map[string]interface{}{
			"status":      "healthy",
			"timestamp":   time.Now().Unix(),
			"version":     "1.0.0", // Could be set via build flags
			"environment": *environment,
			"endpoints": map[string]interface{}{
				"players": "/ws",
				"host":    "/ws/host/" + hostEndpointID,
			},
			"game": map[string]interface{}{
				"phase":   phase.String(),
				"hasHost": hasHost,
				"players": map[string]int{
					"total":     playerManager.GetPlayerCount(),
					"connected": connectedPlayers,
					"ready":     readyPlayers,
				},
			},
			"trivia": map[string]interface{}{
				"totalQuestions": totalQuestions,
				"categories":     len(stats),
			},
		}

		// Add additional status based on game phase
		if phase == PhaseResourceGathering || phase == PhasePuzzleAssembly {
			gameManager.mu.RLock()
			healthStatus["game"].(map[string]interface{})["currentRound"] = gameManager.state.CurrentRound
			healthStatus["game"].(map[string]interface{})["teamTokens"] = gameManager.state.TeamTokens
			gameManager.mu.RUnlock()
		}

		json.NewEncoder(w).Encode(healthStatus)
	})

	// Game statistics endpoint with enhanced data
	mux.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		gameManager.mu.RLock()
		stats := map[string]interface{}{
			"game": map[string]interface{}{
				"phase":      gameManager.state.Phase.String(),
				"difficulty": gameManager.state.Difficulty,
				"round":      gameManager.state.CurrentRound,
				"teamTokens": gameManager.state.TeamTokens,
				"gridSize":   gameManager.state.GridSize,
			},
			"players": map[string]interface{}{
				"total":     playerManager.GetPlayerCount(),
				"connected": playerManager.GetConnectedCount(),
				"ready":     playerManager.GetReadyCount(),
				"roles":     playerManager.GetRoleDistribution(),
				"hasHost":   playerManager.GetHost() != nil,
			},
			"server": map[string]interface{}{
				"uptime":       time.Since(startTime).Seconds(),
				"environment":  *environment,
				"hostEndpoint": "/ws/host/" + hostEndpointID,
			},
		}
		gameManager.mu.RUnlock()

		json.NewEncoder(w).Encode(stats)
	})

	// Admin endpoints for production management
	if *environment == "production" {
		mux.HandleFunc("/admin/reload-trivia", adminAuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}

			if err := triviaManager.ReloadQuestions(); err != nil {
				http.Error(w, fmt.Sprintf("Failed to reload trivia: %v", err), http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"status":  "success",
				"message": "Trivia questions reloaded",
			})
		}))

		// Admin endpoint to get host endpoint (useful for deployment management)
		mux.HandleFunc("/admin/host-endpoint", adminAuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"hostEndpoint": "/ws/host/" + hostEndpointID,
				"hostURL":      fmt.Sprintf("ws://%s:%s/ws/host/%s", *host, *port, hostEndpointID),
			})
		}))
	}

	// Apply middleware chain
	// The order is: loggingMiddleware(securityHeadersMiddleware(corsMiddleware(mux)))
	// This means loggingMiddleware is the outermost, then security, then cors, then the mux.
	handler := loggingMiddleware(securityHeadersMiddleware(corsMiddleware(mux)))

	// Create server with enhanced configuration
	srv := &http.Server{
		Addr:              fmt.Sprintf("%s:%s", *host, *port),
		Handler:           handler,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1MB
	}

	// Start server in a goroutine
	go func() {
		var err error
		if *certFile != "" && *keyFile != "" {
			log.Printf("Starting HTTPS server on %s", srv.Addr)
			err = srv.ListenAndServeTLS(*certFile, *keyFile)
		} else {
			if *environment == "production" {
				log.Println("WARNING: Running production server without HTTPS")
			}
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

	// Shutdown TriviaManager first to stop background goroutines
	triviaManager.Shutdown()

	// Close broadcast channel
	close(broadcastChan)

	// Notify all players of shutdown
	shutdownMessage := map[string]string{
		"error": "Server shutting down for maintenance",
		"type":  "server_shutdown",
	}

	for _, player := range playerManager.GetAllPlayers() {
		sendToPlayer(player, MsgError, shutdownMessage)
	}

	// Allow time for messages to be sent
	time.Sleep(2 * time.Second)

	// Shutdown HTTP server
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}

// Global variables for server lifecycle
var startTime = time.Now()

// CORS configuration
var (
	corsAllowedOrigins map[string]bool
	corsAllowAll       bool
)

// initializeCORS sets up CORS configuration based on environment and flags
func initializeCORS() {
	corsAllowedOrigins = make(map[string]bool)

	origins := *allowedOrigins
	if origins == "" {
		origins = os.Getenv("ALLOWED_ORIGINS")
	}

	if origins == "" {
		if *environment == "development" {
			corsAllowAll = false
			corsAllowedOrigins["http://localhost:3000"] = true
			corsAllowedOrigins["http://localhost:5173"] = true
			corsAllowedOrigins["http://localhost:8080"] = true
			corsAllowedOrigins["http://127.0.0.1:3000"] = true
			corsAllowedOrigins["http://127.0.0.1:5173"] = true
			corsAllowedOrigins["http://127.0.0.1:8080"] = true
			log.Println("CORS: Using development origins")
		} else {
			log.Fatal("CORS: No allowed origins specified for production environment. Use -origins flag or ALLOWED_ORIGINS env var")
		}
	} else if origins == "*" {
		if *environment != "development" {
			log.Fatal("CORS: Wildcard origins not allowed in production")
		}
		corsAllowAll = true
		log.Println("CORS: WARNING - Allowing all origins (development only)")
	} else {
		corsAllowAll = false
		for _, origin := range strings.Split(origins, ",") {
			origin = strings.TrimSpace(origin)
			if origin != "" {
				corsAllowedOrigins[origin] = true
			}
		}
		log.Printf("CORS: Configured %d allowed origins", len(corsAllowedOrigins))
	}
}

// isValidOrigin checks if an origin is allowed for CORS
func isValidOrigin(origin string) bool {
	if corsAllowAll {
		return true
	}
	if origin == "" {
		return false
	}
	return corsAllowedOrigins[origin]
}

// Enhanced CORS middleware with proper security
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		if isValidOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else if origin != "" {
			log.Printf("CORS: Rejected origin: %s", origin)
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Security headers middleware
func securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		if *environment == "production" {
			w.Header().Set("Content-Security-Policy", "default-src 'self'")
		}

		if r.TLS != nil {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		next.ServeHTTP(w, r)
	})
}

// responseWriter wrapper to capture status codes and support hijacking
type responseWriter struct {
	http.ResponseWriter
	statusCode  int
	wroteHeader bool
}

// WriteHeader captures the status code before writing it to the underlying writer.
func (rw *responseWriter) WriteHeader(code int) {
	if rw.wroteHeader {
		return
	}
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
	rw.wroteHeader = true
}

// Write ensures WriteHeader is called if it hasn't been, then writes the bytes.
func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK) // Default to 200 if not set
	}
	return rw.ResponseWriter.Write(b)
}

// Hijack implements the http.Hijacker interface.
func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := rw.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("http.Hijacker interface is not supported by the underlying ResponseWriter")
	}
	return h.Hijack()
}

// Request logging middleware using the updated responseWriter
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapper := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapper, r)

		duration := time.Since(start)
		// For WebSocket upgrades, the status code (101) is handled after hijacking.
		// The wrapper.statusCode might not reflect the 101 if the hijack was successful
		// before the wrapper could capture a header write for that.
		// If no header was written by the wrapped handlers before hijacking,
		// statusCode will remain the default (200).
		// This logging is primarily for regular HTTP requests or failed WS upgrades.
		log.Printf("%s %s %d %v %s",
			r.Method,
			r.URL.Path,
			wrapper.statusCode,
			duration,
			r.Header.Get("User-Agent"))
	})
}

// Admin authentication middleware
func adminAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		expectedToken := os.Getenv("ADMIN_TOKEN")

		if expectedToken == "" {
			http.Error(w, "Admin endpoints disabled", http.StatusNotFound)
			return
		}
		if authHeader != "Bearer "+expectedToken {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}
