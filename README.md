# CryptoWatcher

A lightweight, real-time Terminal User Interface (TUI) cryptocurrency watcher built in Go.

![Go Version](https://img.shields.io/badge/Go-1.27%2B-blue)
![License](https://img.shields.io/badge/license-MIT-green)

## Features

- **Live Market Data**: Monitors spot price ($ USD), 24-hour percentage change (+/-), 24h high/low, and 24h volume.
- **Dynamic Watchlist**: Add any cryptocurrency pair on the fly (e.g. `BTC/USD`, `ETH/USD`, `SOL/USD`, `DOGE`, `AVAX`).
- **Color-Coded Status**: Visual indicators for positive (+%) and negative (-%) price movements.
- **Persistent Configuration**: Watchlist automatically persists across application runs (`~/.config/cryptowatcher/config.json`).
- **Extensible Architecture**: Decoupled `PriceFetcher` interface supporting multiple exchange providers (defaulting to keyless Coinbase Exchange REST API).
- **Fast & Responsive TUI**: Powered by [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lip Gloss](https://github.com/charmbracelet/lipgloss).

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
| `↑` / `k` | Move selection up |
| `↓` / `j` | Move selection down |
| `a` / `+` | Add new cryptocurrency pair |
| `d` / `x` | Remove selected pair |
| `r` | Force refresh prices |
| `q` / `Ctrl+C` | Quit application |

---

## Testing

Run unit tests across all packages:

```bash
go test -v ./...
```

Run in mock mode (useful for offline testing):

```bash
./cryptowatcher -mock
```

---

## Project Structure

```
cryptowatcher/
├── cmd/
│   └── cryptowatcher/
│       └── main.go          # Application entrypoint
├── internal/
│   ├── config/              # Persistent JSON configuration manager
│   ├── fetcher/             # Exchange API clients (Coinbase, Mock)
│   ├── model/               # Core domain models
│   └── ui/                  # Bubble Tea TUI components & Lip Gloss styles
└── go.mod
```

---

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
