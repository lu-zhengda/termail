package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/lu-zhengda/termail/internal/domain"
)

// Messages emitted by readerModel.

type replyMsg struct {
	email    *domain.Email
	replyAll bool
}

type forwardMsg struct {
	email *domain.Email
}

type closeReaderMsg struct{}

// readerModel is a Bubble Tea sub-model for displaying email content
// in a scrollable viewport.
type readerModel struct {
	email        *domain.Email
	thread       *domain.Thread
	content      string
	scrollOffset int
	maxScroll    int
	width        int
	height       int
	focused      bool
	visible      bool
}

func newReader() readerModel {
	return readerModel{}
}

func (r readerModel) Update(msg tea.Msg) (readerModel, tea.Cmd) {
	if !r.focused || !r.visible {
		return r, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Up):
			if r.scrollOffset > 0 {
				r.scrollOffset--
			}

		case key.Matches(msg, keys.Down):
			if r.scrollOffset < r.maxScroll {
				r.scrollOffset++
			}

		case key.Matches(msg, keys.Back):
			return r, func() tea.Msg {
				return closeReaderMsg{}
			}

		case key.Matches(msg, keys.Reply):
			email := r.currentEmail()
			if email != nil {
				return r, func() tea.Msg {
					return replyMsg{email: email, replyAll: false}
				}
			}

		case key.Matches(msg, keys.ReplyAll):
			email := r.currentEmail()
			if email != nil {
				return r, func() tea.Msg {
					return replyMsg{email: email, replyAll: true}
				}
			}

		case key.Matches(msg, keys.Forward):
			email := r.currentEmail()
			if email != nil {
				return r, func() tea.Msg {
					return forwardMsg{email: email}
				}
			}

		case key.Matches(msg, keys.Archive):
			email := r.currentEmail()
			if email != nil {
				return r, func() tea.Msg {
					return emailActionMsg{emailID: email.ID, action: "archive"}
				}
			}

		case key.Matches(msg, keys.Delete):
			email := r.currentEmail()
			if email != nil {
				return r, func() tea.Msg {
					return emailActionMsg{emailID: email.ID, action: "delete"}
				}
			}

		case key.Matches(msg, keys.Star):
			email := r.currentEmail()
			if email != nil {
				return r, func() tea.Msg {
					return emailActionMsg{emailID: email.ID, action: "star"}
				}
			}

		case key.Matches(msg, keys.Unread):
			email := r.currentEmail()
			if email != nil {
				return r, func() tea.Msg {
					return emailActionMsg{emailID: email.ID, action: "unread"}
				}
			}
		}
	}

	return r, nil
}

func (r readerModel) View() string {
	if !r.visible || r.width == 0 || r.height == 0 {
		return ""
	}

	if r.content == "" {
		return mutedTextStyle.Render("No email selected")
	}

	lines := strings.Split(r.content, "\n")

	visibleHeight := r.height
	if visibleHeight < 1 {
		visibleHeight = 1
	}

	end := r.scrollOffset + visibleHeight
	if end > len(lines) {
		end = len(lines)
	}

	start := r.scrollOffset
	if start > len(lines) {
		start = len(lines)
	}

	visible := strings.Join(lines[start:end], "\n")
	return visible
}

// ShowEmail displays a single email in the reader pane.
func (r *readerModel) ShowEmail(email *domain.Email) {
	r.email = email
	r.thread = nil
	r.visible = true
	r.scrollOffset = 0
	r.content = renderEmail(email, r.width)
	r.recalcMaxScroll()
}

// ShowThread displays a thread (all messages) in the reader pane.
func (r *readerModel) ShowThread(thread *domain.Thread) {
	r.thread = thread
	r.email = nil
	r.visible = true
	r.scrollOffset = 0
	r.content = renderThread(thread, r.width)
	r.recalcMaxScroll()
}

// Close hides the reader and clears its content.
func (r *readerModel) Close() {
	r.visible = false
	r.email = nil
	r.thread = nil
	r.content = ""
	r.scrollOffset = 0
	r.maxScroll = 0
}

// SetSize updates the reader dimensions and recalculates scroll bounds.
func (r *readerModel) SetSize(w, h int) {
	r.width = w
	r.height = h
	// Re-render content if we have something to display, since width may affect layout.
	if r.email != nil {
		r.content = renderEmail(r.email, r.width)
	} else if r.thread != nil {
		r.content = renderThread(r.thread, r.width)
	}
	r.recalcMaxScroll()
}

// IsVisible returns whether the reader pane is currently shown.
func (r readerModel) IsVisible() bool {
	return r.visible
}

// --- internal helpers ---

// currentEmail returns the email that should be used for reply/forward actions.
// For single email view it returns the email; for thread view it returns the
// last (most recent) message.
func (r readerModel) currentEmail() *domain.Email {
	if r.email != nil {
		return r.email
	}
	if r.thread != nil && len(r.thread.Messages) > 0 {
		return &r.thread.Messages[len(r.thread.Messages)-1]
	}
	return nil
}

func (r *readerModel) recalcMaxScroll() {
	if r.content == "" {
		r.maxScroll = 0
		r.scrollOffset = 0
		return
	}

	lines := strings.Split(r.content, "\n")
	visibleHeight := r.height
	if visibleHeight < 1 {
		visibleHeight = 1
	}

	r.maxScroll = len(lines) - visibleHeight
	if r.maxScroll < 0 {
		r.maxScroll = 0
	}

	// Clamp current scroll offset.
	if r.scrollOffset > r.maxScroll {
		r.scrollOffset = r.maxScroll
	}
}

// renderEmail formats a single email as a plain-text string with headers and body.
func renderEmail(email *domain.Email, width int) string {
	var b strings.Builder

	// Headers
	b.WriteString(mutedTextStyle.Render("From:    "))
	b.WriteString(email.From.String())
	b.WriteByte('\n')

	b.WriteString(mutedTextStyle.Render("To:      "))
	b.WriteString(formatAddresses(email.To))
	b.WriteByte('\n')

	if len(email.CC) > 0 {
		b.WriteString(mutedTextStyle.Render("CC:      "))
		b.WriteString(formatAddresses(email.CC))
		b.WriteByte('\n')
	}

	b.WriteString(mutedTextStyle.Render("Date:    "))
	b.WriteString(email.Date.Format("Jan 2, 2006 3:04 PM"))
	b.WriteByte('\n')

	b.WriteString(mutedTextStyle.Render("Subject: "))
	b.WriteString(email.Subject)
	b.WriteByte('\n')

	// Separator
	sepWidth := width
	if sepWidth < 20 {
		sepWidth = 20
	}
	b.WriteString(mutedTextStyle.Render(strings.Repeat("\u2500", sepWidth)))
	b.WriteByte('\n')

	// Body
	body := email.Body
	if body == "" && email.BodyHTML != "" {
		body = "[HTML content - plain text not available]"
	}
	if body != "" {
		b.WriteByte('\n')
		b.WriteString(body)
	}

	return b.String()
}

// renderThread formats all messages in a thread, separated by blank lines
// and separator lines, with the most recent message at the bottom.
func renderThread(thread *domain.Thread, width int) string {
	if len(thread.Messages) == 0 {
		return mutedTextStyle.Render("Empty thread")
	}

	var parts []string
	for i := range thread.Messages {
		parts = append(parts, renderEmail(&thread.Messages[i], width))
	}

	sepWidth := width
	if sepWidth < 20 {
		sepWidth = 20
	}
	separator := "\n" + mutedTextStyle.Render(strings.Repeat("\u2500", sepWidth)) + "\n"

	return strings.Join(parts, separator)
}

// formatAddresses joins a slice of addresses into a comma-separated string.
func formatAddresses(addrs []domain.Address) string {
	if len(addrs) == 0 {
		return ""
	}

	parts := make([]string, len(addrs))
	for i, a := range addrs {
		parts[i] = a.String()
	}
	return fmt.Sprintf("%s", strings.Join(parts, ", "))
}
