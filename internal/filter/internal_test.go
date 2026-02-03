package filter

import (
	"testing"

	"gdrive-audit/pkg/models"
)

func TestIsInternalDomain(t *testing.T) {
	internalDomains := []string{"mycompany.com", "mycompany.se"}

	tests := []struct {
		name     string
		email    string
		expected bool
	}{
		{"internal email 1", "user@mycompany.com", true},
		{"internal email 2", "user@mycompany.se", true},
		{"external email", "user@external.com", false},
		{"empty email", "", false},
		{"case insensitive", "USER@MYCOMPANY.COM", true},
		{"domain with @ prefix", "user@mycompany.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsInternalDomain(tt.email, internalDomains)
			if result != tt.expected {
				t.Errorf("IsInternalDomain(%q) = %v, want %v", tt.email, result, tt.expected)
			}
		})
	}
}

func TestShouldExcludeInternalOnly(t *testing.T) {
	internalDomains := []string{"mycompany.com"}

	tests := []struct {
		name    string
		file    models.FileIssue
		exclude bool
	}{
		{
			name: "all internal permissions",
			file: models.FileIssue{
				Permissions: []models.Permission{
					{Email: "user1@mycompany.com", Type: "user", Role: "reader"},
					{Email: "user2@mycompany.com", Type: "user", Role: "writer"},
				},
			},
			exclude: true,
		},
		{
			name: "has external permission",
			file: models.FileIssue{
				Permissions: []models.Permission{
					{Email: "user1@mycompany.com", Type: "user", Role: "reader"},
					{Email: "external@domain.com", Type: "user", Role: "reader"},
				},
			},
			exclude: false,
		},
		{
			name: "has public permission",
			file: models.FileIssue{
				Permissions: []models.Permission{
					{Email: "user1@mycompany.com", Type: "user", Role: "reader"},
					{Type: "anyone", Role: "reader"},
				},
			},
			exclude: false,
		},
		{
			name: "no permissions",
			file: models.FileIssue{
				Permissions: []models.Permission{},
			},
			exclude: false,
		},
		{
			name: "internal domain permission",
			file: models.FileIssue{
				Permissions: []models.Permission{
					{Domain: "mycompany.com", Type: "domain", Role: "reader"},
				},
			},
			exclude: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldExcludeInternalOnly(tt.file, internalDomains)
			if result != tt.exclude {
				t.Errorf("ShouldExcludeInternalOnly() = %v, want %v", result, tt.exclude)
			}
		})
	}
}
