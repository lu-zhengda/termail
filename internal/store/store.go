package store

import (
	"context"

	"github.com/lu-zhengda/termail/internal/domain"
)

// Store defines the persistence interface for the application.
type Store interface {
	// Accounts
	CreateAccount(ctx context.Context, account *domain.Account) error
	GetAccount(ctx context.Context, id string) (*domain.Account, error)
	ListAccounts(ctx context.Context) ([]domain.Account, error)
	DeleteAccount(ctx context.Context, id string) error

	// Emails
	UpsertEmail(ctx context.Context, email *domain.Email, accountID string) error
	GetEmail(ctx context.Context, id string) (*domain.Email, error)
	ListEmails(ctx context.Context, opts ListEmailOptions) ([]domain.Email, error)
	DeleteEmail(ctx context.Context, id string) error
	SetEmailRead(ctx context.Context, emailID string, read bool) error
	SetThreadRead(ctx context.Context, threadID string, read bool) error

	// Labels
	UpsertLabel(ctx context.Context, label *domain.Label) error
	ListLabels(ctx context.Context, accountID string) ([]domain.Label, error)
	SetEmailLabels(ctx context.Context, emailID string, labelIDs []string) error

	// Threads
	GetThread(ctx context.Context, threadID string, accountID string) (*domain.Thread, error)
	ListThreads(ctx context.Context, opts ListEmailOptions) ([]domain.Thread, error)

	// Search
	SearchEmails(ctx context.Context, query string, accountID string) ([]domain.Email, error)

	// Sync state
	GetSyncState(ctx context.Context, accountID string) (*SyncState, error)
	SetSyncState(ctx context.Context, state *SyncState) error

	// Lifecycle
	Close() error
}

// ListEmailOptions configures email listing queries.
type ListEmailOptions struct {
	AccountID string
	LabelID   string
	Limit     int
	Offset    int
}

// SyncState tracks the synchronization progress for an account.
type SyncState struct {
	AccountID string
	HistoryID uint64
	LastSync  int64 // Unix timestamp
}
