package hibachi

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

const (
	defaultAPIURL     = "https://api.hibachi.xyz"
	defaultDataAPIURL = "https://data-api.hibachi.xyz"
	defaultTimeout    = 30 * time.Second
)

// Client is the Hibachi REST API client.
type Client struct {
	apiURL     string
	dataAPIURL string
	apiKey     string
	accountID  int
	privateKey string
	signer     Signer
	transport  HTTPTransport

	exchangeInfoMu sync.Mutex
	exchangeInfo   *ExchangeInfo
}

// Option configures the Client.
type Option func(*Client)

// WithAPIURL sets the API base URL.
func WithAPIURL(url string) Option {
	return func(c *Client) {
		c.apiURL = url
	}
}

// WithDataAPIURL sets the data API base URL.
func WithDataAPIURL(url string) Option {
	return func(c *Client) {
		c.dataAPIURL = url
	}
}

// WithAPIKey sets the API key for authenticated requests.
func WithAPIKey(key string) Option {
	return func(c *Client) {
		c.apiKey = key
	}
}

// WithAccountID sets the account ID.
func WithAccountID(id int) Option {
	return func(c *Client) {
		c.accountID = id
	}
}

// WithPrivateKey sets the private key for signing requests.
func WithPrivateKey(key string) Option {
	return func(c *Client) {
		c.privateKey = key
	}
}

// WithHTTPClient sets a custom HTTP client for the default transport.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		c.transport = &defaultTransport{client: httpClient}
	}
}

// WithHTTPTransport sets a custom HTTP transport implementation.
func WithHTTPTransport(t HTTPTransport) Option {
	return func(c *Client) {
		c.transport = t
	}
}

// NewClient creates a new Hibachi REST API client.
func NewClient(opts ...Option) (*Client, error) {
	c := &Client{
		apiURL:     defaultAPIURL,
		dataAPIURL: defaultDataAPIURL,
	}

	for _, opt := range opts {
		opt(c)
	}

	if c.transport == nil {
		c.transport = &defaultTransport{
			client: &http.Client{Timeout: defaultTimeout},
		}
	}

	if c.privateKey != "" {
		signer, err := NewSigner(c.privateKey)
		if err != nil {
			return nil, fmt.Errorf("creating signer: %w", err)
		}
		c.signer = signer
	}

	return c, nil
}

// requireAuth checks that the client has credentials for authenticated endpoints.
func (c *Client) requireAuth() error {
	if c.apiKey == "" || c.accountID == 0 || c.signer == nil {
		return &MissingCredentialsError{
			ValidationError: ValidationError{
				HibachiError: HibachiError{Message: "apiKey, accountID, and privateKey are required for authenticated endpoints"},
			},
		}
	}
	return nil
}

// getContract looks up contract info by symbol from the cached exchange info.
func (c *Client) getContract(ctx context.Context, symbol string) (*FutureContract, error) {
	info, err := c.GetExchangeInfo(ctx)
	if err != nil {
		return nil, err
	}

	for i := range info.FutureContracts {
		if info.FutureContracts[i].Symbol == symbol {
			return &info.FutureContracts[i], nil
		}
	}

	return nil, fmt.Errorf("contract not found: %s", symbol)
}

// generateNonce generates a unique nonce based on the current time.
func (c *Client) generateNonce() int64 {
	return time.Now().UnixNano() / 1000
}
