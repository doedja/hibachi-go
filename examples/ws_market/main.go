package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"

	hibachi "github.com/doedja/hibachi-go"
	"github.com/doedja/hibachi-go/ws"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	client := ws.NewMarketClient(ws.MarketClientOptions{})

	if err := client.Connect(ctx); err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect()

	// Register handlers before subscribing.
	client.On("mark_price", func(data json.RawMessage) {
		fmt.Printf("Mark price: %s\n", string(data))
	})

	client.On("trades", func(data json.RawMessage) {
		fmt.Printf("Trade: %s\n", string(data))
	})

	// Subscribe to topics.
	err := client.Subscribe(ctx,
		hibachi.WSSubscription{Topic: hibachi.WSTopicMarkPrice, Symbol: "BTC/USDT-P"},
		hibachi.WSSubscription{Topic: hibachi.WSTopicTrades, Symbol: "BTC/USDT-P"},
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Listening for market data (Ctrl+C to stop)...")
	<-ctx.Done()
}
