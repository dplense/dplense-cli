package slack

import (
	"context"
	"fmt"
	"time"

	"github.com/dplense/dplense-cli/internal/logger"
	"github.com/dplense/dplense-cli/pkg/config"
	"github.com/dplense/dplense-cli/pkg/models"
	"github.com/dplense/dplense-cli/pkg/provider"
)

type scanner struct {
	sc  *slackClient
	cfg *config.Config
	log logger.Logger
}

func newScanner(sc *slackClient, cfg *config.Config, log logger.Logger) *scanner {
	return &scanner{sc: sc, cfg: cfg, log: log}
}

func (s *scanner) scan(ctx context.Context, opts provider.ScanOptions) (*models.ScanResult, error) {
	startTime := time.Now()
	result := models.NewScanResult(opts.Scope)
	result.Metadata.Provider = string(provider.Slack)

	totalItems := 0

	switch opts.Scope {
	case "files":
		if err := s.scanFiles(ctx, opts, result, &totalItems); err != nil {
			return nil, err
		}
	case "channels":
		if err := s.scanChannels(ctx, opts, result, &totalItems); err != nil {
			return nil, err
		}
	case "guests":
		if err := s.scanGuests(ctx, opts, result, &totalItems); err != nil {
			return nil, err
		}
	case "all":
		if err := s.scanFiles(ctx, opts, result, &totalItems); err != nil {
			s.log.Warn("Files scan failed: %v", err)
			result.AddError(fmt.Sprintf("files: %v", err))
		}
		if err := s.scanChannels(ctx, opts, result, &totalItems); err != nil {
			s.log.Warn("Channels scan failed: %v", err)
			result.AddError(fmt.Sprintf("channels: %v", err))
		}
		if err := s.scanGuests(ctx, opts, result, &totalItems); err != nil {
			s.log.Warn("Guests scan failed: %v", err)
			result.AddError(fmt.Sprintf("guests: %v", err))
		}
	default:
		return nil, fmt.Errorf("unknown scope: %s (valid: files, channels, guests, all)", opts.Scope)
	}

	result.SetFilesScanned(totalItems)
	result.SetDuration(time.Since(startTime))
	s.log.Info("Slack scan complete: %d items scanned, %d issues found in %s",
		totalItems, len(result.Issues), time.Since(startTime).Round(time.Millisecond))
	return result, nil
}

func (s *scanner) scanFiles(ctx context.Context, opts provider.ScanOptions, result *models.ScanResult, totalItems *int) error {
	s.log.Info("Scanning Slack files...")

	// Get all files
	files, err := s.sc.listAllFiles(ctx)
	if err != nil {
		return fmt.Errorf("list files: %w", err)
	}
	s.log.Info("Found %d files", len(files))
	*totalItems += len(files)

	// Get externally-shared channels for cross-reference
	allChannels, err := s.sc.listConversations(ctx, []string{"public_channel", "private_channel"})
	if err != nil {
		s.log.Warn("Failed to list channels: %v", err)
	}

	extChannelMap := make(map[string]bool)
	channelMap := make(map[string]string) // channelID -> channelName
	for _, ch := range allChannels {
		channelMap[ch.ID] = ch.Name
		if ch.IsExtShared || ch.IsPendingExtShared {
			extChannelMap[ch.ID] = true
		}
	}

	if opts.TargetProgressFunc != nil {
		opts.TargetProgressFunc(1, 3, "Files")
	}

	for _, file := range files {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Check if file has public URL
		if file.IsPublic {
			issue := mapPublicFile(file)
			if provider.MatchesFilters(issue, opts) {
				result.AddIssue(issue)
			}
			continue
		}

		// Check if file is in an externally-shared channel
		for _, chID := range file.Channels {
			if extChannelMap[chID] {
				chName := channelMap[chID]
				if chName == "" {
					chName = chID
				}
				issue := mapFileInExtChannel(file, chID, chName)
				if provider.MatchesFilters(issue, opts) {
					result.AddIssue(issue)
				}
				break // Only add once per file
			}
		}
	}

	if opts.ProgressFunc != nil {
		opts.ProgressFunc(*totalItems, len(result.Issues))
	}

	return nil
}

func (s *scanner) scanChannels(ctx context.Context, opts provider.ScanOptions, result *models.ScanResult, totalItems *int) error {
	s.log.Info("Scanning Slack channels for external sharing...")

	channels, err := s.sc.listConversations(ctx, []string{"public_channel", "private_channel"})
	if err != nil {
		return fmt.Errorf("list conversations: %w", err)
	}
	s.log.Info("Found %d channels", len(channels))
	*totalItems += len(channels)

	if opts.TargetProgressFunc != nil {
		opts.TargetProgressFunc(2, 3, "Channels")
	}

	for _, ch := range channels {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if ch.IsExtShared || ch.IsPendingExtShared {
			issue := mapExternalChannel(ch)
			if provider.MatchesFilters(issue, opts) {
				result.AddIssue(issue)
			}
		}
	}

	if opts.ProgressFunc != nil {
		opts.ProgressFunc(*totalItems, len(result.Issues))
	}

	return nil
}

func (s *scanner) scanGuests(ctx context.Context, opts provider.ScanOptions, result *models.ScanResult, totalItems *int) error {
	s.log.Info("Scanning Slack guest users...")

	users, err := s.sc.listUsers(ctx)
	if err != nil {
		return fmt.Errorf("list users: %w", err)
	}

	if opts.TargetProgressFunc != nil {
		opts.TargetProgressFunc(3, 3, "Guest Users")
	}

	guestCount := 0
	for _, user := range users {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if !user.IsRestricted && !user.IsUltraRestricted {
			continue
		}
		if user.Deleted {
			continue
		}

		guestCount++
		*totalItems++

		// Get channels the guest has access to
		channels, err := s.sc.getUserConversations(ctx, user.ID)
		if err != nil {
			s.log.Debug("Failed to get channels for guest %s: %v", user.RealName, err)
			continue
		}

		for _, ch := range channels {
			issue := mapGuestUser(user, ch)
			if provider.MatchesFilters(issue, opts) {
				result.AddIssue(issue)
			}
		}
	}

	s.log.Info("Found %d guest users", guestCount)

	if opts.ProgressFunc != nil {
		opts.ProgressFunc(*totalItems, len(result.Issues))
	}

	return nil
}


