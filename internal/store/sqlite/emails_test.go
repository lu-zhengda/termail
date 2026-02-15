package sqlite

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/lu-zhengda/termail/internal/domain"
	"github.com/lu-zhengda/termail/internal/store"
)

func seedAccount(t *testing.T, db *DB) {
	t.Helper()
	ctx := context.Background()
	if err := db.CreateAccount(ctx, &domain.Account{
		ID:       "acc-1",
		Email:    "test@gmail.com",
		Provider: "gmail",
	}); err != nil {
		t.Fatalf("seedAccount: %v", err)
	}
}

func TestUpsertAndGetEmail(t *testing.T) {
	db := newTestDB(t)
	seedAccount(t, db)
	ctx := context.Background()

	date := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
	email := &domain.Email{
		ID:       "msg-1",
		ThreadID: "thread-1",
		From:     domain.Address{Name: "Alice", Email: "alice@example.com"},
		To: []domain.Address{
			{Name: "Bob", Email: "bob@example.com"},
		},
		CC: []domain.Address{
			{Name: "Carol", Email: "carol@example.com"},
		},
		Subject:   "Hello World",
		Body:      "This is the body.",
		BodyHTML:  "<p>This is the body.</p>",
		Date:      date,
		Labels:    []string{"INBOX", "STARRED"},
		IsRead:    true,
		IsStarred: true,
		InReplyTo: "msg-0",
	}

	if err := db.UpsertEmail(ctx, email, "acc-1"); err != nil {
		t.Fatalf("UpsertEmail() error: %v", err)
	}

	got, err := db.GetEmail(ctx, "msg-1")
	if err != nil {
		t.Fatalf("GetEmail() error: %v", err)
	}

	if got.ID != "msg-1" {
		t.Errorf("ID = %q, want %q", got.ID, "msg-1")
	}
	if got.ThreadID != "thread-1" {
		t.Errorf("ThreadID = %q, want %q", got.ThreadID, "thread-1")
	}
	if got.From.Email != "alice@example.com" {
		t.Errorf("From.Email = %q, want %q", got.From.Email, "alice@example.com")
	}
	if got.From.Name != "Alice" {
		t.Errorf("From.Name = %q, want %q", got.From.Name, "Alice")
	}
	if len(got.To) != 1 || got.To[0].Email != "bob@example.com" {
		t.Errorf("To = %v, want [{Bob bob@example.com}]", got.To)
	}
	if len(got.CC) != 1 || got.CC[0].Email != "carol@example.com" {
		t.Errorf("CC = %v, want [{Carol carol@example.com}]", got.CC)
	}
	if got.Subject != "Hello World" {
		t.Errorf("Subject = %q, want %q", got.Subject, "Hello World")
	}
	if got.Body != "This is the body." {
		t.Errorf("Body = %q, want %q", got.Body, "This is the body.")
	}
	if got.BodyHTML != "<p>This is the body.</p>" {
		t.Errorf("BodyHTML = %q, want %q", got.BodyHTML, "<p>This is the body.</p>")
	}
	if !got.Date.Equal(date) {
		t.Errorf("Date = %v, want %v", got.Date, date)
	}
	if !got.IsRead {
		t.Error("IsRead = false, want true")
	}
	if !got.IsStarred {
		t.Error("IsStarred = false, want true")
	}
	if got.InReplyTo != "msg-0" {
		t.Errorf("InReplyTo = %q, want %q", got.InReplyTo, "msg-0")
	}
	if len(got.Labels) != 2 {
		t.Fatalf("Labels count = %d, want 2", len(got.Labels))
	}
}

func TestListEmails_ByLabel(t *testing.T) {
	db := newTestDB(t)
	seedAccount(t, db)
	ctx := context.Background()

	date := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)

	for i := 0; i < 3; i++ {
		email := &domain.Email{
			ID:       fmt.Sprintf("msg-%d", i),
			ThreadID: fmt.Sprintf("thread-%d", i),
			From:     domain.Address{Name: "Alice", Email: "alice@example.com"},
			Subject:  fmt.Sprintf("Subject %d", i),
			Date:     date.Add(time.Duration(i) * time.Hour),
			Labels:   []string{"INBOX"},
		}
		if err := db.UpsertEmail(ctx, email, "acc-1"); err != nil {
			t.Fatalf("UpsertEmail(%d) error: %v", i, err)
		}
	}

	// Add a STARRED label to only msg-1
	starredEmail := &domain.Email{
		ID:       "msg-special",
		ThreadID: "thread-special",
		From:     domain.Address{Name: "Bob", Email: "bob@example.com"},
		Subject:  "Special",
		Date:     date.Add(4 * time.Hour),
		Labels:   []string{"STARRED"},
	}
	if err := db.UpsertEmail(ctx, starredEmail, "acc-1"); err != nil {
		t.Fatalf("UpsertEmail(special) error: %v", err)
	}

	// List all INBOX emails
	emails, err := db.ListEmails(ctx, store.ListEmailOptions{
		AccountID: "acc-1",
		LabelID:   "INBOX",
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("ListEmails(INBOX) error: %v", err)
	}
	if len(emails) != 3 {
		t.Errorf("INBOX count = %d, want 3", len(emails))
	}

	// List STARRED emails
	emails, err = db.ListEmails(ctx, store.ListEmailOptions{
		AccountID: "acc-1",
		LabelID:   "STARRED",
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("ListEmails(STARRED) error: %v", err)
	}
	if len(emails) != 1 {
		t.Errorf("STARRED count = %d, want 1", len(emails))
	}
}

func TestUpsertEmail_UpdateExisting(t *testing.T) {
	db := newTestDB(t)
	seedAccount(t, db)
	ctx := context.Background()

	date := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)
	email := &domain.Email{
		ID:       "msg-1",
		ThreadID: "thread-1",
		From:     domain.Address{Name: "Alice", Email: "alice@example.com"},
		Subject:  "Original Subject",
		Date:     date,
		Labels:   []string{"INBOX"},
		IsRead:   false,
	}
	if err := db.UpsertEmail(ctx, email, "acc-1"); err != nil {
		t.Fatalf("UpsertEmail() first call error: %v", err)
	}

	// Update with same ID
	email.Subject = "Updated Subject"
	email.IsRead = true
	email.Labels = []string{"INBOX", "STARRED"}
	if err := db.UpsertEmail(ctx, email, "acc-1"); err != nil {
		t.Fatalf("UpsertEmail() second call error: %v", err)
	}

	got, err := db.GetEmail(ctx, "msg-1")
	if err != nil {
		t.Fatalf("GetEmail() error: %v", err)
	}
	if got.Subject != "Updated Subject" {
		t.Errorf("Subject = %q, want %q", got.Subject, "Updated Subject")
	}
	if !got.IsRead {
		t.Error("IsRead = false, want true after update")
	}
	if len(got.Labels) != 2 {
		t.Errorf("Labels count = %d, want 2 after update", len(got.Labels))
	}
}

func TestDeleteEmail(t *testing.T) {
	db := newTestDB(t)
	seedAccount(t, db)
	ctx := context.Background()

	email := &domain.Email{
		ID:       "msg-1",
		ThreadID: "thread-1",
		From:     domain.Address{Name: "Alice", Email: "alice@example.com"},
		Subject:  "To Delete",
		Date:     time.Now(),
		Labels:   []string{"INBOX"},
	}
	if err := db.UpsertEmail(ctx, email, "acc-1"); err != nil {
		t.Fatalf("UpsertEmail() error: %v", err)
	}

	if err := db.DeleteEmail(ctx, "msg-1"); err != nil {
		t.Fatalf("DeleteEmail() error: %v", err)
	}

	_, err := db.GetEmail(ctx, "msg-1")
	if err == nil {
		t.Fatal("GetEmail() after delete returned no error, want error")
	}
}

func TestSetEmailLabels(t *testing.T) {
	db := newTestDB(t)
	seedAccount(t, db)
	ctx := context.Background()

	email := &domain.Email{
		ID:       "msg-1",
		ThreadID: "thread-1",
		From:     domain.Address{Name: "Alice", Email: "alice@example.com"},
		Subject:  "Label Test",
		Date:     time.Now(),
		Labels:   []string{"INBOX"},
	}
	if err := db.UpsertEmail(ctx, email, "acc-1"); err != nil {
		t.Fatalf("UpsertEmail() error: %v", err)
	}

	// Set new labels
	if err := db.SetEmailLabels(ctx, "msg-1", []string{"INBOX", "STARRED", "IMPORTANT"}); err != nil {
		t.Fatalf("SetEmailLabels() error: %v", err)
	}

	got, err := db.GetEmail(ctx, "msg-1")
	if err != nil {
		t.Fatalf("GetEmail() error: %v", err)
	}
	if len(got.Labels) != 3 {
		t.Errorf("Labels count = %d, want 3", len(got.Labels))
	}

	// Change labels
	if err := db.SetEmailLabels(ctx, "msg-1", []string{"TRASH"}); err != nil {
		t.Fatalf("SetEmailLabels() second call error: %v", err)
	}

	got, err = db.GetEmail(ctx, "msg-1")
	if err != nil {
		t.Fatalf("GetEmail() after relabel error: %v", err)
	}
	if len(got.Labels) != 1 {
		t.Errorf("Labels count = %d, want 1 after relabel", len(got.Labels))
	}
	if got.Labels[0] != "TRASH" {
		t.Errorf("Labels[0] = %q, want %q", got.Labels[0], "TRASH")
	}
}
