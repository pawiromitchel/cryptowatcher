package ui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"cryptowatcher/internal/fetcher"
	"cryptowatcher/internal/model"
)

func TestUIModelNavigation(t *testing.T) {
	cfg := model.DefaultConfig()
	mockFetcher := fetcher.NewMockFetcher()
	m := NewModel(cfg, mockFetcher)

	if m.cryptoCursor != 0 || m.sectionIndex != 0 {
		t.Errorf("expected initial crypto cursor at 0 and section 0")
	}

	// Move right in crypto row
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = updated.(Model)
	if m.cryptoCursor != 1 {
		t.Errorf("expected crypto cursor at 1 after 'l', got %d", m.cryptoCursor)
	}

	// Move left in crypto row
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m = updated.(Model)
	if m.cryptoCursor != 0 {
		t.Errorf("expected crypto cursor at 0 after 'h', got %d", m.cryptoCursor)
	}

	// Move down across sections into Stocks
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(Model)
	if m.sectionIndex != 1 {
		t.Errorf("expected sectionIndex to be 1 (Stocks) after moving down, got %d", m.sectionIndex)
	}

	// Move up across sections into Crypto
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = updated.(Model)
	if m.sectionIndex != 0 {
		t.Errorf("expected sectionIndex to be 0 (Crypto) after moving up, got %d", m.sectionIndex)
	}
}

func TestUIModelModeSwitching(t *testing.T) {
	cfg := model.DefaultConfig()
	mockFetcher := fetcher.NewMockFetcher()
	m := NewModel(cfg, mockFetcher)

	if m.mode != modeNormal {
		t.Fatalf("expected initial modeNormal")
	}

	// Press 'a' to enter add mode
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = updated.(Model)
	if m.mode != modeAdd {
		t.Errorf("expected modeAdd after pressing 'a'")
	}

	// Press 'esc' to cancel back to normal mode
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	if m.mode != modeNormal {
		t.Errorf("expected modeNormal after pressing 'esc'")
	}
}

func TestUIModelRemovePair(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	cfg := model.DefaultConfig()
	mockFetcher := fetcher.NewMockFetcher()
	m := NewModel(cfg, mockFetcher)

	initialCryptoLen := len(m.cryptoPairs)
	if initialCryptoLen != 3 {
		t.Fatalf("expected 3 initial crypto pairs, got %d", initialCryptoLen)
	}

	// 1. Press 'd' to open delete confirmation modal
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = updated.(Model)

	if m.mode != modeDeleteConfirm {
		t.Fatalf("expected modeDeleteConfirm after pressing 'd', got %v", m.mode)
	}

	// 2. Press 'n' to cancel
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = updated.(Model)

	if m.mode != modeNormal {
		t.Fatalf("expected modeNormal after cancelling, got %v", m.mode)
	}
	if len(m.cryptoPairs) != initialCryptoLen {
		t.Fatalf("expected crypto pairs length to remain %d, got %d", initialCryptoLen, len(m.cryptoPairs))
	}

	// 3. Press 'd' then 'y' to confirm removal
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = updated.(Model)

	if m.mode != modeNormal {
		t.Fatalf("expected modeNormal after confirming removal, got %v", m.mode)
	}
	if len(m.cryptoPairs) != initialCryptoLen-1 {
		t.Errorf("expected %d crypto pairs after confirmed deletion, got %d", initialCryptoLen-1, len(m.cryptoPairs))
	}
}

func TestRenderWidgetCard(t *testing.T) {
	pair := model.CryptoPair{
		Symbol:    "BTC-USD",
		Display:   "BTC-USD",
		Name:      "Bitcoin USD",
		Price:     78402.00,
		MarketCap: "$1.57T",
		Change24h: 1.05,
		History:   []float64{77500, 77800, 78100, 78402},
	}

	card := RenderWidgetCard(pair, true, 30)
	if card == "" {
		t.Fatalf("rendered widget card is empty")
	}

	if !containsSubstring(card, "BTC-USD") {
		t.Errorf("expected card to contain BTC-USD")
	}
	if !containsSubstring(card, "Bitcoin USD") {
		t.Errorf("expected card to contain Bitcoin USD")
	}
	if !containsSubstring(card, "1.57T") {
		t.Errorf("expected card to contain 1.57T")
	}
}

func TestUIModelViewRendering(t *testing.T) {
	cfg := model.DefaultConfig()
	mockFetcher := fetcher.NewMockFetcher()
	m := NewModel(cfg, mockFetcher)

	output := m.View()
	if output == "" {
		t.Fatalf("rendered view is empty")
	}

	requiredStrings := []string{"CRYPTOCURRENCY", "STOCKS & EQUITIES"}
	for _, req := range requiredStrings {
		if !containsSubstring(output, req) {
			t.Errorf("rendered view missing expected string %q", req)
		}
	}
}

func TestFormatVolume(t *testing.T) {
	tests := []struct {
		input float64
		want  string
	}{
		{500.50, "$500.50"},
		{10687.52, "$10.69K"},
		{106915.89, "$106.92K"},
		{1880137.64, "$1.88M"},
		{261156235.80, "$261.16M"},
		{1500000000.00, "$1.50B"},
	}

	for _, tt := range tests {
		got := formatVolume(tt.input)
		if got != tt.want {
			t.Errorf("formatVolume(%f) = %q; want %q", tt.input, got, tt.want)
		}
	}
}

func TestUIModelInspectorMode(t *testing.T) {
	cfg := model.DefaultConfig()
	mockFetcher := fetcher.NewMockFetcher()
	m := NewModel(cfg, mockFetcher)

	// 1. Press Enter to open inspector mode
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if m.mode != modeInspector {
		t.Fatalf("expected modeInspector after pressing Enter, got %v", m.mode)
	}

	// 2. Press Right to switch timeframe
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = updated.(Model)
	if m.inspectorTimeframe != 1 {
		t.Errorf("expected inspector timeframe 1 (1W), got %d", m.inspectorTimeframe)
	}

	// 3. Press Left to switch timeframe back
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	m = updated.(Model)
	if m.inspectorTimeframe != 0 {
		t.Errorf("expected inspector timeframe 0 (1D), got %d", m.inspectorTimeframe)
	}

	// 4. Press Esc to return to normal dashboard mode
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	if m.mode != modeNormal {
		t.Fatalf("expected modeNormal after pressing Esc, got %v", m.mode)
	}
}

func TestRenderInspectorView(t *testing.T) {
	pair := model.CryptoPair{
		Symbol:      "BTC-USD",
		Display:     "BTC/USD",
		Name:        "Bitcoin USD",
		Price:       78500.00,
		Open24h:     77000.00,
		High24h:     79000.00,
		Low24h:      76500.00,
		Volume24h:   1500000000.00,
		MarketCap:   "$1.57T",
		Change24h:   1.95,
		History:     []float64{77000, 77500, 78000, 78500},
		LastUpdated: time.Now(),
	}

	view := RenderInspectorView(pair, 0, 100)
	if view == "" {
		t.Fatalf("rendered inspector view is empty")
	}

	expectedSubstrings := []string{
		"BTC/USD",
		"Bitcoin USD",
		"Market Cap: $1.57T",
		"PRICE ACTION & VOLATILITY",
		"VALUATION & LIQUIDITY",
		"24h Open:",
		"24h High:",
		"24h Low:",
	}

	for _, str := range expectedSubstrings {
		if !containsSubstring(view, str) {
			t.Errorf("inspector view missing expected substring %q", str)
		}
	}
}

func TestRenderRangeBar(t *testing.T) {
	bar := RenderRangeBar(78000, 76000, 80000, 20)
	if bar == "" {
		t.Fatalf("range bar is empty")
	}
	if !strings.HasPrefix(bar, "[") || !strings.HasSuffix(bar, "]") {
		t.Errorf("expected range bar to start with [ and end with ], got %s", bar)
	}
}

func containsSubstring(s, sub string) bool {
	return strings.Contains(s, sub)
}
