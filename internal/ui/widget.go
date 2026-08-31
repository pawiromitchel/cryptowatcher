package ui

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"cryptowatcher/internal/model"
)

var bigDigitFont = map[rune][3]string{
	'$': {"╷ ", "─┼", " ╵"},
	'0': {"┌─┐", "│ │", "└─┘"},
	'1': {" ╷ ", " │ ", " ╵ "},
	'2': {"┌─┐", "┌─┘", "└──"},
	'3': {"┌─┐", " ─┤", "└─┘"},
	'4': {"╷ ╷", "└──┤", "  ╵"},
	'5': {"┌─┐", "└─┐", "──┘"},
	'6': {"┌─┐", "├─┐", "└─┘"},
	'7': {"──┐", "  │", "  ╵"},
	'8': {"┌─┐", "├─┤", "└─┘"},
	'9': {"┌─┐", "└─┤", "──┘"},
	'.': {" ", " ", "•"},
	',': {" ", " ", "‚"},
	'k': {"╷ ╷", "├─┴", "╵ ╵"},
	'K': {"╷ ╷", "├─┴", "╵ ╵"},
	'M': {"╷─╷", "│ │", "╵ ╵"},
	'B': {"┌─┐", "├─┤", "└─┘"},
	'T': {"──┬", "  │", "  ╵"},
	'-': {"   ", "───", "   "},
	'+': {" ╷ ", "─┼─", " ╵ "},
}

// RenderWidgetCard renders a tall, prominent macOS Stocks-style widget box for a ticker.
// cardMode: 0 = Line Chart view, 1 = Big Price Focus view.
func RenderWidgetCard(item model.CryptoPair, isSelected bool, cardWidth int, cardMode int) string {
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

	var content string

	if cardMode == 1 {
		// --- BIG FONT PRICE FOCUS VIEW ---
		bigPrice := RenderBigPrice(item.Price, isBullish, innerWidth)

		// Bottom Subtitle: 24h Range or Volume
		lowFormatted := formatCompactPrice(item.Low24h)
		highFormatted := formatCompactPrice(item.High24h)
		rangeStr := fmt.Sprintf("L: %s  ┄┄  H: %s", lowFormatted, highFormatted)
		if item.Low24h == 0 && item.High24h == 0 {
			rangeStr = fmt.Sprintf("24h Vol: %s", formatVolume(item.Volume24h))
		}
		rangeRendered := neutralStyle.Render(rangeStr)
		spaceRange := (innerWidth - lipgloss.Width(rangeRendered)) / 2
		if spaceRange < 0 {
			spaceRange = 0
		}
		lineBottom := strings.Repeat(" ", spaceRange) + rangeRendered

		content = fmt.Sprintf("%s\n%s\n\n%s\n\n%s",
			line1,
			line2,
			bigPrice,
			lineBottom,
		)
	} else {
		// --- DEFAULT 3-ROW BRAILLE MINI LINE CHART VIEW ---
		chartWidth := innerWidth - 2
		miniChart := renderWidgetMiniChart(item.History, isBullish, chartWidth, 3)
		baseLine := lipgloss.NewStyle().Foreground(grayColor).Render(strings.Repeat("┄", chartWidth))

		priceStr := widgetPriceStyle.Render(formatPrice(item.Price))
		if item.Err != nil {
			priceStr = errorStyle.Render("Error")
		}
		spacePrice := innerWidth - lipgloss.Width(priceStr)
		if spacePrice < 0 {
			spacePrice = 0
		}
		linePrice := strings.Repeat(" ", spacePrice) + priceStr

		content = fmt.Sprintf("%s\n%s\n\n%s\n  %s\n\n%s",
			line1,
			line2,
			miniChart,
			baseLine,
			linePrice,
		)
	}

	if isSelected {
		return selectedWidgetCardStyle.Width(cardWidth).Render(content)
	}
	return widgetCardStyle.Width(cardWidth).Render(content)
}

// RenderBigPrice formats a numeric price into a 3-row tall ASCII big digit display.
func RenderBigPrice(price float64, isBullish bool, maxWidth int) string {
	formattedStr := formatPriceForBigDigits(price)

	var row0, row1, row2 strings.Builder
	for _, ch := range formattedStr {
		glyph, found := bigDigitFont[ch]
		if !found {
			glyph = [3]string{" ", " ", string(ch)}
		}
		row0.WriteString(glyph[0])
		row1.WriteString(glyph[1])
		row2.WriteString(glyph[2])
	}

	r0 := row0.String()
	r1 := row1.String()
	r2 := row2.String()

	textWidth := lipgloss.Width(r0)
	pad := (maxWidth - textWidth) / 2
	if pad < 0 {
		pad = 0
	}
	padStr := strings.Repeat(" ", pad)

	res := fmt.Sprintf("%s%s\n%s%s\n%s%s",
		padStr, r0,
		padStr, r1,
		padStr, r2,
	)

	if isBullish {
		return positiveStyle.Render(res)
	}
	return negativeStyle.Render(res)
}

func formatPriceForBigDigits(price float64) string {
	if price >= 100000 {
		return fmt.Sprintf("$%.0f", price)
	}
	if price >= 1000 {
		return fmt.Sprintf("$%.0f", price)
	}
	if price >= 10 {
		return fmt.Sprintf("$%.2f", price)
	}
	if price >= 1 {
		return fmt.Sprintf("$%.2f", price)
	}
	if price >= 0.01 {
		return fmt.Sprintf("$%.4f", price)
	}
	return fmt.Sprintf("$%.4f", price)
}

func formatCompactPrice(price float64) string {
	if price >= 1000 {
		return fmt.Sprintf("$%.1fK", price/1000.0)
	}
	if price >= 1 {
		return fmt.Sprintf("$%.2f", price)
	}
	return fmt.Sprintf("$%.4f", price)
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
	subHeight := chartRows * 4

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
	chartRows := len(subGrid)
	if chartRows == 0 {
		return
	}
	width := len(subGrid[0])
	subHeight := chartRows * 4

	if x < 0 || x >= width*2 || y < 0 || y >= subHeight {
		return
	}

	cellRow := y / 4
	cellCol := x / 2
	subX := x % 2
	subY := y % 4

	if cellRow < 0 || cellRow >= chartRows || cellCol < 0 || cellCol >= width {
		return
	}

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
