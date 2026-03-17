package google

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiter_WaitIfNeeded(t *testing.T) {
	rl := NewRateLimiter(3, 100*time.Millisecond, 1*time.Second)

	ctx := context.Background()
	start := time.Now()
	err := rl.WaitIfNeeded(ctx)
	duration := time.Since(start)

	if err != nil {
		t.Errorf("WaitIfNeeded() error = %v", err)
	}
	if duration > 50*time.Millisecond {
		t.Errorf("Expected no wait, but waited %v", duration)
	}
}

func TestRateLimiter_RetryWithBackoff(t *testing.T) {
	rl := NewRateLimiter(3, 10*time.Millisecond, 100*time.Millisecond)

	ctx := context.Background()
	attempts := 0
	err := rl.RetryWithBackoff(ctx, func() error {
		attempts++
		if attempts < 2 {
			return errors.New("rate limit error: 429")
		}
		return nil
	})

	if err != nil {
		t.Errorf("RetryWithBackoff() error = %v", err)
	}
	if attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts)
	}
}

func TestRateLimiter_MaxRetries(t *testing.T) {
	rl := NewRateLimiter(2, 10*time.Millisecond, 100*time.Millisecond)

	ctx := context.Background()
	attempts := 0
	err := rl.RetryWithBackoff(ctx, func() error {
		attempts++
		return errors.New("rate limit error: 429")
	})

	if err == nil {
		t.Error("Expected error after max retries")
	}
	if attempts != 3 { // initial attempt + 2 retries
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestIsRateLimitError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"429 error", errors.New("rate limit error: 429"), true},
		{"rate limit text", errors.New("rate limit exceeded"), true},
		{"quota text", errors.New("quota exceeded"), true},
		{"other error", errors.New("not found"), false},
		{"nil error", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRateLimitError(tt.err); got != tt.want {
				t.Errorf("isRateLimitError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHTTPClientWithRateLimit(t *testing.T) {
	// Create a test server that returns 429 on first request, 200 on second
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	rl := NewRateLimiter(3, 10*time.Millisecond, 100*time.Millisecond)
	client := NewHTTPClientWithRateLimit(http.DefaultClient, rl)

	req, _ := http.NewRequest("GET", server.URL, nil)
	resp, err := client.Do(req)

	if err != nil {
		t.Errorf("Do() error = %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	if attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts)
	}
}
