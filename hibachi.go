// Package hibachi provides a Go SDK for the Hibachi decentralized perpetual futures exchange.
//
// The SDK supports REST API endpoints for market data, trading, and capital management,
// as well as WebSocket clients for real-time market data, account streaming, and trade execution.
//
// # Quick Start
//
// Create a client for public market data (no authentication required):
//
//	client, err := hibachi.NewClient()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	info, err := client.GetExchangeInfo(ctx)
//
// Create an authenticated client for trading:
//
//	client, err := hibachi.NewClient(
//	    hibachi.WithAPIKey("your-api-key"),
//	    hibachi.WithAccountID(12345),
//	    hibachi.WithPrivateKey("0x..."),
//	)
//
// For WebSocket clients, see the ws sub-package.
package hibachi

// Version is the SDK version string.
const Version = "0.1.0"
