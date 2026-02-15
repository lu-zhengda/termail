package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/lu-zhengda/termail/internal/domain"
	"github.com/lu-zhengda/termail/internal/provider/gmail"
	"github.com/lu-zhengda/termail/internal/store"
)

func newComposeCmd() *cobra.Command {
	var accountFlag, toFlag, ccFlag, subjectFlag, bodyFlag string

	cmd := &cobra.Command{
		Use:   "compose",
		Short: "Compose and send a new email",
		RunE: func(cmd *cobra.Command, args []string) error {
			if toFlag == "" {
				return fmt.Errorf("--to is required")
			}
			if subjectFlag == "" {
				return fmt.Errorf("--subject is required")
			}

			body := bodyFlag
			if body == "-" {
				b, err := io.ReadAll(os.Stdin)
				if err != nil {
					return fmt.Errorf("failed to read body from stdin: %w", err)
				}
				body = string(b)
			}

			provider, _, err := setupProvider(cmd, accountFlag)
			if err != nil {
				return err
			}

			email := &domain.Email{
				To:      parseAddrList(toFlag),
				CC:      parseAddrList(ccFlag),
				Subject: subjectFlag,
				Body:    body,
				Date:    time.Now(),
			}

			if err := provider.SendMessage(cmd.Context(), email); err != nil {
				return fmt.Errorf("failed to send email: %w", err)
			}

			fmt.Println("Email sent.")
			return nil
		},
	}

	cmd.Flags().StringVar(&accountFlag, "account", "", "account ID to send from")
	cmd.Flags().StringVar(&toFlag, "to", "", "recipient email addresses (comma-separated)")
	cmd.Flags().StringVar(&ccFlag, "cc", "", "CC email addresses (comma-separated)")
	cmd.Flags().StringVar(&subjectFlag, "subject", "", "email subject")
	cmd.Flags().StringVar(&bodyFlag, "body", "", "email body (use '-' to read from stdin)")
	return cmd
}

func newReplyCmd() *cobra.Command {
	var accountFlag, bodyFlag string
	var allFlag bool

	cmd := &cobra.Command{
		Use:   "reply <message-id>",
		Short: "Reply to an email",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			messageID := args[0]

			body := bodyFlag
			if body == "-" {
				b, err := io.ReadAll(os.Stdin)
				if err != nil {
					return fmt.Errorf("failed to read body from stdin: %w", err)
				}
				body = string(b)
			}

			provider, accountID, err := setupProvider(cmd, accountFlag)
			if err != nil {
				return err
			}

			db, err := openDB()
			if err != nil {
				return err
			}
			defer db.Close()

			// Fetch the original email to build the reply.
			original, err := db.GetEmail(cmd.Context(), messageID)
			if err != nil {
				return fmt.Errorf("failed to get email %s: %w", messageID, err)
			}
			_ = accountID

			reply := &domain.Email{
				To:        []domain.Address{original.From},
				Subject:   prefixSubject("Re: ", original.Subject),
				Body:      body + "\n\n" + formatQuote(original),
				Date:      time.Now(),
				InReplyTo: original.ID,
				ThreadID:  original.ThreadID,
			}

			if allFlag {
				for _, addr := range original.To {
					reply.CC = append(reply.CC, addr)
				}
				for _, addr := range original.CC {
					reply.CC = append(reply.CC, addr)
				}
			}

			if err := provider.SendMessage(cmd.Context(), reply); err != nil {
				return fmt.Errorf("failed to send reply: %w", err)
			}

			fmt.Println("Reply sent.")
			return nil
		},
	}

	cmd.Flags().StringVar(&accountFlag, "account", "", "account ID")
	cmd.Flags().StringVar(&bodyFlag, "body", "", "reply body (use '-' to read from stdin)")
	cmd.Flags().BoolVar(&allFlag, "all", false, "reply to all recipients")
	return cmd
}

func newForwardCmd() *cobra.Command {
	var accountFlag, toFlag, bodyFlag string

	cmd := &cobra.Command{
		Use:   "forward <message-id>",
		Short: "Forward an email",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			messageID := args[0]

			if toFlag == "" {
				return fmt.Errorf("--to is required")
			}

			body := bodyFlag
			if body == "-" {
				b, err := io.ReadAll(os.Stdin)
				if err != nil {
					return fmt.Errorf("failed to read body from stdin: %w", err)
				}
				body = string(b)
			}

			provider, _, err := setupProvider(cmd, accountFlag)
			if err != nil {
				return err
			}

			db, err := openDB()
			if err != nil {
				return err
			}
			defer db.Close()

			original, err := db.GetEmail(cmd.Context(), messageID)
			if err != nil {
				return fmt.Errorf("failed to get email %s: %w", messageID, err)
			}

			fwd := &domain.Email{
				To:      parseAddrList(toFlag),
				Subject: prefixSubject("Fwd: ", original.Subject),
				Body:    body + "\n\n---------- Forwarded message ----------\n" + formatForward(original),
				Date:    time.Now(),
			}

			if err := provider.SendMessage(cmd.Context(), fwd); err != nil {
				return fmt.Errorf("failed to forward: %w", err)
			}

			fmt.Println("Email forwarded.")
			return nil
		},
	}

	cmd.Flags().StringVar(&accountFlag, "account", "", "account ID")
	cmd.Flags().StringVar(&toFlag, "to", "", "recipient email addresses (comma-separated)")
	cmd.Flags().StringVar(&bodyFlag, "body", "", "optional message to prepend (use '-' for stdin)")
	return cmd
}

func newArchiveCmd() *cobra.Command {
	var accountFlag string

	cmd := &cobra.Command{
		Use:   "archive <message-id>",
		Short: "Archive an email (remove from Inbox)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			provider, _, err := setupProvider(cmd, accountFlag)
			if err != nil {
				return err
			}

			if err := provider.ModifyLabels(cmd.Context(), args[0], nil, []string{domain.LabelInbox}); err != nil {
				return fmt.Errorf("failed to archive: %w", err)
			}

			fmt.Println("Email archived.")
			return nil
		},
	}

	cmd.Flags().StringVar(&accountFlag, "account", "", "account ID")
	return cmd
}

func newTrashCmd() *cobra.Command {
	var accountFlag string

	cmd := &cobra.Command{
		Use:   "trash <message-id>",
		Short: "Move an email to trash",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			provider, _, err := setupProvider(cmd, accountFlag)
			if err != nil {
				return err
			}

			if err := provider.TrashMessage(cmd.Context(), args[0]); err != nil {
				return fmt.Errorf("failed to trash: %w", err)
			}

			fmt.Println("Email moved to trash.")
			return nil
		},
	}

	cmd.Flags().StringVar(&accountFlag, "account", "", "account ID")
	return cmd
}

func newStarCmd() *cobra.Command {
	var accountFlag string
	var removeFlag bool

	cmd := &cobra.Command{
		Use:   "star <message-id>",
		Short: "Star or unstar an email",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			provider, _, err := setupProvider(cmd, accountFlag)
			if err != nil {
				return err
			}

			var add, remove []string
			if removeFlag {
				remove = []string{domain.LabelStarred}
			} else {
				add = []string{domain.LabelStarred}
			}

			if err := provider.ModifyLabels(cmd.Context(), args[0], add, remove); err != nil {
				return fmt.Errorf("failed to update star: %w", err)
			}

			if removeFlag {
				fmt.Println("Star removed.")
			} else {
				fmt.Println("Email starred.")
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&accountFlag, "account", "", "account ID")
	cmd.Flags().BoolVar(&removeFlag, "remove", false, "remove star instead of adding")
	return cmd
}

func newMarkReadCmd() *cobra.Command {
	var accountFlag string
	var unreadFlag bool

	cmd := &cobra.Command{
		Use:   "mark-read <message-id>",
		Short: "Mark an email as read or unread",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			provider, _, err := setupProvider(cmd, accountFlag)
			if err != nil {
				return err
			}

			db, err := openDB()
			if err != nil {
				return err
			}
			defer db.Close()

			read := !unreadFlag

			// Update local DB first.
			if err := db.SetEmailRead(cmd.Context(), args[0], read); err != nil {
				return fmt.Errorf("failed to update local state: %w", err)
			}

			// Sync to Gmail.
			if err := provider.MarkRead(cmd.Context(), args[0], read); err != nil {
				return fmt.Errorf("failed to sync read status: %w", err)
			}

			if read {
				fmt.Println("Marked as read.")
			} else {
				fmt.Println("Marked as unread.")
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&accountFlag, "account", "", "account ID")
	cmd.Flags().BoolVar(&unreadFlag, "unread", false, "mark as unread instead of read")
	return cmd
}

func newLabelModifyCmd() *cobra.Command {
	var accountFlag string

	cmd := &cobra.Command{
		Use:   "label-modify <message-id>",
		Short: "Add or remove labels from an email",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			addLabels, _ := cmd.Flags().GetString("add")
			removeLabels, _ := cmd.Flags().GetString("remove")

			if addLabels == "" && removeLabels == "" {
				return fmt.Errorf("at least one of --add or --remove is required")
			}

			provider, _, err := setupProvider(cmd, accountFlag)
			if err != nil {
				return err
			}

			var add, remove []string
			if addLabels != "" {
				add = splitTrim(addLabels)
			}
			if removeLabels != "" {
				remove = splitTrim(removeLabels)
			}

			if err := provider.ModifyLabels(cmd.Context(), args[0], add, remove); err != nil {
				return fmt.Errorf("failed to modify labels: %w", err)
			}

			fmt.Println("Labels updated.")
			return nil
		},
	}

	cmd.Flags().StringVar(&accountFlag, "account", "", "account ID")
	cmd.Flags().String("add", "", "label IDs to add (comma-separated)")
	cmd.Flags().String("remove", "", "label IDs to remove (comma-separated)")
	return cmd
}

// setupProvider creates an authenticated Gmail provider for the resolved account.
func setupProvider(cmd *cobra.Command, accountFlag string) (*gmail.Provider, string, error) {
	db, err := openDB()
	if err != nil {
		return nil, "", err
	}
	defer db.Close()

	accountID, err := resolveAccountFlag(db, accountFlag)
	if err != nil {
		return nil, "", err
	}

	cfg, err := loadConfig()
	if err != nil {
		return nil, "", err
	}
	if err := resolveGmailCredentials(cfg); err != nil {
		return nil, "", err
	}

	tokenStore := store.NewKeyringTokenStore()
	p := gmail.New(accountID, tokenStore)

	return p, accountID, nil
}

// parseAddrList splits a comma-separated string of email addresses.
func parseAddrList(s string) []domain.Address {
	if s == "" {
		return nil
	}
	parts := splitTrim(s)
	addrs := make([]domain.Address, len(parts))
	for i, p := range parts {
		addrs[i] = domain.Address{Email: p}
	}
	return addrs
}

// splitTrim splits by comma and trims whitespace.
func splitTrim(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// prefixSubject adds a prefix if not already present.
func prefixSubject(prefix, subject string) string {
	if strings.HasPrefix(strings.ToLower(subject), strings.ToLower(prefix)) {
		return subject
	}
	return prefix + subject
}

// formatQuote formats an email for quoting in a reply.
func formatQuote(e *domain.Email) string {
	var b strings.Builder
	fmt.Fprintf(&b, "On %s, %s wrote:\n", e.Date.Format("Mon, Jan 2, 2006 at 3:04 PM"), e.From)
	for _, line := range strings.Split(e.Body, "\n") {
		fmt.Fprintf(&b, "> %s\n", line)
	}
	return b.String()
}

// formatForward formats an email for forwarding.
func formatForward(e *domain.Email) string {
	var b strings.Builder
	fmt.Fprintf(&b, "From: %s\n", e.From)
	fmt.Fprintf(&b, "Date: %s\n", e.Date.Format("Mon, Jan 2, 2006 at 3:04 PM"))
	fmt.Fprintf(&b, "Subject: %s\n", e.Subject)
	if len(e.To) > 0 {
		to := make([]string, len(e.To))
		for i, a := range e.To {
			to[i] = a.String()
		}
		fmt.Fprintf(&b, "To: %s\n", strings.Join(to, ", "))
	}
	b.WriteString("\n")
	b.WriteString(e.Body)
	return b.String()
}
