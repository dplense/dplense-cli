package report

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"

	"gdrive-audit/pkg/models"
)

func TestWriteCSV(t *testing.T) {
	result := models.NewScanResult("active")
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
				Domain:    "",
				Role:      "reader",
				RiskLevel: models.RiskMedium,
			},
		},
	}
	result.AddIssue(issue)

	var buf bytes.Buffer
	if err := WriteCSV(result, &buf); err != nil {
		t.Fatalf("WriteCSV() error = %v", err)
	}

	// Verify CSV format
	reader := csv.NewReader(strings.NewReader(buf.String()))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read CSV: %v", err)
	}

	if len(records) < 2 {
		t.Fatalf("Expected at least header + 1 data row, got %d", len(records))
	}

	// Check header
	header := records[0]
	expectedHeaders := []string{"File ID", "File Name", "Drive Name", "Owner Email"}
	for _, expected := range expectedHeaders {
		found := false
		for _, h := range header {
			if h == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected header '%s' not found", expected)
		}
	}

	// Check data row
	dataRow := records[1]
	if dataRow[0] != "file1" {
		t.Errorf("Expected file ID 'file1', got '%s'", dataRow[0])
	}
}
