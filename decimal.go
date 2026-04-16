package hibachi

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

// DecimalFromString parses a string into a decimal.Decimal.
func DecimalFromString(s string) (decimal.Decimal, error) {
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Decimal{}, fmt.Errorf("invalid decimal string %q: %w", s, err)
	}
	return d, nil
}

// FullPrecisionString formats a decimal without scientific notation.
func FullPrecisionString(d decimal.Decimal) string {
	s := d.String()
	// shopspring/decimal may use scientific notation for very large/small numbers
	if strings.ContainsAny(s, "eE") {
		exp := -d.Exponent()
		if exp < 0 {
			exp = 0
		}
		return d.StringFixed(exp)
	}
	return s
}

// NumericToDecimal converts various numeric types to a decimal.Decimal.
func NumericToDecimal(v interface{}) (decimal.Decimal, error) {
	switch val := v.(type) {
	case decimal.Decimal:
		return val, nil
	case string:
		return DecimalFromString(val)
	case int:
		return decimal.NewFromInt(int64(val)), nil
	case int32:
		return decimal.NewFromInt(int64(val)), nil
	case int64:
		return decimal.NewFromInt(val), nil
	case float64:
		return decimal.NewFromFloat(val), nil
	case float32:
		return decimal.NewFromFloat32(val), nil
	default:
		return decimal.Decimal{}, fmt.Errorf("unsupported numeric type: %T", v)
	}
}
