# termail

A terminal-based email client with Gmail support. Read, compose, reply, search, and manage emails from the command line or an interactive TUI.

## Features

- Interactive TUI with sidebar navigation, threaded view, and compose/reply/forward
- Full CLI for scripting and automation (every TUI action has a CLI equivalent)
- Multi-account support with per-session switching
- Full-text search via SQLite FTS5
- OAuth2 authentication with OS keyring token storage
- Incremental sync via Gmail History API

## Install

Requires Go 1.25+ and CGO (for SQLite).

```bash
CGO_ENABLED=1 go install -tags "fts5" github.com/zhengda-lu/termail/cmd/termail@latest
```

Or build from source:

```bash
git clone https://github.com/lu-zhengda/termail.git
cd termail
CGO_ENABLED=1 go build -tags "fts5" -o termail ./cmd/termail
cp termail ~/.local/bin/  # or wherever you prefer
```

## Quick Start

```bash
# Add a Gmail account (opens browser for OAuth)
termail account add

# Sync emails
termail sync

# Launch the interactive TUI
termail
```

## TUI Keybindings

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate up/down |
| `Enter` | Open thread |
| `Esc` | Go back |
| `@` | Switch account |
| `c` | Compose |
| `r` / `R` | Reply / Reply all |
| `f` | Forward |
| `a` | Archive |
| `d` | Trash |
| `s` | Star |
| `u` | Mark unread |
| `/` | Search |
| `t` | Toggle thread/flat view |
| `Tab` | Switch pane |
| `q` | Quit |

## CLI Commands

### Reading

```bash
termail list                              # List inbox threads
termail list --label SENT --limit 50      # List sent mail
termail list --account user@gmail.com     # List for specific account
termail read <thread-id>                  # Read a thread
termail search "quarterly report"         # Full-text search
termail labels                            # List all labels
```

### Composing

```bash
termail compose --to user@example.com --subject "Hello" --body "Message"
termail reply <message-id> --body "Thanks!"
termail reply <message-id> --body "Thanks!" --all
termail forward <message-id> --to other@example.com
```

### Actions

```bash
termail archive <message-id>
termail trash <message-id>
termail star <message-id>
termail star <message-id> --remove
termail mark-read <message-id>
termail mark-read <message-id> --unread
termail label-modify <message-id> --add STARRED --remove INBOX
```

### Account Management

```bash
termail account list
termail account add
termail account remove user@gmail.com
termail sync --account user@gmail.com
```

## Setup

You need your own Google Cloud OAuth credentials:

1. Go to [Google Cloud Console](https://console.cloud.google.com)
2. Create a project, enable the Gmail API
3. Create OAuth credentials (Desktop application type)
4. Provide them via config file or environment variables

### Option A: Config file

Config file: `~/.config/termail/config.toml`

```toml
[accounts]
default = "user@gmail.com"

[gmail]
client_id = "your-client-id.apps.googleusercontent.com"
client_secret = "your-client-secret"
```

### Option B: Environment variables

```bash
export GMAIL_CLIENT_ID="your-client-id.apps.googleusercontent.com"
export GMAIL_CLIENT_SECRET="your-client-secret"
```

## Architecture

```
cmd/termail/         Entry point
internal/
  cli/               Cobra commands (account, sync, list, read, compose, etc.)
  config/            TOML config loading, XDG paths
  domain/            Core types (Email, Thread, Account, Label)
  provider/          Email provider interface
    gmail/           Gmail API client, OAuth2, message mapping
  store/             Storage interface
    sqlite/          SQLite implementation with FTS5 search
  tui/               Bubble Tea interactive UI
  app/               Sync service (initial + incremental)
```

**Data storage:**
- Database: `~/.local/share/termail/termail.db` (SQLite)
- Tokens: OS keyring (macOS Keychain / Linux secret-service / Windows Credential Manager)
- Config: `~/.config/termail/config.toml`

## Tech Stack

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) + [Lip Gloss](https://github.com/charmbracelet/lipgloss) for the TUI
- [Cobra](https://github.com/spf13/cobra) for CLI
- [mattn/go-sqlite3](https://github.com/mattn/go-sqlite3) with FTS5 for storage and search
- [zalando/go-keyring](https://github.com/zalando/go-keyring) for token storage
- [Google Gmail API](https://developers.google.com/gmail/api) via `google.golang.org/api`

## License

Private repository.
