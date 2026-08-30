package fetcher

import (
	"context"
	"fmt"
	"sync"

	"cryptowatcher/internal/model"
)

// MultiFetcher combines multiple PriceFetcher providers with fallback cascading.
type MultiFetcher struct {
	fetchers []PriceFetcher
}

// NewMultiFetcher initializes a composite PriceFetcher with the given providers in priority order.
func NewMultiFetcher(fetchers ...PriceFetcher) *MultiFetcher {
	return &MultiFetcher{
		fetchers: fetchers,
	}
}

// FetchPair attempts to retrieve market data for a symbol across providers in cascading order.
func (m *MultiFetcher) FetchPair(ctx context.Context, rawInput string) (model.CryptoPair, error) {
	var lastErr error
	var lastPair model.CryptoPair

	for _, f := range m.fetchers {
		pair, err := f.FetchPair(ctx, rawInput)
		if err == nil && pair.Price > 0 && pair.Err == nil {
			return pair, nil
		}
		lastPair = pair
		lastErr = err
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("no provider found for symbol %s", rawInput)
	}
	lastPair.Err = lastErr
	return lastPair, lastErr
}

// FetchPrices resolves market data concurrently for a slice of symbols across configured providers.
func (m *MultiFetcher) FetchPrices(ctx context.Context, symbols []string) ([]model.CryptoPair, error) {
	results := make([]model.CryptoPair, len(symbols))
	var wg sync.WaitGroup

	for i, raw := range symbols {
		wg.Add(1)
		go func(idx int, sym string) {
			defer wg.Done()
			pair, err := m.FetchPair(ctx, sym)
			if err != nil {
				pair.Err = err
			}
			results[idx] = pair
		}(i, raw)
	}

	wg.Wait()
	return results, nil
}
