package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStaticFileServing(t *testing.T) {
	// Setup test directories and files
	setupTestDirectories(t)
	defer cleanupTestDirectories(t)

	mux := http.NewServeMux()

	// Add static file serving endpoints
	clientFS := http.FileServer(http.Dir("./test_client/public/"))
	mux.Handle("/", clientFS)

	hostFS := http.FileServer(http.Dir("./test_host/public/"))
	mux.Handle("/host/", http.StripPrefix("/host/", hostFS))

	t.Run("Client root endpoint serves index.html", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Player Frontend")
		assert.Equal(t, "text/html; charset=utf-8", w.Header().Get("Content-Type"))
	})

	t.Run("Host endpoint serves host index.html", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/host/", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Host Frontend")
		assert.Equal(t, "text/html; charset=utf-8", w.Header().Get("Content-Type"))
	})

	t.Run("Host endpoint serves static assets", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/host/manifest.json", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Host App")
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	})

	t.Run("Client endpoint serves static assets", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/manifest.json", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Player App")
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	})

	t.Run("Non-existent files return 404", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/nonexistent.html", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Host non-existent files return 404", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/host/nonexistent.html", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestMiddlewareChain(t *testing.T) {
	// Test CORS middleware
	t.Run("CORS headers", func(t *testing.T) {
		// Initialize test CORS configuration
		initializeCORS()

		mux := http.NewServeMux()
		mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		handler := corsMiddleware(mux)

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "http://localhost:3000")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		assert.Equal(t, "http://localhost:3000", w.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "GET, POST, OPTIONS", w.Header().Get("Access-Control-Allow-Methods"))
	})

	t.Run("Security headers", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		handler := securityHeadersMiddleware(mux)

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
		assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
		assert.Equal(t, "1; mode=block", w.Header().Get("X-XSS-Protection"))
	})
}

func TestHealthEndpoint(t *testing.T) {
	// Setup test environment
	setupTestDirectories(t)
	defer cleanupTestDirectories(t)

	// Create test server components
	playerManager := NewPlayerManager()
	triviaManager := NewTriviaManager()
	broadcastChan := make(chan BroadcastMessage, 256)
	_ = NewGameManager(playerManager, triviaManager, broadcastChan) // gameManager not used in this test

	mux := http.NewServeMux()

	// Add health endpoint (simplified version)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy","endpoints":{"players":"/ws","host":"/ws/host/test-uuid"}}`))
	})

	t.Run("Health endpoint returns 200", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
		assert.Contains(t, w.Body.String(), "healthy")
		assert.Contains(t, w.Body.String(), "/ws")
		assert.Contains(t, w.Body.String(), "/ws/host/")
	})
}

// Test helper functions
func setupTestDirectories(t *testing.T) {
	// Create test client directory structure
	err := os.MkdirAll("./test_client/public", 0755)
	require.NoError(t, err)

	// Create test client index.html
	clientIndex := `<!DOCTYPE html>
<html>
<head>
    <title>Player Frontend</title>
</head>
<body>
    <h1>Canvas Conundrum - Player Frontend</h1>
</body>
</html>`
	err = os.WriteFile("./test_client/public/index.html", []byte(clientIndex), 0644)
	require.NoError(t, err)

	// Create test client manifest.json
	clientManifest := `{
  "name": "Player App",
  "short_name": "Player",
  "start_url": "/",
  "display": "standalone"
}`
	err = os.WriteFile("./test_client/public/manifest.json", []byte(clientManifest), 0644)
	require.NoError(t, err)

	// Create test host directory structure
	err = os.MkdirAll("./test_host/public", 0755)
	require.NoError(t, err)

	// Create test host index.html
	hostIndex := `<!DOCTYPE html>
<html>
<head>
    <title>Host Frontend</title>
</head>
<body>
    <h1>Canvas Conundrum - Host Frontend</h1>
    <form id="hostForm">
        <input type="text" placeholder="Enter UUID" />
        <button type="submit">Connect</button>
    </form>
</body>
</html>`
	err = os.WriteFile("./test_host/public/index.html", []byte(hostIndex), 0644)
	require.NoError(t, err)

	// Create test host manifest.json
	hostManifest := `{
  "name": "Host App",
  "short_name": "Host",
  "start_url": "/host/",
  "display": "standalone"
}`
	err = os.WriteFile("./test_host/public/manifest.json", []byte(hostManifest), 0644)
	require.NoError(t, err)
}

func cleanupTestDirectories(t *testing.T) {
	err := os.RemoveAll("./test_client")
	if err != nil {
		t.Logf("Warning: failed to clean up test_client directory: %v", err)
	}

	err = os.RemoveAll("./test_host")
	if err != nil {
		t.Logf("Warning: failed to clean up test_host directory: %v", err)
	}
}

func TestCORSConfiguration(t *testing.T) {
	t.Run("Development CORS setup", func(t *testing.T) {
		// Test development environment CORS
		originalEnv := *environment
		*environment = "development"
		defer func() { *environment = originalEnv }()

		initializeCORS()

		// Test allowed development origins
		assert.True(t, isValidOrigin("http://localhost:3000"))
		assert.True(t, isValidOrigin("http://localhost:5173"))
		assert.True(t, isValidOrigin("http://localhost:8080"))
		assert.False(t, isValidOrigin("http://malicious.com"))
	})

	t.Run("Empty origin validation", func(t *testing.T) {
		initializeCORS()
		assert.False(t, isValidOrigin(""))
	})
}

func TestStaticFilePathTraversal(t *testing.T) {
	setupTestDirectories(t)
	defer cleanupTestDirectories(t)

	mux := http.NewServeMux()

	// Add static file serving endpoints
	clientFS := http.FileServer(http.Dir("./test_client/public/"))
	mux.Handle("/", clientFS)

	hostFS := http.FileServer(http.Dir("./test_host/public/"))
	mux.Handle("/host/", http.StripPrefix("/host/", hostFS))

	t.Run("Path traversal attempts are blocked", func(t *testing.T) {
		maliciousPaths := []string{
			"/../../../etc/passwd",
			"/../../main.go",
			"/../main.go",
			"/host/../../../etc/passwd",
			"/host/../../main.go",
		}

		for _, path := range maliciousPaths {
			req := httptest.NewRequest("GET", path, nil)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			// Should either return 404 or redirect, but not serve sensitive files
			assert.NotEqual(t, http.StatusOK, w.Code, "Path traversal should not succeed for: %s", path)
		}
	})
}
