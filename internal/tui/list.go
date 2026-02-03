package tui

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"gdrive-audit/pkg/models"
)

var (
	listStyle       = lipgloss.NewStyle().Padding(1, 2)
	headerStyle     = lipgloss.NewStyle().Bold(true).Underline(true)
	selectedStyle   = lipgloss.NewStyle().Background(lipgloss.Color("238"))
	criticalStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
	highStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("208"))
	mediumStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	lowStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
)

// SortColumn represents the column to sort by
type SortColumn int

const (
	SortByRisk SortColumn = iota
	SortByName
	SortByOwner
)

// ListModel displays the list of issues
type ListModel struct {
	issues        []models.FileIssue
	filteredIssues []models.FileIssue
	selectedIndex int
	selectedIssue *models.FileIssue
	sortColumn    SortColumn
	sortAscending bool
	filterRisk    string // "" = all, "critical", "high", etc.
	searchQuery   string
	pageSize      int
	currentPage   int
}

// NewListModel creates a new list model
func NewListModel() *ListModel {
	return &ListModel{
		issues:        make([]models.FileIssue, 0),
		filteredIssues: make([]models.FileIssue, 0),
		pageSize:      20,
		currentPage:   0,
		sortColumn:    SortByRisk,
		sortAscending: false,
	}
}

// SetIssues sets the issues to display
func (m *ListModel) SetIssues(issues []models.FileIssue) {
	m.issues = issues
	m.applyFilters()
}

// Init initializes the list model
func (m *ListModel) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m *ListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.selectedIndex > 0 {
				m.selectedIndex--
			}
			return m, nil

		case "down", "j":
			if m.selectedIndex < len(m.filteredIssues)-1 {
				m.selectedIndex++
			}
			return m, nil

		case "enter", " ":
			if m.selectedIndex < len(m.filteredIssues) {
				issue := m.filteredIssues[m.selectedIndex]
				m.selectedIssue = &issue
			}
			return m, nil

		case "r":
			// Toggle risk filter
			risks := []string{"", "critical", "high", "medium", "low"}
			currentIdx := 0
			for i, r := range risks {
				if r == m.filterRisk {
					currentIdx = i
					break
				}
			}
			m.filterRisk = risks[(currentIdx+1)%len(risks)]
			m.applyFilters()
			m.selectedIndex = 0
			return m, nil

		case "s":
			// Toggle sort column
			m.sortColumn = (m.sortColumn + 1) % 3
			m.applyFilters()
			return m, nil

		case "S":
			// Toggle sort direction
			m.sortAscending = !m.sortAscending
			m.applyFilters()
			return m, nil

		case "/":
			// Start search (simplified - just clear filter for now)
			m.searchQuery = ""
			return m, nil

		case "backspace":
			if len(m.searchQuery) > 0 {
				m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
				m.applyFilters()
			}
			return m, nil

		default:
			// Handle typing for search
			if len(msg.String()) == 1 && msg.String() != "/" {
				m.searchQuery += msg.String()
				m.applyFilters()
				return m, nil
			}
		}
	}
	return m, nil
}

// applyFilters applies current filters and sorting
func (m *ListModel) applyFilters() {
	m.filteredIssues = make([]models.FileIssue, 0)

	// Apply risk filter
	for _, issue := range m.issues {
		if m.filterRisk != "" {
			hasRisk := false
			for _, perm := range issue.Permissions {
				if string(perm.RiskLevel) == m.filterRisk {
					hasRisk = true
					break
				}
			}
			if !hasRisk {
				continue
			}
		}

		// Apply search filter
		if m.searchQuery != "" {
			query := strings.ToLower(m.searchQuery)
			if !strings.Contains(strings.ToLower(issue.FileName), query) &&
				!strings.Contains(strings.ToLower(issue.OwnerEmail), query) {
				continue
			}
		}

		m.filteredIssues = append(m.filteredIssues, issue)
	}

	// Apply sorting
	sort.Slice(m.filteredIssues, func(i, j int) bool {
		switch m.sortColumn {
		case SortByRisk:
			riskI := getHighestRisk(m.filteredIssues[i])
			riskJ := getHighestRisk(m.filteredIssues[j])
			if m.sortAscending {
				return riskI < riskJ
			}
			return riskI > riskJ

		case SortByName:
			if m.sortAscending {
				return m.filteredIssues[i].FileName < m.filteredIssues[j].FileName
			}
			return m.filteredIssues[i].FileName > m.filteredIssues[j].FileName

		case SortByOwner:
			if m.sortAscending {
				return m.filteredIssues[i].OwnerEmail < m.filteredIssues[j].OwnerEmail
			}
			return m.filteredIssues[i].OwnerEmail > m.filteredIssues[j].OwnerEmail

		default:
			return false
		}
	})

	// Reset selection if out of bounds
	if m.selectedIndex >= len(m.filteredIssues) {
		m.selectedIndex = len(m.filteredIssues) - 1
		if m.selectedIndex < 0 {
			m.selectedIndex = 0
		}
	}
}

// getHighestRisk returns the highest risk level as an integer
func getHighestRisk(issue models.FileIssue) int {
	maxRisk := 0
	for _, perm := range issue.Permissions {
		risk := 0
		switch perm.RiskLevel {
		case models.RiskCritical:
			risk = 4
		case models.RiskHigh:
			risk = 3
		case models.RiskMedium:
			risk = 2
		case models.RiskLow:
			risk = 1
		}
		if risk > maxRisk {
			maxRisk = risk
		}
	}
	return maxRisk
}

// View renders the list view
func (m *ListModel) View() string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString(listStyle.Render("📋 Security Issues\n\n"))

	// Header
	header := fmt.Sprintf("%-5s %-40s %-25s %-20s %-10s",
		"#", "FILE NAME", "DRIVE", "OWNER", "RISK")
	sb.WriteString(headerStyle.Render(header))
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("-", 100))
	sb.WriteString("\n")

	// Display filtered issues
	start := m.currentPage * m.pageSize
	end := start + m.pageSize
	if end > len(m.filteredIssues) {
		end = len(m.filteredIssues)
	}

	if len(m.filteredIssues) == 0 {
		sb.WriteString("\n  No issues found.\n")
	} else {
		for i := start; i < end; i++ {
			issue := m.filteredIssues[i]
			riskLevel := formatRiskLevel(getHighestRiskLevel(issue))
			fileName := truncateString(issue.FileName, 38)
			driveName := truncateString(issue.DriveName, 23)
			ownerName := truncateString(issue.OwnerName, 18)

			line := fmt.Sprintf("%-5d %-40s %-25s %-20s %s",
				i+1, fileName, driveName, ownerName, riskLevel)

			if i == m.selectedIndex {
				sb.WriteString(selectedStyle.Render(line))
			} else {
				sb.WriteString(line)
			}
			sb.WriteString("\n")
		}
	}

	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("Showing %d of %d issues", len(m.filteredIssues), len(m.issues)))
	if m.filterRisk != "" {
		sb.WriteString(fmt.Sprintf(" (filtered: %s)", m.filterRisk))
	}
	sb.WriteString("\n\n")

	// Help text
	sb.WriteString("Controls: ↑/↓ Navigate | Enter View Details | r Filter Risk | s Sort | / Search | q Quit\n")

	return sb.String()
}

// getHighestRiskLevel returns the highest risk level string
func getHighestRiskLevel(issue models.FileIssue) models.RiskLevel {
	maxRisk := models.RiskLow
	for _, perm := range issue.Permissions {
		if perm.RiskLevel == models.RiskCritical {
			return models.RiskCritical
		}
		if perm.RiskLevel == models.RiskHigh && maxRisk != models.RiskCritical {
			maxRisk = models.RiskHigh
		}
		if perm.RiskLevel == models.RiskMedium && maxRisk == models.RiskLow {
			maxRisk = models.RiskMedium
		}
	}
	return maxRisk
}

// formatRiskLevel formats risk level with color
func formatRiskLevel(risk models.RiskLevel) string {
	riskStr := string(risk)
	switch risk {
	case models.RiskCritical:
		return criticalStyle.Render(riskStr)
	case models.RiskHigh:
		return highStyle.Render(riskStr)
	case models.RiskMedium:
		return mediumStyle.Render(riskStr)
	case models.RiskLow:
		return lowStyle.Render(riskStr)
	default:
		return riskStr
	}
}

// truncateString truncates a string
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return "..."
	}
	return s[:maxLen-3] + "..."
}
