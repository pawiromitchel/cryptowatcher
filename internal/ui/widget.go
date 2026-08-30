package ui

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"cryptowatcher/internal/model"
)

// RenderWidgetCard renders a tall, prominent macOS Stocks-style widget box for a ticker.
func RenderWidgetCard(item model.CryptoPair, isSelected bool, cardWidth int) string {
	innerWidth := cardWidth - 4
	if innerWidth < 22 {
		innerWidth = 22
	}

	isBullish := item.Change24h >= 0

	// 1. Top Header: Triangle Indicator + Symbol + Market Cap
	var arrow string
	if isBullish {
		arrow = positiveStyle.Render("▲")
	} else {
		arrow = negativeStyle.Render("▼")
	}

	symStr := fmt.Sprintf("%s %s", arrow, widgetSymbolStyle.Render(item.Display))
	capStr := widgetCapStyle.Render(item.MarketCap)
	space1 := innerWidth - lipgloss.Width(symStr) - lipgloss.Width(capStr)
	if space1 < 1 {
		space1 = 1
	}
	line1 := symStr + strings.Repeat(" ", space1) + capStr

	// 2. Subhead: Asset Name + 24h Change %
	nameStr := widgetNameStyle.Render(truncateString(item.Name, 14))
	changeStr := formatChange(item.Change24h)
	space2 := innerWidth - lipgloss.Width(nameStr) - lipgloss.Width(changeStr)
	if space2 < 1 {
		space2 = 1
	}
	line2 := nameStr + strings.Repeat(" ", space2) + changeStr

	// 3. Tall 3-Row Mini Inline Braille Line Chart
	chartWidth := innerWidth - 2
	miniChart := renderWidgetMiniChart(item.History, isBullish, chartWidth, 3)
	baseLine := lipgloss.NewStyle().Foreground(grayColor).Render(strings.Repeat("┄", chartWidth))

	// 4. Large Bold Price
	priceStr := widgetPriceStyle.Render(formatPrice(item.Price))
	if item.Err != nil {
		priceStr = errorStyle.Render("Error")
	}
	spacePrice := innerWidth - lipgloss.Width(priceStr)
	if spacePrice < 0 {
		spacePrice = 0
	}
	linePrice := strings.Repeat(" ", spacePrice) + priceStr

	content := fmt.Sprintf("%s\n%s\n\n%s\n  %s\n\n%s",
		line1,
		line2,
		miniChart,
		baseLine,
		linePrice,
	)

	if isSelected {
		return selectedWidgetCardStyle.Width(cardWidth).Render(content)
	}
	return widgetCardStyle.Width(cardWidth).Render(content)
}

func renderWidgetMiniChart(history []float64, isBullish bool, width int, chartRows int) string {
	if chartRows < 1 {
		chartRows = 3
	}

	if len(history) < 2 {
		var sb strings.Builder
		for r := 0; r < chartRows; r++ {
			if r == chartRows-1 {
				sb.WriteString("  " + strings.Repeat("⠒", width))
			} else {
				sb.WriteString("  " + strings.Repeat(" ", width))
			}
			if r < chartRows-1 {
				sb.WriteString("\n")
			}
		}
		if isBullish {
			return positiveStyle.Render(sb.String())
		}
		return negativeStyle.Render(sb.String())
	}

	subWidth := width * 2
	subHeight := chartRows * 4 // each character row has 4 sub-pixels vertically

	minVal := history[0]
	maxVal := history[0]
	for _, v := range history {
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
		subGrid[r] = make([]uint8, width)
	}

	numPts := len(history)
	pts := make([][2]int, numPts)
	for j, val := range history {
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
	for r := 0; r < chartRows; r++ {
		sb.WriteString("  ")
		var rowSb strings.Builder
		for c := 0; c < width; c++ {
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

		if r < chartRows-1 {
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

func drawWidgetSubLine(subGrid [][]uint8, x0, y0, x1, y1 int) {
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
		setWidgetSubPixel(subGrid, x0, y0)
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

func setWidgetSubPixel(subGrid [][]uint8, x, y int) {
	if y < 0 || x < 0 {
		return
	}
	cellRow := y / 4
	cellCol := x / 2

	if cellRow < 0 || cellRow >= len(subGrid) || cellCol < 0 || cellCol >= len(subGrid[0]) {
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

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-1] + "…"
}

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
