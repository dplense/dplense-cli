package filter

import "gdrive-audit/pkg/models"

// CalculateRiskLevel calculates the risk level for a permission based on various factors
func CalculateRiskLevel(perm models.Permission, isPublic bool, isExternal bool) models.RiskLevel {
	// Critical: Public "anyone with link" permissions
	if isPublic {
		return models.RiskCritical
	}

	// High: External users with write/owner access
	if isExternal {
		if perm.Role == "owner" || perm.Role == "writer" {
			return models.RiskHigh
		}
		// Medium: External users with read access
		if perm.Role == "reader" || perm.Role == "commenter" {
			return models.RiskMedium
		}
	}

	// Low: Internal users or unknown cases
	return models.RiskLow
}
