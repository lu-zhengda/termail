package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/zhengda-lu/termail/internal/domain"
)

// SearchEmails performs a full-text search across emails using FTS5.
func (s *DB) SearchEmails(ctx context.Context, query string, accountID string) ([]domain.Email, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT e.id, e.thread_id, e.from_addr, e.from_name, e.to_addrs, e.cc_addrs,
			e.subject, e.body_text, e.body_html, e.date, e.is_read, e.is_starred, e.in_reply_to
		FROM emails e
		JOIN emails_fts fts ON fts.rowid = e.rowid
		WHERE emails_fts MATCH ? AND e.account_id = ?
		ORDER BY rank`, query, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to search emails: %w", err)
	}
	defer rows.Close()

	var emails []domain.Email
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
			return nil, fmt.Errorf("failed to scan search result: %w", err)
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

		emails = append(emails, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate search results: %w", err)
	}

	return emails, nil
}
