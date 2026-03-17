package provider

import (
	"strings"

	"github.com/dplense/dplense-cli/pkg/models"
)

// MatchesFilters checks if a FileIssue passes all active scan filters.
// Shared across all providers to eliminate filter logic duplication.
func MatchesFilters(issue models.FileIssue, opts ScanOptions) bool {
	if opts.FilterPublicOnly {
		hasPublic := false
		for _, p := range issue.Permissions {
			if p.Type == "anyone" {
				hasPublic = true
				break
			}
		}
		if !hasPublic {
			return false
		}
	}

	if opts.FilterSharedWith != "" {
		matches := false
		target := strings.ToLower(opts.FilterSharedWith)
		for _, p := range issue.Permissions {
			if strings.Contains(strings.ToLower(p.Email), target) {
				matches = true
				break
			}
		}
		if !matches {
			return false
		}
	}

	if len(opts.FilterRiskLevels) > 0 {
		hasMatch := false
		for _, p := range issue.Permissions {
			for _, level := range opts.FilterRiskLevels {
				if strings.EqualFold(string(p.RiskLevel), level) {
					hasMatch = true
					break
				}
			}
			if hasMatch {
				break
			}
		}
		if !hasMatch {
			return false
		}
	}

	return true
}
