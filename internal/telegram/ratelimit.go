package telegram

import (
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiter controls the frequency of requests to Telegram API.
type RateLimiter struct {
	// main limiter: default 20/sec
	limiter *rate.Limiter

	// additional backoff after FLOOD_WAIT
	floodWaitUntil time.Time
	mu             sync.Mutex
}

// NewRateLimiter creates a rate limiter for Telegram.
// rps - requests per second (recommended 1-2 for safe browsing, 15-20 for scraping)
// burst - allowed burst
func NewRateLimiter(rps float64, burst int) *RateLimiter {
	return &RateLimiter{
		limiter: rate.NewLimiter(rate.Limit(rps), burst),
	}
}

// DefaultRateLimiter returns a limiter with conservative settings.
// Updated to 2.0 rps based on user feedback to prevent connection loss.
func DefaultRateLimiter() *RateLimiter {
	return NewRateLimiter(2.0, 1)
}

// Wait blocks until the next request is allowed.
func (r *RateLimiter) Wait(ctx context.Context) error {
	r.mu.Lock()
	waitUntil := r.floodWaitUntil
	r.mu.Unlock()

	// if flood wait is active - wait for it
	if time.Now().Before(waitUntil) {
		select {
		case <-time.After(time.Until(waitUntil)):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return r.limiter.Wait(ctx)
}

// SetFloodWait sets a pause after a FLOOD_WAIT error.
func (r *RateLimiter) SetFloodWait(seconds int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.floodWaitUntil = time.Now().Add(time.Duration(seconds) * time.Second)
}
