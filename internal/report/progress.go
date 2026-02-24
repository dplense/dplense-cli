package report

import (
	"fmt"
	"io"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Messages sent to the Bubble Tea program from scanner callbacks.

type progressMsg struct {
	filesScanned int
	issuesFound  int
}

type targetMsg struct {
	idx   int
	total int
	name  string
}

type doneMsg struct{}

// progressModel is a Bubble Tea model for animated scan progress.
// The view stays empty until the first targetMsg or progressMsg arrives,
// so pre-scan logger messages ("Identifying all shared drives...") don't
// get interleaved with the spinner output.
type progressModel struct {
	spinner      spinner.Model
	filesScanned int
	issuesFound  int
	targetIdx    int
	targetTotal  int
	targetName   string
	started      bool // true after first data arrives
	done         bool
}

func newProgressModel() progressModel {
	s := spinner.New(spinner.WithSpinner(spinner.Dot))
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("69"))
	return progressModel{
		spinner: s,
	}
}

func (m progressModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m progressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case progressMsg:
		m.started = true
		m.filesScanned = msg.filesScanned
		m.issuesFound = msg.issuesFound
		return m, nil

	case targetMsg:
		m.started = true
		m.targetIdx = msg.idx
		m.targetTotal = msg.total
		m.targetName = msg.name
		return m, nil

	case doneMsg:
		m.done = true
		return m, tea.Quit

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}
		return m, nil
	}

	return m, nil
}

func (m progressModel) View() string {
	if m.done || !m.started {
		return ""
	}

	spinnerView := m.spinner.View()

	if m.targetTotal > 0 && m.targetName != "" {
		name := m.targetName
		if len(name) > 40 {
			name = name[:37] + "..."
		}
		return fmt.Sprintf(" %s Scanning [%d/%d] %s — %d files, %d issues\n",
			spinnerView, m.targetIdx, m.targetTotal, name, m.filesScanned, m.issuesFound)
	}
	return fmt.Sprintf(" %s Scanning... %d files, %d issues\n",
		spinnerView, m.filesScanned, m.issuesFound)
}

// ProgressReporter wraps a Bubble Tea program to show animated scan progress.
type ProgressReporter struct {
	program      *tea.Program
	writer       io.Writer
	startTime    time.Time
	filesScanned int
	issuesFound  int
}

// NewProgressReporter creates a new progress reporter.
func NewProgressReporter(writer io.Writer) *ProgressReporter {
	return &ProgressReporter{
		writer:    writer,
		startTime: time.Now(),
	}
}

// Start launches the Bubble Tea program in a background goroutine.
func (pr *ProgressReporter) Start() {
	model := newProgressModel()
	pr.program = tea.NewProgram(model, tea.WithOutput(pr.writer))
	go func() {
		_, _ = pr.program.Run()
	}()
}

// SetTarget updates the current target being scanned.
func (pr *ProgressReporter) SetTarget(idx, total int, name string) {
	if pr.program != nil {
		pr.program.Send(targetMsg{idx: idx, total: total, name: name})
	}
}

// Update reports a progress update from the scanner.
func (pr *ProgressReporter) Update(filesScanned, issuesFound int) {
	pr.filesScanned = filesScanned
	pr.issuesFound = issuesFound
	if pr.program != nil {
		pr.program.Send(progressMsg{filesScanned: filesScanned, issuesFound: issuesFound})
	}
}

// Finish stops the Bubble Tea program and prints the final summary.
func (pr *ProgressReporter) Finish() {
	if pr.program != nil {
		pr.program.Send(doneMsg{})
		pr.program.Wait()
	}

	duration := time.Since(pr.startTime)
	fmt.Fprintf(pr.writer, "Scan complete: %d files scanned, %d issues found in %s\n",
		pr.filesScanned, pr.issuesFound, formatDuration(duration))
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
