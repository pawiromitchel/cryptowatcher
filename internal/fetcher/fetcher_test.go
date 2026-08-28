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

	// 24h change: (94500 - 90000) / 90000 * 100 = 5.0%
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
