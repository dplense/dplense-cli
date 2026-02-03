package filter

import (
	"testing"

	"gdrive-audit/pkg/models"
)

func TestCalculateRiskLevel(t *testing.T) {
	tests := []struct {
		name       string
		perm       models.Permission
		isPublic   bool
		isExternal bool
		expected   models.RiskLevel
	}{
		{
			name: "public permission is critical",
			perm: models.Permission{
				Type: "anyone",
				Role: "reader",
			},
			isPublic:   true,
			isExternal: true,
			expected:   models.RiskCritical,
		},
		{
			name: "external owner is high",
			perm: models.Permission{
				Type:  "user",
				Email: "external@domain.com",
				Role:  "owner",
			},
			isPublic:   false,
			isExternal: true,
			expected:   models.RiskHigh,
		},
		{
			name: "external writer is high",
			perm: models.Permission{
				Type:  "user",
				Email: "external@domain.com",
				Role:  "writer",
			},
			isPublic:   false,
			isExternal: true,
			expected:   models.RiskHigh,
		},
		{
			name: "external reader is medium",
			perm: models.Permission{
				Type:  "user",
				Email: "external@domain.com",
				Role:  "reader",
			},
			isPublic:   false,
			isExternal: true,
			expected:   models.RiskMedium,
		},
		{
			name: "external commenter is medium",
			perm: models.Permission{
				Type:  "user",
				Email: "external@domain.com",
				Role:  "commenter",
			},
			isPublic:   false,
			isExternal: true,
			expected:   models.RiskMedium,
		},
		{
			name: "internal user is low",
			perm: models.Permission{
				Type:  "user",
				Email: "internal@mycompany.com",
				Role:  "writer",
			},
			isPublic:   false,
			isExternal: false,
			expected:   models.RiskLow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateRiskLevel(tt.perm, tt.isPublic, tt.isExternal)
			if result != tt.expected {
				t.Errorf("CalculateRiskLevel() = %v, want %v", result, tt.expected)
			}
		})
	}
}
