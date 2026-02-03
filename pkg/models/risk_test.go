package models

import (
	"testing"
)

func TestRiskLevel_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		risk     RiskLevel
		expected bool
	}{
		{"critical", RiskCritical, true},
		{"high", RiskHigh, true},
		{"medium", RiskMedium, true},
		{"low", RiskLow, true},
		{"invalid", RiskLevel("invalid"), false},
		{"empty", RiskLevel(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.risk.IsValid(); got != tt.expected {
				t.Errorf("IsValid() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRiskLevel_String(t *testing.T) {
	if RiskCritical.String() != "critical" {
		t.Errorf("Expected 'critical', got '%s'", RiskCritical.String())
	}
	if RiskHigh.String() != "high" {
		t.Errorf("Expected 'high', got '%s'", RiskHigh.String())
	}
}

func TestValidRiskLevels(t *testing.T) {
	levels := ValidRiskLevels()
	expectedCount := 4
	if len(levels) != expectedCount {
		t.Errorf("Expected %d risk levels, got %d", expectedCount, len(levels))
	}

	expectedLevels := map[RiskLevel]bool{
		RiskCritical: true,
		RiskHigh:     true,
		RiskMedium:   true,
		RiskLow:      true,
	}

	for _, level := range levels {
		if !expectedLevels[level] {
			t.Errorf("Unexpected risk level: %s", level)
		}
	}
}
