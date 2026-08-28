package config

import (
	"os"
	"path/filepath"
	"testing"

	"cryptowatcher/internal/model"
)

func TestDefaultConfig(t *testing.T) {
	cfg := model.DefaultConfig()
	if len(cfg.Pairs) == 0 {
		t.Fatalf("expected default pairs, got empty slice")
	}
	if cfg.RefreshInterval <= 0 {
		t.Errorf("expected positive refresh interval, got %d", cfg.RefreshInterval)
	}
}

func TestLoadSaveConfig(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cryptowatcher_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Setenv("XDG_CONFIG_HOME", tempDir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("failed to load default config: %v", err)
	}
	if len(cfg.Pairs) != 3 {
		t.Errorf("expected 3 default pairs, got %d", len(cfg.Pairs))
	}

	cfg.Pairs = append(cfg.Pairs, "DOGE-USD")
	if err := Save(cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("failed to reload config: %v", err)
	}
	if len(loaded.Pairs) != 4 {
		t.Errorf("expected 4 pairs after save, got %d", len(loaded.Pairs))
	}
	if loaded.Pairs[3] != "DOGE-USD" {
		t.Errorf("expected last pair to be DOGE-USD, got %s", loaded.Pairs[3])
	}
}

func TestConfigPath(t *testing.T) {
	path, err := GetConfigPath()
	if err != nil {
		t.Fatalf("unexpected error getting config path: %v", err)
	}
	if filepath.Base(path) != configFileName {
		t.Errorf("expected filename %s, got %s", configFileName, filepath.Base(path))
	}
}
