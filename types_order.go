package hibachi

import (
	"encoding/json"

	"github.com/shopspring/decimal"
)

// CreateOrder represents a request to create a new order.
type CreateOrder struct {
	Symbol           string            `json:"symbol"`
	Side             Side              `json:"side"`
	Quantity         decimal.Decimal   `json:"quantity"`
	MaxFeesPercent   decimal.Decimal   `json:"maxFeesPercent"`
	Price            *decimal.Decimal  `json:"price,omitempty"`
	TriggerPrice     *decimal.Decimal  `json:"triggerPrice,omitempty"`
	TriggerDirection *TriggerDirection `json:"triggerDirection,omitempty"`
	TWAPConfig       *TWAPConfig       `json:"twapConfig,omitempty"`
	CreationDeadline *int64            `json:"creationDeadline,omitempty"`
	ParentOrder      *OrderIdVariant   `json:"parentOrder,omitempty"`
	OrderFlags       *OrderFlags       `json:"orderFlags,omitempty"`
}

// UpdateOrder represents a request to update an existing order.
type UpdateOrder struct {
	OrderID          int64            `json:"orderId"`
	Symbol           string           `json:"symbol"`
	Side             Side             `json:"side"`
	Quantity         decimal.Decimal  `json:"quantity"`
	MaxFeesPercent   decimal.Decimal  `json:"maxFeesPercent"`
	Price            *decimal.Decimal `json:"price,omitempty"`
	TriggerPrice     *decimal.Decimal `json:"triggerPrice,omitempty"`
	CreationDeadline *int64           `json:"creationDeadline,omitempty"`
}

// CancelOrder represents a request to cancel an order.
type CancelOrder struct {
	OrderID *int64 `json:"orderId,omitempty"`
	Nonce   *int64 `json:"nonce,omitempty"`
}

// TWAPConfig represents TWAP order configuration.
type TWAPConfig struct {
	DurationMinutes int              `json:"durationMinutes"`
	QuantityMode    TWAPQuantityMode `json:"quantityMode"`
}

// ToMap converts TWAPConfig to a map for serialization.
func (t *TWAPConfig) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"durationMinutes": t.DurationMinutes,
		"quantityMode":    string(t.QuantityMode),
	}
}

// TPSLConfig represents take-profit/stop-loss configuration.
type TPSLConfig struct {
	Legs []TPSLLeg `json:"legs"`
}

// TPSLLeg represents a single take-profit or stop-loss leg.
type TPSLLeg struct {
	Type     string           `json:"type"`
	Price    decimal.Decimal  `json:"price"`
	Quantity *decimal.Decimal `json:"quantity,omitempty"`
}

// AddTakeProfit adds a take-profit leg and returns the config for chaining.
func (c *TPSLConfig) AddTakeProfit(price decimal.Decimal, quantity *decimal.Decimal) *TPSLConfig {
	c.Legs = append(c.Legs, TPSLLeg{
		Type:     "take_profit",
		Price:    price,
		Quantity: quantity,
	})
	return c
}

// AddStopLoss adds a stop-loss leg and returns the config for chaining.
func (c *TPSLConfig) AddStopLoss(price decimal.Decimal, quantity *decimal.Decimal) *TPSLConfig {
	c.Legs = append(c.Legs, TPSLLeg{
		Type:     "stop_loss",
		Price:    price,
		Quantity: quantity,
	})
	return c
}

// OrderIdVariant represents either a nonce or an order ID reference.
type OrderIdVariant struct {
	Nonce   *int64 `json:"nonce,omitempty"`
	OrderID *int64 `json:"orderId,omitempty"`
}

// FromNonce creates an OrderIdVariant from a nonce.
func FromNonce(nonce int64) OrderIdVariant {
	return OrderIdVariant{Nonce: &nonce}
}

// FromOrderID creates an OrderIdVariant from an order ID.
func FromOrderID(orderID int64) OrderIdVariant {
	return OrderIdVariant{OrderID: &orderID}
}

// ToMap converts OrderIdVariant to a map for serialization.
func (o *OrderIdVariant) ToMap() map[string]interface{} {
	m := make(map[string]interface{})
	if o.Nonce != nil {
		m["nonce"] = *o.Nonce
	}
	if o.OrderID != nil {
		m["orderId"] = *o.OrderID
	}
	return m
}

// PlaceOrderResult represents the result of placing an order.
type PlaceOrderResult struct {
	Nonce   string `json:"nonce"`
	OrderID string `json:"orderId"`
}

// BatchResponse represents a batch order response.
type BatchResponse struct {
	Orders []BatchResponseOrder `json:"orders"`
}

// BatchResponseOrder is the interface for batch response order types.
type BatchResponseOrder interface {
	isBatchResponseOrder()
}

// CreateOrderBatchResponse represents a successful create order in a batch.
type CreateOrderBatchResponse struct {
	Nonce   string `json:"nonce"`
	OrderID string `json:"orderId"`
}

func (CreateOrderBatchResponse) isBatchResponseOrder() {}

// UpdateOrderBatchResponse represents a successful update order in a batch.
type UpdateOrderBatchResponse struct {
	OrderID string `json:"orderId"`
}

func (UpdateOrderBatchResponse) isBatchResponseOrder() {}

// CancelOrderBatchResponse represents a successful cancel order in a batch.
type CancelOrderBatchResponse struct {
	Nonce string `json:"nonce"`
}

func (CancelOrderBatchResponse) isBatchResponseOrder() {}

// ErrorBatchResponse represents an error in a batch order.
type ErrorBatchResponse struct {
	ErrorCode int `json:"errorCode"`
}

func (ErrorBatchResponse) isBatchResponseOrder() {}

// DeserializeBatchResponseOrder deserializes a map into the appropriate BatchResponseOrder type.
func DeserializeBatchResponseOrder(data map[string]interface{}) BatchResponseOrder {
	if errorCode, ok := data["errorCode"]; ok {
		code := 0
		switch v := errorCode.(type) {
		case float64:
			code = int(v)
		case json.Number:
			n, _ := v.Int64()
			code = int(n)
		}
		return ErrorBatchResponse{ErrorCode: code}
	}

	_, hasNonce := data["nonce"]
	_, hasOrderID := data["orderId"]

	if hasNonce && hasOrderID {
		nonce, _ := data["nonce"].(string)
		orderID, _ := data["orderId"].(string)
		return CreateOrderBatchResponse{Nonce: nonce, OrderID: orderID}
	}

	if hasOrderID {
		orderID, _ := data["orderId"].(string)
		return UpdateOrderBatchResponse{OrderID: orderID}
	}

	if hasNonce {
		nonce, _ := data["nonce"].(string)
		return CancelOrderBatchResponse{Nonce: nonce}
	}

	return nil
}
