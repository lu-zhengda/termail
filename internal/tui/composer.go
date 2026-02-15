package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lu-zhengda/termail/internal/domain"
)

// composerMode describes the kind of composition taking place.
type composerMode int

const (
	modeCompose composerMode = iota
	modeReply
	modeReplyAll
	modeForward
)

// Messages emitted by composerModel.

type sendMsg struct {
	email *domain.Email
}

type cancelComposeMsg struct{}

// Field indices within the composer form.
const (
	fieldTo      = 0
	fieldCC      = 1
	fieldSubject = 2
	fieldBody    = 3
	fieldCount   = 4
)

// composerModel is a Bubble Tea sub-model for composing, replying, and forwarding emails.
type composerModel struct {
	toInput      textinput.Model
	ccInput      textinput.Model
	subjectInput textinput.Model
	bodyInput    textarea.Model

	activeField int
	mode        composerMode
	replyTo     *domain.Email

	width   int
	height  int
	visible bool
}

// newComposer creates a new composerModel with text inputs and textarea configured.
func newComposer() composerModel {
	to := textinput.New()
	to.Placeholder = "recipient@example.com"
	to.CharLimit = 500
	to.Prompt = ""

	cc := textinput.New()
	cc.Placeholder = "cc@example.com"
	cc.CharLimit = 500
	cc.Prompt = ""

	subject := textinput.New()
	subject.Placeholder = "Subject"
	subject.CharLimit = 200
	subject.Prompt = ""

	body := textarea.New()
	body.Placeholder = "Write your message..."
	body.SetWidth(40)
	body.SetHeight(6)
	body.CharLimit = 0

	return composerModel{
		toInput:      to,
		ccInput:      cc,
		subjectInput: subject,
		bodyInput:    body,
	}
}

// Update handles key events for the composer form.
func (c composerModel) Update(msg tea.Msg) (composerModel, tea.Cmd) {
	if !c.visible {
		return c, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			c.activeField = (c.activeField + 1) % fieldCount
			c.updateFocus()
			return c, nil

		case "esc":
			return c, func() tea.Msg { return cancelComposeMsg{} }

		case "ctrl+s":
			email := c.BuildEmail()
			return c, func() tea.Msg { return sendMsg{email: email} }
		}
	}

	// Delegate to the active input component.
	var cmd tea.Cmd
	switch c.activeField {
	case fieldTo:
		c.toInput, cmd = c.toInput.Update(msg)
	case fieldCC:
		c.ccInput, cmd = c.ccInput.Update(msg)
	case fieldSubject:
		c.subjectInput, cmd = c.subjectInput.Update(msg)
	case fieldBody:
		c.bodyInput, cmd = c.bodyInput.Update(msg)
	}

	return c, cmd
}

// View renders the compose form inside a bordered box.
func (c composerModel) View() string {
	if !c.visible {
		return ""
	}

	innerWidth := c.width - 4 // account for border + padding
	if innerWidth < 20 {
		innerWidth = 20
	}

	title := c.modeTitle()

	labelWidth := 10 // "Subject: " + spacing
	inputWidth := innerWidth - labelWidth
	if inputWidth < 10 {
		inputWidth = 10
	}

	// Resize inputs to fit.
	c.toInput.Width = inputWidth
	c.ccInput.Width = inputWidth
	c.subjectInput.Width = inputWidth
	c.bodyInput.SetWidth(innerWidth)

	// Calculate body height: total height minus border(2) padding(2) fields(3) separator(1) help(1) spacing(1).
	bodyHeight := c.height - 10
	if bodyHeight < 3 {
		bodyHeight = 3
	}
	c.bodyInput.SetHeight(bodyHeight)

	toLabel := mutedTextStyle.Render(fmt.Sprintf("%-9s", "To:"))
	ccLabel := mutedTextStyle.Render(fmt.Sprintf("%-9s", "CC:"))
	subjectLabel := mutedTextStyle.Render(fmt.Sprintf("%-9s", "Subject:"))

	separator := mutedTextStyle.Render(strings.Repeat("─", innerWidth))

	helpText := mutedTextStyle.Render("Tab:fields  Ctrl+S:send  Esc:cancel")

	var rows []string
	rows = append(rows, toLabel+c.toInput.View())
	rows = append(rows, ccLabel+c.ccInput.View())
	rows = append(rows, subjectLabel+c.subjectInput.View())
	rows = append(rows, separator)
	rows = append(rows, c.bodyInput.View())
	rows = append(rows, "")
	rows = append(rows, helpText)

	content := strings.Join(rows, "\n")

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(primaryColor).
		Padding(0, 1).
		Width(c.width - 2)

	titleRow := titleStyle.Render(" " + title + " ")
	header := lipgloss.NewStyle().
		BorderForeground(primaryColor).
		Render(titleRow)

	return header + "\n" + boxStyle.Render(content)
}

// Compose opens the composer for a new email, clearing all fields.
func (c *composerModel) Compose() {
	c.mode = modeCompose
	c.replyTo = nil
	c.clearFields()
	c.visible = true
	c.activeField = fieldTo
	c.updateFocus()
}

// Reply opens the composer pre-filled for replying to the given email.
// If replyAll is true, CC is populated with the original To and CC recipients.
func (c *composerModel) Reply(email *domain.Email, replyAll bool) {
	c.replyTo = email
	c.clearFields()
	c.visible = true

	if replyAll {
		c.mode = modeReplyAll
	} else {
		c.mode = modeReply
	}

	// Pre-fill To with the original sender.
	c.toInput.SetValue(email.From.String())

	// For reply-all, populate CC with original To and CC (excluding the sender already in To).
	if replyAll {
		var ccAddrs []string
		for _, addr := range email.To {
			ccAddrs = append(ccAddrs, addr.String())
		}
		for _, addr := range email.CC {
			ccAddrs = append(ccAddrs, addr.String())
		}
		c.ccInput.SetValue(strings.Join(ccAddrs, ", "))
	}

	// Pre-fill Subject with "Re: " prefix.
	subject := email.Subject
	if !strings.HasPrefix(strings.ToLower(subject), "re: ") {
		subject = "Re: " + subject
	}
	c.subjectInput.SetValue(subject)

	// Quote the original body.
	quoted := formatReplyQuote(email)
	c.bodyInput.SetValue(quoted)

	c.activeField = fieldBody
	c.updateFocus()
}

// Forward opens the composer pre-filled for forwarding the given email.
func (c *composerModel) Forward(email *domain.Email) {
	c.mode = modeForward
	c.replyTo = email
	c.clearFields()
	c.visible = true

	// Pre-fill Subject with "Fwd: " prefix.
	subject := email.Subject
	if !strings.HasPrefix(strings.ToLower(subject), "fwd: ") {
		subject = "Fwd: " + subject
	}
	c.subjectInput.SetValue(subject)

	// Include forwarded body.
	forwarded := formatForwardBody(email)
	c.bodyInput.SetValue(forwarded)

	c.activeField = fieldTo
	c.updateFocus()
}

// Close hides the composer and clears all fields.
func (c *composerModel) Close() {
	c.visible = false
	c.clearFields()
}

// SetSize updates the available dimensions for the composer.
func (c *composerModel) SetSize(w, h int) {
	c.width = w
	c.height = h
}

// IsVisible reports whether the composer is currently displayed.
func (c composerModel) IsVisible() bool {
	return c.visible
}

// BuildEmail constructs a domain.Email from the current field values.
func (c composerModel) BuildEmail() *domain.Email {
	email := &domain.Email{
		To:      parseAddresses(c.toInput.Value()),
		CC:      parseAddresses(c.ccInput.Value()),
		Subject: c.subjectInput.Value(),
		Body:    c.bodyInput.Value(),
		Date:    time.Now(),
	}

	if c.replyTo != nil {
		email.InReplyTo = c.replyTo.ID
		email.ThreadID = c.replyTo.ThreadID
	}

	return email
}

// --- internal helpers ---

// clearFields resets all input fields to empty.
func (c *composerModel) clearFields() {
	c.toInput.SetValue("")
	c.ccInput.SetValue("")
	c.subjectInput.SetValue("")
	c.bodyInput.SetValue("")
}

// updateFocus sets the correct focus state on all input components.
func (c *composerModel) updateFocus() {
	c.toInput.Blur()
	c.ccInput.Blur()
	c.subjectInput.Blur()
	c.bodyInput.Blur()

	switch c.activeField {
	case fieldTo:
		c.toInput.Focus()
	case fieldCC:
		c.ccInput.Focus()
	case fieldSubject:
		c.subjectInput.Focus()
	case fieldBody:
		c.bodyInput.Focus()
	}
}

// modeTitle returns the title string for the current composer mode.
func (c composerModel) modeTitle() string {
	switch c.mode {
	case modeReply:
		return "Reply"
	case modeReplyAll:
		return "Reply All"
	case modeForward:
		return "Forward"
	default:
		return "Compose"
	}
}

// parseAddresses splits a comma-separated string into Address structs.
// Each entry is trimmed. If the entry contains "<email>", name and email are
// extracted; otherwise, the whole string is treated as an email address.
func parseAddresses(s string) []domain.Address {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}

	parts := strings.Split(s, ",")
	addrs := make([]domain.Address, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		addr := parseOneAddress(part)
		addrs = append(addrs, addr)
	}

	return addrs
}

// parseOneAddress parses a single address string.
// Supports "Name <email>" and bare "email" formats.
func parseOneAddress(s string) domain.Address {
	if idx := strings.LastIndex(s, "<"); idx >= 0 {
		end := strings.Index(s[idx:], ">")
		if end > 0 {
			name := strings.TrimSpace(s[:idx])
			email := s[idx+1 : idx+end]
			return domain.Address{Name: name, Email: email}
		}
	}
	return domain.Address{Email: s}
}

// formatReplyQuote builds the quoted text for a reply.
func formatReplyQuote(email *domain.Email) string {
	date := email.Date.Format("Jan 2, 2006")
	header := fmt.Sprintf("\nOn %s, %s wrote:", date, email.From.String())

	lines := strings.Split(email.Body, "\n")
	var quoted strings.Builder
	for _, line := range lines {
		quoted.WriteString("> ")
		quoted.WriteString(line)
		quoted.WriteString("\n")
	}

	return header + "\n" + quoted.String()
}

// formatForwardBody builds the forwarded message body.
func formatForwardBody(email *domain.Email) string {
	date := email.Date.Format("Jan 2, 2006")
	var b strings.Builder
	b.WriteString("\n---------- Forwarded message ----------\n")
	b.WriteString(fmt.Sprintf("From: %s\n", email.From.String()))
	b.WriteString(fmt.Sprintf("Date: %s\n", date))
	b.WriteString(fmt.Sprintf("Subject: %s\n", email.Subject))
	b.WriteString("\n")
	b.WriteString(email.Body)
	return b.String()
}
