package report

import (
	"fmt"
	"sort"
	"strings"

	"github.com/xuri/excelize/v2"
	"github.com/dplense/dplense-cli/pkg/models"
)

// WriteExcel writes scan results to an Excel file with multiple sheets
func WriteExcel(result *models.ScanResult, filePath string, strategy string) error {
	f := excelize.NewFile()
	defer f.Close()

	// Set default sheet name
	sheetName := "Issues"
	f.SetSheetName("Sheet1", sheetName)
	_ = sheetName // Used in writeExcelFlat

	// Write based on strategy
	switch strings.ToLower(strategy) {
	case "by-owner":
		return writeExcelByOwner(f, result, filePath)
	case "flat", "":
		return writeExcelFlat(f, result, filePath)
	default:
		return fmt.Errorf("invalid strategy: %s (must be 'flat' or 'by-owner')", strategy)
	}
}

// writeExcelFlat writes all issues to a single sheet
func writeExcelFlat(f *excelize.File, result *models.ScanResult, filePath string) error {
	sheetName := "Issues"

	// Write header
	headers := []string{"File ID", "File Name", "Drive Name", "Owner Email", "Owner Name", "Folder Path", "Permission ID", "Permission Type", "Shared With", "Domain", "Role", "Risk Level", "Link"}
	for i, header := range headers {
		cell := fmt.Sprintf("%c1", 'A'+i)
		f.SetCellValue(sheetName, cell, header)
	}

	// Style header row
	headerStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#E0E0E0"}, Pattern: 1},
	})
	if err == nil {
		f.SetCellStyle(sheetName, "A1", fmt.Sprintf("%c1", 'A'+len(headers)-1), headerStyle)
	}

	// Write data rows (one per file-permission combination)
	row := 2
	for _, issue := range result.Issues {
		for _, perm := range issue.Permissions {
			sharedWith := getSharedWith(perm)
			riskLevel := string(perm.RiskLevel)

			values := []interface{}{
				issue.FileID,
				issue.FileName,
				issue.DriveName,
				issue.OwnerEmail,
				issue.OwnerName,
				issue.FolderPath,
				perm.ID,
				perm.Type,
				sharedWith,
				perm.Domain,
				perm.Role,
				riskLevel,
				issue.WebViewLink,
			}

			for i, value := range values {
				cell := fmt.Sprintf("%c%d", 'A'+i, row)
				f.SetCellValue(sheetName, cell, value)
			}

			// Color code risk level
			riskStyle := getRiskStyle(f, perm.RiskLevel)
			if riskStyle > 0 {
				riskCell := fmt.Sprintf("%c%d", 'A'+11, row) // Risk Level column (L)
				f.SetCellStyle(sheetName, riskCell, riskCell, riskStyle)
			}

			row++
		}
	}

	// Auto-size columns
	for i := 0; i < len(headers); i++ {
		col := string(rune('A' + i))
		f.SetColWidth(sheetName, col, col, 15)
	}

	// Set summary sheet
	if err := addSummarySheet(f, result); err != nil {
		return fmt.Errorf("failed to add summary sheet: %w", err)
	}

	return f.SaveAs(filePath)
}

// writeExcelByOwner writes issues grouped by owner into separate sheets
func writeExcelByOwner(f *excelize.File, result *models.ScanResult, filePath string) error {
	// Group issues by owner
	ownerGroups := make(map[string][]models.FileIssue)
	for _, issue := range result.Issues {
		ownerEmail := issue.OwnerEmail
		if ownerEmail == "" {
			ownerEmail = "Unknown"
		}
		ownerGroups[ownerEmail] = append(ownerGroups[ownerEmail], issue)
	}

	// Create a sheet for each owner
	for ownerEmail, issues := range ownerGroups {
		// Sanitize sheet name (Excel has restrictions)
		sheetName := sanitizeSheetName(ownerEmail)
		if len(sheetName) > 31 {
			sheetName = sheetName[:31]
		}

		// Create new sheet
		_, err := f.NewSheet(sheetName)
		if err != nil {
			return fmt.Errorf("failed to create sheet for %s: %w", ownerEmail, err)
		}

		// Write header
		headers := []string{"File ID", "File Name", "Drive Name", "Owner Email", "Owner Name", "Folder Path", "Permission ID", "Permission Type", "Shared With", "Domain", "Role", "Risk Level", "Link"}
		for i, header := range headers {
			cell := fmt.Sprintf("%c1", 'A'+i)
			f.SetCellValue(sheetName, cell, header)
		}

		// Style header row
		headerStyle, err := f.NewStyle(&excelize.Style{
			Font: &excelize.Font{Bold: true},
			Fill: excelize.Fill{Type: "pattern", Color: []string{"#E0E0E0"}, Pattern: 1},
		})
		if err == nil {
			f.SetCellStyle(sheetName, "A1", fmt.Sprintf("%c1", 'A'+len(headers)-1), headerStyle)
		}

		// Write data rows
		row := 2
		for _, issue := range issues {
			for _, perm := range issue.Permissions {
				sharedWith := getSharedWith(perm)
				riskLevel := string(perm.RiskLevel)

				values := []interface{}{
					issue.FileID,
					issue.FileName,
					issue.DriveName,
					issue.OwnerEmail,
					issue.OwnerName,
					issue.FolderPath,
					perm.ID,
					perm.Type,
					sharedWith,
					perm.Domain,
					perm.Role,
					riskLevel,
					issue.WebViewLink,
				}

				for i, value := range values {
					cell := fmt.Sprintf("%c%d", 'A'+i, row)
					f.SetCellValue(sheetName, cell, value)
				}

				// Color code risk level
				riskStyle := getRiskStyle(f, perm.RiskLevel)
				if riskStyle > 0 {
					riskCell := fmt.Sprintf("%c%d", 'A'+11, row) // Risk Level column (L)
					f.SetCellStyle(sheetName, riskCell, riskCell, riskStyle)
				}

				row++
			}
		}

		// Auto-size columns
		for i := 0; i < len(headers); i++ {
			col := string(rune('A' + i))
			f.SetColWidth(sheetName, col, col, 15)
		}
	}

	// Set summary sheet as active
	if err := addSummarySheet(f, result); err != nil {
		return fmt.Errorf("failed to add summary sheet: %w", err)
	}

	// Set first owner sheet as active (after summary)
	if len(ownerGroups) > 0 {
		owners := make([]string, 0, len(ownerGroups))
		for owner := range ownerGroups {
			owners = append(owners, owner)
		}
		sort.Strings(owners)
		if len(owners) > 0 {
			firstSheet := sanitizeSheetName(owners[0])
			if len(firstSheet) > 31 {
				firstSheet = firstSheet[:31]
			}
			sheetIndex, err := f.GetSheetIndex(firstSheet)
			if err == nil && sheetIndex >= 0 {
				f.SetActiveSheet(sheetIndex)
			}
		}
	}

	return f.SaveAs(filePath)
}

// addSummarySheet adds a summary sheet with scan metadata
func addSummarySheet(f *excelize.File, result *models.ScanResult) error {
	sheetName := "Summary"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return err
	}

	// Set as first sheet
	f.SetActiveSheet(index)
	f.DeleteSheet("Sheet1") // Remove default sheet if it exists

	// Write summary information
	f.SetCellValue(sheetName, "A1", "Scan Summary")
	f.SetCellValue(sheetName, "A2", "Scope:")
	f.SetCellValue(sheetName, "B2", result.Metadata.Scope)
	f.SetCellValue(sheetName, "A3", "Files Scanned:")
	f.SetCellValue(sheetName, "B3", result.Metadata.TotalFilesScanned)
	f.SetCellValue(sheetName, "A4", "Issues Found:")
	f.SetCellValue(sheetName, "B4", result.Metadata.IssuesFound)
	f.SetCellValue(sheetName, "A5", "Timestamp:")
	f.SetCellValue(sheetName, "B5", result.Metadata.Timestamp.Format("2006-01-02 15:04:05"))
	if result.Metadata.DurationSeconds > 0 {
		f.SetCellValue(sheetName, "A6", "Duration (seconds):")
		f.SetCellValue(sheetName, "B6", result.Metadata.DurationSeconds)
	}

	// Style title
	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 14},
	})
	if titleStyle > 0 {
		f.SetCellStyle(sheetName, "A1", "A1", titleStyle)
	}

	return nil
}

// getRiskStyle returns an Excel style ID for the risk level
func getRiskStyle(f *excelize.File, risk models.RiskLevel) int {
	var color string
	switch risk {
	case models.RiskCritical:
		color = "#FF0000" // Red
	case models.RiskHigh:
		color = "#FF8800" // Orange
	case models.RiskMedium:
		color = "#FFCC00" // Yellow
	case models.RiskLow:
		color = "#00FF00" // Green
	default:
		return 0
	}

	style, err := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{color}, Pattern: 1},
		Font: &excelize.Font{Bold: true},
	})
	if err != nil {
		return 0
	}
	return style
}

// sanitizeSheetName sanitizes a string for use as an Excel sheet name
func sanitizeSheetName(name string) string {
	// Excel sheet names cannot contain: \ / ? * [ ]
	invalidChars := []string{"\\", "/", "?", "*", "[", "]"}
	result := name
	for _, char := range invalidChars {
		result = strings.ReplaceAll(result, char, "_")
	}
	// Excel sheet names are limited to 31 characters
	if len(result) > 31 {
		result = result[:31]
	}
	return result
}

