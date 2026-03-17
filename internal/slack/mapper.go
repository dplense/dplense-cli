package slack

import (
	"fmt"

	"github.com/dplense/dplense-cli/pkg/models"

	slackapi "github.com/slack-go/slack"
)

// mapPublicFile creates a FileIssue for a file with a public URL.
func mapPublicFile(file slackapi.File) models.FileIssue {
	issue := models.FileIssue{
		FileID:      file.ID,
		FileName:    file.Name,
		WebViewLink: file.Permalink,
		OwnerEmail:  file.User,
		DriveName:   "Slack Files",
	}

	perm := models.Permission{
		ID:         fmt.Sprintf("public-%s", file.ID),
		Type:       "anyone",
		Role:       "reader",
		RiskLevel:  models.RiskCritical,
		IsInternal: false,
	}
	issue.Permissions = append(issue.Permissions, perm)

	return issue
}

// mapFileInExtChannel creates a FileIssue for a file shared in an externally-shared channel.
func mapFileInExtChannel(file slackapi.File, channelID, channelName string) models.FileIssue {
	issue := models.FileIssue{
		FileID:      file.ID,
		FileName:    file.Name,
		WebViewLink: file.Permalink,
		OwnerEmail:  file.User,
		DriveName:   "Slack Files",
		FolderPath:  fmt.Sprintf("#%s", channelName),
	}

	perm := models.Permission{
		ID:         fmt.Sprintf("extchan-%s-%s", channelID, file.ID),
		Type:       "group",
		Domain:     "external-channel",
		Role:       "reader",
		RiskLevel:  models.RiskHigh,
		IsInternal: false,
	}
	issue.Permissions = append(issue.Permissions, perm)

	return issue
}

// mapExternalChannel creates a FileIssue for a shared channel (treating channel as a resource).
func mapExternalChannel(channel slackapi.Channel) models.FileIssue {
	issue := models.FileIssue{
		FileID:      channel.ID,
		FileName:    fmt.Sprintf("#%s", channel.Name),
		DriveName:   "Slack Channels",
		WebViewLink: fmt.Sprintf("slack://channel?id=%s", channel.ID),
	}

	perm := models.Permission{
		ID:         fmt.Sprintf("shared-%s", channel.ID),
		Type:       "domain",
		Domain:     "external-organization",
		Role:       "reader",
		RiskLevel:  models.RiskHigh,
		IsInternal: false,
	}
	issue.Permissions = append(issue.Permissions, perm)

	return issue
}

// mapGuestUser creates a FileIssue per channel the guest has access to.
func mapGuestUser(user slackapi.User, channel slackapi.Channel) models.FileIssue {
	guestType := "multi-channel"
	riskLevel := models.RiskHigh
	if user.IsUltraRestricted {
		guestType = "single-channel"
		riskLevel = models.RiskMedium
	}

	issue := models.FileIssue{
		FileID:      fmt.Sprintf("guest-%s-%s", user.ID, channel.ID),
		FileName:    fmt.Sprintf("#%s", channel.Name),
		DriveName:   "Slack Guest Access",
		OwnerEmail:  user.Profile.Email,
		OwnerName:   user.RealName,
		WebViewLink: fmt.Sprintf("slack://channel?id=%s", channel.ID),
	}

	perm := models.Permission{
		ID:         fmt.Sprintf("guest-%s", user.ID),
		Type:       "user",
		Email:      user.Profile.Email,
		Role:       guestType,
		RiskLevel:  riskLevel,
		IsInternal: false,
	}
	issue.Permissions = append(issue.Permissions, perm)

	return issue
}
