package report

import (
	"os"
	"strings"
	"testing"

	"github.com/dplense/dplense-cli/pkg/models"
)

func TestWriteExcel_Flat(t *testing.T) {
	result := models.NewScanResult("active")
	result.SetFilesScanned(10)
	issue := models.FileIssue{
		FileID:     "file1",
		FileName:   "test.pdf",
		DriveName:  "My Drive",
		OwnerEmail: "owner@example.com",
		OwnerName:  "Owner",
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

	tmpFile := "/tmp/test_excel_flat.xlsx"
	defer os.Remove(tmpFile)

	if err := WriteExcel(result, tmpFile, "flat"); err != nil {
		t.Fatalf("WriteExcel() error = %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Error("Excel file was not created")
	}
}

func TestWriteExcel_ByOwner(t *testing.T) {
	result := models.NewScanResult("active")
	result.SetFilesScanned(10)

	// Add issues from different owners
	issue1 := models.FileIssue{
		FileID:     "file1",
		FileName:   "test1.pdf",
		OwnerEmail: "owner1@example.com",
		OwnerName:  "Owner 1",
		Permissions: []models.Permission{
			{ID: "perm1", Type: "user", Email: "external@domain.com", Role: "reader", RiskLevel: models.RiskMedium},
		},
	}
	issue2 := models.FileIssue{
		FileID:     "file2",
		FileName:   "test2.pdf",
		OwnerEmail: "owner2@example.com",
		OwnerName:  "Owner 2",
		Permissions: []models.Permission{
			{ID: "perm2", Type: "user", Email: "external@domain.com", Role: "reader", RiskLevel: models.RiskHigh},
		},
	}
	result.AddIssue(issue1)
	result.AddIssue(issue2)

	tmpFile := "/tmp/test_excel_byowner.xlsx"
	defer os.Remove(tmpFile)

	if err := WriteExcel(result, tmpFile, "by-owner"); err != nil {
		t.Fatalf("WriteExcel() error = %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Error("Excel file was not created")
	}
}

func TestSanitizeSheetName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal_name", "normal_name"},
		{"name/with/slashes", "name_with_slashes"},
		{"name*with*stars", "name_with_stars"},
		{"name[with]brackets", "name_with_brackets"},
		{"very_long_name_that_exceeds_excel_limit_of_31_characters", "very_long_name_that_exceeds_ex"},
		{"", ""},
		{"a", "a"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeSheetName(tt.input)
			if len(result) > 31 {
				t.Errorf("sanitizeSheetName(%q) = %q (length %d), want length <= 31", tt.input, result, len(result))
			}
			// Check that invalid characters are removed
			invalidChars := []string{"\\", "/", "?", "*", "[", "]"}
			for _, char := range invalidChars {
				if len(tt.input) <= 31 && strings.Contains(result, char) {
					t.Errorf("sanitizeSheetName(%q) contains invalid character %q", tt.input, char)
				}
			}
		})
	}
}

func TestWriteExcel_InvalidStrategy(t *testing.T) {
	result := models.NewScanResult("active")
	tmpFile := "/tmp/test_excel_invalid.xlsx"
	defer os.Remove(tmpFile)

	err := WriteExcel(result, tmpFile, "invalid-strategy")
	if err == nil {
		t.Error("Expected error for invalid strategy")
	}
	if !strings.Contains(err.Error(), "invalid strategy") {
		t.Errorf("Expected error message about invalid strategy, got: %v", err)
	}
}
