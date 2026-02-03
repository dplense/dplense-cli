package directory

import (
	"context"
	"fmt"

	"gdrive-audit/internal/logger"
	"gdrive-audit/pkg/gdrive"

	admin "google.golang.org/api/admin/directory/v1"
)

// service implements the DirectoryClient interface
type service struct {
	adminService *admin.Service
	logger       logger.Logger
}

// NewService creates a new Directory service client
func NewService(adminService *admin.Service, log logger.Logger) gdrive.DirectoryClient {
	return &service{
		adminService: adminService,
		logger:       log,
	}
}

// ListUsers lists users (active or suspended based on suspended parameter)
func (s *service) ListUsers(ctx context.Context, suspended bool) ([]gdrive.User, error) {
	s.logger.Debug("Listing users, suspended: %v", suspended)

	users := make([]gdrive.User, 0)
	pageToken := ""

	for {
		call := s.adminService.Users.List().
			MaxResults(500).
			Projection("full").
			OrderBy("email")

		if suspended {
			call = call.Query("isSuspended=true")
		} else {
			call = call.Query("isSuspended=false")
		}

		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		r, err := call.Context(ctx).Do()
		if err != nil {
			return nil, fmt.Errorf("failed to list users: %w", err)
		}

		for _, u := range r.Users {
			users = append(users, gdrive.User{
				ID:           u.Id,
				Email:        u.PrimaryEmail,
				DisplayName:  u.Name.FullName,
				Suspended:    u.Suspended,
				PrimaryEmail: u.PrimaryEmail,
			})
		}

		if r.NextPageToken == "" {
			break
		}
		pageToken = r.NextPageToken
	}

	return users, nil
}

// GetUser retrieves a user by email
func (s *service) GetUser(ctx context.Context, email string) (*gdrive.User, error) {
	s.logger.Debug("Getting user: %s", email)

	u, err := s.adminService.Users.Get(email).
		Projection("full").
		Context(ctx).
		Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get user %s: %w", email, err)
	}

	return &gdrive.User{
		ID:           u.Id,
		Email:        u.PrimaryEmail,
		DisplayName:  u.Name.FullName,
		Suspended:    u.Suspended,
		PrimaryEmail: u.PrimaryEmail,
	}, nil
}
