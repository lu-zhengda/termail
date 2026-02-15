package app

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/zhengda-lu/termail/internal/provider"
	"github.com/zhengda-lu/termail/internal/store"
)

// SyncService orchestrates synchronization between an email provider and the
// local store for a single account.
type SyncService struct {
	store     store.Store
	provider  provider.EmailProvider
	accountID string
}

// NewSyncService creates a SyncService that syncs the given account between
// the provider and the local store.
func NewSyncService(s store.Store, p provider.EmailProvider, accountID string) *SyncService {
	return &SyncService{store: s, provider: p, accountID: accountID}
}

// InitialSync performs a full initial sync, fetching up to count messages from
// the provider and persisting them locally along with all labels.
func (s *SyncService) InitialSync(ctx context.Context, count int) error {
	// Sync labels first.
	labels, err := s.provider.ListLabels(ctx)
	if err != nil {
		return fmt.Errorf("failed to list labels: %w", err)
	}
	for i := range labels {
		labels[i].AccountID = s.accountID
		if err := s.store.UpsertLabel(ctx, &labels[i]); err != nil {
			return fmt.Errorf("failed to upsert label %s: %w", labels[i].ID, err)
		}
	}
	log.Printf("[sync] synced %d labels for account %s", len(labels), s.accountID)

	// Fetch messages in pages.
	const batchSize = 100
	var (
		pageToken string
		fetched   int
	)
	for fetched < count {
		remaining := count - fetched
		limit := min(batchSize, remaining)

		msgs, nextToken, err := s.provider.ListMessages(ctx, provider.ListOptions{
			PageToken:  pageToken,
			MaxResults: limit,
		})
		if err != nil {
			return fmt.Errorf("failed to list messages (fetched %d so far): %w", fetched, err)
		}

		for i := range msgs {
			if err := s.store.UpsertEmail(ctx, &msgs[i], s.accountID); err != nil {
				return fmt.Errorf("failed to upsert email %s: %w", msgs[i].ID, err)
			}
		}

		fetched += len(msgs)
		log.Printf("[sync] fetched %d/%d messages for account %s", fetched, count, s.accountID)

		if nextToken == "" || len(msgs) == 0 {
			break
		}
		pageToken = nextToken
	}

	// Save sync state.
	if err := s.store.SetSyncState(ctx, &store.SyncState{
		AccountID: s.accountID,
		HistoryID: 0,
		LastSync:  time.Now().Unix(),
	}); err != nil {
		return fmt.Errorf("failed to save sync state: %w", err)
	}

	log.Printf("[sync] initial sync complete: %d messages for account %s", fetched, s.accountID)
	return nil
}

// IncrementalSync performs a delta sync using the provider's history API.
// If no prior sync state exists (historyID == 0), it falls back to an
// InitialSync of 500 messages.
func (s *SyncService) IncrementalSync(ctx context.Context) error {
	state, err := s.store.GetSyncState(ctx, s.accountID)
	if err != nil {
		return fmt.Errorf("failed to get sync state: %w", err)
	}

	if state == nil || state.HistoryID == 0 {
		log.Printf("[sync] no history ID found, falling back to initial sync for account %s", s.accountID)
		return s.InitialSync(ctx, 500)
	}

	events, newHistoryID, err := s.provider.History(ctx, state.HistoryID)
	if err != nil {
		return fmt.Errorf("failed to fetch history: %w", err)
	}

	var added, deleted, modified int

	for _, event := range events {
		switch event.Type {
		case provider.HistoryMessageAdded:
			msg, err := s.provider.GetMessage(ctx, event.MessageID)
			if err != nil {
				return fmt.Errorf("failed to get added message %s: %w", event.MessageID, err)
			}
			if err := s.store.UpsertEmail(ctx, msg, s.accountID); err != nil {
				return fmt.Errorf("failed to upsert added message %s: %w", event.MessageID, err)
			}
			added++

		case provider.HistoryMessageDeleted:
			if err := s.store.DeleteEmail(ctx, event.MessageID); err != nil {
				return fmt.Errorf("failed to delete message %s: %w", event.MessageID, err)
			}
			deleted++

		case provider.HistoryLabelsAdded, provider.HistoryLabelsRemoved:
			msg, err := s.provider.GetMessage(ctx, event.MessageID)
			if err != nil {
				return fmt.Errorf("failed to get message %s for label update: %w", event.MessageID, err)
			}
			if err := s.store.SetEmailLabels(ctx, msg.ID, msg.Labels); err != nil {
				return fmt.Errorf("failed to set labels for message %s: %w", event.MessageID, err)
			}
			modified++
		}
	}

	// Update sync state with new history ID.
	if err := s.store.SetSyncState(ctx, &store.SyncState{
		AccountID: s.accountID,
		HistoryID: newHistoryID,
		LastSync:  time.Now().Unix(),
	}); err != nil {
		return fmt.Errorf("failed to update sync state: %w", err)
	}

	log.Printf("[sync] incremental sync complete for account %s: %d added, %d deleted, %d modified",
		s.accountID, added, deleted, modified)
	return nil
}
