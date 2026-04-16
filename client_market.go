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

// GetKlines retrieves kline/candlestick data for a symbol and interval.
func (c *Client) GetKlines(ctx context.Context, symbol string, interval Interval) (*KlinesResponse, error) {
	path := fmt.Sprintf("/market/data/klines?symbol=%s&interval=%s", symbol, string(interval))
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

// GetOrderbook retrieves the order book for a symbol.
func (c *Client) GetOrderbook(ctx context.Context, symbol string, depth int, granularity int) (*OrderBook, error) {
	path := fmt.Sprintf("/market/data/orderbook?symbol=%s&depth=%d&granularity=%d", symbol, depth, granularity)
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
