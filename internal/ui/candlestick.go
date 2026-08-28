package ui

import (
	"fmt"
	"math"
	"strings"

	"cryptowatcher/internal/model"
)

// RenderCandlestickChart renders a full OHLC candlestick chart for a single pair.
func RenderCandlestickChart(pair model.CryptoPair, totalWidth int) string {
	var sb strings.Builder

	// Header Banner
	title := fmt.Sprintf(" %s - OHLC CANDLESTICK CHART ", pair.Display)
	sb.WriteString(titleStyle.Render(title))
	sb.WriteString("\n\n")

	if len(pair.Candles) == 0 {
		sb.WriteString(neutralStyle.Render("   No OHLC candlestick data available for this pair."))
		return sb.String()
	}

	// Stats Summary Header
	lastCandle := pair.Candles[len(pair.Candles)-1]
	changeStr := formatChange(pair.Change24h)
	summary := fmt.Sprintf(
		"Spot Price: %s  (%s)    Open: %s  |  High: %s  |  Low: %s  |  Volume: %s",
		summaryValueStyle.Render(formatPrice(pair.Price)),
		changeStr,
		formatPrice(lastCandle.Open),
		formatPrice(lastCandle.High),
		formatPrice(lastCandle.Low),
		formatVolume(lastCandle.Volume),
	)
	sb.WriteString(neutralStyle.Render(summary))
	sb.WriteString("\n\n")

	chartRows := 12
	yAxisWidth := 11
	availableWidth := totalWidth - yAxisWidth - 3
	if availableWidth < 30 {
		availableWidth = 30
	}

	numCandles := len(pair.Candles)
	// We allocate 2 columns per candle (1 column for candle body/wick, 1 column space)
	candlesToDisplay := availableWidth / 2
	if candlesToDisplay > numCandles {
		candlesToDisplay = numCandles
	}

	displayCandles := pair.Candles[numCandles-candlesToDisplay:]

	// Find global min low and max high across displayed candles
	globalMin := displayCandles[0].Low
	globalMax := displayCandles[0].High
	for _, c := range displayCandles {
		if c.Low < globalMin {
			globalMin = c.Low
		}
		if c.High > globalMax {
			globalMax = c.High
		}
	}

	if globalMax == globalMin {
		globalMax += 1.0
		globalMin -= 1.0
	} else {
		margin := (globalMax - globalMin) * 0.05
		globalMax += margin
		globalMin -= margin
	}

	// Helper to convert price to row index (row 0 is top, row chartRows-1 is bottom)
	priceToRow := func(p float64) int {
		norm := (p - globalMin) / (globalMax - globalMin)
		row := (chartRows - 1) - int(math.Round(norm*float64(chartRows-1)))
		if row < 0 {
			row = 0
		}
		if row >= chartRows {
			row = chartRows - 1
		}
		return row
	}

	// Build 2D character grid
	plotWidth := candlesToDisplay * 2
	grid := make([][]string, chartRows)
	for r := 0; r < chartRows; r++ {
		grid[r] = make([]string, plotWidth)
		for col := 0; col < plotWidth; col++ {
			grid[r][col] = " "
		}
	}

	// Render each candle
	for i, c := range displayCandles {
		col := i * 2

		highRow := priceToRow(c.High)
		lowRow := priceToRow(c.Low)
		openRow := priceToRow(c.Open)
		closeRow := priceToRow(c.Close)

		bodyTop := openRow
		bodyBottom := closeRow
		if c.Close >= c.Open {
			bodyTop = closeRow
			bodyBottom = openRow
		}

		isBullish := c.Close >= c.Open
		cStyle := positiveStyle
		if !isBullish {
			cStyle = negativeStyle
		}

		wickChar := cStyle.Render("│")
		bodyChar := cStyle.Render("█")
		if bodyTop == bodyBottom {
			bodyChar = cStyle.Render("─")
		}

		// Draw candle elements
		for r := 0; r < chartRows; r++ {
			if r >= highRow && r <= lowRow {
				if r >= bodyTop && r <= bodyBottom {
					grid[r][col] = bodyChar
				} else {
					grid[r][col] = wickChar
				}
			}
		}
	}

	// Format Output String with Y-Axis Labels
	for r := 0; r < chartRows; r++ {
		valAtRow := globalMax - (float64(r)/float64(chartRows-1))*(globalMax-globalMin)
		yLabel := fmt.Sprintf("%10s", formatPrice(valAtRow))
		sb.WriteString(neutralStyle.Render(fmt.Sprintf("%s │ ", yLabel)))

		for col := 0; col < plotWidth; col++ {
			sb.WriteString(grid[r][col])
		}
		sb.WriteString("\n")
	}

	// X-axis timeline line
	xAxisLine := strings.Repeat("─", plotWidth)
	sb.WriteString(neutralStyle.Render(fmt.Sprintf("%11s └%s\n", "", xAxisLine)))

	// X-axis timeline labels
	timelineLabels := fmt.Sprintf(
		"%11s  %-16s %-16s %-16s %16s",
		"", "24h ago", "18h ago", "12h ago", "Now",
	)
	sb.WriteString(neutralStyle.Render(timelineLabels))
	sb.WriteString("\n\n")

	// Legend footer
	bullishTag := positiveStyle.Render("█ Bullish (Close ≥ Open)")
	bearishTag := negativeStyle.Render("█ Bearish (Close < Open)")
	wickTag := neutralStyle.Render("│ High/Low Wick")
	sb.WriteString(fmt.Sprintf("Legend:  %s    %s    %s", bullishTag, bearishTag, wickTag))

	return sb.String()
}
