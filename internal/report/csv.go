package report

import (
	"encoding/csv"
	"fmt"
	"io"

	"gdrive-audit/pkg/models"
)

// WriteCSV writes scan results as CSV to the writer
func WriteCSV(result *models.ScanResult, writer io.Writer) error {
	w := csv.NewWriter(writer)
	defer w.Flush()

	// Write header
	header := []string{
		"File ID",
		"File Name",
		"Drive Name",
		"Owner Email",
		"Owner Name",
		"Folder Path",
		"Permission ID",
		"Permission Type",
		"Shared With",
		"Domain",
		"Role",
		"Risk Level",
		"Link",
	}
	if err := w.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write one row per file-permission combination
	for _, issue := range result.Issues {
		for _, perm := range issue.Permissions {
			row := []string{
				issue.FileID,
				issue.FileName,
				issue.DriveName,
				issue.OwnerEmail,
				issue.OwnerName,
				issue.FolderPath,
				perm.ID,
				perm.Type,
				perm.Email,
				perm.Domain,
				perm.Role,
				string(perm.RiskLevel),
				issue.WebViewLink,
			}
			if err := w.Write(row); err != nil {
				return fmt.Errorf("failed to write CSV row: %w", err)
			}
		}
	}

	return nil
}
