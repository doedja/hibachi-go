package hibachi

import (
	"fmt"
	"time"
)

// HibachiError is the base error type for all SDK errors.
type HibachiError struct {
	Message string
}

func (e *HibachiError) Error() string {
	return e.Message
}

// ExchangeError represents an error returned by the exchange server.
type ExchangeError struct {
	HibachiError
}

func (e *ExchangeError) Unwrap() error {
	return &e.HibachiError
}

// MaintenanceError represents an exchange maintenance window.
type MaintenanceError struct {
	ExchangeError
	StartTime *time.Time
	EndTime   *time.Time
}

func (e *MaintenanceError) Unwrap() error {
	return &e.ExchangeError
}

func (e *MaintenanceError) Error() string {
	if e.StartTime != nil && e.EndTime != nil {
		return fmt.Sprintf("%s (maintenance: %s - %s)", e.Message, e.StartTime.Format(time.RFC3339), e.EndTime.Format(time.RFC3339))
	}
	return e.Message
}

// BadHTTPStatusError represents an error with a specific HTTP status code.
type BadHTTPStatusError struct {
	ExchangeError
	StatusCode int
}

func (e *BadHTTPStatusError) Unwrap() error {
	return &e.ExchangeError
}

func (e *BadHTTPStatusError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Message)
}

// BadRequestError represents a 400 Bad Request error.
type BadRequestError struct {
	BadHTTPStatusError
}

func (e *BadRequestError) Unwrap() error {
	return &e.BadHTTPStatusError
}

// UnauthorizedError represents a 401 Unauthorized error.
type UnauthorizedError struct {
	BadHTTPStatusError
}

func (e *UnauthorizedError) Unwrap() error {
	return &e.BadHTTPStatusError
}

// ForbiddenError represents a 403 Forbidden error.
type ForbiddenError struct {
	BadHTTPStatusError
}

func (e *ForbiddenError) Unwrap() error {
	return &e.BadHTTPStatusError
}

// NotFoundError represents a 404 Not Found error.
type NotFoundError struct {
	BadHTTPStatusError
}

func (e *NotFoundError) Unwrap() error {
	return &e.BadHTTPStatusError
}

// RateLimitedError represents a 429 Too Many Requests error.
type RateLimitedError struct {
	BadHTTPStatusError
}

func (e *RateLimitedError) Unwrap() error {
	return &e.BadHTTPStatusError
}

// InternalServerErrorHTTP represents a 500 Internal Server Error.
type InternalServerErrorHTTP struct {
	BadHTTPStatusError
}

func (e *InternalServerErrorHTTP) Unwrap() error {
	return &e.BadHTTPStatusError
}

// BadGatewayError represents a 502 Bad Gateway error.
type BadGatewayError struct {
	BadHTTPStatusError
}

func (e *BadGatewayError) Unwrap() error {
	return &e.BadHTTPStatusError
}

// ServiceUnavailableError represents a 503 Service Unavailable error.
type ServiceUnavailableError struct {
	BadHTTPStatusError
}

func (e *ServiceUnavailableError) Unwrap() error {
	return &e.BadHTTPStatusError
}

// GatewayTimeoutError represents a 504 Gateway Timeout error.
type GatewayTimeoutError struct {
	BadHTTPStatusError
}

func (e *GatewayTimeoutError) Unwrap() error {
	return &e.BadHTTPStatusError
}

// BadWSResponseError represents a bad WebSocket response from the exchange.
type BadWSResponseError struct {
	ExchangeError
}

func (e *BadWSResponseError) Unwrap() error {
	return &e.ExchangeError
}

// TransportError represents a transport-level error.
type TransportError struct {
	HibachiError
}

func (e *TransportError) Unwrap() error {
	return &e.HibachiError
}

// ConnectionError represents a connection failure.
type ConnectionError struct {
	TransportError
}

func (e *ConnectionError) Unwrap() error {
	return &e.TransportError
}

// TimeoutError represents a timeout.
type TimeoutError struct {
	TransportError
}

func (e *TimeoutError) Unwrap() error {
	return &e.TransportError
}

// WSConnectionError represents a WebSocket connection failure.
type WSConnectionError struct {
	TransportError
}

func (e *WSConnectionError) Unwrap() error {
	return &e.TransportError
}

// WSMessageError represents a WebSocket message error.
type WSMessageError struct {
	TransportError
}

func (e *WSMessageError) Unwrap() error {
	return &e.TransportError
}

// DeserializationError represents a deserialization failure.
type DeserializationError struct {
	TransportError
}

func (e *DeserializationError) Unwrap() error {
	return &e.TransportError
}

// SerializationError represents a serialization failure.
type SerializationError struct {
	TransportError
}

func (e *SerializationError) Unwrap() error {
	return &e.TransportError
}

// ValidationError represents a validation failure.
type ValidationError struct {
	HibachiError
}

func (e *ValidationError) Unwrap() error {
	return &e.HibachiError
}

// MissingCredentialsError represents missing API credentials.
type MissingCredentialsError struct {
	ValidationError
}

func (e *MissingCredentialsError) Unwrap() error {
	return &e.ValidationError
}

// APIError represents an API-level error returned in the response body.
type APIError struct {
	ExchangeError
	Code int
}

func (e *APIError) Unwrap() error {
	return &e.ExchangeError
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error %d: %s", e.Code, e.Message)
}

// NewHTTPError creates the appropriate error type based on HTTP status code.
func NewHTTPError(statusCode int, message string) error {
	base := BadHTTPStatusError{
		ExchangeError: ExchangeError{
			HibachiError: HibachiError{Message: message},
		},
		StatusCode: statusCode,
	}

	switch statusCode {
	case 400:
		return &BadRequestError{BadHTTPStatusError: base}
	case 401:
		return &UnauthorizedError{BadHTTPStatusError: base}
	case 403:
		return &ForbiddenError{BadHTTPStatusError: base}
	case 404:
		return &NotFoundError{BadHTTPStatusError: base}
	case 429:
		return &RateLimitedError{BadHTTPStatusError: base}
	case 500:
		return &InternalServerErrorHTTP{BadHTTPStatusError: base}
	case 502:
		return &BadGatewayError{BadHTTPStatusError: base}
	case 503:
		return &ServiceUnavailableError{BadHTTPStatusError: base}
	case 504:
		return &GatewayTimeoutError{BadHTTPStatusError: base}
	default:
		return &base
	}
}
