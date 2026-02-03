package report

import (
	"encoding/json"
	"fmt"
	"io"

	"gdrive-audit/pkg/models"
)

// WriteJSON writes scan results as JSON to the writer
func WriteJSON(result *models.ScanResult, writer io.Writer) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ") // Pretty print for readability
	if err := encoder.Encode(result); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}
	return nil
}

// WriteJSONStrict writes scan results as compact JSON (no indentation) for headless/CI mode
func WriteJSONStrict(result *models.ScanResult, writer io.Writer) error {
	encoder := json.NewEncoder(writer)
	if err := encoder.Encode(result); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}
	return nil
}
