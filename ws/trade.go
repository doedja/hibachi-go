package ws

import (
	"context"
	"encoding/binary"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	hibachi "github.com/doedja/hibachi-go"
)

// TradeClient sends order operations over WebSocket using a synchronous
// request-response pattern.
type TradeClient struct {
	opts      TradeClientOptions
	conn      *Conn
	requestID int64
	mu        sync.Mutex

	// Auto-reconnect state
	reconnectMu        sync.Mutex
	reconnectHandlers  []func()
	disconnectHandlers []func(error)
}

// NewTradeClient creates a new trade client.
func NewTradeClient(opts TradeClientOptions) *TradeClient {
	if opts.URL == "" {
		opts.URL = defaultTradeURL
	}
	return &TradeClient{opts: opts}
}

// Connect establishes the authenticated WebSocket connection.
func (c *TradeClient) Connect(ctx context.Context) error {
	url := c.opts.URL + "?accountId=" + strconv.Itoa(c.opts.AccountID) + "&hibachiClient=HibachiGoSDK/" + hibachi.Version
	headers := http.Header{}
	headers.Set("Authorization", c.opts.APIKey)
	conn, err := ConnectWithRetry(ctx, url, headers, c.opts.RetryOpts)
	if err != nil {
		return err
	}
	c.conn = conn
	return nil
}

// OnReconnect registers a callback that fires after a successful lazy reconnect
// and re-enabling cancel-on-disconnect.
func (c *TradeClient) OnReconnect(handler func()) {
	c.reconnectHandlers = append(c.reconnectHandlers, handler)
}

// OnDisconnect registers a callback that fires when a connection error is detected.
func (c *TradeClient) OnDisconnect(handler func(error)) {
	c.disconnectHandlers = append(c.disconnectHandlers, handler)
}

// PlaceOrder places a new order. The signature must already be set in params.
func (c *TradeClient) PlaceOrder(ctx context.Context, params hibachi.OrderPlaceParams) (*hibachi.WSResponse, error) {
	sig := params.Signature
	params.Signature = ""
	return c.sendSignedRequest(ctx, "order.place", params, sig)
}

// ModifyOrder modifies an existing order. The signature must already be set in params.
func (c *TradeClient) ModifyOrder(ctx context.Context, params hibachi.OrderModifyParams) (*hibachi.WSResponse, error) {
	sig := params.Signature
	params.Signature = ""
	return c.sendSignedRequest(ctx, "order.modify", params, sig)
}

// CancelOrder cancels an order by order ID or order nonce.
func (c *TradeClient) CancelOrder(ctx context.Context, orderID *int64, orderNonce *int64) (*hibachi.WSResponse, error) {
	nonce := time.Now().UnixNano() / 1000

	nonceBuf := make([]byte, 8)
	binary.BigEndian.PutUint64(nonceBuf, uint64(nonce))

	var signature string
	if c.opts.Signer != nil {
		var err error
		signature, err = c.opts.Signer.Sign(nonceBuf)
		if err != nil {
			return nil, fmt.Errorf("signing order.cancel: %w", err)
		}
	}

	params := map[string]interface{}{
		"accountId": c.opts.AccountID,
		"nonce":     strconv.FormatInt(nonce, 10),
	}
	if orderID != nil {
		params["orderId"] = strconv.FormatInt(*orderID, 10)
	}
	if orderNonce != nil {
		params["orderNonce"] = strconv.FormatInt(*orderNonce, 10)
	}

	id := atomic.AddInt64(&c.requestID, 1)
	msg := map[string]interface{}{
		"id":        id,
		"method":    "order.cancel",
		"params":    params,
		"signature": signature,
	}

	c.mu.Lock()

	if c.conn == nil {
		c.mu.Unlock()
		if err := c.reconnectLazy(ctx); err != nil {
			return nil, newWSConnectionError("order.cancel: reconnect failed: " + err.Error())
		}
		c.mu.Lock()
		if c.conn == nil {
			c.mu.Unlock()
			return nil, newWSConnectionError("order.cancel: connection not available after reconnect")
		}
	}

	if err := c.conn.SendJSON(ctx, msg); err != nil {
		c.conn.Close()
		c.conn = nil
		c.mu.Unlock()
		c.fireDisconnectHandlers(err)
		return nil, newWSConnectionError("sending order.cancel: " + err.Error())
	}

	var resp hibachi.WSResponse
	if err := c.conn.ReadJSON(ctx, &resp); err != nil {
		c.conn.Close()
		c.conn = nil
		c.mu.Unlock()
		c.fireDisconnectHandlers(err)
		return nil, newWSConnectionError("reading order.cancel response: " + err.Error())
	}

	c.mu.Unlock()

	if resp.Status != 200 {
		return &resp, fmt.Errorf("order.cancel failed: status %d, error %s", resp.Status, string(resp.Error))
	}

	return &resp, nil
}

// CancelAllOrders cancels all open orders.
func (c *TradeClient) CancelAllOrders(ctx context.Context) (*hibachi.WSResponse, error) {
	return c.sendSignedNonce(ctx, "orders.cancel")
}

// GetOrderStatus gets the status of a single order by order ID or nonce.
func (c *TradeClient) GetOrderStatus(ctx context.Context, orderID *int64, nonce *int64) (*hibachi.WSResponse, error) {
	params := map[string]interface{}{
		"accountId": c.opts.AccountID,
	}
	if orderID != nil {
		params["orderId"] = *orderID
	}
	if nonce != nil {
		params["nonce"] = *nonce
	}
	return c.sendRequest(ctx, "order.status", params)
}

// GetOrdersStatus gets the status of all orders.
func (c *TradeClient) GetOrdersStatus(ctx context.Context) (*hibachi.WSResponse, error) {
	params := map[string]interface{}{
		"accountId": c.opts.AccountID,
	}
	return c.sendRequest(ctx, "orders.status", params)
}

// BatchOrders sends a batch of order operations.
func (c *TradeClient) BatchOrders(ctx context.Context, orders []interface{}) (*hibachi.WSResponse, error) {
	params := map[string]interface{}{
		"accountId": c.opts.AccountID,
		"orders":    orders,
	}
	return c.sendRequest(ctx, "orders.batch", params)
}

// EnableCancelOnDisconnect enables automatic cancellation of open orders on disconnect.
func (c *TradeClient) EnableCancelOnDisconnect(ctx context.Context) (*hibachi.WSResponse, error) {
	return c.sendSignedNonce(ctx, "orders.enableCancelOnDisconnect")
}

// Reconnect closes the existing connection and re-establishes it.
func (c *TradeClient) Reconnect(ctx context.Context) error {
	c.mu.Lock()
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	c.mu.Unlock()

	if err := c.Connect(ctx); err != nil {
		return err
	}
	if _, err := c.EnableCancelOnDisconnect(ctx); err != nil {
		return err
	}

	for _, h := range c.reconnectHandlers {
		h()
	}

	return nil
}

// Disconnect closes the WebSocket connection.
func (c *TradeClient) Disconnect() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// reconnectLazy reconnects the trade client when conn is nil. Prevents
// concurrent reconnection attempts. Does not retry the failed operation
// (safe for trading — avoids duplicate orders).
func (c *TradeClient) reconnectLazy(ctx context.Context) error {
	c.reconnectMu.Lock()
	defer c.reconnectMu.Unlock()

	// Double-check: another goroutine may have reconnected already
	c.mu.Lock()
	if c.conn != nil {
		c.mu.Unlock()
		return nil
	}
	c.mu.Unlock()

	if err := c.Connect(ctx); err != nil {
		return err
	}
	if _, err := c.EnableCancelOnDisconnect(ctx); err != nil {
		return err
	}

	for _, h := range c.reconnectHandlers {
		h()
	}

	return nil
}

// fireDisconnectHandlers fires all registered disconnect handlers.
func (c *TradeClient) fireDisconnectHandlers(err error) {
	for _, h := range c.disconnectHandlers {
		h(err)
	}
}

// sendSignedNonce sends a request that requires accountId + nonce + signature-over-nonce.
func (c *TradeClient) sendSignedNonce(ctx context.Context, method string) (*hibachi.WSResponse, error) {
	nonce := time.Now().UnixNano() / 1000

	nonceBuf := make([]byte, 8)
	binary.BigEndian.PutUint64(nonceBuf, uint64(nonce))

	var signature string
	if c.opts.Signer != nil {
		var err error
		signature, err = c.opts.Signer.Sign(nonceBuf)
		if err != nil {
			return nil, fmt.Errorf("signing %s: %w", method, err)
		}
	}

	id := atomic.AddInt64(&c.requestID, 1)
	msg := map[string]interface{}{
		"id":     id,
		"method": method,
		"params": map[string]interface{}{
			"accountId": c.opts.AccountID,
			"nonce":     nonce,
		},
		"signature": signature,
	}

	c.mu.Lock()

	if c.conn == nil {
		c.mu.Unlock()
		if err := c.reconnectLazy(ctx); err != nil {
			return nil, newWSConnectionError(method + ": reconnect failed: " + err.Error())
		}
		c.mu.Lock()
		if c.conn == nil {
			c.mu.Unlock()
			return nil, newWSConnectionError(method + ": connection not available after reconnect")
		}
	}

	if err := c.conn.SendJSON(ctx, msg); err != nil {
		c.conn.Close()
		c.conn = nil
		c.mu.Unlock()
		c.fireDisconnectHandlers(err)
		return nil, newWSConnectionError("sending " + method + ": " + err.Error())
	}

	var resp hibachi.WSResponse
	if err := c.conn.ReadJSON(ctx, &resp); err != nil {
		c.conn.Close()
		c.conn = nil
		c.mu.Unlock()
		c.fireDisconnectHandlers(err)
		return nil, newWSConnectionError("reading " + method + " response: " + err.Error())
	}

	c.mu.Unlock()

	if resp.Status != 200 {
		return &resp, fmt.Errorf("%s failed: status %d, error %s", method, resp.Status, string(resp.Error))
	}

	return &resp, nil
}

// sendSignedRequest sends a request with the signature at the top level of
// the WS message (not inside params). Used for order placement and modification.
func (c *TradeClient) sendSignedRequest(ctx context.Context, method string, params interface{}, signature string) (*hibachi.WSResponse, error) {
	id := atomic.AddInt64(&c.requestID, 1)

	msg := map[string]interface{}{
		"id":        id,
		"method":    method,
		"params":    params,
		"signature": signature,
	}

	c.mu.Lock()

	if c.conn == nil {
		c.mu.Unlock()
		if err := c.reconnectLazy(ctx); err != nil {
			return nil, newWSConnectionError(method + ": reconnect failed: " + err.Error())
		}
		c.mu.Lock()
		if c.conn == nil {
			c.mu.Unlock()
			return nil, newWSConnectionError(method + ": connection not available after reconnect")
		}
	}

	if err := c.conn.SendJSON(ctx, msg); err != nil {
		c.conn.Close()
		c.conn = nil
		c.mu.Unlock()
		c.fireDisconnectHandlers(err)
		return nil, newWSConnectionError("sending " + method + ": " + err.Error())
	}

	var resp hibachi.WSResponse
	if err := c.conn.ReadJSON(ctx, &resp); err != nil {
		c.conn.Close()
		c.conn = nil
		c.mu.Unlock()
		c.fireDisconnectHandlers(err)
		return nil, newWSConnectionError("reading " + method + " response: " + err.Error())
	}

	c.mu.Unlock()

	if resp.Status != 200 {
		return &resp, fmt.Errorf("%s failed: status %d, error %s", method, resp.Status, string(resp.Error))
	}

	return &resp, nil
}

// sendRequest sends a request and waits for the response.
func (c *TradeClient) sendRequest(ctx context.Context, method string, params interface{}) (*hibachi.WSResponse, error) {
	id := atomic.AddInt64(&c.requestID, 1)

	msg := map[string]interface{}{
		"id":     id,
		"method": method,
		"params": params,
	}

	c.mu.Lock()

	if c.conn == nil {
		c.mu.Unlock()
		if err := c.reconnectLazy(ctx); err != nil {
			return nil, newWSConnectionError(method + ": reconnect failed: " + err.Error())
		}
		c.mu.Lock()
		if c.conn == nil {
			c.mu.Unlock()
			return nil, newWSConnectionError(method + ": connection not available after reconnect")
		}
	}

	if err := c.conn.SendJSON(ctx, msg); err != nil {
		c.conn.Close()
		c.conn = nil
		c.mu.Unlock()
		c.fireDisconnectHandlers(err)
		return nil, newWSConnectionError("sending " + method + ": " + err.Error())
	}

	var resp hibachi.WSResponse
	if err := c.conn.ReadJSON(ctx, &resp); err != nil {
		c.conn.Close()
		c.conn = nil
		c.mu.Unlock()
		c.fireDisconnectHandlers(err)
		return nil, newWSConnectionError("reading " + method + " response: " + err.Error())
	}

	c.mu.Unlock()

	if resp.Status != 200 {
		return &resp, fmt.Errorf("%s failed: status %d, error %s", method, resp.Status, string(resp.Error))
	}

	return &resp, nil
}

// newWSConnectionError creates a WSConnectionError with the given message.
func newWSConnectionError(msg string) *hibachi.WSConnectionError {
	return &hibachi.WSConnectionError{
		TransportError: hibachi.TransportError{
			HibachiError: hibachi.HibachiError{Message: msg},
		},
	}
}
