package utils

import (
	"sync"
	"sync/atomic"
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
	assert.False(t, timer.IsRunning())
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
	assert.True(t, timer.IsRunning())
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
	assert.True(t, timer.IsRunning())

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

	assert.False(t, timer.IsRunning())
	assert.False(t, timer.isPaused)
	assert.False(t, called)

	// Test stopping when not running
	timer.Stop()
	assert.False(t, timer.IsRunning())
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
	assert.False(t, timer.IsRunning())
}

func TestTimerTick(t *testing.T) {
	var expired int32
	timer := NewTimer(100*time.Millisecond, func() {
		atomic.StoreInt32(&expired, 1)
	})

	timer.Start()

	// Wait longer than duration to ensure expiration
	time.Sleep(200 * time.Millisecond)

	assert.Equal(t, int32(1), atomic.LoadInt32(&expired))
	assert.False(t, timer.IsRunning())
}

func TestTimerTickWithEarlyStop(t *testing.T) {
	var expired int32
	timer := NewTimer(200*time.Millisecond, func() {
		atomic.StoreInt32(&expired, 1)
	})

	timer.Start()

	// Stop before expiration
	time.Sleep(50 * time.Millisecond)
	timer.Stop()

	// Wait to ensure no expiration
	time.Sleep(200 * time.Millisecond)

	assert.Equal(t, int32(0), atomic.LoadInt32(&expired))
	assert.False(t, timer.IsRunning())
}
