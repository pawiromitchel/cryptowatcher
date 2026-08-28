package model

import "time"

// Candle represents a single OHLC (Open, High, Low, Close) price bar.
type Candle struct {
	Timestamp time.Time `json:"timestamp"`
	Open      float64   `json:"open"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Close     float64   `json:"close"`
	Volume    float64   `json:"volume"`
}

// CryptoPair holds market ticker data for a single cryptocurrency trading pair.
type CryptoPair struct {
	Symbol      string    `json:"symbol"`       // Standardized format: BTC-USD
	Display     string    `json:"display"`      // User-friendly display: BTC/USD
	Price       float64   `json:"price"`        // Current spot price in USD
	Open24h     float64   `json:"open_24h"`     // Price 24 hours ago
	High24h     float64   `json:"high_24h"`     // 24-hour high
	Low24h      float64   `json:"low_24h"`      // 24-hour low
	Volume24h   float64   `json:"volume_24h"`   // 24-hour trading volume
	Change24h   float64   `json:"change_24h"`   // 24-hour percentage change
	Change7D    float64   `json:"change_7d"`    // 7-day percentage change
	History     []float64 `json:"history"`      // 24-hour price history for sparkline chart
	History7D   []float64 `json:"history_7d"`   // 7-day price history points for multi-line chart
	Candles     []Candle  `json:"candles"`      // OHLC candlestick data series
	LastUpdated time.Time `json:"last_updated"` // Timestamp of last successful fetch
	Err         error     `json:"-"`            // Error state if fetch failed
}

// Config represents persistent application settings.
type Config struct {
	Pairs           []string `json:"pairs"`            // List of watched pair symbols (e.g. BTC-USD)
	RefreshInterval int      `json:"refresh_interval"` // Refresh interval in seconds
}

// DefaultConfig returns the initial configuration.
func DefaultConfig() *Config {
	return &Config{
		Pairs:           []string{"BTC-USD", "ETH-USD", "SOL-USD"},
		RefreshInterval: 5,
	}
}
