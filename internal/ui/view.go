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
	b.WriteString(titleStyle.Render(" CRYPTOWATCHER "))
	b.WriteString("\n\n")

	// Main Watchlist Table
	b.WriteString(m.renderTable())
	b.WriteString("\n")

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

func (m Model) renderTable() string {
	var sb strings.Builder

	// Column header
	header := fmt.Sprintf(
		"%-3s %-12s %-14s %-12s %-14s %-14s %-14s",
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

		// Use padRight with lipgloss.Width to ensure ANSI colors do not corrupt column alignment
		rowContent := fmt.Sprintf(
			"%s %s %s %s %s %s %s",
			padRight(cursorStr, 3),
			padRight(pair.Display, 12),
			padRight(priceStr, 14),
			padRight(changeStr, 12),
			padRight(highStr, 14),
			padRight(lowStr, 14),
			padRight(volStr, 14),
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
