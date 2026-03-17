package report

import (
	"bytes"
	"strings"
	"testing"

	"github.com/dplense/dplense-cli/pkg/models"
)

func TestWriteTable(t *testing.T) {
	result := models.NewScanResult("active")
	result.SetFilesScanned(10)
	issue := models.FileIssue{
		FileID:     "file1",
		FileName:   "test.pdf",
		DriveName:  "My Drive",
		OwnerEmail: "owner@example.com",
		OwnerName:  "Owner Name",
		FolderPath: "/Documents",
		WebViewLink: "https://drive.google.com/file1",
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
	if err := WriteTable(result, &buf); err != nil {
		t.Fatalf("WriteTable() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "test.pdf") {
		t.Error("Expected file name in output")
	}
	if !strings.Contains(output, "My Drive") {
		t.Error("Expected drive name in output")
	}
	if !strings.Contains(output, "external@domain.com") {
		t.Error("Expected shared with email in output")
	}
	if !strings.Contains(output, "medium") {
		t.Error("Expected risk level in output")
	}
}

func TestWriteTable_Empty(t *testing.T) {
	result := models.NewScanResult("active")
	var buf bytes.Buffer
	if err := WriteTable(result, &buf); err != nil {
		t.Fatalf("WriteTable() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No security issues found") {
		t.Error("Expected 'No security issues found' message")
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"this is a very long string", 10, "this is..."}, // 10 - 3 = 7 chars + "..."
		{"test", 3, "..."},
		{"test", 4, "test"},
		{"", 10, ""},
		{"long string", 500, "long string"}, // Should not truncate if maxLen >= 500
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := truncateString(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
			}
		})
	}
}

func TestGetSharedWith(t *testing.T) {
	tests := []struct {
		name       string
		permission models.Permission
		expected   string
	}{
		{
			name: "email permission",
			permission: models.Permission{
				Email: "user@example.com",
				Type:  "user",
			},
			expected: "user@example.com",
		},
		{
			name: "domain permission",
			permission: models.Permission{
				Domain: "example.com",
				Type:   "domain",
			},
			expected: "@example.com",
		},
		{
			name: "anyone permission",
			permission: models.Permission{
				Type: "anyone",
			},
			expected: "Anyone with link",
		},
		{
			name: "empty permission",
			permission: models.Permission{
				Type: "user",
			},
			expected: "-",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getSharedWith(tt.permission)
			if result != tt.expected {
				t.Errorf("getSharedWith() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestWriteTable_WithInternalPermissions(t *testing.T) {
	result := models.NewScanResult("active")
	result.SetFilesScanned(10)
	issue := models.FileIssue{
		FileID:     "file1",
		FileName:   "test.pdf",
		DriveName:  "My Drive",
		OwnerEmail: "owner@example.com",
		OwnerName:  "Owner Name",
		FolderPath: "/Documents",
		WebViewLink: "https://drive.google.com/file1",
		Permissions: []models.Permission{
			{
				ID:         "perm1",
				Type:       "user",
				Email:      "internal@example.com",
				Role:       "reader",
				RiskLevel:  models.RiskLow,
				IsInternal: true,
			},
			{
				ID:         "perm2",
				Type:       "user",
				Email:      "external@domain.com",
				Role:       "reader",
				RiskLevel:  models.RiskMedium,
				IsInternal: false,
			},
		},
	}
	result.AddIssue(issue)

	var buf bytes.Buffer
	if err := WriteTable(result, &buf); err != nil {
		t.Fatalf("WriteTable() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "internal@example.com") {
		t.Error("Expected internal user in output")
	}
	if !strings.Contains(output, "external@domain.com") {
		t.Error("Expected external user in output")
	}
	if !strings.Contains(output, "Internal Users:") {
		t.Error("Expected 'Internal Users:' section in output")
	}
}
