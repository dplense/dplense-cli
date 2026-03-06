package models

import "time"

// ScanMetadata contains metadata about a scan operation
type ScanMetadata struct {
	Timestamp        time.Time `json:"timestamp"`
	Scope            string    `json:"scope"`
	TotalFilesScanned int      `json:"total_files_scanned"`
	IssuesFound      int      `json:"issues_found"`
	DurationSeconds  float64  `json:"duration_seconds,omitempty"`
}

// FileIssue represents a file with security issues
type FileIssue struct {
	FileID      string       `json:"file_id"`
	FileName    string       `json:"file_name"`
	DriveName   string       `json:"drive_name"`
	DriveID     string       `json:"drive_id,omitempty"`
	OwnerEmail  string       `json:"owner_email"`
	OwnerName   string       `json:"owner_name,omitempty"`
	FolderPath  string       `json:"folder_path"`
	Label       string       `json:"label"`
	WebViewLink string       `json:"web_view_link"`
	Permissions []Permission `json:"permissions"`
}

// ScanResult combines scan metadata with discovered issues
type ScanResult struct {
	Metadata ScanMetadata `json:"scan_metadata"`
	Issues   []FileIssue  `json:"issues"`
}

// NewScanResult creates a new scan result with initialized metadata
func NewScanResult(scope string) *ScanResult {
	return &ScanResult{
		Metadata: ScanMetadata{
			Timestamp:        time.Now(),
			Scope:            scope,
			TotalFilesScanned: 0,
			IssuesFound:      0,
		},
		Issues: make([]FileIssue, 0),
	}
}

// AddIssue adds a file issue to the scan result
func (sr *ScanResult) AddIssue(issue FileIssue) {
	sr.Issues = append(sr.Issues, issue)
	sr.Metadata.IssuesFound = len(sr.Issues)
}

// SetFilesScanned updates the total files scanned count
func (sr *ScanResult) SetFilesScanned(count int) {
	sr.Metadata.TotalFilesScanned = count
}

// SetDuration updates the scan duration
func (sr *ScanResult) SetDuration(duration time.Duration) {
	sr.Metadata.DurationSeconds = duration.Seconds()
}
