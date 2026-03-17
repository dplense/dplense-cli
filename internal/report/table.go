package report

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/dplense/dplense-cli/pkg/models"

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
		fmt.Fprintf(writer, "    Label: %s\n", issue.Label)

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

	// Print summary box at the end
	writeSummaryBox(result, writer)

	return nil
}

// driveRiskStats holds per-drive issue counts by risk level.
type driveRiskStats struct {
	total    int
	critical int
	high     int
	medium   int
	low      int
}

// writeSummaryBox renders a styled summary box at the end of the table output.
func writeSummaryBox(result *models.ScanResult, writer io.Writer) {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("69"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Width(8)
	valueStyle := lipgloss.NewStyle().Bold(true)

	// Count issues by risk level (use highest risk per file) and per-drive stats
	riskCounts := map[models.RiskLevel]int{}
	driveStats := map[string]*driveRiskStats{} // driveName -> stats
	// Track insertion order for stable output
	driveOrder := []string{}

	for _, issue := range result.Issues {
		maxRisk := models.RiskLevel("")
		for _, perm := range issue.Permissions {
			if perm.IsInternal {
				continue
			}
			if maxRisk == "" || riskOrder(perm.RiskLevel) > riskOrder(maxRisk) {
				maxRisk = perm.RiskLevel
			}
		}
		if maxRisk != "" {
			riskCounts[maxRisk]++

			// Per-drive breakdown
			driveName := issue.DriveName
			if driveName == "" {
				driveName = "Unknown Drive"
			}
			stats, ok := driveStats[driveName]
			if !ok {
				stats = &driveRiskStats{}
				driveStats[driveName] = stats
				driveOrder = append(driveOrder, driveName)
			}
			stats.total++
			switch maxRisk {
			case models.RiskCritical:
				stats.critical++
			case models.RiskHigh:
				stats.high++
			case models.RiskMedium:
				stats.medium++
			case models.RiskLow:
				stats.low++
			}
		}
	}

	// Build risk breakdown with colored labels
	riskParts := []string{}
	dot := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(" · ")
	for _, level := range []models.RiskLevel{models.RiskCritical, models.RiskHigh, models.RiskMedium, models.RiskLow} {
		count, ok := riskCounts[level]
		if !ok || count == 0 {
			continue
		}
		part := fmt.Sprintf("%d %s", count, level)
		switch level {
		case models.RiskCritical:
			riskParts = append(riskParts, criticalStyle.Render(part))
		case models.RiskHigh:
			riskParts = append(riskParts, highStyle.Render(part))
		case models.RiskMedium:
			riskParts = append(riskParts, mediumStyle.Render(part))
		case models.RiskLow:
			riskParts = append(riskParts, lowStyle.Render(part))
		}
	}

	// Build content lines
	lines := []string{
		titleStyle.Render("Scan Results"),
		"",
		labelStyle.Render("Scope") + "  " + result.Metadata.Scope,
		labelStyle.Render("Files") + "  " + valueStyle.Render(fmt.Sprintf("%d", result.Metadata.TotalFilesScanned)) +
			" scanned, " + valueStyle.Render(fmt.Sprintf("%d", result.Metadata.IssuesFound)) + " with issues",
	}

	if len(riskParts) > 0 {
		lines = append(lines, labelStyle.Render("Risk") + "  " + strings.Join(riskParts, dot))
	}

	if result.Metadata.DurationSeconds > 0 {
		d := time.Duration(result.Metadata.DurationSeconds * float64(time.Second))
		lines = append(lines, labelStyle.Render("Time") + "  " + formatDuration(d))
	}

	// Per-drive breakdown (only drives with issues)
	if len(driveStats) > 0 {
		lines = append(lines, "")
		lines = append(lines, titleStyle.Render("Issues by Drive"))
		for _, driveName := range driveOrder {
			stats := driveStats[driveName]
			parts := []string{}
			if stats.critical > 0 {
				parts = append(parts, criticalStyle.Render(fmt.Sprintf("%d critical", stats.critical)))
			}
			if stats.high > 0 {
				parts = append(parts, highStyle.Render(fmt.Sprintf("%d high", stats.high)))
			}
			if stats.medium > 0 {
				parts = append(parts, mediumStyle.Render(fmt.Sprintf("%d medium", stats.medium)))
			}
			if stats.low > 0 {
				parts = append(parts, lowStyle.Render(fmt.Sprintf("%d low", stats.low)))
			}
			name := driveName
			if len(name) > 30 {
				name = name[:27] + "..."
			}
			lines = append(lines, fmt.Sprintf("  %-30s %s", name, strings.Join(parts, dot)))
		}
	}

	content := strings.Join(lines, "\n")

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("69")).
		Padding(0, 1)

	fmt.Fprintf(writer, "\n%s\n\n", boxStyle.Render(content))
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
