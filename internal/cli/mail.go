package cli

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/lu-zhengda/termail/internal/store"
	"github.com/lu-zhengda/termail/internal/store/sqlite"
)

func newListCmd() *cobra.Command {
	var accountFlag string
	var labelFlag string
	var limitFlag int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List email threads",
		Long:  "List email threads in a label (defaults to INBOX).",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := openDB()
			if err != nil {
				return err
			}
			defer db.Close()

			accountID, err := resolveAccountFlag(db, accountFlag)
			if err != nil {
				return err
			}

			threads, err := db.ListThreads(cmd.Context(), store.ListEmailOptions{
				AccountID: accountID,
				LabelID:   labelFlag,
				Limit:     limitFlag,
			})
			if err != nil {
				return fmt.Errorf("failed to list threads: %w", err)
			}

			if jsonFlag {
				return printJSON(toJSONThreads(threads))
			}

			if len(threads) == 0 {
				fmt.Println("No messages found.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "UNREAD\tFROM\tSUBJECT\tDATE\tMSGS\tTHREAD_ID")
			for _, t := range threads {
				unread := " "
				if t.HasUnread {
					unread = "*"
				}
				from := t.FromAddress.Name
				if from == "" {
					from = t.FromAddress.Email
				}
				if len(from) > 30 {
					from = from[:27] + "..."
				}
				subject := t.Subject
				if len(subject) > 50 {
					subject = subject[:47] + "..."
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%s\n",
					unread, from, subject,
					t.LastDate.Format("Jan 2, 2006"),
					t.MessageCount(), t.ID,
				)
			}
			return w.Flush()
		},
	}

	cmd.Flags().StringVar(&accountFlag, "account", "", "account ID (defaults to config default)")
	cmd.Flags().StringVar(&labelFlag, "label", "INBOX", "label to list (INBOX, SENT, STARRED, TRASH, SPAM, DRAFT, or custom)")
	cmd.Flags().IntVar(&limitFlag, "limit", 25, "max threads to show")
	return cmd
}

func newReadCmd() *cobra.Command {
	var accountFlag string

	cmd := &cobra.Command{
		Use:   "read <thread-id>",
		Short: "Read an email thread",
		Long:  "Display all messages in a thread by thread ID.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			threadID := args[0]

			db, err := openDB()
			if err != nil {
				return err
			}
			defer db.Close()

			accountID, err := resolveAccountFlag(db, accountFlag)
			if err != nil {
				return err
			}

			thread, err := db.GetThread(cmd.Context(), threadID, accountID)
			if err != nil {
				return fmt.Errorf("failed to get thread: %w", err)
			}

			if jsonFlag {
				return printJSON(toJSONThreadDetail(thread))
			}

			fmt.Printf("Subject: %s\n", thread.Subject)
			fmt.Printf("Thread ID: %s\n", thread.ID)
			fmt.Printf("Messages: %d\n", len(thread.Messages))
			fmt.Println(strings.Repeat("─", 60))

			for i, msg := range thread.Messages {
				if i > 0 {
					fmt.Println()
					fmt.Println(strings.Repeat("─", 60))
				}
				fmt.Printf("From: %s\n", msg.From)
				if len(msg.To) > 0 {
					to := make([]string, len(msg.To))
					for j, a := range msg.To {
						to[j] = a.String()
					}
					fmt.Printf("To: %s\n", strings.Join(to, ", "))
				}
				if len(msg.CC) > 0 {
					cc := make([]string, len(msg.CC))
					for j, a := range msg.CC {
						cc[j] = a.String()
					}
					fmt.Printf("CC: %s\n", strings.Join(cc, ", "))
				}
				fmt.Printf("Date: %s\n", msg.Date.Format("Mon, Jan 2 2006 3:04 PM"))
				readStatus := "read"
				if !msg.IsRead {
					readStatus = "unread"
				}
				fmt.Printf("Status: %s\n", readStatus)
				fmt.Printf("Message ID: %s\n", msg.ID)
				fmt.Println()
				fmt.Println(msg.Body)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&accountFlag, "account", "", "account ID (defaults to config default)")
	return cmd
}

func newSearchCmd() *cobra.Command {
	var accountFlag string
	var limitFlag int

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search emails",
		Long:  "Full-text search across email subject, body, and sender.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.Join(args, " ")

			db, err := openDB()
			if err != nil {
				return err
			}
			defer db.Close()

			accountID, err := resolveAccountFlag(db, accountFlag)
			if err != nil {
				return err
			}

			emails, err := db.SearchEmails(cmd.Context(), query, accountID)
			if err != nil {
				return fmt.Errorf("failed to search: %w", err)
			}

			shown := emails
			if limitFlag > 0 && len(shown) > limitFlag {
				shown = shown[:limitFlag]
			}

			if jsonFlag {
				return printJSON(toJSONEmails(shown))
			}

			if len(shown) == 0 {
				fmt.Println("No results found.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "FROM\tSUBJECT\tDATE\tID")
			for _, e := range shown {
				from := e.From.Name
				if from == "" {
					from = e.From.Email
				}
				if len(from) > 30 {
					from = from[:27] + "..."
				}
				subject := e.Subject
				if len(subject) > 50 {
					subject = subject[:47] + "..."
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
					from, subject,
					e.Date.Format("Jan 2, 2006"),
					e.ID,
				)
			}
			return w.Flush()
		},
	}

	cmd.Flags().StringVar(&accountFlag, "account", "", "account ID (defaults to config default)")
	cmd.Flags().IntVar(&limitFlag, "limit", 25, "max results to show")
	return cmd
}

func newLabelsCmd() *cobra.Command {
	var accountFlag string

	cmd := &cobra.Command{
		Use:   "labels",
		Short: "List labels for an account",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := openDB()
			if err != nil {
				return err
			}
			defer db.Close()

			accountID, err := resolveAccountFlag(db, accountFlag)
			if err != nil {
				return err
			}

			labels, err := db.ListLabels(cmd.Context(), accountID)
			if err != nil {
				return fmt.Errorf("failed to list labels: %w", err)
			}

			if jsonFlag {
				return printJSON(toJSONLabels(labels))
			}

			if len(labels) == 0 {
				fmt.Println("No labels found. Run 'termail sync' first.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tNAME\tTYPE")
			for _, l := range labels {
				fmt.Fprintf(w, "%s\t%s\t%s\n", l.ID, l.Name, l.Type)
			}
			return w.Flush()
		},
	}

	cmd.Flags().StringVar(&accountFlag, "account", "", "account ID (defaults to config default)")
	return cmd
}

// resolveAccountFlag resolves the account ID from flag, config default, or first account.
func resolveAccountFlag(db *sqlite.DB, accountFlag string) (string, error) {
	if accountFlag != "" {
		return accountFlag, nil
	}
	cfg, err := loadConfig()
	if err != nil {
		return "", err
	}
	return resolveAccountID(db, cfg)
}
