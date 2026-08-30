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
)

// Messages
type tickMsg time.Time
type fetchedPricesMsg struct {
	crypto []model.CryptoPair
	stocks []model.CryptoPair
	err    error
}
type addedPairMsg struct {
	pair      model.CryptoPair
	assetType model.AssetType
	err       error
}

// Model represents the Bubble Tea application state for the Widget Dashboard.
type Model struct {
	cfg          *model.Config
	fetcher      fetcher.PriceFetcher
	cryptoPairs  []model.CryptoPair
	stockPairs   []model.CryptoPair
	sectionIndex int // 0 = Crypto, 1 = Stocks
	cryptoCursor int
	stockCursor  int
	mode         viewMode
	textInput    textinput.Model
	keys         KeyMap
	statusMsg    string
	err          error
	loading      bool
	lastRefresh  time.Time
	width        int
	height       int
}

// NewModel initializes the UI model with crypto and stock watchlists.
func NewModel(cfg *model.Config, pf fetcher.PriceFetcher) Model {
	ti := textinput.New()
	ti.Placeholder = "e.g. BTC/USD, DOGE, SPY, TSLA, AAPL"
	ti.CharLimit = 16
	ti.Width = 32

	initialCrypto := make([]model.CryptoPair, len(cfg.CryptoPairs))
	for i, s := range cfg.CryptoPairs {
		sym, disp := fetcher.NormalizeSymbol(s)
		initialCrypto[i] = model.CryptoPair{
			Symbol:  sym,
			Display: disp,
			Name:    fetcher.LookupAssetName(sym),
			Type:    model.AssetCrypto,
		}
	}

	initialStocks := make([]model.CryptoPair, len(cfg.StockPairs))
	for i, s := range cfg.StockPairs {
		sym, disp := fetcher.NormalizeSymbol(s)
		initialStocks[i] = model.CryptoPair{
			Symbol:  sym,
			Display: disp,
			Name:    fetcher.LookupAssetName(sym),
			Type:    model.AssetStock,
		}
	}

	return Model{
		cfg:          cfg,
		fetcher:      pf,
		cryptoPairs:  initialCrypto,
		stockPairs:   initialStocks,
		sectionIndex: 0,
		cryptoCursor: 0,
		stockCursor:  0,
		mode:         modeNormal,
		textInput:    ti,
		keys:         DefaultKeyMap(),
		loading:      true,
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

// fetchPricesCmd asynchronously fetches prices for all watched crypto and stock pairs.
func (m Model) fetchPricesCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
		defer cancel()

		cryptoSymbols := make([]string, len(m.cryptoPairs))
		for i, p := range m.cryptoPairs {
			cryptoSymbols[i] = p.Symbol
		}

		stockSymbols := make([]string, len(m.stockPairs))
		for i, p := range m.stockPairs {
			stockSymbols[i] = p.Symbol
		}

		fetchedCrypto, err1 := m.fetcher.FetchPrices(ctx, cryptoSymbols)
		fetchedStocks, err2 := m.fetcher.FetchPrices(ctx, stockSymbols)

		var err error
		if err1 != nil {
			err = err1
		} else if err2 != nil {
			err = err2
		}

		return fetchedPricesMsg{
			crypto: fetchedCrypto,
			stocks: fetchedStocks,
			err:    err,
		}
	}
}

// addPairCmd asynchronously fetches data for a single newly added ticker.
func (m Model) addPairCmd(rawInput string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
		defer cancel()

		assetType := fetcher.DetectAssetType(rawInput)
		pair, err := m.fetcher.FetchPair(ctx, rawInput)
		if err == nil {
			pair.Type = assetType
		}
		return addedPairMsg{pair: pair, assetType: assetType, err: err}
	}
}

// AllPairs returns combined slice of all monitored pairs for charts and stats.
func (m Model) AllPairs() []model.CryptoPair {
	res := make([]model.CryptoPair, 0, len(m.cryptoPairs)+len(m.stockPairs))
	res = append(res, m.cryptoPairs...)
	res = append(res, m.stockPairs...)
	return res
}

// ActivePair returns the currently highlighted pair.
func (m Model) ActivePair() *model.CryptoPair {
	if m.sectionIndex == 0 {
		if len(m.cryptoPairs) > 0 && m.cryptoCursor < len(m.cryptoPairs) {
			return &m.cryptoPairs[m.cryptoCursor]
		}
	} else {
		if len(m.stockPairs) > 0 && m.stockCursor < len(m.stockPairs) {
			return &m.stockPairs[m.stockCursor]
		}
	}
	return nil
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
		} else {
			if len(msg.crypto) > 0 {
				m.cryptoPairs = msg.crypto
			}
			if len(msg.stocks) > 0 {
				m.stockPairs = msg.stocks
			}
			m.err = nil
		}

	case addedPairMsg:
		m.loading = false
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Error adding ticker: %v", msg.err)
		} else {
			if msg.assetType == model.AssetStock {
				exists := false
				for i, p := range m.stockPairs {
					if p.Symbol == msg.pair.Symbol {
						m.stockPairs[i] = msg.pair
						exists = true
						break
					}
				}
				if !exists {
					m.stockPairs = append(m.stockPairs, msg.pair)
					m.cfg.StockPairs = append(m.cfg.StockPairs, msg.pair.Symbol)
					_ = config.Save(m.cfg)
				}
				m.sectionIndex = 1
				m.stockCursor = len(m.stockPairs) - 1
			} else {
				exists := false
				for i, p := range m.cryptoPairs {
					if p.Symbol == msg.pair.Symbol {
						m.cryptoPairs[i] = msg.pair
						exists = true
						break
					}
				}
				if !exists {
					m.cryptoPairs = append(m.cryptoPairs, msg.pair)
					m.cfg.CryptoPairs = append(m.cfg.CryptoPairs, msg.pair.Symbol)
					_ = config.Save(m.cfg)
				}
				m.sectionIndex = 0
				m.cryptoCursor = len(m.cryptoPairs) - 1
			}
			m.statusMsg = fmt.Sprintf("Added %s (%s) successfully", msg.pair.Display, msg.pair.Name)
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

		// 2D Grid & Normal Mode Keybindings
		cardsPerRow := m.calculateCardsPerRow()

		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "left", "h":
			if m.sectionIndex == 0 {
				if m.cryptoCursor > 0 {
					m.cryptoCursor--
				}
			} else {
				if m.stockCursor > 0 {
					m.stockCursor--
				}
			}

		case "right", "l":
			if m.sectionIndex == 0 {
				if m.cryptoCursor < len(m.cryptoPairs)-1 {
					m.cryptoCursor++
				}
			} else {
				if m.stockCursor < len(m.stockPairs)-1 {
					m.stockCursor++
				}
			}

		case "down", "j":
			if m.sectionIndex == 0 {
				nextIdx := m.cryptoCursor + cardsPerRow
				if nextIdx < len(m.cryptoPairs) {
					m.cryptoCursor = nextIdx
				} else if len(m.stockPairs) > 0 {
					m.sectionIndex = 1
					if m.stockCursor >= len(m.stockPairs) {
						m.stockCursor = 0
					}
				}
			} else {
				nextIdx := m.stockCursor + cardsPerRow
				if nextIdx < len(m.stockPairs) {
					m.stockCursor = nextIdx
				}
			}

		case "up", "k":
			if m.sectionIndex == 1 {
				prevIdx := m.stockCursor - cardsPerRow
				if prevIdx >= 0 {
					m.stockCursor = prevIdx
				} else if len(m.cryptoPairs) > 0 {
					m.sectionIndex = 0
					if m.cryptoCursor >= len(m.cryptoPairs) {
						m.cryptoCursor = len(m.cryptoPairs) - 1
					}
				}
			} else {
				prevIdx := m.cryptoCursor - cardsPerRow
				if prevIdx >= 0 {
					m.cryptoCursor = prevIdx
				}
			}

		case "a", "+":
			m.mode = modeAdd
			m.textInput.Focus()
			return m, textinput.Blink

		case "d", "x", "delete":
			if m.sectionIndex == 0 && len(m.cryptoPairs) > 0 {
				removed := m.cryptoPairs[m.cryptoCursor]
				m.cryptoPairs = append(m.cryptoPairs[:m.cryptoCursor], m.cryptoPairs[m.cryptoCursor+1:]...)
				newPairs := make([]string, len(m.cryptoPairs))
				for i, p := range m.cryptoPairs {
					newPairs[i] = p.Symbol
				}
				m.cfg.CryptoPairs = newPairs
				_ = config.Save(m.cfg)
				if m.cryptoCursor >= len(m.cryptoPairs) && m.cryptoCursor > 0 {
					m.cryptoCursor--
				}
				m.statusMsg = fmt.Sprintf("Removed %s", removed.Display)
			} else if m.sectionIndex == 1 && len(m.stockPairs) > 0 {
				removed := m.stockPairs[m.stockCursor]
				m.stockPairs = append(m.stockPairs[:m.stockCursor], m.stockPairs[m.stockCursor+1:]...)
				newPairs := make([]string, len(m.stockPairs))
				for i, p := range m.stockPairs {
					newPairs[i] = p.Symbol
				}
				m.cfg.StockPairs = newPairs
				_ = config.Save(m.cfg)
				if m.stockCursor >= len(m.stockPairs) && m.stockCursor > 0 {
					m.stockCursor--
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

func (m Model) calculateCardsPerRow() int {
	width := m.width
	if width <= 0 {
		width = 110
	}
	// Each card is ~32 width including margins
	cards := width / 32
	if cards < 1 {
		cards = 1
	}
	if cards > 4 {
		cards = 4
	}
	return cards
}
