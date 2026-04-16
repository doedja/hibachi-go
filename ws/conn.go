package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// Conn wraps a WebSocket connection with mutex-protected writes.
type Conn struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

// SendJSON marshals v to JSON and sends it as a text message.
func (c *Conn) SendJSON(ctx context.Context, v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.Write(ctx, websocket.MessageText, data)
}

// ReadJSON reads a text message and unmarshals JSON into v.
func (c *Conn) ReadJSON(ctx context.Context, v interface{}) error {
	_, data, err := c.conn.Read(ctx)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// Close closes the underlying WebSocket connection.
func (c *Conn) Close() error {
	return c.conn.Close(websocket.StatusNormalClosure, "")
}

// RetryOptions configures exponential backoff for connection retries.
type RetryOptions struct {
	MaxRetries    int
	InitialDelay  time.Duration
	BackoffFactor float64
	MaxDelay      time.Duration
}

// DefaultRetryOptions returns sensible defaults for retry behavior.
func DefaultRetryOptions() RetryOptions {
	return RetryOptions{
		MaxRetries:    10,
		InitialDelay:  time.Second,
		BackoffFactor: 1.5,
		MaxDelay:      30 * time.Second,
	}
}

// ConnectWithRetry dials a WebSocket endpoint with exponential backoff.
// Set MaxRetries to -1 for infinite retries. A zero-valued RetryOptions
// uses DefaultRetryOptions.
func ConnectWithRetry(ctx context.Context, url string, headers http.Header, opts RetryOptions) (*Conn, error) {
	if opts == (RetryOptions{}) {
		opts = DefaultRetryOptions()
	}

	var lastErr error
	for attempt := 0; opts.MaxRetries < 0 || attempt <= opts.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(float64(opts.InitialDelay) * math.Pow(opts.BackoffFactor, float64(attempt-1)))
			if delay > opts.MaxDelay {
				delay = opts.MaxDelay
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		dialOpts := &websocket.DialOptions{}
		if headers != nil {
			dialOpts.HTTPHeader = headers
		}

		conn, _, err := websocket.Dial(ctx, url, dialOpts)
		if err != nil {
			lastErr = err
			continue
		}
		// Raise read limit from default 32KB to 1MB to handle large orderbook snapshots.
		conn.SetReadLimit(1 << 20)
		return &Conn{conn: conn}, nil
	}

	return nil, fmt.Errorf("failed to connect after %d retries: %w", opts.MaxRetries, lastErr)
}
