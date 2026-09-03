package ratelimit

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestTokenBucketRefill(t *testing.T) {
	now := time.Unix(100, 0)
	bucket := NewTokenBucket(2, 1)
	bucket.last = now
	bucket.now = func() time.Time { return now }
	if !bucket.Allow() || !bucket.Allow() || bucket.Allow() {
		t.Fatal("token bucket did not enforce initial capacity")
	}
	now = now.Add(time.Second)
	if !bucket.Allow() || bucket.Allow() {
		t.Fatal("token bucket did not refill at configured rate")
	}
}

func TestLeakyBucketDrain(t *testing.T) {
	now := time.Unix(100, 0)
	bucket := NewLeakyBucket(2, 1)
	bucket.last = now
	bucket.now = func() time.Time { return now }
	if !bucket.Allow() || !bucket.Allow() || bucket.Allow() {
		t.Fatal("leaky bucket did not enforce capacity")
	}
	now = now.Add(time.Second)
	if !bucket.Allow() || bucket.Allow() {
		t.Fatal("leaky bucket did not drain at configured rate")
	}
}

func TestUserWindowPerUserAndReset(t *testing.T) {
	now := time.Unix(100, 0)
	window := NewUserWindow(2, time.Minute)
	window.now = func() time.Time { return now }
	if !window.Allow(1) || !window.Allow(1) || window.Allow(1) {
		t.Fatal("user window did not enforce user limit")
	}
	if !window.Allow(2) {
		t.Fatal("one user exhausted another user's quota")
	}
	now = now.Add(time.Minute)
	if !window.Allow(1) {
		t.Fatal("user quota did not reset after the window")
	}
}

func TestTokenBucketConcurrentCapacity(t *testing.T) {
	bucket := NewTokenBucket(50, 0.000001)
	var allowed atomic.Int64
	var group sync.WaitGroup
	for range 500 {
		group.Add(1)
		go func() {
			defer group.Done()
			if bucket.Allow() {
				allowed.Add(1)
			}
		}()
	}
	group.Wait()
	if got := allowed.Load(); got != 50 {
		t.Fatalf("allowed = %d, want 50", got)
	}
}
