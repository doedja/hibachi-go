package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"

	"github.com/doedja/hibachi-go/ws"
)

func main() {
	apiKey := os.Getenv("HIBACHI_API_KEY")
	accountIDStr := os.Getenv("HIBACHI_ACCOUNT_ID")
	if apiKey == "" || accountIDStr == "" {
		log.Fatal("Set HIBACHI_API_KEY and HIBACHI_ACCOUNT_ID")
	}
	accountID, err := strconv.Atoi(accountIDStr)
	if err != nil {
		log.Fatalf("HIBACHI_ACCOUNT_ID: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	client := ws.NewAccountClient(ws.AccountClientOptions{
		APIKey:    apiKey,
		AccountID: accountID,
	})

	if err := client.Connect(ctx); err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect()

	// Start the account stream and get the initial snapshot.
	result, err := client.StreamStart(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Account %d balance: %s\n", result.AccountSnapshot.AccountID, result.AccountSnapshot.Balance)
	fmt.Printf("Positions: %d\n", len(result.AccountSnapshot.Positions))
	for _, p := range result.AccountSnapshot.Positions {
		fmt.Printf("  %s: qty=%s dir=%s open=%s trading_pnl=%s funding_pnl=%s\n",
			p.Symbol, p.Quantity, p.Direction, p.OpenPrice, p.UnrealizedTradingPnl, p.UnrealizedFundingPnl)
	}

	// Register handlers for account updates.
	client.On("balance", func(data json.RawMessage) {
		fmt.Printf("Balance update: %s\n", string(data))
	})

	client.On("position", func(data json.RawMessage) {
		fmt.Printf("Position update: %s\n", string(data))
	})

	client.On("order", func(data json.RawMessage) {
		fmt.Printf("Order update: %s\n", string(data))
	})

	// Listen for updates until interrupted.
	fmt.Println("Listening for account updates (Ctrl+C to stop)...")
	if err := client.ListenLoop(ctx); err != nil && ctx.Err() == nil {
		log.Fatal(err)
	}
}
