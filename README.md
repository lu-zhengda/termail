# termail

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Platform: macOS](https://img.shields.io/badge/Platform-macOS-lightgrey.svg)](https://github.com/lu-zhengda/termail)
[![Homebrew](https://img.shields.io/badge/Homebrew-lu--zhengda/tap-orange.svg)](https://github.com/lu-zhengda/homebrew-tap)

A terminal-based email client with Gmail support. Read, compose, reply, search, and manage emails from the command line or an interactive TUI.

Full-text search via SQLite FTS5, multi-account support, incremental sync via Gmail History API, and OAuth2 tokens stored securely in the OS keyring.

## Install

```bash
brew install lu-zhengda/tap/termail
```

Or build from source (requires Go 1.25+ and CGO):

```bash
git clone https://github.com/lu-zhengda/termail.git
cd termail
CGO_ENABLED=1 go build -tags fts5 -o termail ./cmd/termail
```

## Setup

termail requires your own Google Cloud OAuth credentials to access Gmail. This is a one-time setup.

### 1. Create a Google Cloud project

1. Go to [Google Cloud Console](https://console.cloud.google.com)
2. Click **Select a project** (top bar) > **New Project**
3. Name it anything (e.g. `termail`) and click **Create**

### 2. Enable the Gmail API

1. In the sidebar, go to **APIs & Services** > **Library**
2. Search for **Gmail API**
3. Click **Gmail API** > **Enable**

### 3. Configure the OAuth consent screen

1. Go to **APIs & Services** > **OAuth consent screen**
2. Select **External** user type, click **Create**
3. Fill in the required fields (App name, User support email, Developer contact)
4. On the **Scopes** page, click **Add or Remove Scopes** and add:
   - `https://www.googleapis.com/auth/gmail.modify`
   - `https://www.googleapis.com/auth/gmail.compose`
   - `https://www.googleapis.com/auth/gmail.send`
5. On the **Test users** page, add your Gmail address(es)
6. Click **Save and Continue** through to the end

> **Note:** While the app is in "Testing" status, only the test users you added can authenticate. This is fine for personal use. You do **not** need to publish or verify the app.

### 4. Create OAuth credentials

1. Go to **APIs & Services** > **Credentials**
2. Click **Create Credentials** > **OAuth client ID**
3. Application type: **Desktop app**
4. Name it anything, click **Create**
5. Copy the **Client ID** and **Client Secret**

### 5. Configure termail

**Option A: Config file** (`~/.config/termail/config.toml`)

```toml
[gmail]
client_id = "123456789-abc.apps.googleusercontent.com"
client_secret = "GOCSPX-xxxxx"

[accounts]
default = "user@gmail.com"
```

**Option B: Environment variables**

```bash
export GMAIL_CLIENT_ID="123456789-abc.apps.googleusercontent.com"
export GMAIL_CLIENT_SECRET="GOCSPX-xxxxx"
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

## Usage

```
$ termail list --limit 5
UNREAD  FROM          SUBJECT                                     DATE          MSGS  THREAD_ID
*       GitHub        [GitHub] A personal access token has been…   Feb 14, 2026  4     19c5d4f2ea3c4478
        Citi Alerts   Your daily alerts summary                   Feb 15, 2026  1     19c5ecc5e553f0b6
        Airmart Team  Your order has been completed                Feb 15, 2026  1     19c5f633501ea8e9

$ termail search "quarterly report"
FROM     SUBJECT                    DATE          ID
Finance  Q4 Quarterly Report 2025   Jan 10, 2026  19c12345abcdef

$ termail account list
ID                   EMAIL                PROVIDER  CREATED
user@gmail.com       user@gmail.com       gmail     2026-02-15
work@gmail.com       work@gmail.com       gmail     2026-02-15
```

## Commands

| Command | Description | Example |
|---------|-------------|---------|
| *(no command)* | Launch interactive TUI | `termail` |
| `list` | List email threads | `termail list --label SENT --limit 50` |
| `read` | Read a thread | `termail read <thread-id>` |
| `search` | Full-text search | `termail search "quarterly report"` |
| `labels` | List all labels | `termail labels` |
| `compose` | Send a new email | `termail compose --to user@example.com --subject "Hi" --body "Hello"` |
| `reply` | Reply to an email | `termail reply <message-id> --body "Thanks!" --all` |
| `forward` | Forward an email | `termail forward <message-id> --to other@example.com` |
| `archive` | Archive (remove from Inbox) | `termail archive <message-id>` |
| `trash` | Move to trash | `termail trash <message-id>` |
| `star` | Star/unstar | `termail star <message-id> --remove` |
| `mark-read` | Mark read/unread | `termail mark-read <message-id> --unread` |
| `label-modify` | Add/remove labels | `termail label-modify <id> --add STARRED --remove INBOX` |
| `account add` | Add Gmail account | `termail account add` |
| `account list` | List accounts | `termail account list` |
| `account remove` | Remove account | `termail account remove user@gmail.com` |
| `sync` | Sync emails | `termail sync --account user@gmail.com` |

## TUI Keybindings

Launch `termail` without arguments for interactive mode:

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

## Data Storage

| Data | Location |
|------|----------|
| Database | `~/.local/share/termail/termail.db` (SQLite with FTS5) |
| OAuth tokens | OS keyring (macOS Keychain / Linux secret-service) |
| Config | `~/.config/termail/config.toml` |

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

## Claude Code

termail is part of the [macos-toolkit](https://github.com/lu-zhengda/macos-toolkit) Claude Code plugin. Install the plugin to let Claude read, compose, search, and manage your email using natural language.

## License

[MIT](LICENSE)
