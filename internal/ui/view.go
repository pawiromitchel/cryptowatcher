package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"cryptowatcher/internal/model"
)

// View renders the terminal user interface string for the macOS-style Widget Dashboard.
func (m Model) View() string {
	var b strings.Builder

	// Header Banner
	b.WriteString(titleStyle.Render(" WATCHER - STOCKS & CRYPTO DASHBOARD "))
	b.WriteString("\n\n")

	// Top Summary Cards Dashboard
	b.WriteString(m.renderSummaryCards())
	b.WriteString("\n")

	cardWidth := 30

	// 1. Cryptocurrency Section
	b.WriteString(cryptoSectionHeaderStyle.Render("🪙  CRYPTOCURRENCY"))
	b.WriteString("\n")
	b.WriteString(m.renderWidgetGrid(m.cryptoPairs, m.sectionIndex == 0, m.cryptoCursor, cardWidth))
	b.WriteString("\n")

	// 2. Stocks & Equities Section
	b.WriteString(stockSectionHeaderStyle.Render("📈  STOCKS & EQUITIES"))
	b.WriteString("\n")
	b.WriteString(m.renderWidgetGrid(m.stockPairs, m.sectionIndex == 1, m.stockCursor, cardWidth))
	b.WriteString("\n")

	// Modal Overlay for Adding Pairs
	if m.mode == modeAdd {
		b.WriteString("\n")
		modalContent := fmt.Sprintf(
			"Enter Ticker to Watch (e.g. BTC, ETH, SOL, SPY, TSLA, AAPL, NVDA):\n\n%s\n\n(Press Enter to submit, Esc to cancel)",
			m.textInput.View(),
		)
		b.WriteString(modalStyle.Render(modalContent))
		b.WriteString("\n")
	}

	// Modal Overlay for Deleting Pairs Confirmation
	if m.mode == modeDeleteConfirm {
		b.WriteString("\n")
		active := m.ActivePair()
		targetName := "selected ticker"
		if active != nil {
			targetName = fmt.Sprintf("%s (%s)", active.Display, active.Name)
		}
		modalContent := fmt.Sprintf(
			"Remove Ticker Confirmation:\n\nAre you sure you want to remove %s?\n\n[y / Enter] Confirm Removal    [n / Esc] Cancel",
			widgetSymbolStyle.Render(targetName),
		)
		b.WriteString(modalStyle.Render(modalContent))
		b.WriteString("\n")
	}

	// Status Bar & Controls Footer
	b.WriteString(m.renderFooter())

	return b.String()
}

func (m Model) renderWidgetGrid(items []model.CryptoPair, isSectionActive bool, selectedIdx int, cardWidth int) string {
	if len(items) == 0 {
		return rowStyle.Render("   No items in this section. Press 'a' to add a ticker.\n")
	}

	cardsPerRow := m.calculateCardsPerRow()
	var sb strings.Builder
	var currentRow []string

	for i, item := range items {
		isSelected := isSectionActive && (i == selectedIdx)
		card := RenderWidgetCard(item, isSelected, cardWidth)
		currentRow = append(currentRow, card)

		if len(currentRow) == cardsPerRow || i == len(items)-1 {
			sb.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, currentRow...))
			sb.WriteString("\n")
			currentRow = nil
		}
	}

	return sb.String()
}

func (m Model) renderSummaryCards() string {
	all := m.AllPairs()
	if len(all) == 0 {
		return ""
	}

	bestPair := all[0]
	worstPair := all[0]

	for _, p := range all {
		if p.Change24h > bestPair.Change24h {
			bestPair = p
		}
		if p.Change24h < worstPair.Change24h {
			worstPair = p
		}
	}

	c1 := summaryCardStyle.Render(fmt.Sprintf("ASSETS\n%s", summaryValueStyle.Render(fmt.Sprintf("%d Monitored (%d Crypto, %d Stocks)", len(all), len(m.cryptoPairs), len(m.stockPairs)))))

	bestStr := fmt.Sprintf("%s (%s)", bestPair.Display, formatChange(bestPair.Change24h))
	c2 := summaryCardStyle.Render(fmt.Sprintf("TOP GAINER\n%s", bestStr))

	worstStr := fmt.Sprintf("%s (%s)", worstPair.Display, formatChange(worstPair.Change24h))
	c3 := summaryCardStyle.Render(fmt.Sprintf("TOP LOSER\n%s", worstStr))

	statusText := "🟢 LIVE (Coinbase, CoinGecko & Yahoo)"
	if m.loading {
		statusText = "🟡 UPDATING..."
	} else if m.err != nil {
		statusText = "🔴 ERROR"
	}
	c4 := summaryCardStyle.Render(fmt.Sprintf("STATUS\n%s", summaryValueStyle.Render(statusText)))

	return lipgloss.JoinHorizontal(lipgloss.Top, c1, c2, c3, c4)
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
		"[a] Add Ticker | [d] Remove | [r] Refresh | [←/→/↑/↓] Navigate Grid | [q] Quit  (Last Updated: %s)",
		lastUpdatedStr,
	)
	sb.WriteString(statusBarStyle.Render(controls))
	sb.WriteString("\n")

	return sb.String()
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
