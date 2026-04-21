package hibachi

import "encoding/json"

// WSSubscription represents a WebSocket subscription request.
// Granularity is a string on the wire (server accepts e.g. "1", "10", "0.1").
type WSSubscription struct {
	Topic       WSSubscriptionTopic `json:"topic"`
	Symbol      string              `json:"symbol"`
	Interval    *Interval           `json:"interval,omitempty"`
	Depth       *int                `json:"depth,omitempty"`
	Granularity *string             `json:"granularity,omitempty"`
}

// WSResponse represents a WebSocket response message.
type WSResponse struct {
	ID     int             `json:"id"`
	Status int             `json:"status"`
	Result json.RawMessage `json:"result"`
	Error  json.RawMessage `json:"error"`
}

// WSMessage represents an incoming WebSocket market data message.
type WSMessage struct {
	Topic  string          `json:"topic"`
	Symbol string          `json:"symbol"`
	Data   json.RawMessage `json:"data"`
}

// OrderPlaceParams represents parameters for placing an order via WebSocket.
type OrderPlaceParams struct {
	AccountID        int                     `json:"accountId"`
	Nonce            int64                   `json:"nonce"`
	Symbol           string                  `json:"symbol"`
	OrderType        string                  `json:"orderType"`
	Quantity         string                  `json:"quantity"`
	Side             string                  `json:"side"`
	Price            *string                 `json:"price,omitempty"`
	MaxFeesPercent   string                  `json:"maxFeesPercent"`
	Signature        string                  `json:"-"`
	TriggerPrice     *string                 `json:"triggerPrice,omitempty"`
	TriggerDirection *string                 `json:"triggerDirection,omitempty"`
	CreationDeadline *string                 `json:"creationDeadline,omitempty"`
	TWAPConfig       *map[string]interface{} `json:"twapConfig,omitempty"`
	OrderFlags       *string                 `json:"orderFlags,omitempty"`
}

// OrderModifyParams represents parameters for modifying an order via WebSocket.
type OrderModifyParams struct {
	AccountID      int     `json:"accountId"`
	OrderID        string  `json:"orderId"`
	Nonce          int64   `json:"nonce"`
	Symbol         string  `json:"symbol"`
	Quantity       string  `json:"quantity"`
	Side           string  `json:"side"`
	Price          *string `json:"price,omitempty"`
	MaxFeesPercent string  `json:"maxFeesPercent"`
	Signature      string  `json:"-"`
}

// AccountStreamStartResult represents the result of starting an account stream.
type AccountStreamStartResult struct {
	AccountSnapshot AccountSnapshot `json:"accountSnapshot"`
	ListenKey       string          `json:"listenKey"`
}

// AccountSnapshot represents the current state of the account.
type AccountSnapshot struct {
	AccountID int        `json:"accountId"`
	Balance   string     `json:"balance"`
	Positions []Position `json:"positions"`
}
