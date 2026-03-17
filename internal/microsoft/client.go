package microsoft

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/dplense/dplense-cli/internal/logger"

	abstractions "github.com/microsoft/kiota-abstractions-go"
	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
	msgraphcore "github.com/microsoftgraph/msgraph-sdk-go-core"
	graphmodels "github.com/microsoftgraph/msgraph-sdk-go/models"
	"github.com/microsoftgraph/msgraph-sdk-go/sites"
	"github.com/microsoftgraph/msgraph-sdk-go/users"
)

// rateLimiter tracks API request quota using a sliding window.
type rateLimiter struct {
	mu           sync.Mutex
	requests     []time.Time
	quotaLimit   int
	quotaWindow  time.Duration
	maxRetries   int
	baseDelay    time.Duration
	maxDelay     time.Duration
}

func newRateLimiter() *rateLimiter {
	return &rateLimiter{
		quotaLimit:  800,                // stay under Graph's 10k/10min limit with margin
		quotaWindow: 10 * time.Minute,
		maxRetries:  5,
		baseDelay:   2 * time.Second,
		maxDelay:    60 * time.Second,
	}
}

// wait blocks until it's safe to make a request, respecting the quota window.
func (rl *rateLimiter) wait(ctx context.Context) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.quotaWindow)

	// Prune old entries
	valid := rl.requests[:0]
	for _, t := range rl.requests {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	rl.requests = valid

	if len(rl.requests) >= rl.quotaLimit {
		waitTime := rl.quotaWindow - now.Sub(rl.requests[0]) + time.Second
		if waitTime > 0 {
			rl.mu.Unlock()
			select {
			case <-ctx.Done():
				rl.mu.Lock()
				return ctx.Err()
			case <-time.After(waitTime):
			}
			rl.mu.Lock()
		}
	}

	rl.requests = append(rl.requests, time.Now())
	return nil
}

// do executes fn with quota tracking and retries on throttling (429) errors.
func (rl *rateLimiter) do(ctx context.Context, fn func() error) error {
	var lastErr error
	delay := rl.baseDelay

	for attempt := 0; attempt <= rl.maxRetries; attempt++ {
		if err := rl.wait(ctx); err != nil {
			return err
		}

		err := fn()
		if err == nil {
			return nil
		}
		lastErr = err

		if !isThrottled(err) {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			delay *= 2
			if delay > rl.maxDelay {
				delay = rl.maxDelay
			}
		}
	}
	return fmt.Errorf("max retries exceeded: %w", lastErr)
}

// isThrottled checks if the error indicates a Graph API throttle (HTTP 429 or 503).
func isThrottled(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "429") || strings.Contains(s, "throttl") ||
		strings.Contains(s, "too many requests") || strings.Contains(s, "503")
}

// graphClient wraps the Microsoft Graph SDK client with rate-limiting and pagination.
type graphClient struct {
	client  *msgraphsdk.GraphServiceClient
	adapter abstractions.RequestAdapter
	log     logger.Logger
	rl      *rateLimiter
}

func newClient(client *msgraphsdk.GraphServiceClient, adapter abstractions.RequestAdapter, log logger.Logger) *graphClient {
	return &graphClient{client: client, adapter: adapter, log: log, rl: newRateLimiter()}
}

// listAllSites returns all SharePoint sites in the tenant.
func (c *graphClient) listAllSites(ctx context.Context) ([]graphmodels.Siteable, error) {
	var allSites []graphmodels.Siteable

	search := "*"
	cfg := &sites.SitesRequestBuilderGetRequestConfiguration{
		QueryParameters: &sites.SitesRequestBuilderGetQueryParameters{
			Search: &search,
		},
	}

	result, err := c.client.Sites().Get(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("list sites: %w", err)
	}

	pageIterator, err := msgraphcore.NewPageIterator[graphmodels.Siteable](
		result, c.adapter, graphmodels.CreateSiteCollectionResponseFromDiscriminatorValue,
	)
	if err != nil {
		return nil, fmt.Errorf("create site page iterator: %w", err)
	}

	err = pageIterator.Iterate(ctx, func(site graphmodels.Siteable) bool {
		allSites = append(allSites, site)
		return true
	})
	if err != nil {
		return nil, fmt.Errorf("iterate sites: %w", err)
	}

	return allSites, nil
}

// listUsers returns all users in the tenant.
func (c *graphClient) listUsers(ctx context.Context) ([]graphmodels.Userable, error) {
	var allUsers []graphmodels.Userable

	selectFields := []string{"id", "displayName", "userPrincipalName", "mail"}
	cfg := &users.UsersRequestBuilderGetRequestConfiguration{
		QueryParameters: &users.UsersRequestBuilderGetQueryParameters{
			Select: selectFields,
		},
	}

	result, err := c.client.Users().Get(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}

	pageIterator, err := msgraphcore.NewPageIterator[graphmodels.Userable](
		result, c.adapter, graphmodels.CreateUserCollectionResponseFromDiscriminatorValue,
	)
	if err != nil {
		return nil, fmt.Errorf("create user page iterator: %w", err)
	}

	err = pageIterator.Iterate(ctx, func(user graphmodels.Userable) bool {
		allUsers = append(allUsers, user)
		return true
	})
	if err != nil {
		return nil, fmt.Errorf("iterate users: %w", err)
	}

	return allUsers, nil
}

// listSiteDrives returns all document libraries for a site.
func (c *graphClient) listSiteDrives(ctx context.Context, siteID string) ([]graphmodels.Driveable, error) {
	var drives []graphmodels.Driveable
	err := c.rl.do(ctx, func() error {
		result, err := c.client.Sites().BySiteId(siteID).Drives().Get(ctx, nil)
		if err != nil {
			return err
		}
		drives = result.GetValue()
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("list site drives: %w", err)
	}
	return drives, nil
}

// listUserDrive returns the user's default OneDrive.
func (c *graphClient) listUserDrive(ctx context.Context, userID string) (graphmodels.Driveable, error) {
	var drive graphmodels.Driveable
	err := c.rl.do(ctx, func() error {
		result, err := c.client.Users().ByUserId(userID).Drive().Get(ctx, nil)
		if err != nil {
			return err
		}
		drive = result
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("get user drive: %w", err)
	}
	return drive, nil
}

// driveItem represents a file/folder in a drive with its permissions.
type driveItem struct {
	ID          string
	Name        string
	WebURL      string
	Path        string
	Permissions []graphmodels.Permissionable
}

// listDriveItems returns all items in a drive using the children endpoint (recursive).
func (c *graphClient) listDriveItems(ctx context.Context, driveID string) ([]driveItem, error) {
	var items []driveItem

	result, err := c.client.Drives().ByDriveId(driveID).Items().Get(ctx, nil)
	if err != nil {
		// Fallback: list root children
		return c.listDriveRootChildren(ctx, driveID)
	}

	for _, item := range result.GetValue() {
		di := driveItem{
			ID:     deref(item.GetId()),
			Name:   deref(item.GetName()),
			WebURL: deref(item.GetWebUrl()),
		}
		items = append(items, di)
	}

	return items, nil
}

const maxFolderDepth = 25

// listDriveRootChildren lists items from the root of a drive.
func (c *graphClient) listDriveRootChildren(ctx context.Context, driveID string) ([]driveItem, error) {
	return c.listFolderChildren(ctx, driveID, "root", 0)
}

// listFolderChildren recursively lists items in a folder with depth limiting.
func (c *graphClient) listFolderChildren(ctx context.Context, driveID, folderID string, depth int) ([]driveItem, error) {
	if depth > maxFolderDepth {
		c.log.Warn("Skipping folder %s: max depth %d exceeded", folderID, maxFolderDepth)
		return nil, nil
	}

	var children []graphmodels.DriveItemable
	err := c.rl.do(ctx, func() error {
		result, err := c.client.Drives().ByDriveId(driveID).Items().ByDriveItemId(folderID).Children().Get(ctx, nil)
		if err != nil {
			return err
		}
		children = result.GetValue()
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("list folder children: %w", err)
	}

	var allItems []driveItem
	for _, item := range children {
		di := driveItem{
			ID:     deref(item.GetId()),
			Name:   deref(item.GetName()),
			WebURL: deref(item.GetWebUrl()),
		}
		allItems = append(allItems, di)

		if item.GetFolder() != nil {
			children, err := c.listFolderChildren(ctx, driveID, deref(item.GetId()), depth+1)
			if err != nil {
				c.log.Warn("Failed to list children of %s: %v", deref(item.GetName()), err)
				continue
			}
			allItems = append(allItems, children...)
		}
	}

	return allItems, nil
}

// listItemPermissions returns permissions for a specific drive item.
func (c *graphClient) listItemPermissions(ctx context.Context, driveID, itemID string) ([]graphmodels.Permissionable, error) {
	var perms []graphmodels.Permissionable
	err := c.rl.do(ctx, func() error {
		result, err := c.client.Drives().ByDriveId(driveID).Items().ByDriveItemId(itemID).Permissions().Get(ctx, nil)
		if err != nil {
			return err
		}
		perms = result.GetValue()
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("list item permissions: %w", err)
	}
	return perms, nil
}

// deletePermission removes a permission from a drive item.
func (c *graphClient) deletePermission(ctx context.Context, driveID, itemID, permID string) error {
	err := c.rl.do(ctx, func() error {
		return c.client.Drives().ByDriveId(driveID).Items().ByDriveItemId(itemID).Permissions().ByPermissionId(permID).Delete(ctx, nil)
	})
	if err != nil {
		return fmt.Errorf("delete permission: %w", err)
	}
	return nil
}
