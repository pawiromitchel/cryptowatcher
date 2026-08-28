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

type chartSeries struct {
	display string
	pcts    []float64
	color   lipgloss.Color
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

// RenderMultiLineChart renders a continuous Braille line chart for 7-day relative performance.
func RenderMultiLineChart(pairs []model.CryptoPair, totalWidth int) string {
	if len(pairs) == 0 {
		return ""
	}

	chartRows := 9
	yAxisWidth := 9
	plotCols := totalWidth - yAxisWidth - 3
	if plotCols < 30 {
		plotCols = 30
	}

	subWidth := plotCols * 2
	subHeight := chartRows * 4

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
		lastPct := pcts[len(pcts)-1]

		seriesList = append(seriesList, chartSeries{
			display: pair.Display,
			pcts:    pcts,
			color:   color,
			change:  lastPct,
		})
	}

	if !hasData || len(seriesList) == 0 {
		return neutralStyle.Render("   Insufficient 7-day candle data for correlation chart.")
	}

	// Add margin padding
	if globalMax == globalMin {
		globalMax += 1.0
		globalMin -= 1.0
	} else {
		padding := (globalMax - globalMin) * 0.1
		globalMax += padding
		globalMin -= padding
	}

	// Calculate zero line row index
	zeroRow := -1
	if globalMin <= 0 && globalMax >= 0 {
		zeroNorm := (0.0 - globalMin) / (globalMax - globalMin)
		zeroRow = chartRows - 1 - int(math.Round(zeroNorm*float64(chartRows-1)))
	}

	// Create styled character grid
	cellGrid := make([][]string, chartRows)
	for r := 0; r < chartRows; r++ {
		cellGrid[r] = make([]string, plotCols)
		for c := 0; c < plotCols; c++ {
			if r == zeroRow {
				cellGrid[r][c] = lipgloss.NewStyle().Foreground(grayColor).Render("─")
			} else {
				cellGrid[r][c] = " "
			}
		}
	}

	// Render continuous Braille lines for each coin series
	for _, s := range seriesList {
		subGrid := make([][]uint8, subHeight)
		for r := 0; r < subHeight; r++ {
			subGrid[r] = make([]uint8, plotCols)
		}

		numPts := len(s.pcts)
		pts := make([][2]int, numPts)

		for j, val := range s.pcts {
			x := int(math.Round((float64(j) / float64(numPts-1)) * float64(subWidth-1)))
			norm := (val - globalMin) / (globalMax - globalMin)
			y := (subHeight - 1) - int(math.Round(norm*float64(subHeight-1)))
			if y < 0 {
				y = 0
			}
			if y >= subHeight {
				y = subHeight - 1
			}
			pts[j] = [2]int{x, y}
		}

		// Connect adjacent sub-pixels with Bresenham's line algorithm
		for j := 0; j < numPts-1; j++ {
			p1 := pts[j]
			p2 := pts[j+1]
			drawSubLine(subGrid, p1[0], p1[1], p2[0], p2[1])
		}

		// Apply braille characters to main cellGrid with series color
		style := lipgloss.NewStyle().Foreground(s.color).Bold(true)
		for r := 0; r < chartRows; r++ {
			for c := 0; c < plotCols; c++ {
				bitmask := subGrid[r][c]
				if bitmask > 0 {
					rRune := rune(0x2800 + uint16(bitmask))
					cellGrid[r][c] = style.Render(string(rRune))
				}
			}
		}
	}

	// Build output string
	var sb strings.Builder
	for r := 0; r < chartRows; r++ {
		valAtRow := globalMax - (float64(r)/float64(chartRows-1))*(globalMax-globalMin)
		yLabel := fmt.Sprintf("%+6.1f%%", valAtRow)

		axisChar := "│"
		if r == zeroRow {
			axisChar = "┼"
		}

		sb.WriteString(neutralStyle.Render(fmt.Sprintf("%s %s ", yLabel, axisChar)))

		for c := 0; c < plotCols; c++ {
			sb.WriteString(cellGrid[r][c])
		}
		sb.WriteString("\n")
	}

	// X-axis timeline
	xAxisLine := strings.Repeat("─", plotCols)
	sb.WriteString(neutralStyle.Render(fmt.Sprintf("%9s └%s\n", "", xAxisLine)))

	timelineLabels := fmt.Sprintf(
		"%9s  %-14s %-14s %-14s %14s",
		"", "7d ago", "5d ago", "3d ago", "Now",
	)
	sb.WriteString(neutralStyle.Render(timelineLabels))
	sb.WriteString("\n\n")

	// Legend
	legendItems := make([]string, 0, len(seriesList))
	for _, s := range seriesList {
		tagStyle := lipgloss.NewStyle().Foreground(s.color).Bold(true)
		item := fmt.Sprintf("%s %s (%s)", tagStyle.Render("─"), tagStyle.Render(s.display), s.formattedChange())
		legendItems = append(legendItems, item)
	}

	sb.WriteString("Legend:  " + strings.Join(legendItems, "    "))
	return sb.String()
}

func drawSubLine(subGrid [][]uint8, x0, y0, x1, y1 int) {
	dx := absInt(x1 - x0)
	dy := absInt(y1 - y0)
	sx := -1
	if x0 < x1 {
		sx = 1
	}
	sy := -1
	if y0 < y1 {
		sy = 1
	}
	err := dx - dy

	for {
		setSubPixel(subGrid, x0, y0)
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x0 += sx
		}
		if e2 < dx {
			err += dx
			y0 += sy
		}
	}
}

func setSubPixel(subGrid [][]uint8, x, y int) {
	if y < 0 || x < 0 {
		return
	}
	chartRows := len(subGrid) / 4
	cellRow := y / 4
	cellCol := x / 2

	if cellRow < 0 || cellRow >= chartRows || cellCol < 0 || cellCol >= len(subGrid[0]) {
		return
	}

	subX := x % 2
	subY := y % 4

	var bit uint8
	if subX == 0 {
		switch subY {
		case 0:
			bit = 0x01
		case 1:
			bit = 0x02
		case 2:
			bit = 0x04
		case 3:
			bit = 0x40
		}
	} else {
		switch subY {
		case 0:
			bit = 0x08
		case 1:
			bit = 0x10
		case 2:
			bit = 0x20
		case 3:
			bit = 0x80
		}
	}

	subGrid[cellRow][cellCol] |= bit
}

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
