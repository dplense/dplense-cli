package google

import (
	"context"
	"fmt"

	"github.com/dplense/dplense-cli/internal/logger"
	"github.com/dplense/dplense-cli/pkg/config"

	admin "google.golang.org/api/admin/directory/v1"
)

// directoryService implements the DirectoryClient interface
type directoryService struct {
	adminService *admin.Service
	logger       logger.Logger
}

// newDirectoryClient creates a new Directory service client with built-in authentication.
// Handles authentication internally using credentials from config.
func newDirectoryClient(ctx context.Context, cfg *config.Config, log logger.Logger) (DirectoryClient, error) {
	adminService, err := newDirectoryService(ctx, cfg.CredentialsPath, cfg.ImpersonateUser)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Directory service: %w", err)
	}
	return &directoryService{
		adminService: adminService,
		logger:       log,
	}, nil
}

// ListUsers lists users (active or suspended based on suspended parameter)
func (s *directoryService) ListUsers(ctx context.Context, suspended bool) ([]User, error) {
	s.logger.Debug("Listing users, suspended: %v", suspended)

	users := make([]User, 0)
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

		s.logger.Info("Users.List API: returned %d users (suspended=%v), hasNextPage=%v", len(r.Users), suspended, r.NextPageToken != "")

		for _, u := range r.Users {
			users = append(users, User{
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

	s.logger.Info("Users.List completed: total %d users (suspended=%v)", len(users), suspended)
	return users, nil
}

// GetUser retrieves a user by email
func (s *directoryService) GetUser(ctx context.Context, email string) (*User, error) {
	s.logger.Debug("Getting user: %s", email)

	u, err := s.adminService.Users.Get(email).
		Projection("full").
		Context(ctx).
		Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get user %s: %w", email, err)
	}

	s.logger.Info("Users.Get API: email=%s, name=%s, suspended=%v", u.PrimaryEmail, u.Name.FullName, u.Suspended)

	return &User{
		ID:           u.Id,
		Email:        u.PrimaryEmail,
		DisplayName:  u.Name.FullName,
		Suspended:    u.Suspended,
		PrimaryEmail: u.PrimaryEmail,
	}, nil
}
