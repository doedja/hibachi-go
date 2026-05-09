package hibachi

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"
)

// RoundPriceToTick returns price rounded to the nearest multiple of tick.
// Uses round-half-away-from-zero. Returns price unchanged if tick is zero or
// negative. Tick is the minimum price increment (e.g. 0.1 for BTC/USDT-P).
func RoundPriceToTick(price, tick decimal.Decimal) decimal.Decimal {
	if !tick.IsPositive() {
		return price
	}
	multiples := price.Div(tick).Round(0)
	return multiples.Mul(tick)
}

// RoundQuantityToStep returns quantity rounded down to the nearest multiple of
// step. Rounding down (rather than to-nearest) avoids exceeding available
// margin or position size. Returns quantity unchanged if step is zero or
// negative.
func RoundQuantityToStep(qty, step decimal.Decimal) decimal.Decimal {
	if !step.IsPositive() {
		return qty
	}
	multiples := qty.Div(step).Floor()
	return multiples.Mul(step)
}

// CheckTickSize returns an error if price is not an exact multiple of tick.
// Pass-through (nil) if tick is zero or negative.
func CheckTickSize(price, tick decimal.Decimal) error {
	if !tick.IsPositive() {
		return nil
	}
	if !price.Mod(tick).IsZero() {
		return fmt.Errorf("price %s is not a multiple of tick size %s", FullPrecisionString(price), FullPrecisionString(tick))
	}
	return nil
}

// GetTickSize returns the tick size for symbol as a decimal. Resolves through
// the cached exchange info.
func (c *Client) GetTickSize(ctx context.Context, symbol string) (decimal.Decimal, error) {
	contract, err := c.getContract(ctx, symbol)
	if err != nil {
		return decimal.Decimal{}, err
	}
	if contract.TickSize == "" {
		return decimal.Zero, nil
	}
	return DecimalFromString(contract.TickSize)
}

// GetStepSize returns the step size for symbol as a decimal. Resolves through
// the cached exchange info.
func (c *Client) GetStepSize(ctx context.Context, symbol string) (decimal.Decimal, error) {
	contract, err := c.getContract(ctx, symbol)
	if err != nil {
		return decimal.Decimal{}, err
	}
	if contract.StepSize == "" {
		return decimal.Zero, nil
	}
	return DecimalFromString(contract.StepSize)
}

// contractTick parses a contract's TickSize. Zero-decimal pass-through if the
// field is empty or unparseable so callers fall back to no-op rounding rather
// than blocking placement on a missing exchange-info field.
func contractTick(contract *FutureContract) decimal.Decimal {
	if contract == nil || contract.TickSize == "" {
		return decimal.Zero
	}
	d, err := decimal.NewFromString(contract.TickSize)
	if err != nil {
		return decimal.Zero
	}
	return d
}

// contractStep parses a contract's StepSize. See contractTick for rationale.
func contractStep(contract *FutureContract) decimal.Decimal {
	if contract == nil || contract.StepSize == "" {
		return decimal.Zero
	}
	d, err := decimal.NewFromString(contract.StepSize)
	if err != nil {
		return decimal.Zero
	}
	return d
}
