package cli

import (
	"time"

	"github.com/lu-zhengda/termail/internal/domain"
)

// ---------------------------------------------------------------------------
// Account JSON types (account list)
// ---------------------------------------------------------------------------

type jsonAccount struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Provider  string `json:"provider"`
	CreatedAt string `json:"created_at"`
}

func toJSONAccounts(accounts []domain.Account) []jsonAccount {
	out := make([]jsonAccount, 0, len(accounts))
	for _, a := range accounts {
		out = append(out, jsonAccount{
			ID:        a.ID,
			Email:     a.Email,
			Provider:  a.Provider,
			CreatedAt: a.CreatedAt.Format(time.DateOnly),
		})
	}
	return out
}

// ---------------------------------------------------------------------------
// Thread JSON types (list)
// ---------------------------------------------------------------------------

type jsonThread struct {
	ID           string      `json:"id"`
	Subject      string      `json:"subject"`
	From         jsonAddress `json:"from"`
	LastDate     string      `json:"last_date"`
	MessageCount int         `json:"message_count"`
	HasUnread    bool        `json:"has_unread"`
	Snippet      string      `json:"snippet,omitempty"`
	Labels       []string    `json:"labels,omitempty"`
}

func toJSONThreads(threads []domain.Thread) []jsonThread {
	out := make([]jsonThread, 0, len(threads))
	for _, t := range threads {
		out = append(out, jsonThread{
			ID:           t.ID,
			Subject:      t.Subject,
			From:         toJSONAddress(t.FromAddress),
			LastDate:     t.LastDate.Format(time.RFC3339),
			MessageCount: t.MessageCount(),
			HasUnread:    t.IsUnread(),
			Snippet:      t.Snippet,
			Labels:       t.Labels,
		})
	}
	return out
}

// ---------------------------------------------------------------------------
// Thread detail JSON type (read)
// ---------------------------------------------------------------------------

type jsonThreadDetail struct {
	ID       string        `json:"id"`
	Subject  string        `json:"subject"`
	Messages []jsonMessage `json:"messages"`
}

type jsonMessage struct {
	ID        string        `json:"id"`
	ThreadID  string        `json:"thread_id"`
	From      jsonAddress   `json:"from"`
	To        []jsonAddress `json:"to,omitempty"`
	CC        []jsonAddress `json:"cc,omitempty"`
	Subject   string        `json:"subject"`
	Body      string        `json:"body"`
	Date      string        `json:"date"`
	IsRead    bool          `json:"is_read"`
	IsStarred bool          `json:"is_starred"`
	Labels    []string      `json:"labels,omitempty"`
}

func toJSONThreadDetail(t *domain.Thread) jsonThreadDetail {
	msgs := make([]jsonMessage, 0, len(t.Messages))
	for _, m := range t.Messages {
		msgs = append(msgs, toJSONMessage(&m))
	}
	return jsonThreadDetail{
		ID:       t.ID,
		Subject:  t.Subject,
		Messages: msgs,
	}
}

func toJSONMessage(e *domain.Email) jsonMessage {
	return jsonMessage{
		ID:        e.ID,
		ThreadID:  e.ThreadID,
		From:      toJSONAddress(e.From),
		To:        toJSONAddresses(e.To),
		CC:        toJSONAddresses(e.CC),
		Subject:   e.Subject,
		Body:      e.Body,
		Date:      e.Date.Format(time.RFC3339),
		IsRead:    e.IsRead,
		IsStarred: e.IsStarred,
		Labels:    e.Labels,
	}
}

// ---------------------------------------------------------------------------
// Email JSON type (search results)
// ---------------------------------------------------------------------------

type jsonEmail struct {
	ID      string      `json:"id"`
	From    jsonAddress `json:"from"`
	Subject string      `json:"subject"`
	Date    string      `json:"date"`
}

func toJSONEmails(emails []domain.Email) []jsonEmail {
	out := make([]jsonEmail, 0, len(emails))
	for _, e := range emails {
		out = append(out, jsonEmail{
			ID:      e.ID,
			From:    toJSONAddress(e.From),
			Subject: e.Subject,
			Date:    e.Date.Format(time.RFC3339),
		})
	}
	return out
}

// ---------------------------------------------------------------------------
// Label JSON type (labels)
// ---------------------------------------------------------------------------

type jsonLabel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

func toJSONLabels(labels []domain.Label) []jsonLabel {
	out := make([]jsonLabel, 0, len(labels))
	for _, l := range labels {
		out = append(out, jsonLabel{
			ID:   l.ID,
			Name: l.Name,
			Type: string(l.Type),
		})
	}
	return out
}

// ---------------------------------------------------------------------------
// Address JSON type (shared)
// ---------------------------------------------------------------------------

type jsonAddress struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email"`
}

func toJSONAddress(a domain.Address) jsonAddress {
	return jsonAddress{Name: a.Name, Email: a.Email}
}

func toJSONAddresses(addrs []domain.Address) []jsonAddress {
	if len(addrs) == 0 {
		return nil
	}
	out := make([]jsonAddress, len(addrs))
	for i, a := range addrs {
		out[i] = toJSONAddress(a)
	}
	return out
}

// ---------------------------------------------------------------------------
// Action JSON type (compose, reply, forward, archive, trash, star, etc.)
// ---------------------------------------------------------------------------

type jsonAction struct {
	OK        bool   `json:"ok"`
	Action    string `json:"action"`
	MessageID string `json:"message_id,omitempty"`
	Email     string `json:"email,omitempty"`
	AccountID string `json:"account_id,omitempty"`
}
