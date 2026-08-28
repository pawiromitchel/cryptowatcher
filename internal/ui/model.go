package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

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
	modeCandlestick
)

// Messages
type tickMsg time.Time
type fetchedPricesMsg struct {
	pairs []model.CryptoPair
	err   error
}
type addedPairMsg struct {
	pair model.CryptoPair
	err  error
}

// Model represents the Bubble Tea application state.
type Model struct {
	cfg         *model.Config
	fetcher     fetcher.PriceFetcher
	pairs       []model.CryptoPair
	cursor      int
	mode        viewMode
	textInput   textinput.Model
	keys        KeyMap
	statusMsg   string
	err         error
	loading     bool
	lastRefresh time.Time
	width       int
	height      int
}

// NewModel initializes the UI model with config and fetcher services.
func NewModel(cfg *model.Config, pf fetcher.PriceFetcher) Model {
	ti := textinput.New()
	ti.Placeholder = "e.g. BTC/USD, DOGE, AVAX"
	ti.CharLimit = 16
	ti.Width = 30

	initialPairs := make([]model.CryptoPair, len(cfg.Pairs))
	for i, s := range cfg.Pairs {
		sym, disp := fetcher.NormalizeSymbol(s)
		initialPairs[i] = model.CryptoPair{Symbol: sym, Display: disp}
	}

	return Model{
		cfg:       cfg,
		fetcher:   pf,
		pairs:     initialPairs,
		cursor:    0,
		mode:      modeNormal,
		textInput: ti,
		keys:      DefaultKeyMap(),
		loading:   true,
	}
}

// Init triggers initial price fetch and sets up periodic timer ticks.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.fetchPricesCmd(),
		m.tickCmd(),
	)
}

// tickCmd returns a command that triggers tickMsg after refresh interval.
func (m Model) tickCmd() tea.Cmd {
	interval := time.Duration(m.cfg.RefreshInterval) * time.Second
	if interval <= 0 {
		interval = 5 * time.Second
	}
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// fetchPricesCmd asynchronously fetches prices for all watched pairs.
func (m Model) fetchPricesCmd() tea.Cmd {
	return func() tea.Msg {
		symbols := make([]string, len(m.pairs))
		for i, p := range m.pairs {
			symbols[i] = p.Symbol
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		fetched, err := m.fetcher.FetchPrices(ctx, symbols)
		return fetchedPricesMsg{pairs: fetched, err: err}
	}
}

// addPairCmd asynchronously fetches data for a single newly added pair.
func (m Model) addPairCmd(rawInput string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		pair, err := m.fetcher.FetchPair(ctx, rawInput)
		return addedPairMsg{pair: pair, err: err}
	}
}

// Update handles state transitions based on incoming messages and keypresses.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tickMsg:
		cmds = append(cmds, m.fetchPricesCmd(), m.tickCmd())

	case fetchedPricesMsg:
		m.loading = false
		m.lastRefresh = time.Now()
		if msg.err != nil {
			m.err = msg.err
		} else if len(msg.pairs) > 0 {
			m.pairs = msg.pairs
			m.err = nil
		}

	case addedPairMsg:
		m.loading = false
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Error adding pair: %v", msg.err)
		} else {
			// Check for duplicates
			exists := false
			for i, p := range m.pairs {
				if p.Symbol == msg.pair.Symbol {
					m.pairs[i] = msg.pair
					exists = true
					break
				}
			}
			if !exists {
				m.pairs = append(m.pairs, msg.pair)
				m.cfg.Pairs = append(m.cfg.Pairs, msg.pair.Symbol)
				_ = config.Save(m.cfg)
			}
			m.cursor = len(m.pairs) - 1
			m.statusMsg = fmt.Sprintf("Added %s successfully", msg.pair.Display)
		}

	case tea.KeyMsg:
		if m.mode == modeAdd {
			switch msg.String() {
			case "enter":
				val := strings.TrimSpace(m.textInput.Value())
				if val != "" {
					m.loading = true
					m.statusMsg = fmt.Sprintf("Fetching %s...", val)
					cmds = append(cmds, m.addPairCmd(val))
				}
				m.textInput.Reset()
				m.mode = modeNormal
				return m, tea.Batch(cmds...)

			case "esc":
				m.textInput.Reset()
				m.mode = modeNormal
				return m, nil
			}

			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
		}

		if m.mode == modeCandlestick {
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit
			case "v", "c", "esc":
				m.mode = modeNormal
				return m, nil
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}
				return m, nil
			case "down", "j":
				if m.cursor < len(m.pairs)-1 {
					m.cursor++
				}
				return m, nil
			case "r":
				m.loading = true
				m.statusMsg = "Refreshing prices..."
				cmds = append(cmds, m.fetchPricesCmd())
				return m, tea.Batch(cmds...)
			}
		}

		// Normal Mode Keybindings
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.pairs)-1 {
				m.cursor++
			}

		case "v", "c":
			if len(m.pairs) > 0 {
				m.mode = modeCandlestick
			}

		case "a", "+":
			m.mode = modeAdd
			m.textInput.Focus()
			return m, textinput.Blink

		case "d", "x", "delete":
			if len(m.pairs) > 0 && m.cursor < len(m.pairs) {
				removed := m.pairs[m.cursor]
				m.pairs = append(m.pairs[:m.cursor], m.pairs[m.cursor+1:]...)

				// Update config
				newConfigPairs := make([]string, 0, len(m.pairs))
				for _, p := range m.pairs {
					newConfigPairs = append(newConfigPairs, p.Symbol)
				}
				m.cfg.Pairs = newConfigPairs
				_ = config.Save(m.cfg)

				if m.cursor >= len(m.pairs) && m.cursor > 0 {
					m.cursor--
				}
				m.statusMsg = fmt.Sprintf("Removed %s", removed.Display)
			}

		case "r":
			m.loading = true
			m.statusMsg = "Refreshing prices..."
			cmds = append(cmds, m.fetchPricesCmd())
		}
	}

	return m, tea.Batch(cmds...)
}
