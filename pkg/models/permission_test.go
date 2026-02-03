package models

import (
	"testing"
)

func TestPermission_IsPublic(t *testing.T) {
	tests := []struct {
		name       string
		permission Permission
		expected   bool
	}{
		{
			name: "public anyone permission",
			permission: Permission{
				Type: "anyone",
			},
			expected: true,
		},
		{
			name: "user permission",
			permission: Permission{
				Type: "user",
			},
			expected: false,
		},
		{
			name: "domain permission",
			permission: Permission{
				Type: "domain",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.permission.IsPublic(); got != tt.expected {
				t.Errorf("IsPublic() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPermission_IsExternal(t *testing.T) {
	internalDomains := []string{"mycompany.com", "mycompany.se"}

	tests := []struct {
		name       string
		permission Permission
		expected   bool
	}{
		{
			name: "public permission is external",
			permission: Permission{
				Type: "anyone",
			},
			expected: true,
		},
		{
			name: "internal user email",
			permission: Permission{
				Type:  "user",
				Email: "user@mycompany.com",
			},
			expected: false,
		},
		{
			name: "external user email",
			permission: Permission{
				Type:  "user",
				Email: "user@external.com",
			},
			expected: true,
		},
		{
			name: "internal domain",
			permission: Permission{
				Type:   "domain",
				Domain: "mycompany.com",
			},
			expected: false,
		},
		{
			name: "external domain",
			permission: Permission{
				Type:   "domain",
				Domain: "external.com",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.permission.IsExternal(internalDomains); got != tt.expected {
				t.Errorf("IsExternal() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPermission_Validate(t *testing.T) {
	tests := []struct {
		name       string
		permission Permission
		wantErr    bool
	}{
		{
			name: "valid permission",
			permission: Permission{
				ID:        "perm1",
				Type:      "user",
				Email:     "user@example.com",
				Role:      "reader",
				RiskLevel: RiskMedium,
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			permission: Permission{
				Type:      "user",
				Role:      "reader",
				RiskLevel: RiskMedium,
			},
			wantErr: true,
		},
		{
			name: "missing type",
			permission: Permission{
				ID:        "perm1",
				Role:      "reader",
				RiskLevel: RiskMedium,
			},
			wantErr: true,
		},
		{
			name: "missing role",
			permission: Permission{
				ID:        "perm1",
				Type:      "user",
				RiskLevel: RiskMedium,
			},
			wantErr: true,
		},
		{
			name: "invalid risk level",
			permission: Permission{
				ID:        "perm1",
				Type:      "user",
				Role:      "reader",
				RiskLevel: RiskLevel("invalid"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.permission.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
