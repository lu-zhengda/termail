# termail Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a terminal-based Gmail client with OAuth2 auth, local SQLite cache, and Bubble Tea TUI.

**Architecture:** Layered monolith — domain models at the core (zero deps), provider interface + SQLite store in infrastructure, application layer for sync/compose/search orchestration, Bubble Tea TUI on top.

**Tech Stack:** Go, Bubble Tea, Gmail REST API, OAuth2, SQLite (mattn/go-sqlite3), Cobra CLI, Lipgloss, go-keyring

**Design Doc:** `docs/plans/2026-02-14-termail-design.md`

---

## Phase 1: Project Scaffolding & Domain Layer

### Task 1: Initialize Go module and directory structure

**Files:**
- Create: `go.mod`
- Create: `cmd/termail/main.go`
- Create: `internal/domain/email.go`
- Create: `internal/domain/thread.go`
- Create: `internal/domain/account.go`
- Create: `internal/domain/label.go`

**Step 1: Initialize Go module**

Run: `go mod init github.com/zhengda-lu/termail`

**Step 2: Create entry point**

```go
// cmd/termail/main.go
package main

import "fmt"

func main() {
	fmt.Println("termail")
}
```

**Step 3: Verify it builds**

Run: `go build ./cmd/termail`
Expected: Binary compiles with no errors.

**Step 4: Commit**

```bash
git add go.mod cmd/
git commit -m "chore: initialize go module and entry point"
```

---

### Task 2: Define domain models

**Files:**
- Create: `internal/domain/email.go`
- Create: `internal/domain/thread.go`
- Create: `internal/domain/account.go`
- Create: `internal/domain/label.go`
- Test: `internal/domain/email_test.go`

**Step 1: Write domain model files**

```go
// internal/domain/email.go
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

// HasLabel reports whether the email has the given label.
func (e *Email) HasLabel(label string) bool {
	for _, l := range e.Labels {
		if l == label {
			return true
		}
	}
	return false
}
```

```go
// internal/domain/thread.go
package domain

import "time"

type Thread struct {
	ID       string
	Subject  string
	Messages []Email
	Labels   []string
	Snippet  string
	LastDate time.Time
}

// MessageCount returns the number of messages in the thread.
func (t *Thread) MessageCount() int {
	return len(t.Messages)
}

// IsUnread reports whether the thread contains any unread messages.
func (t *Thread) IsUnread() bool {
	for i := range t.Messages {
		if !t.Messages[i].IsRead {
			return true
		}
	}
	return false
}
```

```go
// internal/domain/account.go
package domain

import "time"

type Account struct {
	ID          string
	Email       string
	Provider    string
	DisplayName string
	CreatedAt   time.Time
}
```

```go
// internal/domain/label.go
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

// Standard Gmail system label IDs.
const (
	LabelInbox   = "INBOX"
	LabelStarred = "STARRED"
	LabelSent    = "SENT"
	LabelDraft   = "DRAFT"
	LabelTrash   = "TRASH"
	LabelSpam    = "SPAM"
)
```

**Step 2: Write tests for domain helpers**

```go
// internal/domain/email_test.go
package domain

import "testing"

func TestAddress_String(t *testing.T) {
	tests := []struct {
		name string
		addr Address
		want string
	}{
		{"with name", Address{Name: "John", Email: "john@example.com"}, "John <john@example.com>"},
		{"email only", Address{Email: "john@example.com"}, "john@example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.addr.String(); got != tt.want {
				t.Errorf("Address.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEmail_HasLabel(t *testing.T) {
	e := &Email{Labels: []string{"INBOX", "STARRED"}}
	if !e.HasLabel("INBOX") {
		t.Error("expected HasLabel(INBOX) = true")
	}
	if e.HasLabel("TRASH") {
		t.Error("expected HasLabel(TRASH) = false")
	}
}
```

**Step 3: Run tests**

Run: `go test -race ./internal/domain/...`
Expected: PASS

**Step 4: Commit**

```bash
git add internal/domain/
git commit -m "feat: add domain models (email, thread, account, label)"
```

---

## Phase 2: Config & CLI Skeleton

### Task 3: Add config loading

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`
- Modify: `go.mod` (add `github.com/BurntSushi/toml`)

**Step 1: Install dependency**

Run: `go get github.com/BurntSushi/toml`

**Step 2: Write failing test**

```go
// internal/config/config_test.go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Sync.Interval != "5m" {
		t.Errorf("default interval = %q, want %q", cfg.Sync.Interval, "5m")
	}
	if cfg.Sync.InitialCount != 500 {
		t.Errorf("default initial_count = %d, want 500", cfg.Sync.InitialCount)
	}
	if cfg.UI.DefaultView != "thread" {
		t.Errorf("default view = %q, want %q", cfg.UI.DefaultView, "thread")
	}
}

func TestLoad_FromFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")
	content := `
[sync]
interval = "10m"
initial_count = 100

[ui]
default_view = "flat"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Sync.Interval != "10m" {
		t.Errorf("interval = %q, want %q", cfg.Sync.Interval, "10m")
	}
	if cfg.UI.DefaultView != "flat" {
		t.Errorf("view = %q, want %q", cfg.UI.DefaultView, "flat")
	}
}
```

**Step 3: Run test to verify it fails**

Run: `go test -race ./internal/config/...`
Expected: FAIL (package doesn't exist yet)

**Step 4: Implement config**

```go
// internal/config/config.go
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Sync     SyncConfig     `toml:"sync"`
	UI       UIConfig       `toml:"ui"`
	Accounts AccountsConfig `toml:"accounts"`
}

type SyncConfig struct {
	Interval     string `toml:"interval"`
	InitialCount int    `toml:"initial_count"`
}

type UIConfig struct {
	DefaultView string `toml:"default_view"`
	Theme       string `toml:"theme"`
}

type AccountsConfig struct {
	Default string `toml:"default"`
}

func defaults() Config {
	return Config{
		Sync: SyncConfig{
			Interval:     "5m",
			InitialCount: 500,
		},
		UI: UIConfig{
			DefaultView: "thread",
			Theme:       "default",
		},
	}
}

// Load reads config from path. If path is empty, returns defaults.
func Load(path string) (*Config, error) {
	cfg := defaults()
	if path == "" {
		return &cfg, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &cfg, nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	return &cfg, nil
}

// ConfigDir returns the termail config directory path.
func ConfigDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "termail")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "termail")
}

// DataDir returns the termail data directory path.
func DataDir() string {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "termail")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "termail")
}
```

**Step 5: Run tests**

Run: `go test -race ./internal/config/...`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/config/ go.mod go.sum
git commit -m "feat: add config loading with TOML and XDG paths"
```

---

### Task 4: Set up Cobra CLI skeleton

**Files:**
- Create: `internal/cli/root.go`
- Create: `internal/cli/account.go`
- Modify: `cmd/termail/main.go`
- Modify: `go.mod` (add cobra)

**Step 1: Install dependency**

Run: `go get github.com/spf13/cobra`

**Step 2: Create root command**

```go
// internal/cli/root.go
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var cfgFile string

func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "termail",
		Short: "Terminal email client",
		Long:  "A terminal-based email client with Gmail support.",
		RunE: func(cmd *cobra.Command, args []string) error {
			// TUI launch will go here (Phase 9+)
			fmt.Println("termail TUI - coming soon")
			return nil
		},
	}
	root.PersistentFlags().StringVar(&cfgFile, "config", "", "config file path")
	root.AddCommand(newAccountCmd())
	root.AddCommand(newSyncCmd())
	return root
}

func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
```

```go
// internal/cli/account.go
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newAccountCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "account",
		Short: "Manage email accounts",
	}
	cmd.AddCommand(newAccountAddCmd())
	cmd.AddCommand(newAccountListCmd())
	cmd.AddCommand(newAccountRemoveCmd())
	return cmd
}

func newAccountAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add",
		Short: "Add a Gmail account via OAuth",
		RunE: func(cmd *cobra.Command, args []string) error {
			// OAuth flow will go here (Phase 5)
			fmt.Println("account add - coming soon")
			return nil
		},
	}
}

func newAccountListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List configured accounts",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("account list - coming soon")
			return nil
		},
	}
}

func newAccountRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove [email]",
		Short: "Remove an account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("account remove - coming soon")
			return nil
		},
	}
}

func newSyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Manually sync emails",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("sync - coming soon")
			return nil
		},
	}
}
```

**Step 3: Update main.go**

```go
// cmd/termail/main.go
package main

import "github.com/zhengda-lu/termail/internal/cli"

func main() {
	cli.Execute()
}
```

**Step 4: Verify it builds and runs**

Run: `go build ./cmd/termail && ./termail --help`
Expected: Shows help with `account`, `sync` subcommands.

Run: `./termail account --help`
Expected: Shows `add`, `list`, `remove` subcommands.

**Step 5: Commit**

```bash
git add cmd/ internal/cli/ go.mod go.sum
git commit -m "feat: add Cobra CLI skeleton with account and sync commands"
```

---

## Phase 3: SQLite Store

### Task 5: Set up SQLite connection and migrations

**Files:**
- Create: `internal/store/store.go`
- Create: `internal/store/sqlite/sqlite.go`
- Create: `internal/store/sqlite/migrations.go`
- Test: `internal/store/sqlite/sqlite_test.go`

**Step 1: Install dependency**

Run: `go get github.com/mattn/go-sqlite3`

**Step 2: Define store interface**

```go
// internal/store/store.go
package store

import (
	"context"

	"github.com/zhengda-lu/termail/internal/domain"
)

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

	// Labels
	UpsertLabel(ctx context.Context, label *domain.Label) error
	ListLabels(ctx context.Context, accountID string) ([]domain.Label, error)
	SetEmailLabels(ctx context.Context, emailID string, labelIDs []string) error

	// Threads
	GetThread(ctx context.Context, threadID string) (*domain.Thread, error)
	ListThreads(ctx context.Context, opts ListEmailOptions) ([]domain.Thread, error)

	// Search
	SearchEmails(ctx context.Context, query string, accountID string) ([]domain.Email, error)

	// Sync state
	GetSyncState(ctx context.Context, accountID string) (*SyncState, error)
	SetSyncState(ctx context.Context, state *SyncState) error

	// Lifecycle
	Close() error
}

type ListEmailOptions struct {
	AccountID string
	LabelID   string
	Limit     int
	Offset    int
}

type SyncState struct {
	AccountID string
	HistoryID uint64
	LastSync  int64 // Unix timestamp
}
```

**Step 3: Write failing test for DB open + migrations**

```go
// internal/store/sqlite/sqlite_test.go
package sqlite

import (
	"context"
	"testing"
)

func TestNew_CreatesTables(t *testing.T) {
	db, err := New(":memory:")
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer db.Close()

	// Verify tables exist by querying them
	ctx := context.Background()
	_, err = db.ListAccounts(ctx)
	if err != nil {
		t.Errorf("ListAccounts() error: %v", err)
	}
}
```

**Step 4: Run test to verify it fails**

Run: `go test -race ./internal/store/sqlite/...`
Expected: FAIL

**Step 5: Implement SQLite connection and migrations**

```go
// internal/store/sqlite/migrations.go
package sqlite

const schema = `
CREATE TABLE IF NOT EXISTS accounts (
    id          TEXT PRIMARY KEY,
    email       TEXT NOT NULL UNIQUE,
    provider    TEXT NOT NULL DEFAULT 'gmail',
    display_name TEXT,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS emails (
    id          TEXT PRIMARY KEY,
    account_id  TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    thread_id   TEXT NOT NULL,
    from_addr   TEXT NOT NULL,
    from_name   TEXT,
    to_addrs    TEXT,
    cc_addrs    TEXT,
    subject     TEXT,
    body_text   TEXT,
    body_html   TEXT,
    snippet     TEXT,
    date        DATETIME NOT NULL,
    is_read     BOOLEAN DEFAULT FALSE,
    is_starred  BOOLEAN DEFAULT FALSE,
    in_reply_to TEXT,
    history_id  INTEGER,
    raw_size    INTEGER,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS email_labels (
    email_id    TEXT NOT NULL REFERENCES emails(id) ON DELETE CASCADE,
    label_id    TEXT NOT NULL,
    PRIMARY KEY (email_id, label_id)
);

CREATE TABLE IF NOT EXISTS labels (
    id          TEXT PRIMARY KEY,
    account_id  TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    type        TEXT,
    color       TEXT
);

CREATE TABLE IF NOT EXISTS attachments (
    id          TEXT PRIMARY KEY,
    email_id    TEXT NOT NULL REFERENCES emails(id) ON DELETE CASCADE,
    filename    TEXT,
    mime_type   TEXT,
    size        INTEGER
);

CREATE TABLE IF NOT EXISTS sync_state (
    account_id  TEXT PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
    history_id  INTEGER,
    last_sync   DATETIME
);

CREATE INDEX IF NOT EXISTS idx_emails_account ON emails(account_id);
CREATE INDEX IF NOT EXISTS idx_emails_thread ON emails(thread_id);
CREATE INDEX IF NOT EXISTS idx_emails_date ON emails(date DESC);
CREATE INDEX IF NOT EXISTS idx_email_labels_label ON email_labels(label_id);
`

const ftsSchema = `
CREATE VIRTUAL TABLE IF NOT EXISTS emails_fts USING fts5(
    subject, body_text, from_addr, from_name,
    content='emails', content_rowid='rowid'
);

CREATE TRIGGER IF NOT EXISTS emails_ai AFTER INSERT ON emails BEGIN
    INSERT INTO emails_fts(rowid, subject, body_text, from_addr, from_name)
    VALUES (new.rowid, new.subject, new.body_text, new.from_addr, new.from_name);
END;

CREATE TRIGGER IF NOT EXISTS emails_ad AFTER DELETE ON emails BEGIN
    INSERT INTO emails_fts(emails_fts, rowid, subject, body_text, from_addr, from_name)
    VALUES ('delete', old.rowid, old.subject, old.body_text, old.from_addr, old.from_name);
END;

CREATE TRIGGER IF NOT EXISTS emails_au AFTER UPDATE ON emails BEGIN
    INSERT INTO emails_fts(emails_fts, rowid, subject, body_text, from_addr, from_name)
    VALUES ('delete', old.rowid, old.subject, old.body_text, old.from_addr, old.from_name);
    INSERT INTO emails_fts(rowid, subject, body_text, from_addr, from_name)
    VALUES (new.rowid, new.subject, new.body_text, new.from_addr, new.from_name);
END;
`
```

```go
// internal/store/sqlite/sqlite.go
package sqlite

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	db *sql.DB
}

func New(dsn string) (*DB, error) {
	db, err := sql.Open("sqlite3", dsn+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	s := &DB{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}
	return s, nil
}

func (s *DB) migrate() error {
	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("failed to apply schema: %w", err)
	}
	if _, err := s.db.Exec(ftsSchema); err != nil {
		return fmt.Errorf("failed to apply FTS schema: %w", err)
	}
	return nil
}

func (s *DB) Close() error {
	return s.db.Close()
}
```

**Step 6: Run test**

Run: `go test -race ./internal/store/sqlite/...`
Expected: PASS

**Step 7: Commit**

```bash
git add internal/store/ go.mod go.sum
git commit -m "feat: add SQLite store with schema migrations and FTS5"
```

---

### Task 6: Implement account CRUD in SQLite store

**Files:**
- Create: `internal/store/sqlite/accounts.go`
- Test: `internal/store/sqlite/accounts_test.go`

**Step 1: Write failing tests**

```go
// internal/store/sqlite/accounts_test.go
package sqlite

import (
	"context"
	"testing"

	"github.com/zhengda-lu/termail/internal/domain"
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
```

**Step 2: Run tests to verify they fail**

Run: `go test -race ./internal/store/sqlite/...`
Expected: FAIL (methods not defined)

**Step 3: Implement account CRUD**

```go
// internal/store/sqlite/accounts.go
package sqlite

import (
	"context"
	"fmt"

	"github.com/zhengda-lu/termail/internal/domain"
)

func (s *DB) CreateAccount(ctx context.Context, acct *domain.Account) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO accounts (id, email, provider, display_name) VALUES (?, ?, ?, ?)`,
		acct.ID, acct.Email, acct.Provider, acct.DisplayName,
	)
	if err != nil {
		return fmt.Errorf("failed to create account: %w", err)
	}
	return nil
}

func (s *DB) GetAccount(ctx context.Context, id string) (*domain.Account, error) {
	var a domain.Account
	err := s.db.QueryRowContext(ctx,
		`SELECT id, email, provider, display_name, created_at FROM accounts WHERE id = ?`, id,
	).Scan(&a.ID, &a.Email, &a.Provider, &a.DisplayName, &a.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get account %s: %w", id, err)
	}
	return &a, nil
}

func (s *DB) ListAccounts(ctx context.Context) ([]domain.Account, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, email, provider, display_name, created_at FROM accounts ORDER BY created_at`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %w", err)
	}
	defer rows.Close()

	var accounts []domain.Account
	for rows.Next() {
		var a domain.Account
		if err := rows.Scan(&a.ID, &a.Email, &a.Provider, &a.DisplayName, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan account: %w", err)
		}
		accounts = append(accounts, a)
	}
	return accounts, rows.Err()
}

func (s *DB) DeleteAccount(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM accounts WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete account %s: %w", id, err)
	}
	return nil
}
```

**Step 4: Run tests**

Run: `go test -race ./internal/store/sqlite/...`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/store/sqlite/accounts.go internal/store/sqlite/accounts_test.go
git commit -m "feat: implement account CRUD in SQLite store"
```

---

### Task 7: Implement email CRUD and label operations

**Files:**
- Create: `internal/store/sqlite/emails.go`
- Create: `internal/store/sqlite/labels.go`
- Test: `internal/store/sqlite/emails_test.go`
- Test: `internal/store/sqlite/labels_test.go`

**Step 1: Write failing tests for email operations**

```go
// internal/store/sqlite/emails_test.go
package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/zhengda-lu/termail/internal/domain"
	"github.com/zhengda-lu/termail/internal/store"
)

func seedAccount(t *testing.T, db *DB) {
	t.Helper()
	ctx := context.Background()
	db.CreateAccount(ctx, &domain.Account{ID: "acc-1", Email: "test@gmail.com", Provider: "gmail"})
}

func TestUpsertAndGetEmail(t *testing.T) {
	db := newTestDB(t)
	seedAccount(t, db)
	ctx := context.Background()

	email := &domain.Email{
		ID:       "msg-1",
		ThreadID: "thread-1",
		From:     domain.Address{Name: "John", Email: "john@example.com"},
		Subject:  "Hello",
		Body:     "Hello world",
		Date:     time.Date(2026, 2, 14, 10, 0, 0, 0, time.UTC),
		Labels:   []string{"INBOX"},
	}
	if err := db.UpsertEmail(ctx, email, "acc-1"); err != nil {
		t.Fatalf("UpsertEmail() error: %v", err)
	}

	got, err := db.GetEmail(ctx, "msg-1")
	if err != nil {
		t.Fatalf("GetEmail() error: %v", err)
	}
	if got.Subject != "Hello" {
		t.Errorf("subject = %q, want %q", got.Subject, "Hello")
	}
	if got.From.Email != "john@example.com" {
		t.Errorf("from = %q, want %q", got.From.Email, "john@example.com")
	}
}

func TestListEmails_ByLabel(t *testing.T) {
	db := newTestDB(t)
	seedAccount(t, db)
	ctx := context.Background()

	for i, label := range []string{"INBOX", "SENT"} {
		email := &domain.Email{
			ID: fmt.Sprintf("msg-%d", i), ThreadID: "t1",
			From: domain.Address{Email: "a@b.com"}, Subject: "test",
			Date: time.Now(), Labels: []string{label},
		}
		db.UpsertEmail(ctx, email, "acc-1")
	}

	emails, err := db.ListEmails(ctx, store.ListEmailOptions{
		AccountID: "acc-1", LabelID: "INBOX", Limit: 50,
	})
	if err != nil {
		t.Fatalf("ListEmails() error: %v", err)
	}
	if len(emails) != 1 {
		t.Errorf("got %d emails, want 1", len(emails))
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -race ./internal/store/sqlite/...`
Expected: FAIL

**Step 3: Implement email operations**

```go
// internal/store/sqlite/emails.go
package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/zhengda-lu/termail/internal/domain"
	"github.com/zhengda-lu/termail/internal/store"
)

func (s *DB) UpsertEmail(ctx context.Context, email *domain.Email, accountID string) error {
	toJSON, _ := json.Marshal(email.To)
	ccJSON, _ := json.Marshal(email.CC)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO emails (id, account_id, thread_id, from_addr, from_name, to_addrs, cc_addrs,
			subject, body_text, body_html, snippet, date, is_read, is_starred, in_reply_to)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			body_text=excluded.body_text, body_html=excluded.body_html,
			is_read=excluded.is_read, is_starred=excluded.is_starred,
			snippet=excluded.snippet`,
		email.ID, accountID, email.ThreadID, email.From.Email, email.From.Name,
		string(toJSON), string(ccJSON), email.Subject, email.Body, email.BodyHTML,
		email.Subject[:min(len(email.Subject), 100)], email.Date.Format(time.RFC3339),
		email.IsRead, email.IsStarred, email.InReplyTo,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert email: %w", err)
	}

	// Update labels
	tx.ExecContext(ctx, `DELETE FROM email_labels WHERE email_id = ?`, email.ID)
	for _, labelID := range email.Labels {
		tx.ExecContext(ctx, `INSERT INTO email_labels (email_id, label_id) VALUES (?, ?)`,
			email.ID, labelID)
	}

	return tx.Commit()
}

func (s *DB) GetEmail(ctx context.Context, id string) (*domain.Email, error) {
	var e domain.Email
	var toJSON, ccJSON string
	var dateStr string

	err := s.db.QueryRowContext(ctx, `
		SELECT id, thread_id, from_addr, from_name, to_addrs, cc_addrs,
			subject, body_text, body_html, date, is_read, is_starred, in_reply_to
		FROM emails WHERE id = ?`, id,
	).Scan(&e.ID, &e.ThreadID, &e.From.Email, &e.From.Name, &toJSON, &ccJSON,
		&e.Subject, &e.Body, &e.BodyHTML, &dateStr, &e.IsRead, &e.IsStarred, &e.InReplyTo)
	if err != nil {
		return nil, fmt.Errorf("failed to get email %s: %w", id, err)
	}

	e.Date, _ = time.Parse(time.RFC3339, dateStr)
	json.Unmarshal([]byte(toJSON), &e.To)
	json.Unmarshal([]byte(ccJSON), &e.CC)

	rows, _ := s.db.QueryContext(ctx,
		`SELECT label_id FROM email_labels WHERE email_id = ?`, id)
	defer rows.Close()
	for rows.Next() {
		var l string
		rows.Scan(&l)
		e.Labels = append(e.Labels, l)
	}

	return &e, nil
}

func (s *DB) ListEmails(ctx context.Context, opts store.ListEmailOptions) ([]domain.Email, error) {
	query := `SELECT e.id, e.thread_id, e.from_addr, e.from_name, e.subject,
		e.snippet, e.date, e.is_read, e.is_starred
		FROM emails e`
	var args []any

	if opts.LabelID != "" {
		query += ` JOIN email_labels el ON e.id = el.email_id WHERE el.label_id = ? AND e.account_id = ?`
		args = append(args, opts.LabelID, opts.AccountID)
	} else {
		query += ` WHERE e.account_id = ?`
		args = append(args, opts.AccountID)
	}

	query += ` ORDER BY e.date DESC`
	if opts.Limit > 0 {
		query += fmt.Sprintf(` LIMIT %d`, opts.Limit)
	}
	if opts.Offset > 0 {
		query += fmt.Sprintf(` OFFSET %d`, opts.Offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list emails: %w", err)
	}
	defer rows.Close()

	var emails []domain.Email
	for rows.Next() {
		var e domain.Email
		var dateStr string
		if err := rows.Scan(&e.ID, &e.ThreadID, &e.From.Email, &e.From.Name,
			&e.Subject, &e.Body, &dateStr, &e.IsRead, &e.IsStarred); err != nil {
			return nil, fmt.Errorf("failed to scan email: %w", err)
		}
		e.Date, _ = time.Parse(time.RFC3339, dateStr)
		emails = append(emails, e)
	}
	return emails, rows.Err()
}

func (s *DB) DeleteEmail(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM emails WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete email %s: %w", id, err)
	}
	return nil
}

func (s *DB) SetEmailLabels(ctx context.Context, emailID string, labelIDs []string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()

	tx.ExecContext(ctx, `DELETE FROM email_labels WHERE email_id = ?`, emailID)
	for _, lid := range labelIDs {
		tx.ExecContext(ctx, `INSERT INTO email_labels (email_id, label_id) VALUES (?, ?)`,
			emailID, lid)
	}
	return tx.Commit()
}
```

**Step 4: Implement label operations**

```go
// internal/store/sqlite/labels.go
package sqlite

import (
	"context"
	"fmt"

	"github.com/zhengda-lu/termail/internal/domain"
)

func (s *DB) UpsertLabel(ctx context.Context, label *domain.Label) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO labels (id, account_id, name, type, color) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET name=excluded.name, color=excluded.color`,
		label.ID, label.AccountID, label.Name, string(label.Type), label.Color,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert label: %w", err)
	}
	return nil
}

func (s *DB) ListLabels(ctx context.Context, accountID string) ([]domain.Label, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, account_id, name, type, color FROM labels WHERE account_id = ? ORDER BY name`, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to list labels: %w", err)
	}
	defer rows.Close()

	var labels []domain.Label
	for rows.Next() {
		var l domain.Label
		var lt string
		if err := rows.Scan(&l.ID, &l.AccountID, &l.Name, &lt, &l.Color); err != nil {
			return nil, fmt.Errorf("failed to scan label: %w", err)
		}
		l.Type = domain.LabelType(lt)
		labels = append(labels, l)
	}
	return labels, rows.Err()
}
```

**Step 5: Run tests**

Run: `go test -race ./internal/store/sqlite/...`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/store/sqlite/emails.go internal/store/sqlite/emails_test.go \
    internal/store/sqlite/labels.go
git commit -m "feat: implement email CRUD and label operations in SQLite store"
```

---

### Task 8: Implement thread queries and full-text search

**Files:**
- Create: `internal/store/sqlite/threads.go`
- Create: `internal/store/sqlite/search.go`
- Create: `internal/store/sqlite/sync.go`
- Test: `internal/store/sqlite/search_test.go`

**Step 1: Write failing tests**

```go
// internal/store/sqlite/search_test.go
package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/zhengda-lu/termail/internal/domain"
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
		db.UpsertEmail(ctx, &emails[i], "acc-1")
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

func TestGetThread(t *testing.T) {
	db := newTestDB(t)
	seedAccount(t, db)
	ctx := context.Background()

	for i, subj := range []string{"First msg", "Re: First msg"} {
		e := &domain.Email{
			ID: fmt.Sprintf("m%d", i), ThreadID: "thread-1",
			From: domain.Address{Email: "a@b.com"}, Subject: subj,
			Body: "body", Date: time.Now().Add(time.Duration(i) * time.Hour),
		}
		db.UpsertEmail(ctx, e, "acc-1")
	}

	thread, err := db.GetThread(ctx, "thread-1")
	if err != nil {
		t.Fatalf("GetThread() error: %v", err)
	}
	if thread.MessageCount() != 2 {
		t.Errorf("got %d messages, want 2", thread.MessageCount())
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -race ./internal/store/sqlite/...`
Expected: FAIL

**Step 3: Implement threads, search, and sync state**

```go
// internal/store/sqlite/threads.go
package sqlite

import (
	"context"
	"fmt"
	"time"

	"github.com/zhengda-lu/termail/internal/domain"
	"github.com/zhengda-lu/termail/internal/store"
)

func (s *DB) GetThread(ctx context.Context, threadID string) (*domain.Thread, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, thread_id, from_addr, from_name, subject, body_text, date, is_read, is_starred
		FROM emails WHERE thread_id = ? ORDER BY date ASC`, threadID)
	if err != nil {
		return nil, fmt.Errorf("failed to get thread: %w", err)
	}
	defer rows.Close()

	t := &domain.Thread{ID: threadID}
	for rows.Next() {
		var e domain.Email
		var dateStr string
		if err := rows.Scan(&e.ID, &e.ThreadID, &e.From.Email, &e.From.Name,
			&e.Subject, &e.Body, &dateStr, &e.IsRead, &e.IsStarred); err != nil {
			return nil, fmt.Errorf("failed to scan thread message: %w", err)
		}
		e.Date, _ = time.Parse(time.RFC3339, dateStr)
		t.Messages = append(t.Messages, e)
	}

	if len(t.Messages) > 0 {
		t.Subject = t.Messages[0].Subject
		last := t.Messages[len(t.Messages)-1]
		t.LastDate = last.Date
		t.Snippet = last.Body
		if len(t.Snippet) > 100 {
			t.Snippet = t.Snippet[:100]
		}
	}
	return t, rows.Err()
}

func (s *DB) ListThreads(ctx context.Context, opts store.ListEmailOptions) ([]domain.Thread, error) {
	query := `SELECT e.thread_id, e.subject, e.snippet, MAX(e.date) as last_date, COUNT(*) as cnt
		FROM emails e`
	var args []any

	if opts.LabelID != "" {
		query += ` JOIN email_labels el ON e.id = el.email_id WHERE el.label_id = ? AND e.account_id = ?`
		args = append(args, opts.LabelID, opts.AccountID)
	} else {
		query += ` WHERE e.account_id = ?`
		args = append(args, opts.AccountID)
	}

	query += ` GROUP BY e.thread_id ORDER BY last_date DESC`
	if opts.Limit > 0 {
		query += fmt.Sprintf(` LIMIT %d`, opts.Limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list threads: %w", err)
	}
	defer rows.Close()

	var threads []domain.Thread
	for rows.Next() {
		var t domain.Thread
		var dateStr string
		var cnt int
		if err := rows.Scan(&t.ID, &t.Subject, &t.Snippet, &dateStr, &cnt); err != nil {
			return nil, fmt.Errorf("failed to scan thread: %w", err)
		}
		t.LastDate, _ = time.Parse(time.RFC3339, dateStr)
		threads = append(threads, t)
	}
	return threads, rows.Err()
}
```

```go
// internal/store/sqlite/search.go
package sqlite

import (
	"context"
	"fmt"
	"time"

	"github.com/zhengda-lu/termail/internal/domain"
)

func (s *DB) SearchEmails(ctx context.Context, query string, accountID string) ([]domain.Email, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT e.id, e.thread_id, e.from_addr, e.from_name, e.subject, e.snippet, e.date, e.is_read, e.is_starred
		FROM emails e
		JOIN emails_fts fts ON e.rowid = fts.rowid
		WHERE emails_fts MATCH ? AND e.account_id = ?
		ORDER BY rank`, query, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to search emails: %w", err)
	}
	defer rows.Close()

	var emails []domain.Email
	for rows.Next() {
		var e domain.Email
		var dateStr string
		if err := rows.Scan(&e.ID, &e.ThreadID, &e.From.Email, &e.From.Name,
			&e.Subject, &e.Body, &dateStr, &e.IsRead, &e.IsStarred); err != nil {
			return nil, fmt.Errorf("failed to scan search result: %w", err)
		}
		e.Date, _ = time.Parse(time.RFC3339, dateStr)
		emails = append(emails, e)
	}
	return emails, rows.Err()
}
```

```go
// internal/store/sqlite/sync.go
package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/zhengda-lu/termail/internal/store"
)

func (s *DB) GetSyncState(ctx context.Context, accountID string) (*store.SyncState, error) {
	var st store.SyncState
	err := s.db.QueryRowContext(ctx,
		`SELECT account_id, history_id, last_sync FROM sync_state WHERE account_id = ?`,
		accountID,
	).Scan(&st.AccountID, &st.HistoryID, &st.LastSync)
	if err == sql.ErrNoRows {
		return &store.SyncState{AccountID: accountID}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get sync state: %w", err)
	}
	return &st, nil
}

func (s *DB) SetSyncState(ctx context.Context, state *store.SyncState) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO sync_state (account_id, history_id, last_sync) VALUES (?, ?, ?)
		ON CONFLICT(account_id) DO UPDATE SET history_id=excluded.history_id, last_sync=excluded.last_sync`,
		state.AccountID, state.HistoryID, state.LastSync,
	)
	if err != nil {
		return fmt.Errorf("failed to set sync state: %w", err)
	}
	return nil
}
```

**Step 4: Run tests**

Run: `go test -race ./internal/store/sqlite/...`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/store/sqlite/threads.go internal/store/sqlite/search.go \
    internal/store/sqlite/sync.go internal/store/sqlite/search_test.go
git commit -m "feat: add thread queries, FTS5 search, and sync state to SQLite store"
```

---

## Phase 4: Keyring Token Storage

### Task 9: Implement OS keyring token store

**Files:**
- Create: `internal/store/keyring.go`
- Test: `internal/store/keyring_test.go`

**Step 1: Install dependency**

Run: `go get github.com/zalando/go-keyring`

**Step 2: Implement keyring store**

```go
// internal/store/keyring.go
package store

import (
	"encoding/json"
	"fmt"

	"github.com/zalando/go-keyring"
	"golang.org/x/oauth2"
)

const serviceName = "termail"

type KeyringTokenStore struct{}

func NewKeyringTokenStore() *KeyringTokenStore {
	return &KeyringTokenStore{}
}

func (k *KeyringTokenStore) SaveToken(accountID string, token *oauth2.Token) error {
	data, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}
	if err := keyring.Set(serviceName, accountID, string(data)); err != nil {
		return fmt.Errorf("failed to save token to keyring: %w", err)
	}
	return nil
}

func (k *KeyringTokenStore) LoadToken(accountID string) (*oauth2.Token, error) {
	data, err := keyring.Get(serviceName, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to load token from keyring: %w", err)
	}
	var token oauth2.Token
	if err := json.Unmarshal([]byte(data), &token); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token: %w", err)
	}
	return &token, nil
}

func (k *KeyringTokenStore) DeleteToken(accountID string) error {
	if err := keyring.Delete(serviceName, accountID); err != nil {
		return fmt.Errorf("failed to delete token from keyring: %w", err)
	}
	return nil
}
```

Note: Keyring tests require OS keyring access and are hard to run in CI. Manual test by running `termail account add` in Phase 5.

**Step 3: Commit**

```bash
git add internal/store/keyring.go go.mod go.sum
git commit -m "feat: add OS keyring token storage"
```

---

## Phase 5: Gmail OAuth2 & Provider

### Task 10: Implement Gmail OAuth2 flow

**Files:**
- Create: `internal/provider/provider.go`
- Create: `internal/provider/gmail/oauth.go`
- Create: `internal/provider/gmail/client.go`

**Step 1: Install dependencies**

Run: `go get golang.org/x/oauth2 golang.org/x/oauth2/google google.golang.org/api/gmail/v1`

**Step 2: Define provider interface**

```go
// internal/provider/provider.go
package provider

import (
	"context"

	"github.com/zhengda-lu/termail/internal/domain"
)

type ListOptions struct {
	PageToken  string
	MaxResults int
	LabelIDs   []string
	Query      string
}

type EmailProvider interface {
	Authenticate(ctx context.Context) error
	IsAuthenticated() bool

	ListMessages(ctx context.Context, opts ListOptions) ([]domain.Email, string, error)
	GetMessage(ctx context.Context, id string) (*domain.Email, error)
	SendMessage(ctx context.Context, email *domain.Email) error

	ListThreads(ctx context.Context, opts ListOptions) ([]domain.Thread, string, error)
	GetThread(ctx context.Context, id string) (*domain.Thread, error)

	ModifyLabels(ctx context.Context, msgID string, add, remove []string) error
	TrashMessage(ctx context.Context, msgID string) error
	MarkRead(ctx context.Context, msgID string, read bool) error

	ListLabels(ctx context.Context) ([]domain.Label, error)
	Search(ctx context.Context, query string, opts ListOptions) ([]domain.Email, string, error)

	History(ctx context.Context, startHistoryID uint64) ([]HistoryEvent, uint64, error)
}

type HistoryEventType int

const (
	HistoryMessageAdded HistoryEventType = iota
	HistoryMessageDeleted
	HistoryLabelsAdded
	HistoryLabelsRemoved
)

type HistoryEvent struct {
	Type      HistoryEventType
	MessageID string
	LabelIDs  []string
}
```

**Step 3: Implement OAuth flow**

```go
// internal/provider/gmail/oauth.go
package gmail

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	gmailapi "google.golang.org/api/gmail/v1"
)

// These are embedded in the binary. For installed apps, Google considers
// the client secret non-confidential (security comes from localhost redirect).
// Users should create their own credentials at console.cloud.google.com.
var oauthConfig = &oauth2.Config{
	Scopes: []string{
		gmailapi.GmailReadonlyScope,
		gmailapi.GmailSendScope,
		gmailapi.GmailModifyScope,
	},
	Endpoint: google.Endpoint,
}

// SetCredentials sets the OAuth client ID and secret.
// Call before Authenticate. Credentials come from config or env vars.
func SetCredentials(clientID, clientSecret string) {
	oauthConfig.ClientID = clientID
	oauthConfig.ClientSecret = clientSecret
}

func authenticate(ctx context.Context) (*oauth2.Token, error) {
	// Start local server on random port for callback
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return nil, fmt.Errorf("failed to start callback server: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	oauthConfig.RedirectURL = fmt.Sprintf("http://localhost:%d/callback", port)

	// Channel to receive the auth code
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			errCh <- fmt.Errorf("no code in callback: %s", r.URL.Query().Get("error"))
			fmt.Fprint(w, "Authentication failed. You can close this tab.")
			return
		}
		codeCh <- code
		fmt.Fprint(w, "Authentication successful! You can close this tab.")
	})

	server := &http.Server{Handler: mux}
	go server.Serve(listener)
	defer server.Shutdown(ctx)

	// Generate and print the auth URL
	url := oauthConfig.AuthCodeURL("state", oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	fmt.Printf("\nOpen this URL in your browser to authorize termail:\n\n  %s\n\nWaiting for authorization...\n", url)

	// Wait for callback
	select {
	case code := <-codeCh:
		token, err := oauthConfig.Exchange(ctx, code)
		if err != nil {
			return nil, fmt.Errorf("failed to exchange auth code: %w", err)
		}
		return token, nil
	case err := <-errCh:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
```

**Step 4: Commit**

```bash
git add internal/provider/ go.mod go.sum
git commit -m "feat: add provider interface and Gmail OAuth2 flow"
```

---

### Task 11: Implement Gmail API client and mapper

**Files:**
- Create: `internal/provider/gmail/mapper.go`
- Modify: `internal/provider/gmail/client.go`
- Test: `internal/provider/gmail/mapper_test.go`

**Step 1: Write mapper tests**

```go
// internal/provider/gmail/mapper_test.go
package gmail

import (
	"testing"

	gmailapi "google.golang.org/api/gmail/v1"
)

func TestParseAddress(t *testing.T) {
	tests := []struct {
		input     string
		wantName  string
		wantEmail string
	}{
		{"John Doe <john@example.com>", "John Doe", "john@example.com"},
		{"<john@example.com>", "", "john@example.com"},
		{"john@example.com", "", "john@example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			addr := parseAddress(tt.input)
			if addr.Name != tt.wantName {
				t.Errorf("name = %q, want %q", addr.Name, tt.wantName)
			}
			if addr.Email != tt.wantEmail {
				t.Errorf("email = %q, want %q", addr.Email, tt.wantEmail)
			}
		})
	}
}

func TestFindHeader(t *testing.T) {
	headers := []*gmailapi.MessagePartHeader{
		{Name: "From", Value: "alice@test.com"},
		{Name: "Subject", Value: "Hello"},
	}
	if got := findHeader(headers, "Subject"); got != "Hello" {
		t.Errorf("findHeader(Subject) = %q, want %q", got, "Hello")
	}
	if got := findHeader(headers, "Missing"); got != "" {
		t.Errorf("findHeader(Missing) = %q, want empty", got)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -race ./internal/provider/gmail/...`
Expected: FAIL

**Step 3: Implement mapper**

```go
// internal/provider/gmail/mapper.go
package gmail

import (
	"net/mail"
	"strings"
	"time"

	"github.com/zhengda-lu/termail/internal/domain"
	gmailapi "google.golang.org/api/gmail/v1"
)

func mapMessage(msg *gmailapi.Message) *domain.Email {
	headers := msg.Payload.Headers
	email := &domain.Email{
		ID:        msg.Id,
		ThreadID:  msg.ThreadId,
		From:      parseAddress(findHeader(headers, "From")),
		Subject:   findHeader(headers, "Subject"),
		Date:      parseDate(findHeader(headers, "Date")),
		IsRead:    !containsLabel(msg.LabelIds, "UNREAD"),
		IsStarred: containsLabel(msg.LabelIds, "STARRED"),
		Labels:    msg.LabelIds,
		InReplyTo: findHeader(headers, "In-Reply-To"),
	}

	// Parse To and CC
	if to := findHeader(headers, "To"); to != "" {
		email.To = parseAddressList(to)
	}
	if cc := findHeader(headers, "Cc"); cc != "" {
		email.CC = parseAddressList(cc)
	}

	// Extract body
	email.Body, email.BodyHTML = extractBody(msg.Payload)

	// Map attachments
	for _, part := range msg.Payload.Parts {
		if part.Filename != "" {
			email.Attachments = append(email.Attachments, domain.Attachment{
				ID:       part.Body.AttachmentId,
				Filename: part.Filename,
				MIMEType: part.MimeType,
				Size:     part.Body.Size,
			})
		}
	}

	return email
}

func findHeader(headers []*gmailapi.MessagePartHeader, name string) string {
	for _, h := range headers {
		if strings.EqualFold(h.Name, name) {
			return h.Value
		}
	}
	return ""
}

func parseAddress(s string) domain.Address {
	if s == "" {
		return domain.Address{}
	}
	addr, err := mail.ParseAddress(s)
	if err != nil {
		// Fallback: treat whole string as email
		s = strings.TrimSpace(s)
		s = strings.Trim(s, "<>")
		return domain.Address{Email: s}
	}
	return domain.Address{Name: addr.Name, Email: addr.Address}
}

func parseAddressList(s string) []domain.Address {
	list, err := mail.ParseAddressList(s)
	if err != nil {
		return []domain.Address{{Email: s}}
	}
	addrs := make([]domain.Address, len(list))
	for i, a := range list {
		addrs[i] = domain.Address{Name: a.Name, Email: a.Address}
	}
	return addrs
}

func parseDate(s string) time.Time {
	formats := []string{
		time.RFC1123Z,
		time.RFC1123,
		"Mon, 2 Jan 2006 15:04:05 -0700",
		"2 Jan 2006 15:04:05 -0700",
		time.RFC3339,
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

func containsLabel(labels []string, label string) bool {
	for _, l := range labels {
		if l == label {
			return true
		}
	}
	return false
}

func extractBody(payload *gmailapi.MessagePart) (text, html string) {
	if payload.MimeType == "text/plain" && payload.Body.Data != "" {
		return decodeBase64URL(payload.Body.Data), ""
	}
	if payload.MimeType == "text/html" && payload.Body.Data != "" {
		return "", decodeBase64URL(payload.Body.Data)
	}
	for _, part := range payload.Parts {
		t, h := extractBody(part)
		if t != "" {
			text = t
		}
		if h != "" {
			html = h
		}
	}
	return text, html
}

func decodeBase64URL(s string) string {
	// Gmail uses URL-safe base64
	s = strings.ReplaceAll(s, "-", "+")
	s = strings.ReplaceAll(s, "_", "/")
	// Add padding
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}
	decoded, err := base64Decode(s)
	if err != nil {
		return s
	}
	return string(decoded)
}
```

Note: `decodeBase64URL` needs `encoding/base64` import and a helper — add:

```go
import "encoding/base64"

func base64Decode(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}
```

**Step 4: Implement Gmail client**

```go
// internal/provider/gmail/client.go
package gmail

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"github.com/zhengda-lu/termail/internal/domain"
	"github.com/zhengda-lu/termail/internal/provider"
	"github.com/zhengda-lu/termail/internal/store"
	"golang.org/x/oauth2"
	gmailapi "google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

type Provider struct {
	tokenStore *store.KeyringTokenStore
	accountID  string
	service    *gmailapi.Service
	token      *oauth2.Token
}

func New(accountID string, tokenStore *store.KeyringTokenStore) *Provider {
	return &Provider{
		accountID:  accountID,
		tokenStore: tokenStore,
	}
}

func (p *Provider) Authenticate(ctx context.Context) error {
	token, err := authenticate(ctx)
	if err != nil {
		return fmt.Errorf("failed to authenticate: %w", err)
	}
	if err := p.tokenStore.SaveToken(p.accountID, token); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}
	p.token = token
	return p.initService(ctx)
}

func (p *Provider) IsAuthenticated() bool {
	return p.service != nil
}

func (p *Provider) initService(ctx context.Context) error {
	if p.token == nil {
		token, err := p.tokenStore.LoadToken(p.accountID)
		if err != nil {
			return fmt.Errorf("failed to load token: %w", err)
		}
		p.token = token
	}
	client := oauthConfig.Client(ctx, p.token)
	svc, err := gmailapi.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return fmt.Errorf("failed to create gmail service: %w", err)
	}
	p.service = svc
	return nil
}

func (p *Provider) ensureService(ctx context.Context) error {
	if p.service == nil {
		return p.initService(ctx)
	}
	return nil
}

func (p *Provider) ListMessages(ctx context.Context, opts provider.ListOptions) ([]domain.Email, string, error) {
	if err := p.ensureService(ctx); err != nil {
		return nil, "", err
	}
	call := p.service.Users.Messages.List("me")
	if opts.MaxResults > 0 {
		call = call.MaxResults(int64(opts.MaxResults))
	}
	if opts.PageToken != "" {
		call = call.PageToken(opts.PageToken)
	}
	for _, lid := range opts.LabelIDs {
		call = call.LabelIds(lid)
	}
	if opts.Query != "" {
		call = call.Q(opts.Query)
	}

	resp, err := call.Context(ctx).Do()
	if err != nil {
		return nil, "", fmt.Errorf("failed to list messages: %w", err)
	}

	var emails []domain.Email
	for _, msg := range resp.Messages {
		full, err := p.service.Users.Messages.Get("me", msg.Id).
			Format("full").Context(ctx).Do()
		if err != nil {
			continue
		}
		emails = append(emails, *mapMessage(full))
	}
	return emails, resp.NextPageToken, nil
}

func (p *Provider) GetMessage(ctx context.Context, id string) (*domain.Email, error) {
	if err := p.ensureService(ctx); err != nil {
		return nil, err
	}
	msg, err := p.service.Users.Messages.Get("me", id).
		Format("full").Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get message %s: %w", id, err)
	}
	return mapMessage(msg), nil
}

func (p *Provider) SendMessage(ctx context.Context, email *domain.Email) error {
	if err := p.ensureService(ctx); err != nil {
		return err
	}

	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("From: %s\r\n", email.From.String()))
	for _, to := range email.To {
		msg.WriteString(fmt.Sprintf("To: %s\r\n", to.String()))
	}
	for _, cc := range email.CC {
		msg.WriteString(fmt.Sprintf("Cc: %s\r\n", cc.String()))
	}
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", email.Subject))
	if email.InReplyTo != "" {
		msg.WriteString(fmt.Sprintf("In-Reply-To: %s\r\n", email.InReplyTo))
	}
	msg.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(email.Body)

	raw := base64.URLEncoding.EncodeToString([]byte(msg.String()))
	_, err := p.service.Users.Messages.Send("me", &gmailapi.Message{
		Raw:      raw,
		ThreadId: email.ThreadID,
	}).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	return nil
}

func (p *Provider) ModifyLabels(ctx context.Context, msgID string, add, remove []string) error {
	if err := p.ensureService(ctx); err != nil {
		return err
	}
	_, err := p.service.Users.Messages.Modify("me", msgID, &gmailapi.ModifyMessageRequest{
		AddLabelIds:    add,
		RemoveLabelIds: remove,
	}).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to modify labels: %w", err)
	}
	return nil
}

func (p *Provider) TrashMessage(ctx context.Context, msgID string) error {
	if err := p.ensureService(ctx); err != nil {
		return err
	}
	_, err := p.service.Users.Messages.Trash("me", msgID).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to trash message %s: %w", msgID, err)
	}
	return nil
}

func (p *Provider) MarkRead(ctx context.Context, msgID string, read bool) error {
	if read {
		return p.ModifyLabels(ctx, msgID, nil, []string{"UNREAD"})
	}
	return p.ModifyLabels(ctx, msgID, []string{"UNREAD"}, nil)
}

func (p *Provider) ListLabels(ctx context.Context) ([]domain.Label, error) {
	if err := p.ensureService(ctx); err != nil {
		return nil, err
	}
	resp, err := p.service.Users.Labels.List("me").Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list labels: %w", err)
	}
	var labels []domain.Label
	for _, l := range resp.Labels {
		lt := domain.LabelTypeUser
		if l.Type == "system" {
			lt = domain.LabelTypeSystem
		}
		labels = append(labels, domain.Label{
			ID:   l.Id,
			Name: l.Name,
			Type: lt,
		})
	}
	return labels, nil
}

func (p *Provider) ListThreads(ctx context.Context, opts provider.ListOptions) ([]domain.Thread, string, error) {
	if err := p.ensureService(ctx); err != nil {
		return nil, "", err
	}
	call := p.service.Users.Threads.List("me")
	if opts.MaxResults > 0 {
		call = call.MaxResults(int64(opts.MaxResults))
	}
	if opts.PageToken != "" {
		call = call.PageToken(opts.PageToken)
	}
	for _, lid := range opts.LabelIDs {
		call = call.LabelIds(lid)
	}

	resp, err := call.Context(ctx).Do()
	if err != nil {
		return nil, "", fmt.Errorf("failed to list threads: %w", err)
	}

	var threads []domain.Thread
	for _, t := range resp.Threads {
		threads = append(threads, domain.Thread{
			ID:      t.Id,
			Snippet: t.Snippet,
		})
	}
	return threads, resp.NextPageToken, nil
}

func (p *Provider) GetThread(ctx context.Context, id string) (*domain.Thread, error) {
	if err := p.ensureService(ctx); err != nil {
		return nil, err
	}
	t, err := p.service.Users.Threads.Get("me", id).
		Format("full").Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get thread %s: %w", id, err)
	}
	thread := &domain.Thread{ID: t.Id, Snippet: t.Snippet}
	for _, msg := range t.Messages {
		thread.Messages = append(thread.Messages, *mapMessage(msg))
	}
	if len(thread.Messages) > 0 {
		thread.Subject = thread.Messages[0].Subject
		thread.LastDate = thread.Messages[len(thread.Messages)-1].Date
	}
	return thread, nil
}

func (p *Provider) Search(ctx context.Context, query string, opts provider.ListOptions) ([]domain.Email, string, error) {
	opts.Query = query
	return p.ListMessages(ctx, opts)
}

func (p *Provider) History(ctx context.Context, startHistoryID uint64) ([]provider.HistoryEvent, uint64, error) {
	if err := p.ensureService(ctx); err != nil {
		return nil, 0, err
	}

	var events []provider.HistoryEvent
	var latestHistoryID uint64

	call := p.service.Users.History.List("me").StartHistoryId(startHistoryID)
	err := call.Pages(ctx, func(resp *gmailapi.ListHistoryResponse) error {
		latestHistoryID = resp.HistoryId
		for _, h := range resp.History {
			for _, added := range h.MessagesAdded {
				events = append(events, provider.HistoryEvent{
					Type:      provider.HistoryMessageAdded,
					MessageID: added.Message.Id,
					LabelIDs:  added.Message.LabelIds,
				})
			}
			for _, deleted := range h.MessagesDeleted {
				events = append(events, provider.HistoryEvent{
					Type:      provider.HistoryMessageDeleted,
					MessageID: deleted.Message.Id,
				})
			}
			for _, la := range h.LabelsAdded {
				events = append(events, provider.HistoryEvent{
					Type:      provider.HistoryLabelsAdded,
					MessageID: la.Message.Id,
					LabelIDs:  la.LabelIds,
				})
			}
			for _, lr := range h.LabelsRemoved {
				events = append(events, provider.HistoryEvent{
					Type:      provider.HistoryLabelsRemoved,
					MessageID: lr.Message.Id,
					LabelIDs:  lr.LabelIds,
				})
			}
		}
		return nil
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list history: %w", err)
	}
	return events, latestHistoryID, nil
}
```

**Step 5: Run mapper tests**

Run: `go test -race ./internal/provider/gmail/...`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/provider/
git commit -m "feat: implement Gmail API client with mapper and history sync"
```

---

## Phase 6: Sync Engine

### Task 12: Implement sync orchestration

**Files:**
- Create: `internal/app/sync.go`
- Test: `internal/app/sync_test.go`

**Step 1: Implement sync service**

```go
// internal/app/sync.go
package app

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/zhengda-lu/termail/internal/provider"
	"github.com/zhengda-lu/termail/internal/store"
)

type SyncService struct {
	store    store.Store
	provider provider.EmailProvider
	accountID string
}

func NewSyncService(s store.Store, p provider.EmailProvider, accountID string) *SyncService {
	return &SyncService{store: s, provider: p, accountID: accountID}
}

// InitialSync fetches the latest messages and stores them locally.
func (s *SyncService) InitialSync(ctx context.Context, count int) error {
	log.Printf("Starting initial sync for %s (fetching %d messages)...", s.accountID, count)

	// Fetch labels first
	labels, err := s.provider.ListLabels(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch labels: %w", err)
	}
	for i := range labels {
		labels[i].AccountID = s.accountID
		if err := s.store.UpsertLabel(ctx, &labels[i]); err != nil {
			return fmt.Errorf("failed to store label: %w", err)
		}
	}

	// Fetch messages in pages
	fetched := 0
	pageToken := ""
	for fetched < count {
		batchSize := min(100, count-fetched)
		msgs, nextToken, err := s.provider.ListMessages(ctx, provider.ListOptions{
			MaxResults: batchSize,
			PageToken:  pageToken,
		})
		if err != nil {
			return fmt.Errorf("failed to fetch messages: %w", err)
		}
		for i := range msgs {
			if err := s.store.UpsertEmail(ctx, &msgs[i], s.accountID); err != nil {
				log.Printf("Warning: failed to store message %s: %v", msgs[i].ID, err)
			}
		}
		fetched += len(msgs)
		pageToken = nextToken
		if nextToken == "" {
			break
		}
		log.Printf("Synced %d/%d messages...", fetched, count)
	}

	// Save sync state
	state := &store.SyncState{
		AccountID: s.accountID,
		LastSync:  time.Now().Unix(),
	}
	if err := s.store.SetSyncState(ctx, state); err != nil {
		return fmt.Errorf("failed to save sync state: %w", err)
	}

	log.Printf("Initial sync complete: %d messages stored", fetched)
	return nil
}

// IncrementalSync uses Gmail history API to fetch only changes.
func (s *SyncService) IncrementalSync(ctx context.Context) error {
	syncState, err := s.store.GetSyncState(ctx, s.accountID)
	if err != nil {
		return fmt.Errorf("failed to get sync state: %w", err)
	}
	if syncState.HistoryID == 0 {
		return s.InitialSync(ctx, 500)
	}

	events, newHistoryID, err := s.provider.History(ctx, syncState.HistoryID)
	if err != nil {
		return fmt.Errorf("failed to fetch history: %w", err)
	}

	for _, event := range events {
		switch event.Type {
		case provider.HistoryMessageAdded:
			msg, err := s.provider.GetMessage(ctx, event.MessageID)
			if err != nil {
				log.Printf("Warning: failed to fetch new message %s: %v", event.MessageID, err)
				continue
			}
			s.store.UpsertEmail(ctx, msg, s.accountID)

		case provider.HistoryMessageDeleted:
			s.store.DeleteEmail(ctx, event.MessageID)

		case provider.HistoryLabelsAdded, provider.HistoryLabelsRemoved:
			msg, err := s.provider.GetMessage(ctx, event.MessageID)
			if err != nil {
				continue
			}
			s.store.SetEmailLabels(ctx, event.MessageID, msg.Labels)
		}
	}

	syncState.HistoryID = newHistoryID
	syncState.LastSync = time.Now().Unix()
	s.store.SetSyncState(ctx, syncState)

	log.Printf("Incremental sync complete: %d events processed", len(events))
	return nil
}
```

**Step 2: Commit**

```bash
git add internal/app/sync.go
git commit -m "feat: implement sync service with initial and incremental sync"
```

---

## Phase 7: TUI Foundation

### Task 13: Set up Bubble Tea app shell with styles and key bindings

**Files:**
- Create: `internal/tui/styles.go`
- Create: `internal/tui/keys.go`
- Create: `internal/tui/statusbar.go`
- Create: `internal/tui/app.go`

**Step 1: Install dependencies**

Run: `go get github.com/charmbracelet/bubbletea github.com/charmbracelet/lipgloss github.com/charmbracelet/bubbles`

**Step 2: Implement styles**

```go
// internal/tui/styles.go
package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	primaryColor   = lipgloss.Color("#7C3AED")
	secondaryColor = lipgloss.Color("#6366F1")
	mutedColor     = lipgloss.Color("#6B7280")
	accentColor    = lipgloss.Color("#F59E0B")
	errorColor     = lipgloss.Color("#EF4444")
	successColor   = lipgloss.Color("#10B981")

	// Pane styles
	sidebarStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(mutedColor).
			Padding(1, 1)

	listStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(mutedColor).
			Padding(0, 1)

	readerStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(mutedColor).
			Padding(1, 2)

	statusBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#1F2937")).
			Foreground(lipgloss.Color("#D1D5DB")).
			Padding(0, 1)

	// Text styles
	titleStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true)

	selectedStyle = lipgloss.NewStyle().
			Background(primaryColor).
			Foreground(lipgloss.Color("#FFFFFF"))

	unreadStyle = lipgloss.NewStyle().
			Bold(true)

	starStyle = lipgloss.NewStyle().
			Foreground(accentColor)

	mutedTextStyle = lipgloss.NewStyle().
			Foreground(mutedColor)
)
```

**Step 3: Implement key bindings**

```go
// internal/tui/keys.go
package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Up       key.Binding
	Down     key.Binding
	Enter    key.Binding
	Back     key.Binding
	Compose  key.Binding
	Reply    key.Binding
	ReplyAll key.Binding
	Forward  key.Binding
	Archive  key.Binding
	Delete   key.Binding
	Star     key.Binding
	Unread   key.Binding
	Label    key.Binding
	Search   key.Binding
	Tab      key.Binding
	Toggle   key.Binding
	Help     key.Binding
	Quit     key.Binding
}

var keys = keyMap{
	Up:       key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("k/↑", "up")),
	Down:     key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j/↓", "down")),
	Enter:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open")),
	Back:     key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	Compose:  key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "compose")),
	Reply:    key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reply")),
	ReplyAll: key.NewBinding(key.WithKeys("R"), key.WithHelp("R", "reply all")),
	Forward:  key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "forward")),
	Archive:  key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "archive")),
	Delete:   key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "trash")),
	Star:     key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "star")),
	Unread:   key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "unread")),
	Label:    key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "label")),
	Search:   key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
	Tab:      key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "switch pane")),
	Toggle:   key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "thread/flat")),
	Help:     key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
	Quit:     key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
}
```

**Step 4: Implement status bar**

```go
// internal/tui/statusbar.go
package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
)

type statusBar struct {
	message   string
	lastSync  time.Time
	width     int
	isError   bool
}

func newStatusBar() statusBar {
	return statusBar{message: "Ready"}
}

func (s statusBar) View() string {
	syncInfo := "Never synced"
	if !s.lastSync.IsZero() {
		ago := time.Since(s.lastSync).Truncate(time.Second)
		syncInfo = fmt.Sprintf("Synced %s ago", ago)
	}

	msgStyle := statusBarStyle
	if s.isError {
		msgStyle = msgStyle.Foreground(errorColor)
	}

	left := msgStyle.Render(s.message)
	right := statusBarStyle.Render(syncInfo)
	shortcuts := mutedTextStyle.Render(" j/k:nav  enter:open  c:compose  ?:help ")

	gap := s.width - lipgloss.Width(left) - lipgloss.Width(right) - lipgloss.Width(shortcuts)
	if gap < 0 {
		gap = 0
	}

	return statusBarStyle.Width(s.width).Render(
		left + lipgloss.NewStyle().Width(gap).Render("") + shortcuts + right,
	)
}
```

**Step 5: Implement root app model (minimal shell)**

```go
// internal/tui/app.go
package tui

import (
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/zhengda-lu/termail/internal/store"
	"github.com/zhengda-lu/termail/internal/provider"
)

type pane int

const (
	paneSidebar pane = iota
	paneList
	paneReader
)

type viewMode int

const (
	viewThread viewMode = iota
	viewFlat
)

type model struct {
	store      store.Store
	provider   provider.EmailProvider
	accountID  string

	activePane pane
	viewMode   viewMode
	statusBar  statusBar

	width  int
	height int

	// Sub-models will be added in later tasks
}

func NewModel(s store.Store, p provider.EmailProvider, accountID string) model {
	return model{
		store:      s,
		provider:   p,
		accountID:  accountID,
		activePane: paneList,
		viewMode:   viewThread,
		statusBar:  newStatusBar(),
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.statusBar.width = msg.Width
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, keys.Tab):
			m.activePane = (m.activePane + 1) % 3
		case key.Matches(msg, keys.Toggle):
			if m.viewMode == viewThread {
				m.viewMode = viewFlat
				m.statusBar.message = "Switched to flat view"
			} else {
				m.viewMode = viewThread
				m.statusBar.message = "Switched to thread view"
			}
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	sidebarWidth := m.width / 5
	if sidebarWidth < 20 {
		sidebarWidth = 20
	}
	contentWidth := m.width - sidebarWidth - 2

	sidebar := sidebarStyle.
		Width(sidebarWidth).
		Height(m.height - 3).
		Render("Sidebar (TODO)")

	content := listStyle.
		Width(contentWidth).
		Height(m.height - 3).
		Render("Email list (TODO)")

	main := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)
	statusBar := m.statusBar.View()

	return lipgloss.JoinVertical(lipgloss.Left, main, statusBar)
}

// Run starts the Bubble Tea program.
func Run(s store.Store, p provider.EmailProvider, accountID string) error {
	prog := tea.NewProgram(
		NewModel(s, p, accountID),
		tea.WithAltScreen(),
	)
	_, err := prog.Run()
	return err
}
```

Note: This needs `"github.com/charmbracelet/bubbles/key"` import in app.go.

**Step 6: Build and verify**

Run: `go build ./cmd/termail`
Expected: Compiles successfully.

**Step 7: Commit**

```bash
git add internal/tui/ go.mod go.sum
git commit -m "feat: add Bubble Tea app shell with styles, keys, and status bar"
```

---

### Task 14: Implement sidebar component

### Task 15: Implement inbox/message list component

### Task 16: Implement email reader component

### Task 17: Implement composer component

### Task 18: Implement search bar component

> Tasks 14-18 follow the same TDD pattern as above. Each creates a Bubble Tea sub-model in its own file, tests the Update logic, and integrates into `app.go`. Due to plan length, these are described at a higher level — the implementing engineer should follow the exact same step pattern (write test → verify fail → implement → verify pass → commit).

**Task 14 (Sidebar):** `internal/tui/sidebar.go` — Uses `bubbles/list` to show labels with unread counts. Fetches labels from store on init. Highlights active label. Sends a `labelSelectedMsg` when user picks a label.

**Task 15 (Inbox):** `internal/tui/inbox.go` — Uses `bubbles/list` to show emails or threads (based on viewMode). Fetches from store based on active label. Supports pagination. Renders: star icon, from, subject, date, unread bold.

**Task 16 (Reader):** `internal/tui/reader.go` — Uses `bubbles/viewport` for scrollable email content. Renders headers (From, To, Date, Subject) then body. Uses `glamour` for markdown-ish rendering of plain text. Falls back to `html2text` for HTML-only emails.

**Task 17 (Composer):** `internal/tui/composer.go` — Uses `bubbles/textarea` for body and `bubbles/textinput` for To/Subject fields. Tab between fields. Ctrl+Enter or a confirm key to send. Supports reply (pre-fills To, Subject with "Re:", quotes body) and forward (pre-fills body with "Fwd:").

**Task 18 (Search):** `internal/tui/search.go` — Uses `bubbles/textinput` for query. On enter, searches via store's FTS5. Results displayed in same list format as inbox. Esc to exit search mode.

---

## Phase 8: Wire Everything Together

### Task 19: Connect CLI commands to real implementations

**Files:**
- Modify: `internal/cli/root.go`
- Modify: `internal/cli/account.go`
- Modify: `cmd/termail/main.go`

Wire up:
1. `termail` (no args) → opens TUI with default account
2. `termail account add` → runs Gmail OAuth flow → stores account in SQLite + tokens in keyring
3. `termail account list` → reads from SQLite, prints table
4. `termail account remove` → deletes from SQLite + keyring
5. `termail sync` → runs InitialSync or IncrementalSync

**Step 1: Update CLI to use real store and provider**

The root command creates the SQLite DB, loads config, resolves the default account, and starts the TUI. Account commands use the store directly.

**Step 2: Build and manual test**

Run: `go build ./cmd/termail && ./termail --help`

**Step 3: Commit**

```bash
git add cmd/ internal/cli/
git commit -m "feat: wire CLI commands to store, provider, and TUI"
```

---

### Task 20: Add .gitignore and README

**Files:**
- Create: `.gitignore`
- Create: `README.md`

Standard Go `.gitignore` plus `termail` binary. README with project description, installation, usage, and configuration.

**Commit:**
```bash
git add .gitignore README.md
git commit -m "docs: add README and .gitignore"
```

---

## Summary

| Phase | Tasks | Focus |
|-------|-------|-------|
| 1 | 1-2 | Project scaffolding, domain models |
| 2 | 3-4 | Config (TOML), Cobra CLI skeleton |
| 3 | 5-8 | SQLite store (CRUD, search, threads, sync state) |
| 4 | 9 | OS keyring token storage |
| 5 | 10-11 | Gmail OAuth2 + API client |
| 6 | 12 | Sync engine (initial + incremental) |
| 7 | 13-18 | TUI (app shell, sidebar, inbox, reader, composer, search) |
| 8 | 19-20 | Wire together, README |

Total: ~20 tasks, each 2-15 minutes. Phases 1-6 are backend (testable without TUI). Phase 7 builds the UI. Phase 8 integrates.
