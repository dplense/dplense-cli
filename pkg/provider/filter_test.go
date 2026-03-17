package provider

import (
	"testing"

	"github.com/dplense/dplense-cli/pkg/models"
)

func TestMatchesFilters_NoFilters(t *testing.T) {
	issue := models.FileIssue{
		Permissions: []models.Permission{{Type: "user", Email: "ext@example.com"}},
	}
	if !MatchesFilters(issue, ScanOptions{}) {
		t.Error("expected match with no filters")
	}
}

func TestMatchesFilters_PublicOnly(t *testing.T) {
	public := models.FileIssue{
		Permissions: []models.Permission{{Type: "anyone"}},
	}
	private := models.FileIssue{
		Permissions: []models.Permission{{Type: "user", Email: "x@y.com"}},
	}

	opts := ScanOptions{FilterPublicOnly: true}
	if !MatchesFilters(public, opts) {
		t.Error("public file should match public-only filter")
	}
	if MatchesFilters(private, opts) {
		t.Error("private file should not match public-only filter")
	}
}

func TestMatchesFilters_SharedWith(t *testing.T) {
	issue := models.FileIssue{
		Permissions: []models.Permission{
			{Type: "user", Email: "alice@example.com"},
			{Type: "user", Email: "bob@other.com"},
		},
	}

	if !MatchesFilters(issue, ScanOptions{FilterSharedWith: "alice"}) {
		t.Error("should match partial email")
	}
	if !MatchesFilters(issue, ScanOptions{FilterSharedWith: "ALICE"}) {
		t.Error("should match case-insensitive")
	}
	if MatchesFilters(issue, ScanOptions{FilterSharedWith: "charlie"}) {
		t.Error("should not match absent user")
	}
}

func TestMatchesFilters_RiskLevels(t *testing.T) {
	issue := models.FileIssue{
		Permissions: []models.Permission{
			{Type: "anyone", RiskLevel: models.RiskCritical},
		},
	}

	if !MatchesFilters(issue, ScanOptions{FilterRiskLevels: []string{"critical"}}) {
		t.Error("should match critical risk level")
	}
	if !MatchesFilters(issue, ScanOptions{FilterRiskLevels: []string{"Critical"}}) {
		t.Error("should match case-insensitive risk level")
	}
	if MatchesFilters(issue, ScanOptions{FilterRiskLevels: []string{"low"}}) {
		t.Error("should not match non-matching risk level")
	}
}

func TestMatchesFilters_Combined(t *testing.T) {
	issue := models.FileIssue{
		Permissions: []models.Permission{
			{Type: "anyone", Email: "", RiskLevel: models.RiskCritical},
		},
	}

	// Both filters match
	opts := ScanOptions{FilterPublicOnly: true, FilterRiskLevels: []string{"critical"}}
	if !MatchesFilters(issue, opts) {
		t.Error("should match when both filters match")
	}

	// One filter fails
	opts2 := ScanOptions{FilterPublicOnly: true, FilterRiskLevels: []string{"low"}}
	if MatchesFilters(issue, opts2) {
		t.Error("should not match when risk level filter fails")
	}
}
