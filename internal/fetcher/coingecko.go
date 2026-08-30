package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"cryptowatcher/internal/model"
)

const (
	defaultCoinGeckoBaseURL = "https://api.coingecko.com/api/v3"
)

// CoinGeckoSearchResponse represents CoinGecko search API response
type CoinGeckoSearchResponse struct {
	Coins []struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Symbol string `json:"symbol"`
	} `json:"coins"`
}

// CoinGeckoMarketData represents CoinGecko coin detail endpoint response
type CoinGeckoMarketData struct {
	ID         string `json:"id"`
	Symbol     string `json:"symbol"`
	Name       string `json:"name"`
	MarketData struct {
		CurrentPrice struct {
			USD float64 `json:"usd"`
		} `json:"current_price"`
		MarketCap struct {
			USD float64 `json:"usd"`
		} `json:"market_cap"`
		TotalVolume struct {
			USD float64 `json:"usd"`
		} `json:"total_volume"`
		High24h struct {
			USD float64 `json:"usd"`
		} `json:"high_24h"`
		Low24h struct {
			USD float64 `json:"usd"`
		} `json:"low_24h"`
		PriceChange24hPercentage float64 `json:"price_change_percentage_24h"`
		Sparkline7D              struct {
			Price []float64 `json:"price"`
		} `json:"sparkline_7d"`
	} `json:"market_data"`
}

// CoinGeckoFetcher implements PriceFetcher using the CoinGecko public API.
type CoinGeckoFetcher struct {
	baseURL    string
	httpClient *http.Client
	cacheMu    sync.RWMutex
	idCache    map[string]string
}

// NewCoinGeckoFetcher initializes a new CoinGecko API client.
func NewCoinGeckoFetcher() *CoinGeckoFetcher {
	return &CoinGeckoFetcher{
		baseURL: defaultCoinGeckoBaseURL,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
		idCache: map[string]string{
			"HMM":          "thinking-cat",
			"THINKING-CAT": "thinking-cat",
			"PEPE":         "pepe",
			"BONK":         "bonk",
			"WIF":          "dogwifcoin",
			"SHIB":         "shiba-inu",
			"FLOKI":        "floki",
			"RENDER":       "render-token",
			"SUI":          "sui",
			"SEI":          "sei-network",
			"NEAR":         "near",
			"APT":          "aptos",
			"INJ":          "injective-protocol",
			"KAS":          "kaspa",
			"TON":          "the-open-network",
		},
	}
}

// SetBaseURL overrides the default API URL (useful for testing).
func (f *CoinGeckoFetcher) SetBaseURL(url string) {
	f.baseURL = url
}

// FetchPair retrieves market data and price history for a cryptocurrency from CoinGecko.
func (f *CoinGeckoFetcher) FetchPair(ctx context.Context, rawInput string) (model.CryptoPair, error) {
	symbol, display := NormalizeSymbol(rawInput)
	baseTicker := extractBaseTicker(symbol)

	coinID, err := f.resolveCoinID(ctx, baseTicker)
	if err != nil {
		return model.CryptoPair{Symbol: symbol, Display: display, Err: err}, err
	}

	reqURL := fmt.Sprintf("%s/coins/%s?localization=false&tickers=false&market_data=true&community_data=false&developer_data=false&sparkline=true",
		f.baseURL, coinID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return model.CryptoPair{Symbol: symbol, Display: display, Err: err}, err
	}

	req.Header.Set("User-Agent", defaultUserAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return model.CryptoPair{Symbol: symbol, Display: display, Err: err}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		apiErr := fmt.Errorf("CoinGecko HTTP %d: coin not found", resp.StatusCode)
		return model.CryptoPair{Symbol: symbol, Display: display, Err: apiErr}, apiErr
	}

	var coinResp CoinGeckoMarketData
	if err := json.NewDecoder(resp.Body).Decode(&coinResp); err != nil {
		return model.CryptoPair{Symbol: symbol, Display: display, Err: err}, err
	}

	price := coinResp.MarketData.CurrentPrice.USD
	change24h := coinResp.MarketData.PriceChange24hPercentage
	high24h := coinResp.MarketData.High24h.USD
	low24h := coinResp.MarketData.Low24h.USD
	volume24h := coinResp.MarketData.TotalVolume.USD
	open24h := price / (1.0 + (change24h / 100.0))

	// Extract last 24 points from sparkline for 24h mini line chart
	var history []float64
	rawSparkline := coinResp.MarketData.Sparkline7D.Price
	if len(rawSparkline) > 0 {
		if len(rawSparkline) > 24 {
			history = rawSparkline[len(rawSparkline)-24:]
		} else {
			history = rawSparkline
		}
	} else {
		history = generateTrendHistory(open24h, low24h, high24h, price)
	}

	name := coinResp.Name
	if name == "" {
		name = LookupAssetName(baseTicker)
	}

	marketCap := formatRawMarketCap(coinResp.MarketData.MarketCap.USD)

	pair := model.CryptoPair{
		Symbol:      symbol,
		Display:     display,
		Name:        name,
		Type:        model.AssetCrypto,
		Price:       price,
		Open24h:     open24h,
		High24h:     high24h,
		Low24h:      low24h,
		Volume24h:   volume24h,
		MarketCap:   marketCap,
		Change24h:   change24h,
		History:     history,
		LastUpdated: time.Now(),
	}

	return pair, nil
}

// FetchPrices fetches market data concurrently for a slice of symbols.
func (f *CoinGeckoFetcher) FetchPrices(ctx context.Context, symbols []string) ([]model.CryptoPair, error) {
	results := make([]model.CryptoPair, len(symbols))
	for i, s := range symbols {
		p, err := f.FetchPair(ctx, s)
		if err != nil {
			p.Err = err
		}
		results[i] = p
	}
	return results, nil
}

func (f *CoinGeckoFetcher) resolveCoinID(ctx context.Context, ticker string) (string, error) {
	upper := strings.ToUpper(ticker)

	f.cacheMu.RLock()
	id, found := f.idCache[upper]
	f.cacheMu.RUnlock()
	if found {
		return id, nil
	}

	// Query CoinGecko search API
	reqURL := fmt.Sprintf("%s/search?query=%s", f.baseURL, ticker)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", defaultUserAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("search failed with HTTP %d", resp.StatusCode)
	}

	var searchResp CoinGeckoSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return "", err
	}

	if len(searchResp.Coins) == 0 {
		return "", fmt.Errorf("coin %s not found on CoinGecko", ticker)
	}

	// Match exact symbol or take top result
	resolvedID := searchResp.Coins[0].ID
	for _, c := range searchResp.Coins {
		if strings.EqualFold(c.Symbol, ticker) {
			resolvedID = c.ID
			break
		}
	}

	f.cacheMu.Lock()
	f.idCache[upper] = resolvedID
	f.cacheMu.Unlock()

	return resolvedID, nil
}

func formatRawMarketCap(cap float64) string {
	if cap >= 1_000_000_000_000 {
		return fmt.Sprintf("$%.2fT", cap/1e12)
	}
	if cap >= 1_000_000_000 {
		return fmt.Sprintf("$%.1fB", cap/1e9)
	}
	if cap >= 1_000_000 {
		return fmt.Sprintf("$%.1fM", cap/1e6)
	}
	if cap >= 1_000 {
		return fmt.Sprintf("$%.1fK", cap/1e3)
	}
	if cap > 0 {
		return fmt.Sprintf("$%.0f", cap)
	}
	return "$--M"
}
