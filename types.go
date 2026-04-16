package hibachi

// Interval represents kline/candlestick intervals.
type Interval string

const (
	IntervalOneMinute      Interval = "1min"
	IntervalFiveMinutes    Interval = "5min"
	IntervalFifteenMinutes Interval = "15min"
	IntervalOneHour        Interval = "1h"
	IntervalFourHours      Interval = "4h"
	IntervalOneDay         Interval = "1d"
	IntervalOneWeek        Interval = "1w"
)

// Side represents the order side.
type Side string

const (
	SideBid  Side = "BID"
	SideAsk  Side = "ASK"
	SideSell Side = "SELL"
	SideBuy  Side = "BUY"
)

// OrderType represents the type of order.
type OrderType string

const (
	OrderTypeLimit  OrderType = "LIMIT"
	OrderTypeMarket OrderType = "MARKET"
)

// OrderStatus represents the status of an order.
type OrderStatus string

const (
	OrderStatusPending         OrderStatus = "PENDING"
	OrderStatusChildPending    OrderStatus = "CHILD_PENDING"
	OrderStatusFilled          OrderStatus = "FILLED"
	OrderStatusCancelled       OrderStatus = "CANCELLED"
	OrderStatusRejected        OrderStatus = "REJECTED"
	OrderStatusScheduledTWAP   OrderStatus = "SCHEDULED_TWAP"
	OrderStatusPlaced          OrderStatus = "PLACED"
	OrderStatusPartiallyFilled OrderStatus = "PARTIALLY_FILLED"
)

// OrderFlags represents order execution flags.
type OrderFlags string

const (
	OrderFlagsPostOnly   OrderFlags = "POST_ONLY"
	OrderFlagsIOC        OrderFlags = "IOC"
	OrderFlagsReduceOnly OrderFlags = "REDUCE_ONLY"
)

// TriggerDirection represents the trigger direction for conditional orders.
type TriggerDirection string

const (
	TriggerDirectionHigh TriggerDirection = "HIGH"
	TriggerDirectionLow  TriggerDirection = "LOW"
)

// TakerSide represents the taker side of a trade.
type TakerSide string

const (
	TakerSideBuy  TakerSide = "Buy"
	TakerSideSell TakerSide = "Sell"
)

// WSSubscriptionTopic represents WebSocket subscription topics.
type WSSubscriptionTopic string

const (
	WSTopicMarkPrice             WSSubscriptionTopic = "mark_price"
	WSTopicSpotPrice             WSSubscriptionTopic = "spot_price"
	WSTopicFundingRateEstimation WSSubscriptionTopic = "funding_rate_estimation"
	WSTopicTrades                WSSubscriptionTopic = "trades"
	WSTopicKlines                WSSubscriptionTopic = "klines"
	WSTopicOrderbook             WSSubscriptionTopic = "orderbook"
	WSTopicAskBidPrice           WSSubscriptionTopic = "ask_bid_price"
)

// TWAPQuantityMode represents the quantity mode for TWAP orders.
type TWAPQuantityMode string

const (
	TWAPQuantityModeFixed  TWAPQuantityMode = "FIXED"
	TWAPQuantityModeRandom TWAPQuantityMode = "RANDOM"
)

// ExchangeInfo holds exchange configuration from the exchange.
type ExchangeInfo struct {
	FutureContracts []FutureContract `json:"futureContracts"`
	Status          string           `json:"status"`
}

// Contracts returns the list of future contracts (convenience alias).
func (e *ExchangeInfo) Contracts() []FutureContract {
	return e.FutureContracts
}

// FutureContract represents a futures contract.
type FutureContract struct {
	ID                      int      `json:"id"`
	Symbol                  string   `json:"symbol"`
	DisplayName             string   `json:"displayName"`
	SettlementSymbol        string   `json:"settlementSymbol"`
	SettlementDecimals      int      `json:"settlementDecimals"`
	UnderlyingSymbol        string   `json:"underlyingSymbol"`
	UnderlyingDecimals      int      `json:"underlyingDecimals"`
	InitialMarginRate       string   `json:"initialMarginRate"`
	MaintenanceMarginRate   string   `json:"maintenanceMarginRate"`
	MinNotional             string   `json:"minNotional"`
	MinOrderSize            string   `json:"minOrderSize"`
	StepSize                string   `json:"stepSize"`
	TickSize                string   `json:"tickSize"`
	Status                  string   `json:"status"`
	OrderbookGranularities  []string `json:"orderbookGranularities"`
	MarketCreationTimestamp *string  `json:"marketCreationTimestamp"`
	MarketOpenTimestamp     *string  `json:"marketOpenTimestamp"`
	MarketCloseTimestamp    *string  `json:"marketCloseTimestamp"`
}

// PriceResponse represents price data from the exchange.
type PriceResponse struct {
	Symbol                string                 `json:"symbol"`
	MarkPrice             string                 `json:"markPrice"`
	SpotPrice             string                 `json:"spotPrice"`
	TradePrice            string                 `json:"tradePrice"`
	AskPrice              string                 `json:"askPrice"`
	BidPrice              string                 `json:"bidPrice"`
	FundingRateEstimation *FundingRateEstimation `json:"fundingRateEstimation"`
}

// FundingRateEstimation represents funding rate estimation data.
type FundingRateEstimation struct {
	EstimatedFundingRate string `json:"estimatedFundingRate"`
	NextFundingTimestamp int64  `json:"nextFundingTimestamp"`
}

// StatsResponse represents market statistics.
type StatsResponse struct {
	Symbol    string `json:"symbol"`
	High24h   string `json:"high24h"`
	Low24h    string `json:"low24h"`
	Volume24h string `json:"volume24h"`
}

// Trade represents a single trade.
type Trade struct {
	Price     string    `json:"price"`
	Quantity  string    `json:"quantity"`
	Timestamp int64     `json:"timestamp"`
	TakerSide TakerSide `json:"takerSide"`
}

// TradesResponse represents a list of trades.
type TradesResponse struct {
	Trades []Trade `json:"trades"`
}

// Kline represents a single kline/candlestick.
type Kline struct {
	Interval       string `json:"interval"`
	Timestamp      int64  `json:"timestamp"`
	Open           string `json:"open"`
	Close          string `json:"close"`
	High           string `json:"high"`
	Low            string `json:"low"`
	VolumeNotional string `json:"volumeNotional"`
}

// KlinesResponse represents a list of klines.
type KlinesResponse struct {
	Klines []Kline `json:"klines"`
}

// OrderBook represents the order book.
type OrderBook struct {
	Ask OrderBookSide `json:"ask"`
	Bid OrderBookSide `json:"bid"`
}

// OrderBookSide represents one side (ask or bid) of the order book.
type OrderBookSide struct {
	StartPrice string           `json:"startPrice"`
	EndPrice   string           `json:"endPrice"`
	Levels     []OrderBookLevel `json:"levels"`
}

// OrderBookLevel represents a single price level in the order book.
type OrderBookLevel struct {
	Price    string `json:"price"`
	Quantity string `json:"quantity"`
}

// OpenInterest represents open interest data.
type OpenInterest struct {
	TotalQuantity string `json:"totalQuantity"`
}

// Order represents an order on the exchange.
type Order struct {
	AccountID          int         `json:"accountId"`
	AvailableQuantity  string      `json:"availableQuantity"`
	ContractID         *int        `json:"contractId"`
	CreationTime       *int64      `json:"creationTime"`
	FinishTime         *int64      `json:"finishTime"`
	NumOrdersRemaining *int        `json:"numOrdersRemaining"`
	NumOrdersTotal     *int        `json:"numOrdersTotal"`
	OrderFlags         *OrderFlags `json:"orderFlags"`
	OrderID            int64       `json:"orderId"`
	OrderType          OrderType   `json:"orderType"`
	Price              *string     `json:"price"`
	QuantityMode       *string     `json:"quantityMode"`
	Side               Side        `json:"side"`
	Status             OrderStatus `json:"status"`
	Symbol             string      `json:"symbol"`
	TotalQuantity      *string     `json:"totalQuantity"`
	TriggerPrice       *string     `json:"triggerPrice"`
}

// CapitalBalance represents account capital balance.
type CapitalBalance struct {
	AvailableBalance string `json:"availableBalance"`
	LockedBalance    string `json:"lockedBalance"`
	TotalBalance     string `json:"totalBalance"`
}

// CapitalHistory represents capital transaction history.
type CapitalHistory struct {
	Transactions []Transaction `json:"transactions"`
}

// Transaction represents a single capital transaction.
type Transaction struct {
	Asset      string `json:"asset"`
	Amount     string `json:"amount"`
	Status     string `json:"status"`
	UpdateTime int64  `json:"updateTime"`
}

// AccountInfo represents account information.
type AccountInfo struct {
	Balance               string `json:"balance"`
	TotalPositionNotional string `json:"totalPositionNotional"`
}

// Position represents a futures position.
type Position struct {
	Symbol               string `json:"symbol"`
	Quantity             string `json:"quantity"`
	Direction            string `json:"direction"`            // "Long" or "Short"
	OpenPrice            string `json:"openPrice"`            // entry price
	EntryNotional        string `json:"entryNotional"`        // entry notional value
	NotionalValue        string `json:"notionalValue"`        // current notional value
	MarkPrice            string `json:"markPrice"`            // current mark price
	UnrealizedTradingPnl string `json:"unrealizedTradingPnl"` // trading PnL
	UnrealizedFundingPnl string `json:"unrealizedFundingPnl"` // funding PnL
	// Deprecated: use OpenPrice instead. Kept for backwards compat.
	EntryPrice    string `json:"entryPrice"`
	UnrealizedPnl string `json:"unrealizedPnl"`
	Leverage      int    `json:"leverage"`
}

// AccountTrade represents a trade executed on the account.
type AccountTrade struct {
	Symbol   string `json:"symbol"`
	OrderID  int64  `json:"orderId"`
	Price    string `json:"price"`
	Quantity string `json:"quantity"`
	Side     Side   `json:"side"`
	Time     int64  `json:"time"`
}

// Settlement represents a funding settlement.
type Settlement struct {
	Symbol        string `json:"symbol"`
	FundingRate   string `json:"fundingRate"`
	FundingAmount string `json:"fundingAmount"`
	Time          int64  `json:"time"`
}

// DepositInfo represents deposit address information.
type DepositInfo struct {
	Address string `json:"address"`
	Coin    string `json:"coin"`
}

// WithdrawResponse represents a withdrawal response.
type WithdrawResponse struct {
	TransactionID string `json:"transactionId"`
}

// TransferResponse represents a transfer response.
type TransferResponse struct {
	Success bool `json:"success"`
}
