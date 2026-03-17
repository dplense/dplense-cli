package slack

import (
	"context"
	"time"

	"github.com/dplense/dplense-cli/internal/logger"

	slackapi "github.com/slack-go/slack"
)

// slackClient wraps the Slack API client with rate-limit handling.
type slackClient struct {
	api *slackapi.Client
	log logger.Logger
}

func newSlackClient(botToken string, log logger.Logger) *slackClient {
	api := slackapi.New(botToken)
	return &slackClient{api: api, log: log}
}

// listAllFiles returns all files in the workspace.
func (c *slackClient) listAllFiles(ctx context.Context) ([]slackapi.File, error) {
	var allFiles []slackapi.File
	cursor := ""

	for {
		params := slackapi.ListFilesParameters{
			Limit:  100,
			Cursor: cursor,
		}

		files, nextParams, err := c.api.ListFilesContext(ctx, params)
		if err != nil {
			if rateLimitErr, ok := err.(*slackapi.RateLimitedError); ok {
				c.log.Info("Rate limited, waiting %s", rateLimitErr.RetryAfter)
				time.Sleep(rateLimitErr.RetryAfter)
				continue
			}
			return nil, err
		}

		allFiles = append(allFiles, files...)

		if nextParams == nil || nextParams.Cursor == "" {
			break
		}
		cursor = nextParams.Cursor
	}

	return allFiles, nil
}

// listConversations returns all channels matching the given types.
func (c *slackClient) listConversations(ctx context.Context, types []string) ([]slackapi.Channel, error) {
	var allChannels []slackapi.Channel
	cursor := ""

	for {
		params := &slackapi.GetConversationsParameters{
			Cursor:          cursor,
			Limit:           200,
			Types:           types,
			ExcludeArchived: false,
		}

		channels, nextCursor, err := c.api.GetConversationsContext(ctx, params)
		if err != nil {
			if rateLimitErr, ok := err.(*slackapi.RateLimitedError); ok {
				c.log.Info("Rate limited, waiting %s", rateLimitErr.RetryAfter)
				time.Sleep(rateLimitErr.RetryAfter)
				continue
			}
			return nil, err
		}

		allChannels = append(allChannels, channels...)

		if nextCursor == "" {
			break
		}
		cursor = nextCursor
	}

	return allChannels, nil
}

// listUsers returns all users in the workspace.
func (c *slackClient) listUsers(ctx context.Context) ([]slackapi.User, error) {
	var allUsers []slackapi.User

	pager := c.api.GetUsersPaginated(slackapi.GetUsersOptionLimit(200))
	for {
		var err error
		pager, err = pager.Next(ctx)
		if pager.Done(err) {
			if err != nil {
				// Done returns true on both completion and error; check which
				if pager.Failure(err) != nil {
					return allUsers, pager.Failure(err)
				}
			}
			break
		}
		if err != nil {
			if rateLimitErr, ok := err.(*slackapi.RateLimitedError); ok {
				c.log.Info("Rate limited, waiting %s", rateLimitErr.RetryAfter)
				time.Sleep(rateLimitErr.RetryAfter)
				continue
			}
			return nil, err
		}

		allUsers = append(allUsers, pager.Users...)
	}

	return allUsers, nil
}

// getUserConversations returns channels a user belongs to.
func (c *slackClient) getUserConversations(ctx context.Context, userID string) ([]slackapi.Channel, error) {
	var allChannels []slackapi.Channel
	cursor := ""

	for {
		params := &slackapi.GetConversationsForUserParameters{
			UserID: userID,
			Cursor: cursor,
			Limit:  200,
			Types:  []string{"public_channel", "private_channel"},
		}

		channels, nextCursor, err := c.api.GetConversationsForUserContext(ctx, params)
		if err != nil {
			if rateLimitErr, ok := err.(*slackapi.RateLimitedError); ok {
				c.log.Info("Rate limited, waiting %s", rateLimitErr.RetryAfter)
				time.Sleep(rateLimitErr.RetryAfter)
				continue
			}
			return nil, err
		}

		allChannels = append(allChannels, channels...)

		if nextCursor == "" {
			break
		}
		cursor = nextCursor
	}

	return allChannels, nil
}

// revokeFilePublicURL disables the public URL for a file.
func (c *slackClient) revokeFilePublicURL(ctx context.Context, fileID string) error {
	_, err := c.api.RevokeFilePublicURL(fileID)
	return err
}

// kickUserFromConversation removes a user from a channel.
func (c *slackClient) kickUserFromConversation(ctx context.Context, channelID, userID string) error {
	return c.api.KickUserFromConversation(channelID, userID)
}
