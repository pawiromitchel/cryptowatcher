package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"cryptowatcher/internal/model"
)

const (
	minTermWidth  = 40
	minTermHeight = 14
)

// View renders the dashboard.
func (m Model) View() string {
	if m.width > 0 && (m.width < minTermWidth || m.height < minTermHeight) {
		return fmt.Sprintf("Terminal too small (%dx%d).\nNeed at least %dx%d.", m.width, m.height, minTermWidth, minTermHeight)
	}

	if m.mode == modeDetail {
		if a := m.ActiveAsset(); a != nil {
			return lipgloss.JoinVertical(lipgloss.Left, m.renderHeader(), "", renderDetail(*a, m.termWidth()))
		}
	}

	body, selTop, selBottom := m.renderBody()
	footer := m.renderFooter()

	// Scroll the body so the selected card stays visible on short terminals.
	if m.height > 0 {
		avail := m.height - lipgloss.Height(footer)
		lines := strings.Split(body, "\n")
		if avail > 0 && len(lines) > avail {
			top := 0
			if selBottom >= avail {
				top = selBottom - avail + 1
			}
			if selTop < top {
				top = selTop
			}
			end := min(top+avail, len(lines))
			body = strings.Join(lines[top:end], "\n")
		}
	}
	return body + "\n" + footer
}

func (m Model) termWidth() int {
	if m.width > 0 {
		return m.width
	}
	return defaultWidth
}

// renderBody returns the header and all sections, plus the line range of the selected card row.
func (m Model) renderBody() (body string, selTop, selBottom int) {
	var lines []string
	add := func(s string) { lines = append(lines, strings.Split(s, "\n")...) }

	add(m.renderHeader())
	add("")

	perRow, cardW := m.cardsPerRow(), m.cardWidth()
	for i, s := range m.sections {
		style := cryptoSectionHeaderStyle
		if s.kind == model.AssetStock {
			style = stockSectionHeaderStyle
		}
		add(style.Render("▍ "+s.title) + mutedStyle.Render(fmt.Sprintf("  %d", len(s.items))))

		if len(s.items) == 0 {
			add(mutedStyle.Render("  Nothing here yet — press 'a' to add a ticker."))
			add("")
			continue
		}
		for start := 0; start < len(s.items); start += perRow {
			end := min(start+perRow, len(s.items))
			cards := make([]string, 0, perRow)
			for j := start; j < end; j++ {
				cards = append(cards, RenderWidgetCard(s.items[j], i == m.active && j == s.cursor, cardW))
			}
			rowTop := len(lines)
			add(lipgloss.JoinHorizontal(lipgloss.Top, cards...))
			if i == m.active && s.cursor >= start && s.cursor < end {
				selTop, selBottom = rowTop, len(lines)-1
			}
		}
	}
	return strings.Join(lines, "\n"), selTop, selBottom
}

// renderHeader is the title bar plus a one-line market summary.
func (m Model) renderHeader() string {
	title := titleStyle.Render("CRYPTOWATCHER")
	all := m.AllAssets()

	var parts []string
	parts = append(parts, m.statusIndicator(all))

	var best, worst *model.Asset
	for i := range all {
		a := &all[i]
		if !a.HasQuote() {
			continue
		}
		if best == nil || a.Change24h > best.Change24h {
			best = a
		}
		if worst == nil || a.Change24h < worst.Change24h {
			worst = a
		}
	}
	if best != nil && worst != nil && best != worst {
		parts = append(parts,
			"▲ "+best.Display+" "+formatChange(best.Change24h),
			"▼ "+worst.Display+" "+formatChange(worst.Change24h))
	}
	parts = append(parts, mutedStyle.Render(fmt.Sprintf("%d assets", len(all))))

	return ansi.Truncate(title+"  "+strings.Join(parts, mutedStyle.Render("  ·  ")), m.termWidth(), "…")
}

func (m Model) statusIndicator(all []model.Asset) string {
	var failing int
	for _, a := range all {
		if a.Err != nil {
			failing++
		}
	}
	switch {
	case m.lastRefresh.IsZero() && m.refreshing:
		return warnStyle.Render("◌ LOADING")
	case len(all) > 0 && failing == len(all):
		return errorStyle.Render("● OFFLINE")
	case failing > 0:
		return warnStyle.Render(fmt.Sprintf("● PARTIAL %d/%d", len(all)-failing, len(all)))
	case m.refreshing:
		return warnStyle.Render("◌ UPDATING")
	}
	return positiveStyle.Render("● LIVE")
}

// renderFooter shows the active prompt (if any), the status line and key help.
func (m Model) renderFooter() string {
	var lines []string

	switch m.mode {
	case modeAdd:
		lines = append(lines, modalStyle.Render(fmt.Sprintf(
			"Add ticker — crypto (BTC, DOGE, HMM) or stock (SPY, NVDA)\n\n%s\n\nenter  look up    esc  cancel",
			m.textInput.View())))
	case modeDeleteConfirm:
		name := "this ticker"
		if a := m.ActiveAsset(); a != nil {
			name = fmt.Sprintf("%s (%s)", a.Display, a.Name)
		}
		lines = append(lines, modalStyle.Render(fmt.Sprintf(
			"Remove %s from your watchlist?\n\ny / enter  remove    n / esc  cancel",
			widgetSymbolStyle.Render(name))))
	}

	if m.statusMsg != "" && time.Now().Before(m.statusUntil) {
		style := neutralStyle
		if m.statusErr {
			style = errorStyle
		}
		lines = append(lines, style.Render(truncate(m.statusMsg, m.termWidth()-1)))
	} else {
		updated := "never"
		if !m.lastRefresh.IsZero() {
			updated = m.lastRefresh.Format("15:04:05")
		}
		lines = append(lines, mutedStyle.Render("updated "+updated))
	}

	m.help.Width = m.termWidth()
	lines = append(lines, ansi.Truncate(m.help.View(m.keys), m.termWidth(), "…"))
	return strings.Join(lines, "\n")
}
