package report

import (
	"fmt"
	"io"
)

// ProgressReporter reports progress during scanning
type ProgressReporter struct {
	writer      io.Writer
	filesScanned int
	issuesFound  int
}

// NewProgressReporter creates a new progress reporter
func NewProgressReporter(writer io.Writer) *ProgressReporter {
	return &ProgressReporter{
		writer: writer,
	}
}

// Update reports progress update
func (pr *ProgressReporter) Update(filesScanned, issuesFound int) {
	pr.filesScanned = filesScanned
	pr.issuesFound = issuesFound
	
	// Print every 50 files or on first update
	if filesScanned%50 == 0 || filesScanned == 1 {
		fmt.Fprintf(pr.writer, "\rScanning... %d files processed, %d issues found", filesScanned, issuesFound)
	}
}

// Finish prints final progress
func (pr *ProgressReporter) Finish() {
	// Clear the line and print final summary
	fmt.Fprintf(pr.writer, "\r%-80s\r", "") // Clear line
	fmt.Fprintf(pr.writer, "Scan complete: %d files processed, %d issues found\n", pr.filesScanned, pr.issuesFound)
}
