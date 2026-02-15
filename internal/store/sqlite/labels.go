package sqlite

import (
	"context"
	"fmt"

	"github.com/lu-zhengda/termail/internal/domain"
)

// UpsertLabel inserts or updates a label.
func (s *DB) UpsertLabel(ctx context.Context, label *domain.Label) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO labels (id, account_id, name, type, color)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name  = excluded.name,
			type  = excluded.type,
			color = excluded.color`,
		label.ID, label.AccountID, label.Name, label.Type, label.Color,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert label: %w", err)
	}
	return nil
}

// ListLabels returns all labels for an account, ordered by name.
func (s *DB) ListLabels(ctx context.Context, accountID string) ([]domain.Label, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, account_id, name, type, color FROM labels WHERE account_id = ? ORDER BY name`,
		accountID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list labels: %w", err)
	}
	defer rows.Close()

	var labels []domain.Label
	for rows.Next() {
		var l domain.Label
		if err := rows.Scan(&l.ID, &l.AccountID, &l.Name, &l.Type, &l.Color); err != nil {
			return nil, fmt.Errorf("failed to scan label: %w", err)
		}
		labels = append(labels, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate labels: %w", err)
	}

	return labels, nil
}
