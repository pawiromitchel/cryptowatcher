package ui

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"

	"cryptowatcher/internal/model"
)

func lipglossWidth(s string) int { return lipgloss.Width(s) }

func btc() model.Asset {
	return model.Asset{Symbol: "BTC-USD", Display: "BTC/USD", Name: "Bitcoin", Price: 78402, MarketCap: 1.57e12,
		Change24h: 1.05, History: []float64{77500, 77800, 78100, 78402}}
}

func TestCardContent(t *testing.T) {
	card := RenderWidgetCard(btc(), true, 30)
	for _, want := range []string{"BTC/USD", "Bitcoin", "$1.57T", "+1.05%", "$78,402.00", "▲"} {
		if !strings.Contains(card, want) {
			t.Errorf("card missing %q:\n%s", want, card)
		}
	}
}

func TestCardStates(t *testing.T) {
	loading := RenderWidgetCard(model.Asset{Display: "ETH/USD", Name: "Ethereum"}, false, 30)
	if !strings.Contains(loading, "loading") {
		t.Error("expected loading state")
	}

	failed := RenderWidgetCard(model.Asset{Display: "ETH/USD", Name: "Ethereum", Err: errors.New("x")}, false, 30)
	if !strings.Contains(failed, "unavailable") {
		t.Error("expected unavailable state")
	}

	stale := btc()
	stale.Stale, stale.Err, stale.LastUpdated = true, errors.New("429"), time.Date(2026, 1, 1, 14, 3, 0, 0, time.UTC)
	card := RenderWidgetCard(stale, false, 30)
	if !strings.Contains(card, "stale 14:03") || !strings.Contains(card, "$78,402.00") {
		t.Errorf("stale card should keep the last price and flag it:\n%s", card)
	}

	noChart := btc()
	noChart.History = nil
	if !strings.Contains(RenderWidgetCard(noChart, false, 30), "no chart data") {
		t.Error("missing history must show a placeholder, not a fabricated chart")
	}

	noCap := btc()
	noCap.MarketCap = 0
	if !strings.Contains(RenderWidgetCard(noCap, false, 30), "—") {
		t.Error("unknown market cap should render as a dash")
	}
}

func TestCardWidthsAreConsistent(t *testing.T) {
	long := btc()
	long.Name = "A Really Long Asset Name That Overflows"
	long.Display = "SUPERLONGSYMBOL/USD"
	for _, w := range []int{24, 30, 44} {
		for _, a := range []model.Asset{btc(), long, {Display: "X"}} {
			lines := strings.Split(RenderWidgetCard(a, false, w), "\n")
			first := lipgloss.Width(lines[0])
			for i, l := range lines {
				if lipgloss.Width(l) != first {
					t.Errorf("width %d: line %d is %d cells, header is %d", w, i, lipgloss.Width(l), first)
				}
			}
		}
	}
}

func TestChartShape(t *testing.T) {
	out := renderChart([]float64{1, 2, 3, 4, 5, 6}, true, 20, 3, "")
	lines := strings.Split(out, "\n")
	if len(lines) != 3 {
		t.Fatalf("want 3 rows, got %d", len(lines))
	}
	if !strings.ContainsAny(out, "⠁⠂⠄⡀⠈⠐⠠⢀⣀⣿⠉⠒⠤⣤") && !containsBraille(out) {
		t.Error("expected braille output")
	}
	flat := renderChart([]float64{5, 5, 5, 5}, true, 10, 3, "")
	if !containsBraille(flat) {
		t.Error("a flat series should still draw a line")
	}
}

func containsBraille(s string) bool {
	for _, r := range s {
		if r >= 0x2800 && r <= 0x28FF {
			return true
		}
	}
	return false
}

func TestFormatters(t *testing.T) {
	prices := map[float64]string{0: "—", 0.000123: "$0.000123", 0.0000042: "$0.00000420", 0.0000000042: "$0.0000000042", 0.5: "$0.5000", 1: "$1.00", 999.5: "$999.50", 1234.5: "$1,234.50", 78402: "$78,402.00", 1234567.891: "$1,234,567.89"}
	for in, want := range prices {
		if got := formatPrice(in); got != want {
			t.Errorf("formatPrice(%v) = %q, want %q", in, got, want)
		}
	}
	caps := map[float64]string{0: "—", 500: "$500", 12_400: "$12.4K", 29e6: "$29.0M", 297.3e9: "$297.3B", 1.57e12: "$1.57T"}
	for in, want := range caps {
		if got := formatCompactUSD(in); got != want {
			t.Errorf("formatCompactUSD(%v) = %q, want %q", in, got, want)
		}
	}
	vols := map[float64]string{0: "—", 500.5: "500.50", 10687.52: "10.69K", 1880137.64: "1.88M", 1.5e9: "1.50B"}
	for in, want := range vols {
		if got := formatVolume(in); got != want {
			t.Errorf("formatVolume(%v) = %q, want %q", in, got, want)
		}
	}
	if got := truncate("日本語のとても長い名前", 8); lipgloss.Width(got) > 8 {
		t.Errorf("truncate must respect cell width, got %q (%d)", got, lipgloss.Width(got))
	}
}

func TestRangeBar(t *testing.T) {
	if renderRangeBar(5, 10, 10, 20) != "" {
		t.Error("degenerate range should render nothing")
	}
	if got := renderRangeBar(100, 0, 10, 10); !strings.Contains(got, "██████████") {
		t.Errorf("out-of-range values should clamp: %q", got)
	}
}
