package utils

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewTimer(t *testing.T) {
	onExpire := func() {
		// Test callback
	}

	timer := NewTimer(100*time.Millisecond, onExpire)

	assert.NotNil(t, timer)
	assert.Equal(t, 100*time.Millisecond, timer.duration)
	assert.NotNil(t, timer.onExpire)
	assert.NotNil(t, timer.done)
	assert.False(t, timer.isRunning)
	assert.False(t, timer.isPaused)
}

func TestTimerStart(t *testing.T) {
	var called bool
	var mu sync.Mutex

	onExpire := func() {
		mu.Lock()
		called = true
		mu.Unlock()
	}

	timer := NewTimer(50*time.Millisecond, onExpire)

	// Test starting timer
	timer.Start()
	assert.True(t, timer.isRunning)
	assert.False(t, timer.isPaused)

	// Test that starting again does nothing
	oldStartTime := timer.startTime
	timer.Start()
	assert.Equal(t, oldStartTime, timer.startTime)

	// Wait for timer to expire - allow more time for CI environments
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	wasCalled := called
	mu.Unlock()

	assert.True(t, wasCalled)

	timer.Stop()
}

func TestTimerStartWithoutExpireFunction(t *testing.T) {
	timer := NewTimer(50*time.Millisecond, nil)

	timer.Start()
	assert.True(t, timer.isRunning)

	// Wait for timer to expire (should not panic)
	time.Sleep(100 * time.Millisecond)

	timer.Stop()
}

func TestTimerStop(t *testing.T) {
	called := false
	onExpire := func() {
		called = true
	}

	timer := NewTimer(200*time.Millisecond, onExpire)
	timer.Start()

	// Stop before expiration
	time.Sleep(50 * time.Millisecond)
	timer.Stop()

	assert.False(t, timer.isRunning)
	assert.False(t, timer.isPaused)
	assert.False(t, called)

	// Test stopping when not running
	timer.Stop()
	assert.False(t, timer.isRunning)
}

func TestTimerConcurrentAccess(t *testing.T) {
	var wg sync.WaitGroup
	timer := NewTimer(100*time.Millisecond, func() {})

	// Start multiple goroutines trying to start/stop timer
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			timer.Start()
			time.Sleep(10 * time.Millisecond)
			timer.Stop()
		}()
	}

	wg.Wait()
	assert.False(t, timer.isRunning)
}

func TestTimerTick(t *testing.T) {
	expired := false
	timer := NewTimer(50*time.Millisecond, func() {
		expired = true
	})

	timer.Start()

	// Wait longer than duration to ensure expiration
	time.Sleep(100 * time.Millisecond)

	assert.True(t, expired)
	assert.False(t, timer.isRunning)
}

func TestTimerTickWithEarlyStop(t *testing.T) {
	expired := false
	timer := NewTimer(200*time.Millisecond, func() {
		expired = true
	})

	timer.Start()

	// Stop before expiration
	time.Sleep(50 * time.Millisecond)
	timer.Stop()

	// Wait to ensure no expiration
	time.Sleep(200 * time.Millisecond)

	assert.False(t, expired)
	assert.False(t, timer.isRunning)
}
