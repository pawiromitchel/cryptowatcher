// Package fetcher retrieves market data from the supported providers and
// routes each symbol to the right one.
package fetcher

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"cryptowatcher/internal/model"
)

// PriceFetcher defines the contract for retrieving market data for crypto and equities.
type PriceFetcher interface {
	// FetchPrices resolves many symbols concurrently. The result has the same
	// length and order as symbols; a failed entry carries its error in Err.
	FetchPrices(ctx context.Context, symbols []string) ([]model.Asset, error)
	// FetchPair resolves a single symbol.
	FetchPair(ctx context.Context, symbol string) (model.Asset, error)
}

// cryptoNames maps well-known crypto tickers to display names.
var cryptoNames = map[string]string{
	"BTC": "Bitcoin", "ETH": "Ethereum", "SOL": "Solana", "DOGE": "Dogecoin",
	"ADA": "Cardano", "AVAX": "Avalanche", "LINK": "Chainlink", "DOT": "Polkadot",
	"XRP": "XRP", "LTC": "Litecoin", "BNB": "BNB", "HMM": "Thinking Cat",
	"PEPE": "Pepe", "WIF": "dogwifhat", "BONK": "Bonk", "SHIB": "Shiba Inu",
}

// stockNames maps well-known equity tickers to display names.
var stockNames = map[string]string{
	"SPY": "S&P 500 ETF", "S&P": "S&P 500 Index", "QQQ": "Invesco QQQ Trust",
	"TSLA": "Tesla Inc", "GOOGL": "Alphabet Inc", "GOOG": "Alphabet Inc",
	"AAPL": "Apple Inc", "NVDA": "NVIDIA Corp", "MSFT": "Microsoft Corp",
	"AMZN": "Amazon.com Inc", "META": "Meta Platforms", "COIN": "Coinbase Global",
}

// coinGeckoIDs seeds the ticker -> CoinGecko ID lookup so common coins need no search call.
var coinGeckoIDs = map[string]string{
	"BTC": "bitcoin", "ETH": "ethereum", "SOL": "solana", "DOGE": "dogecoin",
	"ADA": "cardano", "AVAX": "avalanche-2", "LINK": "chainlink", "DOT": "polkadot",
	"XRP": "ripple", "LTC": "litecoin", "BNB": "binancecoin",
	"HMM": "thinking-cat", "THINKING-CAT": "thinking-cat", "PEPE": "pepe",
	"BONK": "bonk", "WIF": "dogwifcoin", "SHIB": "shiba-inu", "FLOKI": "floki",
	"RENDER": "render-token", "SUI": "sui", "SEI": "sei-network", "NEAR": "near",
	"APT": "aptos", "INJ": "injective-protocol", "KAS": "kaspa", "TON": "the-open-network",
}

var inputPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9&._/-]{0,23}$`)

// ValidateInput rejects ticker input that can never be a valid symbol.
func ValidateInput(input string) error {
	input = strings.TrimSpace(input)
	if input == "" {
		return fmt.Errorf("enter a ticker symbol")
	}
	if !inputPattern.MatchString(input) {
		return fmt.Errorf("%q is not a valid ticker (letters, digits and - / . & only)", input)
	}
	return nil
}

func clean(input string) string {
	c := strings.ToUpper(strings.TrimSpace(input))
	c = strings.ReplaceAll(c, "/", "-")
	return strings.ReplaceAll(c, "_", "-")
}

// DetectAssetType classifies a symbol. It returns model.AssetStock or
// model.AssetCrypto when the symbol is recognised (or explicitly quoted, e.g.
// "ABC-USD"), and "" when the symbol is ambiguous and must be resolved by probing.
func DetectAssetType(input string) model.AssetType {
	c := clean(input)
	base := strings.SplitN(c, "-", 2)[0]
	if _, ok := stockNames[base]; ok && !strings.Contains(c, "-") {
		return model.AssetStock
	}
	if strings.Contains(c, "-") {
		return model.AssetCrypto
	}
	if _, ok := coinGeckoIDs[base]; ok {
		return model.AssetCrypto
	}
	if _, ok := cryptoNames[base]; ok {
		return model.AssetCrypto
	}
	return ""
}

// LookupAssetName returns the human-readable name for a symbol, or the bare ticker.
func LookupAssetName(symbol string) string {
	base := extractBaseTicker(symbol)
	if n, ok := stockNames[base]; ok {
		return n
	}
	if n, ok := cryptoNames[base]; ok {
		return n
	}
	return base
}

// NormalizeSymbol turns raw user input into an API symbol and a display label.
// Bare tickers that are not known stocks get a -USD quote suffix.
func NormalizeSymbol(input string) (symbol string, display string) {
	c := clean(input)
	if c == "" {
		return "", ""
	}

	if _, ok := stockNames[c]; ok {
		if c == "SPY" || c == "S&P" {
			return c, "S&P 500"
		}
		return c, c
	}

	if !strings.Contains(c, "-") {
		c += "-USD"
	}
	if p := strings.Split(c, "-"); len(p) == 2 {
		return c, p[0] + "/" + p[1]
	}
	return c, c
}

func extractBaseTicker(symbol string) string {
	return strings.SplitN(clean(symbol), "-", 2)[0]
}
