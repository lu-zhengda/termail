package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/zhengda-lu/termail/internal/domain"
)

// Messages emitted by searchModel.

type searchQueryMsg struct {
	query string
}

type searchResultSelectedMsg struct {
	emailID string
}

type closeSearchMsg struct{}

// searchModel is a Bubble Tea sub-model for searching emails.
type searchModel struct {
	input     textinput.Model
	results   []domain.Email
	cursor    int
	searching bool
	inputMode bool
	width     int
	height    int
	focused   bool
}

func newSearch() searchModel {
	ti := textinput.New()
	ti.Placeholder = "Search emails..."
	ti.Prompt = "/ "
	ti.CharLimit = 256
	return searchModel{
		input:     ti,
		inputMode: true,
	}
}

func (s searchModel) Update(msg tea.Msg) (searchModel, tea.Cmd) {
	if !s.searching {
		return s, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Back):
			return s, func() tea.Msg { return closeSearchMsg{} }

		case key.Matches(msg, keys.Enter):
			if s.inputMode {
				// Execute search: emit query message and switch to results mode.
				q := s.input.Value()
				if q == "" {
					return s, nil
				}
				s.inputMode = false
				s.input.Blur()
				s.cursor = 0
				return s, func() tea.Msg { return searchQueryMsg{query: q} }
			}
			// In results mode: select highlighted result.
			id := s.SelectedEmailID()
			if id == "" {
				return s, nil
			}
			return s, func() tea.Msg { return searchResultSelectedMsg{emailID: id} }

		case key.Matches(msg, keys.Up):
			if !s.inputMode && s.cursor > 0 {
				s.cursor--
			}
			return s, nil

		case key.Matches(msg, keys.Down):
			if !s.inputMode && s.cursor < len(s.results)-1 {
				s.cursor++
			}
			return s, nil
		}
	}

	// Forward remaining key events to the text input when in input mode.
	if s.inputMode {
		var cmd tea.Cmd
		s.input, cmd = s.input.Update(msg)
		return s, cmd
	}

	return s, nil
}

func (s searchModel) View() string {
	if !s.searching || s.width == 0 {
		return ""
	}

	var b strings.Builder

	// Search input line.
	b.WriteString(s.input.View())
	b.WriteByte('\n')

	// Results section.
	if len(s.results) == 0 {
		if !s.inputMode {
			b.WriteByte('\n')
			b.WriteString(mutedTextStyle.Render("No results"))
		}
		return b.String()
	}

	b.WriteByte('\n')
	b.WriteString(titleStyle.Render(fmt.Sprintf("Results (%d):", len(s.results))))
	b.WriteByte('\n')

	// Determine how many results we can show.
	maxRows := s.height - 4 // input(1) + blank(1) + header(1) + padding(1)
	if maxRows < 1 {
		maxRows = 1
	}
	end := len(s.results)
	if end > maxRows {
		end = maxRows
	}

	for i := 0; i < end; i++ {
		if i > 0 {
			b.WriteByte('\n')
		}
		line := s.renderResultRow(i)
		if !s.inputMode && i == s.cursor && s.focused {
			line = selectedStyle.Width(s.width).Render(line)
		}
		b.WriteString(line)
	}

	return b.String()
}

// Open activates search mode and focuses the text input.
func (s *searchModel) Open() {
	s.searching = true
	s.inputMode = true
	s.focused = true
	s.input.Focus()
}

// Close deactivates search mode, clearing input and results.
func (s *searchModel) Close() {
	s.searching = false
	s.inputMode = true
	s.focused = false
	s.input.SetValue("")
	s.input.Blur()
	s.results = nil
	s.cursor = 0
}

// SetResults updates the results list after a search query completes.
func (s *searchModel) SetResults(results []domain.Email) {
	s.results = results
	s.cursor = 0
}

// SetSize updates the dimensions available for rendering.
func (s *searchModel) SetSize(w, h int) {
	s.width = w
	s.height = h
	s.input.Width = w - 4 // account for prompt and padding
}

// IsActive reports whether the search overlay is currently shown.
func (s searchModel) IsActive() bool {
	return s.searching
}

// Query returns the current search query text.
func (s searchModel) Query() string {
	return s.input.Value()
}

// SelectedEmailID returns the ID of the currently highlighted result.
func (s searchModel) SelectedEmailID() string {
	if len(s.results) == 0 || s.cursor >= len(s.results) {
		return ""
	}
	return s.results[s.cursor].ID
}

// --- internal helpers ---

func (s searchModel) renderResultRow(idx int) string {
	if idx >= len(s.results) {
		return ""
	}
	e := s.results[idx]

	star := "  "
	if e.IsStarred {
		star = starStyle.Render("★ ")
	}

	from := addressDisplayName(e.From)
	date := relativeDate(e.Date)

	fromWidth := 18
	dateWidth := len(date)
	subjectWidth := s.width - fromWidth - dateWidth - 4 // star(2) + gaps(2)
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
