package ui

import (
	"fmt"
	"math"
	"strings"

	"github.com/mattn/go-runewidth"
)

const dash = "—"

// formatPrice renders a USD price with precision appropriate to its magnitude.
func formatPrice(price float64) string {
	switch {
	case price <= 0:
		return dash
	case price < 0.000001:
		return fmt.Sprintf("$%.10f", price)
	case price < 0.0001:
		return fmt.Sprintf("$%.8f", price)
	case price < 0.01:
		return fmt.Sprintf("$%.6f", price)
	case price < 1:
		return fmt.Sprintf("$%.4f", price)
	default:
		return "$" + groupThousands(fmt.Sprintf("%.2f", price))
	}
}

func groupThousands(s string) string {
	intPart, frac, _ := strings.Cut(s, ".")
	if len(intPart) <= 3 {
		return s
	}
	var b strings.Builder
	pre := len(intPart) % 3
	if pre > 0 {
		b.WriteString(intPart[:pre])
	}
	for i := pre; i < len(intPart); i += 3 {
		if b.Len() > 0 {
			b.WriteByte(',')
		}
		b.WriteString(intPart[i : i+3])
	}
	if frac != "" {
		b.WriteString("." + frac)
	}
	return b.String()
}

// formatCompactUSD renders large dollar amounts as $1.57T / $297.3B / $12.4M.
func formatCompactUSD(v float64) string {
	switch {
	case v <= 0 || math.IsNaN(v):
		return dash
	case v >= 1e12:
		return fmt.Sprintf("$%.2fT", v/1e12)
	case v >= 1e9:
		return fmt.Sprintf("$%.1fB", v/1e9)
	case v >= 1e6:
		return fmt.Sprintf("$%.1fM", v/1e6)
	case v >= 1e3:
		return fmt.Sprintf("$%.1fK", v/1e3)
	default:
		return fmt.Sprintf("$%.0f", v)
	}
}

func formatVolume(vol float64) string {
	switch {
	case vol <= 0:
		return dash
	case vol >= 1e9:
		return fmt.Sprintf("%.2fB", vol/1e9)
	case vol >= 1e6:
		return fmt.Sprintf("%.2fM", vol/1e6)
	case vol >= 1e3:
		return fmt.Sprintf("%.2fK", vol/1e3)
	default:
		return fmt.Sprintf("%.2f", vol)
	}
}

func formatChange(change float64) string {
	switch {
	case change > 0:
		return positiveStyle.Render(fmt.Sprintf("+%.2f%%", change))
	case change < 0:
		return negativeStyle.Render(fmt.Sprintf("%.2f%%", change))
	}
	return neutralStyle.Render("0.00%")
}

// truncate shortens s to at most width terminal cells, adding an ellipsis.
func truncate(s string, width int) string {
	return runewidth.Truncate(s, width, "…")
}

// padBetween places left and right at opposite ends of a line of the given width.
func padBetween(left, right string, width, leftW, rightW int) string {
	gap := width - leftW - rightW
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}
