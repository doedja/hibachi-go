package hibachi

import (
	"context"
	"encoding/json"
	"fmt"
)

// GetExchangeInfo retrieves exchange information. The result is cached on
// the client after the first successful call; subsequent calls return the
// cached value. Failed calls are not cached.
func (c *Client) GetExchangeInfo(ctx context.Context) (*ExchangeInfo, error) {
	c.exchangeInfoMu.Lock()
	defer c.exchangeInfoMu.Unlock()

	if c.exchangeInfo != nil {
		return c.exchangeInfo, nil
	}

	data, err := c.transport.SendSimpleRequest(ctx, c.dataAPIURL, "/market/exchange-info")
	if err != nil {
		return nil, err
	}

	var info ExchangeInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("unmarshaling exchange info: %w", err)
	}
	c.exchangeInfo = &info
	return c.exchangeInfo, nil
}

// GetInventory retrieves the market inventory.
func (c *Client) GetInventory(ctx context.Context) (json.RawMessage, error) {
	data, err := c.transport.SendSimpleRequest(ctx, c.dataAPIURL, "/market/inventory")
	if err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}

// GetPrices retrieves price data for a symbol.
func (c *Client) GetPrices(ctx context.Context, symbol string) (*PriceResponse, error) {
	path := fmt.Sprintf("/market/data/prices?symbol=%s", symbol)
	data, err := c.transport.SendSimpleRequest(ctx, c.dataAPIURL, path)
	if err != nil {
		return nil, err
	}

	var resp PriceResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("unmarshaling prices: %w", err)
	}
	return &resp, nil
}

// GetStats retrieves market statistics for a symbol.
func (c *Client) GetStats(ctx context.Context, symbol string) (*StatsResponse, error) {
	path := fmt.Sprintf("/market/data/stats?symbol=%s", symbol)
	data, err := c.transport.SendSimpleRequest(ctx, c.dataAPIURL, path)
	if err != nil {
		return nil, err
	}

	var resp StatsResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("unmarshaling stats: %w", err)
	}
	return &resp, nil
}

// GetTrades retrieves recent trades for a symbol.
func (c *Client) GetTrades(ctx context.Context, symbol string) (*TradesResponse, error) {
	path := fmt.Sprintf("/market/data/trades?symbol=%s", symbol)
	data, err := c.transport.SendSimpleRequest(ctx, c.dataAPIURL, path)
	if err != nil {
		return nil, err
	}

	var resp TradesResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("unmarshaling trades: %w", err)
	}
	return &resp, nil
}

// klineParams holds optional kline query parameters.
type klineParams struct {
	fromMs int64
	toMs   int64
}

// KlineOption configures an optional kline query parameter.
type KlineOption func(*klineParams)

// WithKlineRange limits klines to the [fromMs, toMs] window (Unix
// milliseconds). Either bound may be 0 to leave it open. Useful for pulling a
// historical OHLC range for charting rather than just the latest candles.
func WithKlineRange(fromMs, toMs int64) KlineOption {
	return func(p *klineParams) {
		p.fromMs = fromMs
		p.toMs = toMs
	}
}

// GetKlines retrieves kline/candlestick (OHLC) data for a symbol and interval.
// By default the exchange returns its most recent candles; pass WithKlineRange
// to fetch a specific historical window.
func (c *Client) GetKlines(ctx context.Context, symbol string, interval Interval, opts ...KlineOption) (*KlinesResponse, error) {
	var p klineParams
	for _, o := range opts {
		o(&p)
	}
	path := fmt.Sprintf("/market/data/klines?symbol=%s&interval=%s", symbol, string(interval))
	if p.fromMs > 0 {
		path += fmt.Sprintf("&fromMs=%d", p.fromMs)
	}
	if p.toMs > 0 {
		path += fmt.Sprintf("&toMs=%d", p.toMs)
	}
	data, err := c.transport.SendSimpleRequest(ctx, c.dataAPIURL, path)
	if err != nil {
		return nil, err
	}

	var resp KlinesResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("unmarshaling klines: %w", err)
	}
	return &resp, nil
}

// fundingRateParams holds optional funding-rate-history query parameters.
type fundingRateParams struct {
	startTime int64
	endTime   int64
	limit     int
}

// FundingRateOption configures an optional funding-rate-history query parameter.
type FundingRateOption func(*fundingRateParams)

// WithFundingRange limits funding-rate history to [startTime, endTime] (Unix
// seconds). Either bound may be 0 to leave it open.
func WithFundingRange(startTime, endTime int64) FundingRateOption {
	return func(p *fundingRateParams) {
		p.startTime = startTime
		p.endTime = endTime
	}
}

// WithFundingLimit caps the number of funding-rate records returned.
func WithFundingLimit(limit int) FundingRateOption {
	return func(p *fundingRateParams) { p.limit = limit }
}

// GetFundingRates retrieves historical realized funding rates for a symbol.
// Each record carries the funding timestamp, the realized rate, and the index
// price at that time. FX and crypto perps both report funding history.
func (c *Client) GetFundingRates(ctx context.Context, symbol string, opts ...FundingRateOption) ([]FundingRate, error) {
	var p fundingRateParams
	for _, o := range opts {
		o(&p)
	}
	path := fmt.Sprintf("/market/data/funding-rates?symbol=%s", symbol)
	if p.startTime > 0 {
		path += fmt.Sprintf("&startTime=%d", p.startTime)
	}
	if p.endTime > 0 {
		path += fmt.Sprintf("&endTime=%d", p.endTime)
	}
	if p.limit > 0 {
		path += fmt.Sprintf("&limit=%d", p.limit)
	}
	// The transport unwraps the {"data": [...]} envelope, so the body here is
	// the bare array of funding-rate records (pagination is dropped upstream).
	data, err := c.transport.SendSimpleRequest(ctx, c.dataAPIURL, path)
	if err != nil {
		return nil, err
	}

	var rates []FundingRate
	if err := json.Unmarshal(data, &rates); err != nil {
		return nil, fmt.Errorf("unmarshaling funding rates: %w", err)
	}
	return rates, nil
}

// GetOpenInterest retrieves open interest for a symbol.
func (c *Client) GetOpenInterest(ctx context.Context, symbol string) (*OpenInterest, error) {
	path := fmt.Sprintf("/market/data/open-interest?symbol=%s", symbol)
	data, err := c.transport.SendSimpleRequest(ctx, c.dataAPIURL, path)
	if err != nil {
		return nil, err
	}

	var resp OpenInterest
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("unmarshaling open interest: %w", err)
	}
	return &resp, nil
}

// GetOrderbook retrieves the order book for a symbol. Granularity is the
// price-bucket size as a string, matching the values in the contract's
// OrderbookGranularities (e.g. "1", "0.01", or "0.00001" for FX). The exchange
// rejects a granularity that is not in that list, so an integer form like "1"
// is invalid for FX markets whose buckets are all sub-1.
func (c *Client) GetOrderbook(ctx context.Context, symbol string, depth int, granularity string) (*OrderBook, error) {
	if granularity == "" {
		// Auto-pick a valid granularity: the contract's coarsest bucket. Falls
		// back to "1" if the contract or its granularity list is unavailable.
		granularity = "1"
		if contract, err := c.getContract(ctx, symbol); err == nil && len(contract.OrderbookGranularities) > 0 {
			granularity = contract.OrderbookGranularities[len(contract.OrderbookGranularities)-1]
		}
	}
	path := fmt.Sprintf("/market/data/orderbook?symbol=%s&depth=%d&granularity=%s", symbol, depth, granularity)
	data, err := c.transport.SendSimpleRequest(ctx, c.dataAPIURL, path)
	if err != nil {
		return nil, err
	}

	var resp OrderBook
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("unmarshaling orderbook: %w", err)
	}
	return &resp, nil
}
