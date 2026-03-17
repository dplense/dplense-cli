package report

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/dplense/dplense-cli/pkg/models"
	"github.com/dplense/dplense-cli/pkg/provider"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── styles ──────────────────────────────────────────────────────────────────

var (
	tuiTitle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("69"))
	tuiDim       = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	tuiSubtle    = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	tuiSelected  = lipgloss.NewStyle().Bold(true)
	tuiCursorRow = lipgloss.NewStyle().Background(lipgloss.Color("236"))
	tuiHelp      = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	tuiSuccess   = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	tuiErrorSt   = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
	tuiLabel     = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	tuiKey       = lipgloss.NewStyle().Foreground(lipgloss.Color("69")).Bold(true)
	tuiSection   = lipgloss.NewStyle().Foreground(lipgloss.Color("69")).Bold(true).MarginTop(1)

	tuiHeaderBar = lipgloss.NewStyle().
			Background(lipgloss.Color("235")).
			Foreground(lipgloss.Color("252")).
			Padding(0, 1)

	tuiMetaBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("238")).
			Padding(0, 1)

	tuiConfirmBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("9")).
			Padding(0, 1).
			MarginTop(1)

	tuiScrollThumb = lipgloss.NewStyle().Foreground(lipgloss.Color("69")).Render("┃")
	tuiScrollTrack = lipgloss.NewStyle().Foreground(lipgloss.Color("236")).Render("│")
)

// ── data types ──────────────────────────────────────────────────────────────

type fileEntry struct {
	issue    models.FileIssue
	maxRisk  models.RiskLevel
	extPerms []models.Permission
	intPerms []models.Permission
}

// ── bubbletea model ─────────────────────────────────────────────────────────

type tuiModel struct {
	entries  []fileEntry
	filtered []int // indices into entries matching current filter
	provider provider.Provider
	ctx      context.Context

	// risk totals for header
	riskCounts map[models.RiskLevel]int

	// view state
	view string // "list" | "detail"

	// list view
	cursor    int
	offset    int
	filter    string
	filtering bool

	// detail view
	selected *fileEntry
	permIdx  int

	// revoke
	confirming bool
	statusMsg  string

	// terminal
	width  int
	height int
}

// RunTUI launches the interactive TUI for browsing scan results.
func RunTUI(result *models.ScanResult, p provider.Provider) error {
	entries := buildEntries(result)
	if len(entries) == 0 {
		fmt.Println("No security issues found.")
		return nil
	}

	// Pre-compute risk totals
	rc := map[models.RiskLevel]int{}
	for _, e := range entries {
		rc[e.maxRisk]++
	}

	m := tuiModel{
		entries:    entries,
		provider:   p,
		ctx:        context.Background(),
		view:       "list",
		riskCounts: rc,
	}
	m.applyFilter()

	prog := tea.NewProgram(m, tea.WithAltScreen())
	_, err := prog.Run()
	return err
}

func buildEntries(result *models.ScanResult) []fileEntry {
	entries := make([]fileEntry, 0, len(result.Issues))
	for _, issue := range result.Issues {
		var ext, internal []models.Permission
		maxRisk := models.RiskLevel("")
		for _, p := range issue.Permissions {
			if p.IsInternal {
				internal = append(internal, p)
			} else {
				ext = append(ext, p)
				if maxRisk == "" || riskOrder(p.RiskLevel) > riskOrder(maxRisk) {
					maxRisk = p.RiskLevel
				}
			}
		}
		if len(ext) == 0 {
			continue
		}
		entries = append(entries, fileEntry{
			issue:    issue,
			maxRisk:  maxRisk,
			extPerms: ext,
			intPerms: internal,
		})
	}
	return entries
}

// ── Init / Update / View ────────────────────────────────────────────────────

func (m tuiModel) Init() tea.Cmd {
	return nil
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if m.confirming {
			return m.handleConfirm(msg)
		}
		if m.filtering {
			return m.handleFilterInput(msg)
		}
		if m.view == "detail" {
			return m.handleDetail(msg)
		}
		return m.handleList(msg)
	}
	return m, nil
}

func (m tuiModel) View() string {
	if m.width == 0 {
		return ""
	}
	if m.view == "detail" {
		return m.viewDetail()
	}
	return m.viewList()
}

// ── list view ───────────────────────────────────────────────────────────────

func (m tuiModel) handleList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
			if m.cursor < m.offset {
				m.offset = m.cursor
			}
		}
	case "down", "j":
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
			if m.cursor >= m.offset+m.listHeight() {
				m.offset = m.cursor - m.listHeight() + 1
			}
		}
	case "pgup":
		m.cursor -= m.listHeight()
		if m.cursor < 0 {
			m.cursor = 0
		}
		m.offset = m.cursor
	case "pgdown":
		m.cursor += m.listHeight()
		if m.cursor >= len(m.filtered) {
			m.cursor = len(m.filtered) - 1
		}
		if m.cursor >= m.offset+m.listHeight() {
			m.offset = m.cursor - m.listHeight() + 1
		}
	case "enter":
		if len(m.filtered) > 0 {
			entry := &m.entries[m.filtered[m.cursor]]
			m.selected = entry
			m.permIdx = 0
			m.view = "detail"
			m.statusMsg = ""
		}
	case "/":
		m.filtering = true
		m.statusMsg = ""
	case "home", "g":
		m.cursor = 0
		m.offset = 0
	case "end", "G":
		if len(m.filtered) > 0 {
			m.cursor = len(m.filtered) - 1
			if m.cursor >= m.listHeight() {
				m.offset = m.cursor - m.listHeight() + 1
			}
		}
	}
	return m, nil
}

func (m tuiModel) handleFilterInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter", "esc":
		m.filtering = false
		if msg.String() == "esc" {
			m.filter = ""
			m.applyFilter()
		}
	case "backspace":
		if len(m.filter) > 0 {
			m.filter = m.filter[:len(m.filter)-1]
			m.applyFilter()
		}
	default:
		if len(msg.String()) == 1 {
			m.filter += msg.String()
			m.applyFilter()
		}
	}
	return m, nil
}

func (m *tuiModel) applyFilter() {
	m.filtered = m.filtered[:0]
	f := strings.ToLower(m.filter)
	for i, e := range m.entries {
		if f == "" || strings.Contains(strings.ToLower(e.issue.FileName), f) ||
			strings.Contains(strings.ToLower(e.issue.DriveName), f) {
			m.filtered = append(m.filtered, i)
		}
	}
	m.cursor = 0
	m.offset = 0
}

func (m tuiModel) listHeight() int {
	h := m.height - 5 // header(2) + footer(2) + padding(1)
	if m.filtering || m.filter != "" {
		h--
	}
	if h < 1 {
		h = 1
	}
	return h
}

func (m tuiModel) viewList() string {
	var b strings.Builder
	w := m.width

	// ── Header bar ──
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("69")).Render("dplense")
	pos := fmt.Sprintf("%d/%d", m.cursor+1, len(m.filtered))
	riskSummary := m.renderRiskSummary()

	headerLeft := fmt.Sprintf(" %s  %s", title, riskSummary)
	headerRight := fmt.Sprintf("%s ", pos)

	gap := w - lipgloss.Width(headerLeft) - lipgloss.Width(headerRight)
	if gap < 0 {
		gap = 0
	}
	headerLine := tuiHeaderBar.Width(w).Render(headerLeft + strings.Repeat(" ", gap) + headerRight)
	b.WriteString(headerLine + "\n")

	// ── Filter bar ──
	if m.filtering {
		searchIcon := tuiKey.Render("/")
		b.WriteString(fmt.Sprintf(" %s %s█\n", searchIcon, m.filter))
	} else if m.filter != "" {
		b.WriteString(fmt.Sprintf(" %s %s  %s\n", tuiKey.Render("/"), tuiDim.Render(m.filter), tuiSubtle.Render("(Esc clear)")))
	} else {
		b.WriteString("\n")
	}

	// ── List items ──
	visible := m.listHeight()
	end := m.offset + visible
	if end > len(m.filtered) {
		end = len(m.filtered)
	}

	nameW := 38
	driveW := 22
	if w > 110 {
		nameW = w - 55
		if nameW > 65 {
			nameW = 65
		}
	}

	totalItems := len(m.filtered)
	for i := m.offset; i < end; i++ {
		entry := m.entries[m.filtered[i]]
		name := truncateString(entry.issue.FileName, nameW)
		drive := truncateString(entry.issue.DriveName, driveW)
		risk := riskBadge(entry.maxRisk)
		extCount := tuiDim.Render(fmt.Sprintf("(%d)", len(entry.extPerms)))

		// Scrollbar
		scrollChar := m.scrollbar(i-m.offset, visible, totalItems)

		if i == m.cursor {
			row := fmt.Sprintf(" ▸ %-*s  %-*s  %s %s", nameW, name, driveW, drive, risk, extCount)
			b.WriteString(tuiCursorRow.Width(w - 1).Render(row) + scrollChar + "\n")
		} else {
			row := fmt.Sprintf("   %-*s  %-*s  %s %s", nameW, name, driveW, drive, risk, extCount)
			b.WriteString(row + strings.Repeat(" ", max(0, w-lipgloss.Width(row)-1)) + scrollChar + "\n")
		}
	}

	// Pad
	for i := end - m.offset; i < visible; i++ {
		b.WriteString(strings.Repeat(" ", w-1) + m.scrollbar(i, visible, totalItems) + "\n")
	}

	// ── Footer ──
	b.WriteString("\n")
	if m.filtering {
		b.WriteString(tuiHelp.Render(" Type to search  ") + tuiKey.Render("Enter") + tuiHelp.Render(" apply  ") + tuiKey.Render("Esc") + tuiHelp.Render(" clear"))
	} else {
		footer := keyHint("↑↓", "navigate") + "  " + keyHint("Enter", "open") + "  " + keyHint("/", "search") + "  " + keyHint("q", "quit")
		if m.statusMsg != "" {
			footer = " " + m.statusMsg + "    " + footer
		}
		b.WriteString(footer)
	}

	return b.String()
}

func (m tuiModel) renderRiskSummary() string {
	parts := []string{}
	dot := tuiDim.Render(" · ")
	for _, level := range []models.RiskLevel{models.RiskCritical, models.RiskHigh, models.RiskMedium, models.RiskLow} {
		c := m.riskCounts[level]
		if c == 0 {
			continue
		}
		s := fmt.Sprintf("%d %s", c, level)
		switch level {
		case models.RiskCritical:
			parts = append(parts, criticalStyle.Render(s))
		case models.RiskHigh:
			parts = append(parts, highStyle.Render(s))
		case models.RiskMedium:
			parts = append(parts, mediumStyle.Render(s))
		case models.RiskLow:
			parts = append(parts, lowStyle.Render(s))
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, dot)
}

func (m tuiModel) scrollbar(row, visible, total int) string {
	if total <= visible {
		return " "
	}
	thumbSize := max(1, visible*visible/total)
	thumbStart := m.offset * visible / total
	if row >= thumbStart && row < thumbStart+thumbSize {
		return tuiScrollThumb
	}
	return tuiScrollTrack
}

// ── detail view ─────────────────────────────────────────────────────────────

func (m tuiModel) handleDetail(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "backspace":
		m.view = "list"
		m.statusMsg = ""
	case "q", "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		if m.permIdx > 0 {
			m.permIdx--
		}
	case "down", "j":
		if m.selected != nil && m.permIdx < len(m.selected.extPerms)-1 {
			m.permIdx++
		}
	case "r":
		if m.selected != nil && len(m.selected.extPerms) > 0 {
			m.confirming = true
			m.statusMsg = ""
		}
	case "o":
		if m.selected != nil && m.selected.issue.WebViewLink != "" {
			openBrowser(m.selected.issue.WebViewLink)
			m.statusMsg = tuiDim.Render("Opening in browser...")
		}
	}
	return m, nil
}

func (m tuiModel) handleConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		m.confirming = false
		if m.selected != nil && m.permIdx < len(m.selected.extPerms) {
			perm := m.selected.extPerms[m.permIdx]
			identifier := perm.Email
			if perm.Type == "anyone" {
				identifier = "anyone"
			}

			summary, err := m.provider.RevokePermission(
				m.ctx, m.selected.issue.FileID, identifier, false,
			)
			if err != nil {
				m.statusMsg = tuiErrorSt.Render("Error: " + sanitizeError(err, m.width-16))
			} else {
				m.selected.extPerms = append(
					m.selected.extPerms[:m.permIdx],
					m.selected.extPerms[m.permIdx+1:]...,
				)
				if m.permIdx >= len(m.selected.extPerms) && m.permIdx > 0 {
					m.permIdx--
				}
				if summary != "" {
					m.statusMsg = tuiSuccess.Render("Revoked: " + summary)
				} else {
					m.statusMsg = tuiSuccess.Render("Permission revoked")
				}
				if len(m.selected.extPerms) == 0 {
					m.view = "list"
					m.statusMsg = tuiSuccess.Render("All external permissions revoked for " + m.selected.issue.FileName)
				}
			}
		}
	case "n", "N", "esc":
		m.confirming = false
	}
	return m, nil
}

func (m tuiModel) viewDetail() string {
	if m.selected == nil {
		return ""
	}
	entry := m.selected
	w := m.width
	var b strings.Builder

	// ── Header bar ──
	backHint := tuiKey.Render("Esc") + tuiHelp.Render(" back")
	title := tuiTitle.Render(truncateString(entry.issue.FileName, w-20))
	headerLeft := " " + backHint + "    " + title
	headerLine := tuiHeaderBar.Width(w).Render(headerLeft)
	b.WriteString(headerLine + "\n\n")

	// ── Metadata box ──
	metaLines := []string{}
	if entry.issue.DriveName != "" {
		metaLines = append(metaLines, tuiLabel.Render("Drive  ")+entry.issue.DriveName)
	}
	if entry.issue.FolderPath != "" && entry.issue.FolderPath != "/" {
		metaLines = append(metaLines, tuiLabel.Render("Path   ")+entry.issue.FolderPath)
	}
	if entry.issue.OwnerEmail != "" {
		metaLines = append(metaLines, tuiLabel.Render("Owner  ")+entry.issue.OwnerEmail)
	}
	if entry.issue.Label != "" && entry.issue.Label != "Cannot be retrieved" && entry.issue.Label != "Unlabeled" {
		metaLines = append(metaLines, tuiLabel.Render("Label  ")+entry.issue.Label)
	}
	if entry.issue.WebViewLink != "" {
		metaLines = append(metaLines, tuiLabel.Render("Link   ")+tuiDim.Render(entry.issue.WebViewLink))
	}

	boxW := w - 6
	if boxW > 100 {
		boxW = 100
	}
	if len(metaLines) > 0 {
		box := tuiMetaBox.Width(boxW).Render(strings.Join(metaLines, "\n"))
		b.WriteString(indent(box, 2) + "\n")
	}

	// ── Internal users ──
	if len(entry.intPerms) > 0 {
		b.WriteString("\n" + indent(tuiSection.Render("Internal Users"), 2) + "\n")
		for _, p := range entry.intPerms {
			line := fmt.Sprintf("  %s %s", permIdentifier(p), tuiDim.Render("("+p.Role+")"))
			b.WriteString(indent(tuiSubtle.Render(line), 2) + "\n")
		}
	}

	// ── External shares ──
	b.WriteString("\n" + indent(tuiSection.Render(fmt.Sprintf("External Shares (%d)", len(entry.extPerms))), 2) + "\n\n")

	emailW := 35
	if w > 100 {
		emailW = w - 55
		if emailW > 50 {
			emailW = 50
		}
	}

	for i, p := range entry.extPerms {
		who := permIdentifier(p)
		risk := riskBadge(p.RiskLevel)
		role := tuiDim.Render(p.Role)

		row := fmt.Sprintf("  %s  %-*s  %s", risk, emailW, who, role)
		if i == m.permIdx {
			b.WriteString("  " + tuiCursorRow.Width(w-4).Render("▸"+row) + "\n")
		} else {
			b.WriteString("   " + row + "\n")
		}
	}

	// ── Confirmation dialog ──
	if m.confirming && m.permIdx < len(entry.extPerms) {
		perm := entry.extPerms[m.permIdx]
		who := permIdentifier(perm)
		prompt := fmt.Sprintf("Revoke %s from this file?", tuiSelected.Render(who))
		hint := tuiKey.Render("y") + tuiHelp.Render(" confirm  ") + tuiKey.Render("n") + tuiHelp.Render(" cancel")
		dialog := tuiConfirmBox.Width(boxW).Render(prompt + "\n" + hint)
		b.WriteString("\n" + indent(dialog, 2) + "\n")
	} else if m.statusMsg != "" {
		// Render status in a width-constrained box to avoid layout breakage
		statusBox := tuiMetaBox.Width(boxW).Render(m.statusMsg)
		b.WriteString("\n" + indent(statusBox, 2) + "\n")
	}

	// ── Footer ──
	// Fill remaining height
	lines := strings.Count(b.String(), "\n")
	for i := lines; i < m.height-2; i++ {
		b.WriteString("\n")
	}

	footer := keyHint("↑↓", "navigate") + "  " + keyHint("r", "revoke") + "  " + keyHint("o", "open in browser") + "  " + keyHint("Esc", "back")
	b.WriteString(footer)

	return b.String()
}

// ── helpers ─────────────────────────────────────────────────────────────────

func riskBadge(r models.RiskLevel) string {
	label := string(r)
	padded := fmt.Sprintf("%-8s", label)
	switch r {
	case models.RiskCritical:
		return criticalStyle.Render("● " + padded)
	case models.RiskHigh:
		return highStyle.Render("● " + padded)
	case models.RiskMedium:
		return mediumStyle.Render("● " + padded)
	case models.RiskLow:
		return lowStyle.Render("● " + padded)
	default:
		return "          "
	}
}

// sanitizeError extracts a short, single-line error message safe for TUI display.
// Google API errors contain multi-line JSON details that would break the layout.
func sanitizeError(err error, maxWidth int) string {
	msg := err.Error()

	// Google API errors follow pattern: "...: googleapi: Error NNN: <message>, ..."
	// Extract just the human-readable part
	if idx := strings.Index(msg, "googleapi: Error"); idx >= 0 {
		msg = msg[idx:]
	}

	// Collapse all whitespace (newlines, tabs, multi-spaces) into single spaces
	fields := strings.Fields(msg)
	msg = strings.Join(fields, " ")

	// Cut at "Details:" — everything after is JSON noise
	if idx := strings.Index(msg, "Details:"); idx > 0 {
		msg = strings.TrimSpace(msg[:idx])
	}
	// Also cut at "More details:" suffix
	if idx := strings.Index(msg, "More details:"); idx > 0 {
		msg = strings.TrimSpace(msg[:idx])
	}

	if maxWidth > 10 && len(msg) > maxWidth {
		msg = msg[:maxWidth-3] + "..."
	}
	return msg
}

func permIdentifier(p models.Permission) string {
	if p.Email != "" {
		return p.Email
	}
	if p.Domain != "" {
		return "@" + p.Domain
	}
	if p.Type == "anyone" {
		return "Anyone with link"
	}
	return "-"
}

func keyHint(key, desc string) string {
	return tuiKey.Render(key) + " " + tuiHelp.Render(desc)
}

func indent(s string, n int) string {
	pad := strings.Repeat(" ", n)
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = pad + l
	}
	return strings.Join(lines, "\n")
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		return
	}
	_ = cmd.Start()
}
