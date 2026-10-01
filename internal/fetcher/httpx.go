package fetcher

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const (
	defaultTimeout   = 5 * time.Second
	defaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) CryptoWatcher/1.0"
	defaultCooldown  = 60 * time.Second
	maxCooldown      = 10 * time.Minute
)

// ErrRateLimited is returned while a provider is cooling down after an HTTP 429.
var ErrRateLimited = errors.New("rate limited")

// StatusError is a non-200 HTTP response from a provider.
type StatusError struct {
	Provider string
	Code     int
}

func (e *StatusError) Error() string { return fmt.Sprintf("%s: HTTP %d", e.Provider, e.Code) }

// apiClient wraps http.Client with JSON decoding and per-provider rate-limit backoff.
type apiClient struct {
	name string
	http *http.Client

	mu       sync.Mutex
	cooldown time.Time
	now      func() time.Time
}

func newAPIClient(name string) *apiClient {
	return &apiClient{name: name, http: &http.Client{Timeout: defaultTimeout}, now: time.Now}
}

// getJSON GETs url and decodes the JSON body into out. After a 429 the client
// short-circuits further calls until the provider's Retry-After (or a default) elapses.
func (c *apiClient) getJSON(ctx context.Context, url string, out any) error {
	c.mu.Lock()
	until := c.cooldown
	c.mu.Unlock()
	if wait := until.Sub(c.now()); wait > 0 {
		return fmt.Errorf("%s: %w (retry in %s)", c.name, ErrRateLimited, wait.Round(time.Second))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", defaultUserAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("%s: %w", c.name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		d := defaultCooldown
		if secs, convErr := strconv.Atoi(resp.Header.Get("Retry-After")); convErr == nil && secs > 0 {
			d = time.Duration(secs) * time.Second
		}
		if d > maxCooldown {
			d = maxCooldown
		}
		c.mu.Lock()
		c.cooldown = c.now().Add(d)
		c.mu.Unlock()
		return fmt.Errorf("%s: %w", c.name, ErrRateLimited)
	}
	if resp.StatusCode != http.StatusOK {
		return &StatusError{Provider: c.name, Code: resp.StatusCode}
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("%s: decoding response: %w", c.name, err)
	}
	return nil
}

// ttlCache is a small concurrency-safe map with per-entry expiry.
type ttlCache[V any] struct {
	mu  sync.Mutex
	ttl time.Duration
	m   map[string]ttlEntry[V]
}

type ttlEntry[V any] struct {
	v  V
	at time.Time
}

func newTTLCache[V any](ttl time.Duration) *ttlCache[V] {
	return &ttlCache[V]{ttl: ttl, m: map[string]ttlEntry[V]{}}
}

func (c *ttlCache[V]) get(key string) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.m[key]
	if !ok || time.Since(e.at) > c.ttl {
		var zero V
		return zero, false
	}
	return e.v, true
}

func (c *ttlCache[V]) set(key string, v V) {
	c.mu.Lock()
	c.m[key] = ttlEntry[V]{v: v, at: time.Now()}
	c.mu.Unlock()
}

// fetchAll runs fetch for every symbol with bounded concurrency, preserving order.
func fetchAll[T any](symbols []string, limit int, fetch func(string) T) []T {
	results := make([]T, len(symbols))
	sem := make(chan struct{}, limit)
	var wg sync.WaitGroup
	for i, s := range symbols {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, s string) {
			defer wg.Done()
			defer func() { <-sem }()
			results[i] = fetch(s)
		}(i, s)
	}
	wg.Wait()
	return results
}
