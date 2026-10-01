package fetcher

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"cryptowatcher/internal/model"
)

// MarketCapSource is implemented by providers that can look up a market cap by ticker.
type MarketCapSource interface {
	MarketCap(ctx context.Context, base string) float64
}

// MultiFetcher routes each symbol to the right providers:
//
//   - known stocks            -> equity provider only
//   - known / quoted crypto   -> exchange, then aggregator
//   - anything else (unknown) -> exchange, equity, aggregator (first hit wins)
//
// Resolved symbols are remembered so the probing happens once.
type MultiFetcher struct {
	exchange   PriceFetcher
	equity     PriceFetcher
	aggregator PriceFetcher

	mu       sync.Mutex
	resolved map[string]model.AssetType
}

// NewMultiFetcher builds the router. Any provider may be nil.
func NewMultiFetcher(exchange, aggregator, equity PriceFetcher) *MultiFetcher {
	return &MultiFetcher{
		exchange:   exchange,
		equity:     equity,
		aggregator: aggregator,
		resolved:   map[string]model.AssetType{},
	}
}

func (m *MultiFetcher) kindOf(ctx context.Context, raw string) model.AssetType {
	if k := DetectAssetType(raw); k != "" {
		return k
	}
	if k, ok := ctx.Value(kindCtxKey{}).(model.AssetType); ok && k != "" {
		return k
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.resolved[clean(raw)]
}

func (m *MultiFetcher) providersFor(kind model.AssetType) []PriceFetcher {
	var list []PriceFetcher
	switch kind {
	case model.AssetStock:
		list = []PriceFetcher{m.equity}
	case model.AssetCrypto:
		list = []PriceFetcher{m.exchange, m.aggregator}
	default:
		list = []PriceFetcher{m.exchange, m.equity, m.aggregator}
	}
	out := list[:0:0]
	for _, p := range list {
		if p != nil {
			out = append(out, p)
		}
	}
	return out
}

// FetchPair resolves one symbol, trying providers in routing order.
func (m *MultiFetcher) FetchPair(ctx context.Context, raw string) (model.Asset, error) {
	if err := ValidateInput(raw); err != nil {
		return model.Asset{Err: err}, err
	}

	var errs []error
	var last model.Asset
	for _, p := range m.providersFor(m.kindOf(ctx, raw)) {
		if ctx.Err() != nil {
			break
		}
		asset, err := p.FetchPair(ctx, raw)
		if err == nil && asset.Err == nil && asset.Price > 0 {
			m.mu.Lock()
			m.resolved[clean(raw)] = asset.Type
			m.mu.Unlock()
			m.enrich(ctx, &asset)
			return asset, nil
		}
		if err != nil {
			errs = append(errs, err)
		}
		last = asset
	}

	err := errors.Join(errs...)
	if err == nil {
		err = fmt.Errorf("no provider could resolve %s", raw)
	} else if ctx.Err() == nil && len(errs) > 1 {
		err = fmt.Errorf("could not find %s: %s", strings.ToUpper(strings.TrimSpace(raw)), summarize(errs))
	}
	if last.Symbol == "" {
		last.Symbol, last.Display = NormalizeSymbol(raw)
	}
	last.Err = err
	return last, err
}

func summarize(errs []error) string {
	parts := make([]string, len(errs))
	for i, e := range errs {
		parts[i] = e.Error()
	}
	return strings.Join(parts, "; ")
}

// enrich fills in the market cap for crypto quotes whose provider lacks one.
func (m *MultiFetcher) enrich(ctx context.Context, a *model.Asset) {
	if a.Type != model.AssetCrypto || a.MarketCap > 0 {
		return
	}
	if src, ok := m.aggregator.(MarketCapSource); ok {
		a.MarketCap = src.MarketCap(ctx, extractBaseTicker(a.Symbol))
	}
}

// FetchPrices resolves many symbols concurrently, preserving order.
func (m *MultiFetcher) FetchPrices(ctx context.Context, symbols []string) ([]model.Asset, error) {
	return fetchAll(symbols, 6, func(s string) model.Asset {
		a, err := m.FetchPair(ctx, s)
		if err != nil {
			a.Err = err
		}
		return a
	}), nil
}

type kindCtxKey struct{}

// WithKind hints the asset kind of the symbols being fetched so MultiFetcher
// can skip probing providers that cannot serve them (e.g. a saved stock ticker
// that is not in the built-in list).
func WithKind(ctx context.Context, kind model.AssetType) context.Context {
	return context.WithValue(ctx, kindCtxKey{}, kind)
}
