package labels

import (
	"context"
	"fmt"
	"strings"

	"gdrive-audit/internal/auth"
	"gdrive-audit/internal/logger"
	"gdrive-audit/pkg/config"

	drive "google.golang.org/api/drive/v3"
)

// Resolver fetches organization labels from the Drive Labels API and
// maps opaque label/field/choice IDs to human-readable titles.
type Resolver struct {
	labelTitles  map[string]string            // labelID → label title
	choiceTitles map[string]map[string]string // labelID → choiceID → choice displayName
	labelIDs     []string                     // all label IDs for IncludeLabels
	logger       logger.Logger
}

// NewResolver creates a Resolver by calling the Drive Labels API to list all
// published labels in the organization. Returns an error if the API is not
// available (e.g. scope not configured).
func NewResolver(ctx context.Context, cfg *config.Config, log logger.Logger) (*Resolver, error) {
	svc, err := auth.NewLabelsService(ctx, cfg.CredentialsPath, cfg.ImpersonateUser)
	if err != nil {
		return nil, fmt.Errorf("failed to create Labels service: %w", err)
	}

	r := &Resolver{
		labelTitles:  make(map[string]string),
		choiceTitles: make(map[string]map[string]string),
		logger:       log,
	}

	// Fetch all published labels with LABEL_VIEW_FULL to get field definitions
	pageToken := ""
	for {
		call := svc.Labels.List().
			PublishedOnly(true).
			View("LABEL_VIEW_FULL").
			PageSize(200).
			Context(ctx)
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		resp, err := call.Do()
		if err != nil {
			return nil, fmt.Errorf("failed to list labels: %w", err)
		}

		for _, label := range resp.Labels {
			r.labelTitles[label.Id] = label.Properties.Title
			r.labelIDs = append(r.labelIDs, label.Id)

			// Build choice ID → display name mapping for selection fields
			choices := make(map[string]string)
			for _, field := range label.Fields {
				if field.SelectionOptions != nil {
					for _, choice := range field.SelectionOptions.Choices {
						if choice.Properties != nil {
							choices[choice.Id] = choice.Properties.DisplayName
						}
					}
				}
			}
			if len(choices) > 0 {
				r.choiceTitles[label.Id] = choices
			}
		}

		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}

	log.Info("Labels resolver: found %d published labels", len(r.labelIDs))
	return r, nil
}

// LabelIDs returns a comma-separated list of all known label IDs,
// suitable for passing to Files.List().IncludeLabels().
func (r *Resolver) LabelIDs() string {
	return strings.Join(r.labelIDs, ",")
}

// Resolve converts raw Drive API labels (from File.LabelInfo) into
// human-readable strings. For selection fields, it resolves choice IDs
// to their display names. Returns nil if no labels are present.
func (r *Resolver) Resolve(fileLabels []*drive.Label) []string {
	if len(fileLabels) == 0 {
		return nil
	}

	var result []string
	for _, label := range fileLabels {
		title := r.labelTitles[label.Id]
		if title == "" {
			title = label.Id // fallback to ID
		}

		// Try to extract selection values from label fields
		var values []string
		for _, field := range label.Fields {
			if field.ValueType == "selection" && len(field.Selection) > 0 {
				for _, choiceID := range field.Selection {
					if choices, ok := r.choiceTitles[label.Id]; ok {
						if name, ok := choices[choiceID]; ok {
							values = append(values, name)
							continue
						}
					}
					values = append(values, choiceID) // fallback
				}
			} else if field.ValueType == "text" && len(field.Text) > 0 {
				values = append(values, field.Text...)
			}
		}

		if len(values) > 0 {
			// Show "LabelTitle: Value1, Value2"
			result = append(result, fmt.Sprintf("%s: %s", title, strings.Join(values, ", ")))
		} else {
			// Label applied but no field values
			result = append(result, title)
		}
	}

	return result
}
