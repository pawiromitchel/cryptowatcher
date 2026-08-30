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
		{"aapl", "AAPL-USD", "AAPL/USD"},
		{"aaplc", "AAPLC-USD", "AAPLC/USD"},
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

func TestPythFetcher_FetchPair(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/updates/price/latest" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"parsed": [{
					"id": "49f6b65eb1de5cfdb87dbcf55b6dbdb69db463eec2e6ecb4408ca672edd80894",
					"price": {
						"price": "22550",
						"expo": -2
					},
					"ema_price": {
						"price": "22000",
						"expo": -2
					}
				}]
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
		t.Fatalf("unexpected pyth fetch error: %v", err)
	}

	if pair.Symbol != "AAPL-USD" {
		t.Errorf("expected symbol AAPL-USD, got %s", pair.Symbol)
	}
	if pair.Price != 225.50 {
		t.Errorf("expected price 225.50, got %f", pair.Price)
	}
	if len(pair.History7D) == 0 {
		t.Errorf("expected 7-day history to be populated")
	}
}

func TestMultiFetcher_CascadingFallback(t *testing.T) {
	cbServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/products/BTC-USD/stats" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"open":"90000","high":"95000","low":"89000","last":"94000","volume":"1000"}`))
			return
		}
		// Coinbase returns 404 for AAPL
		w.WriteHeader(http.StatusNotFound)
	}))
	defer cbServer.Close()

	pythServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/updates/price/latest" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"parsed": [{
					"id": "49f6b65eb1de5cfdb87dbcf55b6dbdb69db463eec2e6ecb4408ca672edd80894",
					"price": {"price": "22550", "expo": -2},
					"ema_price": {"price": "22000", "expo": -2}
				}]
			}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer pythServer.Close()

	cbFetcher := NewCoinbaseFetcher()
	cbFetcher.SetBaseURL(cbServer.URL)

	pythFetcher := NewPythFetcher()
	pythFetcher.SetBaseURL(pythServer.URL)

	multi := NewMultiFetcher(cbFetcher, pythFetcher)
	ctx := context.Background()

	// 1. BTC should resolve via Coinbase
	btc, err := multi.FetchPair(ctx, "BTC-USD")
	if err != nil {
		t.Fatalf("failed to fetch BTC via multi fetcher: %v", err)
	}
	if btc.Price != 94000 {
		t.Errorf("expected BTC price 94000, got %f", btc.Price)
	}

	// 2. AAPL should cascade to Pyth
	aapl, err := multi.FetchPair(ctx, "AAPLC")
	if err != nil {
		t.Fatalf("failed to fetch AAPLC via multi fetcher: %v", err)
	}
	if aapl.Price != 225.50 {
		t.Errorf("expected AAPL price 225.50, got %f", aapl.Price)
	}
}
