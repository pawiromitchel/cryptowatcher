package config

import (
	"os"
	"path/filepath"
	"testing"

	"cryptowatcher/internal/model"
)

func TestDefaultConfig(t *testing.T) {
	cfg := model.DefaultConfig()
	if len(cfg.CryptoPairs) == 0 {
		t.Fatalf("expected default crypto pairs, got empty slice")
	}
	if len(cfg.StockPairs) == 0 {
		t.Fatalf("expected default stock pairs, got empty slice")
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
	if len(cfg.CryptoPairs) != 3 {
		t.Errorf("expected 3 default crypto pairs, got %d", len(cfg.CryptoPairs))
	}
	if len(cfg.StockPairs) != 4 {
		t.Errorf("expected 4 default stock pairs, got %d", len(cfg.StockPairs))
	}

	cfg.CryptoPairs = append(cfg.CryptoPairs, "DOGE-USD")
	cfg.StockPairs = append(cfg.StockPairs, "NVDA")
	if err := Save(cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("failed to reload config: %v", err)
	}
	if len(loaded.CryptoPairs) != 4 {
		t.Errorf("expected 4 crypto pairs after save, got %d", len(loaded.CryptoPairs))
	}
	if len(loaded.StockPairs) != 5 {
		t.Errorf("expected 5 stock pairs after save, got %d", len(loaded.StockPairs))
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
