package hibachi

import (
	"encoding/json"
	"testing"
	"time"
)

// liveFXContract mirrors the exchange-info shape for AUD/USDT-P observed live:
// FX category, OPEN, opened in the past, next close in the future.
const liveFXContract = `{
	"category": "FX",
	"displayName": "AUD/USDT Perps",
	"id": 60,
	"status": "LIVE",
	"symbolStatus": "OPEN",
	"symbol": "AUD/USDT-P",
	"underlyingSymbol": "AUD",
	"tickSize": "0.00001",
	"stepSize": "0.00001",
	"marketOpenTimestamp": "1779825600",
	"marketCloseTimestamp": null,
	"nextOpenTimestamp": null,
	"nextCloseTimestamp": "1780088400"
}`

func TestFutureContractParsesFXFields(t *testing.T) {
	var c FutureContract
	if err := json.Unmarshal([]byte(liveFXContract), &c); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !c.IsFX() {
		t.Errorf("IsFX() = false, want true for category=%q", c.Category)
	}
	if c.SymbolStatus != "OPEN" {
		t.Errorf("SymbolStatus = %q, want OPEN", c.SymbolStatus)
	}
	if c.NextCloseTimestamp == nil || *c.NextCloseTimestamp != "1780088400" {
		t.Errorf("NextCloseTimestamp = %v, want 1780088400", c.NextCloseTimestamp)
	}
}

func TestMarketOpen(t *testing.T) {
	var c FutureContract
	if err := json.Unmarshal([]byte(liveFXContract), &c); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	open := time.Unix(1779825600, 0).Add(time.Hour)  // just after open
	closed := time.Unix(1780088400, 0).Add(time.Hour) // after next close
	if !c.MarketOpen(open) {
		t.Errorf("MarketOpen during session = false, want true")
	}
	if c.MarketOpen(closed) {
		t.Errorf("MarketOpen after close = true, want false")
	}
	if d, ok := c.TimeToClose(open); !ok || d <= 0 {
		t.Errorf("TimeToClose during session = (%v,%v), want positive", d, ok)
	}
}

func TestCryptoAlwaysOpen(t *testing.T) {
	// Crypto contract: no schedule timestamps, no symbolStatus restriction.
	c := FutureContract{Category: "CRYPTO", Symbol: "BTC/USDT-P"}
	if c.IsFX() {
		t.Errorf("IsFX() = true for crypto")
	}
	if !c.MarketOpen(time.Now()) {
		t.Errorf("MarketOpen for crypto = false, want true (24/7)")
	}
	if _, ok := c.TimeToClose(time.Now()); ok {
		t.Errorf("TimeToClose for crypto returned ok=true, want false")
	}
}

func TestFXWaitingToReopen(t *testing.T) {
	// Closed market with a future reopen and no next close.
	next := "1780000000"
	c := FutureContract{Category: "FX", SymbolStatus: "OPEN", NextOpenTimestamp: &next}
	before := time.Unix(1779000000, 0)
	if c.MarketOpen(before) {
		t.Errorf("MarketOpen before scheduled reopen = true, want false")
	}
}
