package ui

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"cryptowatcher/internal/model"
)

var lineColors = []lipgloss.Color{
	lipgloss.Color("86"),  // Cyan (BTC)
	lipgloss.Color("212"), // Magenta (ETH)
	lipgloss.Color("220"), // Gold / Yellow (SOL)
	lipgloss.Color("42"),  // Mint Green (DOGE)
	lipgloss.Color("208"), // Orange
	lipgloss.Color("141"), // Purple
}

var coinMarkers = []rune{'●', '▲', '■', '◆', '★', '⯌'}

type chartSeries struct {
	display string
	pcts    []float64
	color   lipgloss.Color
	marker  rune
	change  float64
}

func (s chartSeries) formattedChange() string {
	if s.change > 0 {
		return positiveStyle.Render(fmt.Sprintf("+%.2f%%", s.change))
	} else if s.change < 0 {
		return negativeStyle.Render(fmt.Sprintf("%.2f%%", s.change))
	}
	return neutralStyle.Render("0.00%")
}

// RenderMultiLineChart renders a normalized 7-day relative performance line chart.
func RenderMultiLineChart(pairs []model.CryptoPair, totalWidth int) string {
	if len(pairs) == 0 {
		return ""
	}

	chartHeight := 9
	yAxisWidth := 9
	plotWidth := totalWidth - yAxisWidth - 3
	if plotWidth < 30 {
		plotWidth = 30
	}

	seriesList := make([]chartSeries, 0, len(pairs))
	globalMin := 0.0
	globalMax := 0.0
	hasData := false

	for i, pair := range pairs {
		if pair.Err != nil || len(pair.History7D) == 0 {
			continue
		}

		basePrice := pair.History7D[0]
		if basePrice == 0 {
			basePrice = pair.Price
		}
		if basePrice == 0 {
			continue
		}

		pcts := make([]float64, len(pair.History7D))
		for j, p := range pair.History7D {
			pct := ((p - basePrice) / basePrice) * 100.0
			pcts[j] = pct

			if !hasData {
				globalMin = pct
				globalMax = pct
				hasData = true
			} else {
				if pct < globalMin {
					globalMin = pct
				}
				if pct > globalMax {
					globalMax = pct
				}
			}
		}

		color := lineColors[i%len(lineColors)]
		marker := coinMarkers[i%len(coinMarkers)]
		lastPct := pcts[len(pcts)-1]

		seriesList = append(seriesList, chartSeries{
			display: pair.Display,
			pcts:    pcts,
			color:   color,
			marker:  marker,
			change:  lastPct,
		})
	}

	if !hasData || len(seriesList) == 0 {
		return neutralStyle.Render("   Insufficient 7-day candle data for correlation chart.")
	}

	// Add margin padding to min/max
	if globalMax == globalMin {
		globalMax += 1.0
		globalMin -= 1.0
	} else {
		padding := (globalMax - globalMin) * 0.1
		globalMax += padding
		globalMin -= padding
	}

	// Build 2D character grid
	grid := make([][]string, chartHeight)
	for r := 0; r < chartHeight; r++ {
		grid[r] = make([]string, plotWidth)
		for c := 0; c < plotWidth; c++ {
			grid[r][c] = " "
		}
	}

	// Calculate zero line row index
	zeroRow := -1
	if globalMin <= 0 && globalMax >= 0 {
		zeroNorm := (0.0 - globalMin) / (globalMax - globalMin)
		zeroRow = chartHeight - 1 - int(math.Round(zeroNorm*float64(chartHeight-1)))
	}

	// Draw subtle zero baseline
	if zeroRow >= 0 && zeroRow < chartHeight {
		zeroStyle := lipgloss.NewStyle().Foreground(grayColor)
		for c := 0; c < plotWidth; c++ {
			grid[zeroRow][c] = zeroStyle.Render("─")
		}
	}

	// Plot each coin's points onto grid
	for _, s := range seriesList {
		numPts := len(s.pcts)
		style := lipgloss.NewStyle().Foreground(s.color).Bold(true)
		markerStr := style.Render(string(s.marker))

		for col := 0; col < plotWidth; col++ {
			idxFloat := (float64(col) / float64(plotWidth - 1)) * float64(numPts-1)
			idx := int(math.Round(idxFloat))
			if idx < 0 {
				idx = 0
			}
			if idx >= numPts {
				idx = numPts - 1
			}

			val := s.pcts[idx]
			norm := (val - globalMin) / (globalMax - globalMin)
			row := chartHeight - 1 - int(math.Round(norm*float64(chartHeight-1)))
			if row < 0 {
				row = 0
			}
			if row >= chartHeight {
				row = chartHeight - 1
			}

			grid[row][col] = markerStr
		}
	}

	// Render Grid into output string with Y-axis
	var sb strings.Builder
	for r := 0; r < chartHeight; r++ {
		valAtRow := globalMax - (float64(r)/float64(chartHeight-1))*(globalMax-globalMin)
		yLabel := fmt.Sprintf("%+6.1f%%", valAtRow)

		axisChar := "│"
		if r == zeroRow {
			axisChar = "┼"
		}

		sb.WriteString(neutralStyle.Render(fmt.Sprintf("%s %s ", yLabel, axisChar)))

		for c := 0; c < plotWidth; c++ {
			sb.WriteString(grid[r][c])
		}
		sb.WriteString("\n")
	}

	// X-axis timeline line
	xAxisLine := strings.Repeat("─", plotWidth)
	sb.WriteString(neutralStyle.Render(fmt.Sprintf("%9s └%s\n", "", xAxisLine)))

	// X-axis time labels
	timelineLabels := fmt.Sprintf(
		"%9s  %-14s %-14s %-14s %14s",
		"", "7d ago", "5d ago", "3d ago", "Now",
	)
	sb.WriteString(neutralStyle.Render(timelineLabels))
	sb.WriteString("\n\n")

	// Legend footer below chart
	legendItems := make([]string, 0, len(seriesList))
	for _, s := range seriesList {
		tagStyle := lipgloss.NewStyle().Foreground(s.color).Bold(true)
		item := fmt.Sprintf("%s %s (%s)", tagStyle.Render(string(s.marker)), tagStyle.Render(s.display), s.formattedChange())
		legendItems = append(legendItems, item)
	}

	sb.WriteString("Legend:  " + strings.Join(legendItems, "    "))
	return sb.String()
}
