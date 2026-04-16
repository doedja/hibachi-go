package hibachi

import "testing"

func TestDeserializeBatchResponseOrder_Error(t *testing.T) {
	got := DeserializeBatchResponseOrder(map[string]interface{}{
		"errorCode": float64(7),
	})
	e, ok := got.(ErrorBatchResponse)
	if !ok {
		t.Fatalf("got %T, want ErrorBatchResponse", got)
	}
	if e.ErrorCode != 7 {
		t.Fatalf("code: got %d, want 7", e.ErrorCode)
	}
}

func TestDeserializeBatchResponseOrder_Create(t *testing.T) {
	got := DeserializeBatchResponseOrder(map[string]interface{}{
		"nonce":   "123",
		"orderId": "456",
	})
	c, ok := got.(CreateOrderBatchResponse)
	if !ok {
		t.Fatalf("got %T, want CreateOrderBatchResponse", got)
	}
	if c.Nonce != "123" || c.OrderID != "456" {
		t.Fatalf("fields: %+v", c)
	}
}

func TestDeserializeBatchResponseOrder_Update(t *testing.T) {
	got := DeserializeBatchResponseOrder(map[string]interface{}{
		"orderId": "789",
	})
	u, ok := got.(UpdateOrderBatchResponse)
	if !ok {
		t.Fatalf("got %T, want UpdateOrderBatchResponse", got)
	}
	if u.OrderID != "789" {
		t.Fatalf("orderId: got %q", u.OrderID)
	}
}

func TestDeserializeBatchResponseOrder_Cancel(t *testing.T) {
	got := DeserializeBatchResponseOrder(map[string]interface{}{
		"nonce": "999",
	})
	c, ok := got.(CancelOrderBatchResponse)
	if !ok {
		t.Fatalf("got %T, want CancelOrderBatchResponse", got)
	}
	if c.Nonce != "999" {
		t.Fatalf("nonce: got %q", c.Nonce)
	}
}

func TestDeserializeBatchResponseOrder_EmptyReturnsNil(t *testing.T) {
	if got := DeserializeBatchResponseOrder(map[string]interface{}{}); got != nil {
		t.Fatalf("got %v, want nil", got)
	}
}

func TestOrderIdVariant_ToMap(t *testing.T) {
	n := FromNonce(42)
	if m := n.ToMap(); m["nonce"] != int64(42) {
		t.Fatalf("FromNonce: %+v", m)
	}
	o := FromOrderID(99)
	if m := o.ToMap(); m["orderId"] != int64(99) {
		t.Fatalf("FromOrderID: %+v", m)
	}
}
