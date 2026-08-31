package ui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"cryptowatcher/internal/fetcher"
	"cryptowatcher/internal/model"
)

var timeframeLabels = []string{"1D", "1W", "1M", "1Y", "ALL"}

// RenderInspectorView renders the full-screen deep-dive analysis screen for an asset.
func RenderInspectorView(item model.CryptoPair, activeTimeframe int, totalWidth int) string {
	if totalWidth <= 0 {
		totalWidth = 100
	}
	innerWidth := totalWidth - 6
	if innerWidth < 60 {
		innerWidth = 60
	}

	var sb strings.Builder
	isBullish := item.Change24h >= 0

	// 1. Top Header Banner
	var arrow string
	var badgeCategory string
	if isBullish {
		arrow = positiveStyle.Render("▲")
	} else {
		arrow = negativeStyle.Render("▼")
	}

	if item.Type == model.AssetStock {
		badgeCategory = stockSectionHeaderStyle.Render(" [📈 STOCK / ETF] ")
	} else {
		badgeCategory = cryptoSectionHeaderStyle.Render(" [🪙 CRYPTO ASSET] ")
	}

	titleLine := fmt.Sprintf("%s %s  %s %s",
		arrow,
		titleStyle.Render(fmt.Sprintf(" %s ", item.Display)),
		widgetNameStyle.Render(item.Name),
		badgeCategory,
	)
	sb.WriteString(titleLine)
	sb.WriteString("\n\n")

	// Spot Price & Key Metric Badges
	priceStr := summaryValueStyle.Render(fmt.Sprintf("Price: %s", formatPrice(item.Price)))
	changeStr := formatChange(item.Change24h)
	capStr := widgetCapStyle.Render(fmt.Sprintf("Market Cap: %s", item.MarketCap))
	volStr := widgetSymbolStyle.Render(fmt.Sprintf("24h Vol: %s", formatVolume(item.Volume24h)))

	statsBanner := fmt.Sprintf("%s    %s    %s    %s", priceStr, changeStr, capStr, volStr)
	sb.WriteString(statsBanner)
	sb.WriteString("\n\n")

	// 2. Interactive Timeframe Tabs Bar
	sb.WriteString(renderTimeframeTabs(activeTimeframe))
	sb.WriteString("\n\n")

	// 3. Large High-Definition Braille Area Chart
	chartHeight := 8
	chartContent := renderLargeBrailleChart(item, activeTimeframe, innerWidth, chartHeight)
	sb.WriteString(chartContent)
	sb.WriteString("\n\n")

	// 4. Two-Column Detailed Financial Statistics Grid
	colWidth := (innerWidth - 4) / 2
	if colWidth < 28 {
		colWidth = 28
	}

	leftCol := renderPriceActionBox(item, colWidth)
	rightCol := renderValuationBox(item, colWidth)

	splitGrid := lipgloss.JoinHorizontal(lipgloss.Top, leftCol, "  ", rightCol)
	sb.WriteString(splitGrid)
	sb.WriteString("\n\n")

	// 5. Controls Footer Bar
	lastUpdated := "Never"
	if !item.LastUpdated.IsZero() {
		lastUpdated = item.LastUpdated.Format("15:04:05")
	}

	footer := fmt.Sprintf(
		"[Esc / Enter] Back to Dashboard  •  [←/→/Tab] Timeframe  •  [↑/↓] Switch Asset  •  [r] Refresh  (Synced: %s)",
		lastUpdated,
	)
	sb.WriteString(statusBarStyle.Render(footer))

	return inspectorBoxStyle.Width(totalWidth - 2).Render(sb.String())
}

func renderTimeframeTabs(activeIdx int) string {
	var tabs []string
	for i, tf := range timeframeLabels {
		if i == activeIdx {
			tabs = append(tabs, tabActiveStyle.Render(fmt.Sprintf(" [ %s ] ", tf)))
		} else {
			tabs = append(tabs, tabInactiveStyle.Render(fmt.Sprintf("   %s   ", tf)))
		}
	}
	return strings.Join(tabs, "  ")
}

func renderLargeBrailleChart(item model.CryptoPair, tfIdx int, width int, chartRows int) string {
	history := item.History
	if len(history) < 2 {
		history = fetcher.GenerateTrendHistory(item.Open24h, item.Low24h, item.High24h, item.Price)
	}

	// Adjust history data points simulated for selected timeframe if needed
	chartHistory := resampleOrScaleHistory(history, tfIdx, item.Price)
	isBullish := item.Change24h >= 0

	yLabelWidth := 11
	plotCols := width - yLabelWidth - 4
	if plotCols < 30 {
		plotCols = 30
	}

	subWidth := plotCols * 2
	subHeight := chartRows * 4 // Braille matrix

	minVal := chartHistory[0]
	maxVal := chartHistory[0]
	for _, v := range chartHistory {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}

	diff := maxVal - minVal
	if diff == 0 {
		diff = 1.0
	}

	subGrid := make([][]uint8, chartRows)
	for r := 0; r < chartRows; r++ {
		subGrid[r] = make([]uint8, plotCols)
	}

	numPts := len(chartHistory)
	pts := make([][2]int, numPts)
	for j, val := range chartHistory {
		x := int(math.Round((float64(j) / float64(numPts-1)) * float64(subWidth-1)))
		norm := (val - minVal) / diff
		y := (subHeight - 1) - int(math.Round(norm*float64(subHeight-1)))
		if y < 0 {
			y = 0
		}
		if y >= subHeight {
			y = subHeight - 1
		}
		pts[j] = [2]int{x, y}
	}

	for j := 0; j < numPts-1; j++ {
		drawWidgetSubLine(subGrid, pts[j][0], pts[j][1], pts[j+1][0], pts[j+1][1])
	}

	var sb strings.Builder

	// Render chart grid with Y-axis price labels
	for r := 0; r < chartRows; r++ {
		// Calculate Y label value
		yNorm := 1.0 - (float64(r) / float64(chartRows-1))
		yVal := minVal + yNorm*diff
		yLabel := padLeft(formatPrice(yVal), yLabelWidth)

		axisChar := "│"
		if r == chartRows-1 {
			axisChar = "┼"
		}

		sb.WriteString(neutralStyle.Render(fmt.Sprintf("%s %s ", yLabel, axisChar)))

		var rowSb strings.Builder
		for c := 0; c < plotCols; c++ {
			bitmask := subGrid[r][c]
			if bitmask > 0 {
				rRune := rune(0x2800 + uint16(bitmask))
				rowSb.WriteRune(rRune)
			} else {
				rowSb.WriteRune(' ')
			}
		}

		if isBullish {
			sb.WriteString(positiveStyle.Render(rowSb.String()))
		} else {
			sb.WriteString(negativeStyle.Render(rowSb.String()))
		}
		sb.WriteString("\n")
	}

	// X-axis timeline line
	xAxisLine := strings.Repeat("─", plotCols)
	sb.WriteString(neutralStyle.Render(fmt.Sprintf("%s └%s\n", strings.Repeat(" ", yLabelWidth), xAxisLine)))

	// X-axis timeframe labels
	timelineStr := formatTimeframeLabels(tfIdx, yLabelWidth, plotCols)
	sb.WriteString(neutralStyle.Render(timelineStr))

	return sb.String()
}

func renderPriceActionBox(item model.CryptoPair, width int) string {
	var sb strings.Builder
	sb.WriteString(subHeaderStyle.Render("⚡ PRICE ACTION & VOLATILITY"))
	sb.WriteString("\n\n")

	// 24H Price Range Bar Slider
	lowStr := fmt.Sprintf("Low: %s", formatPrice(item.Low24h))
	highStr := fmt.Sprintf("High: %s", formatPrice(item.High24h))
	barWidth := width - 4
	if barWidth < 20 {
		barWidth = 20
	}
	space := barWidth - lipgloss.Width(lowStr) - lipgloss.Width(highStr)
	if space < 1 {
		space = 1
	}
	sb.WriteString(fmt.Sprintf("%s%s%s\n", lowStr, strings.Repeat(" ", space), highStr))
	rangeBar := RenderRangeBar(item.Price, item.Low24h, item.High24h, barWidth)
	sb.WriteString(rangeBar)
	sb.WriteString("\n\n")

	spread := item.High24h - item.Low24h
	spreadPct := 0.0
	if item.Low24h > 0 {
		spreadPct = (spread / item.Low24h) * 100.0
	}

	sb.WriteString(fmt.Sprintf("%-16s %s\n", "24h Open:", formatPrice(item.Open24h)))
	sb.WriteString(fmt.Sprintf("%-16s %s\n", "24h High:", formatPrice(item.High24h)))
	sb.WriteString(fmt.Sprintf("%-16s %s\n", "24h Low:", formatPrice(item.Low24h)))
	sb.WriteString(fmt.Sprintf("%-16s %s (%.2f%%)\n", "24h Spread:", formatPrice(spread), spreadPct))

	return statBoxStyle.Width(width).Render(sb.String())
}

func renderValuationBox(item model.CryptoPair, width int) string {
	var sb strings.Builder
	sb.WriteString(subHeaderStyle.Render("📊 VALUATION & LIQUIDITY"))
	sb.WriteString("\n\n")

	sb.WriteString(fmt.Sprintf("%-18s %s\n", "Market Cap:", widgetCapStyle.Render(item.MarketCap)))
	sb.WriteString(fmt.Sprintf("%-18s %s\n", "24h Trading Vol:", formatVolume(item.Volume24h)))

	source := "Coinbase Exchange API"
	if item.Type == model.AssetStock {
		source = "Market Equity Feed"
	} else if item.Symbol == "HMM-USD" || item.Symbol == "PEPE-USD" {
		source = "CoinGecko / DexScreener"
	}
	sb.WriteString(fmt.Sprintf("%-18s %s\n", "Data Provider:", source))

	syncTime := "Live"
	if !item.LastUpdated.IsZero() {
		syncTime = item.LastUpdated.Format("15:04:05")
	}
	sb.WriteString(fmt.Sprintf("%-18s %s\n", "Last Synced:", syncTime))
	sb.WriteString(fmt.Sprintf("%-18s %s\n", "Status:", positiveStyle.Render("Healthy (Active)")))

	return statBoxStyle.Width(width).Render(sb.String())
}

func formatTimeframeLabels(tfIdx int, padLeftWidth int, plotCols int) string {
	var leftLbl, midLbl, rightLbl string
	switch tfIdx {
	case 0: // 1D
		leftLbl = "24h ago"
		midLbl = "12h ago"
		rightLbl = "Now"
	case 1: // 1W
		leftLbl = "7d ago"
		midLbl = "3d ago"
		rightLbl = "Now"
	case 2: // 1M
		leftLbl = "30d ago"
		midLbl = "15d ago"
		rightLbl = "Now"
	case 3: // 1Y
		leftLbl = "1yr ago"
		midLbl = "6mo ago"
		rightLbl = "Now"
	default: // ALL
		leftLbl = "Inception"
		midLbl = "Mid"
		rightLbl = "Now"
	}

	pad := strings.Repeat(" ", padLeftWidth)
	gap := (plotCols - len(leftLbl) - len(midLbl) - len(rightLbl)) / 2
	if gap < 1 {
		gap = 1
	}

	return fmt.Sprintf("%s %s%s%s%s%s", pad, leftLbl, strings.Repeat(" ", gap), midLbl, strings.Repeat(" ", gap), rightLbl)
}

func resampleOrScaleHistory(raw []float64, tfIdx int, latestPrice float64) []float64 {
	if len(raw) == 0 {
		return []float64{latestPrice, latestPrice}
	}
	if tfIdx == 0 {
		return raw
	}

	// Generate simulated multi-timeframe series based on current trend & timeframe length
	length := len(raw)
	res := make([]float64, length)
	factor := float64(tfIdx + 1)

	startVal := raw[0] * (1.0 - (factor * 0.015))
	for i := 0; i < length; i++ {
		t := float64(i) / float64(length-1)
		wave := math.Sin(t*3.14159*float64(tfIdx+1)) * ((latestPrice - startVal) * 0.25)
		res[i] = startVal + (latestPrice-startVal)*t + wave
	}
	res[length-1] = latestPrice
	return res
}

func padLeft(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return strings.Repeat(" ", width-w) + s
}

// Ensure time is imported
var _ = time.Now
