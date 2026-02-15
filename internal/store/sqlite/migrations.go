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
