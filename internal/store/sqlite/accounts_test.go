package sqlite

import (
	"context"
	"testing"

	"github.com/lu-zhengda/termail/internal/domain"
)

func newTestDB(t *testing.T) *DB {
	t.Helper()
	db, err := New(":memory:")
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestCreateAccount(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	acct := &domain.Account{
		ID:          "acc-1",
		Email:       "test@gmail.com",
		Provider:    "gmail",
		DisplayName: "Test User",
	}
	if err := db.CreateAccount(ctx, acct); err != nil {
		t.Fatalf("CreateAccount() error: %v", err)
	}

	got, err := db.GetAccount(ctx, "acc-1")
	if err != nil {
		t.Fatalf("GetAccount() error: %v", err)
	}
	if got.Email != "test@gmail.com" {
		t.Errorf("email = %q, want %q", got.Email, "test@gmail.com")
	}
	if got.DisplayName != "Test User" {
		t.Errorf("display_name = %q, want %q", got.DisplayName, "Test User")
	}
}

func TestListAccounts(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	db.CreateAccount(ctx, &domain.Account{ID: "a1", Email: "a@test.com", Provider: "gmail"})
	db.CreateAccount(ctx, &domain.Account{ID: "a2", Email: "b@test.com", Provider: "gmail"})

	accounts, err := db.ListAccounts(ctx)
	if err != nil {
		t.Fatalf("ListAccounts() error: %v", err)
	}
	if len(accounts) != 2 {
		t.Errorf("got %d accounts, want 2", len(accounts))
	}
}

func TestDeleteAccount(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	db.CreateAccount(ctx, &domain.Account{ID: "a1", Email: "a@test.com", Provider: "gmail"})
	if err := db.DeleteAccount(ctx, "a1"); err != nil {
		t.Fatalf("DeleteAccount() error: %v", err)
	}

	accounts, err := db.ListAccounts(ctx)
	if err != nil {
		t.Fatalf("ListAccounts() error: %v", err)
	}
	if len(accounts) != 0 {
		t.Errorf("got %d accounts after delete, want 0", len(accounts))
	}
}
