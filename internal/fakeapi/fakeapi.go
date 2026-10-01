// Package fakeapi provides in-process HTTP servers that imitate the Coinbase,
// CoinGecko, DexScreener and Yahoo Finance endpoints CryptoWatcher consumes.
// It exists for tests: every response is deterministic and failures
// (HTTP errors, rate limits) can be injected per provider.
package fakeapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// Provider names used by Fail, RateLimit and Calls.
const (
	Coinbase = "coinbase"
	Gecko    = "coingecko"
	Dex      = "dexscreener"
	Yahoo    = "yahoo"
)

// Quote is the data served for one asset.
type Quote struct {
	Price, Open, High, Low, Volume float64
	Name                           string
	MarketCap                      float64
	ChangePct                      float64 // used by CoinGecko / DexScreener / Yahoo
	ID                             string  // CoinGecko coin id
	Chain                          string  // DexScreener chain
	Liquidity                      float64 // DexScreener liquidity
}

// API is a set of fake provider servers sharing mutable state.
type API struct {
	CoinbaseURL, GeckoURL, DexURL, YahooURL string

	mu         sync.Mutex
	coinbase   map[string]Quote   // product id (BTC-USD) -> quote
	gecko      map[string]Quote   // symbol (HMM) -> quote
	dex        map[string][]Quote // symbol -> pairs
	yahoo      map[string]Quote   // ticker -> quote
	status     map[string]int     // provider -> forced HTTP status
	retryAfter map[string]int
	calls      map[string]int
	noCandles  bool
	servers    []*httptest.Server
}

// New starts the fake servers; they are closed automatically when t finishes.
func New(t testing.TB) *API {
	t.Helper()
	a := &API{
		coinbase:   map[string]Quote{},
		gecko:      map[string]Quote{},
		dex:        map[string][]Quote{},
		yahoo:      map[string]Quote{},
		status:     map[string]int{},
		retryAfter: map[string]int{},
		calls:      map[string]int{},
	}
	start := func(name string, h http.HandlerFunc) string {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			a.mu.Lock()
			a.calls[name]++
			code := a.status[name]
			retry := a.retryAfter[name]
			a.mu.Unlock()
			if code != 0 {
				if retry > 0 {
					w.Header().Set("Retry-After", strconv.Itoa(retry))
				}
				w.WriteHeader(code)
				return
			}
			h(w, r)
		}))
		a.servers = append(a.servers, s)
		t.Cleanup(s.Close)
		return s.URL
	}
	a.CoinbaseURL = start(Coinbase, a.serveCoinbase)
	a.GeckoURL = start(Gecko, a.serveGecko)
	a.DexURL = start(Dex, a.serveDex)
	a.YahooURL = start(Yahoo, a.serveYahoo)
	return a
}

// SetCoinbase lists a product (e.g. "BTC-USD") on the fake exchange.
func (a *API) SetCoinbase(product string, q Quote) { a.set(func() { a.coinbase[product] = q }) }

// SetGecko lists a coin by ticker; q.ID and q.Name should be set.
func (a *API) SetGecko(symbol string, q Quote) {
	a.set(func() { a.gecko[strings.ToUpper(symbol)] = q })
}

// SetDex lists a DEX pair under a token symbol.
func (a *API) SetDex(symbol string, q ...Quote) { a.set(func() { a.dex[strings.ToUpper(symbol)] = q }) }

// SetYahoo lists an equity ticker.
func (a *API) SetYahoo(ticker string, q Quote) {
	a.set(func() { a.yahoo[strings.ToUpper(ticker)] = q })
}

// Fail makes a provider answer every request with the given HTTP status.
func (a *API) Fail(provider string, status int) { a.set(func() { a.status[provider] = status }) }

// RateLimit makes a provider answer 429 with a Retry-After header.
func (a *API) RateLimit(provider string, retryAfterSecs int) {
	a.set(func() { a.status[provider], a.retryAfter[provider] = http.StatusTooManyRequests, retryAfterSecs })
}

// Heal clears an injected failure.
func (a *API) Heal(provider string) {
	a.set(func() { delete(a.status, provider); delete(a.retryAfter, provider) })
}

// NoCandles makes the Coinbase candles endpoint return 500.
func (a *API) NoCandles(v bool) { a.set(func() { a.noCandles = v }) }

// Calls reports how many requests a provider has received.
func (a *API) Calls(provider string) int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.calls[provider]
}

func (a *API) set(f func()) { a.mu.Lock(); f(); a.mu.Unlock() }

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func f2s(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }

func (a *API) serveCoinbase(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/") // products/{id}/{stats|candles}
	if len(parts) != 3 || parts[0] != "products" {
		http.NotFound(w, r)
		return
	}
	a.mu.Lock()
	q, ok := a.coinbase[parts[1]]
	noCandles := a.noCandles
	a.mu.Unlock()
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		writeJSON(w, map[string]string{"message": "NotFound"})
		return
	}
	switch parts[2] {
	case "stats":
		writeJSON(w, map[string]string{
			"open": f2s(q.Open), "high": f2s(q.High), "low": f2s(q.Low),
			"last": f2s(q.Price), "volume": f2s(q.Volume),
		})
	case "candles":
		if noCandles {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		const n = 30
		rows := make([][]float64, n) // newest first
		for i := 0; i < n; i++ {
			frac := float64(n-1-i) / float64(n-1) // newest = 1
			closePx := q.Open + (q.Price-q.Open)*frac
			rows[i] = []float64{float64(1_700_000_000 - i*3600), closePx, closePx, closePx, closePx, 1}
		}
		writeJSON(w, rows)
	default:
		http.NotFound(w, r)
	}
}

func (a *API) serveGecko(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	defer a.mu.Unlock()
	switch {
	case r.URL.Path == "/search":
		term := strings.ToUpper(r.URL.Query().Get("query"))
		coins := []map[string]string{}
		for sym, q := range a.gecko {
			if strings.Contains(sym, term) || strings.Contains(strings.ToUpper(q.Name), term) {
				coins = append(coins, map[string]string{"id": q.ID, "name": q.Name, "symbol": strings.ToLower(sym)})
			}
		}
		writeJSON(w, map[string]any{"coins": coins})
	case strings.HasPrefix(r.URL.Path, "/coins/"):
		id := strings.TrimPrefix(r.URL.Path, "/coins/")
		for sym, q := range a.gecko {
			if q.ID != id {
				continue
			}
			spark := make([]float64, 168)
			start := q.Price / (1 + q.ChangePct/100)
			for i := range spark {
				spark[i] = start + (q.Price-start)*float64(i)/float64(len(spark)-1)
			}
			writeJSON(w, map[string]any{
				"id": id, "symbol": strings.ToLower(sym), "name": q.Name,
				"market_data": map[string]any{
					"current_price":               map[string]float64{"usd": q.Price},
					"market_cap":                  map[string]float64{"usd": q.MarketCap},
					"total_volume":                map[string]float64{"usd": q.Volume},
					"high_24h":                    map[string]float64{"usd": q.High},
					"low_24h":                     map[string]float64{"usd": q.Low},
					"price_change_percentage_24h": q.ChangePct,
					"sparkline_7d":                map[string]any{"price": spark},
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	case r.URL.Path == "/simple/price":
		out := map[string]map[string]float64{}
		for _, id := range strings.Split(r.URL.Query().Get("ids"), ",") {
			for _, q := range a.gecko {
				if q.ID == id {
					out[id] = map[string]float64{"usd": q.Price, "usd_market_cap": q.MarketCap}
				}
			}
		}
		writeJSON(w, out)
	default:
		http.NotFound(w, r)
	}
}

func (a *API) serveDex(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if r.URL.Path != "/search" {
		http.NotFound(w, r)
		return
	}
	term := strings.ToUpper(r.URL.Query().Get("q"))
	pairs := []map[string]any{}
	for sym, qs := range a.dex {
		if !strings.Contains(sym, term) {
			continue
		}
		for _, q := range qs {
			pairs = append(pairs, map[string]any{
				"chainId":     orDefault(q.Chain, "solana"),
				"dexId":       "raydium",
				"baseToken":   map[string]string{"name": q.Name, "symbol": sym},
				"priceUsd":    f2s(q.Price),
				"marketCap":   q.MarketCap,
				"priceChange": map[string]float64{"h24": q.ChangePct},
				"volume":      map[string]float64{"h24": q.Volume},
				"liquidity":   map[string]float64{"usd": q.Liquidity},
			})
		}
	}
	writeJSON(w, map[string]any{"pairs": pairs})
}

func (a *API) serveYahoo(w http.ResponseWriter, r *http.Request) {
	const prefix = "/v8/finance/chart/"
	if !strings.HasPrefix(r.URL.Path, prefix) {
		http.NotFound(w, r)
		return
	}
	ticker := strings.ToUpper(strings.TrimPrefix(r.URL.Path, prefix))
	a.mu.Lock()
	q, ok := a.yahoo[ticker]
	a.mu.Unlock()
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		writeJSON(w, map[string]any{"chart": map[string]any{"result": nil,
			"error": map[string]string{"code": "Not Found", "description": "No data found, symbol may be delisted"}}})
		return
	}
	closes := make([]any, 26) // include nulls like the real API
	for i := range closes {
		if i%9 == 8 {
			closes[i] = nil
			continue
		}
		closes[i] = q.Open + (q.Price-q.Open)*float64(i)/float64(len(closes)-1)
	}
	writeJSON(w, map[string]any{"chart": map[string]any{
		"result": []map[string]any{{
			"meta": map[string]any{
				"instrumentType": "EQUITY", "symbol": ticker,
				"regularMarketPrice":         q.Price,
				"regularMarketChangePercent": q.ChangePct,
				"regularMarketDayHigh":       q.High,
				"regularMarketDayLow":        q.Low,
				"regularMarketVolume":        q.Volume,
				"chartPreviousClose":         q.Open,
				"longName":                   q.Name,
			},
			"indicators": map[string]any{"quote": []map[string]any{{"close": closes}}},
		}},
		"error": nil,
	}})
}

func orDefault(s, d string) string {
	if s == "" {
		return d
	}
	return s
}

// String lists the base URLs, handy in test failure messages.
func (a *API) String() string {
	return fmt.Sprintf("coinbase=%s gecko=%s dex=%s yahoo=%s", a.CoinbaseURL, a.GeckoURL, a.DexURL, a.YahooURL)
}
