package e2e

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"cryptowatcher/internal/fakeapi"
)

func TestStartupShowsLivePricesFromAllProviders(t *testing.T) {
	e := newEnv(t)
	a := e.start()

	// Crypto via Coinbase (+ market cap enrichment from CoinGecko), stocks via Yahoo.
	a.waitFor("BTC/USD", "$94,500.00", "$1.87T", "+5.00%", "$571.20", "S&P 500", "$225.50", "LIVE")
	v := a.view()
	for _, want := range []string{"CRYPTOCURRENCY", "STOCKS & EQUITIES", "Bitcoin", "7 assets", "Tesla, Inc."} {
		if !strings.Contains(v, want) {
			t.Errorf("screen missing %q", want)
		}
	}
	if !fileExists(e.configPath()) {
		t.Error("first run should write the default config")
	}
	a.quit()
}

func TestAddCryptoFromCoinGeckoAndPersist(t *testing.T) {
	e := newEnv(t)
	e.api.SetGecko("HMM", fakeapi.Quote{ID: "thinking-cat", Name: "Thinking Cat", Price: 0.029, MarketCap: 29e6, ChangePct: 16.38})
	a := e.start()
	a.waitFor("LIVE")

	a.key("a")
	a.waitFor("Add ticker")
	a.typeText("hmm")
	a.key("enter")

	a.waitFor("Added HMM/USD", "via CoinGecko", "$0.0290", "$29.0M", "Thinking Cat")
	if got := e.readConfig().CryptoPairs; !reflect.DeepEqual(got, []string{"BTC-USD", "ETH-USD", "SOL-USD", "HMM-USD"}) {
		t.Errorf("config crypto = %v", got)
	}
	a.quit()
}

func TestAddUnlistedStockIsResolvedAsStockNotCrypto(t *testing.T) {
	e := newEnv(t)
	e.api.SetYahoo("AMD", fakeapi.Quote{Name: "Advanced Micro Devices, Inc.", Price: 160.25, Open: 158, ChangePct: 1.4})
	// A lookalike on-chain token must never win over the real stock.
	e.api.SetDex("AMD", fakeapi.Quote{Name: "AMD Meme", Price: 0.0002, Liquidity: 9e6})
	a := e.start()
	a.waitFor("LIVE")

	a.key("a")
	a.typeText("amd")
	a.key("enter")
	a.waitFor("Added AMD", "Yahoo Finance", "$160.25", "Advanced Micro")

	cfg := e.readConfig()
	if cfg.StockPairs[len(cfg.StockPairs)-1] != "AMD" || len(cfg.CryptoPairs) != 3 {
		t.Errorf("AMD should be saved as a stock: %+v", cfg)
	}

	// After a restart the unlisted stock must still load as a stock.
	a.quit()
	b := e.start()
	b.waitFor("$160.25")
	b.quit()
}

func TestUnknownAndInvalidTickersAreRejected(t *testing.T) {
	e := newEnv(t)
	a := e.start()
	a.waitFor("LIVE")

	a.key("a")
	a.typeText("zzzz")
	a.key("enter")
	a.waitFor("Could not add ZZZZ")

	a.key("a")
	a.typeText("bad one!")
	a.key("enter")
	a.waitFor("not a valid ticker")
	a.key("esc")

	if got := e.readConfig(); len(got.CryptoPairs) != 3 || len(got.StockPairs) != 4 {
		t.Errorf("failed adds must not change the watchlist: %+v", got)
	}
	a.quit()
}

func TestProviderOutageKeepsLastPriceAndRecovers(t *testing.T) {
	e := newEnv(t)
	a := e.start()
	a.waitFor("$94,500.00", "$571.20", "LIVE")

	// Everything goes down. Prices must stay on screen, flagged stale.
	e.api.Fail(fakeapi.Coinbase, 500)
	e.api.Fail(fakeapi.Gecko, 500)
	e.api.Fail(fakeapi.Dex, 500)
	e.api.Fail(fakeapi.Yahoo, 503)
	a.key("r")
	a.waitFor("OFFLINE", "stale")
	if v := a.view(); !strings.Contains(v, "$94,500.00") || !strings.Contains(v, "$571.20") {
		t.Errorf("last good prices must remain visible during an outage:\n%s", v)
	}

	// Partial outage: only stocks down.
	e.api.Heal(fakeapi.Coinbase)
	e.api.Heal(fakeapi.Gecko)
	e.api.SetCoinbase("BTC-USD", fakeapi.Quote{Price: 95000, Open: 90000, High: 96000, Low: 89000})
	a.key("r")
	a.waitFor("PARTIAL 3/7", "$95,000.00")

	// Full recovery.
	e.api.Heal(fakeapi.Yahoo)
	a.key("r")
	a.waitFor("LIVE")
	a.waitGone("stale")
	a.quit()
}

func TestRateLimitBacksOffInsteadOfHammering(t *testing.T) {
	e := newEnv(t)
	a := e.start()
	a.waitFor("LIVE")

	before := e.api.Calls(fakeapi.Yahoo)
	e.api.RateLimit(fakeapi.Yahoo, 300)
	a.key("r")
	a.waitFor("stale")
	for i := 0; i < 3; i++ { // further refreshes must be served by the cooldown, not the network
		a.key("r")
		time.Sleep(150 * time.Millisecond)
	}
	// 4 stocks → at most one 429 per symbol on the first limited refresh, then none.
	if extra := e.api.Calls(fakeapi.Yahoo) - before; extra > 4 {
		t.Errorf("client kept calling a rate-limited provider: %d extra requests", extra)
	}
	a.quit()
}

func TestRemoveReorderSortAndRestartPreserveWatchlist(t *testing.T) {
	e := newEnv(t)
	a := e.start()
	a.waitFor("LIVE")

	// Remove BTC (first card is selected).
	a.key("d")
	a.waitFor("Remove BTC/USD")
	a.key("y")
	a.waitFor("Removed BTC/USD")
	a.waitGone("Bitcoin")

	// Move ETH after SOL.
	a.key("]")
	// Sort stocks by 24h change: TSLA(3.5) > GOOGL(2.7) > AAPL(2.5) > SPY(2).
	a.key("j", "s")
	a.waitFor("sorted by 24h change")

	cfg := e.readConfig()
	if !reflect.DeepEqual(cfg.CryptoPairs, []string{"SOL-USD", "ETH-USD"}) {
		t.Errorf("crypto order = %v", cfg.CryptoPairs)
	}
	if !reflect.DeepEqual(cfg.StockPairs, []string{"TSLA", "GOOGL", "AAPL", "SPY"}) {
		t.Errorf("stock order = %v", cfg.StockPairs)
	}
	a.quit()

	// A fresh process sees exactly the saved state.
	b := e.start()
	b.waitFor("6 assets", "LIVE")
	if strings.Contains(b.view(), "Bitcoin") {
		t.Error("removed asset came back after restart")
	}
	b.quit()
}

func TestDetailViewShowsRealStats(t *testing.T) {
	e := newEnv(t)
	a := e.start()
	a.waitFor("$94,500.00", "$1.87T")

	a.key("enter")
	a.waitFor("24h range", "$89,000.00", "$95,000.00", "Volume", "15.00K", "Market cap", "Coinbase", "Updated")
	a.key("esc")
	a.waitFor("CRYPTOCURRENCY")
	a.quit()
}
