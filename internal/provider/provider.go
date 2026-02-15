package provider

import (
	"context"

	"github.com/zhengda-lu/termail/internal/domain"
)

type ListOptions struct {
	PageToken  string
	MaxResults int
	LabelIDs   []string
	Query      string
}

type EmailProvider interface {
	Authenticate(ctx context.Context) error
	IsAuthenticated() bool

	ListMessages(ctx context.Context, opts ListOptions) ([]domain.Email, string, error)
	GetMessage(ctx context.Context, id string) (*domain.Email, error)
	SendMessage(ctx context.Context, email *domain.Email) error

	ListThreads(ctx context.Context, opts ListOptions) ([]domain.Thread, string, error)
	GetThread(ctx context.Context, id string) (*domain.Thread, error)

	ModifyLabels(ctx context.Context, msgID string, add, remove []string) error
	TrashMessage(ctx context.Context, msgID string) error
	MarkRead(ctx context.Context, msgID string, read bool) error

	ListLabels(ctx context.Context) ([]domain.Label, error)
	Search(ctx context.Context, query string, opts ListOptions) ([]domain.Email, string, error)

	History(ctx context.Context, startHistoryID uint64) ([]HistoryEvent, uint64, error)
}

type HistoryEventType int

const (
	HistoryMessageAdded HistoryEventType = iota
	HistoryMessageDeleted
	HistoryLabelsAdded
	HistoryLabelsRemoved
)

type HistoryEvent struct {
	Type      HistoryEventType
	MessageID string
	LabelIDs  []string
}
