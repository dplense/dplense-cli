package microsoft

import (
	"testing"

	"github.com/dplense/dplense-cli/pkg/models"
)

func TestMapRole(t *testing.T) {
	tests := []struct {
		name  string
		roles []string
		want  string
	}{
		{"empty", nil, "reader"},
		{"owner", []string{"owner"}, "owner"},
		{"write", []string{"write"}, "writer"},
		{"read", []string{"read"}, "reader"},
		{"Write uppercase", []string{"Write"}, "writer"},
		{"multiple roles picks first", []string{"write", "read"}, "writer"},
		{"unknown role", []string{"contributor"}, "contributor"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapRole(tt.roles)
			if got != tt.want {
				t.Errorf("mapRole(%v) = %q, want %q", tt.roles, got, tt.want)
			}
		})
	}
}

func TestDeref(t *testing.T) {
	s := "hello"
	if got := deref(&s); got != "hello" {
		t.Errorf("deref(&hello) = %q, want hello", got)
	}
	if got := deref(nil); got != "" {
		t.Errorf("deref(nil) = %q, want empty", got)
	}
}

func TestIsThrottled(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"429 error", errMsg("429 Too Many Requests"), true},
		{"throttled", errMsg("request throttled by server"), true},
		{"503", errMsg("503 Service Unavailable"), true},
		{"too many requests", errMsg("too many requests"), true},
		{"normal error", errMsg("not found"), false},
		{"auth error", errMsg("unauthorized"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isThrottled(tt.err)
			if got != tt.want {
				t.Errorf("isThrottled(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

type errMsg string

func (e errMsg) Error() string { return string(e) }

func TestRateLimiterQuota(t *testing.T) {
	rl := newRateLimiter()
	// Default quota should be 800
	if rl.quotaLimit != 800 {
		t.Errorf("default quotaLimit = %d, want 800", rl.quotaLimit)
	}
	if rl.maxRetries != 5 {
		t.Errorf("default maxRetries = %d, want 5", rl.maxRetries)
	}
}

func TestScanResultErrorAggregation(t *testing.T) {
	result := models.NewScanResult("test")

	if result.HasErrors() {
		t.Fatal("new result should not have errors")
	}

	result.AddError("site X: failed")
	result.AddError("drive Y: timeout")

	if !result.HasErrors() {
		t.Fatal("result should have errors after AddError")
	}
	if len(result.Metadata.Errors) != 2 {
		t.Errorf("expected 2 errors, got %d", len(result.Metadata.Errors))
	}
	if result.Metadata.Errors[0] != "site X: failed" {
		t.Errorf("unexpected first error: %s", result.Metadata.Errors[0])
	}
}
