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
