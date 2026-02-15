package gmail

import (
	"encoding/base64"
	"net/mail"
	"strings"
	"time"

	"github.com/zhengda-lu/termail/internal/domain"
	gmailapi "google.golang.org/api/gmail/v1"
)

// mapMessage converts a Gmail API Message to a domain Email.
func mapMessage(msg *gmailapi.Message) *domain.Email {
	var headers []*gmailapi.MessagePartHeader
	if msg.Payload != nil {
		headers = msg.Payload.Headers
	}

	text, html := extractBody(msg.Payload)
	attachments := extractAttachments(msg.Payload)

	return &domain.Email{
		ID:          msg.Id,
		ThreadID:    msg.ThreadId,
		From:        parseAddress(findHeader(headers, "From")),
		To:          parseAddressList(findHeader(headers, "To")),
		CC:          parseAddressList(findHeader(headers, "Cc")),
		Subject:     findHeader(headers, "Subject"),
		Body:        text,
		BodyHTML:    html,
		Date:        parseDate(findHeader(headers, "Date")),
		Labels:      msg.LabelIds,
		IsRead:      !containsLabel(msg.LabelIds, "UNREAD"),
		IsStarred:   containsLabel(msg.LabelIds, "STARRED"),
		Attachments: attachments,
		InReplyTo:   findHeader(headers, "In-Reply-To"),
	}
}

// findHeader performs a case-insensitive lookup for a header value.
func findHeader(headers []*gmailapi.MessagePartHeader, name string) string {
	lower := strings.ToLower(name)
	for _, h := range headers {
		if strings.ToLower(h.Name) == lower {
			return h.Value
		}
	}
	return ""
}

// parseAddress parses an RFC 5322 address string into a domain Address.
// Falls back to treating the entire string as a bare email if parsing fails.
func parseAddress(s string) domain.Address {
	s = strings.TrimSpace(s)
	if s == "" {
		return domain.Address{}
	}

	addr, err := mail.ParseAddress(s)
	if err != nil {
		// Fallback: treat as bare email
		return domain.Address{Email: s}
	}
	return domain.Address{
		Name:  addr.Name,
		Email: addr.Address,
	}
}

// parseAddressList parses a comma-separated list of RFC 5322 addresses.
func parseAddressList(s string) []domain.Address {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}

	parsed, err := mail.ParseAddressList(s)
	if err != nil {
		// Fallback: split by comma and parse individually
		parts := strings.Split(s, ",")
		var addrs []domain.Address
		for _, p := range parts {
			if a := parseAddress(p); a.Email != "" {
				addrs = append(addrs, a)
			}
		}
		return addrs
	}

	addrs := make([]domain.Address, 0, len(parsed))
	for _, a := range parsed {
		addrs = append(addrs, domain.Address{
			Name:  a.Name,
			Email: a.Address,
		})
	}
	return addrs
}

// parseDate tries multiple date formats commonly used in email headers.
func parseDate(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}

	formats := []string{
		time.RFC1123Z,                      // "Mon, 02 Jan 2006 15:04:05 -0700"
		time.RFC1123,                       // "Mon, 02 Jan 2006 15:04:05 MST"
		time.RFC822Z,                       // "02 Jan 06 15:04 -0700"
		time.RFC822,                        // "02 Jan 06 15:04 MST"
		"Mon, 2 Jan 2006 15:04:05 -0700",  // single-digit day
		"Mon, 2 Jan 2006 15:04:05 MST",    // single-digit day with named zone
		"2 Jan 2006 15:04:05 -0700",        // no weekday
		"2006-01-02T15:04:05Z07:00",        // ISO 8601
		"Mon, 02 Jan 2006 15:04:05 -0700 (MST)", // with parenthesized zone
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

// containsLabel checks if a label is present in the list.
func containsLabel(labels []string, label string) bool {
	for _, l := range labels {
		if l == label {
			return true
		}
	}
	return false
}

// extractBody recursively extracts text/plain and text/html content from a message payload.
func extractBody(payload *gmailapi.MessagePart) (text, html string) {
	if payload == nil {
		return "", ""
	}

	// If this part has sub-parts, recurse into them
	if len(payload.Parts) > 0 {
		for _, part := range payload.Parts {
			t, h := extractBody(part)
			if text == "" && t != "" {
				text = t
			}
			if html == "" && h != "" {
				html = h
			}
		}
		return text, html
	}

	// Leaf part: decode the body
	data := ""
	if payload.Body != nil {
		data = decodeBase64URL(payload.Body.Data)
	}

	switch payload.MimeType {
	case "text/plain":
		return data, ""
	case "text/html":
		return "", data
	}
	return "", ""
}

// extractAttachments collects attachment metadata from message parts.
func extractAttachments(payload *gmailapi.MessagePart) []domain.Attachment {
	if payload == nil {
		return nil
	}
	var attachments []domain.Attachment
	collectAttachments(payload, &attachments)
	return attachments
}

func collectAttachments(part *gmailapi.MessagePart, attachments *[]domain.Attachment) {
	if part.Filename != "" && part.Body != nil {
		*attachments = append(*attachments, domain.Attachment{
			ID:       part.Body.AttachmentId,
			Filename: part.Filename,
			MIMEType: part.MimeType,
			Size:     part.Body.Size,
		})
	}
	for _, p := range part.Parts {
		collectAttachments(p, attachments)
	}
}

// decodeBase64URL decodes Gmail's URL-safe base64 encoded strings (without padding).
func decodeBase64URL(s string) string {
	if s == "" {
		return ""
	}
	data, err := base64.URLEncoding.WithPadding(base64.NoPadding).DecodeString(s)
	if err != nil {
		return ""
	}
	return string(data)
}
