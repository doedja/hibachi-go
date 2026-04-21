package hibachi

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/shopspring/decimal"
)

// orderOptions holds optional parameters for limit orders.
type orderOptions struct {
	triggerPrice     *decimal.Decimal
	triggerDirection *TriggerDirection
	orderFlags       *OrderFlags
	creationDeadline *int64
	twapConfig       *TWAPConfig
	tpslConfig       *TPSLConfig
	parentOrder      *OrderIdVariant
}

// OrderOption is a functional option for PlaceLimitOrder.
type OrderOption func(*orderOptions)

// WithTriggerPrice sets a trigger price and direction for a conditional order.
func WithTriggerPrice(price decimal.Decimal, direction TriggerDirection) OrderOption {
	return func(o *orderOptions) {
		o.triggerPrice = &price
		o.triggerDirection = &direction
	}
}

// WithOrderFlags sets execution flags on the order.
func WithOrderFlags(flags OrderFlags) OrderOption {
	return func(o *orderOptions) {
		o.orderFlags = &flags
	}
}

// WithCreationDeadline sets a creation deadline as relative seconds from now.
func WithCreationDeadline(seconds int64) OrderOption {
	return func(o *orderOptions) {
		deadline := time.Now().Unix() + seconds
		o.creationDeadline = &deadline
	}
}

// WithTWAP sets TWAP configuration on the order.
func WithTWAP(config TWAPConfig) OrderOption {
	return func(o *orderOptions) {
		o.twapConfig = &config
	}
}

// WithTPSL sets take-profit/stop-loss configuration on the order.
func WithTPSL(config TPSLConfig) OrderOption {
	return func(o *orderOptions) {
		o.tpslConfig = &config
	}
}

// WithParentOrder sets a parent order reference for the order.
func WithParentOrder(variant OrderIdVariant) OrderOption {
	return func(o *orderOptions) {
		o.parentOrder = &variant
	}
}

// GetAccountInfo retrieves account information for the authenticated account.
func (c *Client) GetAccountInfo(ctx context.Context) (*AccountInfo, error) {
	if err := c.requireAuth(); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/trade/account/info?accountId=%d", c.accountID)
	data, err := c.transport.SendAuthorizedRequest(ctx, c.apiURL, http.MethodGet, path, nil, c.apiKey)
	if err != nil {
		return nil, err
	}

	var resp AccountInfo
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("unmarshaling account info: %w", err)
	}
	return &resp, nil
}

// GetAccountTrades retrieves trade history for the authenticated account.
func (c *Client) GetAccountTrades(ctx context.Context) ([]AccountTrade, error) {
	if err := c.requireAuth(); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/trade/account/trades?accountId=%d", c.accountID)
	data, err := c.transport.SendAuthorizedRequest(ctx, c.apiURL, http.MethodGet, path, nil, c.apiKey)
	if err != nil {
		return nil, err
	}

	// Server wraps trades in an object: {"trades": [...]}. Fall back to a
	// bare array for older endpoints.
	var wrapped struct {
		Trades []AccountTrade `json:"trades"`
	}
	if err := json.Unmarshal(data, &wrapped); err == nil && wrapped.Trades != nil {
		return wrapped.Trades, nil
	}
	var resp []AccountTrade
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("unmarshaling account trades: %w", err)
	}
	return resp, nil
}

// GetSettlementsHistory retrieves settlement history for the authenticated account.
func (c *Client) GetSettlementsHistory(ctx context.Context) ([]Settlement, error) {
	if err := c.requireAuth(); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/trade/account/settlements_history?accountId=%d", c.accountID)
	data, err := c.transport.SendAuthorizedRequest(ctx, c.apiURL, http.MethodGet, path, nil, c.apiKey)
	if err != nil {
		return nil, err
	}

	var resp []Settlement
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("unmarshaling settlements: %w", err)
	}
	return resp, nil
}

// GetPendingOrders retrieves pending orders for the authenticated account.
func (c *Client) GetPendingOrders(ctx context.Context) ([]Order, error) {
	if err := c.requireAuth(); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/trade/orders?accountId=%d", c.accountID)
	data, err := c.transport.SendAuthorizedRequest(ctx, c.apiURL, http.MethodGet, path, nil, c.apiKey)
	if err != nil {
		return nil, err
	}

	var resp []Order
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("unmarshaling pending orders: %w", err)
	}
	return resp, nil
}

// GetOrderDetails retrieves details of a specific order by order ID or nonce.
func (c *Client) GetOrderDetails(ctx context.Context, orderID *int64, nonce *int64) (*Order, error) {
	if err := c.requireAuth(); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/trade/order?accountId=%d", c.accountID)
	if orderID != nil {
		path += fmt.Sprintf("&orderId=%d", *orderID)
	} else if nonce != nil {
		path += fmt.Sprintf("&nonce=%d", *nonce)
	}

	data, err := c.transport.SendAuthorizedRequest(ctx, c.apiURL, http.MethodGet, path, nil, c.apiKey)
	if err != nil {
		return nil, err
	}

	var resp Order
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("unmarshaling order details: %w", err)
	}
	return &resp, nil
}

// PlaceMarketOrder places a market order.
func (c *Client) PlaceMarketOrder(ctx context.Context, symbol string, side Side, quantity, maxFeesPercent decimal.Decimal) (*PlaceOrderResult, error) {
	if err := c.requireAuth(); err != nil {
		return nil, err
	}

	contract, err := c.getContract(ctx, symbol)
	if err != nil {
		return nil, err
	}

	nonce := c.generateNonce()

	payload := CreateOrderPayload(
		nonce,
		contract.ID,
		quantity,
		side,
		nil,
		maxFeesPercent,
		contract.UnderlyingDecimals,
		contract.SettlementDecimals,
	)

	signature, err := c.signer.Sign(payload)
	if err != nil {
		return nil, fmt.Errorf("signing order: %w", err)
	}

	body := map[string]interface{}{
		"accountId":      c.accountID,
		"contractId":     contract.ID,
		"symbol":         symbol,
		"nonce":          nonce,
		"side":           string(side),
		"quantity":       FullPrecisionString(quantity),
		"maxFeesPercent": FullPrecisionString(maxFeesPercent),
		"signature":      signature,
		"orderType":      "MARKET",
	}

	data, err := c.transport.SendAuthorizedRequest(ctx, c.apiURL, http.MethodPost, "/trade/order", body, c.apiKey)
	if err != nil {
		return nil, err
	}

	var resp PlaceOrderResult
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("unmarshaling order result: %w", err)
	}
	return &resp, nil
}

// PlaceLimitOrder places a limit order with optional configurations.
func (c *Client) PlaceLimitOrder(ctx context.Context, symbol string, side Side, quantity, price, maxFeesPercent decimal.Decimal, opts ...OrderOption) (*PlaceOrderResult, error) {
	if err := c.requireAuth(); err != nil {
		return nil, err
	}

	contract, err := c.getContract(ctx, symbol)
	if err != nil {
		return nil, err
	}

	options := &orderOptions{}
	for _, opt := range opts {
		opt(options)
	}

	nonce := c.generateNonce()

	payload := CreateOrderPayload(
		nonce,
		contract.ID,
		quantity,
		side,
		&price,
		maxFeesPercent,
		contract.UnderlyingDecimals,
		contract.SettlementDecimals,
	)

	signature, err := c.signer.Sign(payload)
	if err != nil {
		return nil, fmt.Errorf("signing order: %w", err)
	}

	body := map[string]interface{}{
		"accountId":      c.accountID,
		"contractId":     contract.ID,
		"symbol":         symbol,
		"nonce":          nonce,
		"side":           string(side),
		"quantity":       FullPrecisionString(quantity),
		"price":          FullPrecisionString(price),
		"maxFeesPercent": FullPrecisionString(maxFeesPercent),
		"signature":      signature,
		"orderType":      "LIMIT",
	}

	if options.triggerPrice != nil {
		body["triggerPrice"] = FullPrecisionString(*options.triggerPrice)
	}
	if options.triggerDirection != nil {
		body["triggerDirection"] = string(*options.triggerDirection)
	}
	if options.orderFlags != nil {
		body["orderFlags"] = string(*options.orderFlags)
	}
	if options.creationDeadline != nil {
		body["creationDeadline"] = *options.creationDeadline
	}
	if options.twapConfig != nil {
		body["twapConfig"] = options.twapConfig.ToMap()
	}
	if options.tpslConfig != nil {
		legs := make([]map[string]interface{}, len(options.tpslConfig.Legs))
		for i, leg := range options.tpslConfig.Legs {
			m := map[string]interface{}{
				"type":  leg.Type,
				"price": FullPrecisionString(leg.Price),
			}
			if leg.Quantity != nil {
				m["quantity"] = FullPrecisionString(*leg.Quantity)
			}
			legs[i] = m
		}
		body["tpslConfig"] = map[string]interface{}{"legs": legs}
	}
	if options.parentOrder != nil {
		body["parentOrder"] = options.parentOrder.ToMap()
	}

	data, err := c.transport.SendAuthorizedRequest(ctx, c.apiURL, http.MethodPost, "/trade/order", body, c.apiKey)
	if err != nil {
		return nil, err
	}

	var resp PlaceOrderResult
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("unmarshaling order result: %w", err)
	}
	return &resp, nil
}

// UpdateOrder updates an existing order.
func (c *Client) UpdateOrder(ctx context.Context, update UpdateOrder) (*PlaceOrderResult, error) {
	if err := c.requireAuth(); err != nil {
		return nil, err
	}

	contract, err := c.getContract(ctx, update.Symbol)
	if err != nil {
		return nil, err
	}

	nonce := c.generateNonce()

	payload := CreateOrderPayload(
		nonce,
		contract.ID,
		update.Quantity,
		update.Side,
		update.Price,
		update.MaxFeesPercent,
		contract.UnderlyingDecimals,
		contract.SettlementDecimals,
	)

	signature, err := c.signer.Sign(payload)
	if err != nil {
		return nil, fmt.Errorf("signing order update: %w", err)
	}

	body := map[string]interface{}{
		"accountId":      c.accountID,
		"contractId":     contract.ID,
		"symbol":         update.Symbol,
		"orderId":        update.OrderID,
		"nonce":          nonce,
		"side":           string(update.Side),
		"quantity":       FullPrecisionString(update.Quantity),
		"maxFeesPercent": FullPrecisionString(update.MaxFeesPercent),
		"signature":      signature,
	}

	if update.Price != nil {
		body["price"] = FullPrecisionString(*update.Price)
	}
	if update.TriggerPrice != nil {
		body["triggerPrice"] = FullPrecisionString(*update.TriggerPrice)
	}
	if update.CreationDeadline != nil {
		body["creationDeadline"] = *update.CreationDeadline
	}

	data, err := c.transport.SendAuthorizedRequest(ctx, c.apiURL, http.MethodPut, "/trade/order", body, c.apiKey)
	if err != nil {
		return nil, err
	}

	var resp PlaceOrderResult
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("unmarshaling order update result: %w", err)
	}
	return &resp, nil
}

// CancelOrder cancels an order by order ID or nonce.
func (c *Client) CancelOrder(ctx context.Context, cancel CancelOrder) error {
	if err := c.requireAuth(); err != nil {
		return err
	}

	body := map[string]interface{}{
		"accountId": c.accountID,
	}

	// Server expects orderId and nonce as decimal strings.
	if cancel.OrderID != nil {
		body["orderId"] = fmt.Sprintf("%d", *cancel.OrderID)
	} else if cancel.Nonce != nil {
		body["nonce"] = fmt.Sprintf("%d", *cancel.Nonce)
	}

	// Server signs cancel with the 8-byte big-endian representation of the
	// target (orderId or nonce). The previous decimal-string signing produced
	// MacError on the server.
	var signData []byte
	if cancel.OrderID != nil {
		signData = make([]byte, 8)
		binary.BigEndian.PutUint64(signData, uint64(*cancel.OrderID))
	} else if cancel.Nonce != nil {
		signData = make([]byte, 8)
		binary.BigEndian.PutUint64(signData, uint64(*cancel.Nonce))
	}

	signature, err := c.signer.Sign(signData)
	if err != nil {
		return fmt.Errorf("signing cancel order: %w", err)
	}
	body["signature"] = signature

	_, err = c.transport.SendAuthorizedRequest(ctx, c.apiURL, http.MethodDelete, "/trade/order", body, c.apiKey)
	return err
}

// CancelAllOrders cancels all pending orders for the authenticated account.
func (c *Client) CancelAllOrders(ctx context.Context) error {
	if err := c.requireAuth(); err != nil {
		return err
	}

	body := map[string]interface{}{
		"accountId": c.accountID,
		"cancelAll": true,
	}

	_, err := c.transport.SendAuthorizedRequest(ctx, c.apiURL, http.MethodDelete, "/trade/order", body, c.apiKey)
	return err
}

// BatchOrderAction is an interface for batch order actions.
type BatchOrderAction interface {
	batchAction() string
}

func (c CreateOrder) batchAction() string { return "create" }
func (u UpdateOrder) batchAction() string { return "update" }
func (d CancelOrder) batchAction() string { return "cancel" }

// BatchOrders submits a batch of order actions (create, update, cancel).
func (c *Client) BatchOrders(ctx context.Context, orders []BatchOrderAction) (*BatchResponse, error) {
	if err := c.requireAuth(); err != nil {
		return nil, err
	}

	actions := make([]map[string]interface{}, 0, len(orders))

	for _, order := range orders {
		switch o := order.(type) {
		case CreateOrder:
			contract, err := c.getContract(ctx, o.Symbol)
			if err != nil {
				return nil, err
			}

			nonce := c.generateNonce()

			payload := CreateOrderPayload(
				nonce,
				contract.ID,
				o.Quantity,
				o.Side,
				o.Price,
				o.MaxFeesPercent,
				contract.UnderlyingDecimals,
				contract.SettlementDecimals,
			)

			signature, err := c.signer.Sign(payload)
			if err != nil {
				return nil, fmt.Errorf("signing batch create order: %w", err)
			}

			action := map[string]interface{}{
				"action":         "create",
				"accountId":      c.accountID,
				"contractId":     contract.ID,
				"nonce":          nonce,
				"side":           string(o.Side),
				"quantity":       FullPrecisionString(o.Quantity),
				"maxFeesPercent": FullPrecisionString(o.MaxFeesPercent),
				"signature":      signature,
			}

			if o.Price != nil {
				action["price"] = FullPrecisionString(*o.Price)
				action["orderType"] = "LIMIT"
			} else {
				action["orderType"] = "MARKET"
			}
			if o.TriggerPrice != nil {
				action["triggerPrice"] = FullPrecisionString(*o.TriggerPrice)
			}
			if o.TriggerDirection != nil {
				action["triggerDirection"] = string(*o.TriggerDirection)
			}
			if o.OrderFlags != nil {
				action["orderFlags"] = string(*o.OrderFlags)
			}
			if o.CreationDeadline != nil {
				action["creationDeadline"] = *o.CreationDeadline
			}
			if o.TWAPConfig != nil {
				action["twapConfig"] = o.TWAPConfig.ToMap()
			}
			if o.ParentOrder != nil {
				action["parentOrder"] = o.ParentOrder.ToMap()
			}

			actions = append(actions, action)

		case UpdateOrder:
			contract, err := c.getContract(ctx, o.Symbol)
			if err != nil {
				return nil, err
			}

			nonce := c.generateNonce()

			payload := CreateOrderPayload(
				nonce,
				contract.ID,
				o.Quantity,
				o.Side,
				o.Price,
				o.MaxFeesPercent,
				contract.UnderlyingDecimals,
				contract.SettlementDecimals,
			)

			signature, err := c.signer.Sign(payload)
			if err != nil {
				return nil, fmt.Errorf("signing batch update order: %w", err)
			}

			action := map[string]interface{}{
				"action":         "update",
				"accountId":      c.accountID,
				"contractId":     contract.ID,
				"orderId":        o.OrderID,
				"nonce":          nonce,
				"side":           string(o.Side),
				"quantity":       FullPrecisionString(o.Quantity),
				"maxFeesPercent": FullPrecisionString(o.MaxFeesPercent),
				"signature":      signature,
			}

			if o.Price != nil {
				action["price"] = FullPrecisionString(*o.Price)
			}
			if o.TriggerPrice != nil {
				action["triggerPrice"] = FullPrecisionString(*o.TriggerPrice)
			}
			if o.CreationDeadline != nil {
				action["creationDeadline"] = *o.CreationDeadline
			}

			actions = append(actions, action)

		case CancelOrder:
			action := map[string]interface{}{
				"action":    "cancel",
				"accountId": c.accountID,
			}

			if o.OrderID != nil {
				action["orderId"] = *o.OrderID
			}
			if o.Nonce != nil {
				action["nonce"] = *o.Nonce
			}

			var signData []byte
			if o.OrderID != nil {
				signData = []byte(fmt.Sprintf("%d", *o.OrderID))
			} else if o.Nonce != nil {
				signData = []byte(fmt.Sprintf("%d", *o.Nonce))
			}

			signature, err := c.signer.Sign(signData)
			if err != nil {
				return nil, fmt.Errorf("signing batch cancel order: %w", err)
			}
			action["signature"] = signature

			actions = append(actions, action)
		}
	}

	body := map[string]interface{}{
		"accountId": c.accountID,
		"orders":    actions,
	}

	data, err := c.transport.SendAuthorizedRequest(ctx, c.apiURL, http.MethodPost, "/trade/orders", body, c.apiKey)
	if err != nil {
		return nil, err
	}

	// Parse the batch response - it's an array of order results
	var rawOrders []map[string]interface{}
	if err := json.Unmarshal(data, &rawOrders); err != nil {
		// Try parsing as an object with "orders" field
		var wrapper struct {
			Orders []map[string]interface{} `json:"orders"`
		}
		if err2 := json.Unmarshal(data, &wrapper); err2 != nil {
			return nil, fmt.Errorf("unmarshaling batch response: %w", err)
		}
		rawOrders = wrapper.Orders
	}

	resp := &BatchResponse{
		Orders: make([]BatchResponseOrder, 0, len(rawOrders)),
	}
	for _, raw := range rawOrders {
		order := DeserializeBatchResponseOrder(raw)
		if order != nil {
			resp.Orders = append(resp.Orders, order)
		}
	}

	return resp, nil
}
