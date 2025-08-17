package main

import (
	"canvas-conundrum/config"
	"canvas-conundrum/handlers"
	"canvas-conundrum/services"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Setup logging
	setupLogging(cfg)

	log.Printf("Starting Canvas Conundrum Server on port %s in %s mode", cfg.Port, cfg.Environment)

	// Initialize services
	if err := initializeServices(); err != nil {
		log.Fatalf("Failed to initialize services: %v", err)
	}

	// Setup routes
	router := setupRoutes()

	// Setup CORS
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "http://localhost:3001"}, // Client and Host frontends
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
		MaxAge:           300,
	})

	handler := c.Handler(router)

	// Create HTTP server
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Server listening on port %s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

func setupLogging(cfg *config.Config) {
	// Set log format
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	// In production, you might want to log to a file
	if cfg.IsProduction() {
		// TODO: Setup file logging
	}
}

func initializeServices() error {
	// Initialize game manager (singleton)
	gameManager := services.GetGameInstance()

	// Initialize trivia service
	triviaService := services.NewTriviaService()
	if err := triviaService.LoadQuestions(); err != nil {
		return fmt.Errorf("failed to load trivia questions: %w", err)
	}

	// Set services in game manager
	gameManager.SetTriviaService(triviaService)
	gameManager.SetPuzzleService(services.NewPuzzleService())
	gameManager.SetBroadcastService(services.NewBroadcastService())
	gameManager.SetAnalyticsService(services.NewAnalyticsService())

	log.Println("All services initialized successfully")
	return nil
}

func setupRoutes() *mux.Router {
	r := mux.NewRouter()

	// Health check endpoint
	r.HandleFunc("/health", handleHealth).Methods("GET")

	// WebSocket endpoints
	r.HandleFunc("/ws", handlers.HandlePlayerWebSocket).Methods("GET")
	r.HandleFunc("/ws/host/{uuid}", handlers.HandleHostWebSocket).Methods("GET")

	// Serve puzzle images
	r.PathPrefix("/images/puzzle/").Handler(
		http.StripPrefix("/images/puzzle/",
			http.FileServer(http.Dir("./puzzle_images/puzzle_segments/"))))

	r.PathPrefix("/host/").Handler(http.StripPrefix("/host/", http.FileServer(http.Dir("./public/host/"))))

	// Serve client static files (if built)
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("./public/client/")))

	return r
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	gameManager := services.GetGameInstance()

	health := map[string]interface{}{
		"status":        "healthy",
		"timestamp":     time.Now().Unix(),
		"gamePhase":     gameManager.GetCurrentPhase(),
		"playerCount":   gameManager.GetPlayerCount(),
		"hostConnected": gameManager.IsHostConnected(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Simple JSON encoding for health check
	fmt.Fprintf(w, `{"status":"%s","timestamp":%d,"gamePhase":"%s","playerCount":%d,"hostConnected":%v}`,
		health["status"], health["timestamp"], health["gamePhase"],
		health["playerCount"], health["hostConnected"])
}
