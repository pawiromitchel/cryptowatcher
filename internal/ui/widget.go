package ui

import (
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"cryptowatcher/internal/model"
)

// RenderWidgetCard renders a macOS Stocks-style card for one asset. cardWidth
// is the lipgloss width of the card (content plus horizontal padding).
func RenderWidgetCard(item model.Asset, isSelected bool, cardWidth int) string {
	inner := cardWidth - 2 // minus horizontal padding
	if inner < 20 {
		inner = 20
	}
	bullish := item.Change24h >= 0

	// Line 1: direction arrow + symbol, market cap on the right.
	arrow := positiveStyle.Render("▲")
	if !bullish {
		arrow = negativeStyle.Render("▼")
	}
	if !item.HasQuote() {
		arrow = mutedStyle.Render("•")
	}
	left := arrow + " " + widgetSymbolStyle.Render(truncate(item.Display, inner-12))
	right := widgetCapStyle.Render(formatCompactUSD(item.MarketCap))
	line1 := padBetween(left, right, inner, lipgloss.Width(left), lipgloss.Width(right))

	// Line 2: name on the left, 24h change on the right.
	change := mutedStyle.Render(dash)
	if item.HasQuote() {
		change = formatChange(item.Change24h)
	}
	name := widgetNameStyle.Render(truncate(item.Name, inner-lipgloss.Width(change)-2))
	line2 := padBetween(name, change, inner, lipgloss.Width(name), lipgloss.Width(change))

	// Chart rows.
	chart := renderChart(item.History, bullish, inner-2, 3, "no chart data")

	// Price line: price right-aligned, with a state note on the left.
	var note, price string
	switch {
	case !item.HasQuote() && item.Err != nil:
		note = errorStyle.Render("unavailable")
		price = mutedStyle.Render(dash)
	case !item.HasQuote():
		note = mutedStyle.Render("loading…")
		price = mutedStyle.Render(dash)
	case item.Stale:
		note = warnStyle.Render("stale " + item.LastUpdated.Format("15:04"))
		price = mutedStyle.Render(formatPrice(item.Price))
	default:
		price = widgetPriceStyle.Render(formatPrice(item.Price))
	}
	priceLine := padBetween(note, price, inner, lipgloss.Width(note), lipgloss.Width(price))

	content := strings.Join([]string{line1, line2, "", chart, "", priceLine}, "\n")
	if isSelected {
		return selectedWidgetCardStyle.Width(cardWidth).Render(content)
	}
	return widgetCardStyle.Width(cardWidth).Render(content)
}

// renderChart draws history as a braille line chart rows terminal rows tall and
// width cells wide, indented by two spaces. With fewer than two points it draws
// a muted placeholder instead.
func renderChart(history []float64, bullish bool, width, rows int, placeholder string) string {
	if width < 4 {
		width = 4
	}
	pad := strings.Repeat(" ", 2)

	if len(history) < 2 {
		lines := make([]string, rows)
		for i := range lines {
			lines[i] = pad + strings.Repeat(" ", width)
		}
		msg := truncate(placeholder, width)
		mid := rows / 2
		lines[mid] = pad + strings.Repeat(" ", (width-lipgloss.Width(msg))/2) + mutedStyle.Render(msg)
		return strings.Join(lines, "\n")
	}

	subWidth, subHeight := width*2, rows*4
	minVal, maxVal := history[0], history[0]
	for _, v := range history {
		minVal, maxVal = math.Min(minVal, v), math.Max(maxVal, v)
	}
	diff := maxVal - minVal
	if diff == 0 {
		diff = 1
	}

	grid := make([][]uint8, rows)
	for r := range grid {
		grid[r] = make([]uint8, width)
	}

	pts := make([][2]int, len(history))
	for j, v := range history {
		x := int(math.Round(float64(j) / float64(len(history)-1) * float64(subWidth-1)))
		y := (subHeight - 1) - int(math.Round((v-minVal)/diff*float64(subHeight-1)))
		pts[j] = [2]int{x, clampInt(y, 0, subHeight-1)}
	}
	for j := 0; j < len(pts)-1; j++ {
		drawLine(grid, pts[j][0], pts[j][1], pts[j+1][0], pts[j+1][1])
	}

	style := positiveStyle
	if !bullish {
		style = negativeStyle
	}
	lines := make([]string, rows)
	for r := 0; r < rows; r++ {
		var row strings.Builder
		for c := 0; c < width; c++ {
			if mask := grid[r][c]; mask > 0 {
				row.WriteRune(rune(0x2800 + uint16(mask)))
			} else {
				row.WriteRune(' ')
			}
		}
		lines[r] = pad + style.Render(row.String())
	}
	return strings.Join(lines, "\n")
}

// brailleBits maps a sub-pixel (x in 0..1, y in 0..3) to its braille dot bit.
var brailleBits = [2][4]uint8{
	{0x01, 0x02, 0x04, 0x40},
	{0x08, 0x10, 0x20, 0x80},
}

// drawLine rasterizes a Bresenham line onto the braille sub-pixel grid.
func drawLine(grid [][]uint8, x0, y0, x1, y1 int) {
	dx, dy := absInt(x1-x0), absInt(y1-y0)
	sx, sy := -1, -1
	if x0 < x1 {
		sx = 1
	}
	if y0 < y1 {
		sy = 1
	}
	err := dx - dy
	for {
		setSubPixel(grid, x0, y0)
		if x0 == x1 && y0 == y1 {
			return
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

func setSubPixel(grid [][]uint8, x, y int) {
	if x < 0 || y < 0 {
		return
	}
	row, col := y/4, x/2
	if row >= len(grid) || col >= len(grid[0]) {
		return
	}
	grid[row][col] |= brailleBits[x%2][y%4]
}

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
