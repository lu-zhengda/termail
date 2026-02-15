package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lu-zhengda/termail/internal/store"
)

// GetSyncState retrieves the sync state for an account.
// If no state exists, it returns an empty SyncState with the AccountID set.
func (s *DB) GetSyncState(ctx context.Context, accountID string) (*store.SyncState, error) {
	var state store.SyncState
	var lastSync time.Time
	err := s.db.QueryRowContext(ctx,
		`SELECT account_id, history_id, last_sync FROM sync_state WHERE account_id = ?`,
		accountID,
	).Scan(&state.AccountID, &state.HistoryID, &lastSync)

	if errors.Is(err, sql.ErrNoRows) {
		return &store.SyncState{AccountID: accountID}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get sync state for %s: %w", accountID, err)
	}

	state.LastSync = lastSync.Unix()
	return &state, nil
}

// SetSyncState inserts or updates the sync state for an account.
func (s *DB) SetSyncState(ctx context.Context, state *store.SyncState) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO sync_state (account_id, history_id, last_sync)
		VALUES (?, ?, ?)
		ON CONFLICT(account_id) DO UPDATE SET
			history_id = excluded.history_id,
			last_sync  = excluded.last_sync`,
		state.AccountID, state.HistoryID, state.LastSync,
	)
	if err != nil {
		return fmt.Errorf("failed to set sync state for %s: %w", state.AccountID, err)
	}
	return nil
}
