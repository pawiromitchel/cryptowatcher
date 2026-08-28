package fetcher

import (
	"context"
	"fmt"
	"strings"

	"cryptowatcher/internal/model"
)

// PriceFetcher defines the contract for retrieving crypto market data.
type PriceFetcher interface {
	FetchPrices(ctx context.Context, symbols []string) ([]model.CryptoPair, error)
	FetchPair(ctx context.Context, symbol string) (model.CryptoPair, error)
}

// NormalizeSymbol normalizes raw user input into standard API symbol format (e.g. BTC-USD)
// and user display format (e.g. BTC/USD).
func NormalizeSymbol(input string) (symbol string, display string) {
	cleaned := strings.ToUpper(strings.TrimSpace(input))
	if cleaned == "" {
		return "", ""
	}

	// Replace slashes or underscores with hyphens
	cleaned = strings.ReplaceAll(cleaned, "/", "-")
	cleaned = strings.ReplaceAll(cleaned, "_", "-")

	// If no pair separator exists, default quote currency to USD
	if !strings.Contains(cleaned, "-") {
		symbol = fmt.Sprintf("%s-USD", cleaned)
	} else {
		symbol = cleaned
	}

	parts := strings.Split(symbol, "-")
	if len(parts) == 2 {
		display = fmt.Sprintf("%s/%s", parts[0], parts[1])
	} else {
		display = symbol
	}

	return symbol, display
}
