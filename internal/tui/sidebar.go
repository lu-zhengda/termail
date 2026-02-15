package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lu-zhengda/termail/internal/domain"
)

// labelSelectedMsg is sent when the user selects a label via Enter.
type labelSelectedMsg struct {
	labelID string
}

// systemLabelOrder defines the display order for system labels.
var systemLabelOrder = []string{
	domain.LabelInbox,
	domain.LabelStarred,
	domain.LabelSent,
	domain.LabelDraft,
	domain.LabelTrash,
	domain.LabelSpam,
}

// systemLabelNames maps system label IDs to human-friendly display names.
var systemLabelNames = map[string]string{
	domain.LabelInbox:   "Inbox",
	domain.LabelStarred: "Starred",
	domain.LabelSent:    "Sent",
	domain.LabelDraft:   "Drafts",
	domain.LabelTrash:   "Trash",
	domain.LabelSpam:    "Spam",
}

// sidebarModel displays a navigable list of email labels.
type sidebarModel struct {
	labels       []domain.Label
	cursor       int
	activeLabel  string
	accountEmail string
	width        int
	height       int
	focused      bool
}

// newSidebar creates a new sidebar with INBOX as the default active label.
func newSidebar() sidebarModel {
	return sidebarModel{
		activeLabel: domain.LabelInbox,
	}
}

// SetLabels updates the label list displayed in the sidebar.
func (s *sidebarModel) SetLabels(labels []domain.Label) {
	s.labels = labels
}

// SetSize updates the sidebar dimensions.
func (s *sidebarModel) SetSize(w, h int) {
	s.width = w
	s.height = h
}

// Update handles key events for sidebar navigation.
func (s sidebarModel) Update(msg tea.Msg) (sidebarModel, tea.Cmd) {
	if !s.focused {
		return s, nil
	}

	total := s.totalItems()
	if total == 0 {
		return s, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Up):
			s.cursor--
			if s.cursor < 0 {
				s.cursor = total - 1
			}
		case key.Matches(msg, keys.Down):
			s.cursor++
			if s.cursor >= total {
				s.cursor = 0
			}
		case key.Matches(msg, keys.Enter):
			if labelID, ok := s.labelIDAtCursor(); ok {
				s.activeLabel = labelID
				return s, func() tea.Msg {
					return labelSelectedMsg{labelID: labelID}
				}
			}
		}
	}

	return s, nil
}

// View renders the sidebar.
func (s sidebarModel) View() string {
	var b strings.Builder

	// Title and account
	b.WriteString(titleStyle.Render("termail"))
	b.WriteString("\n")
	if s.accountEmail != "" {
		b.WriteString(mutedTextStyle.Render(truncateEmail(s.accountEmail, max(s.width, 10))))
	}
	b.WriteString("\n")

	if len(s.labels) == 0 {
		b.WriteString(mutedTextStyle.Render("Loading labels..."))
		return b.String()
	}

	systemLabels, userLabels := s.partitionLabels()
	itemIdx := 0

	// System labels
	for _, label := range systemLabels {
		name := displayName(label)
		line := s.renderLine(name, label.ID, itemIdx)
		b.WriteString(line)
		b.WriteString("\n")
		itemIdx++
	}

	// Separator and user labels section
	if len(userLabels) > 0 {
		b.WriteString("\n")
		b.WriteString(mutedTextStyle.Render(strings.Repeat("─", max(s.width, 10))))
		b.WriteString("\n")
		b.WriteString(mutedTextStyle.Render("Labels:"))
		b.WriteString("\n")

		for _, label := range userLabels {
			name := displayName(label)
			line := s.renderLine(name, label.ID, itemIdx)
			b.WriteString(line)
			b.WriteString("\n")
			itemIdx++
		}
	}

	return b.String()
}

// renderLine renders a single label line with cursor highlighting and active marker.
func (s sidebarModel) renderLine(name, labelID string, idx int) string {
	prefix := "  "
	if labelID == s.activeLabel {
		prefix = "▶ "
	}

	line := fmt.Sprintf("%s%s", prefix, name)

	// Pad to width so highlight covers the full line.
	padded := lipgloss.NewStyle().Width(max(s.width, 10)).Render(line)

	if s.focused && idx == s.cursor {
		return selectedStyle.Render(padded)
	}

	return padded
}

// partitionLabels splits labels into system and user groups, keeping system labels
// in the canonical display order.
func (s sidebarModel) partitionLabels() (system, user []domain.Label) {
	labelMap := make(map[string]domain.Label, len(s.labels))
	for _, l := range s.labels {
		labelMap[l.ID] = l
	}

	for _, id := range systemLabelOrder {
		if l, ok := labelMap[id]; ok {
			system = append(system, l)
		}
	}

	for _, l := range s.labels {
		if l.Type == domain.LabelTypeUser {
			user = append(user, l)
		}
	}

	return system, user
}

// totalItems returns the total number of navigable items.
func (s sidebarModel) totalItems() int {
	sys, usr := s.partitionLabels()
	return len(sys) + len(usr)
}

// labelIDAtCursor returns the label ID at the current cursor position.
func (s sidebarModel) labelIDAtCursor() (string, bool) {
	sys, usr := s.partitionLabels()
	all := append(sys, usr...)
	if s.cursor < 0 || s.cursor >= len(all) {
		return "", false
	}
	return all[s.cursor].ID, true
}

// displayName returns the human-friendly name for a label.
func displayName(l domain.Label) string {
	if name, ok := systemLabelNames[l.ID]; ok {
		return name
	}
	return l.Name
}

// truncateEmail shortens an email address to fit within maxLen.
func truncateEmail(email string, maxLen int) string {
	if len(email) <= maxLen {
		return email
	}
	if maxLen <= 3 {
		return email[:maxLen]
	}
	return email[:maxLen-1] + "…"
}
