# hibachi-go

Go SDK for the [Hibachi](https://hibachi.xyz) decentralized perpetual futures exchange.

- Public REST endpoints for market data (no auth)
- Authenticated REST endpoints for trading, accounts, and capital flows
- WebSocket clients for market data, account streams, and trade execution
- Auto-reconnect on all three WebSocket clients
- Typed error hierarchy for precise handling

Requires Go 1.25 or newer.

## Install

```
go get github.com/doedja/hibachi-go
```

## Public market data (no auth)

```go
package main

import (
    "context"
    "fmt"
    "log"

    hibachi "github.com/doedja/hibachi-go"
)

func main() {
    ctx := context.Background()
    client, err := hibachi.NewClient()
    if err != nil {
        log.Fatal(err)
    }

    info, err := client.GetExchangeInfo(ctx)
    if err != nil {
        log.Fatal(err)
    }
    for _, c := range info.FutureContracts {
        fmt.Printf("%-12s id=%d tick=%s step=%s\n", c.Symbol, c.ID, c.TickSize, c.StepSize)
    }

    p, err := client.GetPrices(ctx, "BTC/USDT-P")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("BTC mark=%s bid=%s ask=%s\n", p.MarkPrice, p.BidPrice, p.AskPrice)
}
```

Available REST methods on `Client`:

- Market: `GetExchangeInfo`, `GetInventory`, `GetPrices`, `GetStats`, `GetTrades`, `GetKlines`, `GetOpenInterest`, `GetOrderbook`
- Account: `GetAccountInfo`, `GetAccountTrades`, `GetSettlementsHistory`, `GetPendingOrders`, `GetOrderDetails`
- Trading: `PlaceMarketOrder`, `PlaceLimitOrder`, `UpdateOrder`, `CancelOrder`, `CancelAllOrders`, `BatchOrders`
- Capital: `GetCapitalBalance`, `GetCapitalHistory`, `GetDepositInfo`, `Withdraw`, `Transfer`

## Authenticated client

Authenticated endpoints require an API key, account ID, and ECDSA private key.

```go
client, err := hibachi.NewClient(
    hibachi.WithAPIKey(os.Getenv("HIBACHI_API_KEY")),
    hibachi.WithAccountID(accountID),
    hibachi.WithPrivateKey(os.Getenv("HIBACHI_PRIVATE_KEY")),
)
```

Place a post-only limit buy, fetching the contract on the fly:

```go
import "github.com/shopspring/decimal"

result, err := client.PlaceLimitOrder(ctx,
    "BTC/USDT-P",
    hibachi.SideBid,
    decimal.RequireFromString("0.001"),
    decimal.RequireFromString("50000"),
    decimal.RequireFromString("0.001"),
    hibachi.WithOrderFlags(hibachi.OrderFlagsPostOnly),
)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("placed: orderId=%s nonce=%s\n", result.OrderID, result.Nonce)
```

Batch orders in a single request:

```go
price := decimal.RequireFromString("50000")
orderID := int64(123)

resp, err := client.BatchOrders(ctx, []hibachi.BatchOrderAction{
    hibachi.CreateOrder{
        Symbol:         "BTC/USDT-P",
        Side:           hibachi.SideBid,
        Quantity:       decimal.RequireFromString("0.001"),
        Price:          &price,
        MaxFeesPercent: decimal.RequireFromString("0.001"),
    },
    hibachi.CancelOrder{OrderID: &orderID},
})
```

## WebSocket: market data

```go
client := ws.NewMarketClient(ws.MarketClientOptions{})
if err := client.Connect(ctx); err != nil {
    log.Fatal(err)
}
defer client.Disconnect()

client.On("mark_price", func(data json.RawMessage) {
    fmt.Printf("mark price: %s\n", data)
})

err := client.Subscribe(ctx,
    hibachi.WSSubscription{Topic: hibachi.WSTopicMarkPrice, Symbol: "BTC/USDT-P"},
    hibachi.WSSubscription{Topic: hibachi.WSTopicTrades, Symbol: "BTC/USDT-P"},
)
```

Handlers receive the raw message payload as `json.RawMessage`. Unmarshal it
into whatever shape the topic produces.

## WebSocket: account stream

```go
client := ws.NewAccountClient(ws.AccountClientOptions{
    APIKey:    apiKey,
    AccountID: accountID,
})
if err := client.Connect(ctx); err != nil {
    log.Fatal(err)
}
defer client.Disconnect()

snapshot, err := client.StreamStart(ctx)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("balance: %s positions: %d\n",
    snapshot.AccountSnapshot.Balance, len(snapshot.AccountSnapshot.Positions))

client.On("balance", func(data json.RawMessage) { /* ... */ })
client.On("position", func(data json.RawMessage) { /* ... */ })
client.On("order", func(data json.RawMessage) { /* ... */ })

// ListenLoop blocks, auto-pings every 10s, auto-reconnects and re-StreamStarts.
if err := client.ListenLoop(ctx); err != nil && ctx.Err() == nil {
    log.Fatal(err)
}
```

## WebSocket: trading

The trade client does **not** auto-sign. You build the binary order payload,
sign it, then pass the signature into `PlaceOrder`. This keeps the SDK
agnostic to where the signer lives (local key, HSM, remote signer).

```go
signer, _ := hibachi.NewSigner(privateKey)

rest, _ := hibachi.NewClient()
info, _ := rest.GetExchangeInfo(ctx)
var contract *hibachi.FutureContract
for i := range info.FutureContracts {
    if info.FutureContracts[i].Symbol == "BTC/USDT-P" {
        contract = &info.FutureContracts[i]
        break
    }
}

trade := ws.NewTradeClient(ws.TradeClientOptions{
    APIKey:    apiKey,
    AccountID: accountID,
    Signer:    signer,
})
if err := trade.Connect(ctx); err != nil {
    log.Fatal(err)
}
defer trade.Disconnect()

// Recommended: if the WS drops, let the exchange cancel all orders for us.
trade.EnableCancelOnDisconnect(ctx)

nonce := time.Now().UnixNano() / 1000
qty := decimal.RequireFromString("0.001")
price := decimal.RequireFromString("50000")
fees := decimal.RequireFromString("0.001")

payload := hibachi.CreateOrderPayload(
    nonce, contract.ID, qty, hibachi.SideBid, &price, fees,
    contract.UnderlyingDecimals, contract.SettlementDecimals,
)
signature, _ := signer.Sign(payload)

priceStr := hibachi.FullPrecisionString(price)
resp, err := trade.PlaceOrder(ctx, hibachi.OrderPlaceParams{
    AccountID:      accountID,
    Nonce:          nonce,
    Symbol:         "BTC/USDT-P",
    OrderType:      "LIMIT",
    Side:           string(hibachi.SideBid),
    Quantity:       hibachi.FullPrecisionString(qty),
    Price:          &priceStr,
    MaxFeesPercent: hibachi.FullPrecisionString(fees),
    Signature:      signature,
})
```

### Auto-reconnect

All three WebSocket clients reconnect automatically.

- **Market**: reconnects in the read pump; re-subscribes to tracked topics.
  `Done()` returns only if the reconnect budget is exhausted.
- **Account**: reconnects in `ListenLoop`; re-issues `StreamStart` to get a
  fresh snapshot + listen key, restarts the ping loop.
- **Trade**: reconnects lazily on the next call after a failure. The call
  that hits the dead connection returns `*hibachi.WSConnectionError` so the
  caller's operation fails cleanly — nothing is retried automatically, which
  avoids duplicate orders.

Register `OnDisconnect(func(error))` and `OnReconnect(...)` to observe the
lifecycle. Configure budget via `RetryOptions` (set `MaxRetries: -1` for
infinite).

## Error handling

The SDK exposes a rooted error hierarchy. Every concrete error unwraps to
`*HibachiError`, so `errors.As` works at any level.

```
HibachiError
├── ExchangeError
│   ├── APIError
│   ├── MaintenanceError
│   └── BadHTTPStatusError → {BadRequest,Unauthorized,Forbidden,NotFound,RateLimited,…}Error
├── TransportError
│   ├── ConnectionError
│   ├── TimeoutError
│   ├── WSConnectionError
│   ├── WSMessageError
│   ├── SerializationError
│   └── DeserializationError
└── ValidationError
    └── MissingCredentialsError
```

```go
var rateLimited *hibachi.RateLimitedError
if errors.As(err, &rateLimited) {
    backoff()
}

var wsErr *hibachi.WSConnectionError
if errors.As(err, &wsErr) {
    // trade client: operation failed, reconnect will be attempted lazily
}
```

## Signing

All private endpoints and order operations are signed. Pass the hex private
key to `WithPrivateKey(...)` or build the signer directly with
`NewSigner(...)`. A key that is 64 hex characters (with or without `0x`) is
treated as an ECDSA (secp256k1) key. Otherwise it is treated as raw HMAC-SHA256
input.

The order payload layout for reference:

```
 0..8   nonce          big-endian int64
 8..12  contractId     big-endian int32
12..20  quantity       big-endian int64, scaled by 10^underlyingDecimals
20..24  side           0 = ASK / SELL, 1 = BID / BUY
[24..32 price          big-endian int64, price * 2^32 * 10^(settlementDecimals - underlyingDecimals) — LIMIT only]
N..N+8  maxFeesPercent big-endian int64, scaled by 10^8
```

Market orders omit the price block (32-byte payload); limit orders include
it (40 bytes). `CreateOrderPayload` and `PriceToBytes` implement this.

## Examples

Runnable examples live in `examples/`:

- `examples/rest` — public REST calls, authenticated trading is commented out
- `examples/ws_market` — subscribe to `mark_price` and `trades`
- `examples/ws_account` — stream balance, position, and order events
- `examples/ws_trade` — place + cancel a limit order end-to-end

Run the WebSocket examples with credentials in env vars:

```
HIBACHI_API_KEY=... HIBACHI_ACCOUNT_ID=... HIBACHI_PRIVATE_KEY=0x... \
    go run ./examples/ws_trade
```

## License

MIT. See [LICENSE](LICENSE).
