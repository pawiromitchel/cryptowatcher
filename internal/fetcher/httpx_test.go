package fetcher

import (
	"context"
	"errors"
	"testing"
	"time"

	"cryptowatcher/internal/fakeapi"
)

func TestAPIClient_429SetsCooldownFromRetryAfter(t *testing.T) {
	api := fakeapi.New(t)
	api.RateLimit(fakeapi.Yahoo, 30)
	c := newAPIClient("test")
	now := time.Now()
	c.now = func() time.Time { return now }

	var out map[string]any
	if err := c.getJSON(context.Background(), api.YahooURL+"/x", &out); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("want ErrRateLimited, got %v", err)
	}
	if err := c.getJSON(context.Background(), api.YahooURL+"/x", &out); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("want cooldown error, got %v", err)
	}
	if api.Calls(fakeapi.Yahoo) != 1 {
		t.Errorf("cooldown must suppress requests, got %d", api.Calls(fakeapi.Yahoo))
	}

	now = now.Add(31 * time.Second)
	api.Heal(fakeapi.Yahoo)
	if err := c.getJSON(context.Background(), api.YahooURL+"/v8/finance/chart/NONE", &out); err == nil || errors.Is(err, ErrRateLimited) {
		t.Fatalf("after cooldown the request should go out again (and 404), got %v", err)
	}
}

func TestTTLCache(t *testing.T) {
	c := newTTLCache[int](10 * time.Millisecond)
	c.set("a", 1)
	if v, ok := c.get("a"); !ok || v != 1 {
		t.Fatal("expected hit")
	}
	time.Sleep(20 * time.Millisecond)
	if _, ok := c.get("a"); ok {
		t.Fatal("expected expiry")
	}
}
