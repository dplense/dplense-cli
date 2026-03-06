package directory

import (
	"context"
	"fmt"

	"gdrive-audit/internal/auth"
	"gdrive-audit/internal/logger"
	"gdrive-audit/pkg/config"
	"gdrive-audit/pkg/gdrive"

	admin "google.golang.org/api/admin/directory/v1"
)

// service implements the DirectoryClient interface
type service struct {
	adminService *admin.Service
	logger       logger.Logger
}

// NewService creates a new Directory service client with built-in authentication.
// Handles authentication internally using credentials from config.
func NewService(ctx context.Context, cfg *config.Config, log logger.Logger) (gdrive.DirectoryClient, error) {
	adminService, err := auth.NewDirectoryService(ctx, cfg.CredentialsPath, cfg.ImpersonateUser)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Directory service: %w", err)
	}
	return &service{
		adminService: adminService,
		logger:       log,
	}, nil
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

		s.logger.Info("Users.List API: returned %d users (suspended=%v), hasNextPage=%v", len(r.Users), suspended, r.NextPageToken != "")

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

	s.logger.Info("Users.List completed: total %d users (suspended=%v)", len(users), suspended)
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

	s.logger.Info("Users.Get API: email=%s, name=%s, suspended=%v", u.PrimaryEmail, u.Name.FullName, u.Suspended)

	return &gdrive.User{
		ID:           u.Id,
		Email:        u.PrimaryEmail,
		DisplayName:  u.Name.FullName,
		Suspended:    u.Suspended,
		PrimaryEmail: u.PrimaryEmail,
	}, nil
}
