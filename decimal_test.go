package hibachi

import (
	"strings"
	"testing"

	"github.com/shopspring/decimal"
)

func TestDecimalFromString_Valid(t *testing.T) {
	d, err := DecimalFromString("1.23")
	if err != nil {
		t.Fatalf("DecimalFromString: %v", err)
	}
	if d.String() != "1.23" {
		t.Fatalf("got %q, want 1.23", d.String())
	}
}

func TestDecimalFromString_Invalid(t *testing.T) {
	_, err := DecimalFromString("not-a-number")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFullPrecisionString_NoScientificNotation(t *testing.T) {
	// shopspring/decimal may emit scientific notation for very small numbers.
	// FullPrecisionString must always produce a plain decimal string that the
	// exchange can parse.
	d := decimal.New(1, -15) // 0.000000000000001
	s := FullPrecisionString(d)
	if strings.ContainsAny(s, "eE") {
		t.Fatalf("unexpected scientific notation: %q", s)
	}
	// Should round-trip through decimal.NewFromString.
	parsed, err := decimal.NewFromString(s)
	if err != nil {
		t.Fatalf("parse round-trip: %v", err)
	}
	if !parsed.Equal(d) {
		t.Fatalf("round-trip: got %s, want %s", parsed, d)
	}
}

func TestFullPrecisionString_LargeInteger(t *testing.T) {
	d := decimal.New(1, 20) // 1e20
	s := FullPrecisionString(d)
	if strings.ContainsAny(s, "eE") {
		t.Fatalf("unexpected scientific notation: %q", s)
	}
}

func TestNumericToDecimal_StringInput(t *testing.T) {
	d, err := NumericToDecimal("2.5")
	if err != nil {
		t.Fatalf("NumericToDecimal: %v", err)
	}
	if !d.Equal(decimal.NewFromFloat(2.5)) {
		t.Fatalf("got %s, want 2.5", d)
	}
}

func TestNumericToDecimal_IntTypes(t *testing.T) {
	cases := []interface{}{int(3), int32(3), int64(3)}
	for _, c := range cases {
		d, err := NumericToDecimal(c)
		if err != nil {
			t.Fatalf("NumericToDecimal(%T): %v", c, err)
		}
		if !d.Equal(decimal.NewFromInt(3)) {
			t.Fatalf("NumericToDecimal(%T): got %s, want 3", c, d)
		}
	}
}

func TestNumericToDecimal_Unsupported(t *testing.T) {
	_, err := NumericToDecimal([]int{1, 2, 3})
	if err == nil {
		t.Fatal("expected error for unsupported type")
	}
}
