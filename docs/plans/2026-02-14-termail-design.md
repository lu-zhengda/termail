# termail Design Document

**Date:** 2026-02-14
**Status:** Approved

## Overview

termail is a terminal-based email client written in Go, targeting Gmail as the initial provider. It uses Bubble Tea for the TUI, Gmail REST API with OAuth2 for email access, and SQLite with FTS5 for local caching and search.

## Decisions

| Decision | Choice |
|----------|--------|
| Language | Go |
| Scope | Full email client (read, compose, reply, forward, search, labels) |
| TUI Framework | Bubble Tea (charmbracelet/bubbletea) |
| Gmail Access | Gmail REST API via OAuth2 |
| Local Storage | SQLite with FTS5 for full-text search |
| View Mode | Thread + flat, user-switchable |
| Architecture | Layered monolith with provider interfaces |

## Architecture

```
┌──────────────────────────┐
│   TUI Layer (Bubble Tea) │  Views, key bindings, rendering
├──────────────────────────┤
│   Application Layer      │  Use cases: sync, compose, search
├──────────────────────────┤
│   Domain Layer           │  Email, Thread, Account, Label models
├──────────────────────────┤
│   Infrastructure Layer   │  Gmail API client, SQLite store, OAuth
└──────────────────────────┘
```

## Project Structure

```
termail/
├── cmd/
│   └── termail/
│       └── main.go
├── internal/
│   ├── app/
│   │   ├── sync.go
│   │   ├── compose.go
│   │   └── search.go
│   ├── domain/
│   │   ├── email.go
│   │   ├── thread.go
│   │   ├── account.go
│   │   └── label.go
│   ├── provider/
│   │   ├── provider.go
│   │   └── gmail/
│   │       ├── client.go
│   │       ├── oauth.go
│   │       └── mapper.go
│   ├── store/
│   │   ├── store.go
│   │   ├── sqlite/
│   │   │   ├── sqlite.go
│   │   │   ├── migrations.go
│   │   │   └── queries.go
│   │   └── keyring.go
│   └── tui/
│       ├── app.go
│       ├── inbox.go
│       ├── reader.go
│       ├── composer.go
│       ├── sidebar.go
│       ├── search.go
│       ├── statusbar.go
│       ├── styles.go
│       └── keys.go
├── go.mod
├── go.sum
└── README.md
```

## Domain Models

### Email

```go
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
```

### Thread

```go
type Thread struct {
    ID       string
    Subject  string
    Messages []Email
    Labels   []string
    Snippet  string
    LastDate time.Time
}
```

### Provider Interface

```go
type EmailProvider interface {
    Authenticate(ctx context.Context) error
    IsAuthenticated() bool
    ListMessages(ctx context.Context, opts ListOptions) ([]domain.Email, string, error)
    GetMessage(ctx context.Context, id string) (*domain.Email, error)
    SendMessage(ctx context.Context, email *domain.Email) error
    ListThreads(ctx context.Context, opts ListOptions) ([]domain.Thread, string, error)
    GetThread(ctx context.Context, id string) (*domain.Thread, error)
    ModifyLabels(ctx context.Context, id string, add, remove []string) error
    TrashMessage(ctx context.Context, id string) error
    MarkRead(ctx context.Context, id string, read bool) error
    ListLabels(ctx context.Context) ([]domain.Label, error)
    Search(ctx context.Context, query string, opts ListOptions) ([]domain.Email, string, error)
}
```

## OAuth2 Flow

1. User runs `termail account add`
2. App starts local HTTP server on random port
3. Opens browser to Google consent screen
4. User grants permissions (scopes: gmail.readonly, gmail.send, gmail.modify)
5. Google redirects to localhost callback with auth code
6. App exchanges code for access + refresh tokens
7. Tokens stored in OS keyring (zalando/go-keyring)

## SQLite Schema

```sql
CREATE TABLE accounts (
    id          TEXT PRIMARY KEY,
    email       TEXT NOT NULL UNIQUE,
    provider    TEXT NOT NULL DEFAULT 'gmail',
    display_name TEXT,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE emails (
    id          TEXT PRIMARY KEY,
    account_id  TEXT NOT NULL REFERENCES accounts(id),
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

CREATE TABLE email_labels (
    email_id    TEXT NOT NULL REFERENCES emails(id),
    label_id    TEXT NOT NULL,
    PRIMARY KEY (email_id, label_id)
);

CREATE TABLE labels (
    id          TEXT PRIMARY KEY,
    account_id  TEXT NOT NULL REFERENCES accounts(id),
    name        TEXT NOT NULL,
    type        TEXT,
    color       TEXT
);

CREATE TABLE attachments (
    id          TEXT PRIMARY KEY,
    email_id    TEXT NOT NULL REFERENCES emails(id),
    filename    TEXT,
    mime_type   TEXT,
    size        INTEGER
);

CREATE VIRTUAL TABLE emails_fts USING fts5(
    subject, body_text, from_addr, from_name,
    content='emails', content_rowid='rowid'
);

CREATE TABLE sync_state (
    account_id  TEXT PRIMARY KEY REFERENCES accounts(id),
    history_id  INTEGER,
    last_sync   DATETIME
);
```

## Sync Strategy

- **Initial sync:** Fetch last 500 messages, store historyId
- **Incremental sync:** Use `users.history.list(startHistoryId)` for deltas
- **Background polling:** Every 5 minutes (configurable)

## TUI Layout

```
┌─ termail ─────────────────────────────────────────────────────┐
│ [Sidebar]      │ [Message List]                               │
│                │                                              │
│  ▶ Inbox (12)  │  ★ John Doe        Re: Project update  2h   │
│    Starred     │    Jane Smith      Meeting notes       5h   │
│    Sent        │    GitHub     [PR] Fix auth bug       1d   │
│    Drafts (1)  │    ...                                      │
│    Trash       │                                              │
│  ──────────    │──────────────────────────────────────────────│
│  Labels:       │ [Email Reader / Compose Pane]                │
│    Work        │                                              │
│    Personal    │  From: John Doe <john@example.com>           │
│    Projects    │  Date: Feb 14, 2026 10:30 AM                 │
│                │  Subject: Re: Project update                 │
│                │                                              │
│                │  Hey, just wanted to follow up on...          │
│                │                                              │
├────────────────┴──────────────────────────────────────────────┤
│ [Status Bar]  Synced 2m ago │ j/k:navigate │ Enter:read │ ?  │
└───────────────────────────────────────────────────────────────┘
```

## Key Bindings

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate up/down |
| `Enter` | Open email/thread |
| `Esc` | Back / close |
| `c` | Compose |
| `r` | Reply |
| `R` | Reply all |
| `f` | Forward |
| `a` | Archive |
| `d` | Trash |
| `s` | Star/unstar |
| `u` | Mark unread |
| `l` | Apply label |
| `/` | Search |
| `Tab` | Switch pane focus |
| `t` | Toggle thread/flat view |
| `gi` | Go to inbox |
| `gs` | Go to starred |
| `gd` | Go to drafts |
| `?` | Help |
| `q` | Quit |

## Dependencies

| Package | Purpose |
|---------|---------|
| charmbracelet/bubbletea | TUI framework |
| charmbracelet/lipgloss | Terminal styling |
| charmbracelet/bubbles | Reusable components |
| charmbracelet/glamour | Markdown rendering |
| google.golang.org/api/gmail/v1 | Gmail API client |
| golang.org/x/oauth2 | OAuth2 |
| mattn/go-sqlite3 | SQLite with FTS5 |
| zalando/go-keyring | OS keyring |
| jaytaylor/html2text | HTML to text |
| spf13/cobra | CLI commands |

## CLI Commands

```
termail                     # Launch TUI
termail account add         # Add Gmail account (OAuth)
termail account list        # List accounts
termail account remove      # Remove account
termail sync                # Manual sync
termail config              # Edit config
```

## Error Handling

- **Network errors:** Status bar notification, auto-retry with exponential backoff, TUI reads from local cache
- **Token expiry:** Auto-refresh via refresh token, prompt re-auth if refresh fails
- **SQLite errors:** Fatal on open/migration, wrapped errors with context
- **Rate limits:** Respect 429 responses, back off per Google guidelines

## Config

Location: `~/.config/termail/config.toml`
Database: `~/.local/share/termail/termail.db`

```toml
[sync]
interval = "5m"
initial_count = 500

[ui]
default_view = "thread"
theme = "default"

[accounts]
default = "user@gmail.com"
```
