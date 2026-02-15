package domain

import "time"

type Thread struct {
	ID       string
	Subject  string
	Messages []Email
	Labels   []string
	Snippet  string
	LastDate time.Time

	// Summary fields populated by list queries (Messages may be empty).
	FromAddress Address
	TotalCount  int
	HasUnread   bool
}

func (t *Thread) MessageCount() int {
	if len(t.Messages) > 0 {
		return len(t.Messages)
	}
	return t.TotalCount
}

func (t *Thread) IsUnread() bool {
	if len(t.Messages) == 0 {
		return t.HasUnread
	}
	for i := range t.Messages {
		if !t.Messages[i].IsRead {
			return true
		}
	}
	return false
}
