package fetcher

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNormalizeSymbol(t *testing.T) {
	tests := []struct {
		input       string
		wantSymbol  string
		wantDisplay string
	}{
		{"btc", "BTC-USD", "BTC/USD"},
		{"BTC/USD", "BTC-USD", "BTC/USD"},
		{"eth_usd", "ETH-USD", "ETH/USD"},
		{"SOL-USD", "SOL-USD", "SOL/USD"},
		{"doge", "DOGE-USD", "DOGE/USD"},
		{"  ada/eur  ", "ADA-EUR", "ADA/EUR"},
		{"aapl", "AAPL", "AAPL"},
		{"tsla", "TSLA", "TSLA"},
		{"spy", "SPY", "S&P 500"},
		{"hmm", "HMM-USD", "HMM/USD"},
		{"", "", ""},
	}

	for _, tt := range tests {
		gotSymbol, gotDisplay := NormalizeSymbol(tt.input)
		if gotSymbol != tt.wantSymbol || gotDisplay != tt.wantDisplay {
			t.Errorf("NormalizeSymbol(%q) = (%q, %q); want (%q, %q)",
				tt.input, gotSymbol, gotDisplay, tt.wantSymbol, tt.wantDisplay)
		}
	}
}

func TestCoinbaseFetcher_FetchPair(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/products/BTC-USD/stats" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"open": "90000.00",
				"high": "95000.00",
				"low": "89000.00",
				"last": "94500.00",
				"volume": "15000.5"
			}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	fetcher := NewCoinbaseFetcher()
	fetcher.SetBaseURL(server.URL)

	ctx := context.Background()
	pair, err := fetcher.FetchPair(ctx, "BTC/USD")
	if err != nil {
		t.Fatalf("unexpected fetch error: %v", err)
	}

	if pair.Symbol != "BTC-USD" {
		t.Errorf("expected symbol BTC-USD, got %s", pair.Symbol)
	}
	if pair.Price != 94500.00 {
		t.Errorf("expected price 94500.00, got %f", pair.Price)
	}

	wantChange := 5.0
	if pair.Change24h != wantChange {
		t.Errorf("expected change 24h %f, got %f", wantChange, pair.Change24h)
	}
}

func TestCoinbaseFetcher_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	fetcher := NewCoinbaseFetcher()
	fetcher.SetBaseURL(server.URL)

	ctx := context.Background()
	_, err := fetcher.FetchPair(ctx, "INVALID-PAIR")
	if err == nil {
		t.Fatalf("expected error for non-existent pair, got nil")
	}
}

func TestCoinGeckoFetcher_FetchPair(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/coins/thinking-cat" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "thinking-cat",
				"symbol": "hmm",
				"name": "Thinking Cat",
				"market_data": {
					"current_price": {"usd": 0.029},
					"market_cap": {"usd": 29000000},
					"total_volume": {"usd": 4000000},
					"high_24h": {"usd": 0.033},
					"low_24h": {"usd": 0.015},
					"price_change_percentage_24h": 16.38,
					"sparkline_7d": {
						"price": [0.012, 0.015, 0.020, 0.025, 0.029]
					}
				}
			}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	geckoFetcher := NewCoinGeckoFetcher()
	geckoFetcher.SetBaseURL(server.URL)

	ctx := context.Background()
	pair, err := geckoFetcher.FetchPair(ctx, "HMM")
	if err != nil {
		t.Fatalf("unexpected gecko fetch error: %v", err)
	}

	if pair.Symbol != "HMM-USD" {
		t.Errorf("expected symbol HMM-USD, got %s", pair.Symbol)
	}
	if pair.Name != "Thinking Cat" {
		t.Errorf("expected name Thinking Cat, got %s", pair.Name)
	}
	if pair.Price != 0.029 {
		t.Errorf("expected price 0.029, got %f", pair.Price)
	}
	if pair.MarketCap != "$29.0M" {
		t.Errorf("expected market cap $29.0M, got %s", pair.MarketCap)
	}
}

func TestPythFetcher_FetchPair(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v8/finance/chart/AAPL" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"chart": {
					"result": [{
						"meta": {
							"symbol": "AAPL",
							"regularMarketPrice": 319.70,
							"regularMarketChangePercent": 1.63,
							"regularMarketDayHigh": 322.37,
							"regularMarketDayLow": 315.45,
							"regularMarketVolume": 38500185,
							"chartPreviousClose": 309.35,
							"shortName": "Apple Inc."
						},
						"indicators": {
							"quote": [{
								"close": [310.34, 309.90, 313.45, 314.58, 319.70]
							}]
						}
					}]
				}
			}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	fetcher := NewPythFetcher()
	fetcher.SetBaseURL(server.URL)

	ctx := context.Background()
	pair, err := fetcher.FetchPair(ctx, "AAPL")
	if err != nil {
		t.Fatalf("unexpected equity fetch error: %v", err)
	}

	if pair.Symbol != "AAPL" {
		t.Errorf("expected symbol AAPL, got %s", pair.Symbol)
	}
	if pair.Price != 319.70 {
		t.Errorf("expected price 319.70, got %f", pair.Price)
	}
	if len(pair.History) == 0 {
		t.Errorf("expected history to be populated")
	}
}

func TestMultiFetcher_CascadingFallback(t *testing.T) {
	cbServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/products/BTC-USD/stats" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"open":"90000","high":"95000","low":"89000","last":"94000","volume":"1000"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer cbServer.Close()

	geckoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/coins/thinking-cat" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "thinking-cat",
				"symbol": "hmm",
				"name": "Thinking Cat",
				"market_data": {
					"current_price": {"usd": 0.029},
					"market_cap": {"usd": 29000000},
					"total_volume": {"usd": 4000000},
					"high_24h": {"usd": 0.033},
					"low_24h": {"usd": 0.015},
					"price_change_percentage_24h": 16.38,
					"sparkline_7d": {"price": [0.012, 0.029]}
				}
			}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer geckoServer.Close()

	equityServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v8/finance/chart/AAPL" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"chart": {
					"result": [{
						"meta": {
							"symbol": "AAPL",
							"regularMarketPrice": 319.70,
							"regularMarketChangePercent": 1.63,
							"regularMarketDayHigh": 322.37,
							"regularMarketDayLow": 315.45,
							"regularMarketVolume": 38500185,
							"chartPreviousClose": 309.35,
							"shortName": "Apple Inc."
						},
						"indicators": {
							"quote": [{
								"close": [310.34, 309.90, 313.45, 314.58, 319.70]
							}]
						}
					}]
				}
			}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer equityServer.Close()

	cbFetcher := NewCoinbaseFetcher()
	cbFetcher.SetBaseURL(cbServer.URL)

	geckoFetcher := NewCoinGeckoFetcher()
	geckoFetcher.SetBaseURL(geckoServer.URL)

	equityFetcher := NewPythFetcher()
	equityFetcher.SetBaseURL(equityServer.URL)

	multi := NewMultiFetcher(cbFetcher, geckoFetcher, equityFetcher)
	ctx := context.Background()

	// 1. BTC should resolve via Coinbase
	btc, err := multi.FetchPair(ctx, "BTC-USD")
	if err != nil {
		t.Fatalf("failed to fetch BTC via multi fetcher: %v", err)
	}
	if btc.Price != 94000 {
		t.Errorf("expected BTC price 94000, got %f", btc.Price)
	}

	// 2. HMM should cascade to CoinGecko
	hmm, err := multi.FetchPair(ctx, "HMM")
	if err != nil {
		t.Fatalf("failed to fetch HMM via multi fetcher: %v", err)
	}
	if hmm.Price != 0.029 {
		t.Errorf("expected HMM price 0.029, got %f", hmm.Price)
	}

	// 3. AAPL should cascade to Equity fetcher
	aapl, err := multi.FetchPair(ctx, "AAPL")
	if err != nil {
		t.Fatalf("failed to fetch AAPL via multi fetcher: %v", err)
	}
	if aapl.Price != 319.70 {
		t.Errorf("expected AAPL price 319.70, got %f", aapl.Price)
	}
}
