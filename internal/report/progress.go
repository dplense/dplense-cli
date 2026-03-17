package report

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/dplense/dplense-cli/pkg/models"

	"github.com/charmbracelet/lipgloss"
)

// spinner frames (Braille dots)
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// ProgressReporter shows animated scan progress using a simple \r-based spinner.
// Writes to a single writer (typically os.Stderr) and never touches terminal state,
// so subsequent output on stdout is unaffected.
type ProgressReporter struct {
	writer    io.Writer
	startTime time.Time

	mu           sync.Mutex
	filesScanned int
	issuesFound  int
	targetIdx    int
	targetTotal  int
	targetName   string

	// ETA tracking
	completedTargets  int
	lastTargetStart   time.Time
	completedDuration time.Duration // total time of completed targets only

	done chan struct{}
}

// NewProgressReporter creates a new progress reporter.
func NewProgressReporter(writer io.Writer) *ProgressReporter {
	return &ProgressReporter{
		writer:    writer,
		startTime: time.Now(),
	}
}

// Start launches the spinner goroutine.
func (pr *ProgressReporter) Start() {
	pr.done = make(chan struct{})

	spinnerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("69"))

	go func() {
		frame := 0
		ticker := time.NewTicker(80 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-pr.done:
				// Clear the spinner line
				fmt.Fprint(pr.writer, "\r\033[K")
				return
			case <-ticker.C:
				pr.mu.Lock()
				spinner := spinnerStyle.Render(spinnerFrames[frame%len(spinnerFrames)])
				line := pr.renderLine(spinner)
				fmt.Fprint(pr.writer, "\r\033[K"+line)
				pr.mu.Unlock()
				frame++
			}
		}
	}()
}

// renderLine builds the status line (must be called under pr.mu).
func (pr *ProgressReporter) renderLine(spinner string) string {
	etaSuffix := pr.etaSuffix()

	if pr.targetTotal > 0 && pr.targetName != "" {
		name := pr.targetName
		if len(name) > 40 {
			name = name[:37] + "..."
		}
		return fmt.Sprintf(" %s Scanning [%d/%d] %s — %s files, %d issues%s",
			spinner, pr.targetIdx, pr.targetTotal, name,
			formatNumber(pr.filesScanned), pr.issuesFound, etaSuffix)
	}
	return fmt.Sprintf(" %s Scanning... %s files, %d issues%s",
		spinner, formatNumber(pr.filesScanned), pr.issuesFound, etaSuffix)
}

// etaSuffix returns " — ~Xm Ys left" or "" (must be called under pr.mu).
func (pr *ProgressReporter) etaSuffix() string {
	if pr.completedTargets == 0 || pr.targetTotal == 0 {
		return ""
	}
	// Average based only on completed targets (excludes current in-progress target)
	avgPerTarget := pr.completedDuration / time.Duration(pr.completedTargets)
	remaining := pr.targetTotal - pr.completedTargets
	eta := avgPerTarget * time.Duration(remaining)
	if eta > time.Second {
		return fmt.Sprintf(" — ~%s left", formatETA(eta))
	}
	return ""
}

// SetTarget updates the current target being scanned.
func (pr *ProgressReporter) SetTarget(idx, total int, name string) {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	// If this is not the first target, the previous one just completed
	if pr.targetIdx > 0 && idx > pr.targetIdx {
		pr.completedTargets++
		pr.completedDuration += time.Since(pr.lastTargetStart)
	}
	pr.lastTargetStart = time.Now()
	pr.targetIdx = idx
	pr.targetTotal = total
	pr.targetName = name
}

// Update reports a progress update from the scanner.
func (pr *ProgressReporter) Update(filesScanned, issuesFound int) {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	pr.filesScanned = filesScanned
	pr.issuesFound = issuesFound
}

// Stop stops the spinner. The scan summary is displayed by the table's summary box.
func (pr *ProgressReporter) Stop() {
	if pr.done != nil {
		close(pr.done)
		// Small delay to let the goroutine clear the line
		time.Sleep(10 * time.Millisecond)
	}
}

// PrintStatus prints a plain status message, clearing the spinner line first.
func (pr *ProgressReporter) PrintStatus(msg string) {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	fmt.Fprint(pr.writer, "\r\033[K")
	fmt.Fprintln(pr.writer, msg)
}

// LogWriter returns an io.Writer that safely interleaves log output with the
// spinner. When the spinner is active, each write clears the spinner line first,
// prints the log message, and lets the next spinner tick redraw the status line.
func (pr *ProgressReporter) LogWriter() io.Writer {
	return &spinnerAwareWriter{pr: pr}
}

type spinnerAwareWriter struct {
	pr *ProgressReporter
}

func (w *spinnerAwareWriter) Write(p []byte) (int, error) {
	w.pr.mu.Lock()
	defer w.pr.mu.Unlock()

	// Clear the spinner line, print the log message, spinner redraws on next tick
	fmt.Fprint(w.pr.writer, "\r\033[K")
	return w.pr.writer.Write(p)
}

// riskOrder returns a numeric order for risk levels (higher = more severe).
func riskOrder(r models.RiskLevel) int {
	switch r {
	case models.RiskCritical:
		return 4
	case models.RiskHigh:
		return 3
	case models.RiskMedium:
		return 2
	case models.RiskLow:
		return 1
	default:
		return 0
	}
}

// formatNumber formats an integer with thousands separators: 4700 → "4,700".
func formatNumber(n int) string {
	if n < 0 {
		return "-" + formatNumber(-n)
	}
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}

	// Build from right to left, inserting commas every 3 digits
	s := fmt.Sprintf("%d", n)
	result := make([]byte, 0, len(s)+len(s)/3)
	for i, ch := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(ch))
	}
	return string(result)
}

// formatETA formats ETA duration in a compact human-friendly way: "4m 40s", "1h 5m", "45s".
func formatETA(d time.Duration) string {
	d = d.Round(time.Second)
	if d < time.Second {
		return "<1s"
	}

	totalSeconds := int(d.Seconds())
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60

	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}

// formatDuration formats a duration in a human-friendly way.
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm%02ds", minutes, seconds)
}
