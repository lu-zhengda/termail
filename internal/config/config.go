package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config holds all termail configuration.
type Config struct {
	Sync     SyncConfig     `toml:"sync"`
	UI       UIConfig       `toml:"ui"`
	Accounts AccountsConfig `toml:"accounts"`
	Gmail    GmailConfig    `toml:"gmail"`
}

// GmailConfig holds Gmail OAuth credentials.
// Users can override the embedded defaults via config file or env vars.
type GmailConfig struct {
	ClientID     string `toml:"client_id"`
	ClientSecret string `toml:"client_secret"`
}

// SyncConfig holds email synchronization settings.
type SyncConfig struct {
	Interval     string `toml:"interval"`
	InitialCount int    `toml:"initial_count"`
}

// UIConfig holds TUI display settings.
type UIConfig struct {
	DefaultView string `toml:"default_view"`
	Theme       string `toml:"theme"`
}

// AccountsConfig holds account selection settings.
type AccountsConfig struct {
	Default string `toml:"default"`
}

func defaults() Config {
	return Config{
		Sync: SyncConfig{
			Interval:     "5m",
			InitialCount: 500,
		},
		UI: UIConfig{
			DefaultView: "thread",
			Theme:       "default",
		},
	}
}

// Load reads config from path. If path is empty, returns defaults.
func Load(path string) (*Config, error) {
	cfg := defaults()
	if path == "" {
		return &cfg, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &cfg, nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	return &cfg, nil
}

// ConfigDir returns the termail config directory path.
func ConfigDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "termail")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "termail")
}

// DataDir returns the termail data directory path.
func DataDir() string {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "termail")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "termail")
}
