package ui

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"cryptowatcher/internal/model"
)

// renderRangeBar draws the current price's position between the 24h low and high.
func renderRangeBar(current, low, high float64, width int) string {
	if high <= low || width < 3 {
		return ""
	}
	ratio := math.Max(0, math.Min(1, (current-low)/(high-low)))
	filled := int(math.Round(ratio * float64(width)))
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	if ratio >= 0.5 {
		return positiveStyle.Render(bar)
	}
	return negativeStyle.Render(bar)
}

// renderDetail draws the full-screen detail panel for one asset.
func renderDetail(a model.Asset, width int) string {
	inner := width - 8 // border + padding
	if inner > 72 {
		inner = 72
	}
	if inner < 30 {
		inner = 30
	}
	bullish := a.Change24h >= 0

	arrow := positiveStyle.Render("▲")
	if !bullish {
		arrow = negativeStyle.Render("▼")
	}
	head := fmt.Sprintf("%s %s  %s", arrow, widgetSymbolStyle.Render(a.Display), widgetNameStyle.Render(a.Name))

	price := widgetPriceStyle.Render(formatPrice(a.Price))
	if !a.HasQuote() {
		price = mutedStyle.Render("no quote yet")
	}
	priceLine := price
	if a.HasQuote() {
		priceLine += "  " + formatChange(a.Change24h) + labelStyle.Render(" 24h")
	}

	chart := renderChart(a.History, bullish, inner-2, 8, "no chart data from "+orDash(a.Source))

	var rangeLine string
	if bar := renderRangeBar(a.Price, a.Low24h, a.High24h, inner-26); bar != "" {
		rangeLine = fmt.Sprintf("%s %s %s %s", labelStyle.Render("24h range"), formatPrice(a.Low24h), bar, formatPrice(a.High24h))
	}

	row := func(label, value string) string {
		return labelStyle.Render(fmt.Sprintf("%-12s", label)) + value
	}
	updated := dash
	if !a.LastUpdated.IsZero() {
		updated = a.LastUpdated.Format("15:04:05")
	}
	stats := []string{
		row("Open (24h)", formatPrice(a.Open24h)),
		row("Volume", formatVolume(a.Volume24h)),
		row("Market cap", formatCompactUSD(a.MarketCap)),
		row("Source", orDash(a.Source)),
		row("Updated", updated),
	}

	lines := []string{head, "", priceLine, "", chart, ""}
	if rangeLine != "" {
		lines = append(lines, rangeLine, "")
	}
	lines = append(lines, stats...)
	if a.Err != nil {
		lines = append(lines, "", warnStyle.Render(truncate("Last refresh failed: "+a.Err.Error(), inner)))
	}
	lines = append(lines, "", mutedStyle.Render("esc / enter  close"))

	return detailStyle.Width(inner + 4).Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func orDash(s string) string {
	if s == "" {
		return dash
	}
	return s
}
