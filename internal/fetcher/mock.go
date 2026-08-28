package fetcher

import (
	"context"
	"math"
	"time"

	"cryptowatcher/internal/model"
)

// MockFetcher provides synthetic ticker data for testing and offline modes.
type MockFetcher struct {
	MockData map[string]model.CryptoPair
}

// NewMockFetcher initializes a MockFetcher populated with default crypto pair data.
func NewMockFetcher() *MockFetcher {
	btcCandles := generateMockOHLC(91600.00, 96420.50, 0.0)
	ethCandles := generateMockOHLC(2792.00, 2750.25, 0.5)
	solCandles := generateMockOHLC(165.25, 185.75, 1.0)

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
				Change7D:    5.20,
				History:     []float64{94100, 94500, 93800, 95200, 96000, 97100, 96800, 96420.50},
				History7D:   extractCloses(btcCandles),
				Candles:     btcCandles,
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
				Change7D:    -1.50,
				History:     []float64{2810, 2840, 2820, 2790, 2750, 2710, 2730, 2750.25},
				History7D:   extractCloses(ethCandles),
				Candles:     ethCandles,
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
				Change7D:    12.40,
				History:     []float64{175, 173.2, 178, 182, 186, 189.5, 184, 185.75},
				History7D:   extractCloses(solCandles),
				Candles:     solCandles,
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

	mockCandles := generateMockOHLC(96.6, 100.0, 0.2)
	return model.CryptoPair{
		Symbol:      symbol,
		Display:     display,
		Price:       100.00,
		Open24h:     95.00,
		High24h:     105.00,
		Low24h:      90.00,
		Volume24h:   10000.00,
		Change24h:   5.26,
		Change7D:    3.50,
		History:     []float64{95, 90, 93, 98, 105, 102, 100},
		History7D:   extractCloses(mockCandles),
		Candles:     mockCandles,
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

func generateMockOHLC(start, end float64, phase float64) []model.Candle {
	candles := make([]model.Candle, 28)
	now := time.Now()

	for i := 0; i < 28; i++ {
		t := float64(i) / 27.0
		wave := math.Sin(t*3.14159*3.0+phase) * ((end - start) * 0.2)
		cOpen := start + (end-start)*t + wave
		cClose := cOpen * (1.0 + math.Cos(t*10.0)*0.015)
		cHigh := math.Max(cOpen, cClose) * 1.01
		cLow := math.Min(cOpen, cClose) * 0.99

		candles[i] = model.Candle{
			Timestamp: now.Add(time.Duration(i-28) * 6 * time.Hour),
			Open:      cOpen,
			High:      cHigh,
			Low:       cLow,
			Close:     cClose,
			Volume:    5000.0 * (1.0 + t),
		}
	}
	return candles
}

func extractCloses(candles []model.Candle) []float64 {
	closes := make([]float64, len(candles))
	for i, c := range candles {
		closes[i] = c.Close
	}
	return closes
}
