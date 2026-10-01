package fetcher

import (
	"testing"

	"cryptowatcher/internal/model"
)

func TestNormalizeSymbol(t *testing.T) {
	tests := []struct{ in, wantSymbol, wantDisplay string }{
		{"btc", "BTC-USD", "BTC/USD"},
		{"BTC/USD", "BTC-USD", "BTC/USD"},
		{"eth_usd", "ETH-USD", "ETH/USD"},
		{"  ada/eur  ", "ADA-EUR", "ADA/EUR"},
		{"aapl", "AAPL", "AAPL"},
		{"spy", "SPY", "S&P 500"},
		{"hmm", "HMM-USD", "HMM/USD"},
		{"", "", ""},
	}
	for _, tt := range tests {
		s, d := NormalizeSymbol(tt.in)
		if s != tt.wantSymbol || d != tt.wantDisplay {
			t.Errorf("NormalizeSymbol(%q) = (%q, %q); want (%q, %q)", tt.in, s, d, tt.wantSymbol, tt.wantDisplay)
		}
	}
}

func TestDetectAssetType(t *testing.T) {
	tests := []struct {
		in   string
		want model.AssetType
	}{
		{"AAPL", model.AssetStock},
		{"spy", model.AssetStock},
		{"BTC", model.AssetCrypto},
		{"hmm", model.AssetCrypto},
		{"AMD-USD", model.AssetCrypto}, // explicit quote currency means crypto
		{"AMD", ""},                    // unknown: must be probed
		{"NFLX", ""},
	}
	for _, tt := range tests {
		if got := DetectAssetType(tt.in); got != tt.want {
			t.Errorf("DetectAssetType(%q) = %q; want %q", tt.in, got, tt.want)
		}
	}
}

func TestValidateInput(t *testing.T) {
	for _, ok := range []string{"BTC", "btc/usd", "S&P", "BRK.B", "eth_usd"} {
		if err := ValidateInput(ok); err != nil {
			t.Errorf("ValidateInput(%q) unexpected error: %v", ok, err)
		}
	}
	for _, bad := range []string{"", "   ", "BT C", "a?b=c", "../etc", "BTC;DROP", "ÄÖÜ", "WAYTOOLONGTICKERSYMBOLNAME1"} {
		if err := ValidateInput(bad); err == nil {
			t.Errorf("ValidateInput(%q) expected error", bad)
		}
	}
}

func TestLookupAssetName(t *testing.T) {
	if got := LookupAssetName("BTC-USD"); got != "Bitcoin" {
		t.Errorf("got %q", got)
	}
	if got := LookupAssetName("ZZZ-USD"); got != "ZZZ" {
		t.Errorf("unknown symbols should fall back to the ticker, got %q", got)
	}
}
