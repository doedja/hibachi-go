package main

import (
	"context"
	"fmt"
	"log"

	hibachi "github.com/doedja/hibachi-go"
)

func main() {
	ctx := context.Background()

	// Public market data — no authentication required.
	client, err := hibachi.NewClient()
	if err != nil {
		log.Fatal(err)
	}

	// Get exchange info (cached after first call).
	info, err := client.GetExchangeInfo(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Exchange has %d contracts\n", len(info.FutureContracts))
	for _, c := range info.FutureContracts {
		fmt.Printf("  %s (id=%d, settlement=%s)\n", c.Symbol, c.ID, c.SettlementSymbol)
	}

	// Get current prices for a symbol.
	prices, err := client.GetPrices(ctx, "BTC/USDT-P")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\nBTC/USDT-P mark=%s spot=%s\n", prices.MarkPrice, prices.SpotPrice)

	// Get order book.
	book, err := client.GetOrderbook(ctx, "BTC/USDT-P", 5, 1)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\nOrderbook (top 5):\n")
	for _, ask := range book.Ask.Levels {
		fmt.Printf("  ASK %s @ %s\n", ask.Quantity, ask.Price)
	}
	for _, bid := range book.Bid.Levels {
		fmt.Printf("  BID %s @ %s\n", bid.Quantity, bid.Price)
	}

	// --- Authenticated trading (uncomment with real credentials) ---
	//
	// authClient, err := hibachi.NewClient(
	//     hibachi.WithAPIKey("your-api-key"),
	//     hibachi.WithAccountID(12345),
	//     hibachi.WithPrivateKey("0xYourPrivateKey"),
	// )
	// if err != nil {
	//     log.Fatal(err)
	// }
	//
	// result, err := authClient.PlaceLimitOrder(ctx,
	//     "BTC/USDT-P",
	//     hibachi.SideBid,
	//     decimal.NewFromFloat(0.001),
	//     decimal.NewFromFloat(50000),
	//     decimal.NewFromFloat(0.1),
	//     hibachi.WithOrderFlags(hibachi.OrderFlagsPostOnly),
	// )
	// if err != nil {
	//     log.Fatal(err)
	// }
	// fmt.Printf("Order placed: orderId=%s nonce=%s\n", result.OrderID, result.Nonce)
}
