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

// Pause pauses the timer
func (t *Timer) Pause() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.isRunning || t.isPaused {
		return
	}

	t.isPaused = true
	t.pausedTime += time.Since(t.startTime)
}

// Resume resumes the timer
func (t *Timer) Resume() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.isRunning || !t.isPaused {
		return
	}

	t.isPaused = false
	t.startTime = time.Now()
}

// GetRemaining returns the remaining time
func (t *Timer) GetRemaining() time.Duration {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if !t.isRunning {
		return t.duration
	}

	elapsed := time.Since(t.startTime) - t.pausedTime
	if t.isPaused {
		elapsed = t.pausedTime
	}

	remaining := t.duration - elapsed
	if remaining < 0 {
		remaining = 0
	}

	return remaining
}

// GetElapsed returns the elapsed time
func (t *Timer) GetElapsed() time.Duration {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if !t.isRunning {
		return 0
	}

	elapsed := time.Since(t.startTime) - t.pausedTime
	if t.isPaused {
		elapsed = t.pausedTime
	}

	return elapsed
}

// IsRunning returns whether the timer is running
func (t *Timer) IsRunning() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.isRunning
}

// IsPaused returns whether the timer is paused
func (t *Timer) IsPaused() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.isPaused
}

// CountdownTimer is a simple countdown timer
type CountdownTimer struct {
	duration time.Duration
	onTick   func(remaining int)
	onExpire func()
	stop     chan bool
}

// NewCountdownTimer creates a new countdown timer
func NewCountdownTimer(seconds int, onTick func(int), onExpire func()) *CountdownTimer {
	return &CountdownTimer{
		duration: time.Duration(seconds) * time.Second,
		onTick:   onTick,
		onExpire: onExpire,
		stop:     make(chan bool),
	}
}

// Start starts the countdown
func (ct *CountdownTimer) Start() {
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		remaining := int(ct.duration.Seconds())

		for {
			select {
			case <-ticker.C:
				remaining--
				if ct.onTick != nil {
					ct.onTick(remaining)
				}
				if remaining <= 0 {
					if ct.onExpire != nil {
						ct.onExpire()
					}
					return
				}
			case <-ct.stop:
				return
			}
		}
	}()
}

// Stop stops the countdown
func (ct *CountdownTimer) Stop() {
	select {
	case ct.stop <- true:
	default:
	}
}
