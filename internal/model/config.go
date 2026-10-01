package model

const (
	// DefaultRefreshInterval is the default number of seconds between refreshes.
	DefaultRefreshInterval = 15
	// MinRefreshInterval is the lowest accepted refresh interval; lower values
	// would trip the free-tier rate limits of the upstream APIs.
	MinRefreshInterval = 5
)

// Config represents persistent application settings.
type Config struct {
	CryptoPairs     []string `json:"crypto_pairs"`     // Watched crypto symbols (e.g. BTC-USD)
	StockPairs      []string `json:"stock_pairs"`      // Watched stock symbols (e.g. SPY, TSLA)
	Pairs           []string `json:"pairs,omitempty"`  // Legacy single-list field, migrated on load
	RefreshInterval int      `json:"refresh_interval"` // Refresh interval in seconds
}

// DefaultConfig returns the initial configuration.
func DefaultConfig() *Config {
	return &Config{
		CryptoPairs:     []string{"BTC-USD", "ETH-USD", "SOL-USD"},
		StockPairs:      []string{"SPY", "TSLA", "GOOGL", "AAPL"},
		RefreshInterval: DefaultRefreshInterval,
	}
}
