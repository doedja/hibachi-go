package hibachi

import (
	"strings"
	"testing"

	"github.com/shopspring/decimal"
)

func dec(t *testing.T, s string) decimal.Decimal {
	t.Helper()
	d, err := decimal.NewFromString(s)
	if err != nil {
		t.Fatalf("parse %q: %v", s, err)
	}
	return d
}

func TestRoundPriceToTick(t *testing.T) {
	cases := []struct {
		name  string
		price string
		tick  string
		want  string
	}{
		{"already aligned", "71286.1", "0.1", "71286.1"},
		{"round down", "71286.12", "0.1", "71286.1"},
		{"round up", "71286.16", "0.1", "71286.2"},
		{"half rounds away", "71286.15", "0.1", "71286.2"},
		{"sub-tick precision", "71286.1234567", "0.1", "71286.1"},
		{"penny tick", "3543.678", "0.01", "3543.68"},
		{"milli tick", "143.4567", "0.001", "143.457"},
		{"non-power-of-ten tick", "100.7", "0.5", "100.5"},
		{"non-power-of-ten round up", "100.8", "0.5", "101"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := RoundPriceToTick(dec(t, tc.price), dec(t, tc.tick))
			if !got.Equal(dec(t, tc.want)) {
				t.Fatalf("got %s, want %s", got, tc.want)
			}
		})
	}
}

func TestRoundPriceToTick_ZeroTickPassthrough(t *testing.T) {
	price := dec(t, "71286.1234567")
	if got := RoundPriceToTick(price, decimal.Zero); !got.Equal(price) {
		t.Fatalf("zero tick should pass through, got %s", got)
	}
	if got := RoundPriceToTick(price, dec(t, "-0.1")); !got.Equal(price) {
		t.Fatalf("negative tick should pass through, got %s", got)
	}
}

func TestRoundQuantityToStep(t *testing.T) {
	cases := []struct {
		name string
		qty  string
		step string
		want string
	}{
		{"already aligned", "0.001", "0.001", "0.001"},
		{"floors down", "0.0019", "0.001", "0.001"},
		{"never rounds up", "0.0029999", "0.001", "0.002"},
		{"whole-number step", "7.9", "1", "7"},
		{"sub-step pass to zero", "0.0009", "0.001", "0"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := RoundQuantityToStep(dec(t, tc.qty), dec(t, tc.step))
			if !got.Equal(dec(t, tc.want)) {
				t.Fatalf("got %s, want %s", got, tc.want)
			}
		})
	}
}

func TestRoundQuantityToStep_ZeroStepPassthrough(t *testing.T) {
	qty := dec(t, "0.001234")
	if got := RoundQuantityToStep(qty, decimal.Zero); !got.Equal(qty) {
		t.Fatalf("zero step should pass through, got %s", got)
	}
}

func TestCheckTickSize(t *testing.T) {
	if err := CheckTickSize(dec(t, "71286.1"), dec(t, "0.1")); err != nil {
		t.Fatalf("aligned price should not error: %v", err)
	}
	err := CheckTickSize(dec(t, "71286.12"), dec(t, "0.1"))
	if err == nil {
		t.Fatal("misaligned price should error")
	}
	if !strings.Contains(err.Error(), "tick size") {
		t.Fatalf("error should mention tick size, got %q", err.Error())
	}
}

func TestCheckTickSize_ZeroTickPassthrough(t *testing.T) {
	if err := CheckTickSize(dec(t, "71286.123"), decimal.Zero); err != nil {
		t.Fatalf("zero tick should be a no-op, got %v", err)
	}
}
