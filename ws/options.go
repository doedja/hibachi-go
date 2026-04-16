package ws

import hibachi "github.com/doedja/hibachi-go"

const (
	defaultMarketURL  = "wss://data-api.hibachi.xyz/ws/market"
	defaultAccountURL = "wss://api.hibachi.xyz/ws/account"
	defaultTradeURL   = "wss://api.hibachi.xyz/ws/trade"
)

// MarketClientOptions configures the market data WebSocket client.
type MarketClientOptions struct {
	URL       string
	RetryOpts RetryOptions
}

// AccountClientOptions configures the account stream WebSocket client.
type AccountClientOptions struct {
	URL       string
	APIKey    string
	AccountID int
	RetryOpts RetryOptions
}

// TradeClientOptions configures the trade WebSocket client.
type TradeClientOptions struct {
	URL       string
	APIKey    string
	AccountID int
	Signer    hibachi.Signer
	RetryOpts RetryOptions
}
