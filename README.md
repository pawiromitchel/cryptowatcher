# CryptoWatcher

A real-time Terminal User Interface (TUI) stocks and cryptocurrency dashboard inspired by the macOS Stocks widget design, built in Go.

![CI](https://github.com/pawiromitchel/cryptowatcher/actions/workflows/ci.yml/badge.svg)
![Go Version](https://img.shields.io/badge/Go-1.24%2B-blue)
![License](https://img.shields.io/badge/license-MIT-green)

<img width="2257" height="2215" alt="bettershot_1788104229515" src="https://github.com/user-attachments/assets/7a3d5e5f-c2ff-4676-bc89-67fe9c3e9bf2" />

---

## Features

- **Widget cards** for every asset: direction arrow, symbol, name, market cap, 24h change, a high-resolution braille line chart and the spot price.
- **Honest data.** A missing value is shown as `—`. Nothing is estimated: no made-up market caps, no synthetic charts, no guessed highs/lows.
- **Resilient refresh.** If a provider fails or rate-limits you, the card keeps its last price and is flagged `stale HH:MM`; the header shows `LIVE`, `PARTIAL n/m` or `OFFLINE`. Providers back off automatically after an HTTP 429.
- **Smart ticker routing.** Type any symbol and it is resolved to the right asset class:
  - known stocks go straight to Yahoo Finance;
  - crypto is looked up on Coinbase, then CoinGecko, then DexScreener (on-chain tokens);
  - an unlisted ticker (e.g. `AMD`) is probed on Coinbase, then Yahoo, then CoinGecko/DexScreener, and only **exact** symbol matches are accepted — it will never silently show a different asset.
- **Detail view** (`enter`): large chart, 24h range bar, open, volume, market cap, data source and last-update time.
- **Reorder and sort** your lists (`[` / `]`, `s`); the order is saved.
- **Responsive layout**: 1–4 cards per row depending on terminal width, with scrolling on short terminals.
- **Safe configuration**: atomic writes; a corrupt `config.json` is moved to `config.json.bak` and you are told, never silently overwritten.

---

## Installation

### Via Homebrew (macOS & Linux)

This repository is its own Homebrew tap, so there is no separate tap to maintain:

```bash
brew tap pawiromitchel/cryptowatcher https://github.com/pawiromitchel/cryptowatcher
brew install cryptowatcher
```

The formula installs the prebuilt binary from the latest GitHub release (macOS and Linux, arm64 and x86_64), so no Go toolchain is needed. To update later:

```bash
brew upgrade cryptowatcher
```

> Recent Homebrew versions refuse to load formulae from third-party taps until you trust them. If `brew install` tells you the tap is untrusted, run `brew trust pawiromitchel/cryptowatcher` and retry.

Coming from the old `pawiromitchel/tap`? Run `brew uninstall cryptowatcher && brew untap pawiromitchel/tap`, then install from the new tap above.

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

Prerequisites: Go 1.24+ installed.

```bash
git clone git@github.com:pawiromitchel/cryptowatcher.git
cd cryptowatcher
make build
./cryptowatcher
```

---

---

## Keybindings

| Key | Action |
| --- | --- |
| `←` `↓` `↑` `→` / `h` `j` `k` `l` | Move between cards (rows wrap into the next section) |
| `enter` | Open / close the detail view |
| `a` / `+` | Add a ticker |
| `d` / `x` | Remove the selected ticker (asks for confirmation) |
| `[` / `]` | Move the selected card earlier / later |
| `s` | Sort the current section by 24h change |
| `r` | Refresh now |
| `?` | Toggle full help |
| `q` / `Ctrl+C` | Quit (`q` closes the detail view first) |

---

## Configuration

Settings and watchlists are stored in `config.json` inside a `cryptowatcher` folder in your user config directory (`~/Library/Application Support` on macOS, `~/.config` on Linux, or `$XDG_CONFIG_HOME` if set). Run `cryptowatcher -config-path` to print the exact path:

```json
{
  "crypto_pairs": ["BTC-USD", "ETH-USD", "SOL-USD"],
  "stock_pairs": ["SPY", "TSLA", "GOOGL", "AAPL"],
  "refresh_interval": 15
}
```

`refresh_interval` is in seconds (default 15, minimum 5). The lists are edited from the UI.

CLI flags:

| Flag | Description |
| --- | --- |
| `-interval <seconds>` | Override the refresh interval (minimum 5) |
| `-mock` | Use deterministic synthetic data (offline demo) |
| `-config-path` | Print the configuration file path and exit |
| `-version` | Print the version and exit |

---

## Data sources and limits

| Asset | Provider | Notes |
| --- | --- | --- |
| Major crypto | Coinbase Exchange | Price, 24h stats, hourly candles (cached 5 min) |
| Long-tail crypto | CoinGecko (free tier) | Quotes cached 60s; ~30 calls/min limit, handled with backoff |
| On-chain tokens | DexScreener | Price/volume only; no high/low or chart |
| Stocks, ETFs | Yahoo Finance (unofficial endpoint) | May change without notice; no market cap |
| Market cap for crypto | CoinGecko | Cached 10 min |

This is an informational tool, not investment advice. Prices may be delayed or unavailable.

---

## Development

```bash
make test        # go test -race ./...  (unit + e2e)
make test-e2e    # end-to-end only, verbose
make cover       # unit-test coverage
make run-mock    # run the UI with synthetic data
make release     # cross-compile all targets into dist/
./scripts/test-release.sh   # test the formula tooling
```

### Releasing

Push a tag: `git tag -a vX.Y.Z -m vX.Y.Z && git push origin vX.Y.Z`. The Release workflow tests, cross-compiles, publishes the GitHub release, then commits the updated Homebrew formula (new version and checksums) to `main`. If only the formula step fails, re-run it from the Actions tab with **Run workflow** and the tag.

### Test layers

- **Unit** (`internal/...`): symbol routing, providers (against in-process fake APIs), caching and rate-limit backoff, config safety, UI state machine and rendering.
- **End-to-end** (`e2e/`): the real UI, router and config file driven against fake provider servers — startup, add/remove/reorder/sort, outages and recovery, rate limiting, restart persistence — plus the compiled binary run on a real pseudo-terminal.

The fake providers live in `internal/fakeapi` and can inject HTTP errors and rate limits.

### Project structure

```
cmd/cryptowatcher/   entrypoint and flag handling
internal/config/     load/save of the watchlist (atomic, corruption-safe)
internal/fetcher/    Coinbase, CoinGecko/DexScreener, Yahoo providers; routing MultiFetcher; mock
internal/fakeapi/    in-process fake provider servers for tests
internal/model/      shared types (Asset, Config)
internal/ui/         Bubble Tea model, views, cards, charts, keymap
e2e/                 end-to-end tests
scripts/             Homebrew formula template and release tooling (run by the release workflow)
```

---

## License

MIT — see [LICENSE](LICENSE).
