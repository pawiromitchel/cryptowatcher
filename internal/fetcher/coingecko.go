package fetcher

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"cryptowatcher/internal/model"
)

const (
	defaultCoinGeckoBaseURL   = "https://api.coingecko.com/api/v3"
	defaultDexScreenerBaseURL = "https://api.dexscreener.com/latest/dex"

	quoteCacheTTL     = 60 * time.Second // CoinGecko's free tier is ~30 calls/min
	marketCapCacheTTL = 10 * time.Minute
	marketCapRetry    = time.Minute
)

// CoinGeckoSearchResponse is the CoinGecko /search payload.
type CoinGeckoSearchResponse struct {
	Coins []struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Symbol string `json:"symbol"`
	} `json:"coins"`
}

// CoinGeckoMarketData is the CoinGecko /coins/{id} payload (fields we use).
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

// DexScreenerSearchResponse is the DexScreener /search payload (fields we use).
type DexScreenerSearchResponse struct {
	Pairs []struct {
		ChainID   string `json:"chainId"`
		DexID     string `json:"dexId"`
		BaseToken struct {
			Name   string `json:"name"`
			Symbol string `json:"symbol"`
		} `json:"baseToken"`
		PriceUSD    string  `json:"priceUsd"`
		FDV         float64 `json:"fdv"`
		MarketCap   float64 `json:"marketCap"`
		PriceChange struct {
			H24 float64 `json:"h24"`
		} `json:"priceChange"`
		Volume struct {
			H24 float64 `json:"h24"`
		} `json:"volume"`
		Liquidity struct {
			USD float64 `json:"usd"`
		} `json:"liquidity"`
	} `json:"pairs"`
}

type marketCapEntry struct {
	value     float64
	fetchedAt time.Time
	retryAt   time.Time
}

// CoinGeckoFetcher implements PriceFetcher for long-tail crypto using CoinGecko,
// falling back to DexScreener for on-chain tokens CoinGecko does not list.
//
// Only exact ticker matches are accepted; a lookup never "best guesses" a
// different asset.
type CoinGeckoFetcher struct {
	coingeckoURL   string
	dexscreenerURL string
	gecko          *apiClient
	dex            *apiClient

	cacheMu sync.RWMutex
	idCache map[string]string

	quotes *ttlCache[model.Asset]
	capsMu sync.Mutex
	caps   map[string]marketCapEntry
}

// NewCoinGeckoFetcher initializes the CoinGecko/DexScreener client.
func NewCoinGeckoFetcher() *CoinGeckoFetcher {
	ids := make(map[string]string, len(coinGeckoIDs))
	for k, v := range coinGeckoIDs {
		ids[k] = v
	}
	return &CoinGeckoFetcher{
		coingeckoURL:   defaultCoinGeckoBaseURL,
		dexscreenerURL: defaultDexScreenerBaseURL,
		gecko:          newAPIClient("CoinGecko"),
		dex:            newAPIClient("DexScreener"),
		idCache:        ids,
		quotes:         newTTLCache[model.Asset](quoteCacheTTL),
		caps:           map[string]marketCapEntry{},
	}
}

// SetBaseURL overrides the CoinGecko API URL (for tests and staging).
func (f *CoinGeckoFetcher) SetBaseURL(url string) { f.coingeckoURL = url }

// SetDexScreenerURL overrides the DexScreener API URL (for tests and staging).
func (f *CoinGeckoFetcher) SetDexScreenerURL(url string) { f.dexscreenerURL = url }

// FetchPair retrieves a crypto quote, preferring CoinGecko and falling back to DexScreener.
func (f *CoinGeckoFetcher) FetchPair(ctx context.Context, rawInput string) (model.Asset, error) {
	if DetectAssetType(rawInput) == model.AssetStock {
		err := fmt.Errorf("%s is a stock, not a crypto asset", rawInput)
		return model.Asset{Err: err}, err
	}

	symbol, display := NormalizeSymbol(rawInput)
	base := extractBaseTicker(symbol)

	if a, ok := f.quotes.get(symbol); ok {
		return a, nil
	}

	asset, geckoErr := f.fetchCoinGecko(ctx, base, symbol, display)
	if geckoErr == nil {
		f.quotes.set(symbol, asset)
		return asset, nil
	}

	asset, dexErr := f.fetchDexScreener(ctx, base, symbol, display)
	if dexErr == nil {
		f.quotes.set(symbol, asset)
		return asset, nil
	}

	err := fmt.Errorf("%v; %v", geckoErr, dexErr)
	return model.Asset{Symbol: symbol, Display: display, Type: model.AssetCrypto, Err: err}, err
}

func (f *CoinGeckoFetcher) fetchCoinGecko(ctx context.Context, base, symbol, display string) (model.Asset, error) {
	id, err := f.resolveCoinID(ctx, base)
	if err != nil {
		return model.Asset{}, err
	}

	var resp CoinGeckoMarketData
	endpoint := fmt.Sprintf("%s/coins/%s?localization=false&tickers=false&market_data=true&community_data=false&developer_data=false&sparkline=true",
		f.coingeckoURL, url.PathEscape(id))
	if err := f.gecko.getJSON(ctx, endpoint, &resp); err != nil {
		return model.Asset{}, err
	}

	md := resp.MarketData
	price := md.CurrentPrice.USD
	if price <= 0 {
		return model.Asset{}, fmt.Errorf("CoinGecko: no price for %s", base)
	}
	change := md.PriceChange24hPercentage

	// The 7d sparkline is hourly; its last 24 points are the last day.
	history := md.Sparkline7D.Price
	if len(history) > 24 {
		history = history[len(history)-24:]
	}
	if len(history) < 2 {
		history = nil
	}

	name := resp.Name
	if name == "" {
		name = LookupAssetName(base)
	}

	f.rememberMarketCap(base, md.MarketCap.USD)

	return model.Asset{
		Symbol:      symbol,
		Display:     display,
		Name:        name,
		Type:        model.AssetCrypto,
		Source:      "CoinGecko",
		Price:       price,
		Open24h:     price / (1 + change/100),
		High24h:     md.High24h.USD,
		Low24h:      md.Low24h.USD,
		Volume24h:   md.TotalVolume.USD,
		MarketCap:   md.MarketCap.USD,
		Change24h:   change,
		History:     history,
		LastUpdated: time.Now(),
	}, nil
}

func (f *CoinGeckoFetcher) fetchDexScreener(ctx context.Context, base, symbol, display string) (model.Asset, error) {
	var resp DexScreenerSearchResponse
	if err := f.dex.getJSON(ctx, fmt.Sprintf("%s/search?q=%s", f.dexscreenerURL, url.QueryEscape(base)), &resp); err != nil {
		return model.Asset{}, err
	}

	// Exact symbol matches only; among them take the deepest liquidity.
	best := -1
	for i, p := range resp.Pairs {
		if !strings.EqualFold(p.BaseToken.Symbol, base) {
			continue
		}
		if price, _ := strconv.ParseFloat(p.PriceUSD, 64); price <= 0 {
			continue
		}
		if best < 0 || p.Liquidity.USD > resp.Pairs[best].Liquidity.USD {
			best = i
		}
	}
	if best < 0 {
		return model.Asset{}, fmt.Errorf("DexScreener: no token with symbol %s", base)
	}

	p := resp.Pairs[best]
	price, _ := strconv.ParseFloat(p.PriceUSD, 64)
	capUSD := p.MarketCap
	if capUSD == 0 {
		capUSD = p.FDV
	}
	name := p.BaseToken.Name
	if name == "" {
		name = LookupAssetName(base)
	}

	return model.Asset{
		Symbol:      symbol,
		Display:     display,
		Name:        name,
		Type:        model.AssetCrypto,
		Source:      "DexScreener (" + p.ChainID + ")",
		Price:       price,
		Open24h:     price / (1 + p.PriceChange.H24/100),
		Volume24h:   p.Volume.H24,
		MarketCap:   capUSD,
		Change24h:   p.PriceChange.H24,
		LastUpdated: time.Now(),
	}, nil
}

// FetchPrices fetches several symbols with bounded concurrency.
func (f *CoinGeckoFetcher) FetchPrices(ctx context.Context, symbols []string) ([]model.Asset, error) {
	return fetchAll(symbols, 2, func(s string) model.Asset {
		a, err := f.FetchPair(ctx, s)
		if err != nil {
			a.Err = err
		}
		return a
	}), nil
}

// MarketCap returns the USD market cap for a base ticker. Values are cached
// for ten minutes and failures are retried at most once a minute, so it is
// cheap to call on every refresh. It returns 0 when the cap is unknown.
func (f *CoinGeckoFetcher) MarketCap(ctx context.Context, base string) float64 {
	base = strings.ToUpper(base)
	now := time.Now()

	f.capsMu.Lock()
	e, ok := f.caps[base]
	f.capsMu.Unlock()
	if ok && (now.Sub(e.fetchedAt) < marketCapCacheTTL || now.Before(e.retryAt)) {
		return e.value
	}

	fail := func() float64 {
		f.capsMu.Lock()
		e.retryAt = now.Add(marketCapRetry)
		f.caps[base] = e
		f.capsMu.Unlock()
		return e.value
	}

	id, err := f.resolveCoinID(ctx, base)
	if err != nil {
		return fail()
	}
	var resp map[string]struct {
		MarketCap float64 `json:"usd_market_cap"`
	}
	endpoint := fmt.Sprintf("%s/simple/price?ids=%s&vs_currencies=usd&include_market_cap=true", f.coingeckoURL, url.QueryEscape(id))
	if err := f.gecko.getJSON(ctx, endpoint, &resp); err != nil {
		return fail()
	}
	f.rememberMarketCap(base, resp[id].MarketCap)
	return resp[id].MarketCap
}

func (f *CoinGeckoFetcher) rememberMarketCap(base string, value float64) {
	if value <= 0 {
		return
	}
	f.capsMu.Lock()
	f.caps[strings.ToUpper(base)] = marketCapEntry{value: value, fetchedAt: time.Now()}
	f.capsMu.Unlock()
}

// resolveCoinID maps a ticker to a CoinGecko ID, accepting exact symbol matches only.
func (f *CoinGeckoFetcher) resolveCoinID(ctx context.Context, ticker string) (string, error) {
	upper := strings.ToUpper(ticker)

	f.cacheMu.RLock()
	id, found := f.idCache[upper]
	f.cacheMu.RUnlock()
	if found {
		return id, nil
	}

	var resp CoinGeckoSearchResponse
	if err := f.gecko.getJSON(ctx, fmt.Sprintf("%s/search?query=%s", f.coingeckoURL, url.QueryEscape(ticker)), &resp); err != nil {
		return "", err
	}

	// Search results are ordered by market-cap rank, so the first exact match is the major coin.
	for _, c := range resp.Coins {
		if strings.EqualFold(c.Symbol, ticker) {
			f.cacheMu.Lock()
			f.idCache[upper] = c.ID
			f.cacheMu.Unlock()
			return c.ID, nil
		}
	}
	return "", fmt.Errorf("CoinGecko: no coin with symbol %s", ticker)
}
