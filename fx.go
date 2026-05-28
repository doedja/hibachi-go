package hibachi

import (
	"strconv"
	"strings"
	"time"
)

// Contract categories as reported by the exchange in exchange-info.
const (
	CategoryCrypto = "CRYPTO"
	CategoryFX     = "FX"
)

// IsFX reports whether the contract is a foreign-exchange market. FX markets
// trade on a schedule (they close on weekends) rather than 24/7 like crypto.
func (c FutureContract) IsFX() bool {
	return strings.EqualFold(c.Category, CategoryFX)
}

// parseExchangeTime parses an exchange timestamp string into a time.Time. The
// exchange sends Unix seconds, sometimes with a fractional part
// (e.g. "1778625450.039"). Returns the zero time and false on a nil or
// unparseable value.
func parseExchangeTime(s *string) (time.Time, bool) {
	if s == nil || *s == "" {
		return time.Time{}, false
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(*s), 64)
	if err != nil {
		return time.Time{}, false
	}
	sec := int64(f)
	nsec := int64((f - float64(sec)) * 1e9)
	return time.Unix(sec, nsec).UTC(), true
}

// NextClose returns the contract's next scheduled close time, if any. Crypto
// markets have no close and return ok == false.
func (c FutureContract) NextClose() (time.Time, bool) {
	return parseExchangeTime(c.NextCloseTimestamp)
}

// NextOpen returns the contract's next scheduled open time, if any. A non-nil
// next-open generally means the market is currently closed and will reopen at
// that time.
func (c FutureContract) NextOpen() (time.Time, bool) {
	return parseExchangeTime(c.NextOpenTimestamp)
}

// MarketOpen reports whether the contract is tradeable at the given time.
// Crypto markets are always open. FX markets are open between their scheduled
// open and the next scheduled close. The check is conservative: when the
// exchange explicitly reports a non-open symbol status, it is treated as closed
// regardless of timestamps.
func (c FutureContract) MarketOpen(now time.Time) bool {
	if c.SymbolStatus != "" && !strings.EqualFold(c.SymbolStatus, "OPEN") {
		return false
	}
	// A scheduled future open means we are currently before the next session.
	if t, ok := c.NextOpen(); ok && now.Before(t) {
		return false
	}
	// Past the next scheduled close means the session has ended.
	if t, ok := c.NextClose(); ok && !now.Before(t) {
		return false
	}
	return true
}

// TimeToClose returns how long until the contract's next scheduled close. The
// bool is false when there is no scheduled close (e.g. crypto) or it is in the
// past.
func (c FutureContract) TimeToClose(now time.Time) (time.Duration, bool) {
	t, ok := c.NextClose()
	if !ok {
		return 0, false
	}
	d := t.Sub(now)
	if d <= 0 {
		return 0, false
	}
	return d, true
}
