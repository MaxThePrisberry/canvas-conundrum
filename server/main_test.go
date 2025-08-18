package main

import (
	"canvas-conundrum/config"
	"canvas-conundrum/services"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupFileLogging(t *testing.T) {
	// Create a temporary directory for test logs
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()

	// Change to temp directory to avoid cluttering the main project
	err := os.Chdir(tempDir)
	require.NoError(t, err)

	// Ensure we return to original directory after test
	defer func() {
		os.Chdir(originalDir)
	}()

	// Test setupFileLogging function
	setupFileLogging()

	// Verify logs directory was created
	logsDir := filepath.Join(tempDir, "logs")
	stat, err := os.Stat(logsDir)
	require.NoError(t, err)
	assert.True(t, stat.IsDir(), "logs directory should be created")

	// Verify log file was created
	files, err := os.ReadDir(logsDir)
	require.NoError(t, err)
	assert.Len(t, files, 1, "exactly one log file should be created")

	// Verify log file has correct naming pattern
	logFile := files[0]
	assert.True(t, strings.HasPrefix(logFile.Name(), "canvas-conundrum_"))
	assert.True(t, strings.HasSuffix(logFile.Name(), ".log"))

	// Verify log file contains expected content (the setup message)
	logPath := filepath.Join(logsDir, logFile.Name())
	content, err := os.ReadFile(logPath)
	require.NoError(t, err)

	logContent := string(content)
	assert.Contains(t, logContent, "Production logging enabled - writing to")
	assert.Contains(t, logContent, logFile.Name())
}

func TestSetupLogging_Development(t *testing.T) {
	// Create a mock config for development
	cfg := &config.Config{
		Environment: "development",
	}

	// This should not create any log files
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	err := os.Chdir(tempDir)
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	// Call setupLogging with development config
	setupLogging(cfg)

	// Verify no logs directory was created
	logsDir := filepath.Join(tempDir, "logs")
	_, err = os.Stat(logsDir)
	assert.True(t, os.IsNotExist(err), "logs directory should not be created in development")
}

func TestSetupLogging_Production(t *testing.T) {
	// Create a mock config for production
	cfg := &config.Config{
		Environment: "production",
	}

	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	err := os.Chdir(tempDir)
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	// Call setupLogging with production config
	setupLogging(cfg)

	// Verify logs directory was created
	logsDir := filepath.Join(tempDir, "logs")
	stat, err := os.Stat(logsDir)
	require.NoError(t, err)
	assert.True(t, stat.IsDir(), "logs directory should be created in production")
}

func TestInitializeServices(t *testing.T) {
	t.Run("Successful Initialization", func(t *testing.T) {
		err := initializeServices()
		assert.NoError(t, err)

		// Verify services were set up
		gameManager := services.GetGameInstance()
		assert.NotNil(t, gameManager.GetTriviaService())
		assert.NotNil(t, gameManager.GetPuzzleService())
		assert.NotNil(t, gameManager.GetBroadcastService())
		assert.NotNil(t, gameManager.GetAnalyticsService())
	})
}

func TestSetupRoutes(t *testing.T) {
	router := setupRoutes()
	assert.NotNil(t, router)

	t.Run("Health Endpoint", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/health", nil)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

		// Verify response contains expected fields
		body := rr.Body.String()
		assert.Contains(t, body, "status")
		assert.Contains(t, body, "timestamp")
		assert.Contains(t, body, "gamePhase")
		assert.Contains(t, body, "playerCount")
		assert.Contains(t, body, "hostConnected")
	})

	t.Run("WebSocket Player Endpoint", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/ws", nil)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		// Should get a bad request because it's not a proper WebSocket upgrade
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("WebSocket Host Endpoint", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/ws/host/test-uuid", nil)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		// Should get unauthorized or bad request for invalid UUID
		assert.True(t, rr.Code == http.StatusUnauthorized || rr.Code == http.StatusBadRequest)
	})

	t.Run("Static File Endpoints", func(t *testing.T) {
		// Test puzzle images endpoint
		req, err := http.NewRequest("GET", "/images/puzzle/test.jpg", nil)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		// Should get 404 since file doesn't exist, but route is handled
		assert.Equal(t, http.StatusNotFound, rr.Code)

		// Test host files endpoint
		req, err = http.NewRequest("GET", "/host/index.html", nil)
		require.NoError(t, err)

		rr = httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		// May get 301 redirect or 404 depending on file server behavior
		assert.True(t, rr.Code == http.StatusNotFound || rr.Code == http.StatusMovedPermanently)

		// Test client files endpoint
		req, err = http.NewRequest("GET", "/index.html", nil)
		require.NoError(t, err)

		rr = httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		// May get 301 redirect or 404 depending on file server behavior
		assert.True(t, rr.Code == http.StatusNotFound || rr.Code == http.StatusMovedPermanently)
	})
}

func TestHandleHealth(t *testing.T) {
	// Initialize services for a realistic health check
	err := initializeServices()
	require.NoError(t, err)

	t.Run("Valid Health Check Response", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/health", nil)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(handleHealth)
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

		// Parse JSON response
		var healthData map[string]interface{}
		body := rr.Body.String()

		// Since the response is manually formatted, let's verify it contains the right structure
		assert.Contains(t, body, `"status":"healthy"`)
		assert.Contains(t, body, `"timestamp":`)
		assert.Contains(t, body, `"gamePhase":`)
		assert.Contains(t, body, `"playerCount":`)
		assert.Contains(t, body, `"hostConnected":`)

		// Try to parse as JSON to ensure it's valid
		err = json.Unmarshal([]byte(body), &healthData)
		assert.NoError(t, err, "Health response should be valid JSON")

		// Verify expected fields
		assert.Equal(t, "healthy", healthData["status"])
		assert.NotNil(t, healthData["timestamp"])
		assert.NotNil(t, healthData["gamePhase"])
		assert.NotNil(t, healthData["playerCount"])
		assert.NotNil(t, healthData["hostConnected"])
	})
}

func TestRouteMethodRestrictions(t *testing.T) {
	router := setupRoutes()

	t.Run("Health Endpoint - POST Not Allowed", func(t *testing.T) {
		req, err := http.NewRequest("POST", "/health", strings.NewReader("{}"))
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		// Router returns 404 for unmatched routes, which is expected behavior
		assert.True(t, rr.Code == http.StatusMethodNotAllowed || rr.Code == http.StatusNotFound)
	})

	t.Run("WebSocket Endpoints - POST Not Allowed", func(t *testing.T) {
		req, err := http.NewRequest("POST", "/ws", strings.NewReader("{}"))
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		// Router returns 404 for unmatched routes, which is expected behavior
		assert.True(t, rr.Code == http.StatusMethodNotAllowed || rr.Code == http.StatusNotFound)
	})
}

func TestMainRouterIntegration(t *testing.T) {
	// This tests the overall router setup without starting the actual server
	t.Run("Router Setup Complete", func(t *testing.T) {
		router := setupRoutes()
		assert.NotNil(t, router)

		// Test that the router can handle a variety of requests without panicking
		testPaths := []string{
			"/health",
			"/ws",
			"/ws/host/invalid-uuid",
			"/images/puzzle/test.jpg",
			"/host/index.html",
			"/nonexistent-path",
		}

		for _, path := range testPaths {
			req, err := http.NewRequest("GET", path, nil)
			require.NoError(t, err)

			rr := httptest.NewRecorder()

			// Should not panic
			assert.NotPanics(t, func() {
				router.ServeHTTP(rr, req)
			}, "Router should handle path %s without panicking", path)
		}
	})
}
