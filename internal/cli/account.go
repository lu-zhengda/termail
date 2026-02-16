package cli

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/lu-zhengda/termail/internal/app"
	"github.com/lu-zhengda/termail/internal/domain"
	"github.com/lu-zhengda/termail/internal/provider/gmail"
	"github.com/lu-zhengda/termail/internal/store"
)

func newAccountCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "account",
		Short: "Manage email accounts",
	}
	cmd.AddCommand(newAccountAddCmd())
	cmd.AddCommand(newAccountListCmd())
	cmd.AddCommand(newAccountRemoveCmd())
	return cmd
}

func newAccountAddCmd() *cobra.Command {
	var email string

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a Gmail account via OAuth",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			if err := resolveGmailCredentials(cfg); err != nil {
				return err
			}

			db, err := openDB()
			if err != nil {
				return err
			}
			defer db.Close()

			tokenStore := store.NewKeyringTokenStore()

			// Use email as account ID if provided, otherwise use a temporary ID
			// that will be replaced after OAuth when we learn the real email.
			accountID := email
			if accountID == "" {
				accountID = fmt.Sprintf("gmail-%d", time.Now().UnixNano())
			}

			provider := gmail.New(accountID, tokenStore)

			ctx := cmd.Context()
			fmt.Println("Starting Gmail OAuth flow...")
			if err := provider.Authenticate(ctx); err != nil {
				return fmt.Errorf("failed to authenticate: %w", err)
			}

			// If no email was provided, fetch it from the Gmail profile.
			if email == "" {
				profileEmail, err := provider.GetProfile(ctx)
				if err != nil {
					return fmt.Errorf("failed to get profile email: %w", err)
				}
				email = profileEmail

				// Re-save the token under the real email as account ID,
				// and clean up the temporary one.
				token, err := tokenStore.LoadToken(accountID)
				if err != nil {
					return fmt.Errorf("failed to reload token: %w", err)
				}
				if err := tokenStore.SaveToken(email, token); err != nil {
					return fmt.Errorf("failed to re-save token: %w", err)
				}
				if delErr := tokenStore.DeleteToken(accountID); delErr != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to delete temporary token: %v\n", delErr)
			}
				accountID = email
			}

			account := &domain.Account{
				ID:          accountID,
				Email:       email,
				Provider:    "gmail",
				DisplayName: email,
				CreatedAt:   time.Now(),
			}

			if err := db.CreateAccount(ctx, account); err != nil {
				return fmt.Errorf("failed to store account: %w", err)
			}

			if jsonFlag {
				return printJSON(jsonAction{OK: true, Action: "add", Email: email})
			}

			fmt.Printf("Account added: %s\n", email)
			return nil
		},
	}

	cmd.Flags().StringVar(&email, "email", "", "email address (auto-detected if omitted)")
	return cmd
}

func newAccountListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List configured accounts",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := openDB()
			if err != nil {
				return err
			}
			defer db.Close()

			accounts, err := db.ListAccounts(cmd.Context())
			if err != nil {
				return fmt.Errorf("failed to list accounts: %w", err)
			}

			if jsonFlag {
				return printJSON(toJSONAccounts(accounts))
			}

			if len(accounts) == 0 {
				fmt.Println("No accounts configured. Run 'termail account add' to add one.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tEMAIL\tPROVIDER\tCREATED")
			for _, a := range accounts {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
					a.ID,
					a.Email,
					a.Provider,
					a.CreatedAt.Format(time.DateOnly),
				)
			}
			return w.Flush()
		},
	}
}

func newAccountRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove [email]",
		Short: "Remove an account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			email := args[0]

			db, err := openDB()
			if err != nil {
				return err
			}
			defer db.Close()

			ctx := cmd.Context()
			accounts, err := db.ListAccounts(ctx)
			if err != nil {
				return fmt.Errorf("failed to list accounts: %w", err)
			}

			var target *domain.Account
			for i := range accounts {
				if accounts[i].Email == email || accounts[i].ID == email {
					target = &accounts[i]
					break
				}
			}
			if target == nil {
				return fmt.Errorf("account not found: %s", email)
			}

			if err := db.DeleteAccount(ctx, target.ID); err != nil {
				return fmt.Errorf("failed to delete account: %w", err)
			}

			tokenStore := store.NewKeyringTokenStore()
			if err := tokenStore.DeleteToken(target.ID); err != nil {
				// Non-fatal: token may already be gone.
				fmt.Fprintf(os.Stderr, "Warning: could not remove token from keyring: %v\n", err)
			}

			if jsonFlag {
				return printJSON(jsonAction{OK: true, Action: "remove", Email: target.Email})
			}

			fmt.Printf("Account removed: %s\n", target.Email)
			return nil
		},
	}
}

func newSyncCmd() *cobra.Command {
	var accountFlag string

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Manually sync emails",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := openDB()
			if err != nil {
				return err
			}
			defer db.Close()

			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			accountID := accountFlag
			if accountID == "" {
				accountID, err = resolveAccountID(db, cfg)
				if err != nil {
					return err
				}
			}

			if err := resolveGmailCredentials(cfg); err != nil {
				return err
			}

			tokenStore := store.NewKeyringTokenStore()
			provider := gmail.New(accountID, tokenStore)

			ctx := cmd.Context()
			svc := app.NewSyncService(db, provider, accountID)

			if !jsonFlag {
				fmt.Printf("Syncing account %s...\n", accountID)
			}
			if err := svc.IncrementalSync(ctx); err != nil {
				return fmt.Errorf("failed to sync: %w", err)
			}

			if jsonFlag {
				return printJSON(jsonAction{OK: true, Action: "sync", AccountID: accountID})
			}

			fmt.Println("Sync complete.")
			return nil
		},
	}

	cmd.Flags().StringVar(&accountFlag, "account", "", "account ID to sync (defaults to config default or first account)")
	return cmd
}
