package ui

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"cryptowatcher/internal/config"
	"cryptowatcher/internal/fetcher"
	"cryptowatcher/internal/model"
)

func newTestModel(t *testing.T) Model {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := NewModel(model.DefaultConfig(), fetcher.NewMockFetcher())
	m.width, m.height = 140, 60
	return m
}

func press(m Model, keys ...string) Model {
	for _, k := range keys {
		var msg tea.KeyMsg
		switch k {
		case "enter":
			msg = tea.KeyMsg{Type: tea.KeyEnter}
		case "esc":
			msg = tea.KeyMsg{Type: tea.KeyEsc}
		default:
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
		}
		next, _ := m.Update(msg)
		m = next.(Model)
	}
	return m
}

func send(m Model, msg tea.Msg) (Model, tea.Cmd) {
	next, cmd := m.Update(msg)
	return next.(Model), cmd
}

func ok(symbol string, price float64) model.Asset {
	return model.Asset{Symbol: symbol, Display: symbol, Name: symbol, Price: price, Change24h: 1, History: []float64{1, 2, 3}}
}

func TestNavigationAcrossSections(t *testing.T) {
	m := newTestModel(t) // 140 cols → 4 cards per row; 3 crypto, 4 stocks
	if m.cardsPerRow() != 4 {
		t.Fatalf("test assumes 4 cards per row, got %d", m.cardsPerRow())
	}

	m = press(m, "l", "l")
	if m.sections[secCrypto].cursor != 2 {
		t.Fatalf("cursor = %d, want 2", m.sections[secCrypto].cursor)
	}
	m = press(m, "l") // clamp at the last card
	if m.sections[secCrypto].cursor != 2 {
		t.Errorf("cursor must clamp at the end")
	}

	m = press(m, "j") // crypto has a single row → crosses into stocks, same column
	if m.active != secStock || m.sections[secStock].cursor != 2 {
		t.Errorf("want stocks col 2, got section %d cursor %d", m.active, m.sections[secStock].cursor)
	}
	m = press(m, "k")
	if m.active != secCrypto || m.sections[secCrypto].cursor != 2 {
		t.Errorf("want crypto col 2, got section %d cursor %d", m.active, m.sections[secCrypto].cursor)
	}
	m = press(m, "h", "h", "h")
	if m.sections[secCrypto].cursor != 0 {
		t.Errorf("cursor must clamp at 0")
	}
}

func TestNavigationMultiRowGrid(t *testing.T) {
	m := newTestModel(t)
	m.width = 80 // 2 cards per row
	if m.cardsPerRow() != 2 {
		t.Fatalf("cardsPerRow = %d", m.cardsPerRow())
	}
	// stocks: 4 items → 2 rows. Go to stocks, then down within the grid, then back up across sections.
	m = press(m, "j") // crypto: 3 items = rows [0,1],[2] ; cursor 0 → row 1
	if m.active != secCrypto || m.sections[secCrypto].cursor != 2 {
		t.Fatalf("expected crypto cursor 2, got %d/%d", m.active, m.sections[secCrypto].cursor)
	}
	m = press(m, "j") // bottom of crypto → stocks col 0
	m = press(m, "j") // stocks row 1
	if m.active != secStock || m.sections[secStock].cursor != 2 {
		t.Fatalf("expected stocks cursor 2, got %d/%d", m.active, m.sections[secStock].cursor)
	}
	m = press(m, "k", "k") // back to stocks row 0, then up into last crypto row
	if m.active != secCrypto || m.sections[secCrypto].cursor != 2 {
		t.Errorf("up from the stock grid should land on the last crypto row, got %d/%d", m.active, m.sections[secCrypto].cursor)
	}
}

func TestAddFlow(t *testing.T) {
	m := newTestModel(t)

	m = press(m, "a")
	if m.mode != modeAdd {
		t.Fatal("expected add mode")
	}

	// Invalid input is reported and keeps the prompt open.
	m.textInput.SetValue("bad ticker!")
	m, cmd := send(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.mode != modeAdd || cmd != nil || !m.statusErr {
		t.Errorf("invalid input should keep the prompt open with an error (mode=%v err=%v)", m.mode, m.statusErr)
	}

	m.textInput.SetValue("doge")
	m, cmd = send(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.mode != modeNormal || cmd == nil {
		t.Fatal("valid input should close the prompt and start a lookup")
	}
	msg := cmd()
	added, ok := msg.(addedMsg)
	if !ok || added.err != nil {
		t.Fatalf("lookup failed: %#v", msg)
	}

	m, _ = send(m, added)
	crypto := m.sections[secCrypto]
	if len(crypto.items) != 4 || crypto.items[3].Symbol != "DOGE-USD" || crypto.cursor != 3 {
		t.Fatalf("DOGE not appended and selected: %+v", symbols(crypto.items))
	}
	saved, err := config.Load()
	if err != nil || len(saved.CryptoPairs) != 4 {
		t.Errorf("config not persisted: %+v %v", saved, err)
	}

	// Adding it again selects the existing card instead of duplicating.
	m, _ = send(m, added)
	if len(m.sections[secCrypto].items) != 4 || !strings.Contains(m.statusMsg, "Already watching") {
		t.Errorf("duplicate add: status=%q n=%d", m.statusMsg, len(m.sections[secCrypto].items))
	}
}

func TestAddRoutesStocksToStockSection(t *testing.T) {
	m := newTestModel(t)
	m, _ = send(m, addedMsg{raw: "NVDA", asset: model.Asset{Symbol: "NVDA", Display: "NVDA", Name: "NVIDIA", Type: model.AssetStock, Price: 100}})
	if m.active != secStock || len(m.sections[secStock].items) != 5 {
		t.Errorf("stock should land in the stock section: active=%d n=%d", m.active, len(m.sections[secStock].items))
	}
}

func TestAddErrorShowsStatus(t *testing.T) {
	m := newTestModel(t)
	m, _ = send(m, addedMsg{raw: "zzz", err: errors.New("not found")})
	if !m.statusErr || !strings.Contains(m.statusMsg, "ZZZ") || len(m.sections[secCrypto].items) != 3 {
		t.Errorf("status=%q", m.statusMsg)
	}
}

func TestRemoveFlow(t *testing.T) {
	m := newTestModel(t)

	m = press(m, "d")
	if m.mode != modeDeleteConfirm || m.pendingDelete != "BTC-USD" {
		t.Fatalf("mode=%v pending=%q", m.mode, m.pendingDelete)
	}
	m = press(m, "n")
	if m.mode != modeNormal || len(m.sections[secCrypto].items) != 3 {
		t.Fatal("cancel must not remove")
	}

	m = press(m, "d", "y")
	if len(m.sections[secCrypto].items) != 2 || m.sections[secCrypto].items[0].Symbol != "ETH-USD" {
		t.Errorf("remaining: %v", symbols(m.sections[secCrypto].items))
	}
	saved, _ := config.Load()
	if len(saved.CryptoPairs) != 2 {
		t.Errorf("deletion not persisted: %v", saved.CryptoPairs)
	}
}

func TestRemoveLastItemMovesFocus(t *testing.T) {
	m := newTestModel(t)
	for i := 0; i < 3; i++ {
		m = press(m, "d", "y")
	}
	if len(m.sections[secCrypto].items) != 0 || m.active != secStock {
		t.Errorf("focus should move to the non-empty section, active=%d", m.active)
	}
	if m.ActiveAsset() == nil {
		t.Error("an asset should be selected")
	}
	if !strings.Contains(m.View(), "press 'a' to add") {
		t.Error("empty section should show a hint")
	}
}

func TestEmptyDashboardIsSafe(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := NewModel(&model.Config{CryptoPairs: []string{}, StockPairs: []string{}, RefreshInterval: 15}, fetcher.NewMockFetcher())
	m.width, m.height = 100, 40
	m = press(m, "j", "k", "h", "l", "d", "s", "[", "]", "enter")
	if m.ActiveAsset() != nil || m.mode != modeNormal {
		t.Error("nothing should be selectable")
	}
	_ = m.View()
}

func TestRefreshMergesBySymbol(t *testing.T) {
	m := newTestModel(t)
	m, _ = send(m, refreshedMsg{batches: []refreshBatch{{
		section: secCrypto,
		symbols: []string{"BTC-USD", "ETH-USD", "SOL-USD"},
		assets:  []model.Asset{ok("BTC-USD", 100), ok("ETH-USD", 200), ok("SOL-USD", 300)},
	}}})
	if m.sections[secCrypto].items[1].Price != 200 || m.refreshing {
		t.Fatalf("merge failed: %+v", m.sections[secCrypto].items)
	}

	// A failed refresh keeps the last good quote and flags it stale.
	m, _ = send(m, refreshedMsg{batches: []refreshBatch{{
		section: secCrypto,
		symbols: []string{"BTC-USD", "ETH-USD", "SOL-USD"},
		assets:  []model.Asset{{Err: errors.New("429")}, ok("ETH-USD", 210), {Err: errors.New("timeout")}},
	}}})
	items := m.sections[secCrypto].items
	if items[0].Price != 100 || !items[0].Stale || items[0].Err == nil {
		t.Errorf("BTC should keep its price and become stale: %+v", items[0])
	}
	if items[1].Price != 210 || items[1].Stale {
		t.Errorf("ETH should update: %+v", items[1])
	}

	// Recovery clears the stale flag.
	m, _ = send(m, refreshedMsg{batches: []refreshBatch{{
		section: secCrypto, symbols: []string{"BTC-USD"}, assets: []model.Asset{ok("BTC-USD", 105)},
	}}})
	if it := m.sections[secCrypto].items[0]; it.Stale || it.Err != nil || it.Price != 105 {
		t.Errorf("recovery should clear stale: %+v", it)
	}
}

func TestRefreshNeverReviveRemovedOrDropAddedItems(t *testing.T) {
	m := newTestModel(t)
	// A refresh starts with [BTC, ETH, SOL] in flight...
	inFlight := []string{"BTC-USD", "ETH-USD", "SOL-USD"}
	// ...meanwhile the user deletes BTC and adds DOGE.
	m = press(m, "d", "y")
	m, _ = send(m, addedMsg{raw: "DOGE", asset: model.Asset{Symbol: "DOGE-USD", Display: "DOGE/USD", Type: model.AssetCrypto, Price: 1}})

	m, _ = send(m, refreshedMsg{batches: []refreshBatch{{
		section: secCrypto, symbols: inFlight,
		assets: []model.Asset{ok("BTC-USD", 1), ok("ETH-USD", 2), ok("SOL-USD", 3)},
	}}})
	got := symbols(m.sections[secCrypto].items)
	want := []string{"ETH-USD", "SOL-USD", "DOGE-USD"}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestOverlappingRefreshesAreSkipped(t *testing.T) {
	m := newTestModel(t) // starts with refreshing = true (initial fetch in flight)
	_, cmd := send(m, tickMsg{})
	if cmd == nil {
		t.Fatal("tick must always reschedule itself")
	}
	m = press(m, "r")
	if !strings.Contains(m.statusMsg, "already in progress") {
		t.Errorf("manual refresh during a refresh should be a no-op, status=%q", m.statusMsg)
	}

	m.refreshing = false
	m, cmd = send(m, tickMsg{})
	if !m.refreshing || cmd == nil {
		t.Error("an idle tick should start a refresh")
	}
}

func TestReorderAndSortPersist(t *testing.T) {
	m := newTestModel(t)
	m = press(m, "]")
	if got := symbols(m.sections[secCrypto].items); got[0] != "ETH-USD" || got[1] != "BTC-USD" || m.sections[secCrypto].cursor != 1 {
		t.Errorf("move later: %v", got)
	}
	m = press(m, "[", "[") // second press is a no-op at the edge
	if m.sections[secCrypto].cursor != 0 || symbols(m.sections[secCrypto].items)[0] != "BTC-USD" {
		t.Errorf("move earlier failed")
	}
	saved, _ := config.Load()
	if saved.CryptoPairs[0] != "BTC-USD" {
		t.Errorf("order not persisted: %v", saved.CryptoPairs)
	}

	items := m.sections[secCrypto].items
	items[0].Change24h, items[1].Change24h, items[2].Change24h = -1, 5, 2
	m = press(m, "s")
	if got := symbols(m.sections[secCrypto].items); fmt.Sprint(got) != "[ETH-USD SOL-USD BTC-USD]" {
		t.Errorf("sort by change: %v", got)
	}
	if a := m.ActiveAsset(); a.Symbol != "BTC-USD" {
		t.Errorf("selection should follow the asset, on %s", a.Symbol)
	}
}

func TestDetailMode(t *testing.T) {
	m := newTestModel(t)
	m.sections[secCrypto].items[0] = model.Asset{Symbol: "BTC-USD", Display: "BTC/USD", Name: "Bitcoin", Price: 78402, Open24h: 77584,
		High24h: 78500, Low24h: 77557, Volume24h: 24510, MarketCap: 1.57e12, Source: "Coinbase", Change24h: 1.05, History: []float64{1, 3, 2, 5}}
	m = press(m, "enter")
	if m.mode != modeDetail {
		t.Fatal("enter should open details")
	}
	out := m.View()
	for _, want := range []string{"BTC/USD", "Bitcoin", "$78,402.00", "$1.57T", "Coinbase", "24h range", "24.51K"} {
		if !strings.Contains(out, want) {
			t.Errorf("detail view missing %q:\n%s", want, out)
		}
	}
	m = press(m, "esc")
	if m.mode != modeNormal {
		t.Error("esc should close details")
	}
	m = press(m, "enter", "q") // q closes details rather than quitting
	if m.mode != modeNormal {
		t.Error("q should close details")
	}
}

func TestQuitKeys(t *testing.T) {
	m := newTestModel(t)
	for _, k := range []tea.KeyMsg{{Type: tea.KeyRunes, Runes: []rune{'q'}}, {Type: tea.KeyCtrlC}} {
		_, cmd := m.Update(k)
		if cmd == nil || fmt.Sprintf("%T", cmd()) != "tea.QuitMsg" {
			t.Errorf("%v should quit", k)
		}
	}
}

func TestViewAcrossTerminalSizes(t *testing.T) {
	m := newTestModel(t)
	for _, size := range [][2]int{{40, 14}, {60, 20}, {80, 24}, {120, 40}, {200, 60}} {
		m.width, m.height = size[0], size[1]
		out := m.View()
		for i, line := range strings.Split(out, "\n") {
			if w := lipglossWidth(line); w > size[0] {
				t.Errorf("%dx%d: line %d is %d cells wide: %q", size[0], size[1], i, w, line)
			}
		}
		if h := strings.Count(out, "\n") + 1; h > size[1] {
			t.Errorf("%dx%d: output is %d lines tall", size[0], size[1], h)
		}
	}
	m.width, m.height = 30, 10
	if !strings.Contains(m.View(), "Terminal too small") {
		t.Error("tiny terminals should get a friendly message")
	}
}

func TestSelectedCardStaysVisibleWhenScrolling(t *testing.T) {
	m := newTestModel(t)
	m.width, m.height = 80, 20 // far too short for both sections
	for _, it := range m.sections[secStock].items {
		_ = it
	}
	m = press(m, "j") // → stocks (width 80 → 2 per row; crypto has 2 rows, so go all the way)
	m = press(m, "j", "j")
	sel := m.ActiveAsset().Display
	if !strings.Contains(m.View(), sel) {
		t.Errorf("selected card %q scrolled out of view:\n%s", sel, m.View())
	}
	m = press(m, "k", "k", "k", "k")
	sel = m.ActiveAsset().Display
	if !strings.Contains(m.View(), sel) {
		t.Errorf("selected card %q scrolled out of view going up", sel)
	}
}

func TestStatusIndicator(t *testing.T) {
	m := newTestModel(t)
	if got := m.View(); !strings.Contains(got, "LOADING") {
		t.Error("expected LOADING before the first refresh")
	}
	all := func(f func(*model.Asset)) {
		for s := range m.sections {
			for i := range m.sections[s].items {
				f(&m.sections[s].items[i])
			}
		}
	}
	m.refreshing = false
	m.lastRefresh = m.lastRefresh.Add(1)
	all(func(a *model.Asset) { a.Price = 1 })
	if !strings.Contains(m.View(), "LIVE") {
		t.Error("expected LIVE")
	}
	m.sections[0].items[0].Err = errors.New("x")
	if !strings.Contains(m.View(), "PARTIAL 6/7") {
		t.Error("expected PARTIAL 6/7")
	}
	all(func(a *model.Asset) { a.Err = errors.New("x") })
	if !strings.Contains(m.View(), "OFFLINE") {
		t.Error("expected OFFLINE")
	}
}

func TestStockPlaceholdersKeepUnlistedTickers(t *testing.T) {
	items := placeholders([]string{"AMD", "spy", "AAPL"}, model.AssetStock)
	got := []string{items[0].Symbol, items[1].Symbol, items[2].Symbol}
	if fmt.Sprint(got) != "[AMD SPY AAPL]" {
		t.Errorf("stock symbols must not get a -USD suffix: %v", got)
	}
	if items[1].Display != "S&P 500" {
		t.Errorf("display = %q", items[1].Display)
	}
}
