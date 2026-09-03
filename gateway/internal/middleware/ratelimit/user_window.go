package ratelimit

import (
	"sync"
	"time"
)

type windowEntry struct {
	startedAt time.Time
	count     int
}

type UserWindow struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	entries map[uint64]windowEntry
	now     func() time.Time
	checks  uint64
}

func NewUserWindow(limit int, window time.Duration) *UserWindow {
	return &UserWindow{limit: limit, window: window, entries: make(map[uint64]windowEntry), now: time.Now}
}

func (w *UserWindow) Allow(userID uint64) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	now := w.now()
	entry, exists := w.entries[userID]
	if !exists || now.Sub(entry.startedAt) >= w.window {
		w.entries[userID] = windowEntry{startedAt: now, count: 1}
		w.cleanup(now)
		return true
	}
	if entry.count >= w.limit {
		w.cleanup(now)
		return false
	}
	entry.count++
	w.entries[userID] = entry
	w.cleanup(now)
	return true
}

// Periodic lazy cleanup bounds memory without a background goroutine.
func (w *UserWindow) cleanup(now time.Time) {
	w.checks++
	if w.checks%1024 != 0 {
		return
	}
	for userID, entry := range w.entries {
		if now.Sub(entry.startedAt) >= 2*w.window {
			delete(w.entries, userID)
		}
	}
}
