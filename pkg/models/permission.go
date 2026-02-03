package models

import (
	"fmt"
	"strings"
)

// PermissionType represents the type of permission
type PermissionType string

const (
	// PermissionTypeUser indicates a user permission
	PermissionTypeUser PermissionType = "user"
	// PermissionTypeGroup indicates a group permission
	PermissionTypeGroup PermissionType = "group"
	// PermissionTypeDomain indicates a domain permission
	PermissionTypeDomain PermissionType = "domain"
	// PermissionTypeAnyone indicates public "anyone with link" permission
	PermissionTypeAnyone PermissionType = "anyone"
)

// PermissionRole represents the role/access level
type PermissionRole string

const (
	// RoleOwner indicates full ownership
	RoleOwner PermissionRole = "owner"
	// RoleWriter indicates write access
	RoleWriter PermissionRole = "writer"
	// RoleCommenter indicates comment access
	RoleCommenter PermissionRole = "commenter"
	// RoleReader indicates read-only access
	RoleReader PermissionRole = "reader"
)

// Permission represents a file permission with risk assessment
type Permission struct {
	ID         string    `json:"id"`
	Type       string    `json:"type"`        // "user", "group", "domain", "anyone"
	Email      string    `json:"email,omitempty"`
	Domain     string    `json:"domain,omitempty"`
	Role       string    `json:"role"`        // "owner", "writer", "commenter", "reader"
	RiskLevel  RiskLevel `json:"risk_level"`  // Calculated risk level
	IsInternal bool      `json:"is_internal"` // True if from internal_domains
}

// IsPublic returns true if this is a public "anyone with link" permission
func (p *Permission) IsPublic() bool {
	return strings.ToLower(p.Type) == string(PermissionTypeAnyone)
}

// IsExternal checks if the permission is for an external user/domain
func (p *Permission) IsExternal(internalDomains []string) bool {
	if p.IsPublic() {
		return true
	}

	email := strings.ToLower(p.Email)
	domain := strings.ToLower(p.Domain)

	for _, internalDomain := range internalDomains {
		internalDomainLower := strings.ToLower(internalDomain)
		if strings.HasSuffix(email, "@"+internalDomainLower) {
			return false
		}
		if domain == internalDomainLower {
			return false
		}
	}

	return true
}

// Validate checks if the permission has required fields
func (p *Permission) Validate() error {
	if p.ID == "" {
		return fmt.Errorf("permission ID is required")
	}
	if p.Type == "" {
		return fmt.Errorf("permission type is required")
	}
	if p.Role == "" {
		return fmt.Errorf("permission role is required")
	}
	if !p.RiskLevel.IsValid() {
		return fmt.Errorf("invalid risk level: %s", p.RiskLevel)
	}
	return nil
}
