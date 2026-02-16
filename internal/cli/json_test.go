package cli

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/lu-zhengda/termail/internal/domain"
)

func TestToJSONAccounts(t *testing.T) {
	accounts := []domain.Account{
		{
			ID:        "user@example.com",
			Email:     "user@example.com",
			Provider:  "gmail",
			CreatedAt: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			ID:        "other@example.com",
			Email:     "other@example.com",
			Provider:  "gmail",
			CreatedAt: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	got := toJSONAccounts(accounts)

	if len(got) != 2 {
		t.Fatalf("got %d accounts, want 2", len(got))
	}
	if got[0].ID != "user@example.com" {
		t.Errorf("got ID %q, want %q", got[0].ID, "user@example.com")
	}
	if got[0].Provider != "gmail" {
		t.Errorf("got provider %q, want %q", got[0].Provider, "gmail")
	}
	if got[0].CreatedAt != "2025-01-15" {
		t.Errorf("got created_at %q, want %q", got[0].CreatedAt, "2025-01-15")
	}

	// Verify JSON round-trip.
	var buf bytes.Buffer
	if err := fprintJSON(&buf, got); err != nil {
		t.Fatalf("fprintJSON() error = %v", err)
	}
	var parsed []jsonAccount
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if len(parsed) != 2 {
		t.Fatalf("round-trip: got %d accounts, want 2", len(parsed))
	}
	if parsed[1].Email != "other@example.com" {
		t.Errorf("round-trip: got email %q, want %q", parsed[1].Email, "other@example.com")
	}
}

func TestToJSONAccounts_Empty(t *testing.T) {
	got := toJSONAccounts(nil)
	if len(got) != 0 {
		t.Errorf("got %d accounts for nil input, want 0", len(got))
	}

	var buf bytes.Buffer
	if err := fprintJSON(&buf, got); err != nil {
		t.Fatalf("fprintJSON() error = %v", err)
	}
	if got := buf.String(); got != "[]\n" {
		t.Errorf("got %q, want %q", got, "[]\n")
	}
}

func TestToJSONThreads(t *testing.T) {
	threads := []domain.Thread{
		{
			ID:          "thread-1",
			Subject:     "Hello World",
			FromAddress: domain.Address{Name: "Alice", Email: "alice@example.com"},
			LastDate:    time.Date(2025, 3, 10, 14, 30, 0, 0, time.UTC),
			TotalCount:  3,
			HasUnread:   true,
			Snippet:     "Hey there...",
			Labels:      []string{"INBOX", "STARRED"},
		},
		{
			ID:          "thread-2",
			Subject:     "Meeting Notes",
			FromAddress: domain.Address{Email: "bob@example.com"},
			LastDate:    time.Date(2025, 3, 11, 9, 0, 0, 0, time.UTC),
			TotalCount:  1,
			HasUnread:   false,
		},
	}

	got := toJSONThreads(threads)

	if len(got) != 2 {
		t.Fatalf("got %d threads, want 2", len(got))
	}
	if got[0].ID != "thread-1" {
		t.Errorf("got ID %q, want %q", got[0].ID, "thread-1")
	}
	if got[0].From.Name != "Alice" {
		t.Errorf("got from name %q, want %q", got[0].From.Name, "Alice")
	}
	if got[0].MessageCount != 3 {
		t.Errorf("got message_count %d, want 3", got[0].MessageCount)
	}
	if !got[0].HasUnread {
		t.Error("got has_unread=false, want true")
	}
	if got[0].Snippet != "Hey there..." {
		t.Errorf("got snippet %q, want %q", got[0].Snippet, "Hey there...")
	}
	if got[1].From.Email != "bob@example.com" {
		t.Errorf("got from email %q, want %q", got[1].From.Email, "bob@example.com")
	}

	// Verify JSON round-trip.
	var buf bytes.Buffer
	if err := fprintJSON(&buf, got); err != nil {
		t.Fatalf("fprintJSON() error = %v", err)
	}
	var parsed []jsonThread
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if parsed[0].Subject != "Hello World" {
		t.Errorf("round-trip: got subject %q, want %q", parsed[0].Subject, "Hello World")
	}
}

func TestToJSONThreadDetail(t *testing.T) {
	thread := &domain.Thread{
		ID:      "thread-abc",
		Subject: "Test Thread",
		Messages: []domain.Email{
			{
				ID:       "msg-1",
				ThreadID: "thread-abc",
				From:     domain.Address{Name: "Sender", Email: "sender@example.com"},
				To:       []domain.Address{{Email: "receiver@example.com"}},
				CC:       []domain.Address{{Name: "CC Person", Email: "cc@example.com"}},
				Subject:  "Test Thread",
				Body:     "Hello, this is a test.",
				Date:     time.Date(2025, 3, 10, 14, 0, 0, 0, time.UTC),
				IsRead:   true,
				Labels:   []string{"INBOX"},
			},
			{
				ID:        "msg-2",
				ThreadID:  "thread-abc",
				From:      domain.Address{Email: "receiver@example.com"},
				To:        []domain.Address{{Name: "Sender", Email: "sender@example.com"}},
				Subject:   "Re: Test Thread",
				Body:      "Thanks for the test!",
				Date:      time.Date(2025, 3, 10, 15, 0, 0, 0, time.UTC),
				IsRead:    false,
				IsStarred: true,
			},
		},
	}

	got := toJSONThreadDetail(thread)

	if got.ID != "thread-abc" {
		t.Errorf("got ID %q, want %q", got.ID, "thread-abc")
	}
	if got.Subject != "Test Thread" {
		t.Errorf("got subject %q, want %q", got.Subject, "Test Thread")
	}
	if len(got.Messages) != 2 {
		t.Fatalf("got %d messages, want 2", len(got.Messages))
	}
	if got.Messages[0].From.Name != "Sender" {
		t.Errorf("got from name %q, want %q", got.Messages[0].From.Name, "Sender")
	}
	if got.Messages[0].Body != "Hello, this is a test." {
		t.Errorf("got body %q, want %q", got.Messages[0].Body, "Hello, this is a test.")
	}
	if !got.Messages[0].IsRead {
		t.Error("got is_read=false for msg-1, want true")
	}
	if !got.Messages[1].IsStarred {
		t.Error("got is_starred=false for msg-2, want true")
	}
	if len(got.Messages[0].CC) != 1 {
		t.Errorf("got %d CC addresses, want 1", len(got.Messages[0].CC))
	}

	// Verify JSON round-trip.
	var buf bytes.Buffer
	if err := fprintJSON(&buf, got); err != nil {
		t.Fatalf("fprintJSON() error = %v", err)
	}
	var parsed jsonThreadDetail
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if len(parsed.Messages) != 2 {
		t.Fatalf("round-trip: got %d messages, want 2", len(parsed.Messages))
	}
	if parsed.Messages[1].Subject != "Re: Test Thread" {
		t.Errorf("round-trip: got subject %q, want %q", parsed.Messages[1].Subject, "Re: Test Thread")
	}
}

func TestToJSONEmails(t *testing.T) {
	emails := []domain.Email{
		{
			ID:      "email-1",
			From:    domain.Address{Name: "Alice", Email: "alice@example.com"},
			Subject: "Search Result 1",
			Date:    time.Date(2025, 3, 10, 12, 0, 0, 0, time.UTC),
		},
	}

	got := toJSONEmails(emails)

	if len(got) != 1 {
		t.Fatalf("got %d emails, want 1", len(got))
	}
	if got[0].ID != "email-1" {
		t.Errorf("got ID %q, want %q", got[0].ID, "email-1")
	}
	if got[0].From.Name != "Alice" {
		t.Errorf("got from name %q, want %q", got[0].From.Name, "Alice")
	}

	// Verify JSON round-trip.
	var buf bytes.Buffer
	if err := fprintJSON(&buf, got); err != nil {
		t.Fatalf("fprintJSON() error = %v", err)
	}
	var parsed []jsonEmail
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if parsed[0].Subject != "Search Result 1" {
		t.Errorf("round-trip: got subject %q, want %q", parsed[0].Subject, "Search Result 1")
	}
}

func TestToJSONEmails_Empty(t *testing.T) {
	got := toJSONEmails(nil)
	if len(got) != 0 {
		t.Errorf("got %d emails for nil input, want 0", len(got))
	}

	var buf bytes.Buffer
	if err := fprintJSON(&buf, got); err != nil {
		t.Fatalf("fprintJSON() error = %v", err)
	}
	if got := buf.String(); got != "[]\n" {
		t.Errorf("got %q, want %q", got, "[]\n")
	}
}

func TestToJSONLabels(t *testing.T) {
	labels := []domain.Label{
		{ID: "INBOX", Name: "Inbox", Type: domain.LabelTypeSystem},
		{ID: "Label_1", Name: "Work", Type: domain.LabelTypeUser},
	}

	got := toJSONLabels(labels)

	if len(got) != 2 {
		t.Fatalf("got %d labels, want 2", len(got))
	}
	if got[0].ID != "INBOX" {
		t.Errorf("got ID %q, want %q", got[0].ID, "INBOX")
	}
	if got[0].Type != "system" {
		t.Errorf("got type %q, want %q", got[0].Type, "system")
	}
	if got[1].Name != "Work" {
		t.Errorf("got name %q, want %q", got[1].Name, "Work")
	}
	if got[1].Type != "user" {
		t.Errorf("got type %q, want %q", got[1].Type, "user")
	}

	// Verify JSON round-trip.
	var buf bytes.Buffer
	if err := fprintJSON(&buf, got); err != nil {
		t.Fatalf("fprintJSON() error = %v", err)
	}
	var parsed []jsonLabel
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if parsed[0].Name != "Inbox" {
		t.Errorf("round-trip: got name %q, want %q", parsed[0].Name, "Inbox")
	}
}

func TestToJSONAddresses(t *testing.T) {
	t.Run("with addresses", func(t *testing.T) {
		addrs := []domain.Address{
			{Name: "Alice", Email: "alice@example.com"},
			{Email: "bob@example.com"},
		}

		got := toJSONAddresses(addrs)

		if len(got) != 2 {
			t.Fatalf("got %d addresses, want 2", len(got))
		}
		if got[0].Name != "Alice" {
			t.Errorf("got name %q, want %q", got[0].Name, "Alice")
		}
		if got[1].Name != "" {
			t.Errorf("got name %q, want empty", got[1].Name)
		}

		// Verify name is omitted from JSON when empty.
		var buf bytes.Buffer
		if err := fprintJSON(&buf, got[1]); err != nil {
			t.Fatalf("fprintJSON() error = %v", err)
		}
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(buf.Bytes(), &raw); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}
		if _, ok := raw["name"]; ok {
			t.Error("name field should be omitted when empty")
		}
	})

	t.Run("nil input", func(t *testing.T) {
		got := toJSONAddresses(nil)
		if got != nil {
			t.Errorf("got %v for nil input, want nil", got)
		}
	})

	t.Run("empty input", func(t *testing.T) {
		got := toJSONAddresses([]domain.Address{})
		if got != nil {
			t.Errorf("got %v for empty input, want nil", got)
		}
	})
}

func TestJSONAction_RoundTrip(t *testing.T) {
	actions := []struct {
		name   string
		input  jsonAction
	}{
		{
			name:  "compose",
			input: jsonAction{OK: true, Action: "compose"},
		},
		{
			name:  "reply",
			input: jsonAction{OK: true, Action: "reply", MessageID: "msg-123"},
		},
		{
			name:  "forward",
			input: jsonAction{OK: true, Action: "forward", MessageID: "msg-456"},
		},
		{
			name:  "archive",
			input: jsonAction{OK: true, Action: "archive", MessageID: "msg-789"},
		},
		{
			name:  "trash",
			input: jsonAction{OK: true, Action: "trash", MessageID: "msg-abc"},
		},
		{
			name:  "star",
			input: jsonAction{OK: true, Action: "star", MessageID: "msg-def"},
		},
		{
			name:  "unstar",
			input: jsonAction{OK: true, Action: "unstar", MessageID: "msg-ghi"},
		},
		{
			name:  "mark-read",
			input: jsonAction{OK: true, Action: "mark-read", MessageID: "msg-jkl"},
		},
		{
			name:  "mark-unread",
			input: jsonAction{OK: true, Action: "mark-unread", MessageID: "msg-mno"},
		},
		{
			name:  "label-modify",
			input: jsonAction{OK: true, Action: "label-modify", MessageID: "msg-pqr"},
		},
		{
			name:  "add account",
			input: jsonAction{OK: true, Action: "add", Email: "user@example.com"},
		},
		{
			name:  "remove account",
			input: jsonAction{OK: true, Action: "remove", Email: "user@example.com"},
		},
		{
			name:  "sync",
			input: jsonAction{OK: true, Action: "sync", AccountID: "user@example.com"},
		},
	}

	for _, tc := range actions {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := fprintJSON(&buf, tc.input); err != nil {
				t.Fatalf("fprintJSON() error = %v", err)
			}

			var got jsonAction
			if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
				t.Fatalf("failed to parse JSON: %v", err)
			}
			if !got.OK {
				t.Error("got ok=false, want true")
			}
			if got.Action != tc.input.Action {
				t.Errorf("got action %q, want %q", got.Action, tc.input.Action)
			}
			if got.MessageID != tc.input.MessageID {
				t.Errorf("got message_id %q, want %q", got.MessageID, tc.input.MessageID)
			}
			if got.Email != tc.input.Email {
				t.Errorf("got email %q, want %q", got.Email, tc.input.Email)
			}
			if got.AccountID != tc.input.AccountID {
				t.Errorf("got account_id %q, want %q", got.AccountID, tc.input.AccountID)
			}
		})
	}
}

func TestJSONAction_OmitsEmpty(t *testing.T) {
	input := jsonAction{OK: true, Action: "compose"}

	var buf bytes.Buffer
	if err := fprintJSON(&buf, input); err != nil {
		t.Fatalf("fprintJSON() error = %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(buf.Bytes(), &raw); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	omittedFields := []string{"message_id", "email", "account_id"}
	for _, field := range omittedFields {
		if _, ok := raw[field]; ok {
			t.Errorf("field %q should be omitted when empty, got %s", field, string(raw[field]))
		}
	}

	requiredFields := []string{"ok", "action"}
	for _, field := range requiredFields {
		if _, ok := raw[field]; !ok {
			t.Errorf("field %q should always be present", field)
		}
	}
}

func TestToJSONThreadDetail_EmptyMessages(t *testing.T) {
	thread := &domain.Thread{
		ID:      "thread-empty",
		Subject: "No Messages",
	}

	got := toJSONThreadDetail(thread)

	if got.ID != "thread-empty" {
		t.Errorf("got ID %q, want %q", got.ID, "thread-empty")
	}
	if len(got.Messages) != 0 {
		t.Errorf("got %d messages, want 0", len(got.Messages))
	}

	// Verify JSON output contains empty array, not null.
	var buf bytes.Buffer
	if err := fprintJSON(&buf, got); err != nil {
		t.Fatalf("fprintJSON() error = %v", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(buf.Bytes(), &raw); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if string(raw["messages"]) != "[]" {
		t.Errorf("got messages %s, want []", string(raw["messages"]))
	}
}

func TestToJSONThreads_OmitsEmpty(t *testing.T) {
	threads := []domain.Thread{
		{
			ID:          "thread-minimal",
			Subject:     "Minimal",
			FromAddress: domain.Address{Email: "test@example.com"},
			LastDate:    time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			TotalCount:  1,
		},
	}

	got := toJSONThreads(threads)

	var buf bytes.Buffer
	if err := fprintJSON(&buf, got[0]); err != nil {
		t.Fatalf("fprintJSON() error = %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(buf.Bytes(), &raw); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	// snippet and labels should be omitted when empty.
	if _, ok := raw["snippet"]; ok {
		t.Errorf("snippet should be omitted when empty")
	}
	if _, ok := raw["labels"]; ok {
		t.Errorf("labels should be omitted when empty")
	}

	// These should always be present.
	requiredFields := []string{"id", "subject", "from", "last_date", "message_count", "has_unread"}
	for _, field := range requiredFields {
		if _, ok := raw[field]; !ok {
			t.Errorf("field %q should always be present", field)
		}
	}
}
