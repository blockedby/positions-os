package telegram

import (
	"context"
	"testing"
	"time"
)

func TestRateLimiter_Wait(t *testing.T) {
	// Test basic wait - should not block for first request
	rl := NewRateLimiter(10.0, 1) // 10 requests per second

	ctx := context.Background()
	start := time.Now()
	err := rl.Wait(ctx)
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// First request should be immediate (within burst)
	if elapsed > 50*time.Millisecond {
		t.Errorf("expected immediate response, got %v", elapsed)
	}
}

func TestRateLimiter_Wait_ContextCanceled(t *testing.T) {
	// Test that Wait respects context cancellation
	rl := NewRateLimiter(0.1, 1) // Very slow: 1 request per 10 seconds

	// Use up the burst
	_ = rl.Wait(context.Background())

	// Now the next request should block, but we cancel context
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := rl.Wait(ctx)

	// Should return an error (either DeadlineExceeded or the limiter's own context error)
	if err == nil {
		t.Error("expected error due to context timeout, got nil")
	}
}

func TestRateLimiter_SetFloodWait(t *testing.T) {
	rl := NewRateLimiter(10.0, 1)

	// Set a flood wait for 100ms
	rl.SetFloodWait(1) // 1 second

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := rl.Wait(ctx)
	elapsed := time.Since(start)

	// Should timeout because flood wait is 1 second but context is 200ms
	if err != context.DeadlineExceeded {
		t.Errorf("expected DeadlineExceeded due to flood wait, got %v", err)
	}

	// Should have waited close to the context timeout
	if elapsed < 150*time.Millisecond || elapsed > 250*time.Millisecond {
		t.Errorf("expected ~200ms wait (context timeout), got %v", elapsed)
	}
}

func TestRateLimiter_RateLimiting(t *testing.T) {
	// Test that rate limiting actually throttles requests
	rl := NewRateLimiter(10.0, 1) // 10 requests per second = 100ms between requests

	ctx := context.Background()

	// Make 3 requests
	start := time.Now()
	for i := 0; i < 3; i++ {
		err := rl.Wait(ctx)
		if err != nil {
			t.Errorf("request %d: unexpected error: %v", i, err)
		}
	}
	elapsed := time.Since(start)

	// Should take at least 200ms for 3 requests at 10 rps (100ms between each)
	// First one is immediate (burst), then 100ms wait, then 100ms wait
	if elapsed < 150*time.Millisecond {
		t.Errorf("expected at least 150ms for 3 requests at 10 rps, got %v", elapsed)
	}
}

func TestDefaultRateLimiter(t *testing.T) {
	rl := DefaultRateLimiter()

	if rl == nil {
		t.Error("DefaultRateLimiter returned nil")
	}

	// Verify it allows at least one immediate request
	ctx := context.Background()
	err := rl.Wait(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRateLimiter_FloodWaitExpires(t *testing.T) {
	rl := NewRateLimiter(10.0, 1)

	// Set a very short flood wait (effectively 0 by the time we check)
	rl.floodWaitUntil = time.Now().Add(-100 * time.Millisecond) // Already expired

	ctx := context.Background()
	start := time.Now()
	err := rl.Wait(ctx)
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Should be immediate since flood wait is expired
	if elapsed > 50*time.Millisecond {
		t.Errorf("expected immediate response (flood wait expired), got %v", elapsed)
	}
}
