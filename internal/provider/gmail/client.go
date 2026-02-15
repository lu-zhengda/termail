package gmail

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/zhengda-lu/termail/internal/domain"
	"github.com/zhengda-lu/termail/internal/provider"
	"github.com/zhengda-lu/termail/internal/store"
	"golang.org/x/oauth2"
	gmailapi "google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

const userID = "me"

// Provider implements the provider.EmailProvider interface for Gmail.
type Provider struct {
	tokenStore *store.KeyringTokenStore
	accountID  string
	service    *gmailapi.Service
	token      *oauth2.Token
}

// New creates a new Gmail provider for the given account.
func New(accountID string, tokenStore *store.KeyringTokenStore) *Provider {
	return &Provider{
		accountID:  accountID,
		tokenStore: tokenStore,
	}
}

// Authenticate runs the OAuth2 flow, saves the token, and initializes the Gmail service.
func (p *Provider) Authenticate(ctx context.Context) error {
	token, err := authenticate(ctx)
	if err != nil {
		return fmt.Errorf("failed to authenticate gmail: %w", err)
	}

	if err := p.tokenStore.SaveToken(p.accountID, token); err != nil {
		return fmt.Errorf("failed to save gmail token: %w", err)
	}

	p.token = token
	srv, err := gmailapi.NewService(ctx, option.WithTokenSource(oauthConfig.TokenSource(ctx, token)))
	if err != nil {
		return fmt.Errorf("failed to create gmail service: %w", err)
	}
	p.service = srv
	return nil
}

// IsAuthenticated returns true if the Gmail service is initialized.
func (p *Provider) IsAuthenticated() bool {
	return p.service != nil
}

// initService loads the token from the keyring and creates the Gmail service.
func (p *Provider) initService(ctx context.Context) error {
	token, err := p.tokenStore.LoadToken(p.accountID)
	if err != nil {
		return fmt.Errorf("failed to load gmail token: %w", err)
	}

	p.token = token
	srv, err := gmailapi.NewService(ctx, option.WithTokenSource(oauthConfig.TokenSource(ctx, token)))
	if err != nil {
		return fmt.Errorf("failed to create gmail service: %w", err)
	}
	p.service = srv
	return nil
}

// ensureService lazily initializes the Gmail service if not already done.
func (p *Provider) ensureService(ctx context.Context) error {
	if p.service != nil {
		return nil
	}
	return p.initService(ctx)
}

// ListMessages returns a page of emails matching the given options.
func (p *Provider) ListMessages(ctx context.Context, opts provider.ListOptions) ([]domain.Email, string, error) {
	if err := p.ensureService(ctx); err != nil {
		return nil, "", fmt.Errorf("failed to ensure gmail service: %w", err)
	}

	call := p.service.Users.Messages.List(userID)
	if opts.MaxResults > 0 {
		call = call.MaxResults(int64(opts.MaxResults))
	}
	if opts.PageToken != "" {
		call = call.PageToken(opts.PageToken)
	}
	if len(opts.LabelIDs) > 0 {
		call = call.LabelIds(opts.LabelIDs...)
	}
	if opts.Query != "" {
		call = call.Q(opts.Query)
	}

	resp, err := call.Context(ctx).Do()
	if err != nil {
		return nil, "", fmt.Errorf("failed to list gmail messages: %w", err)
	}

	emails := make([]domain.Email, 0, len(resp.Messages))
	for _, m := range resp.Messages {
		msg, err := p.service.Users.Messages.Get(userID, m.Id).
			Format("full").Context(ctx).Do()
		if err != nil {
			return nil, "", fmt.Errorf("failed to get gmail message %s: %w", m.Id, err)
		}
		emails = append(emails, *mapMessage(msg))
	}

	return emails, resp.NextPageToken, nil
}

// GetMessage returns a single email by ID.
func (p *Provider) GetMessage(ctx context.Context, id string) (*domain.Email, error) {
	if err := p.ensureService(ctx); err != nil {
		return nil, fmt.Errorf("failed to ensure gmail service: %w", err)
	}

	msg, err := p.service.Users.Messages.Get(userID, id).
		Format("full").Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get gmail message %s: %w", id, err)
	}

	email := mapMessage(msg)
	return email, nil
}

// SendMessage composes and sends an email via the Gmail API.
func (p *Provider) SendMessage(ctx context.Context, email *domain.Email) error {
	if err := p.ensureService(ctx); err != nil {
		return fmt.Errorf("failed to ensure gmail service: %w", err)
	}

	raw := buildRawMessage(email)
	encoded := base64.URLEncoding.EncodeToString([]byte(raw))

	msg := &gmailapi.Message{Raw: encoded}
	_, err := p.service.Users.Messages.Send(userID, msg).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to send gmail message: %w", err)
	}
	return nil
}

// buildRawMessage constructs an RFC 2822 message from a domain Email.
func buildRawMessage(email *domain.Email) string {
	var b strings.Builder

	b.WriteString("From: " + email.From.String() + "\r\n")

	to := make([]string, 0, len(email.To))
	for _, a := range email.To {
		to = append(to, a.String())
	}
	b.WriteString("To: " + strings.Join(to, ", ") + "\r\n")

	if len(email.CC) > 0 {
		cc := make([]string, 0, len(email.CC))
		for _, a := range email.CC {
			cc = append(cc, a.String())
		}
		b.WriteString("Cc: " + strings.Join(cc, ", ") + "\r\n")
	}

	if len(email.BCC) > 0 {
		bcc := make([]string, 0, len(email.BCC))
		for _, a := range email.BCC {
			bcc = append(bcc, a.String())
		}
		b.WriteString("Bcc: " + strings.Join(bcc, ", ") + "\r\n")
	}

	b.WriteString("Subject: " + email.Subject + "\r\n")

	if email.InReplyTo != "" {
		b.WriteString("In-Reply-To: " + email.InReplyTo + "\r\n")
	}

	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
	b.WriteString("\r\n")
	b.WriteString(email.Body)

	return b.String()
}

// ListThreads returns a page of threads matching the given options.
func (p *Provider) ListThreads(ctx context.Context, opts provider.ListOptions) ([]domain.Thread, string, error) {
	if err := p.ensureService(ctx); err != nil {
		return nil, "", fmt.Errorf("failed to ensure gmail service: %w", err)
	}

	call := p.service.Users.Threads.List(userID)
	if opts.MaxResults > 0 {
		call = call.MaxResults(int64(opts.MaxResults))
	}
	if opts.PageToken != "" {
		call = call.PageToken(opts.PageToken)
	}
	if len(opts.LabelIDs) > 0 {
		call = call.LabelIds(opts.LabelIDs...)
	}
	if opts.Query != "" {
		call = call.Q(opts.Query)
	}

	resp, err := call.Context(ctx).Do()
	if err != nil {
		return nil, "", fmt.Errorf("failed to list gmail threads: %w", err)
	}

	threads := make([]domain.Thread, 0, len(resp.Threads))
	for _, t := range resp.Threads {
		threads = append(threads, domain.Thread{
			ID:      t.Id,
			Snippet: t.Snippet,
		})
	}

	return threads, resp.NextPageToken, nil
}

// GetThread returns a thread with all its messages.
func (p *Provider) GetThread(ctx context.Context, id string) (*domain.Thread, error) {
	if err := p.ensureService(ctx); err != nil {
		return nil, fmt.Errorf("failed to ensure gmail service: %w", err)
	}

	t, err := p.service.Users.Threads.Get(userID, id).
		Format("full").Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get gmail thread %s: %w", id, err)
	}

	messages := make([]domain.Email, 0, len(t.Messages))
	for _, m := range t.Messages {
		messages = append(messages, *mapMessage(m))
	}

	thread := &domain.Thread{
		ID:       t.Id,
		Snippet:  t.Snippet,
		Messages: messages,
	}

	// Set subject and last date from messages
	if len(messages) > 0 {
		thread.Subject = messages[0].Subject
		thread.LastDate = messages[len(messages)-1].Date
	}

	// Collect unique labels from all messages
	labelSet := make(map[string]struct{})
	for _, m := range messages {
		for _, l := range m.Labels {
			labelSet[l] = struct{}{}
		}
	}
	labels := make([]string, 0, len(labelSet))
	for l := range labelSet {
		labels = append(labels, l)
	}
	thread.Labels = labels

	return thread, nil
}

// ModifyLabels adds and removes labels on a message.
func (p *Provider) ModifyLabels(ctx context.Context, msgID string, add, remove []string) error {
	if err := p.ensureService(ctx); err != nil {
		return fmt.Errorf("failed to ensure gmail service: %w", err)
	}

	req := &gmailapi.ModifyMessageRequest{
		AddLabelIds:    add,
		RemoveLabelIds: remove,
	}
	_, err := p.service.Users.Messages.Modify(userID, msgID, req).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to modify labels on message %s: %w", msgID, err)
	}
	return nil
}

// TrashMessage moves a message to trash.
func (p *Provider) TrashMessage(ctx context.Context, msgID string) error {
	if err := p.ensureService(ctx); err != nil {
		return fmt.Errorf("failed to ensure gmail service: %w", err)
	}

	_, err := p.service.Users.Messages.Trash(userID, msgID).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to trash gmail message %s: %w", msgID, err)
	}
	return nil
}

// MarkRead marks a message as read or unread by modifying the UNREAD label.
func (p *Provider) MarkRead(ctx context.Context, msgID string, read bool) error {
	if read {
		return p.ModifyLabels(ctx, msgID, nil, []string{"UNREAD"})
	}
	return p.ModifyLabels(ctx, msgID, []string{"UNREAD"}, nil)
}

// ListLabels returns all labels for the authenticated user.
func (p *Provider) ListLabels(ctx context.Context) ([]domain.Label, error) {
	if err := p.ensureService(ctx); err != nil {
		return nil, fmt.Errorf("failed to ensure gmail service: %w", err)
	}

	resp, err := p.service.Users.Labels.List(userID).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list gmail labels: %w", err)
	}

	labels := make([]domain.Label, 0, len(resp.Labels))
	for _, l := range resp.Labels {
		labelType := domain.LabelTypeUser
		if l.Type == "system" {
			labelType = domain.LabelTypeSystem
		}

		color := ""
		if l.Color != nil {
			color = l.Color.BackgroundColor
		}

		labels = append(labels, domain.Label{
			ID:   l.Id,
			Name: l.Name,
			Type: labelType,
			Color: color,
		})
	}

	return labels, nil
}

// Search searches for messages matching the query.
func (p *Provider) Search(ctx context.Context, query string, opts provider.ListOptions) ([]domain.Email, string, error) {
	opts.Query = query
	return p.ListMessages(ctx, opts)
}

// History returns history events since the given history ID.
func (p *Provider) History(ctx context.Context, startHistoryID uint64) ([]provider.HistoryEvent, uint64, error) {
	if err := p.ensureService(ctx); err != nil {
		return nil, 0, fmt.Errorf("failed to ensure gmail service: %w", err)
	}

	var events []provider.HistoryEvent
	var latestHistoryID uint64

	call := p.service.Users.History.List(userID).
		StartHistoryId(startHistoryID)

	err := call.Pages(ctx, func(resp *gmailapi.ListHistoryResponse) error {
		latestHistoryID = resp.HistoryId

		for _, h := range resp.History {
			for _, added := range h.MessagesAdded {
				events = append(events, provider.HistoryEvent{
					Type:      provider.HistoryMessageAdded,
					MessageID: added.Message.Id,
					LabelIDs:  added.Message.LabelIds,
				})
			}
			for _, deleted := range h.MessagesDeleted {
				events = append(events, provider.HistoryEvent{
					Type:      provider.HistoryMessageDeleted,
					MessageID: deleted.Message.Id,
				})
			}
			for _, labelsAdded := range h.LabelsAdded {
				events = append(events, provider.HistoryEvent{
					Type:      provider.HistoryLabelsAdded,
					MessageID: labelsAdded.Message.Id,
					LabelIDs:  labelsAdded.LabelIds,
				})
			}
			for _, labelsRemoved := range h.LabelsRemoved {
				events = append(events, provider.HistoryEvent{
					Type:      provider.HistoryLabelsRemoved,
					MessageID: labelsRemoved.Message.Id,
					LabelIDs:  labelsRemoved.LabelIds,
				})
			}
		}
		return nil
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list gmail history: %w", err)
	}

	return events, latestHistoryID, nil
}

// GetProfile returns the authenticated user's email address.
func (p *Provider) GetProfile(ctx context.Context) (string, error) {
	if err := p.ensureService(ctx); err != nil {
		return "", fmt.Errorf("failed to ensure gmail service: %w", err)
	}

	profile, err := p.service.Users.GetProfile(userID).Context(ctx).Do()
	if err != nil {
		return "", fmt.Errorf("failed to get gmail profile: %w", err)
	}
	return profile.EmailAddress, nil
}

// Compile-time interface compliance check.
var _ provider.EmailProvider = (*Provider)(nil)
