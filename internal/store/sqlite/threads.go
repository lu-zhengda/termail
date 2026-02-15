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

// GetThread retrieves a thread by ID, including all its messages ordered by date ascending.
func (s *DB) GetThread(ctx context.Context, threadID string, accountID string) (*domain.Thread, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, thread_id, from_addr, from_name, to_addrs, cc_addrs,
			subject, body_text, body_html, date, is_read, is_starred, in_reply_to
		FROM emails
		WHERE thread_id = ? AND account_id = ?
		ORDER BY date ASC`, threadID, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to query thread %s: %w", threadID, err)
	}
	defer rows.Close()

	var messages []domain.Email
	for rows.Next() {
		var e domain.Email
		var fromAddr, fromName string
		var toJSON, ccJSON string
		var dateStr string

		if err := rows.Scan(
			&e.ID, &e.ThreadID, &fromAddr, &fromName, &toJSON, &ccJSON,
			&e.Subject, &e.Body, &e.BodyHTML, &dateStr,
			&e.IsRead, &e.IsStarred, &e.InReplyTo,
		); err != nil {
			return nil, fmt.Errorf("failed to scan thread message: %w", err)
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

		messages = append(messages, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate thread messages: %w", err)
	}

	if len(messages) == 0 {
		return nil, fmt.Errorf("thread %s not found: %w", threadID, sql.ErrNoRows)
	}

	// Build Thread struct from messages.
	first := messages[0]
	last := messages[len(messages)-1]

	snippet := last.Body
	if len(snippet) > 100 {
		snippet = snippet[:100]
	}

	return &domain.Thread{
		ID:       threadID,
		Subject:  first.Subject,
		Messages: messages,
		Snippet:  snippet,
		LastDate: last.Date,
	}, nil
}

// ListThreads returns threads grouped by thread_id, optionally filtered by label.
func (s *DB) ListThreads(ctx context.Context, opts store.ListEmailOptions) ([]domain.Thread, error) {
	var query string
	var args []any

	if opts.LabelID != "" {
		query = `
			SELECT e.thread_id,
				(SELECT e2.subject FROM emails e2 WHERE e2.thread_id = e.thread_id ORDER BY e2.date ASC LIMIT 1) AS first_subject,
				(SELECT e2.from_name FROM emails e2 WHERE e2.thread_id = e.thread_id ORDER BY e2.date ASC LIMIT 1) AS first_from_name,
				(SELECT e2.from_addr FROM emails e2 WHERE e2.thread_id = e.thread_id ORDER BY e2.date ASC LIMIT 1) AS first_from_addr,
				MAX(e.date) AS last_date,
				(SELECT e3.body_text FROM emails e3 WHERE e3.thread_id = e.thread_id ORDER BY e3.date DESC LIMIT 1) AS last_body,
				COUNT(*) AS msg_count,
				MIN(e.is_read) AS all_read
			FROM emails e
			JOIN email_labels el ON el.email_id = e.id
			WHERE e.account_id = ? AND el.label_id = ?
			GROUP BY e.thread_id
			ORDER BY last_date DESC`
		args = append(args, opts.AccountID, opts.LabelID)
	} else {
		query = `
			SELECT e.thread_id,
				(SELECT e2.subject FROM emails e2 WHERE e2.thread_id = e.thread_id ORDER BY e2.date ASC LIMIT 1) AS first_subject,
				(SELECT e2.from_name FROM emails e2 WHERE e2.thread_id = e.thread_id ORDER BY e2.date ASC LIMIT 1) AS first_from_name,
				(SELECT e2.from_addr FROM emails e2 WHERE e2.thread_id = e.thread_id ORDER BY e2.date ASC LIMIT 1) AS first_from_addr,
				MAX(e.date) AS last_date,
				(SELECT e3.body_text FROM emails e3 WHERE e3.thread_id = e.thread_id ORDER BY e3.date DESC LIMIT 1) AS last_body,
				COUNT(*) AS msg_count,
				MIN(e.is_read) AS all_read
			FROM emails e
			WHERE e.account_id = ?
			GROUP BY e.thread_id
			ORDER BY last_date DESC`
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
		return nil, fmt.Errorf("failed to list threads: %w", err)
	}
	defer rows.Close()

	var threads []domain.Thread
	for rows.Next() {
		var t domain.Thread
		var fromName, fromAddr sql.NullString
		var lastDateStr string
		var lastBody sql.NullString
		var msgCount int
		var allRead bool

		if err := rows.Scan(&t.ID, &t.Subject, &fromName, &fromAddr, &lastDateStr, &lastBody, &msgCount, &allRead); err != nil {
			return nil, fmt.Errorf("failed to scan thread row: %w", err)
		}

		parsedDate, err := time.Parse(time.RFC3339, lastDateStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse thread last date: %w", err)
		}
		t.LastDate = parsedDate

		if lastBody.Valid {
			t.Snippet = lastBody.String
			if len(t.Snippet) > 100 {
				t.Snippet = t.Snippet[:100]
			}
		}

		t.FromAddress = domain.Address{
			Name:  fromName.String,
			Email: fromAddr.String,
		}
		t.TotalCount = msgCount
		t.HasUnread = !allRead

		threads = append(threads, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate threads: %w", err)
	}

	return threads, nil
}
