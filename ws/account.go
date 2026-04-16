package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	hibachi "github.com/doedja/hibachi-go"
)

// AllEventHandler receives every dispatched event regardless of topic.
type AllEventHandler func(topic string, data json.RawMessage)

// AccountClient manages an account event stream over WebSocket.
type AccountClient struct {
	opts       AccountClientOptions
	conn       *Conn
	handlers   map[string][]EventHandler
	allHandler AllEventHandler
	mu         sync.RWMutex
	listenKey  string
	requestID  int64

	// Auto-reconnect state
	url                string
	headers            http.Header
	reconnectHandlers  []func(*hibachi.AccountStreamStartResult)
	disconnectHandlers []func(error)
	pingCancel         context.CancelFunc
}

// NewAccountClient creates a new account stream client.
func NewAccountClient(opts AccountClientOptions) *AccountClient {
	if opts.URL == "" {
		opts.URL = defaultAccountURL
	}
	return &AccountClient{
		opts:     opts,
		handlers: make(map[string][]EventHandler),
	}
}

// Connect establishes the authenticated WebSocket connection.
func (c *AccountClient) Connect(ctx context.Context) error {
	c.url = c.opts.URL + "?accountId=" + strconv.Itoa(c.opts.AccountID) + "&hibachiClient=HibachiGoSDK/" + hibachi.Version
	c.headers = http.Header{}
	c.headers.Set("Authorization", c.opts.APIKey)
	conn, err := ConnectWithRetry(ctx, c.url, c.headers, c.opts.RetryOpts)
	if err != nil {
		return err
	}
	c.conn = conn
	return nil
}

// StreamStart sends a stream.start request and returns the account snapshot and listen key.
func (c *AccountClient) StreamStart(ctx context.Context) (*hibachi.AccountStreamStartResult, error) {
	id := atomic.AddInt64(&c.requestID, 1)
	msg := map[string]interface{}{
		"id":     id,
		"method": "stream.start",
		"params": map[string]interface{}{
			"accountId": c.opts.AccountID,
		},
		"timestamp": time.Now().Unix(),
	}
	if err := c.conn.SendJSON(ctx, msg); err != nil {
		return nil, fmt.Errorf("sending stream.start: %w", err)
	}

	var resp hibachi.WSResponse
	if err := c.conn.ReadJSON(ctx, &resp); err != nil {
		return nil, fmt.Errorf("reading stream.start response: %w", err)
	}
	if resp.Status != 200 {
		return nil, fmt.Errorf("stream.start failed: status %d, error %s", resp.Status, string(resp.Error))
	}

	var result hibachi.AccountStreamStartResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("parsing stream.start result: %w", err)
	}

	c.listenKey = result.ListenKey
	return &result, nil
}

// ListenLoop reads messages and dispatches to handlers until ctx is cancelled.
// On connection errors, it automatically reconnects, re-starts the stream,
// and fires OnReconnect handlers with the new snapshot.
func (c *AccountClient) ListenLoop(ctx context.Context) error {
	c.startPingLoop(ctx)

	for {
		if ctx.Err() != nil {
			c.stopPingLoop()
			return ctx.Err()
		}

		var raw json.RawMessage
		if err := c.conn.ReadJSON(ctx, &raw); err != nil {
			if ctx.Err() != nil {
				c.stopPingLoop()
				return ctx.Err()
			}

			c.stopPingLoop()

			for _, h := range c.disconnectHandlers {
				h(err)
			}

			if reconnErr := c.reconnect(ctx); reconnErr != nil {
				return fmt.Errorf("account ws reconnect failed: %w", reconnErr)
			}

			c.startPingLoop(ctx)
			continue
		}

		var envelope struct {
			Event  string          `json:"event"`
			Data   json.RawMessage `json:"data"`
			ID     int             `json:"id"`
			Status int             `json:"status"`
		}
		_ = json.Unmarshal(raw, &envelope)

		c.mu.RLock()
		handlers := c.handlers[envelope.Event]
		allH := c.allHandler
		c.mu.RUnlock()

		if allH != nil {
			allH(envelope.Event, raw)
		}

		for _, h := range handlers {
			h(envelope.Data)
		}
	}
}

// OnReconnect registers a callback that fires after a successful reconnect
// and StreamStart. The callback receives the new account snapshot.
func (c *AccountClient) OnReconnect(handler func(*hibachi.AccountStreamStartResult)) {
	c.reconnectHandlers = append(c.reconnectHandlers, handler)
}

// OnDisconnect registers a callback that fires when the connection drops,
// before a reconnect attempt.
func (c *AccountClient) OnDisconnect(handler func(error)) {
	c.disconnectHandlers = append(c.disconnectHandlers, handler)
}

// On registers a handler for a given topic (e.g. "balance", "position", "order").
func (c *AccountClient) On(topic string, handler EventHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handlers[topic] = append(c.handlers[topic], handler)
}

// OnAll registers a handler that receives every event (for debugging).
func (c *AccountClient) OnAll(handler AllEventHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.allHandler = handler
}

// Disconnect closes the WebSocket connection.
func (c *AccountClient) Disconnect() error {
	c.stopPingLoop()
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// reconnect closes the old connection, re-establishes it, re-starts the
// stream, and fires reconnect handlers with the new snapshot.
func (c *AccountClient) reconnect(ctx context.Context) error {
	if c.conn != nil {
		c.conn.Close()
	}

	conn, err := ConnectWithRetry(ctx, c.url, c.headers, c.opts.RetryOpts)
	if err != nil {
		return err
	}
	c.conn = conn

	result, err := c.StreamStart(ctx)
	if err != nil {
		return fmt.Errorf("re-start stream: %w", err)
	}

	for _, h := range c.reconnectHandlers {
		h(result)
	}

	return nil
}

// startPingLoop starts a background goroutine that sends pings every 10 seconds.
func (c *AccountClient) startPingLoop(ctx context.Context) {
	pingCtx, cancel := context.WithCancel(ctx)
	c.pingCancel = cancel
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-pingCtx.Done():
				return
			case <-ticker.C:
				if err := c.sendPing(pingCtx); err != nil {
					return
				}
			}
		}
	}()
}

// stopPingLoop stops the current ping goroutine.
func (c *AccountClient) stopPingLoop() {
	if c.pingCancel != nil {
		c.pingCancel()
		c.pingCancel = nil
	}
}

func (c *AccountClient) sendPing(ctx context.Context) error {
	id := atomic.AddInt64(&c.requestID, 1)
	msg := map[string]interface{}{
		"id":     id,
		"method": "stream.ping",
		"params": map[string]interface{}{
			"accountId": c.opts.AccountID,
			"listenKey": c.listenKey,
		},
		"timestamp": time.Now().Unix(),
	}
	return c.conn.SendJSON(ctx, msg)
}
