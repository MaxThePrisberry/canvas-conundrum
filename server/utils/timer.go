package utils

import (
	"sync"
	"time"
)

// Timer represents a game timer with pause/resume capabilities
type Timer struct {
	mu         sync.RWMutex
	startTime  time.Time
	pausedTime time.Duration
	isPaused   bool
	isRunning  bool
	duration   time.Duration
	onExpire   func()
	ticker     *time.Ticker
	done       chan bool
}

// NewTimer creates a new timer
func NewTimer(duration time.Duration, onExpire func()) *Timer {
	return &Timer{
		duration: duration,
		onExpire: onExpire,
		done:     make(chan bool),
	}
}

// Start starts the timer
func (t *Timer) Start() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.isRunning {
		return
	}

	t.startTime = time.Now()
	t.isRunning = true
	t.isPaused = false

	// Start ticker in goroutine
	go t.tick()
}

// tick handles the timer tick
func (t *Timer) tick() {
	t.ticker = time.NewTicker(100 * time.Millisecond) // Check every 100ms
	defer t.ticker.Stop()

	for {
		select {
		case <-t.ticker.C:
			t.mu.RLock()
			if !t.isPaused && t.isRunning {
				elapsed := time.Since(t.startTime) - t.pausedTime
				if elapsed >= t.duration {
					t.mu.RUnlock()
					t.Stop()
					if t.onExpire != nil {
						t.onExpire()
					}
					return
				}
			}
			t.mu.RUnlock()
		case <-t.done:
			return
		}
	}
}

// Stop stops the timer
func (t *Timer) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.isRunning {
		return
	}

	t.isRunning = false
	t.isPaused = false

	// Signal done
	select {
	case t.done <- true:
	default:
	}
}
