package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"cryptowatcher/internal/model"
)

func setup(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	return filepath.Join(dir, configDirName, configFileName)
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestConfigPathHonoursXDG(t *testing.T) {
	want := setup(t)
	got, err := GetConfigPath()
	if err != nil || got != want {
		t.Fatalf("GetConfigPath() = %q, %v; want %q", got, err, want)
	}
}

func TestLoadMissingFileCreatesDefaults(t *testing.T) {
	path := setup(t)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cfg, model.DefaultConfig()) {
		t.Errorf("got %+v", cfg)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("defaults should be persisted: %v", err)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	setup(t)
	cfg := &model.Config{CryptoPairs: []string{"BTC-USD", "DOGE-USD"}, StockPairs: []string{"NVDA"}, RefreshInterval: 20}
	if err := Save(cfg); err != nil {
		t.Fatal(err)
	}
	got, err := Load()
	if err != nil || !reflect.DeepEqual(got, cfg) {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func TestSaveLeavesNoTempFiles(t *testing.T) {
	path := setup(t)
	for i := 0; i < 3; i++ {
		if err := Save(model.DefaultConfig()); err != nil {
			t.Fatal(err)
		}
	}
	entries, _ := os.ReadDir(filepath.Dir(path))
	if len(entries) != 1 {
		t.Errorf("expected only config.json, found %d entries", len(entries))
	}
}

func TestCorruptConfigIsBackedUpNotOverwritten(t *testing.T) {
	path := setup(t)
	const junk = `{"crypto_pairs": ["BTC-USD",` // truncated JSON
	write(t, path, junk)

	cfg, err := Load()
	if err == nil {
		t.Fatal("expected a warning error for corrupt config")
	}
	if cfg == nil || len(cfg.CryptoPairs) == 0 {
		t.Fatal("Load must still return usable defaults")
	}
	backup, readErr := os.ReadFile(path + ".bak")
	if readErr != nil || string(backup) != junk {
		t.Fatalf("original content must be preserved in .bak, got %q (%v)", backup, readErr)
	}
}

func TestEmptiedListsStayEmpty(t *testing.T) {
	path := setup(t)
	write(t, path, `{"crypto_pairs": ["BTC-USD"], "stock_pairs": [], "refresh_interval": 10}`)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.StockPairs) != 0 {
		t.Errorf("a deliberately empty stock list must not be refilled with defaults: %v", cfg.StockPairs)
	}

	cfg.StockPairs = nil
	if err := Save(cfg); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if !reflect.DeepEqual(mustLoad(t).StockPairs, []string{}) {
		t.Errorf("round trip of empty list failed; file: %s", data)
	}
}

func mustLoad(t *testing.T) *model.Config {
	t.Helper()
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func TestLegacyPairsMigration(t *testing.T) {
	path := setup(t)
	write(t, path, `{"pairs": ["BTC-USD", "ETH-USD"]}`)
	cfg := mustLoad(t)
	if !reflect.DeepEqual(cfg.CryptoPairs, []string{"BTC-USD", "ETH-USD"}) {
		t.Errorf("legacy pairs not migrated: %+v", cfg)
	}
	if cfg.Pairs != nil {
		t.Error("legacy field should be cleared after migration")
	}
}

func TestNormalisation(t *testing.T) {
	path := setup(t)
	write(t, path, `{"crypto_pairs": ["BTC-USD","btc-usd"," ","ETH-USD"], "stock_pairs": ["AAPL","AAPL"], "refresh_interval": 1}`)
	cfg := mustLoad(t)
	if !reflect.DeepEqual(cfg.CryptoPairs, []string{"BTC-USD", "ETH-USD"}) || !reflect.DeepEqual(cfg.StockPairs, []string{"AAPL"}) {
		t.Errorf("not deduplicated: %+v", cfg)
	}
	if cfg.RefreshInterval != model.MinRefreshInterval {
		t.Errorf("interval below the minimum must be clamped, got %d", cfg.RefreshInterval)
	}

	write(t, path, `{"crypto_pairs":["BTC-USD"],"stock_pairs":["AAPL"]}`)
	if got := mustLoad(t).RefreshInterval; got != model.DefaultRefreshInterval {
		t.Errorf("missing interval should default, got %d", got)
	}
}
