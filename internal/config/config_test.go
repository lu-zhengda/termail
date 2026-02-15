package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Sync.Interval != "5m" {
		t.Errorf("default interval = %q, want %q", cfg.Sync.Interval, "5m")
	}
	if cfg.Sync.InitialCount != 500 {
		t.Errorf("default initial_count = %d, want 500", cfg.Sync.InitialCount)
	}
	if cfg.UI.DefaultView != "thread" {
		t.Errorf("default view = %q, want %q", cfg.UI.DefaultView, "thread")
	}
}

func TestLoad_FromFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")
	content := `
[sync]
interval = "10m"
initial_count = 100

[ui]
default_view = "flat"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Sync.Interval != "10m" {
		t.Errorf("interval = %q, want %q", cfg.Sync.Interval, "10m")
	}
	if cfg.UI.DefaultView != "flat" {
		t.Errorf("view = %q, want %q", cfg.UI.DefaultView, "flat")
	}
}

func TestLoad_NonExistentFile(t *testing.T) {
	cfg, err := Load("/nonexistent/path/config.toml")
	if err != nil {
		t.Fatalf("Load() should return defaults for missing file, got error: %v", err)
	}
	if cfg.Sync.Interval != "5m" {
		t.Errorf("interval = %q, want default %q", cfg.Sync.Interval, "5m")
	}
}

func TestLoad_InvalidTOML(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(cfgPath, []byte("not valid [[ toml"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("Load() should return error for invalid TOML")
	}
	if !strings.Contains(err.Error(), "failed to parse config") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "failed to parse config")
	}
}

func TestConfigDir(t *testing.T) {
	t.Run("with XDG_CONFIG_HOME", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "/custom/config")
		dir := ConfigDir()
		want := "/custom/config/termail"
		if dir != want {
			t.Errorf("ConfigDir() = %q, want %q", dir, want)
		}
	})
	t.Run("without XDG_CONFIG_HOME", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "")
		dir := ConfigDir()
		if !strings.HasSuffix(dir, filepath.Join(".config", "termail")) {
			t.Errorf("ConfigDir() = %q, want suffix %q", dir, filepath.Join(".config", "termail"))
		}
	})
}

func TestDataDir(t *testing.T) {
	t.Run("with XDG_DATA_HOME", func(t *testing.T) {
		t.Setenv("XDG_DATA_HOME", "/custom/data")
		dir := DataDir()
		want := "/custom/data/termail"
		if dir != want {
			t.Errorf("DataDir() = %q, want %q", dir, want)
		}
	})
	t.Run("without XDG_DATA_HOME", func(t *testing.T) {
		t.Setenv("XDG_DATA_HOME", "")
		dir := DataDir()
		if !strings.HasSuffix(dir, filepath.Join(".local", "share", "termail")) {
			t.Errorf("DataDir() = %q, want suffix %q", dir, filepath.Join(".local", "share", "termail"))
		}
	})
}
