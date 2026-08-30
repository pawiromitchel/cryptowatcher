package fetcher

import (
	"context"
	"fmt"
	"strings"

	"cryptowatcher/internal/model"
)

// PriceFetcher defines the contract for retrieving market data for crypto and equities.
type PriceFetcher interface {
	FetchPrices(ctx context.Context, symbols []string) ([]model.CryptoPair, error)
	FetchPair(ctx context.Context, symbol string) (model.CryptoPair, error)
}

// Known asset names dictionary for professional display
var assetNames = map[string]string{
	"BTC":    "Bitcoin USD",
	"ETH":    "Ethereum USD",
	"SOL":    "Solana USD",
	"DOGE":   "Dogecoin USD",
	"ADA":    "Cardano USD",
	"AVAX":   "Avalanche USD",
	"LINK":   "Chainlink USD",
	"DOT":    "Polkadot USD",
	"XRP":    "Ripple USD",
	"SPY":    "S&P 500 ETF",
	"S&P":    "S&P 500 Index",
	"TSLA":   "Tesla Inc",
	"GOOGL":  "Alphabet Inc",
	"GOOG":   "Alphabet Inc",
	"AAPL":   "Apple Inc",
	"AAPLC":  "Apple Inc (Base)",
	"NVDA":   "NVIDIA Corp",
	"NVDAC":  "NVIDIA Corp (Base)",
	"MSFT":   "Microsoft Corp",
	"MSFTC":  "Microsoft Corp (Base)",
	"AMZN":   "Amazon.com Inc",
	"AMZNC":  "Amazon.com (Base)",
	"META":   "Meta Platforms",
	"METAC":  "Meta Platforms (Base)",
	"COIN":   "Coinbase Global",
	"COINC":  "Coinbase (Base)",
	"QQQ":    "Invesco QQQ Trust",
}

// Known stock tickers set
var knownStockTickers = map[string]bool{
	"SPY": true, "S&P": true, "TSLA": true, "GOOGL": true, "GOOG": true,
	"AAPL": true, "AAPLC": true, "NVDA": true, "NVDAC": true,
	"MSFT": true, "MSFTC": true, "AMZN": true, "AMZNC": true,
	"META": true, "METAC": true, "COIN": true, "COINC": true, "QQQ": true,
}

// DetectAssetType checks if an input symbol belongs to Stocks or Crypto.
func DetectAssetType(input string) model.AssetType {
	cleaned := strings.ToUpper(strings.TrimSpace(input))
	cleaned = strings.ReplaceAll(cleaned, "/", "-")
	cleaned = strings.ReplaceAll(cleaned, "_", "-")
	parts := strings.Split(cleaned, "-")
	base := parts[0]

	if knownStockTickers[base] {
		return model.AssetStock
	}
	return model.AssetCrypto
}

// LookupAssetName returns the human-readable asset name.
func LookupAssetName(symbol string) string {
	parts := strings.Split(symbol, "-")
	base := strings.ToUpper(parts[0])
	if name, found := assetNames[base]; found {
		return name
	}
	return base
}

// NormalizeSymbol normalizes raw user input into standard API symbol format and display format.
func NormalizeSymbol(input string) (symbol string, display string) {
	cleaned := strings.ToUpper(strings.TrimSpace(input))
	if cleaned == "" {
		return "", ""
	}

	cleaned = strings.ReplaceAll(cleaned, "/", "-")
	cleaned = strings.ReplaceAll(cleaned, "_", "-")

	// Stock tickers (e.g. SPY, TSLA, AAPL) display cleanly as SPY, TSLA, AAPL
	parts := strings.Split(cleaned, "-")
	base := parts[0]

	if knownStockTickers[base] && len(parts) == 1 {
		symbol = base
		if base == "SPY" || base == "S&P" {
			display = "S&P 500"
		} else {
			display = base
		}
		return symbol, display
	}

	if !strings.Contains(cleaned, "-") {
		symbol = fmt.Sprintf("%s-USD", cleaned)
	} else {
		symbol = cleaned
	}

	p := strings.Split(symbol, "-")
	if len(p) == 2 {
		display = fmt.Sprintf("%s/%s", p[0], p[1])
	} else {
		display = symbol
	}

	return symbol, display
}

func extractBaseTicker(symbol string) string {
	parts := strings.Split(symbol, "-")
	return strings.ToUpper(parts[0])
}
