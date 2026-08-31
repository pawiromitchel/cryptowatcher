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
  - **Coinbase Exchange REST API**: High-frequency orderbook data and 24-hour hourly candle series for crypto pairs (`BTC`, `ETH`, `SOL`, `DOGE`).
  - **CoinGecko REST API**: Dynamic resolution for thousands of altcoins, meme tokens, and on-chain assets (e.g. `HMM` / Thinking Cat, `PEPE`, `WIF`, `BONK`, `RENDER`).
  - **Equity Market Feed**: Intraday 15-minute price quotes and historical trends for US stocks and ETFs (`SPY`, `TSLA`, `GOOGL`, `AAPL`, `NVDA`).
  - **Smart Routing & Auto-Categorization**: Enter any symbol (e.g. `HMM`, `NVDA`, `DOGE`, `AAPL`, `SOL`) and the engine automatically routes, fetches, and appends it to the matching category.
- **2D Interactive Grid Navigation**: Navigate horizontally and vertically across widget cards and category sections using arrow keys or Vim keys (`h`/`j`/`k`/`l`).
- **Persistent Configuration**: Watchlists automatically persist across runs in `~/.config/cryptowatcher/config.json`.
- **Top Summary Dashboard**: Real-time asset count, top 24h gainer, top 24h loser, and feed connection status.

---

## Installation

### Via Homebrew (macOS & Linux)

Install directly with one command:

```bash
brew install pawiromitchel/tap/cryptowatcher
```

Or tap first:

```bash
brew tap pawiromitchel/tap
brew install cryptowatcher
```

---

### Pre-built Binaries

Download the pre-compiled binary for your operating system from the [Latest GitHub Release](https://github.com/pawiromitchel/cryptowatcher/releases/latest):

| Platform | Architecture | Binary Package |
| :--- | :--- | :--- |
| **macOS** | Apple Silicon (M1/M2/M3/M4) | `cryptowatcher-darwin-arm64.tar.gz` |
| **macOS** | Intel x86_64 | `cryptowatcher-darwin-amd64.tar.gz` |
| **Linux** | x86_64 | `cryptowatcher-linux-amd64.tar.gz` |
| **Linux** | ARM64 / Raspberry Pi | `cryptowatcher-linux-arm64.tar.gz` |
| **Windows**| 64-bit | `cryptowatcher-windows-amd64.exe.zip` |

**Quick run (macOS / Linux):**
```bash
tar -xzf cryptowatcher-darwin-arm64.tar.gz
chmod +x cryptowatcher-darwin-arm64
./cryptowatcher-darwin-arm64
```

### Build from Source

Prerequisites: Go 1.22+ installed.

```bash
git clone git@github.com:pawiromitchel/cryptowatcher.git
cd cryptowatcher
make build
./cryptowatcher
```

---

## Keybindings

| Key | Action |
| --- | --- |
| `c` / `v` | Toggle Widget Card Mode: **Big Font Price** / **3-Row Line Chart** |
| `←` / `h` | Move selection left |
| `→` / `l` | Move selection right |
| `↑` / `k` | Move selection up (within row or across category sections) |
| `↓` / `j` | Move selection down (within row or across category sections) |
| `a` / `+` | Add new crypto pair or stock ticker (interactive modal) |
| `d` / `x` | Remove selected widget card (with confirmation prompt) |
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
