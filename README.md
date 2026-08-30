# CryptoWatcher

A real-time Terminal User Interface (TUI) stocks and cryptocurrency dashboard inspired by the macOS Stocks widget design, built in Go.

![Go Version](https://img.shields.io/badge/Go-1.22%2B-blue)
![License](https://img.shields.io/badge/license-MIT-green)

---

## Features

- **macOS Stocks-Style Widget Cards**: Clean, rounded widget cards for every monitored asset featuring:
  - Up/Down indicator (`▲`/`▼`) with ticker symbol & full asset name.
  - Formatted Market Cap badge (e.g. `$1.57T`, `$297.3B`, `$3.45T`) and color-coded 24h percentage change (+/-).
  - **Inline High-Definition Braille Mini Line Chart** ($3 \times 4 = 12$ sub-pixel vertical resolution) with reference dotted baseline (`┄┄┄┄┄┄`).
  - Prominent spot price formatted to currency precision.
- **Categorized Dashboard Layout**:
  - **🪙 Cryptocurrency**: Bitcoin, Ethereum, Solana, Dogecoin, Avalanche, etc.
  - **📈 Stocks & Equities**: S&P 500 (`SPY`), Tesla (`TSLA`), Alphabet (`GOOGL`), Apple (`AAPL`), NVIDIA (`NVDA`), etc.
- **Composite Multi-Feed Engine (`MultiFetcher`)**:
  - **Coinbase Exchange REST API**: High-frequency orderbook data and 24-hour hourly candle series for crypto.
  - **Equity Market Feed**: Intraday 15-minute price quotes and historical trends for US stocks and ETFs.
  - **Smart Routing & Auto-Categorization**: Enter any symbol (e.g. `NVDA`, `DOGE`, `AAPL`, `SOL`) and the engine automatically routes, fetches, and appends it to the matching category.
- **2D Interactive Grid Navigation**: Navigate horizontally and vertically across widget cards and category sections using arrow keys or Vim keys (`h`/`j`/`k`/`l`).
- **Persistent Configuration**: Watchlists automatically persist across runs in `~/.config/cryptowatcher/config.json`.
- **Top Summary Dashboard**: Real-time asset count, top 24h gainer, top 24h loser, and feed connection status.

---

## Installation

### Prerequisites
- Go 1.22+ installed on your system.

### Build from Source

```bash
git clone git@github.com:pawiromitchel/cryptowatcher.git
cd cryptowatcher
go build -o cryptowatcher ./cmd/cryptowatcher
./cryptowatcher
```

---

## Keybindings

| Key | Action |
| --- | --- |
| `←` / `h` | Move selection left |
| `→` / `l` | Move selection right |
| `↑` / `k` | Move selection up (within row or across category sections) |
| `↓` / `j` | Move selection down (within row or across category sections) |
| `a` / `+` | Add new crypto pair or stock ticker (modal input) |
| `d` / `x` | Delete selected widget card |
| `r` | Force instant refresh across all tickers |
| `q` / `Ctrl+C` | Quit application |

---

## Configuration

Settings and watchlists are stored in JSON format at `~/.config/cryptowatcher/config.json`:

```json
{
  "crypto_pairs": [
    "BTC-USD",
    "ETH-USD",
    "SOL-USD"
  ],
  "stock_pairs": [
    "SPY",
    "TSLA",
    "GOOGL",
    "AAPL"
  ],
  "refresh_interval": 5
}
```

CLI flags:
* `-interval <seconds>`: Override live refresh polling interval (default: `5s`).
* `-mock`: Run with synthetic data generator (useful for offline demo/testing).
* `-config-path`: Print absolute path to configuration file and exit.

---

## Testing

Run unit tests across all packages:

```bash
go test -v ./...
```

Run in mock mode:

```bash
./cryptowatcher -mock
```

---

## Project Structure

```
cryptowatcher/
├── cmd/
│   └── cryptowatcher/
│       └── main.go          # Application entrypoint & multi-provider wiring
├── internal/
│   ├── config/              # Persistent JSON configuration manager
│   ├── fetcher/             # Data providers (Coinbase, Equity, MultiFetcher, Mock)
│   ├── model/               # Core domain models (CryptoPair, AssetType, Config)
│   └── ui/                  # Bubble Tea TUI components, widget card renderer & styles
└── go.mod
```

---

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
