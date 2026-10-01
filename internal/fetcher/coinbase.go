package fetcher

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"cryptowatcher/internal/model"
)

const (
	defaultCoinbaseBaseURL = "https://api.exchange.coinbase.com"
	candleCacheTTL         = 5 * time.Minute
)

// CoinbaseStatsResponse is the payload of Coinbase Exchange /products/{id}/stats.
type CoinbaseStatsResponse struct {
	Open   string `json:"open"`
	High   string `json:"high"`
	Low    string `json:"low"`
	Last   string `json:"last"`
	Volume string `json:"volume"`
}

// CoinbaseFetcher implements PriceFetcher for exchange-listed crypto pairs.
type CoinbaseFetcher struct {
	baseURL string
	api     *apiClient
	candles *ttlCache[[]float64]
}

// NewCoinbaseFetcher initializes a Coinbase Exchange API client.
func NewCoinbaseFetcher() *CoinbaseFetcher {
	return &CoinbaseFetcher{
		baseURL: defaultCoinbaseBaseURL,
		api:     newAPIClient("Coinbase"),
		candles: newTTLCache[[]float64](candleCacheTTL),
	}
}

// SetBaseURL overrides the API URL (for tests and staging).
func (f *CoinbaseFetcher) SetBaseURL(url string) { f.baseURL = url }

// FetchPair retrieves 24h stats and hourly candles for a pair such as BTC-USD.
func (f *CoinbaseFetcher) FetchPair(ctx context.Context, rawInput string) (model.Asset, error) {
	symbol, display := NormalizeSymbol(rawInput)
	fail := func(err error) (model.Asset, error) {
		return model.Asset{Symbol: symbol, Display: display, Type: model.AssetCrypto, Err: err}, err
	}

	var stats CoinbaseStatsResponse
	if err := f.api.getJSON(ctx, fmt.Sprintf("%s/products/%s/stats", f.baseURL, symbol), &stats); err != nil {
		return fail(err)
	}

	price, _ := strconv.ParseFloat(stats.Last, 64)
	open, _ := strconv.ParseFloat(stats.Open, 64)
	high, _ := strconv.ParseFloat(stats.High, 64)
	low, _ := strconv.ParseFloat(stats.Low, 64)
	volume, _ := strconv.ParseFloat(stats.Volume, 64)
	if price <= 0 {
		return fail(fmt.Errorf("Coinbase: no price for %s", symbol))
	}

	var change float64
	if open > 0 {
		change = (price - open) / open * 100
	}

	history, err := f.fetchCandles(ctx, symbol)
	if err != nil {
		history = nil // chart is optional; never invent one
	}

	return model.Asset{
		Symbol:      symbol,
		Display:     display,
		Name:        LookupAssetName(symbol),
		Type:        model.AssetCrypto,
		Source:      "Coinbase",
		Price:       price,
		Open24h:     open,
		High24h:     high,
		Low24h:      low,
		Volume24h:   volume,
		Change24h:   change,
		History:     history,
		LastUpdated: time.Now(),
	}, nil
}

// fetchCandles returns up to 24 hourly close prices, oldest first. Results are
// cached because hourly candles barely change between refreshes.
func (f *CoinbaseFetcher) fetchCandles(ctx context.Context, symbol string) ([]float64, error) {
	if h, ok := f.candles.get(symbol); ok {
		return h, nil
	}

	var raw [][]float64 // [time, low, high, open, close, volume], newest first
	if err := f.api.getJSON(ctx, fmt.Sprintf("%s/products/%s/candles?granularity=3600", f.baseURL, symbol), &raw); err != nil {
		return nil, err
	}
	if len(raw) > 24 {
		raw = raw[:24]
	}

	prices := make([]float64, 0, len(raw))
	for i := len(raw) - 1; i >= 0; i-- {
		if len(raw[i]) >= 5 {
			prices = append(prices, raw[i][4])
		}
	}
	if len(prices) < 2 {
		return nil, fmt.Errorf("Coinbase: not enough candles for %s", symbol)
	}
	f.candles.set(symbol, prices)
	return prices, nil
}

// FetchPrices fetches several pairs concurrently.
func (f *CoinbaseFetcher) FetchPrices(ctx context.Context, symbols []string) ([]model.Asset, error) {
	return fetchAll(symbols, 4, func(s string) model.Asset {
		a, err := f.FetchPair(ctx, s)
		if err != nil {
			a.Err = err
		}
		return a
	}), nil
}
