package sqlite

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/zhengda-lu/termail/internal/domain"
	"github.com/zhengda-lu/termail/internal/store"
)

func TestSearchEmails(t *testing.T) {
	db := newTestDB(t)
	seedAccount(t, db)
	ctx := context.Background()

	emails := []domain.Email{
		{ID: "m1", ThreadID: "t1", From: domain.Address{Email: "alice@test.com", Name: "Alice"},
			Subject: "Project deadline", Body: "The project deadline is Friday", Date: time.Now()},
		{ID: "m2", ThreadID: "t2", From: domain.Address{Email: "bob@test.com", Name: "Bob"},
			Subject: "Lunch plans", Body: "Want to grab lunch?", Date: time.Now()},
	}
	for i := range emails {
		if err := db.UpsertEmail(ctx, &emails[i], "acc-1"); err != nil {
			t.Fatalf("UpsertEmail(%d) error: %v", i, err)
		}
	}

	results, err := db.SearchEmails(ctx, "project", "acc-1")
	if err != nil {
		t.Fatalf("SearchEmails() error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].ID != "m1" {
		t.Errorf("got ID %q, want %q", results[0].ID, "m1")
	}
}

func TestSearchEmails_NoResults(t *testing.T) {
	db := newTestDB(t)
	seedAccount(t, db)
	ctx := context.Background()

	email := &domain.Email{
		ID:       "m1",
		ThreadID: "t1",
		From:     domain.Address{Email: "alice@test.com", Name: "Alice"},
		Subject:  "Hello world",
		Body:     "This is a test email",
		Date:     time.Now(),
	}
	if err := db.UpsertEmail(ctx, email, "acc-1"); err != nil {
		t.Fatalf("UpsertEmail() error: %v", err)
	}

	results, err := db.SearchEmails(ctx, "nonexistent", "acc-1")
	if err != nil {
		t.Fatalf("SearchEmails() error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("got %d results, want 0", len(results))
	}
}

func TestGetThread(t *testing.T) {
	db := newTestDB(t)
	seedAccount(t, db)
	ctx := context.Background()

	baseDate := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)

	// Insert 2 emails with the same thread_id
	emails := []domain.Email{
		{
			ID:       "m1",
			ThreadID: "thread-1",
			From:     domain.Address{Email: "alice@test.com", Name: "Alice"},
			Subject:  "Thread subject",
			Body:     "First message in thread",
			Date:     baseDate,
		},
		{
			ID:        "m2",
			ThreadID:  "thread-1",
			From:      domain.Address{Email: "bob@test.com", Name: "Bob"},
			Subject:   "Re: Thread subject",
			Body:      "Second message - a reply that is somewhat longer than one hundred characters to test truncation behavior properly",
			Date:      baseDate.Add(time.Hour),
			InReplyTo: "m1",
		},
	}
	for i := range emails {
		if err := db.UpsertEmail(ctx, &emails[i], "acc-1"); err != nil {
			t.Fatalf("UpsertEmail(%d) error: %v", i, err)
		}
	}

	thread, err := db.GetThread(ctx, "thread-1", "acc-1")
	if err != nil {
		t.Fatalf("GetThread() error: %v", err)
	}

	if thread.ID != "thread-1" {
		t.Errorf("ID = %q, want %q", thread.ID, "thread-1")
	}
	if thread.Subject != "Thread subject" {
		t.Errorf("Subject = %q, want %q", thread.Subject, "Thread subject")
	}
	if len(thread.Messages) != 2 {
		t.Fatalf("Messages count = %d, want 2", len(thread.Messages))
	}
	// Verify ordered by date ascending
	if thread.Messages[0].ID != "m1" {
		t.Errorf("Messages[0].ID = %q, want %q", thread.Messages[0].ID, "m1")
	}
	if thread.Messages[1].ID != "m2" {
		t.Errorf("Messages[1].ID = %q, want %q", thread.Messages[1].ID, "m2")
	}
	// LastDate should be the date of the last message
	if !thread.LastDate.Equal(baseDate.Add(time.Hour)) {
		t.Errorf("LastDate = %v, want %v", thread.LastDate, baseDate.Add(time.Hour))
	}
	// Snippet should be truncated to 100 chars
	if len(thread.Snippet) > 100 {
		t.Errorf("Snippet length = %d, want <= 100", len(thread.Snippet))
	}
}

func TestGetThread_NotFound(t *testing.T) {
	db := newTestDB(t)
	seedAccount(t, db)
	ctx := context.Background()

	_, err := db.GetThread(ctx, "nonexistent", "acc-1")
	if err == nil {
		t.Fatal("GetThread() expected error for nonexistent thread, got nil")
	}
}

func TestListThreads(t *testing.T) {
	db := newTestDB(t)
	seedAccount(t, db)
	ctx := context.Background()

	baseDate := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)

	// Thread 1: 2 messages
	for i := 0; i < 2; i++ {
		email := &domain.Email{
			ID:       fmt.Sprintf("t1-m%d", i),
			ThreadID: "thread-1",
			From:     domain.Address{Email: "alice@test.com", Name: "Alice"},
			Subject:  "First thread",
			Body:     fmt.Sprintf("Message %d of thread 1", i),
			Date:     baseDate.Add(time.Duration(i) * time.Hour),
			Labels:   []string{"INBOX"},
		}
		if err := db.UpsertEmail(ctx, email, "acc-1"); err != nil {
			t.Fatalf("UpsertEmail() error: %v", err)
		}
	}

	// Thread 2: 1 message, later date
	email := &domain.Email{
		ID:       "t2-m0",
		ThreadID: "thread-2",
		From:     domain.Address{Email: "bob@test.com", Name: "Bob"},
		Subject:  "Second thread",
		Body:     "Only message in thread 2",
		Date:     baseDate.Add(3 * time.Hour),
		Labels:   []string{"INBOX"},
	}
	if err := db.UpsertEmail(ctx, email, "acc-1"); err != nil {
		t.Fatalf("UpsertEmail() error: %v", err)
	}

	threads, err := db.ListThreads(ctx, store.ListEmailOptions{
		AccountID: "acc-1",
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("ListThreads() error: %v", err)
	}
	if len(threads) != 2 {
		t.Fatalf("got %d threads, want 2", len(threads))
	}

	// Ordered by last_date DESC, so thread-2 should be first
	if threads[0].ID != "thread-2" {
		t.Errorf("threads[0].ID = %q, want %q", threads[0].ID, "thread-2")
	}
	if threads[1].ID != "thread-1" {
		t.Errorf("threads[1].ID = %q, want %q", threads[1].ID, "thread-1")
	}

	// Verify message count (from Messages slice populated by ListThreads)
	if threads[0].Subject != "Second thread" {
		t.Errorf("threads[0].Subject = %q, want %q", threads[0].Subject, "Second thread")
	}
	if threads[1].Subject != "First thread" {
		t.Errorf("threads[1].Subject = %q, want %q", threads[1].Subject, "First thread")
	}
}

func TestListThreads_ByLabel(t *testing.T) {
	db := newTestDB(t)
	seedAccount(t, db)
	ctx := context.Background()

	baseDate := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)

	// Thread 1: INBOX label
	email1 := &domain.Email{
		ID:       "m1",
		ThreadID: "thread-1",
		From:     domain.Address{Email: "alice@test.com", Name: "Alice"},
		Subject:  "Inbox thread",
		Body:     "In the inbox",
		Date:     baseDate,
		Labels:   []string{"INBOX"},
	}
	if err := db.UpsertEmail(ctx, email1, "acc-1"); err != nil {
		t.Fatalf("UpsertEmail() error: %v", err)
	}

	// Thread 2: STARRED label only
	email2 := &domain.Email{
		ID:       "m2",
		ThreadID: "thread-2",
		From:     domain.Address{Email: "bob@test.com", Name: "Bob"},
		Subject:  "Starred thread",
		Body:     "Starred only",
		Date:     baseDate.Add(time.Hour),
		Labels:   []string{"STARRED"},
	}
	if err := db.UpsertEmail(ctx, email2, "acc-1"); err != nil {
		t.Fatalf("UpsertEmail() error: %v", err)
	}

	threads, err := db.ListThreads(ctx, store.ListEmailOptions{
		AccountID: "acc-1",
		LabelID:   "INBOX",
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("ListThreads(INBOX) error: %v", err)
	}
	if len(threads) != 1 {
		t.Fatalf("got %d threads, want 1", len(threads))
	}
	if threads[0].ID != "thread-1" {
		t.Errorf("threads[0].ID = %q, want %q", threads[0].ID, "thread-1")
	}
}

func TestListThreads_Limit(t *testing.T) {
	db := newTestDB(t)
	seedAccount(t, db)
	ctx := context.Background()

	baseDate := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)

	for i := 0; i < 5; i++ {
		email := &domain.Email{
			ID:       fmt.Sprintf("msg-%d", i),
			ThreadID: fmt.Sprintf("thread-%d", i),
			From:     domain.Address{Email: "alice@test.com", Name: "Alice"},
			Subject:  fmt.Sprintf("Thread %d", i),
			Body:     fmt.Sprintf("Body %d", i),
			Date:     baseDate.Add(time.Duration(i) * time.Hour),
		}
		if err := db.UpsertEmail(ctx, email, "acc-1"); err != nil {
			t.Fatalf("UpsertEmail(%d) error: %v", i, err)
		}
	}

	threads, err := db.ListThreads(ctx, store.ListEmailOptions{
		AccountID: "acc-1",
		Limit:     3,
	})
	if err != nil {
		t.Fatalf("ListThreads() error: %v", err)
	}
	if len(threads) != 3 {
		t.Errorf("got %d threads, want 3", len(threads))
	}
}

func TestSyncState(t *testing.T) {
	db := newTestDB(t)
	seedAccount(t, db)
	ctx := context.Background()

	// Get non-existent state should return empty SyncState with AccountID set
	state, err := db.GetSyncState(ctx, "acc-1")
	if err != nil {
		t.Fatalf("GetSyncState() error: %v", err)
	}
	if state.AccountID != "acc-1" {
		t.Errorf("AccountID = %q, want %q", state.AccountID, "acc-1")
	}
	if state.HistoryID != 0 {
		t.Errorf("HistoryID = %d, want 0", state.HistoryID)
	}
	if state.LastSync != 0 {
		t.Errorf("LastSync = %d, want 0", state.LastSync)
	}

	// Set state
	now := time.Now().Unix()
	if err := db.SetSyncState(ctx, &store.SyncState{
		AccountID: "acc-1",
		HistoryID: 12345,
		LastSync:  now,
	}); err != nil {
		t.Fatalf("SetSyncState() error: %v", err)
	}

	// Get state again
	state, err = db.GetSyncState(ctx, "acc-1")
	if err != nil {
		t.Fatalf("GetSyncState() after set error: %v", err)
	}
	if state.AccountID != "acc-1" {
		t.Errorf("AccountID = %q, want %q", state.AccountID, "acc-1")
	}
	if state.HistoryID != 12345 {
		t.Errorf("HistoryID = %d, want 12345", state.HistoryID)
	}
	if state.LastSync != now {
		t.Errorf("LastSync = %d, want %d", state.LastSync, now)
	}

	// Update state
	if err := db.SetSyncState(ctx, &store.SyncState{
		AccountID: "acc-1",
		HistoryID: 67890,
		LastSync:  now + 100,
	}); err != nil {
		t.Fatalf("SetSyncState() update error: %v", err)
	}

	state, err = db.GetSyncState(ctx, "acc-1")
	if err != nil {
		t.Fatalf("GetSyncState() after update error: %v", err)
	}
	if state.HistoryID != 67890 {
		t.Errorf("HistoryID = %d, want 67890", state.HistoryID)
	}
	if state.LastSync != now+100 {
		t.Errorf("LastSync = %d, want %d", state.LastSync, now+100)
	}
}
