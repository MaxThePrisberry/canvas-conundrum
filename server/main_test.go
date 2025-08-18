package main

import (
	"canvas-conundrum/config"
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
