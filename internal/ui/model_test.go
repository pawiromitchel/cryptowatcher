package ui

import (
	"testing"

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

	// Remove highlighted crypto pair
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = updated.(Model)

	if len(m.cryptoPairs) != initialCryptoLen-1 {
		t.Errorf("expected %d crypto pairs after deletion, got %d", initialCryptoLen-1, len(m.cryptoPairs))
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
		{500.50, "500.50"},
		{10687.52, "10.69K"},
		{106915.89, "106.92K"},
		{1880137.64, "1.88M"},
		{261156235.80, "261.16M"},
		{1500000000.00, "1.50B"},
	}

	for _, tt := range tests {
		got := formatVolume(tt.input)
		if got != tt.want {
			t.Errorf("formatVolume(%f) = %q; want %q", tt.input, got, tt.want)
		}
	}
}

func containsSubstring(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || (len(s) > len(sub) && stringSearch(s, sub)))
}

func stringSearch(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
