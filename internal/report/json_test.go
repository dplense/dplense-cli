package report

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/dplense/dplense-cli/pkg/models"
)

func TestWriteJSON(t *testing.T) {
	result := models.NewScanResult("active")
	result.SetFilesScanned(10)
	issue := models.FileIssue{
		FileID:     "test123",
		FileName:   "test.pdf",
		OwnerEmail: "user@example.com",
		Permissions: []models.Permission{
			{
				ID:        "perm1",
				Type:      "user",
				Email:     "external@domain.com",
				Role:      "reader",
				RiskLevel: models.RiskMedium,
			},
		},
	}
	result.AddIssue(issue)

	var buf bytes.Buffer
	if err := WriteJSON(result, &buf); err != nil {
		t.Fatalf("WriteJSON() error = %v", err)
	}

	// Verify it's valid JSON
	var decoded models.ScanResult
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if decoded.Metadata.Scope != "active" {
		t.Errorf("Expected scope 'active', got '%s'", decoded.Metadata.Scope)
	}
	if len(decoded.Issues) != 1 {
		t.Errorf("Expected 1 issue, got %d", len(decoded.Issues))
	}
}

func TestWriteJSONStrict(t *testing.T) {
	result := models.NewScanResult("active")
	var buf bytes.Buffer
	if err := WriteJSONStrict(result, &buf); err != nil {
		t.Fatalf("WriteJSONStrict() error = %v", err)
	}

	// Verify it's valid JSON
	var decoded models.ScanResult
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Verify it's compact (no indentation)
	output := buf.String()
	if len(output) < 50 {
		t.Error("Expected compact JSON output")
	}
}
