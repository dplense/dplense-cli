package models

import (
	"sync"
	"time"
)

// ScanMetadata contains metadata about a scan operation
type ScanMetadata struct {
	Provider          string    `json:"provider"`
	Timestamp         time.Time `json:"timestamp"`
	Scope             string    `json:"scope"`
	TotalFilesScanned int       `json:"total_files_scanned"`
	IssuesFound       int       `json:"issues_found"`
	DurationSeconds   float64   `json:"duration_seconds,omitempty"`
	Errors            []string  `json:"errors,omitempty"`
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

// ScanResult combines scan metadata with discovered issues.
// All mutation methods are safe for concurrent use.
type ScanResult struct {
	mu       sync.Mutex   `json:"-"`
	Metadata ScanMetadata `json:"scan_metadata"`
	Issues   []FileIssue  `json:"issues"`
}

// NewScanResult creates a new scan result with initialized metadata
func NewScanResult(scope string) *ScanResult {
	return &ScanResult{
		Metadata: ScanMetadata{
			Timestamp:         time.Now(),
			Scope:             scope,
			TotalFilesScanned: 0,
			IssuesFound:       0,
		},
		Issues: make([]FileIssue, 0),
	}
}

// AddIssue adds a file issue to the scan result
func (sr *ScanResult) AddIssue(issue FileIssue) {
	sr.mu.Lock()
	sr.Issues = append(sr.Issues, issue)
	sr.Metadata.IssuesFound = len(sr.Issues)
	sr.mu.Unlock()
}

// SetFilesScanned updates the total files scanned count
func (sr *ScanResult) SetFilesScanned(count int) {
	sr.mu.Lock()
	sr.Metadata.TotalFilesScanned = count
	sr.mu.Unlock()
}

// SetDuration updates the scan duration
func (sr *ScanResult) SetDuration(duration time.Duration) {
	sr.mu.Lock()
	sr.Metadata.DurationSeconds = duration.Seconds()
	sr.mu.Unlock()
}

// AddError records a partial failure that did not abort the scan.
func (sr *ScanResult) AddError(msg string) {
	sr.mu.Lock()
	sr.Metadata.Errors = append(sr.Metadata.Errors, msg)
	sr.mu.Unlock()
}

// HasErrors returns true if any partial failures were recorded.
func (sr *ScanResult) HasErrors() bool {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	return len(sr.Metadata.Errors) > 0
}
