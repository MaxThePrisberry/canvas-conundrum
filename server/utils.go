package main

import (
	"encoding/json"
	"log"
	"time"
)

// sendToPlayer sends a message to a specific player
func sendToPlayer(player *Player, msgType string, payload interface{}) error {
	player.mu.RLock()
	conn := player.Connection
	player.mu.RUnlock()

	if conn == nil {
		return nil // Player disconnected, silently ignore
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	msg := BaseMessage{
		Type:    msgType,
		Payload: payloadBytes,
	}

	// Synchronize WebSocket writes to prevent concurrent access
	player.writeMu.Lock()
	defer player.writeMu.Unlock()

	// Set write deadline to prevent hanging on slow connections
	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

	return conn.WriteJSON(msg)
}

// mustMarshal marshals data or panics
func mustMarshal(v interface{}) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		log.Panicf("Failed to marshal data: %v", err)
	}
	return data
}

// logError logs an error with context
func logError(context string, err error) {
	if err != nil {
		log.Printf("ERROR [%s]: %v", context, err)
	}
}

// logInfo logs informational messages
func logInfo(format string, args ...interface{}) {
	log.Printf("INFO: "+format, args...)
}

// logDebug logs debug messages (could be toggled with a debug flag)
func logDebug(format string, args ...interface{}) {
	// In production, this could be controlled by a debug flag
	log.Printf("DEBUG: "+format, args...)
}
