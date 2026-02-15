package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/zhengda-lu/termail/internal/domain"
)

// Messages emitted by inboxModel.

type emailSelectedMsg struct {
	emailID string
}

type threadSelectedMsg struct {
	threadID string
}

type emailActionMsg struct {
	emailID string
	action  string
}

// inboxModel is a Bubble Tea sub-model that displays the email or thread list.
type inboxModel struct {
	emails      []domain.Email
	threads     []domain.Thread
	cursor      int
	offset      int
	viewMode    viewMode
	activeLabel string
	width       int
	height      int
	focused     bool
}

func newInbox() inboxModel {
	return inboxModel{
		viewMode: viewThread,
	}
}

func (m inboxModel) Update(msg tea.Msg) (inboxModel, tea.Cmd) {
	if !m.focused {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Up):
			if m.cursor > 0 {
				m.cursor--
				m.adjustScroll()
			}

		case key.Matches(msg, keys.Down):
			if m.cursor < m.itemCount()-1 {
				m.cursor++
				m.adjustScroll()
			}

		case key.Matches(msg, keys.Enter):
			return m, m.selectItem()

		case key.Matches(msg, keys.Archive):
			return m, m.actionCmd("archive")

		case key.Matches(msg, keys.Delete):
			return m, m.actionCmd("delete")

		case key.Matches(msg, keys.Star):
			return m, m.actionCmd("star")

		case key.Matches(msg, keys.Unread):
			return m, m.actionCmd("unread")
		}
	}

	return m, nil
}

func (m inboxModel) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	visible := m.visibleRows()
	count := m.itemCount()
	if count == 0 {
		return mutedTextStyle.Render("No messages")
	}

	var b strings.Builder
	end := m.offset + visible
	if end > count {
		end = count
	}

	for i := m.offset; i < end; i++ {
		if i > m.offset {
			b.WriteByte('\n')
		}
		line := m.renderRow(i)
		if i == m.cursor && m.focused {
			line = selectedStyle.Width(m.width).Render(line)
		}
		b.WriteString(line)
	}

	return b.String()
}

// SetEmails updates the email list for flat view.
func (m *inboxModel) SetEmails(emails []domain.Email) {
	m.emails = emails
	m.clampCursor()
}

// SetThreads updates the thread list for thread view.
func (m *inboxModel) SetThreads(threads []domain.Thread) {
	m.threads = threads
	m.clampCursor()
}

// SetSize updates the dimensions available for rendering.
func (m *inboxModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.adjustScroll()
}

// SetViewMode switches between thread and flat view.
func (m *inboxModel) SetViewMode(vm viewMode) {
	m.viewMode = vm
	m.cursor = 0
	m.offset = 0
}

// SelectedEmailID returns the ID of the currently highlighted email (flat view).
func (m inboxModel) SelectedEmailID() string {
	if m.viewMode != viewFlat || len(m.emails) == 0 || m.cursor >= len(m.emails) {
		return ""
	}
	return m.emails[m.cursor].ID
}

// SelectedThreadID returns the ID of the currently highlighted thread (thread view).
func (m inboxModel) SelectedThreadID() string {
	if m.viewMode != viewThread || len(m.threads) == 0 || m.cursor >= len(m.threads) {
		return ""
	}
	return m.threads[m.cursor].ID
}

// --- internal helpers ---

func (m inboxModel) itemCount() int {
	if m.viewMode == viewThread {
		return len(m.threads)
	}
	return len(m.emails)
}

func (m inboxModel) visibleRows() int {
	if m.height < 1 {
		return 1
	}
	return m.height
}

func (m *inboxModel) adjustScroll() {
	visible := m.visibleRows()
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+visible {
		m.offset = m.cursor - visible + 1
	}
}

func (m *inboxModel) clampCursor() {
	count := m.itemCount()
	if count == 0 {
		m.cursor = 0
		m.offset = 0
		return
	}
	if m.cursor >= count {
		m.cursor = count - 1
	}
	m.adjustScroll()
}

func (m inboxModel) selectItem() tea.Cmd {
	if m.viewMode == viewThread {
		id := m.SelectedThreadID()
		if id == "" {
			return nil
		}
		return func() tea.Msg {
			return threadSelectedMsg{threadID: id}
		}
	}
	id := m.SelectedEmailID()
	if id == "" {
		return nil
	}
	return func() tea.Msg {
		return emailSelectedMsg{emailID: id}
	}
}

func (m inboxModel) actionCmd(action string) tea.Cmd {
	var emailID string
	if m.viewMode == viewThread {
		if len(m.threads) > 0 && m.cursor < len(m.threads) {
			msgs := m.threads[m.cursor].Messages
			if len(msgs) > 0 {
				emailID = msgs[len(msgs)-1].ID
			}
		}
	} else {
		emailID = m.SelectedEmailID()
	}
	if emailID == "" {
		return nil
	}
	return func() tea.Msg {
		return emailActionMsg{emailID: emailID, action: action}
	}
}

func (m inboxModel) renderRow(idx int) string {
	if m.viewMode == viewThread {
		return m.renderThreadRow(idx)
	}
	return m.renderEmailRow(idx)
}

func (m inboxModel) renderEmailRow(idx int) string {
	if idx >= len(m.emails) {
		return ""
	}
	e := m.emails[idx]

	star := "  "
	if e.IsStarred {
		star = starStyle.Render("★ ")
	}

	from := addressDisplayName(e.From)
	date := relativeDate(e.Date)

	fromWidth := 18
	dateWidth := len(date)
	subjectWidth := m.width - fromWidth - dateWidth - 6 // star(2) + two "  " gaps(4)
	if subjectWidth < 10 {
		subjectWidth = 10
	}

	from = truncate(from, fromWidth)
	subject := truncate(e.Subject, subjectWidth)

	fromCol := lipgloss.NewStyle().Width(fromWidth).Render(from)
	subjectCol := lipgloss.NewStyle().Width(subjectWidth).Render(subject)
	dateCol := mutedTextStyle.Width(dateWidth).Render(date)

	line := star + fromCol + "  " + subjectCol + "  " + dateCol

	if !e.IsRead {
		line = unreadStyle.Render(line)
	}

	return line
}

func (m inboxModel) renderThreadRow(idx int) string {
	if idx >= len(m.threads) {
		return ""
	}
	t := m.threads[idx]

	starred := false
	for i := range t.Messages {
		if t.Messages[i].IsStarred {
			starred = true
			break
		}
	}

	star := "  "
	if starred {
		star = starStyle.Render("★ ")
	}

	from := threadFromName(t)
	count := fmt.Sprintf("(%d)", t.MessageCount())
	date := relativeDate(t.LastDate)

	fromWidth := 18
	countWidth := len(count) + 1 // +1 for leading space
	dateWidth := len(date)
	subjectWidth := m.width - fromWidth - countWidth - dateWidth - 6 // star(2) + two "  " gaps(4)
	if subjectWidth < 10 {
		subjectWidth = 10
	}

	from = truncate(from, fromWidth)
	subject := truncate(t.Subject, subjectWidth)

	fromCol := lipgloss.NewStyle().Width(fromWidth).Render(from)
	countCol := mutedTextStyle.Render(" " + count)
	subjectCol := lipgloss.NewStyle().Width(subjectWidth).Render(subject)
	dateCol := mutedTextStyle.Width(dateWidth).Render(date)

	line := star + fromCol + countCol + "  " + subjectCol + "  " + dateCol

	if t.IsUnread() {
		line = unreadStyle.Render(line)
	}

	return line
}

// --- utility functions ---

func addressDisplayName(addr domain.Address) string {
	if addr.Name != "" {
		return addr.Name
	}
	return addr.Email
}

func threadFromName(t domain.Thread) string {
	if t.FromAddress.Name != "" || t.FromAddress.Email != "" {
		return addressDisplayName(t.FromAddress)
	}
	if len(t.Messages) > 0 {
		return addressDisplayName(t.Messages[0].From)
	}
	return "Unknown"
}

func truncate(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen <= 1 {
		return string(runes[:maxLen])
	}
	return string(runes[:maxLen-1]) + "…"
}

func relativeDate(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	default:
		return t.Format("Jan 2")
	}
}
