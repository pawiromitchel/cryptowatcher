// Package ui implements the Bubble Tea terminal dashboard.
package ui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"cryptowatcher/internal/config"
	"cryptowatcher/internal/fetcher"
	"cryptowatcher/internal/model"
)

type viewMode int

const (
	modeNormal viewMode = iota
	modeAdd
	modeDeleteConfirm
	modeDetail
)

const (
	refreshTimeout = 15 * time.Second
	addTimeout     = 15 * time.Second
	statusDuration = 6 * time.Second
	defaultWidth   = 110
)

// Messages.
type tickMsg time.Time

type refreshBatch struct {
	section int
	symbols []string // symbols requested, aligned with assets
	assets  []model.Asset
}

type refreshedMsg struct{ batches []refreshBatch }

type addedMsg struct {
	raw   string
	asset model.Asset
	err   error
}

// section is one titled group of watched assets.
type section struct {
	title  string
	kind   model.AssetType
	items  []model.Asset
	cursor int
}

// Model is the Bubble Tea application state.
type Model struct {
	cfg      *model.Config
	fetcher  fetcher.PriceFetcher
	sections []section
	active   int

	mode          viewMode
	pendingDelete string // symbol awaiting delete confirmation
	textInput     textinput.Model
	help          help.Model
	keys          KeyMap

	statusMsg   string
	statusErr   bool
	statusUntil time.Time
	refreshing  bool
	lastRefresh time.Time
	width       int
	height      int
}

const (
	secCrypto = 0
	secStock  = 1
)

// NewModel builds the dashboard from the saved configuration.
func NewModel(cfg *model.Config, pf fetcher.PriceFetcher) Model {
	ti := textinput.New()
	ti.Placeholder = "BTC, DOGE, SPY, NVDA, ..."
	ti.CharLimit = 24
	ti.Width = 32

	return Model{
		cfg:     cfg,
		fetcher: pf,
		sections: []section{
			{title: "CRYPTOCURRENCY", kind: model.AssetCrypto, items: placeholders(cfg.CryptoPairs, model.AssetCrypto)},
			{title: "STOCKS & EQUITIES", kind: model.AssetStock, items: placeholders(cfg.StockPairs, model.AssetStock)},
		},
		textInput:  ti,
		help:       help.New(),
		keys:       DefaultKeyMap(),
		refreshing: true,
	}
}

// placeholders creates not-yet-fetched cards for the saved symbols.
func placeholders(symbols []string, kind model.AssetType) []model.Asset {
	out := make([]model.Asset, 0, len(symbols))
	for _, s := range symbols {
		var sym, disp string
		if kind == model.AssetStock {
			sym, _ = fetcher.NormalizeSymbol(s)
			sym = strings.TrimSuffix(sym, "-USD")
			disp = sym
			if sym == "SPY" || sym == "S&P" {
				disp = "S&P 500"
			}
		} else {
			sym, disp = fetcher.NormalizeSymbol(s)
		}
		if sym == "" {
			continue
		}
		out = append(out, model.Asset{Symbol: sym, Display: disp, Name: fetcher.LookupAssetName(sym), Type: kind})
	}
	return out
}

// Init triggers the first fetch and the periodic refresh timer.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.refreshCmd(), m.tickCmd())
}

func (m Model) tickCmd() tea.Cmd {
	interval := time.Duration(m.cfg.RefreshInterval) * time.Second
	if interval <= 0 {
		interval = model.DefaultRefreshInterval * time.Second
	}
	return tea.Tick(interval, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// refreshCmd fetches every watched symbol. The symbol lists are captured now,
// and results are merged by symbol, so edits made while it runs are safe.
func (m Model) refreshCmd() tea.Cmd {
	type req struct {
		section int
		kind    model.AssetType
		symbols []string
	}
	reqs := make([]req, len(m.sections))
	for i, s := range m.sections {
		syms := make([]string, len(s.items))
		for j, it := range s.items {
			syms[j] = it.Symbol
		}
		reqs[i] = req{i, s.kind, syms}
	}
	pf := m.fetcher

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), refreshTimeout)
		defer cancel()

		batches := make([]refreshBatch, 0, len(reqs))
		for _, r := range reqs {
			if len(r.symbols) == 0 {
				continue
			}
			assets, err := pf.FetchPrices(fetcher.WithKind(ctx, r.kind), r.symbols)
			if err != nil || len(assets) != len(r.symbols) {
				assets = make([]model.Asset, len(r.symbols))
				for i := range assets {
					assets[i].Err = fmt.Errorf("refresh failed: %v", err)
				}
			}
			batches = append(batches, refreshBatch{section: r.section, symbols: r.symbols, assets: assets})
		}
		return refreshedMsg{batches: batches}
	}
}

func (m Model) addCmd(raw string) tea.Cmd {
	pf := m.fetcher
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), addTimeout)
		defer cancel()
		asset, err := pf.FetchPair(ctx, raw)
		return addedMsg{raw: raw, asset: asset, err: err}
	}
}

// AllAssets returns every watched asset.
func (m Model) AllAssets() []model.Asset {
	var all []model.Asset
	for _, s := range m.sections {
		all = append(all, s.items...)
	}
	return all
}

// ActiveAsset returns the highlighted asset, or nil if the dashboard is empty.
func (m Model) ActiveAsset() *model.Asset {
	if m.active < 0 || m.active >= len(m.sections) {
		return nil
	}
	s := &m.sections[m.active]
	if s.cursor < 0 || s.cursor >= len(s.items) {
		return nil
	}
	return &s.items[s.cursor]
}

func (m *Model) setStatus(msg string, isErr bool) {
	m.statusMsg, m.statusErr = msg, isErr
	m.statusUntil = time.Now().Add(statusDuration)
}

// Update handles incoming messages and keypresses.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.help.Width = msg.Width

	case tickMsg:
		cmds := []tea.Cmd{m.tickCmd()}
		if !m.refreshing {
			m.refreshing = true
			cmds = append(cmds, m.refreshCmd())
		}
		return m, tea.Batch(cmds...)

	case refreshedMsg:
		m.refreshing = false
		m.lastRefresh = time.Now()
		m.mergeRefresh(msg)

	case addedMsg:
		m.handleAdded(msg)

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

// mergeRefresh applies fetch results by symbol. A failed fetch keeps the last
// good data and flags it stale instead of blanking the card.
func (m *Model) mergeRefresh(msg refreshedMsg) {
	for _, b := range msg.batches {
		s := &m.sections[b.section]
		for i, sym := range b.symbols {
			idx := indexOf(s.items, sym)
			if idx < 0 {
				continue // removed while the fetch was in flight
			}
			a := b.assets[i]
			if a.Err == nil && a.Price > 0 {
				a.Symbol, a.Stale, a.Err = sym, false, nil
				s.items[idx] = a
				continue
			}
			it := &s.items[idx]
			it.Stale = it.HasQuote()
			it.Err = a.Err
			if it.Err == nil {
				it.Err = fmt.Errorf("no data")
			}
		}
	}
}

func (m *Model) handleAdded(msg addedMsg) {
	if msg.err != nil {
		m.setStatus(fmt.Sprintf("Could not add %s: %v", strings.ToUpper(strings.TrimSpace(msg.raw)), msg.err), true)
		return
	}
	a := msg.asset
	secIdx := secCrypto
	if a.Type == model.AssetStock {
		secIdx = secStock
	}
	s := &m.sections[secIdx]

	if idx := indexOf(s.items, a.Symbol); idx >= 0 {
		s.items[idx] = a
		s.cursor, m.active = idx, secIdx
		m.setStatus(fmt.Sprintf("Already watching %s", a.Display), false)
		return
	}

	s.items = append(s.items, a)
	s.cursor, m.active = len(s.items)-1, secIdx
	if err := m.persist(); err != nil {
		m.setStatus(fmt.Sprintf("Added %s, but saving config failed: %v", a.Display, err), true)
		return
	}
	m.setStatus(fmt.Sprintf("Added %s (%s) via %s", a.Display, a.Name, orDash(a.Source)), false)
}

// persist writes the current watchlists to the config file.
func (m *Model) persist() error {
	m.cfg.CryptoPairs = symbols(m.sections[secCrypto].items)
	m.cfg.StockPairs = symbols(m.sections[secStock].items)
	return config.Save(m.cfg)
}

func symbols(items []model.Asset) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.Symbol
	}
	return out
}

func indexOf(items []model.Asset, symbol string) int {
	for i, it := range items {
		if it.Symbol == symbol {
			return i
		}
	}
	return -1
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "ctrl+c" {
		return m, tea.Quit
	}

	switch m.mode {
	case modeDeleteConfirm:
		return m.handleDeleteKey(msg)
	case modeAdd:
		return m.handleAddKey(msg)
	case modeDetail:
		if key.Matches(msg, m.keys.Cancel) || key.Matches(msg, m.keys.Detail) {
			m.mode = modeNormal
		}
		return m, nil
	}

	s := &m.sections[m.active]
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keys.Help):
		m.help.ShowAll = !m.help.ShowAll

	case key.Matches(msg, m.keys.Left):
		if s.cursor > 0 {
			s.cursor--
		}
	case key.Matches(msg, m.keys.Right):
		if s.cursor < len(s.items)-1 {
			s.cursor++
		}
	case key.Matches(msg, m.keys.Down):
		m.moveVertical(+1)
	case key.Matches(msg, m.keys.Up):
		m.moveVertical(-1)

	case key.Matches(msg, m.keys.Detail):
		if m.ActiveAsset() != nil {
			m.mode = modeDetail
		}

	case key.Matches(msg, m.keys.Add):
		m.mode = modeAdd
		m.textInput.Reset()
		m.textInput.Focus()
		return m, textinput.Blink

	case key.Matches(msg, m.keys.Delete):
		if a := m.ActiveAsset(); a != nil {
			m.pendingDelete = a.Symbol
			m.mode = modeDeleteConfirm
		} else {
			m.setStatus("Nothing selected to remove", false)
		}

	case key.Matches(msg, m.keys.MoveEarlier):
		m.moveItem(-1)
	case key.Matches(msg, m.keys.MoveLater):
		m.moveItem(+1)
	case key.Matches(msg, m.keys.Sort):
		m.sortActive()

	case key.Matches(msg, m.keys.Refresh):
		if m.refreshing {
			m.setStatus("Refresh already in progress", false)
		} else {
			m.refreshing = true
			m.setStatus("Refreshing prices…", false)
			return m, m.refreshCmd()
		}
	}
	return m, nil
}

func (m Model) handleDeleteKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Confirm):
		s := &m.sections[m.active]
		if idx := indexOf(s.items, m.pendingDelete); idx >= 0 {
			removed := s.items[idx]
			s.items = append(s.items[:idx], s.items[idx+1:]...)
			if s.cursor >= len(s.items) && s.cursor > 0 {
				s.cursor--
			}
			if len(s.items) == 0 {
				m.focusNonEmpty()
			}
			if err := m.persist(); err != nil {
				m.setStatus(fmt.Sprintf("Removed %s, but saving config failed: %v", removed.Display, err), true)
			} else {
				m.setStatus("Removed "+removed.Display, false)
			}
		}
		m.mode, m.pendingDelete = modeNormal, ""
	case key.Matches(msg, m.keys.Cancel):
		m.mode, m.pendingDelete = modeNormal, ""
		m.setStatus("Removal cancelled", false)
	}
	return m, nil
}

func (m Model) handleAddKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		val := strings.TrimSpace(m.textInput.Value())
		if err := fetcher.ValidateInput(val); err != nil {
			m.setStatus(err.Error(), true)
			return m, nil // stay in the prompt so the input can be corrected
		}
		m.textInput.Reset()
		m.mode = modeNormal
		m.setStatus(fmt.Sprintf("Looking up %s…", strings.ToUpper(val)), false)
		return m, m.addCmd(val)
	case "esc":
		m.textInput.Reset()
		m.mode = modeNormal
		return m, nil
	}
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

// focusNonEmpty moves focus to a section that has items, if the active one is empty.
func (m *Model) focusNonEmpty() {
	if len(m.sections[m.active].items) > 0 {
		return
	}
	for i, s := range m.sections {
		if len(s.items) > 0 {
			m.active = i
			return
		}
	}
}

// moveVertical moves the cursor one row up/down, crossing into the neighbouring
// section at the grid edge while keeping the column where possible.
func (m *Model) moveVertical(dir int) {
	per := m.cardsPerRow()
	s := &m.sections[m.active]
	n := len(s.items)
	if n == 0 {
		return
	}
	col := s.cursor % per
	row, lastRow := s.cursor/per, (n-1)/per

	if dir > 0 {
		if row < lastRow {
			s.cursor = min(s.cursor+per, n-1)
			return
		}
		for i := m.active + 1; i < len(m.sections); i++ {
			if t := &m.sections[i]; len(t.items) > 0 {
				m.active, t.cursor = i, min(col, len(t.items)-1)
				return
			}
		}
		return
	}

	if row > 0 {
		s.cursor -= per
		return
	}
	for i := m.active - 1; i >= 0; i-- {
		if t := &m.sections[i]; len(t.items) > 0 {
			lastRowStart := (len(t.items) - 1) / per * per
			m.active, t.cursor = i, min(lastRowStart+col, len(t.items)-1)
			return
		}
	}
}

// moveItem swaps the selected asset with its neighbour and saves the new order.
func (m *Model) moveItem(dir int) {
	s := &m.sections[m.active]
	j := s.cursor + dir
	if len(s.items) == 0 || j < 0 || j >= len(s.items) {
		return
	}
	s.items[s.cursor], s.items[j] = s.items[j], s.items[s.cursor]
	s.cursor = j
	if err := m.persist(); err != nil {
		m.setStatus("Saving order failed: "+err.Error(), true)
	}
}

// sortActive orders the active section by 24h change, best first, keeping the selection.
func (m *Model) sortActive() {
	s := &m.sections[m.active]
	if len(s.items) < 2 {
		return
	}
	var selected string
	if a := m.ActiveAsset(); a != nil {
		selected = a.Symbol
	}
	sort.SliceStable(s.items, func(i, j int) bool { return s.items[i].Change24h > s.items[j].Change24h })
	if idx := indexOf(s.items, selected); idx >= 0 {
		s.cursor = idx
	}
	if err := m.persist(); err != nil {
		m.setStatus("Saving order failed: "+err.Error(), true)
		return
	}
	m.setStatus(s.title+" sorted by 24h change", false)
}

const (
	minCardWidth = 30
	maxCardWidth = 44
	cardChrome   = 4 // border (2) + right margin (2)
)

// cardsPerRow is how many cards fit across the terminal (1–4).
func (m Model) cardsPerRow() int {
	w := m.width
	if w <= 0 {
		w = defaultWidth
	}
	return clampInt(w/(minCardWidth+cardChrome), 1, 4)
}

// cardWidth stretches cards to use the available width.
func (m Model) cardWidth() int {
	w := m.width
	if w <= 0 {
		w = defaultWidth
	}
	return clampInt(w/m.cardsPerRow()-cardChrome, 24, maxCardWidth)
}

// WithNotice returns the model with a startup message (e.g. a config warning) in the status line.
func (m Model) WithNotice(msg string, isErr bool) Model {
	m.statusMsg, m.statusErr = msg, isErr
	m.statusUntil = time.Now().Add(4 * statusDuration)
	return m
}
