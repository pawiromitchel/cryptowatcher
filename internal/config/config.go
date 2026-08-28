package config

import (
	"encoding/json"
	"os"
	"path/filepath"

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

// Load reads the configuration from disk. If the file does not exist,
// it saves and returns DefaultConfig().
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
		return model.DefaultConfig(), err
	}

	var cfg model.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return model.DefaultConfig(), nil
	}

	if len(cfg.Pairs) == 0 {
		cfg.Pairs = model.DefaultConfig().Pairs
	}
	if cfg.RefreshInterval <= 0 {
		cfg.RefreshInterval = 5
	}

	return &cfg, nil
}

// Save writes the given configuration to disk.
func Save(cfg *model.Config) error {
	path, err := GetConfigPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
