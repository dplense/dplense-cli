package microsoft

import (
	"strings"

	"github.com/dplense/dplense-cli/internal/filter"
	"github.com/dplense/dplense-cli/pkg/models"

	graphmodels "github.com/microsoftgraph/msgraph-sdk-go/models"
)

// mapPermission converts a Microsoft Graph permission to a models.Permission.
func mapPermission(gp graphmodels.Permissionable, internalDomains []string) models.Permission {
	perm := models.Permission{
		ID:   deref(gp.GetId()),
		Role: mapRole(gp.GetRoles()),
	}

	// Determine permission type from the Graph permission facets
	if link := gp.GetLink(); link != nil {
		// Sharing link
		scope := deref(link.GetScope())
		switch scope {
		case "anonymous":
			perm.Type = "anyone"
		case "organization":
			perm.Type = "domain"
		case "users":
			perm.Type = "user"
			// Extract email from grantedToIdentitiesV2
			if identities := gp.GetGrantedToIdentitiesV2(); len(identities) > 0 {
				if user := identities[0].GetUser(); user != nil {
					perm.Email = deref(user.GetDisplayName()) // Graph may not always return email here
				}
			}
		default:
			perm.Type = "user"
		}
	} else if granted := gp.GetGrantedToV2(); granted != nil {
		// Direct permission
		perm.Type = "user"
		if user := granted.GetUser(); user != nil {
			perm.Email = deref(user.GetDisplayName())
			if additionalData := user.GetAdditionalData(); additionalData != nil {
				if email, ok := additionalData["email"]; ok {
					if emailStr, ok := email.(*string); ok && emailStr != nil {
						perm.Email = *emailStr
					}
				}
			}
		}
	} else {
		perm.Type = "user"
	}

	// Determine if internal
	isPublic := perm.Type == "anyone"
	isExternal := isPublic
	if !isPublic && perm.Email != "" {
		isExternal = !filter.IsInternalDomain(perm.Email, internalDomains)
	}
	perm.IsInternal = !isExternal

	// Calculate risk
	perm.RiskLevel = filter.CalculateRiskLevel(perm, isPublic, isExternal)

	return perm
}

// mapRole converts Graph roles slice to a single role string.
func mapRole(roles []string) string {
	if len(roles) == 0 {
		return "reader"
	}
	role := strings.ToLower(roles[0])
	switch role {
	case "owner":
		return "owner"
	case "write":
		return "writer"
	case "read":
		return "reader"
	default:
		return role
	}
}

// isInherited checks if a permission is inherited from a parent.
func isInherited(gp graphmodels.Permissionable) bool {
	return gp.GetInheritedFrom() != nil
}

// deref safely dereferences a string pointer.
func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
