package hibachi

import (
	"errors"
	"testing"
)

func TestExtractData_UnwrapsDataField(t *testing.T) {
	body := []byte(`{"status":"ok","data":{"price":"1"}}`)
	out, err := extractData(body)
	if err != nil {
		t.Fatalf("extractData: %v", err)
	}
	if string(out) != `{"price":"1"}` {
		t.Fatalf("got %s, want {\"price\":\"1\"}", out)
	}
}

func TestExtractData_UnwrapsResultField(t *testing.T) {
	body := []byte(`{"result":{"orderId":"42"}}`)
	out, err := extractData(body)
	if err != nil {
		t.Fatalf("extractData: %v", err)
	}
	if string(out) != `{"orderId":"42"}` {
		t.Fatalf("got %s", out)
	}
}

func TestExtractData_ReturnsFullBodyWhenNoEnvelope(t *testing.T) {
	// Hibachi's market/exchange-info has no "data" or "result" field — the
	// caller must receive the whole body.
	body := []byte(`{"status":"NORMAL","futureContracts":[]}`)
	out, err := extractData(body)
	if err != nil {
		t.Fatalf("extractData: %v", err)
	}
	if string(out) != string(body) {
		t.Fatalf("got %s, want full body", out)
	}
}

func TestExtractData_MaintenanceStatus(t *testing.T) {
	for _, s := range []string{"MAINTENANCE", "SCHEDULED_MAINTENANCE", "UNSCHEDULED_MAINTENANCE"} {
		body := []byte(`{"status":"` + s + `"}`)
		_, err := extractData(body)
		var me *MaintenanceError
		if !errors.As(err, &me) {
			t.Errorf("status %s: got %T, want *MaintenanceError", s, err)
		}
	}
}

func TestExtractData_FailedStatusReturnsAPIError(t *testing.T) {
	// Real Hibachi error shape: {"errorCode":4,"message":"...","status":"failed"}
	body := []byte(`{"errorCode":4,"message":"bad interval","status":"failed"}`)
	_, err := extractData(body)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("got %T, want *APIError", err)
	}
	if apiErr.Code != 4 {
		t.Fatalf("code: got %d, want 4", apiErr.Code)
	}
	if apiErr.Message != "bad interval" {
		t.Fatalf("message: got %q, want 'bad interval'", apiErr.Message)
	}
}

func TestExtractData_NonJSONReturnsBytesAsIs(t *testing.T) {
	body := []byte("plain text response")
	out, err := extractData(body)
	if err != nil {
		t.Fatalf("extractData: %v", err)
	}
	if string(out) != string(body) {
		t.Fatalf("got %s, want raw body", out)
	}
}
