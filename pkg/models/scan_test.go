package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestScanResult_NewScanResult(t *testing.T) {
	result := NewScanResult("active")
	if result.Metadata.Scope != "active" {
		t.Errorf("Expected scope 'active', got '%s'", result.Metadata.Scope)
	}
	if result.Metadata.TotalFilesScanned != 0 {
		t.Errorf("Expected 0 files scanned, got %d", result.Metadata.TotalFilesScanned)
	}
	if result.Metadata.IssuesFound != 0 {
		t.Errorf("Expected 0 issues, got %d", result.Metadata.IssuesFound)
	}
	if result.Issues == nil {
		t.Error("Issues slice should be initialized")
	}
}

func TestScanResult_AddIssue(t *testing.T) {
	result := NewScanResult("active")
	issue := FileIssue{
		FileID:     "test123",
		FileName:   "test.pdf",
		OwnerEmail: "user@example.com",
	}

	result.AddIssue(issue)

	if len(result.Issues) != 1 {
		t.Errorf("Expected 1 issue, got %d", len(result.Issues))
	}
	if result.Metadata.IssuesFound != 1 {
		t.Errorf("Expected 1 issue found, got %d", result.Metadata.IssuesFound)
	}
}

func TestScanResult_SetFilesScanned(t *testing.T) {
	result := NewScanResult("active")
	result.SetFilesScanned(100)

	if result.Metadata.TotalFilesScanned != 100 {
		t.Errorf("Expected 100 files scanned, got %d", result.Metadata.TotalFilesScanned)
	}
}

func TestScanResult_SetDuration(t *testing.T) {
	result := NewScanResult("active")
	duration := 5 * time.Second
	result.SetDuration(duration)

	if result.Metadata.DurationSeconds != 5.0 {
		t.Errorf("Expected 5.0 seconds, got %f", result.Metadata.DurationSeconds)
	}
}

func TestScanResult_JSONSerialization(t *testing.T) {
	result := NewScanResult("active")
	result.SetFilesScanned(10)
	issue := FileIssue{
		FileID:     "test123",
		FileName:   "test.pdf",
		OwnerEmail: "user@example.com",
		Permissions: []Permission{
			{
				ID:        "perm1",
				Type:      "user",
				Email:     "external@domain.com",
				Role:      "reader",
				RiskLevel: RiskMedium,
			},
		},
	}
	result.AddIssue(issue)

	jsonData, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	var decoded ScanResult
	if err := json.Unmarshal(jsonData, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if decoded.Metadata.Scope != "active" {
		t.Errorf("Expected scope 'active', got '%s'", decoded.Metadata.Scope)
	}
	if len(decoded.Issues) != 1 {
		t.Errorf("Expected 1 issue, got %d", len(decoded.Issues))
	}
}
