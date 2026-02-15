package domain

type LabelType string

const (
	LabelTypeSystem LabelType = "system"
	LabelTypeUser   LabelType = "user"
)

type Label struct {
	ID        string
	AccountID string
	Name      string
	Type      LabelType
	Color     string
}

const (
	LabelInbox   = "INBOX"
	LabelStarred = "STARRED"
	LabelSent    = "SENT"
	LabelDraft   = "DRAFT"
	LabelTrash   = "TRASH"
	LabelSpam    = "SPAM"
)
