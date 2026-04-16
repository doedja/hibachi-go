package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/shopspring/decimal"

	hibachi "github.com/doedja/hibachi-go"
	"github.com/doedja/hibachi-go/ws"
)

func main() {
	apiKey := os.Getenv("HIBACHI_API_KEY")
	privateKey := os.Getenv("HIBACHI_PRIVATE_KEY")
	accountIDStr := os.Getenv("HIBACHI_ACCOUNT_ID")
	if apiKey == "" || privateKey == "" || accountIDStr == "" {
		log.Fatal("Set HIBACHI_API_KEY, HIBACHI_PRIVATE_KEY, HIBACHI_ACCOUNT_ID")
	}
	accountID, err := strconv.Atoi(accountIDStr)
	if err != nil {
		log.Fatalf("HIBACHI_ACCOUNT_ID: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	signer, err := hibachi.NewSigner(privateKey)
	if err != nil {
		log.Fatal(err)
	}

	// Fetch contract metadata (id + decimals) via REST. The trade WebSocket
	// takes symbols, but the signature payload is built from contractID and
	// the contract's decimals.
	rest, err := hibachi.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	info, err := rest.GetExchangeInfo(ctx)
	if err != nil {
		log.Fatalf("exchange info: %v", err)
	}
	const symbol = "BTC/USDT-P"
	var contract *hibachi.FutureContract
	for i := range info.FutureContracts {
		if info.FutureContracts[i].Symbol == symbol {
			contract = &info.FutureContracts[i]
			break
		}
	}
	if contract == nil {
		log.Fatalf("contract %s not found", symbol)
	}

	// Connect the trade WebSocket.
	client := ws.NewTradeClient(ws.TradeClientOptions{
		APIKey:    apiKey,
		AccountID: accountID,
		Signer:    signer,
	})
	if err := client.Connect(ctx); err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect()

	// Enable cancel-on-disconnect before placing orders. If the WebSocket
	// drops, the exchange will cancel all open orders for this session.
	if _, err := client.EnableCancelOnDisconnect(ctx); err != nil {
		log.Fatalf("enable CoD: %v", err)
	}

	// Build + sign a limit buy well below market so it rests on the book.
	nonce := time.Now().UnixNano() / 1000
	quantity := decimal.RequireFromString("0.001")
	price := decimal.RequireFromString("10000")
	maxFees := decimal.RequireFromString("0.001")

	payload := hibachi.CreateOrderPayload(
		nonce,
		contract.ID,
		quantity,
		hibachi.SideBid,
		&price,
		maxFees,
		contract.UnderlyingDecimals,
		contract.SettlementDecimals,
	)
	signature, err := signer.Sign(payload)
	if err != nil {
		log.Fatalf("sign: %v", err)
	}

	priceStr := hibachi.FullPrecisionString(price)
	resp, err := client.PlaceOrder(ctx, hibachi.OrderPlaceParams{
		AccountID:      accountID,
		Nonce:          nonce,
		Symbol:         symbol,
		OrderType:      "LIMIT",
		Quantity:       hibachi.FullPrecisionString(quantity),
		Side:           string(hibachi.SideBid),
		Price:          &priceStr,
		MaxFeesPercent: hibachi.FullPrecisionString(maxFees),
		Signature:      signature,
	})
	if err != nil {
		var wsErr *hibachi.WSConnectionError
		if errors.As(err, &wsErr) {
			log.Fatalf("order failed (connection): %v", err)
		}
		log.Fatalf("order.place: %v", err)
	}
	fmt.Printf("order placed: status=%d result=%s\n", resp.Status, string(resp.Result))

	// Cancel every open order for the account.
	if _, err := client.CancelAllOrders(ctx); err != nil {
		log.Fatalf("orders.cancel: %v", err)
	}
	fmt.Println("all orders cancelled")
}
