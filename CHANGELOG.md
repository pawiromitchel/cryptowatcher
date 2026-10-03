# Changelog

## Unreleased

### Changed
- Homebrew: the repo is now its own tap (`brew tap pawiromitchel/cryptowatcher <repo url>`). The formula installs the prebuilt release binaries (no Go toolchain needed) and is updated automatically by the release workflow after each tag. The separate `pawiromitchel/tap` is deprecated.
- README now documents the real config location per OS.

## v1.2.0 - 2026-10-01

### Fixed
- A failed or rate-limited refresh no longer blanks cards; the last good price is kept and marked stale.
- Refresh results are merged by symbol, so adding/removing a ticker while a refresh is in flight is no longer undone.
- Unlisted stock tickers (e.g. `AMD`) resolve to the stock instead of a similarly named crypto token; symbol matching is now exact everywhere.
- A corrupt `config.json` is backed up to `config.json.bak` instead of being silently overwritten; emptied watchlists stay empty; config writes are atomic.
- Truncation of long names/symbols is width-aware (no more broken multi-byte text).
- `go.mod` / README / CI now agree on the Go version.

### Changed
- Market caps, highs/lows and charts are never fabricated; unknown values show `—`. Crypto market caps now come from CoinGecko.
- Default refresh interval is 15s (minimum 5s) with provider backoff on HTTP 429 and response caching.
- `PythFetcher` renamed `YahooFetcher`; `CryptoPair` renamed `Asset`.
- Redesigned header, responsive card layout, scrolling on short terminals, `bubbles/help` footer.

### Added
- Detail view, card reordering (`[` `]`), sort by 24h change (`s`), `?` help, `-version` flag.
- Input validation for tickers.
- CI workflow, race-enabled tests, LICENSE, `make` targets for vet/fmt/cover/e2e.
- Fake provider servers (`internal/fakeapi`) and an end-to-end test suite, including a PTY test of the real binary.

### Removed
- Unused code (`RenderSparkline`, dead fields and keybinding plumbing).
