package report

import (
	"fmt"
	"io"
	"strings"

	"gdrive-audit/pkg/models"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Styles for risk levels
	criticalStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))  // Red
	highStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("208")) // Orange
	mediumStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))  // Yellow
	lowStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))  // Green

	// Table styles
	headerStyle = lipgloss.NewStyle().Bold(true).Underline(true)
	borderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

// WriteTable writes scan results as a formatted table to the writer
func WriteTable(result *models.ScanResult, writer io.Writer) error {
	if len(result.Issues) == 0 {
		fmt.Fprintf(writer, "\nNo security issues found.\n")
		return nil
	}

	// Print summary
	fmt.Fprintf(writer, "\n%s\n", strings.Repeat("=", 120))
	fmt.Fprintf(writer, "Scan Results Summary\n")
	fmt.Fprintf(writer, "%s\n", strings.Repeat("=", 120))
	fmt.Fprintf(writer, "Scope: %s\n", result.Metadata.Scope)
	fmt.Fprintf(writer, "Files Scanned: %d\n", result.Metadata.TotalFilesScanned)
	fmt.Fprintf(writer, "Issues Found: %d\n", result.Metadata.IssuesFound)
	if result.Metadata.DurationSeconds > 0 {
		fmt.Fprintf(writer, "Duration: %.2f seconds\n", result.Metadata.DurationSeconds)
	}
	fmt.Fprintf(writer, "%s\n\n", strings.Repeat("=", 120))

	// Print results - one entry per file with all external shares listed
	entryNum := 0
	for _, issue := range result.Issues {
		// Separate internal and external permissions
		internalPerms := []models.Permission{}
		externalPerms := []models.Permission{}
		
		for _, perm := range issue.Permissions {
			if perm.IsInternal {
				internalPerms = append(internalPerms, perm)
			} else {
				externalPerms = append(externalPerms, perm)
			}
		}
		
		// Skip files with no external shares
		if len(externalPerms) == 0 {
			continue
		}
		
		entryNum++
		
		// Entry separator
		fmt.Fprintln(writer, strings.Repeat("-", 120))
		
		// File information
		fmt.Fprintf(writer, "[%d] File: %s\n", entryNum, issue.FileName)
		fmt.Fprintf(writer, "    File ID: %s\n", issue.FileID)
		fmt.Fprintf(writer, "    Drive: %s\n", issue.DriveName)
		
		fullPath := issue.FolderPath
		if fullPath == "" {
			fullPath = "/"
		}
		fmt.Fprintf(writer, "    Path: %s\n", fullPath)
		
		// Link
		if issue.WebViewLink != "" {
			fmt.Fprintf(writer, "    Link: %s\n", issue.WebViewLink)
		}
		
		// Internal users - list all internal users for this file
		if len(internalPerms) > 0 {
			fmt.Fprintf(writer, "    Internal Users:\n")
			for idx, perm := range internalPerms {
				sharedWith := getSharedWith(perm)
				fmt.Fprintf(writer, "      %d. %s (Role: %s)\n", idx+1, sharedWith, perm.Role)
			}
		}
		
		// External shares - list all external users for this file
		fmt.Fprintf(writer, "    External Shares:\n")
		for idx, perm := range externalPerms {
			sharedWith := getSharedWith(perm)
			riskLevel := formatRiskLevel(perm.RiskLevel)
			fmt.Fprintf(writer, "      %d. %s (Role: %s, Risk: %s)\n", idx+1, sharedWith, perm.Role, riskLevel)
		}
	}
	
	if entryNum > 0 {
		fmt.Fprintln(writer, strings.Repeat("-", 120))
	}

	fmt.Fprintf(writer, "\n%s\n", strings.Repeat("=", 120))
	return nil
}

// getInternalUsersSummary returns a comma-separated list of internal users
func getInternalUsersSummary(perms []models.Permission, maxLen int) string {
	if len(perms) == 0 {
		return "-"
	}
	
	// Collect unique internal users
	users := make([]string, 0)
	for _, perm := range perms {
		user := getSharedWith(perm)
		if user != "-" {
			users = append(users, user)
		}
	}
	
	if len(users) == 0 {
		return "-"
	}
	
	// Show all users as comma-separated list
	result := strings.Join(users, ", ")
	
	// Only truncate if maxLen is reasonable (less than 500)
	if maxLen < 500 {
		return truncateString(result, maxLen)
	}
	return result
}

// getSharedWith returns the email or domain for a permission
func getSharedWith(perm models.Permission) string {
	if perm.Email != "" {
		return perm.Email
	}
	if perm.Domain != "" {
		return "@" + perm.Domain
	}
	if perm.Type == "anyone" {
		return "Anyone with link"
	}
	return "-"
}

// formatRiskLevel formats risk level with color
func formatRiskLevel(risk models.RiskLevel) string {
	riskStr := string(risk)
	switch risk {
	case models.RiskCritical:
		return criticalStyle.Render(riskStr)
	case models.RiskHigh:
		return highStyle.Render(riskStr)
	case models.RiskMedium:
		return mediumStyle.Render(riskStr)
	case models.RiskLow:
		return lowStyle.Render(riskStr)
	default:
		return riskStr
	}
}

// truncateString truncates a string to maxLen with ellipsis
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return "..."
	}
	return s[:maxLen-3] + "..."
}
