package gmail

import (
	"testing"
	"time"

	gmailapi "google.golang.org/api/gmail/v1"
)

func TestParseAddress(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantName  string
		wantEmail string
	}{
		{
			name:      "name and email",
			input:     "John Doe <john@example.com>",
			wantName:  "John Doe",
			wantEmail: "john@example.com",
		},
		{
			name:      "email in angle brackets",
			input:     "<john@example.com>",
			wantName:  "",
			wantEmail: "john@example.com",
		},
		{
			name:      "bare email",
			input:     "john@example.com",
			wantName:  "",
			wantEmail: "john@example.com",
		},
		{
			name:      "quoted name",
			input:     `"Jane Doe" <jane@example.com>`,
			wantName:  "Jane Doe",
			wantEmail: "jane@example.com",
		},
		{
			name:      "empty string",
			input:     "",
			wantName:  "",
			wantEmail: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseAddress(tt.input)
			if got.Name != tt.wantName {
				t.Errorf("parseAddress(%q).Name = %q, want %q", tt.input, got.Name, tt.wantName)
			}
			if got.Email != tt.wantEmail {
				t.Errorf("parseAddress(%q).Email = %q, want %q", tt.input, got.Email, tt.wantEmail)
			}
		})
	}
}

func TestParseAddressList(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"single address", "john@example.com", 1},
		{"multiple addresses", "john@example.com, jane@example.com", 2},
		{"with names", "John <john@example.com>, Jane <jane@example.com>", 2},
		{"empty string", "", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseAddressList(tt.input)
			if len(got) != tt.want {
				t.Errorf("parseAddressList(%q) returned %d addresses, want %d", tt.input, len(got), tt.want)
			}
		})
	}
}

func TestFindHeader(t *testing.T) {
	headers := []*gmailapi.MessagePartHeader{
		{Name: "From", Value: "john@example.com"},
		{Name: "Subject", Value: "Hello"},
		{Name: "Date", Value: "Mon, 1 Jan 2024 00:00:00 +0000"},
	}

	tests := []struct {
		name string
		key  string
		want string
	}{
		{"existing header", "From", "john@example.com"},
		{"case insensitive", "from", "john@example.com"},
		{"subject header", "Subject", "Hello"},
		{"missing header", "Bcc", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findHeader(headers, tt.key)
			if got != tt.want {
				t.Errorf("findHeader(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

func TestContainsLabel(t *testing.T) {
	labels := []string{"INBOX", "UNREAD", "STARRED"}

	tests := []struct {
		name  string
		label string
		want  bool
	}{
		{"present label", "INBOX", true},
		{"absent label", "TRASH", false},
		{"starred present", "STARRED", true},
		{"empty label", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsLabel(labels, tt.label)
			if got != tt.want {
				t.Errorf("containsLabel(%v, %q) = %v, want %v", labels, tt.label, got, tt.want)
			}
		})
	}
}

func TestParseDate(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "RFC1123Z format",
			input:   "Mon, 01 Jan 2024 12:00:00 -0500",
			wantErr: false,
		},
		{
			name:    "RFC1123 format",
			input:   "Mon, 01 Jan 2024 12:00:00 UTC",
			wantErr: false,
		},
		{
			name:    "RFC822Z format",
			input:   "01 Jan 24 12:00 -0500",
			wantErr: false,
		},
		{
			name:    "custom format with day name",
			input:   "Mon, 1 Jan 2024 12:00:00 -0500",
			wantErr: false,
		},
		{
			name:    "empty string returns zero time",
			input:   "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseDate(tt.input)
			if tt.wantErr {
				if !got.IsZero() {
					t.Errorf("parseDate(%q) = %v, want zero time", tt.input, got)
				}
			} else {
				if got.IsZero() {
					t.Errorf("parseDate(%q) returned zero time", tt.input)
				}
				if got.Year() != 2024 {
					t.Errorf("parseDate(%q).Year() = %d, want 2024", tt.input, got.Year())
				}
			}
		})
	}
}

func TestDecodeBase64URL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"simple text", "SGVsbG8gV29ybGQ", "Hello World"},
		{"empty", "", ""},
		{"with special chars", "SGVsbG8rV29ybGQ", "Hello+World"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decodeBase64URL(tt.input)
			if got != tt.want {
				t.Errorf("decodeBase64URL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestExtractBody(t *testing.T) {
	tests := []struct {
		name     string
		payload  *gmailapi.MessagePart
		wantText string
		wantHTML string
	}{
		{
			name: "plain text body",
			payload: &gmailapi.MessagePart{
				MimeType: "text/plain",
				Body:     &gmailapi.MessagePartBody{Data: "SGVsbG8"},
			},
			wantText: "Hello",
			wantHTML: "",
		},
		{
			name: "html body",
			payload: &gmailapi.MessagePart{
				MimeType: "text/html",
				Body:     &gmailapi.MessagePartBody{Data: "PGI-SGk8L2I-"},
			},
			wantText: "",
			wantHTML: "<b>Hi</b>",
		},
		{
			name: "multipart with text and html",
			payload: &gmailapi.MessagePart{
				MimeType: "multipart/alternative",
				Parts: []*gmailapi.MessagePart{
					{
						MimeType: "text/plain",
						Body:     &gmailapi.MessagePartBody{Data: "SGVsbG8"},
					},
					{
						MimeType: "text/html",
						Body:     &gmailapi.MessagePartBody{Data: "PGI-SGk8L2I-"},
					},
				},
			},
			wantText: "Hello",
			wantHTML: "<b>Hi</b>",
		},
		{
			name:     "nil payload",
			payload:  nil,
			wantText: "",
			wantHTML: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text, html := extractBody(tt.payload)
			if text != tt.wantText {
				t.Errorf("extractBody() text = %q, want %q", text, tt.wantText)
			}
			if html != tt.wantHTML {
				t.Errorf("extractBody() html = %q, want %q", html, tt.wantHTML)
			}
		})
	}
}

func TestMapMessage(t *testing.T) {
	msg := &gmailapi.Message{
		Id:       "msg123",
		ThreadId: "thread456",
		LabelIds: []string{"INBOX", "STARRED"},
		Payload: &gmailapi.MessagePart{
			MimeType: "text/plain",
			Headers: []*gmailapi.MessagePartHeader{
				{Name: "From", Value: "Alice <alice@example.com>"},
				{Name: "To", Value: "Bob <bob@example.com>"},
				{Name: "Subject", Value: "Test Subject"},
				{Name: "Date", Value: "Mon, 01 Jan 2024 12:00:00 +0000"},
				{Name: "In-Reply-To", Value: "<ref123@example.com>"},
			},
			Body: &gmailapi.MessagePartBody{Data: "SGVsbG8"},
		},
	}

	email := mapMessage(msg)
	if email.ID != "msg123" {
		t.Errorf("ID = %q, want %q", email.ID, "msg123")
	}
	if email.ThreadID != "thread456" {
		t.Errorf("ThreadID = %q, want %q", email.ThreadID, "thread456")
	}
	if email.Subject != "Test Subject" {
		t.Errorf("Subject = %q, want %q", email.Subject, "Test Subject")
	}
	if email.From.Name != "Alice" {
		t.Errorf("From.Name = %q, want %q", email.From.Name, "Alice")
	}
	if email.From.Email != "alice@example.com" {
		t.Errorf("From.Email = %q, want %q", email.From.Email, "alice@example.com")
	}
	if len(email.To) != 1 || email.To[0].Email != "bob@example.com" {
		t.Errorf("To = %v, want [bob@example.com]", email.To)
	}
	if !email.IsRead {
		t.Error("expected IsRead = true (UNREAD label absent)")
	}
	if !email.IsStarred {
		t.Error("expected IsStarred = true")
	}
	if email.Body != "Hello" {
		t.Errorf("Body = %q, want %q", email.Body, "Hello")
	}
	if email.InReplyTo != "<ref123@example.com>" {
		t.Errorf("InReplyTo = %q, want %q", email.InReplyTo, "<ref123@example.com>")
	}
	if email.Date.Year() != 2024 {
		t.Errorf("Date.Year() = %d, want 2024", email.Date.Year())
	}
}

func TestMapMessage_IsRead(t *testing.T) {
	// UNREAD label present means IsRead = false
	msg := &gmailapi.Message{
		Id:       "msg1",
		LabelIds: []string{"INBOX", "UNREAD"},
		Payload: &gmailapi.MessagePart{
			MimeType: "text/plain",
			Headers:  []*gmailapi.MessagePartHeader{},
			Body:     &gmailapi.MessagePartBody{},
		},
	}
	email := mapMessage(msg)
	if email.IsRead {
		t.Error("expected IsRead = false when UNREAD label present")
	}

	// No UNREAD label means IsRead = true
	msg.LabelIds = []string{"INBOX"}
	email = mapMessage(msg)
	if !email.IsRead {
		t.Error("expected IsRead = true when UNREAD label absent")
	}
}

func TestMapMessage_Attachments(t *testing.T) {
	msg := &gmailapi.Message{
		Id: "msg1",
		Payload: &gmailapi.MessagePart{
			MimeType: "multipart/mixed",
			Headers:  []*gmailapi.MessagePartHeader{},
			Parts: []*gmailapi.MessagePart{
				{
					MimeType: "text/plain",
					Body:     &gmailapi.MessagePartBody{Data: "SGVsbG8"},
				},
				{
					MimeType: "application/pdf",
					Filename: "doc.pdf",
					Body: &gmailapi.MessagePartBody{
						AttachmentId: "att123",
						Size:         1024,
					},
				},
			},
		},
	}
	email := mapMessage(msg)
	if len(email.Attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(email.Attachments))
	}
	att := email.Attachments[0]
	if att.ID != "att123" {
		t.Errorf("Attachment.ID = %q, want %q", att.ID, "att123")
	}
	if att.Filename != "doc.pdf" {
		t.Errorf("Attachment.Filename = %q, want %q", att.Filename, "doc.pdf")
	}
	if att.MIMEType != "application/pdf" {
		t.Errorf("Attachment.MIMEType = %q, want %q", att.MIMEType, "application/pdf")
	}
	if att.Size != 1024 {
		t.Errorf("Attachment.Size = %d, want 1024", att.Size)
	}
}

func TestParseDate_Consistency(t *testing.T) {
	// Both formats should parse to the same point in time
	rfc1123z := "Mon, 01 Jan 2024 12:00:00 +0000"
	custom := "Mon, 1 Jan 2024 12:00:00 +0000"

	t1 := parseDate(rfc1123z)
	t2 := parseDate(custom)

	if !t1.Equal(t2) {
		t.Errorf("same datetime in different formats should be equal: %v vs %v", t1, t2)
	}
}

func TestParseAddressList_Content(t *testing.T) {
	input := "Alice <alice@example.com>, bob@example.com"
	addrs := parseAddressList(input)
	if len(addrs) != 2 {
		t.Fatalf("expected 2 addresses, got %d", len(addrs))
	}
	if addrs[0].Name != "Alice" || addrs[0].Email != "alice@example.com" {
		t.Errorf("first address = %+v, want Alice <alice@example.com>", addrs[0])
	}
	if addrs[1].Email != "bob@example.com" {
		t.Errorf("second address email = %q, want %q", addrs[1].Email, "bob@example.com")
	}
}

// Verify zero-time for unparseable date
func TestParseDate_Invalid(t *testing.T) {
	got := parseDate("not a date")
	if !got.IsZero() {
		t.Errorf("parseDate(invalid) = %v, want zero time", got)
	}
}

// Verify the function returns a valid time (not just non-zero)
func TestParseDate_Precision(t *testing.T) {
	got := parseDate("Mon, 01 Jan 2024 15:30:45 -0500")
	expected := time.Date(2024, 1, 1, 15, 30, 45, 0, time.FixedZone("", -5*60*60))
	if !got.Equal(expected) {
		t.Errorf("parseDate() = %v, want %v", got, expected)
	}
}
