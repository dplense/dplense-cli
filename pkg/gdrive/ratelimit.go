package gdrive

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"
)

// RateLimiter handles rate limiting and exponential backoff for API calls
type RateLimiter struct {
	maxRetries      int
	baseDelay       time.Duration
	maxDelay        time.Duration
	quotaLimit      int           // requests per quotaWindow
	quotaWindow     time.Duration // time window for quota
	requestTimes    []time.Time
	mu              sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(maxRetries int, baseDelay, maxDelay time.Duration) *RateLimiter {
	return &RateLimiter{
		maxRetries:   maxRetries,
		baseDelay:    baseDelay,
		maxDelay:     maxDelay,
		quotaLimit:   1000,              // 1000 requests per 100 seconds
		quotaWindow: 100 * time.Second,
		requestTimes: make([]time.Time, 0),
	}
}

// WaitIfNeeded checks if we need to wait before making a request
func (rl *RateLimiter) WaitIfNeeded(ctx context.Context) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	// Remove old request times outside the quota window
	cutoff := now.Add(-rl.quotaWindow)
	validTimes := make([]time.Time, 0)
	for _, t := range rl.requestTimes {
		if t.After(cutoff) {
			validTimes = append(validTimes, t)
		}
	}
	rl.requestTimes = validTimes

	// If we're at quota limit, wait until oldest request expires
	if len(rl.requestTimes) >= rl.quotaLimit {
		oldestTime := rl.requestTimes[0]
		waitTime := rl.quotaWindow - now.Sub(oldestTime) + time.Second
		if waitTime > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(waitTime):
				// Continue after waiting
			}
		}
	}

	// Record this request
	rl.requestTimes = append(rl.requestTimes, now)
	return nil
}

// RetryWithBackoff executes a function with exponential backoff retry logic
func (rl *RateLimiter) RetryWithBackoff(ctx context.Context, fn func() error) error {
	var lastErr error
	delay := rl.baseDelay

	for attempt := 0; attempt <= rl.maxRetries; attempt++ {
		// Check if we need to wait for quota
		if err := rl.WaitIfNeeded(ctx); err != nil {
			return fmt.Errorf("rate limiter wait failed: %w", err)
		}

		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if it's a rate limit error (429)
		if isRateLimitError(err) {
			// Exponential backoff
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
				delay = time.Duration(math.Min(float64(delay*2), float64(rl.maxDelay)))
			}
			continue
		}

		// For non-rate-limit errors, return immediately
		return err
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}

// isRateLimitError checks if an error is a rate limit error
func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}

	// Check for HTTP 429 status code in error message
	errStr := err.Error()
	return contains(errStr, "429") || contains(errStr, "rate limit") || contains(errStr, "quota")
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || 
		(len(s) > len(substr) && 
			(s[:len(substr)] == substr || 
			 s[len(s)-len(substr):] == substr ||
			 containsMiddle(s, substr))))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// HTTPClientWithRateLimit wraps an HTTP client with rate limiting
type HTTPClientWithRateLimit struct {
	client     *http.Client
	rateLimiter *RateLimiter
}

// NewHTTPClientWithRateLimit creates a new HTTP client with rate limiting
func NewHTTPClientWithRateLimit(client *http.Client, rateLimiter *RateLimiter) *HTTPClientWithRateLimit {
	return &HTTPClientWithRateLimit{
		client:      client,
		rateLimiter: rateLimiter,
	}
}

// Do executes an HTTP request with rate limiting
func (c *HTTPClientWithRateLimit) Do(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error

	err = c.rateLimiter.RetryWithBackoff(req.Context(), func() error {
		resp, err = c.client.Do(req)
		if err != nil {
			return err
		}
		if resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			return fmt.Errorf("rate limit error: %d", resp.StatusCode)
		}
		return nil
	})

	return resp, err
}
