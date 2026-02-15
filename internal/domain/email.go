package domain

import "time"

type Address struct {
	Name  string
	Email string
}

func (a Address) String() string {
	if a.Name == "" {
		return a.Email
	}
	return a.Name + " <" + a.Email + ">"
}

type Attachment struct {
	ID       string
	Filename string
	MIMEType string
	Size     int64
}

type Email struct {
	ID          string
	ThreadID    string
	From        Address
	To          []Address
	CC          []Address
	BCC         []Address
	Subject     string
	Body        string
	BodyHTML    string
	Date        time.Time
	Labels      []string
	IsRead      bool
	IsStarred   bool
	Attachments []Attachment
	InReplyTo   string
}

func (e *Email) HasLabel(label string) bool {
	for _, l := range e.Labels {
		if l == label {
			return true
		}
	}
	return false
}
