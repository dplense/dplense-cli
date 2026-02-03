package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	progressStyle     = lipgloss.NewStyle().Padding(1, 2)
	progressBarStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("69"))
	progressBarEmpty  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	progressBarFilled = lipgloss.NewStyle().Foreground(lipgloss.Color("69")).Background(lipgloss.Color("69"))
)

// ProgressModel shows scanning progress
type ProgressModel struct {
	filesScanned int
	issuesFound  int
	animationPos int // For animated progress bar
}

// NewProgressModel creates a new progress model
func NewProgressModel() *ProgressModel {
	return &ProgressModel{}
}

// Init initializes the progress model
func (m *ProgressModel) Init() tea.Cmd {
	return tick()
}

// tick returns a command that sends a tick message after a delay
func tick() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(time.Time) tea.Msg {
		return tickMsg{}
	})
}

// tickMsg is sent periodically to animate the spinner
type tickMsg struct{}

// Update handles messages
func (m *ProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case scanProgressMsg:
		m.filesScanned = msg.filesScanned
		m.issuesFound = msg.issuesFound
		return m, tick() // Continue animating
	case tickMsg:
		// Increment animation position for progress bar animation
		m.animationPos++
		return m, tick() // Continue animating
	case tea.KeyMsg:
		return m, nil
	}
	return m, nil
}

// View renders the progress view
func (m *ProgressModel) View() string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString(progressStyle.Render("Scanning Google Drive...\n\n"))

	// Create progress bar
	// Since we don't know total files, we'll show a visual progress bar
	// that grows based on files scanned (scaled to show progress)
	barWidth := 50
	progressBar := m.renderProgressBar(barWidth)

	sb.WriteString("  " + progressBar + "\n\n")
	sb.WriteString(fmt.Sprintf("  Files Scanned: %d\n", m.filesScanned))
	sb.WriteString(fmt.Sprintf("  Issues Found:  %d\n", m.issuesFound))
	sb.WriteString("\n")
	sb.WriteString("  Press 'q' to quit\n")

	return sb.String()
}

// renderProgressBar renders a visual progress bar
func (m *ProgressModel) renderProgressBar(width int) string {
	// Since we don't know total files upfront, we'll create a progress bar
	// that fills based on files scanned with a visual scale that provides
	// good feedback. We'll use a square root scale so it fills faster initially
	// and slows down as more files are scanned.

	var filled int
	if m.filesScanned == 0 {
		filled = 0
	} else {
		// Use a linear scale with a reasonable max estimate
		// This provides good visual feedback as files are scanned
		maxExpected := 50000.0 // Reasonable estimate for most scans

		// Linear progress: files / maxExpected * width
		progress := (float64(m.filesScanned) / maxExpected) * float64(width)

		// Cap at full width
		if progress > float64(width) {
			progress = float64(width)
		}
		filled = int(progress)
	}

	// Add animated effect for visual feedback
	animationOffset := (m.animationPos / 3) % 4 // Animation speed

	var bar strings.Builder
	for i := 0; i < width; i++ {
		if i < filled {
			// Filled portion - solid block
			bar.WriteString(progressBarFilled.Render("█"))
		} else if i == filled && m.filesScanned > 0 {
			// Animated leading edge
			animChars := []string{"█", "▓", "▒", "░"}
			char := animChars[animationOffset%len(animChars)]
			bar.WriteString(progressBarStyle.Render(char))
		} else {
			// Empty portion
			bar.WriteString(progressBarEmpty.Render("░"))
		}
	}

	return bar.String()
}
