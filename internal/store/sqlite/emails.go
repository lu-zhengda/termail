package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/zhengda-lu/termail/internal/domain"
	"github.com/zhengda-lu/termail/internal/store"
)

// UpsertEmail inserts or updates an email and its label associations.
func (s *DB) UpsertEmail(ctx context.Context, email *domain.Email, accountID string) error {
	toJSON, err := json.Marshal(email.To)
	if err != nil {
		return fmt.Errorf("failed to marshal To addresses: %w", err)
	}
	ccJSON, err := json.Marshal(email.CC)
	if err != nil {
		return fmt.Errorf("failed to marshal CC addresses: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO emails (id, account_id, thread_id, from_addr, from_name, to_addrs, cc_addrs,
			subject, body_text, body_html, date, is_read, is_starred, in_reply_to)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			account_id = excluded.account_id,
			thread_id  = excluded.thread_id,
			from_addr  = excluded.from_addr,
			from_name  = excluded.from_name,
			to_addrs   = excluded.to_addrs,
			cc_addrs   = excluded.cc_addrs,
			subject    = excluded.subject,
			body_text  = excluded.body_text,
			body_html  = excluded.body_html,
			date       = excluded.date,
			is_read    = excluded.is_read,
			is_starred = excluded.is_starred,
			in_reply_to = excluded.in_reply_to`,
		email.ID, accountID, email.ThreadID,
		email.From.Email, email.From.Name,
		string(toJSON), string(ccJSON),
		email.Subject, email.Body, email.BodyHTML,
		email.Date.Format(time.RFC3339),
		email.IsRead, email.IsStarred, email.InReplyTo,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert email: %w", err)
	}

	// Delete existing labels, then reinsert.
	if _, err := tx.ExecContext(ctx, `DELETE FROM email_labels WHERE email_id = ?`, email.ID); err != nil {
		return fmt.Errorf("failed to delete email labels: %w", err)
	}

	for _, labelID := range email.Labels {
		if _, err := tx.ExecContext(ctx, `INSERT INTO email_labels (email_id, label_id) VALUES (?, ?)`,
			email.ID, labelID); err != nil {
			return fmt.Errorf("failed to insert email label: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit email upsert: %w", err)
	}
	return nil
}

// GetEmail retrieves a single email by ID, including its labels.
func (s *DB) GetEmail(ctx context.Context, id string) (*domain.Email, error) {
	var e domain.Email
	var fromAddr, fromName string
	var toJSON, ccJSON string
	var dateStr string

	err := s.db.QueryRowContext(ctx, `
		SELECT id, thread_id, from_addr, from_name, to_addrs, cc_addrs,
			subject, body_text, body_html, date, is_read, is_starred, in_reply_to
		FROM emails WHERE id = ?`, id,
	).Scan(
		&e.ID, &e.ThreadID, &fromAddr, &fromName, &toJSON, &ccJSON,
		&e.Subject, &e.Body, &e.BodyHTML, &dateStr,
		&e.IsRead, &e.IsStarred, &e.InReplyTo,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get email %s: %w", id, err)
	}

	e.From = domain.Address{Name: fromName, Email: fromAddr}

	if toJSON != "" {
		if err := json.Unmarshal([]byte(toJSON), &e.To); err != nil {
			return nil, fmt.Errorf("failed to unmarshal To addresses: %w", err)
		}
	}
	if ccJSON != "" {
		if err := json.Unmarshal([]byte(ccJSON), &e.CC); err != nil {
			return nil, fmt.Errorf("failed to unmarshal CC addresses: %w", err)
		}
	}

	parsedDate, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse email date: %w", err)
	}
	e.Date = parsedDate

	// Fetch labels.
	rows, err := s.db.QueryContext(ctx, `SELECT label_id FROM email_labels WHERE email_id = ?`, id)
	if err != nil {
		return nil, fmt.Errorf("failed to query email labels: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var labelID string
		if err := rows.Scan(&labelID); err != nil {
			return nil, fmt.Errorf("failed to scan email label: %w", err)
		}
		e.Labels = append(e.Labels, labelID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate email labels: %w", err)
	}

	return &e, nil
}

// ListEmails returns a summary list of emails, optionally filtered by label.
func (s *DB) ListEmails(ctx context.Context, opts store.ListEmailOptions) ([]domain.Email, error) {
	var query string
	var args []any

	if opts.LabelID != "" {
		query = `
			SELECT e.id, e.thread_id, e.from_addr, e.from_name, e.subject, e.snippet,
				e.date, e.is_read, e.is_starred
			FROM emails e
			JOIN email_labels el ON el.email_id = e.id
			WHERE e.account_id = ? AND el.label_id = ?
			ORDER BY e.date DESC`
		args = append(args, opts.AccountID, opts.LabelID)
	} else {
		query = `
			SELECT e.id, e.thread_id, e.from_addr, e.from_name, e.subject, e.snippet,
				e.date, e.is_read, e.is_starred
			FROM emails e
			WHERE e.account_id = ?
			ORDER BY e.date DESC`
		args = append(args, opts.AccountID)
	}

	if opts.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, opts.Limit)
	}
	if opts.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, opts.Offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list emails: %w", err)
	}
	defer rows.Close()

	var emails []domain.Email
	for rows.Next() {
		var e domain.Email
		var fromAddr, fromName string
		var snippet sql.NullString
		var dateStr string

		if err := rows.Scan(
			&e.ID, &e.ThreadID, &fromAddr, &fromName, &e.Subject, &snippet,
			&dateStr, &e.IsRead, &e.IsStarred,
		); err != nil {
			return nil, fmt.Errorf("failed to scan email row: %w", err)
		}

		e.From = domain.Address{Name: fromName, Email: fromAddr}

		parsedDate, err := time.Parse(time.RFC3339, dateStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse email date: %w", err)
		}
		e.Date = parsedDate
		emails = append(emails, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate emails: %w", err)
	}

	return emails, nil
}

// SetEmailRead updates the is_read flag for a single email.
func (s *DB) SetEmailRead(ctx context.Context, emailID string, read bool) error {
	_, err := s.db.ExecContext(ctx, `UPDATE emails SET is_read = ? WHERE id = ?`, read, emailID)
	if err != nil {
		return fmt.Errorf("failed to set email %s read=%v: %w", emailID, read, err)
	}
	return nil
}

// SetThreadRead updates the is_read flag for all emails in a thread.
func (s *DB) SetThreadRead(ctx context.Context, threadID string, read bool) error {
	_, err := s.db.ExecContext(ctx, `UPDATE emails SET is_read = ? WHERE thread_id = ?`, read, threadID)
	if err != nil {
		return fmt.Errorf("failed to set thread %s read=%v: %w", threadID, read, err)
	}
	return nil
}

// DeleteEmail removes an email by ID.
func (s *DB) DeleteEmail(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM emails WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete email %s: %w", id, err)
	}
	return nil
}

// SetEmailLabels replaces the label set for an email.
func (s *DB) SetEmailLabels(ctx context.Context, emailID string, labelIDs []string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM email_labels WHERE email_id = ?`, emailID); err != nil {
		return fmt.Errorf("failed to delete email labels: %w", err)
	}

	for _, labelID := range labelIDs {
		if _, err := tx.ExecContext(ctx, `INSERT INTO email_labels (email_id, label_id) VALUES (?, ?)`,
			emailID, labelID); err != nil {
			return fmt.Errorf("failed to insert email label: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit label update: %w", err)
	}
	return nil
}
