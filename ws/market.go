package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	hibachi "github.com/doedja/hibachi-go"
)

// EventHandler is a callback for WebSocket events. The data argument
// is the raw JSON payload for the event.
type EventHandler func(data json.RawMessage)

// MarketClient subscribes to real-time market data over WebSocket.
type MarketClient struct {
	opts     MarketClientOptions
	conn     *Conn
	handlers map[string][]EventHandler
	mu       sync.RWMutex
	cancel   context.CancelFunc
	pumpDone chan error // signals when readPump exits

	// Auto-reconnect state
	url                string
	subscriptions      []hibachi.WSSubscription
	subsMu             sync.Mutex
	reconnectHandlers  []func()
	disconnectHandlers []func(error)
}

// NewMarketClient creates a new market data client.
func NewMarketClient(opts MarketClientOptions) *MarketClient {
	if opts.URL == "" {
		opts.URL = defaultMarketURL
	}
	return &MarketClient{
		opts:     opts,
		handlers: make(map[string][]EventHandler),
	}
}

// Connect establishes the WebSocket connection and starts the read pump.
func (c *MarketClient) Connect(ctx context.Context) error {
	c.url = c.opts.URL + "?hibachiClient=HibachiGoSDK/" + hibachi.Version
	conn, err := ConnectWithRetry(ctx, c.url, nil, c.opts.RetryOpts)
	if err != nil {
		return err
	}
	c.conn = conn
	c.pumpDone = make(chan error, 1)

	pumpCtx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel
	go c.readPump(pumpCtx)
	return nil
}

// Done returns a channel that receives the read pump error when it exits.
// Nil error means clean shutdown via context cancellation.
func (c *MarketClient) Done() <-chan error {
	return c.pumpDone
}

// Subscribe sends a subscribe request for the given subscriptions and
// tracks them for automatic re-subscription on reconnect.
func (c *MarketClient) Subscribe(ctx context.Context, subs ...hibachi.WSSubscription) error {
	if err := c.sendSubscribe(ctx, subs...); err != nil {
		return err
	}
	c.subsMu.Lock()
	c.subscriptions = append(c.subscriptions, subs...)
	c.subsMu.Unlock()
	return nil
}

// Unsubscribe sends an unsubscribe request and removes the subscriptions
// from the tracked set.
func (c *MarketClient) Unsubscribe(ctx context.Context, subs ...hibachi.WSSubscription) error {
	if err := c.sendUnsubscribe(ctx, subs...); err != nil {
		return err
	}
	c.subsMu.Lock()
	for _, unsub := range subs {
		for i := 0; i < len(c.subscriptions); i++ {
			if c.subscriptions[i].Topic == unsub.Topic && c.subscriptions[i].Symbol == unsub.Symbol {
				c.subscriptions = append(c.subscriptions[:i], c.subscriptions[i+1:]...)
				i--
			}
		}
	}
	c.subsMu.Unlock()
	return nil
}

// OnReconnect registers a callback that fires after a successful reconnect
// and re-subscribe.
func (c *MarketClient) OnReconnect(handler func()) {
	c.reconnectHandlers = append(c.reconnectHandlers, handler)
}

// OnDisconnect registers a callback that fires when the connection drops,
// before a reconnect attempt.
func (c *MarketClient) OnDisconnect(handler func(error)) {
	c.disconnectHandlers = append(c.disconnectHandlers, handler)
}

// On registers a handler for a given topic. Multiple handlers per topic are allowed.
func (c *MarketClient) On(topic string, handler EventHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handlers[topic] = append(c.handlers[topic], handler)
}

// Disconnect closes the connection and stops the read pump.
func (c *MarketClient) Disconnect() error {
	if c.cancel != nil {
		c.cancel()
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// readPump reads messages from the WebSocket and dispatches them to handlers.
func (c *MarketClient) readPump(ctx context.Context) {
	defer func() {
		select {
		case c.pumpDone <- nil:
		default:
		}
	}()

	for {
		var msg hibachi.WSMessage
		err := c.conn.ReadJSON(ctx, &msg)
		if err != nil {
			if ctx.Err() != nil {
				return
			}

			for _, h := range c.disconnectHandlers {
				h(err)
			}

			if reconnErr := c.reconnect(ctx); reconnErr != nil {
				select {
				case c.pumpDone <- fmt.Errorf("market ws reconnect failed: %w", reconnErr):
				default:
				}
				return
			}
			continue
		}

		c.mu.RLock()
		handlers := c.handlers[msg.Topic]
		if msg.Symbol != "" {
			handlers = append(handlers, c.handlers[msg.Topic+":"+msg.Symbol]...)
		}
		c.mu.RUnlock()

		for _, h := range handlers {
			h(msg.Data)
		}
	}
}

// reconnect closes the old connection, re-establishes it, re-subscribes,
// and fires reconnect handlers.
func (c *MarketClient) reconnect(ctx context.Context) error {
	if c.conn != nil {
		c.conn.Close()
	}

	conn, err := ConnectWithRetry(ctx, c.url, nil, c.opts.RetryOpts)
	if err != nil {
		return err
	}
	c.conn = conn

	c.subsMu.Lock()
	subs := make([]hibachi.WSSubscription, len(c.subscriptions))
	copy(subs, c.subscriptions)
	c.subsMu.Unlock()

	if len(subs) > 0 {
		if err := c.sendSubscribe(ctx, subs...); err != nil {
			return fmt.Errorf("re-subscribe: %w", err)
		}
	}

	for _, h := range c.reconnectHandlers {
		h()
	}

	return nil
}

// sendSubscribe sends a raw subscribe message without tracking.
func (c *MarketClient) sendSubscribe(ctx context.Context, subs ...hibachi.WSSubscription) error {
	msg := map[string]interface{}{
		"method": "subscribe",
		"parameters": map[string]interface{}{
			"subscriptions": subs,
		},
	}
	return c.conn.SendJSON(ctx, msg)
}

// sendUnsubscribe sends a raw unsubscribe message without tracking.
func (c *MarketClient) sendUnsubscribe(ctx context.Context, subs ...hibachi.WSSubscription) error {
	msg := map[string]interface{}{
		"method": "unsubscribe",
		"parameters": map[string]interface{}{
			"subscriptions": subs,
		},
	}
	return c.conn.SendJSON(ctx, msg)
}
