package fetcher

import (
	"context"
	"time"

	"cryptowatcher/internal/model"
)

// MockFetcher provides synthetic ticker data for testing and offline modes.
type MockFetcher struct {
	MockData map[string]model.CryptoPair
}

// NewMockFetcher initializes a MockFetcher populated with default crypto pair data.
func NewMockFetcher() *MockFetcher {
	return &MockFetcher{
		MockData: map[string]model.CryptoPair{
			"BTC-USD": {
				Symbol:      "BTC-USD",
				Display:     "BTC/USD",
				Price:       96420.50,
				Open24h:     94100.00,
				High24h:     97100.00,
				Low24h:      93800.00,
				Volume24h:   24510.33,
				Change24h:   2.46,
				LastUpdated: time.Now(),
			},
			"ETH-USD": {
				Symbol:      "ETH-USD",
				Display:     "ETH/USD",
				Price:       2750.25,
				Open24h:     2810.00,
				High24h:     2840.00,
				Low24h:      2710.00,
				Volume24h:   184200.50,
				Change24h:   -2.12,
				LastUpdated: time.Now(),
			},
			"SOL-USD": {
				Symbol:      "SOL-USD",
				Display:     "SOL/USD",
				Price:       185.75,
				Open24h:     175.00,
				High24h:     189.50,
				Low24h:      173.20,
				Volume24h:   892000.10,
				Change24h:   6.14,
				LastUpdated: time.Now(),
			},
		},
	}
}

// FetchPair implements PriceFetcher.
func (m *MockFetcher) FetchPair(ctx context.Context, rawInput string) (model.CryptoPair, error) {
	symbol, display := NormalizeSymbol(rawInput)
	if pair, exists := m.MockData[symbol]; exists {
		pair.LastUpdated = time.Now()
		return pair, nil
	}

	return model.CryptoPair{
		Symbol:      symbol,
		Display:     display,
		Price:       100.00,
		Open24h:     95.00,
		High24h:     105.00,
		Low24h:      90.00,
		Volume24h:   10000.00,
		Change24h:   5.26,
		LastUpdated: time.Now(),
	}, nil
}

// FetchPrices implements PriceFetcher.
func (m *MockFetcher) FetchPrices(ctx context.Context, symbols []string) ([]model.CryptoPair, error) {
	pairs := make([]model.CryptoPair, len(symbols))
	for i, s := range symbols {
		p, _ := m.FetchPair(ctx, s)
		pairs[i] = p
	}
	return pairs, nil
}
