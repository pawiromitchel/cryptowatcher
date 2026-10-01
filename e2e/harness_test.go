// Package e2e drives the complete application — real UI, real routing
// fetcher, real config file — against in-process fake provider servers.
package e2e

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"cryptowatcher/internal/config"
	"cryptowatcher/internal/fakeapi"
	"cryptowatcher/internal/fetcher"
	"cryptowatcher/internal/model"
	"cryptowatcher/internal/ui"
)

// screen records the most recent frame rendered by the program.
type screen struct {
	mu   sync.Mutex
	last string
}

func (s *screen) set(v string) { s.mu.Lock(); s.last = v; s.mu.Unlock() }
func (s *screen) get() string  { s.mu.Lock(); defer s.mu.Unlock(); return s.last }

// probe wraps the real model and captures every frame it renders.
type probe struct {
	inner tea.Model
	scr   *screen
}

func (p probe) Init() tea.Cmd { return p.inner.Init() }
func (p probe) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	next, cmd := p.inner.Update(msg)
	return probe{inner: next, scr: p.scr}, cmd
}
func (p probe) View() string {
	v := p.inner.View()
	p.scr.set(v)
	return v
}

// app is a running instance of the dashboard.
type app struct {
	t    *testing.T
	prog *tea.Program
	scr  *screen
	done chan error
}

// env is a fake-provider world plus an isolated config directory.
type env struct {
	t      *testing.T
	api    *fakeapi.API
	cfgDir string
}

func newEnv(t *testing.T) *env {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	api := fakeapi.New(t)
	api.SetCoinbase("BTC-USD", fakeapi.Quote{Price: 94500, Open: 90000, High: 95000, Low: 89000, Volume: 15000})
	api.SetCoinbase("ETH-USD", fakeapi.Quote{Price: 2500, Open: 2400, High: 2600, Low: 2300, Volume: 90000})
	api.SetCoinbase("SOL-USD", fakeapi.Quote{Price: 180, Open: 190, High: 195, Low: 175, Volume: 70000})
	for sym, id := range map[string]string{"BTC": "bitcoin", "ETH": "ethereum", "SOL": "solana"} {
		api.SetGecko(sym, fakeapi.Quote{ID: id, Name: sym, Price: 1, MarketCap: map[string]float64{"BTC": 1.87e12, "ETH": 300e9, "SOL": 80e9}[sym]})
	}
	api.SetYahoo("SPY", fakeapi.Quote{Name: "SPDR S&P 500 ETF Trust", Price: 571.2, Open: 560, High: 572, Low: 559, Volume: 5e7, ChangePct: 2})
	api.SetYahoo("TSLA", fakeapi.Quote{Name: "Tesla, Inc.", Price: 215.3, Open: 208, High: 216, Low: 207, Volume: 9e7, ChangePct: 3.5})
	api.SetYahoo("GOOGL", fakeapi.Quote{Name: "Alphabet Inc.", Price: 168.4, Open: 164, High: 169, Low: 163, Volume: 2e7, ChangePct: 2.7})
	api.SetYahoo("AAPL", fakeapi.Quote{Name: "Apple Inc.", Price: 225.5, Open: 220, High: 226, Low: 219, Volume: 4e7, ChangePct: 2.5})
	return &env{t: t, api: api, cfgDir: dir}
}

func (e *env) configPath() string {
	return filepath.Join(e.cfgDir, "cryptowatcher", "config.json")
}

func (e *env) readConfig() *model.Config {
	e.t.Helper()
	cfg, err := config.Load()
	if err != nil {
		e.t.Fatalf("loading config: %v", err)
	}
	return cfg
}

func (e *env) fetcher() fetcher.PriceFetcher {
	cb := fetcher.NewCoinbaseFetcher()
	cb.SetBaseURL(e.api.CoinbaseURL)
	cg := fetcher.NewCoinGeckoFetcher()
	cg.SetBaseURL(e.api.GeckoURL)
	cg.SetDexScreenerURL(e.api.DexURL)
	y := fetcher.NewYahooFetcher()
	y.SetBaseURL(e.api.YahooURL)
	return fetcher.NewMultiFetcher(cb, cg, y)
}

// start launches the app exactly as main does (config from disk, real fetchers).
func (e *env) start() *app {
	e.t.Helper()
	cfg, err := config.Load()
	if err != nil {
		e.t.Fatalf("loading config: %v", err)
	}
	cfg.RefreshInterval = model.MinRefreshInterval

	a := &app{t: e.t, scr: &screen{}, done: make(chan error, 1)}
	a.prog = tea.NewProgram(probe{inner: ui.NewModel(cfg, e.fetcher()), scr: a.scr},
		tea.WithInput(nil), tea.WithOutput(io.Discard), tea.WithoutSignals())
	go func() { _, err := a.prog.Run(); a.done <- err }()
	a.prog.Send(tea.WindowSizeMsg{Width: 140, Height: 50})
	e.t.Cleanup(func() { a.prog.Kill() })
	return a
}

func (a *app) key(keys ...string) {
	for _, k := range keys {
		switch k {
		case "enter":
			a.prog.Send(tea.KeyMsg{Type: tea.KeyEnter})
		case "esc":
			a.prog.Send(tea.KeyMsg{Type: tea.KeyEsc})
		default:
			a.prog.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)})
		}
	}
}

func (a *app) typeText(s string) {
	for _, r := range s {
		a.prog.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
}

func (a *app) view() string { return a.scr.get() }

// waitFor polls the rendered screen until it contains every want string.
func (a *app) waitFor(wants ...string) {
	a.t.Helper()
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		v := a.view()
		ok := true
		for _, w := range wants {
			if !strings.Contains(v, w) {
				ok = false
				break
			}
		}
		if ok {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	a.t.Fatalf("timed out waiting for %q on screen:\n%s", wants, a.view())
}

// waitGone polls until none of the strings appear on screen.
func (a *app) waitGone(unwanted string) {
	a.t.Helper()
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		if !strings.Contains(a.view(), unwanted) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	a.t.Fatalf("%q never disappeared from the screen:\n%s", unwanted, a.view())
}

func (a *app) quit() {
	a.t.Helper()
	a.key("q")
	select {
	case err := <-a.done:
		if err != nil {
			a.t.Fatalf("program exited with error: %v", err)
		}
	case <-time.After(5 * time.Second):
		a.t.Fatal("program did not quit")
	}
}

func fileExists(p string) bool { _, err := os.Stat(p); return err == nil }
