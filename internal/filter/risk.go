package filter

import "github.com/dplense/dplense-cli/pkg/models"

// CalculateRiskLevel calculates the risk level for a permission based on various factors
func CalculateRiskLevel(perm models.Permission, isPublic bool, isExternal bool) models.RiskLevel {
	// Critical: Public "anyone with link" permissions
	if isPublic {
		return models.RiskCritical
	}

	// High: External users with write/owner access
	if isExternal {
		if perm.Role == "owner" || perm.Role == "writer" || perm.Role == "fileOrganizer" || perm.Role == "organizer" {
			return models.RiskHigh
		}
		// Medium: External users with read/commenter access (or any other role)
		return models.RiskMedium
	}

	// Low: Internal users
	return models.RiskLow
}
