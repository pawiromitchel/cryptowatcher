package model

import "time"

// AssetType categorizes financial instruments.
type AssetType string

const (
	AssetCrypto AssetType = "crypto"
	AssetStock  AssetType = "stock"
)

// Candle represents a single OHLC (Open, High, Low, Close) price bar.
type Candle struct {
	Timestamp time.Time `json:"timestamp"`
	Open      float64   `json:"open"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Close     float64   `json:"close"`
	Volume    float64   `json:"volume"`
}

// CryptoPair holds market ticker data for a cryptocurrency or equity asset.
type CryptoPair struct {
	Symbol      string    `json:"symbol"`       // Standardized format: BTC-USD, TSLA, SPY
	Display     string    `json:"display"`      // User-friendly display: BTC/USD, TSLA, S&P 500
	Name        string    `json:"name"`         // Full asset name: Bitcoin USD, Tesla Inc, S&P 500 ETF
	Type        AssetType `json:"type"`         // AssetCrypto or AssetStock
	Price       float64   `json:"price"`        // Current spot price in USD
	Open24h     float64   `json:"open_24h"`     // Price 24 hours ago
	High24h     float64   `json:"high_24h"`     // 24-hour high
	Low24h      float64   `json:"low_24h"`      // 24-hour low
	Volume24h   float64   `json:"volume_24h"`   // 24-hour trading volume
	MarketCap   string    `json:"market_cap"`   // Formatted Market Cap e.g. "1.57T", "297.3B"
	Change24h   float64   `json:"change_24h"`   // 24-hour percentage change
	Change7D    float64   `json:"change_7d"`    // 7-day percentage change
	History     []float64 `json:"history"`      // Recent price history for inline widget chart
	History7D   []float64 `json:"history_7d"`   // 7-day price history points
	Candles     []Candle  `json:"candles"`      // OHLC candlestick series
	LastUpdated time.Time `json:"last_updated"` // Timestamp of last successful fetch
	Err         error     `json:"-"`            // Error state if fetch failed
}

// Config represents persistent application settings for crypto and stock watchlists.
type Config struct {
	CryptoPairs     []string `json:"crypto_pairs"`     // List of watched crypto symbols (e.g. BTC-USD)
	StockPairs      []string `json:"stock_pairs"`      // List of watched stock symbols (e.g. SPY, TSLA)
	Pairs           []string `json:"pairs,omitempty"`  // Legacy fallback
	RefreshInterval int      `json:"refresh_interval"` // Refresh interval in seconds
}

// DefaultConfig returns the initial configuration with default crypto and stock watchlists.
func DefaultConfig() *Config {
	return &Config{
		CryptoPairs:     []string{"BTC-USD", "ETH-USD", "SOL-USD"},
		StockPairs:      []string{"SPY", "TSLA", "GOOGL", "AAPL"},
		RefreshInterval: 5,
	}
}
