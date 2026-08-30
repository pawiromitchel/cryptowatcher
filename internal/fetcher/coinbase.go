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

// FetchPair retrieves market stats and 24h candle history for a product ID (e.g. BTC-USD).
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

	// Fetch 24h 1-hour candles for detailed widget mini-chart curve
	history, errHist := f.FetchCandles24H(ctx, symbol)
	if errHist != nil || len(history) < 5 {
		history = generateTrendHistory(open, low, high, price)
	}

	name := LookupAssetName(symbol)
	marketCap := formatCryptoMarketCap(symbol, price)

	pair := model.CryptoPair{
		Symbol:      symbol,
		Display:     display,
		Name:        name,
		Type:        model.AssetCrypto,
		Price:       price,
		Open24h:     open,
		High24h:     high,
		Low24h:      low,
		Volume24h:   volume,
		MarketCap:   marketCap,
		Change24h:   change24h,
		History:     history,
		LastUpdated: time.Now(),
	}

	return pair, nil
}

// FetchCandles24H retrieves 24-hour hourly candle close prices from Coinbase API.
func (f *CoinbaseFetcher) FetchCandles24H(ctx context.Context, symbol string) ([]float64, error) {
	reqURL := fmt.Sprintf("%s/products/%s/candles?granularity=3600", f.baseURL, symbol)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "CryptoWatcher/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var raw [][]float64
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	if len(raw) > 24 {
		raw = raw[:24]
	}

	prices := make([]float64, len(raw))
	for i := 0; i < len(raw); i++ {
		if len(raw[i]) >= 5 {
			prices[len(raw)-1-i] = raw[i][4]
		}
	}

	return prices, nil
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
		open + (mid1-open)*0.4,
		low,
		low + (mid2-low)*0.3,
		mid2,
		mid2 + (high-mid2)*0.5,
		high,
		high - (high-mid3)*0.3,
		mid3,
		price,
	}
}

func formatCryptoMarketCap(symbol string, price float64) string {
	base := extractBaseTicker(symbol)
	switch base {
	case "BTC":
		return fmt.Sprintf("$%.2fT", (price*19800000)/1e12)
	case "ETH":
		return fmt.Sprintf("$%.1fB", (price*120400000)/1e9)
	case "SOL":
		return fmt.Sprintf("$%.1fB", (price*470000000)/1e9)
	case "DOGE":
		return fmt.Sprintf("$%.1fB", (price*147000000000)/1e9)
	case "ADA":
		return fmt.Sprintf("$%.1fB", (price*35000000000)/1e9)
	case "AVAX":
		return fmt.Sprintf("$%.1fB", (price*400000000)/1e9)
	default:
		return "$--B"
	}
}
