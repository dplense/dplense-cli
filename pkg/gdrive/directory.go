package gdrive

import "context"

// User represents a Google Workspace user
type User struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	DisplayName  string `json:"displayName"`
	Suspended    bool   `json:"suspended"`
	PrimaryEmail string `json:"primaryEmail"`
}

// DirectoryClient interface for Admin Directory API operations
type DirectoryClient interface {
	// ListUsers lists users (active or suspended based on suspended parameter)
	ListUsers(ctx context.Context, suspended bool) ([]User, error)

	// GetUser retrieves a user by email
	GetUser(ctx context.Context, email string) (*User, error)
}
