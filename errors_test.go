package hibachi

import (
	"errors"
	"testing"
)

func TestNewHTTPError_MapsKnownStatus(t *testing.T) {
	cases := []struct {
		status int
		check  func(error) bool
	}{
		{400, func(e error) bool { var t *BadRequestError; return errors.As(e, &t) }},
		{401, func(e error) bool { var t *UnauthorizedError; return errors.As(e, &t) }},
		{403, func(e error) bool { var t *ForbiddenError; return errors.As(e, &t) }},
		{404, func(e error) bool { var t *NotFoundError; return errors.As(e, &t) }},
		{429, func(e error) bool { var t *RateLimitedError; return errors.As(e, &t) }},
		{500, func(e error) bool { var t *InternalServerErrorHTTP; return errors.As(e, &t) }},
		{502, func(e error) bool { var t *BadGatewayError; return errors.As(e, &t) }},
		{503, func(e error) bool { var t *ServiceUnavailableError; return errors.As(e, &t) }},
		{504, func(e error) bool { var t *GatewayTimeoutError; return errors.As(e, &t) }},
	}
	for _, tc := range cases {
		err := NewHTTPError(tc.status, "boom")
		if !tc.check(err) {
			t.Errorf("status %d: got %T, wanted specific type", tc.status, err)
		}
	}
}

func TestNewHTTPError_UnknownStatusFallsBackToBase(t *testing.T) {
	err := NewHTTPError(418, "teapot")
	var base *BadHTTPStatusError
	if !errors.As(err, &base) {
		t.Fatalf("expected BadHTTPStatusError, got %T", err)
	}
	if base.StatusCode != 418 {
		t.Fatalf("status: got %d, want 418", base.StatusCode)
	}
}

func TestErrorHierarchy_UnwrapsToHibachiError(t *testing.T) {
	// Every concrete error should unwrap to the HibachiError root via errors.As.
	err := NewHTTPError(404, "missing")
	var root *HibachiError
	if !errors.As(err, &root) {
		t.Fatalf("expected to unwrap to *HibachiError, got %T", err)
	}
	if root.Message != "missing" {
		t.Fatalf("message: got %q, want 'missing'", root.Message)
	}
}

func TestWSConnectionError_IsTransportError(t *testing.T) {
	err := &WSConnectionError{
		TransportError: TransportError{
			HibachiError: HibachiError{Message: "conn closed"},
		},
	}
	var transport *TransportError
	if !errors.As(err, &transport) {
		t.Fatal("expected WSConnectionError to unwrap to *TransportError")
	}
}

func TestAPIError_FormattedMessage(t *testing.T) {
	e := &APIError{
		ExchangeError: ExchangeError{HibachiError: HibachiError{Message: "not allowed"}},
		Code:          4,
	}
	got := e.Error()
	want := "API error 4: not allowed"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
