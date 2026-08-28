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

	if m.cursor != 0 {
		t.Errorf("expected initial cursor at 0, got %d", m.cursor)
	}

	// Move down
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(Model)
	if m.cursor != 1 {
		t.Errorf("expected cursor at 1 after 'j', got %d", m.cursor)
	}

	// Move down again
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(Model)
	if m.cursor != 2 {
		t.Errorf("expected cursor at 2 after 'j', got %d", m.cursor)
	}

	// Move up
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = updated.(Model)
	if m.cursor != 1 {
		t.Errorf("expected cursor at 1 after 'k', got %d", m.cursor)
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

	initialLen := len(m.pairs)
	if initialLen != 3 {
		t.Fatalf("expected 3 initial pairs, got %d", initialLen)
	}

	// Remove highlighted pair
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = updated.(Model)

	if len(m.pairs) != initialLen-1 {
		t.Errorf("expected %d pairs after deletion, got %d", initialLen-1, len(m.pairs))
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

	if !testing.Short() {
		// Check that header elements appear
		requiredStrings := []string{"CRYPTOWATCHER", "PAIR", "PRICE (USD)", "24H CHANGE"}
		for _, req := range requiredStrings {
			if !containsSubstring(output, req) {
				t.Errorf("rendered view missing expected string %q", req)
			}
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

func TestSparklineAndRangeBar(t *testing.T) {
	sparkline := RenderSparkline([]float64{10, 15, 20, 25, 30}, true)
	if sparkline == "" {
		t.Errorf("expected non-empty sparkline string")
	}

	rangeBar := RenderRangeBar(150, 100, 200, 10)
	if rangeBar == "" {
		t.Errorf("expected non-empty range bar string")
	}
}

func TestRenderMultiLineChart(t *testing.T) {
	cfg := model.DefaultConfig()
	mockFetcher := fetcher.NewMockFetcher()
	m := NewModel(cfg, mockFetcher)

	chartStr := RenderMultiLineChart(m.pairs, 80)
	if chartStr == "" {
		t.Fatalf("expected non-empty correlation chart string")
	}

	if !containsSubstring(chartStr, "Legend:") {
		t.Errorf("expected legend in chart output")
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
