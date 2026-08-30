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

// NewMockFetcher initializes a MockFetcher populated with default crypto and stock data.
func NewMockFetcher() *MockFetcher {
	btcCandles := generateMockOHLC(91600.00, 96420.50, 0.0)
	ethCandles := generateMockOHLC(2792.00, 2750.25, 0.5)
	solCandles := generateMockOHLC(165.25, 185.75, 1.0)
	spyCandles := generateMockOHLC(560.00, 571.20, 0.2)
	tslaCandles := generateMockOHLC(208.00, 215.30, 0.8)
	googlCandles := generateMockOHLC(164.00, 168.40, 0.4)
	aaplCandles := generateMockOHLC(220.00, 225.50, 0.6)

	return &MockFetcher{
		MockData: map[string]model.CryptoPair{
			"BTC-USD": {
				Symbol:      "BTC-USD",
				Display:     "BTC-USD",
				Name:        "Bitcoin USD",
				Type:        model.AssetCrypto,
				Price:       78402.00,
				Open24h:     77584.00,
				High24h:     78500.00,
				Low24h:      77557.00,
				Volume24h:   24510.33,
				MarketCap:   "$1.57T",
				Change24h:   1.05,
				Change7D:    5.20,
				History:     generateMock7D(77584.00, 78402.00, 0.0),
				History7D:   extractCloses(btcCandles),
				Candles:     btcCandles,
				LastUpdated: time.Now(),
			},
			"ETH-USD": {
				Symbol:      "ETH-USD",
				Display:     "ETH-USD",
				Name:        "Ethereum USD",
				Type:        model.AssetCrypto,
				Price:       2467.42,
				Open24h:     2435.00,
				High24h:     2470.00,
				Low24h:      2432.00,
				Volume24h:   184200.50,
				MarketCap:   "$297.3B",
				Change24h:   1.31,
				Change7D:    -1.50,
				History:     generateMock7D(2435.00, 2467.42, 0.5),
				History7D:   extractCloses(ethCandles),
				Candles:     ethCandles,
				LastUpdated: time.Now(),
			},
			"SOL-USD": {
				Symbol:      "SOL-USD",
				Display:     "SOL-USD",
				Name:        "Solana USD",
				Type:        model.AssetCrypto,
				Price:       105.52,
				Open24h:     103.60,
				High24h:     106.20,
				Low24h:      103.10,
				Volume24h:   892000.10,
				MarketCap:   "$49.6B",
				Change24h:   1.82,
				Change7D:    10.50,
				History:     generateMock7D(103.60, 105.52, 1.0),
				History7D:   extractCloses(solCandles),
				Candles:     solCandles,
				LastUpdated: time.Now(),
			},
			"SPY": {
				Symbol:      "SPY",
				Display:     "S&P 500",
				Name:        "S&P 500 ETF",
				Type:        model.AssetStock,
				Price:       571.20,
				Open24h:     573.20,
				High24h:     574.00,
				Low24h:      570.10,
				Volume24h:   55000000.0,
				MarketCap:   "$550B",
				Change24h:   -0.35,
				Change7D:    1.20,
				History:     generateMock7D(573.20, 571.20, 0.2),
				History7D:   extractCloses(spyCandles),
				Candles:     spyCandles,
				LastUpdated: time.Now(),
			},
			"TSLA": {
				Symbol:      "TSLA",
				Display:     "TSLA",
				Name:        "Tesla Inc",
				Type:        model.AssetStock,
				Price:       215.30,
				Open24h:     208.20,
				High24h:     216.50,
				Low24h:      207.80,
				Volume24h:   82000000.0,
				MarketCap:   "$685B",
				Change24h:   3.41,
				Change7D:    6.80,
				History:     generateMock7D(208.20, 215.30, 0.8),
				History7D:   extractCloses(tslaCandles),
				Candles:     tslaCandles,
				LastUpdated: time.Now(),
			},
			"GOOGL": {
				Symbol:      "GOOGL",
				Display:     "GOOGL",
				Name:        "Alphabet Inc",
				Type:        model.AssetStock,
				Price:       168.40,
				Open24h:     167.00,
				High24h:     169.10,
				Low24h:      166.50,
				Volume24h:   21000000.0,
				MarketCap:   "$2.05T",
				Change24h:   0.84,
				Change7D:    2.10,
				History:     generateMock7D(167.00, 168.40, 0.4),
				History7D:   extractCloses(googlCandles),
				Candles:     googlCandles,
				LastUpdated: time.Now(),
			},
			"AAPL": {
				Symbol:      "AAPL",
				Display:     "AAPL",
				Name:        "Apple Inc",
				Type:        model.AssetStock,
				Price:       225.50,
				Open24h:     223.10,
				High24h:     226.20,
				Low24h:      222.80,
				Volume24h:   45000000.0,
				MarketCap:   "$3.45T",
				Change24h:   1.08,
				Change7D:    3.50,
				History:     generateMock7D(223.10, 225.50, 0.6),
				History7D:   extractCloses(aaplCandles),
				Candles:     aaplCandles,
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

	assetType := DetectAssetType(symbol)
	mockCandles := generateMockOHLC(96.6, 100.0, 0.2)
	name := LookupAssetName(symbol)
	return model.CryptoPair{
		Symbol:      symbol,
		Display:     display,
		Name:        name,
		Type:        assetType,
		Price:       100.00,
		Open24h:     95.00,
		High24h:     105.00,
		Low24h:      90.00,
		Volume24h:   10000.00,
		MarketCap:   "$10.0B",
		Change24h:   5.26,
		Change7D:    3.50,
		History:     generateMock7D(95.0, 100.0, 0.2),
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

func generateMock7D(start, end float64, phase float64) []float64 {
	pts := make([]float64, 28)
	for i := 0; i < 28; i++ {
		t := float64(i) / 27.0
		wave := math.Sin(t*3.14159*3.0+phase) * ((end - start) * 0.2)
		pts[i] = start + (end-start)*t + wave
	}
	return pts
}

func extractCloses(candles []model.Candle) []float64 {
	closes := make([]float64, len(candles))
	for i, c := range candles {
		closes[i] = c.Close
	}
	return closes
}
