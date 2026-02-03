package models

// RiskLevel represents the security risk level of a permission
type RiskLevel string

const (
	// RiskCritical indicates a critical security risk (e.g., public access)
	RiskCritical RiskLevel = "critical"
	// RiskHigh indicates a high security risk (e.g., external user with write access)
	RiskHigh RiskLevel = "high"
	// RiskMedium indicates a medium security risk (e.g., external user with read access)
	RiskMedium RiskLevel = "medium"
	// RiskLow indicates a low security risk (e.g., internal user access)
	RiskLow RiskLevel = "low"
)

// ValidRiskLevels returns all valid risk levels
func ValidRiskLevels() []RiskLevel {
	return []RiskLevel{RiskCritical, RiskHigh, RiskMedium, RiskLow}
}

// IsValid checks if a risk level is valid
func (r RiskLevel) IsValid() bool {
	for _, valid := range ValidRiskLevels() {
		if r == valid {
			return true
		}
	}
	return false
}

// String returns the string representation of the risk level
func (r RiskLevel) String() string {
	return string(r)
}
