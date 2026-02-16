package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/lu-zhengda/termail/internal/config"
	"github.com/lu-zhengda/termail/internal/provider"
	"github.com/lu-zhengda/termail/internal/provider/gmail"
	"github.com/lu-zhengda/termail/internal/store"
	"github.com/lu-zhengda/termail/internal/store/sqlite"
	"github.com/lu-zhengda/termail/internal/tui"
)

var (
	// version is set via ldflags at build time.
	version = "dev"
	cfgFile string

	// jsonFlag enables JSON output for all commands.
	jsonFlag bool
)

func NewRootCmd() *cobra.Command {
	var accountFlag string

	root := &cobra.Command{
		Use:     "termail",
		Short:   "Terminal email client",
		Long:    "A terminal-based email client with Gmail support.",
		Version: version,
		RunE: func(cmd *cobra.Command, args []string) error {
			if shell, _ := cmd.Flags().GetString("generate-completion"); shell != "" {
				switch shell {
				case "bash":
					return cmd.Root().GenBashCompletion(os.Stdout)
				case "zsh":
					return cmd.Root().GenZshCompletion(os.Stdout)
				case "fish":
					return cmd.Root().GenFishCompletion(os.Stdout, true)
				default:
					return fmt.Errorf("unsupported shell: %s (use bash, zsh, or fish)", shell)
				}
			}

			db, err := openDB()
			if err != nil {
				return err
			}
			defer db.Close()

			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			if err := resolveGmailCredentials(cfg); err != nil {
				return err
			}

			// Determine the initial account.
			accountID := accountFlag
			if accountID == "" {
				accountID, err = resolveAccountID(db, cfg)
				if err != nil {
					return err
				}
			}

			// Load all accounts for account switching.
			accounts, err := db.ListAccounts(cmd.Context())
			if err != nil {
				return fmt.Errorf("failed to list accounts: %w", err)
			}

			tokenStore := store.NewKeyringTokenStore()
			p := gmail.New(accountID, tokenStore)

			factory := tui.ProviderFactory(func(accID string) provider.EmailProvider {
				return gmail.New(accID, tokenStore)
			})

			return tui.Run(db, p, accountID, accounts, factory)
		},
	}
	root.SetVersionTemplate(fmt.Sprintf("termail %s\n", version))
	root.CompletionOptions.DisableDefaultCmd = true
	root.Flags().String("generate-completion", "", "Generate shell completion (bash, zsh, fish)")
	root.Flags().MarkHidden("generate-completion")
	root.PersistentFlags().StringVar(&cfgFile, "config", "", "config file path")
	root.PersistentFlags().BoolVar(&jsonFlag, "json", false, "output in JSON format")
	root.Flags().StringVar(&accountFlag, "account", "", "account ID to use (defaults to config default or first account)")
	root.AddCommand(newAccountCmd())
	root.AddCommand(newSyncCmd())
	root.AddCommand(newListCmd())
	root.AddCommand(newReadCmd())
	root.AddCommand(newSearchCmd())
	root.AddCommand(newLabelsCmd())
	root.AddCommand(newComposeCmd())
	root.AddCommand(newReplyCmd())
	root.AddCommand(newForwardCmd())
	root.AddCommand(newArchiveCmd())
	root.AddCommand(newTrashCmd())
	root.AddCommand(newStarCmd())
	root.AddCommand(newMarkReadCmd())
	root.AddCommand(newLabelModifyCmd())
	return root
}

func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

// openDB creates the data directory and opens the SQLite database.
func openDB() (*sqlite.DB, error) {
	dataDir := config.DataDir()
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	dbPath := filepath.Join(dataDir, "termail.db")
	db, err := sqlite.New(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	return db, nil
}

// loadConfig loads the application configuration from the config file.
func loadConfig() (*config.Config, error) {
	path := cfgFile
	if path == "" {
		path = filepath.Join(config.ConfigDir(), "config.toml")
	}
	cfg, err := config.Load(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	return cfg, nil
}

// resolveAccountID determines which account to use based on config default
// or falls back to the first account in the database.
func resolveAccountID(db *sqlite.DB, cfg *config.Config) (string, error) {
	if cfg.Accounts.Default != "" {
		return cfg.Accounts.Default, nil
	}

	accounts, err := db.ListAccounts(context.Background())
	if err != nil {
		return "", fmt.Errorf("failed to list accounts: %w", err)
	}
	if len(accounts) == 0 {
		return "", fmt.Errorf("no accounts configured; run 'termail account add' first")
	}
	return accounts[0].ID, nil
}

// resolveGmailCredentials sets Gmail OAuth credentials using the first
// available source: config file → environment variables.
func resolveGmailCredentials(cfg *config.Config) error {
	// 1. Config file
	if cfg.Gmail.ClientID != "" && cfg.Gmail.ClientSecret != "" {
		gmail.SetCredentials(cfg.Gmail.ClientID, cfg.Gmail.ClientSecret)
		return nil
	}

	// 2. Environment variables
	clientID := os.Getenv("GMAIL_CLIENT_ID")
	clientSecret := os.Getenv("GMAIL_CLIENT_SECRET")
	if clientID != "" && clientSecret != "" {
		gmail.SetCredentials(clientID, clientSecret)
		return nil
	}

	return gmail.EnsureCredentials()
}
