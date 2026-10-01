package fetcher

import (
	"context"
	"testing"

	"cryptowatcher/internal/fakeapi"
	"cryptowatcher/internal/model"
)

func newMulti(t *testing.T) (*MultiFetcher, *fakeapi.API) {
	t.Helper()
	api := fakeapi.New(t)
	cb := NewCoinbaseFetcher()
	cb.SetBaseURL(api.CoinbaseURL)
	cg := NewCoinGeckoFetcher()
	cg.SetBaseURL(api.GeckoURL)
	cg.SetDexScreenerURL(api.DexURL)
	y := NewYahooFetcher()
	y.SetBaseURL(api.YahooURL)
	return NewMultiFetcher(cb, cg, y), api
}

func TestMulti_Routing(t *testing.T) {
	m, api := newMulti(t)
	api.SetCoinbase("BTC-USD", btcQuote())
	api.SetGecko("BTC", fakeapi.Quote{ID: "bitcoin", Name: "Bitcoin", Price: 1, MarketCap: 1.8e12})
	api.SetGecko("HMM", fakeapi.Quote{ID: "thinking-cat", Name: "Thinking Cat", Price: 0.029, MarketCap: 29e6, ChangePct: 16})
	api.SetYahoo("AAPL", fakeapi.Quote{Name: "Apple Inc.", Price: 319.7, Open: 309.35, ChangePct: 1.63})
	ctx := context.Background()

	btc, err := m.FetchPair(ctx, "BTC-USD")
	if err != nil || btc.Source != "Coinbase" {
		t.Fatalf("BTC should resolve via Coinbase: %+v %v", btc, err)
	}
	if btc.MarketCap != 1.8e12 {
		t.Errorf("Coinbase quotes should be enriched with a real market cap, got %v", btc.MarketCap)
	}

	hmm, err := m.FetchPair(ctx, "HMM")
	if err != nil || hmm.Source != "CoinGecko" {
		t.Fatalf("HMM should cascade to CoinGecko: %+v %v", hmm, err)
	}

	cbBefore := api.Calls(fakeapi.Coinbase)
	aapl, err := m.FetchPair(ctx, "AAPL")
	if err != nil || aapl.Type != model.AssetStock {
		t.Fatalf("AAPL should resolve via Yahoo: %+v %v", aapl, err)
	}
	if api.Calls(fakeapi.Coinbase) != cbBefore {
		t.Error("known stocks must not probe Coinbase")
	}
}

// An unlisted ticker such as AMD must be found as a stock, not mistaken for a
// similarly named on-chain token.
func TestMulti_UnknownTickerResolvesToStockBeforeCryptoLongTail(t *testing.T) {
	m, api := newMulti(t)
	api.SetYahoo("AMD", fakeapi.Quote{Name: "Advanced Micro Devices", Price: 160, Open: 158, ChangePct: 1.2})
	api.SetDex("AMD", fakeapi.Quote{Name: "Random Meme Token", Price: 0.0001, Liquidity: 5_000_000})

	a, err := m.FetchPair(context.Background(), "amd")
	if err != nil {
		t.Fatal(err)
	}
	if a.Type != model.AssetStock || a.Name != "Advanced Micro Devices" || a.Symbol != "AMD" {
		t.Fatalf("AMD must resolve to the stock, got %+v", a)
	}
}

func TestMulti_ResolutionIsRemembered(t *testing.T) {
	m, api := newMulti(t)
	api.SetYahoo("AMD", fakeapi.Quote{Name: "AMD", Price: 160, Open: 158})
	ctx := context.Background()
	if _, err := m.FetchPair(ctx, "AMD"); err != nil {
		t.Fatal(err)
	}
	before := api.Calls(fakeapi.Coinbase)
	if _, err := m.FetchPair(ctx, "AMD"); err != nil {
		t.Fatal(err)
	}
	if api.Calls(fakeapi.Coinbase) != before {
		t.Error("second lookup of a resolved stock must skip the exchange")
	}
}

func TestMulti_KindHintSkipsProbing(t *testing.T) {
	m, api := newMulti(t)
	api.SetYahoo("AMD", fakeapi.Quote{Name: "AMD", Price: 160, Open: 158})
	if _, err := m.FetchPair(WithKind(context.Background(), model.AssetStock), "AMD"); err != nil {
		t.Fatal(err)
	}
	if api.Calls(fakeapi.Coinbase) != 0 {
		t.Error("a stock hint must not call the exchange")
	}
}

func TestMulti_NotFoundAnywhere(t *testing.T) {
	m, _ := newMulti(t)
	a, err := m.FetchPair(context.Background(), "NOSUCHTHING")
	if err == nil || a.Err == nil {
		t.Fatal("expected an error")
	}
	if a.Symbol == "" {
		t.Error("failed lookups should still carry a symbol")
	}
}

func TestMulti_InvalidInputNeverHitsNetwork(t *testing.T) {
	m, api := newMulti(t)
	if _, err := m.FetchPair(context.Background(), "a b?c"); err == nil {
		t.Fatal("expected validation error")
	}
	if api.Calls(fakeapi.Coinbase)+api.Calls(fakeapi.Gecko)+api.Calls(fakeapi.Yahoo) != 0 {
		t.Error("invalid input must be rejected before any request")
	}
}

func TestMulti_FetchPricesPreservesOrderAndReportsPerSymbolErrors(t *testing.T) {
	m, api := newMulti(t)
	api.SetCoinbase("BTC-USD", btcQuote())
	api.SetCoinbase("ETH-USD", fakeapi.Quote{Price: 2500, Open: 2400, High: 2600, Low: 2300})
	api.SetGecko("BTC", fakeapi.Quote{ID: "bitcoin", Name: "Bitcoin", Price: 1})

	res, err := m.FetchPrices(context.Background(), []string{"ETH-USD", "NOSUCH-USD", "BTC-USD"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 3 || res[0].Symbol != "ETH-USD" || res[2].Symbol != "BTC-USD" {
		t.Fatalf("order not preserved: %+v", res)
	}
	if res[1].Err == nil || res[0].Err != nil || res[2].Err != nil {
		t.Errorf("only the unknown symbol should fail: %v / %v / %v", res[0].Err, res[1].Err, res[2].Err)
	}
}
