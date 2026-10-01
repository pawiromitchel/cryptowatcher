// Package model defines the shared data types used across CryptoWatcher.
package model

import "time"

// AssetType categorizes financial instruments.
type AssetType string

const (
	AssetCrypto AssetType = "crypto"
	AssetStock  AssetType = "stock"
)

// Asset holds market data for a cryptocurrency or an equity.
//
// Optional numeric fields (High24h, Low24h, Volume24h, MarketCap) are zero when
// the data source does not provide them; the UI renders those as "—" instead of
// guessing a value.
type Asset struct {
	Symbol      string    `json:"symbol"`       // Standardized form: BTC-USD, TSLA
	Display     string    `json:"display"`      // User-facing form: BTC/USD, TSLA
	Name        string    `json:"name"`         // Full asset name
	Type        AssetType `json:"type"`         // AssetCrypto or AssetStock
	Source      string    `json:"source"`       // Provider that produced this quote
	Price       float64   `json:"price"`        // Current price in USD
	Open24h     float64   `json:"open_24h"`     // Reference price 24h ago
	High24h     float64   `json:"high_24h"`     // 24h high (0 = unknown)
	Low24h      float64   `json:"low_24h"`      // 24h low (0 = unknown)
	Volume24h   float64   `json:"volume_24h"`   // 24h volume (0 = unknown)
	MarketCap   float64   `json:"market_cap"`   // Market capitalization in USD (0 = unknown)
	Change24h   float64   `json:"change_24h"`   // 24h percentage change
	History     []float64 `json:"history"`      // Real price series for the chart; empty if unavailable
	LastUpdated time.Time `json:"last_updated"` // Time of the last successful fetch
	Stale       bool      `json:"-"`            // True when the latest refresh failed and data is old
	Err         error     `json:"-"`            // Last fetch error, if any
}

// HasQuote reports whether the asset has ever received a valid price.
func (a Asset) HasQuote() bool { return a.Price > 0 }
