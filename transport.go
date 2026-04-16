package hibachi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// HTTPTransport is the interface for making HTTP requests.
type HTTPTransport interface {
	// SendSimpleRequest sends an unauthenticated GET request to the data API.
	SendSimpleRequest(ctx context.Context, baseURL, path string) ([]byte, error)

	// SendAuthorizedRequest sends an authenticated request with API key.
	SendAuthorizedRequest(ctx context.Context, baseURL, method, path string, body interface{}, apiKey string) ([]byte, error)
}

// defaultTransport implements HTTPTransport using net/http.
type defaultTransport struct {
	client *http.Client
}

func (t *defaultTransport) SendSimpleRequest(ctx context.Context, baseURL, path string) ([]byte, error) {
	url := baseURL + path

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, NewHTTPError(resp.StatusCode, string(respBody))
	}

	return extractData(respBody)
}

func (t *defaultTransport) SendAuthorizedRequest(ctx context.Context, baseURL, method, path string, body interface{}, apiKey string) ([]byte, error) {
	url := baseURL + path

	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", apiKey)

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, NewHTTPError(resp.StatusCode, string(respBody))
	}

	return extractData(respBody)
}

// extractData parses the JSON response and extracts the data field.
// It also checks for API-level errors and maintenance status.
func extractData(body []byte) ([]byte, error) {
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err != nil {
		// If it's not a JSON object, return as-is
		return body, nil
	}

	// Check for maintenance/error status (string status values)
	if statusRaw, ok := envelope["status"]; ok {
		var status string
		if json.Unmarshal(statusRaw, &status) == nil {
			switch status {
			case "MAINTENANCE", "SCHEDULED_MAINTENANCE", "UNSCHEDULED_MAINTENANCE":
				return nil, &MaintenanceError{ExchangeError: ExchangeError{HibachiError: HibachiError{Message: "exchange is under maintenance"}}}
			case "error", "failed":
				msg := "unknown error"
				if msgRaw, ok := envelope["message"]; ok {
					_ = json.Unmarshal(msgRaw, &msg)
				}
				code := 0
				if codeRaw, ok := envelope["errorCode"]; ok {
					_ = json.Unmarshal(codeRaw, &code)
				} else if codeRaw, ok := envelope["code"]; ok {
					_ = json.Unmarshal(codeRaw, &code)
				}
				return nil, &APIError{ExchangeError: ExchangeError{HibachiError: HibachiError{Message: msg}}, Code: code}
			}
		}
	}

	// Return "data" field if present
	if data, ok := envelope["data"]; ok {
		return data, nil
	}

	// Return "result" field if present
	if result, ok := envelope["result"]; ok {
		return result, nil
	}

	// Return the full body if no envelope structure found
	return body, nil
}
