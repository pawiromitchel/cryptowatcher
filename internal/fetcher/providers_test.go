package fetcher

import (
	"context"
	"errors"
	"testing"
	"time"

	"cryptowatcher/internal/fakeapi"
	"cryptowatcher/internal/model"
)

func btcQuote() fakeapi.Quote {
	return fakeapi.Quote{Price: 94500, Open: 90000, High: 95000, Low: 89000, Volume: 15000.5}
}

func TestCoinbase_FetchPair(t *testing.T) {
	api := fakeapi.New(t)
	api.SetCoinbase("BTC-USD", btcQuote())
	f := NewCoinbaseFetcher()
	f.SetBaseURL(api.CoinbaseURL)

	a, err := f.FetchPair(context.Background(), "BTC/USD")
	if err != nil {
		t.Fatal(err)
	}
	if a.Symbol != "BTC-USD" || a.Price != 94500 || a.Source != "Coinbase" || a.Name != "Bitcoin" {
		t.Errorf("unexpected asset: %+v", a)
	}
	if a.Change24h != 5.0 {
		t.Errorf("change = %v, want 5", a.Change24h)
	}
	if len(a.History) != 24 || a.History[len(a.History)-1] != 94500 {
		t.Errorf("history should be 24 real candles ending at the spot price, got len %d", len(a.History))
	}
	if a.MarketCap != 0 {
		t.Errorf("Coinbase has no market cap; must not invent one, got %v", a.MarketCap)
	}
}

func TestCoinbase_NoCandlesMeansNoChart_NotFakeChart(t *testing.T) {
	api := fakeapi.New(t)
	api.SetCoinbase("BTC-USD", btcQuote())
	api.NoCandles(true)
	f := NewCoinbaseFetcher()
	f.SetBaseURL(api.CoinbaseURL)

	a, err := f.FetchPair(context.Background(), "BTC")
	if err != nil {
		t.Fatal(err)
	}
	if len(a.History) != 0 {
		t.Errorf("history must be empty when candles are unavailable, got %v", a.History)
	}
}

func TestCoinbase_CandlesAreCached(t *testing.T) {
	api := fakeapi.New(t)
	api.SetCoinbase("BTC-USD", btcQuote())
	f := NewCoinbaseFetcher()
	f.SetBaseURL(api.CoinbaseURL)

	for i := 0; i < 3; i++ {
		if _, err := f.FetchPair(context.Background(), "BTC"); err != nil {
			t.Fatal(err)
		}
	}
	if got := api.Calls(fakeapi.Coinbase); got != 4 { // 3 stats + 1 candles
		t.Errorf("expected 4 Coinbase calls (candles cached), got %d", got)
	}
}

func TestCoinbase_NotFound(t *testing.T) {
	api := fakeapi.New(t)
	f := NewCoinbaseFetcher()
	f.SetBaseURL(api.CoinbaseURL)
	_, err := f.FetchPair(context.Background(), "NOPE-PAIR")
	var se *StatusError
	if !errors.As(err, &se) || se.Code != 404 {
		t.Fatalf("expected 404 StatusError, got %v", err)
	}
}

func TestCoinGecko_FetchPair(t *testing.T) {
	api := fakeapi.New(t)
	api.SetGecko("HMM", fakeapi.Quote{ID: "thinking-cat", Name: "Thinking Cat", Price: 0.029,
		MarketCap: 29_000_000, Volume: 4_000_000, High: 0.033, Low: 0.015, ChangePct: 16.38})
	f := NewCoinGeckoFetcher()
	f.SetBaseURL(api.GeckoURL)

	a, err := f.FetchPair(context.Background(), "HMM")
	if err != nil {
		t.Fatal(err)
	}
	if a.Symbol != "HMM-USD" || a.Name != "Thinking Cat" || a.Price != 0.029 || a.MarketCap != 29_000_000 {
		t.Errorf("unexpected asset: %+v", a)
	}
	if len(a.History) != 24 {
		t.Errorf("history should be the last 24 hourly points, got %d", len(a.History))
	}
}

func TestCoinGecko_QuotesAreCached(t *testing.T) {
	api := fakeapi.New(t)
	api.SetGecko("HMM", fakeapi.Quote{ID: "thinking-cat", Name: "Thinking Cat", Price: 0.029, ChangePct: 1})
	f := NewCoinGeckoFetcher()
	f.SetBaseURL(api.GeckoURL)

	for i := 0; i < 5; i++ {
		if _, err := f.FetchPair(context.Background(), "HMM"); err != nil {
			t.Fatal(err)
		}
	}
	if got := api.Calls(fakeapi.Gecko); got != 1 {
		t.Errorf("expected 1 CoinGecko call within the cache TTL, got %d", got)
	}
}

func TestCoinGecko_ExactSymbolMatchOnly(t *testing.T) {
	api := fakeapi.New(t)
	// Search for "AMD" returns an unrelated coin whose *name* contains it.
	api.SetGecko("AMDX", fakeapi.Quote{ID: "amdx-coin", Name: "AMD Extra Coin", Price: 1})
	f := NewCoinGeckoFetcher()
	f.SetBaseURL(api.GeckoURL)
	f.SetDexScreenerURL(api.DexURL)

	if _, err := f.FetchPair(context.Background(), "AMD"); err == nil {
		t.Fatal("a non-exact symbol match must not resolve to a different asset")
	}
}

func TestCoinGecko_DexScreenerFallbackPicksDeepestLiquidity(t *testing.T) {
	api := fakeapi.New(t)
	api.SetDex("WOOF",
		fakeapi.Quote{Name: "Woof Scam", Price: 9, Liquidity: 1_000, Chain: "bsc"},
		fakeapi.Quote{Name: "Woof", Price: 0.5, Liquidity: 900_000, MarketCap: 5_000_000, ChangePct: 4, Chain: "solana"},
	)
	f := NewCoinGeckoFetcher()
	f.SetBaseURL(api.GeckoURL)
	f.SetDexScreenerURL(api.DexURL)

	a, err := f.FetchPair(context.Background(), "WOOF")
	if err != nil {
		t.Fatal(err)
	}
	if a.Name != "Woof" || a.Price != 0.5 {
		t.Errorf("expected the most liquid pair, got %+v", a)
	}
	if a.High24h != 0 || a.Low24h != 0 || len(a.History) != 0 {
		t.Errorf("DexScreener gives no high/low/history; must not be fabricated: %+v", a)
	}
	if a.Source != "DexScreener (solana)" {
		t.Errorf("source = %q", a.Source)
	}
}

func TestCoinGecko_RateLimitCooldown(t *testing.T) {
	api := fakeapi.New(t)
	api.SetGecko("HMM", fakeapi.Quote{ID: "thinking-cat", Name: "Thinking Cat", Price: 1})
	api.RateLimit(fakeapi.Gecko, 120)
	api.Fail(fakeapi.Dex, 500)
	f := NewCoinGeckoFetcher()
	f.SetBaseURL(api.GeckoURL)
	f.SetDexScreenerURL(api.DexURL)

	for i := 0; i < 4; i++ {
		_, err := f.FetchPair(context.Background(), "HMM")
		if !errors.Is(err, ErrRateLimited) && err == nil {
			t.Fatal("expected an error while rate limited")
		}
	}
	if got := api.Calls(fakeapi.Gecko); got != 1 {
		t.Errorf("after a 429 the client must back off; got %d requests", got)
	}
}

func TestCoinGecko_MarketCapCached(t *testing.T) {
	api := fakeapi.New(t)
	api.SetGecko("BTC", fakeapi.Quote{ID: "bitcoin", Name: "Bitcoin", Price: 1, MarketCap: 1.5e12})
	f := NewCoinGeckoFetcher()
	f.SetBaseURL(api.GeckoURL)

	for i := 0; i < 3; i++ {
		if got := f.MarketCap(context.Background(), "btc"); got != 1.5e12 {
			t.Fatalf("MarketCap = %v", got)
		}
	}
	if got := api.Calls(fakeapi.Gecko); got != 1 {
		t.Errorf("expected 1 call, got %d", got)
	}
}

func TestYahoo_FetchPair(t *testing.T) {
	api := fakeapi.New(t)
	api.SetYahoo("AAPL", fakeapi.Quote{Name: "Apple Inc.", Price: 319.7, Open: 309.35, High: 322.37, Low: 315.45, Volume: 38_500_185, ChangePct: 1.63})
	f := NewYahooFetcher()
	f.SetBaseURL(api.YahooURL)

	a, err := f.FetchPair(context.Background(), "aapl")
	if err != nil {
		t.Fatal(err)
	}
	if a.Symbol != "AAPL" || a.Type != model.AssetStock || a.Name != "Apple Inc." || a.Price != 319.7 || a.Change24h != 1.63 {
		t.Errorf("unexpected asset: %+v", a)
	}
	if len(a.History) == 0 {
		t.Error("expected intraday history with null closes dropped")
	}
	for _, v := range a.History {
		if v <= 0 {
			t.Errorf("history contains non-positive value %v", v)
		}
	}
	if a.MarketCap != 0 {
		t.Errorf("Yahoo chart endpoint has no market cap; must not invent one, got %v", a.MarketCap)
	}
}

func TestYahoo_UnknownSymbol(t *testing.T) {
	api := fakeapi.New(t)
	f := NewYahooFetcher()
	f.SetBaseURL(api.YahooURL)
	if _, err := f.FetchPair(context.Background(), "ZZZZ"); err == nil {
		t.Fatal("expected error for unknown ticker")
	}
}

func TestMockFetcher(t *testing.T) {
	m := NewMockFetcher()
	a, err := m.FetchPair(context.Background(), "tsla")
	if err != nil || a.Type != model.AssetStock || a.Symbol != "TSLA" || !a.HasQuote() {
		t.Fatalf("unexpected: %+v %v", a, err)
	}
	if _, err := m.FetchPair(context.Background(), "bad symbol"); err == nil {
		t.Error("mock should validate input too")
	}
	m.Now = func() time.Time { return time.Unix(0, 0) }
	b1, _ := m.FetchPair(context.Background(), "BTC")
	b2, _ := m.FetchPair(context.Background(), "BTC")
	if b1.Price != b2.Price {
		t.Error("mock must be deterministic for a fixed clock")
	}
}
