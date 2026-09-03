package ratelimit

import (
	"sync"
	"time"
)

type TokenBucket struct {
	mu       sync.Mutex
	capacity float64
	rate     float64
	tokens   float64
	last     time.Time
	now      func() time.Time
}

func NewTokenBucket(capacity int, rate float64) *TokenBucket {
	now := time.Now()
	return &TokenBucket{capacity: float64(capacity), rate: rate, tokens: float64(capacity), last: now, now: time.Now}
}

func (b *TokenBucket) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := b.now()
	b.tokens += now.Sub(b.last).Seconds() * b.rate
	if b.tokens > b.capacity {
		b.tokens = b.capacity
	}
	b.last = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}
