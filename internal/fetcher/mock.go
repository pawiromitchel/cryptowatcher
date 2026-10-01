package fetcher

import (
	"context"
	"hash/fnv"
	"math"
	"time"

	"cryptowatcher/internal/model"
)

// MockFetcher produces deterministic synthetic quotes for demos (-mock) and
// tests. Unlike the real providers it resolves any valid symbol.
type MockFetcher struct {
	// Now returns the clock used for LastUpdated and price drift; override in tests.
	Now func() time.Time
}

// NewMockFetcher returns a MockFetcher.
func NewMockFetcher() *MockFetcher { return &MockFetcher{Now: time.Now} }

var mockBase = map[string]float64{
	"BTC": 78402, "ETH": 2467.42, "SOL": 185.75, "DOGE": 0.1632, "SPY": 571.2,
	"TSLA": 215.3, "GOOGL": 168.4, "AAPL": 225.5, "NVDA": 131.2, "MSFT": 420.1,
}

// FetchPair returns a synthetic quote for symbol.
func (m *MockFetcher) FetchPair(_ context.Context, raw string) (model.Asset, error) {
	if err := ValidateInput(raw); err != nil {
		return model.Asset{Err: err}, err
	}
	symbol, display := NormalizeSymbol(raw)
	kind := DetectAssetType(raw)
	if kind == "" {
		kind = model.AssetCrypto
	}
	if kind == model.AssetStock {
		symbol = extractBaseTicker(symbol)
	}
	base := extractBaseTicker(symbol)

	h := fnv.New32a()
	_, _ = h.Write([]byte(base))
	seed := float64(h.Sum32()%1000) / 1000

	price, ok := mockBase[base]
	if !ok {
		price = 1 + seed*500
	}
	now := m.Now()
	phase := seed * 2 * math.Pi
	drift := 1 + 0.004*math.Sin(float64(now.Unix())/30+phase)
	price *= drift

	history := make([]float64, 24)
	for i := range history {
		t := float64(i) / 23
		history[i] = price * (1 + 0.03*math.Sin(t*4+phase) - 0.02*(1-t)*(seed-0.5))
	}
	history[len(history)-1] = price

	change := (price/history[0] - 1) * 100
	return model.Asset{
		Symbol:      symbol,
		Display:     display,
		Name:        LookupAssetName(symbol),
		Type:        kind,
		Source:      "mock",
		Price:       price,
		Open24h:     history[0],
		High24h:     price * 1.02,
		Low24h:      price * 0.98,
		Volume24h:   1e6 * (1 + seed*50),
		MarketCap:   price * (1e6 + seed*1e9),
		Change24h:   change,
		History:     history,
		LastUpdated: now,
	}, nil
}

// FetchPrices returns synthetic quotes for symbols.
func (m *MockFetcher) FetchPrices(ctx context.Context, symbols []string) ([]model.Asset, error) {
	out := make([]model.Asset, len(symbols))
	for i, s := range symbols {
		a, err := m.FetchPair(ctx, s)
		if err != nil {
			a.Err = err
		}
		out[i] = a
	}
	return out, nil
}
