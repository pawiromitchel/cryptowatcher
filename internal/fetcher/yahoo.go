package fetcher

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"strings"
	"time"

	"cryptowatcher/internal/model"
)

const defaultYahooBaseURL = "https://query1.finance.yahoo.com"

// YahooChartResponse is the Yahoo Finance v8 chart payload (only the fields we use).
type YahooChartResponse struct {
	Chart struct {
		Result []struct {
			Meta struct {
				InstrumentType             string  `json:"instrumentType"`
				RegularMarketPrice         float64 `json:"regularMarketPrice"`
				RegularMarketChangePercent float64 `json:"regularMarketChangePercent"`
				RegularMarketDayHigh       float64 `json:"regularMarketDayHigh"`
				RegularMarketDayLow        float64 `json:"regularMarketDayLow"`
				RegularMarketVolume        float64 `json:"regularMarketVolume"`
				ChartPreviousClose         float64 `json:"chartPreviousClose"`
				LongName                   string  `json:"longName"`
				ShortName                  string  `json:"shortName"`
			} `json:"meta"`
			Indicators struct {
				Quote []struct {
					Close []float64 `json:"close"`
				} `json:"quote"`
			} `json:"indicators"`
		} `json:"result"`
		Error *struct {
			Code        string `json:"code"`
			Description string `json:"description"`
		} `json:"error"`
	} `json:"chart"`
}

// YahooFetcher implements PriceFetcher for stocks, ETFs and indices using
// Yahoo Finance's public chart endpoint (unofficial; may change without notice).
type YahooFetcher struct {
	baseURL string
	api     *apiClient
}

// NewYahooFetcher initializes an equity market data client.
func NewYahooFetcher() *YahooFetcher {
	return &YahooFetcher{baseURL: defaultYahooBaseURL, api: newAPIClient("Yahoo Finance")}
}

// SetBaseURL overrides the API URL (for tests and staging).
func (f *YahooFetcher) SetBaseURL(url string) { f.baseURL = url }

// FetchPair retrieves a quote and the intraday series for an equity ticker.
func (f *YahooFetcher) FetchPair(ctx context.Context, rawInput string) (model.Asset, error) {
	base := extractBaseTicker(rawInput)
	display := base
	if base == "SPY" || base == "S&P" {
		display = "S&P 500"
	}
	fail := func(err error) (model.Asset, error) {
		return model.Asset{Symbol: base, Display: display, Type: model.AssetStock, Err: err}, err
	}

	query := base
	if query == "S&P" {
		query = "SPY"
	}

	var resp YahooChartResponse
	endpoint := fmt.Sprintf("%s/v8/finance/chart/%s?interval=15m&range=1d", f.baseURL, url.PathEscape(query))
	if err := f.api.getJSON(ctx, endpoint, &resp); err != nil {
		return fail(err)
	}
	if resp.Chart.Error != nil {
		return fail(fmt.Errorf("Yahoo Finance: %s", resp.Chart.Error.Description))
	}
	if len(resp.Chart.Result) == 0 {
		return fail(fmt.Errorf("Yahoo Finance: no data for %s", base))
	}

	res := resp.Chart.Result[0]
	meta := res.Meta
	switch strings.ToUpper(meta.InstrumentType) {
	case "CRYPTOCURRENCY", "CURRENCY":
		return fail(fmt.Errorf("Yahoo Finance: %s is not an equity", base))
	}
	if meta.RegularMarketPrice <= 0 {
		return fail(fmt.Errorf("Yahoo Finance: no price for %s", base))
	}

	change := meta.RegularMarketChangePercent
	if change == 0 && meta.ChartPreviousClose > 0 {
		change = (meta.RegularMarketPrice - meta.ChartPreviousClose) / meta.ChartPreviousClose * 100
	}

	var history []float64
	if len(res.Indicators.Quote) > 0 {
		for _, c := range res.Indicators.Quote[0].Close {
			if !math.IsNaN(c) && c > 0 {
				history = append(history, c)
			}
		}
	}
	if len(history) < 2 {
		history = nil
	}

	name := meta.LongName
	if name == "" {
		name = meta.ShortName
	}
	if name == "" {
		name = LookupAssetName(base)
	}

	return model.Asset{
		Symbol:      base,
		Display:     display,
		Name:        name,
		Type:        model.AssetStock,
		Source:      "Yahoo Finance",
		Price:       meta.RegularMarketPrice,
		Open24h:     meta.ChartPreviousClose,
		High24h:     meta.RegularMarketDayHigh,
		Low24h:      meta.RegularMarketDayLow,
		Volume24h:   meta.RegularMarketVolume,
		Change24h:   change,
		History:     history,
		LastUpdated: time.Now(),
	}, nil
}

// FetchPrices fetches several tickers concurrently.
func (f *YahooFetcher) FetchPrices(ctx context.Context, symbols []string) ([]model.Asset, error) {
	return fetchAll(symbols, 4, func(s string) model.Asset {
		a, err := f.FetchPair(ctx, s)
		if err != nil {
			a.Err = err
		}
		return a
	}), nil
}
