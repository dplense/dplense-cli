package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"gdrive-audit/pkg/models"
)

var (
	detailStyle = lipgloss.NewStyle().Padding(1, 2)
	titleStyle  = lipgloss.NewStyle().Bold(true).Underline(true)
)

// DetailModel shows detailed information about a file issue
type DetailModel struct {
	issue models.FileIssue
	back  bool
}

// NewDetailModel creates a new detail model
func NewDetailModel() *DetailModel {
	return &DetailModel{}
}

// SetIssue sets the issue to display
func (m *DetailModel) SetIssue(issue models.FileIssue) {
	m.issue = issue
}

// Init initializes the detail model
func (m *DetailModel) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m *DetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "b":
			m.back = true
			return m, nil
		case "q":
			return m, tea.Quit
		}
	}
	return m, nil
}

// View renders the detail view
func (m *DetailModel) View() string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString(detailStyle.Render("📄 File Details\n\n"))

	// File information
	sb.WriteString(titleStyle.Render("File Information"))
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("  Name:     %s\n", m.issue.FileName))
	sb.WriteString(fmt.Sprintf("  ID:       %s\n", m.issue.FileID))
	sb.WriteString(fmt.Sprintf("  Drive:    %s\n", m.issue.DriveName))
	sb.WriteString(fmt.Sprintf("  Owner:    %s (%s)\n", m.issue.OwnerName, m.issue.OwnerEmail))
	sb.WriteString(fmt.Sprintf("  Folder:   %s\n", m.issue.FolderPath))
	if m.issue.WebViewLink != "" {
		sb.WriteString(fmt.Sprintf("  Link:     %s\n", m.issue.WebViewLink))
	}
	sb.WriteString("\n")

	// Permissions
	sb.WriteString(titleStyle.Render("Permissions"))
	sb.WriteString("\n")
	if len(m.issue.Permissions) == 0 {
		sb.WriteString("  No permissions found.\n")
	} else {
		for i, perm := range m.issue.Permissions {
			riskLevel := formatRiskLevel(perm.RiskLevel)
			sharedWith := getSharedWith(perm)
			sb.WriteString(fmt.Sprintf("  [%d] %s - %s (%s) - %s\n",
				i+1, sharedWith, perm.Role, perm.Type, riskLevel))
		}
	}

	sb.WriteString("\n")
	sb.WriteString("Controls: Esc/Backspace Back | q Quit\n")

	return sb.String()
}

// getSharedWith returns the email or domain for a permission
func getSharedWith(perm models.Permission) string {
	if perm.Email != "" {
		return perm.Email
	}
	if perm.Domain != "" {
		return "@" + perm.Domain
	}
	if perm.Type == "anyone" {
		return "Anyone with link"
	}
	return "-"
}
