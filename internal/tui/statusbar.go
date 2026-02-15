package tui

import "github.com/charmbracelet/lipgloss"

type statusBar struct {
	message       string
	width         int
	isError       bool
	multiAccount  bool
	readerVisible bool
}

func newStatusBar() statusBar {
	return statusBar{message: "Ready"}
}

func (s *statusBar) setMessage(msg string) {
	s.message = msg
	s.isError = false
}

func (s *statusBar) setError(msg string) {
	s.message = msg
	s.isError = true
}

func (s statusBar) View() string {
	msgStyle := statusBarStyle
	if s.isError {
		msgStyle = msgStyle.Foreground(errorColor)
	}

	left := s.message
	shortcuts := s.shortcuts()

	gap := s.width - lipgloss.Width(left) - lipgloss.Width(shortcuts) - 2
	if gap < 0 {
		gap = 0
	}

	content := left + lipgloss.NewStyle().Width(gap).Render("") + mutedTextStyle.Render(shortcuts)
	return msgStyle.Width(s.width).Render(content)
}

func (s statusBar) shortcuts() string {
	if s.readerVisible {
		base := "r:reply  a:archive  d:trash  s:star  u:unread  esc:back"
		if s.multiAccount {
			return base + "  @:account"
		}
		return base
	}
	base := "j/k:nav  enter:open  c:compose  /:search"
	if s.multiAccount {
		return base + "  @:account"
	}
	return base
}
