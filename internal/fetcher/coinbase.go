package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"cryptowatcher/internal/model"
)

const (
	defaultCoinbaseBaseURL = "https://api.exchange.coinbase.com"
	defaultTimeout        = 5 * time.Second
)

// CoinbaseStatsResponse represents the payload returned by Coinbase Exchange API /products/{id}/stats
type CoinbaseStatsResponse struct {
	Open   string `json:"open"`
	High   string `json:"high"`
	Low    string `json:"low"`
	Last   string `json:"last"`
	Volume string `json:"volume"`
}

// CoinbaseFetcher implements PriceFetcher using the Coinbase Exchange REST API.
type CoinbaseFetcher struct {
	baseURL    string
	httpClient *http.Client
}

// NewCoinbaseFetcher initializes a new Coinbase API client.
func NewCoinbaseFetcher() *CoinbaseFetcher {
	return &CoinbaseFetcher{
		baseURL: defaultCoinbaseBaseURL,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// SetBaseURL overrides default API URL (useful for testing).
func (f *CoinbaseFetcher) SetBaseURL(url string) {
	f.baseURL = url
}

// FetchPair retrieves 24h market stats for a single product ID (e.g. BTC-USD).
func (f *CoinbaseFetcher) FetchPair(ctx context.Context, rawInput string) (model.CryptoPair, error) {
	symbol, display := NormalizeSymbol(rawInput)

	reqURL := fmt.Sprintf("%s/products/%s/stats", f.baseURL, symbol)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return model.CryptoPair{Symbol: symbol, Display: display, Err: err}, err
	}

	req.Header.Set("User-Agent", "CryptoWatcher/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return model.CryptoPair{Symbol: symbol, Display: display, Err: err}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		apiErr := fmt.Errorf("HTTP %d: product not found or API error", resp.StatusCode)
		return model.CryptoPair{Symbol: symbol, Display: display, Err: apiErr}, apiErr
	}

	var stats CoinbaseStatsResponse
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return model.CryptoPair{Symbol: symbol, Display: display, Err: err}, err
	}

	price, _ := strconv.ParseFloat(stats.Last, 64)
	open, _ := strconv.ParseFloat(stats.Open, 64)
	high, _ := strconv.ParseFloat(stats.High, 64)
	low, _ := strconv.ParseFloat(stats.Low, 64)
	volume, _ := strconv.ParseFloat(stats.Volume, 64)

	var change24h float64
	if open > 0 {
		change24h = ((price - open) / open) * 100.0
	}

	history := generateTrendHistory(open, low, high, price)

	pair := model.CryptoPair{
		Symbol:      symbol,
		Display:     display,
		Price:       price,
		Open24h:     open,
		High24h:     high,
		Low24h:      low,
		Volume24h:   volume,
		Change24h:   change24h,
		History:     history,
		LastUpdated: time.Now(),
	}

	return pair, nil
}

// FetchPrices fetches market stats concurrently for a slice of cryptocurrency pair symbols.
func (f *CoinbaseFetcher) FetchPrices(ctx context.Context, symbols []string) ([]model.CryptoPair, error) {
	results := make([]model.CryptoPair, len(symbols))
	var wg sync.WaitGroup

	for i, raw := range symbols {
		wg.Add(1)
		go func(idx int, sym string) {
			defer wg.Done()
			pair, err := f.FetchPair(ctx, sym)
			if err != nil {
				pair.Err = err
			}
			results[idx] = pair
		}(i, raw)
	}

	wg.Wait()
	return results, nil
}

func generateTrendHistory(open, low, high, price float64) []float64 {
	if open == 0 && low == 0 && high == 0 {
		return []float64{price, price, price, price, price, price, price, price, price, price}
	}
	mid1 := (open + low) / 2.0
	mid2 := (low + high) / 2.0
	mid3 := (high + price) / 2.0

	return []float64{
		open,
		open + (mid1-open)*0.5,
		low,
		low + (mid2-low)*0.4,
		mid2,
		mid2 + (high-mid2)*0.6,
		high,
		high - (high-mid3)*0.4,
		mid3,
		price,
	}
}
