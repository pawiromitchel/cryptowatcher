// Package config loads and saves the persistent watchlist configuration.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cryptowatcher/internal/model"
)

const (
	configDirName  = "cryptowatcher"
	configFileName = "config.json"
)

// GetConfigPath returns the absolute path to the configuration file.
func GetConfigPath() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, configDirName, configFileName), nil
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		home, homeErr := os.UserHomeDir()
		if homeErr != nil {
			return "", homeErr
		}
		configDir = filepath.Join(home, ".config")
	}
	return filepath.Join(configDir, configDirName, configFileName), nil
}

// Load reads the configuration from disk.
//
// A missing file yields (and persists) the defaults. A file that cannot be
// parsed is moved aside to config.json.bak so it is never silently overwritten;
// defaults are returned together with a descriptive error. Load never returns a
// nil config.
func Load() (*model.Config, error) {
	path, err := GetConfigPath()
	if err != nil {
		return model.DefaultConfig(), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			cfg := model.DefaultConfig()
			_ = Save(cfg)
			return cfg, nil
		}
		return model.DefaultConfig(), fmt.Errorf("reading %s: %w", path, err)
	}

	var cfg model.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		backup := path + ".bak"
		if renameErr := os.Rename(path, backup); renameErr != nil {
			return model.DefaultConfig(), fmt.Errorf("%s is invalid (%v) and could not be backed up: %w", path, err, renameErr)
		}
		return model.DefaultConfig(), fmt.Errorf("%s is invalid (%v); saved a copy to %s and using defaults", path, err, backup)
	}

	// Legacy migration: a single "pairs" list. Only applies when the new keys
	// are absent, so a deliberately emptied list stays empty.
	if cfg.CryptoPairs == nil && cfg.StockPairs == nil {
		if len(cfg.Pairs) > 0 {
			cfg.CryptoPairs = cfg.Pairs
		} else {
			def := model.DefaultConfig()
			cfg.CryptoPairs, cfg.StockPairs = def.CryptoPairs, def.StockPairs
		}
	}
	cfg.Pairs = nil
	cfg.CryptoPairs = dedupe(cfg.CryptoPairs)
	cfg.StockPairs = dedupe(cfg.StockPairs)

	switch {
	case cfg.RefreshInterval <= 0:
		cfg.RefreshInterval = model.DefaultRefreshInterval
	case cfg.RefreshInterval < model.MinRefreshInterval:
		cfg.RefreshInterval = model.MinRefreshInterval
	}

	return &cfg, nil
}

// Save atomically writes the configuration to disk.
func Save(cfg *model.Config) error {
	path, err := GetConfigPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	out := *cfg
	out.Pairs = nil
	if out.CryptoPairs == nil {
		out.CryptoPairs = []string{}
	}
	if out.StockPairs == nil {
		out.StockPairs = []string{}
	}

	data, err := json.MarshalIndent(&out, "", "  ")
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, configFileName+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpName) }

	if _, err := tmp.Write(append(data, '\n')); err != nil {
		_ = tmp.Close()
		cleanup()
		return err
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return err
	}
	if err := os.Chmod(tmpName, 0o644); err != nil {
		cleanup()
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		cleanup()
		return err
	}
	return nil
}

func dedupe(in []string) []string {
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		key := strings.ToUpper(s)
		if s == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, s)
	}
	return out
}
