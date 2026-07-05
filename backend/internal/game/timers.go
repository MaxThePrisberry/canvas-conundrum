package game

import (
	"time"
)

// timerService manages the engine's named one-shot timers. Fires are
// delivered as cmdTimer commands, never direct callbacks, so all state
// mutation stays on the engine goroutine. Generation counters make
// cancel/pause race-free against in-flight fires: a stale fire's generation
// no longer matches and is discarded by consume.
//
// Pausable timers implement the host-disconnect pause (game-design.md
// § Disconnections): PauseAll freezes remaining durations, ResumeAll re-arms
// them, extending every deadline by the pause duration.
type timerService struct {
	post   func(cmdTimer)
	timers map[string]*timerEntry
}

type timerEntry struct {
	gen       uint64
	deadline  time.Time
	pausable  bool
	paused    bool
	remaining time.Duration // meaningful while paused
	timer     *time.Timer
}

var timerGen uint64

func newTimerService(post func(cmdTimer)) *timerService {
	return &timerService{post: post, timers: map[string]*timerEntry{}}
}

// Schedule (re)arms the named timer to fire after d.
func (ts *timerService) Schedule(name string, d time.Duration, pausable bool) {
	ts.Cancel(name)
	timerGen++
	gen := timerGen
	entry := &timerEntry{
		gen:      gen,
		deadline: time.Now().Add(d),
		pausable: pausable,
	}
	entry.timer = time.AfterFunc(d, func() { ts.post(cmdTimer{name: name, gen: gen}) })
	ts.timers[name] = entry
}

// Cancel stops and forgets the named timer (no-op if absent).
func (ts *timerService) Cancel(name string) {
	if e, ok := ts.timers[name]; ok {
		e.timer.Stop()
		delete(ts.timers, name)
	}
}

// consume validates a fire notification. It returns true exactly once per
// scheduled fire; stale generations and paused/cancelled timers return false.
func (ts *timerService) consume(c cmdTimer) bool {
	e, ok := ts.timers[c.name]
	if !ok || e.gen != c.gen || e.paused {
		return false
	}
	delete(ts.timers, c.name)
	return true
}

// Remaining reports time left on the named timer (0 if absent).
func (ts *timerService) Remaining(name string) time.Duration {
	e, ok := ts.timers[name]
	if !ok {
		return 0
	}
	if e.paused {
		return e.remaining
	}
	return max(0, time.Until(e.deadline))
}

// PauseAll freezes every pausable, unpaused timer.
func (ts *timerService) PauseAll() {
	for _, e := range ts.timers {
		if !e.pausable || e.paused {
			continue
		}
		e.timer.Stop()
		e.remaining = max(0, time.Until(e.deadline))
		e.paused = true
	}
}

// ResumeAll re-arms every paused timer with its frozen remaining duration.
func (ts *timerService) ResumeAll() {
	for name, e := range ts.timers {
		if !e.paused {
			continue
		}
		timerGen++
		gen := timerGen
		e.gen = gen
		e.paused = false
		e.deadline = time.Now().Add(e.remaining)
		n := name
		e.timer = time.AfterFunc(e.remaining, func() { ts.post(cmdTimer{name: n, gen: gen}) })
	}
}
