package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"cryptowatcher/internal/model"
)

// View renders the terminal user interface string.
func (m Model) View() string {
	var b strings.Builder

	// Header Banner
	b.WriteString(titleStyle.Render(" CRYPTOWATCHER - REAL-TIME DASHBOARD "))
	b.WriteString("\n\n")

	// Top Summary Cards Dashboard
	b.WriteString(m.renderSummaryCards())
	b.WriteString("\n\n")

	// Split View: Main Table (Left) + Detailed Inspector Panel (Right)
	tableStr := m.renderTable()
	tableBox := tableBoxStyle.Render(tableStr)

	panelStr := m.renderInspectorPanel()
	panelBox := panelBoxStyle.Render(panelStr)

	mainContent := lipgloss.JoinHorizontal(lipgloss.Top, tableBox, "  ", panelBox)
	b.WriteString(mainContent)
	b.WriteString("\n")

	// 7-Day Multi-Coin Correlation Line Chart Box
	chartContent := m.renderChartSection()
	if chartContent != "" {
		chartBox := chartBoxStyle.Render(chartContent)
		b.WriteString(chartBox)
		b.WriteString("\n")
	}

	// Modal Overlay for Adding Pairs
	if m.mode == modeAdd {
		b.WriteString("\n")
		modalContent := fmt.Sprintf(
			"Enter Cryptocurrency Pair:\n\n%s\n\n(Press Enter to submit, Esc to cancel)",
			m.textInput.View(),
		)
		b.WriteString(modalStyle.Render(modalContent))
		b.WriteString("\n")
	}

	// Status Bar & Controls Footer
	b.WriteString(m.renderFooter())

	return b.String()
}

func (m Model) renderSummaryCards() string {
	if len(m.pairs) == 0 {
		return ""
	}

	totalPairs := len(m.pairs)
	bestPair := m.pairs[0]
	worstPair := m.pairs[0]

	for _, p := range m.pairs {
		if p.Change24h > bestPair.Change24h {
			bestPair = p
		}
		if p.Change24h < worstPair.Change24h {
			worstPair = p
		}
	}

	c1 := summaryCardStyle.Render(fmt.Sprintf("PAIRS\n%s", summaryValueStyle.Render(fmt.Sprintf("%d Monitored", totalPairs))))

	bestStr := fmt.Sprintf("%s (%s)", bestPair.Display, formatChange(bestPair.Change24h))
	c2 := summaryCardStyle.Render(fmt.Sprintf("TOP GAINER\n%s", bestStr))

	worstStr := fmt.Sprintf("%s (%s)", worstPair.Display, formatChange(worstPair.Change24h))
	c3 := summaryCardStyle.Render(fmt.Sprintf("TOP LOSER\n%s", worstStr))

	statusText := "🟢 LIVE (Multi-Feed: Coinbase & Pyth)"
	if m.loading {
		statusText = "🟡 UPDATING..."
	} else if m.err != nil {
		statusText = "🔴 ERROR"
	}
	c4 := summaryCardStyle.Render(fmt.Sprintf("STATUS\n%s", summaryValueStyle.Render(statusText)))

	return lipgloss.JoinHorizontal(lipgloss.Top, c1, c2, c3, c4)
}

func (m Model) renderTable() string {
	var sb strings.Builder

	// Column header
	header := fmt.Sprintf(
		"%-3s %-10s %-14s %-12s %-12s %-12s %-12s",
		"", "PAIR", "PRICE (USD)", "24H CHANGE", "24H HIGH", "24H LOW", "24H VOLUME",
	)
	sb.WriteString(headerStyle.Render(header))
	sb.WriteString("\n")

	if len(m.pairs) == 0 {
		sb.WriteString(rowStyle.Render("   No pairs monitored. Press 'a' to add a pair."))
		sb.WriteString("\n")
		return sb.String()
	}

	for i, pair := range m.pairs {
		cursorStr := "  "
		if i == m.cursor {
			cursorStr = "> "
		}

		priceStr := formatPrice(pair.Price)
		changeStr := formatChange(pair.Change24h)
		highStr := formatPrice(pair.High24h)
		lowStr := formatPrice(pair.Low24h)
		volStr := formatVolume(pair.Volume24h)

		if pair.Err != nil {
			priceStr = errorStyle.Render("Error")
			changeStr = "-"
		}

		rowContent := fmt.Sprintf(
			"%s %s %s %s %s %s %s",
			padRight(cursorStr, 3),
			padRight(pair.Display, 10),
			padRight(priceStr, 14),
			padRight(changeStr, 12),
			padRight(highStr, 12),
			padRight(lowStr, 12),
			padRight(volStr, 12),
		)

		if i == m.cursor {
			sb.WriteString(selectedRowStyle.Render(rowContent))
		} else {
			sb.WriteString(rowStyle.Render(rowContent))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func (m Model) renderInspectorPanel() string {
	var sb strings.Builder

	if len(m.pairs) == 0 || m.cursor >= len(m.pairs) {
		sb.WriteString(subHeaderStyle.Render("INSPECTOR"))
		sb.WriteString("\n\nNo pair selected.")
		return sb.String()
	}

	pair := m.pairs[m.cursor]

	sb.WriteString(subHeaderStyle.Render(fmt.Sprintf("🔍 %s MARKET INSPECTOR", pair.Display)))
	sb.WriteString("\n\n")

	// Price & 24h Badge
	priceStr := formatPrice(pair.Price)
	changeStr := formatChange(pair.Change24h)
	sb.WriteString(fmt.Sprintf("Spot Price:  %s   (%s)\n\n", summaryValueStyle.Render(priceStr), changeStr))

	// 24H Price Range Bar Slider
	sb.WriteString("24H Price Range:\n")
	lowStr := fmt.Sprintf("Low: %s", formatPrice(pair.Low24h))
	highStr := fmt.Sprintf("High: %s", formatPrice(pair.High24h))
	panelInnerWidth := 36
	spaces := panelInnerWidth - lipgloss.Width(lowStr) - lipgloss.Width(highStr)
	if spaces < 1 {
		spaces = 1
	}
	sb.WriteString(fmt.Sprintf("%s%s%s\n", lowStr, strings.Repeat(" ", spaces), highStr))
	rangeBar := RenderRangeBar(pair.Price, pair.Low24h, pair.High24h, panelInnerWidth-2)
	sb.WriteString(fmt.Sprintf("%s\n\n", rangeBar))

	// 24H Sparkline Trend
	sb.WriteString("24H Trend Sparkline:\n")
	sparkline := RenderSparkline(pair.History, pair.Change24h >= 0)
	sb.WriteString(fmt.Sprintf("  %s\n\n", sparkline))

	// Detailed Metrics Breakdown
	spread := pair.High24h - pair.Low24h
	spreadPct := 0.0
	if pair.Low24h > 0 {
		spreadPct = (spread / pair.Low24h) * 100.0
	}

	sb.WriteString(fmt.Sprintf("%-16s %s\n", "24h Open:", formatPrice(pair.Open24h)))
	sb.WriteString(fmt.Sprintf("%-16s %s\n", "24h High:", formatPrice(pair.High24h)))
	sb.WriteString(fmt.Sprintf("%-16s %s\n", "24h Low:", formatPrice(pair.Low24h)))
	sb.WriteString(fmt.Sprintf("%-16s %s (%.2f%%)\n", "24h Spread:", formatPrice(spread), spreadPct))
	sb.WriteString(fmt.Sprintf("%-16s %s\n", "24h Volume:", formatVolume(pair.Volume24h)))

	updatedTime := "Never"
	if !pair.LastUpdated.IsZero() {
		updatedTime = pair.LastUpdated.Format("15:04:05")
	}
	sb.WriteString(fmt.Sprintf("%-16s %s\n", "Last Sync:", updatedTime))

	return sb.String()
}

func (m Model) renderChartSection() string {
	var sb strings.Builder
	sb.WriteString(subHeaderStyle.Render("📈 7-DAY RELATIVE PERFORMANCE & CORRELATION CHART (NORMALIZED %)"))
	sb.WriteString("\n\n")

	totalWidth := m.width
	if totalWidth <= 0 {
		totalWidth = 110
	}
	chartWidth := totalWidth - 6
	if chartWidth < 60 {
		chartWidth = 60
	}

	chartStr := RenderMultiLineChart(m.pairs, chartWidth)
	sb.WriteString(chartStr)

	return sb.String()
}

func (m Model) renderFooter() string {
	var sb strings.Builder

	if m.statusMsg != "" {
		sb.WriteString(neutralStyle.Render(m.statusMsg))
		sb.WriteString("\n")
	}

	lastUpdatedStr := "Never"
	if !m.lastRefresh.IsZero() {
		lastUpdatedStr = m.lastRefresh.Format("15:04:05")
	}

	controls := fmt.Sprintf(
		"[a] Add Pair | [d] Remove | [r] Refresh | [↑/↓] Navigate | [q] Quit  (Last Updated: %s)",
		lastUpdatedStr,
	)
	sb.WriteString(statusBarStyle.Render(controls))
	sb.WriteString("\n")

	return sb.String()
}

func padRight(str string, width int) string {
	w := lipgloss.Width(str)
	if w >= width {
		return str
	}
	return str + strings.Repeat(" ", width-w)
}

func formatPrice(price float64) string {
	if price == 0 {
		return "$0.00"
	}
	if price < 0.01 {
		return fmt.Sprintf("$%.6f", price)
	}
	if price < 1.0 {
		return fmt.Sprintf("$%.4f", price)
	}
	return fmt.Sprintf("$%.2f", price)
}

func formatChange(change float64) string {
	if change > 0 {
		return positiveStyle.Render(fmt.Sprintf("+%.2f%%", change))
	} else if change < 0 {
		return negativeStyle.Render(fmt.Sprintf("%.2f%%", change))
	}
	return neutralStyle.Render("0.00%")
}

func formatVolume(vol float64) string {
	if vol >= 1_000_000_000 {
		return fmt.Sprintf("%.2fB", vol/1_000_000_000)
	}
	if vol >= 1_000_000 {
		return fmt.Sprintf("%.2fM", vol/1_000_000)
	}
	if vol >= 1_000 {
		return fmt.Sprintf("%.2fK", vol/1_000)
	}
	return fmt.Sprintf("%.2f", vol)
}

// Ensure time import is referenced properly
var _ = time.Now
var _ = model.CryptoPair{}
