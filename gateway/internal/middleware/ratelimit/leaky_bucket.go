package ratelimit

import (
	"sync"
	"time"
)

// LeakyBucket models a finite queue whose water drains continuously. Requests
// are rejected instead of blocked when the queue has reached capacity.
type LeakyBucket struct {
	mu       sync.Mutex
	capacity float64
	rate     float64
	water    float64
	last     time.Time
	now      func() time.Time
}

func NewLeakyBucket(capacity int, rate float64) *LeakyBucket {
	now := time.Now()
	return &LeakyBucket{capacity: float64(capacity), rate: rate, last: now, now: time.Now}
}

func (b *LeakyBucket) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := b.now()
	b.water -= now.Sub(b.last).Seconds() * b.rate
	if b.water < 0 {
		b.water = 0
	}
	b.last = now
	if b.water+1 > b.capacity {
		return false
	}
	b.water++
	return true
}
