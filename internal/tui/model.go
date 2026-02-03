package tui

import (
	"context"
	"fmt"

	"gdrive-audit/internal/audit"
	"gdrive-audit/internal/logger"
	"gdrive-audit/pkg/models"

	tea "github.com/charmbracelet/bubbletea"
)

// State represents the current TUI state
type State int

const (
	StateScanning State = iota
	StateListing
	StateViewingDetail
)

// Model represents the main TUI model
type Model struct {
	state      State
	progress   *ProgressModel
	list       *ListModel
	detail     *DetailModel
	scanResult *models.ScanResult
	scanning   bool
	scanError  error
	logger     logger.Logger
}

// NewModel creates a new TUI model
func NewModel(log logger.Logger) *Model {
	return &Model{
		state:    StateScanning,
		progress: NewProgressModel(),
		list:     NewListModel(),
		detail:   NewDetailModel(),
		logger:   log,
	}
}

// Init initializes the model
func (m *Model) Init() tea.Cmd {
	return m.progress.Init()
}

// Update handles messages and updates the model
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "esc":
			if m.state == StateViewingDetail {
				m.state = StateListing
				return m, nil
			}
			return m, tea.Quit
		}

	case ScanCompleteMsg:
		m.scanning = false
		m.scanResult = msg.Result
		m.scanError = msg.Err
		if msg.Err == nil {
			m.state = StateListing
			m.list.SetIssues(msg.Result.Issues)
			return m, m.list.Init()
		}
		return m, nil

	case ScanProgressMsg:
		// Convert exported message to internal format
		internalMsg := scanProgressMsg{
			filesScanned: msg.FilesScanned,
			issuesFound:  msg.IssuesFound,
		}
		if m.state == StateScanning {
			updated, cmd := m.progress.Update(internalMsg)
			if p, ok := updated.(*ProgressModel); ok {
				m.progress = p
			}
			return m, cmd
		}
		return m, nil
	case scanProgressMsg:
		if m.state == StateScanning {
			updated, cmd := m.progress.Update(msg)
			if p, ok := updated.(*ProgressModel); ok {
				m.progress = p
			}
			return m, cmd
		}
		return m, nil
	}

	// Route messages to current state
	switch m.state {
	case StateScanning:
		updated, cmd := m.progress.Update(msg)
		if p, ok := updated.(*ProgressModel); ok {
			m.progress = p
		}
		return m, cmd

	case StateListing:
		updated, cmd := m.list.Update(msg)
		if l, ok := updated.(*ListModel); ok {
			m.list = l
			if l.selectedIssue != nil {
				m.state = StateViewingDetail
				m.detail.SetIssue(*l.selectedIssue)
				m.list.selectedIssue = nil // Reset selection
				return m, m.detail.Init()
			}
		}
		return m, cmd

	case StateViewingDetail:
		updated, cmd := m.detail.Update(msg)
		if d, ok := updated.(*DetailModel); ok {
			m.detail = d
			if d.back {
				m.state = StateListing
				m.detail.back = false
				return m, nil
			}
		}
		return m, cmd
	}

	return m, nil
}

// View renders the current state
func (m *Model) View() string {
	if m.scanError != nil {
		return errorView(m.scanError)
	}

	switch m.state {
	case StateScanning:
		return m.progress.View()
	case StateListing:
		return m.list.View()
	case StateViewingDetail:
		return m.detail.View()
	default:
		return "Unknown state\n"
	}
}

// StartScan starts a scan operation (called from CLI)
func (m *Model) StartScan(ctx context.Context, scanner *audit.Scanner, scope string) {
	m.scanning = true
	m.state = StateScanning
	// Scan will be started from CLI via goroutine
}

// ScanCompleteMsg is sent when scan completes
type ScanCompleteMsg struct {
	Result *models.ScanResult
	Err    error
}

// ScanProgressMsg is sent during scan progress (exported so it can be sent from outside package)
type ScanProgressMsg struct {
	FilesScanned int
	IssuesFound  int
}

// scanProgressMsg is the internal message type
type scanProgressMsg struct {
	filesScanned int
	issuesFound  int
}

// errorView renders an error message
func errorView(err error) string {
	return fmt.Sprintf("Error: %v\n\nPress 'q' to quit.", err)
}
