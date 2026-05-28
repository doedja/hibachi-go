package hibachi

import (
	"encoding/json"
	"testing"
)

// The transport unwraps the {"data":[...]} envelope, so GetFundingRates
// unmarshals a bare array. Lock that shape.
func TestFundingRateParse(t *testing.T) {
	body := `[
		{"contractId":62,"fundingTimestamp":1779404400.0,"fundingRate":"0.000063","indexPrice":"1.161840000"},
		{"contractId":62,"fundingTimestamp":1779408000.0,"fundingRate":"-0.000129","indexPrice":"1.161515000"}
	]`
	var rates []FundingRate
	if err := json.Unmarshal([]byte(body), &rates); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(rates) != 2 {
		t.Fatalf("len = %d, want 2", len(rates))
	}
	if rates[0].ContractID != 62 || rates[0].FundingRate != "0.000063" {
		t.Errorf("rate[0] = %+v", rates[0])
	}
	if rates[0].FundingTimestamp != 1779404400.0 {
		t.Errorf("timestamp = %v", rates[0].FundingTimestamp)
	}
}

func TestKlineRangeOption(t *testing.T) {
	var p klineParams
	WithKlineRange(1000, 2000)(&p)
	if p.fromMs != 1000 || p.toMs != 2000 {
		t.Errorf("klineParams = %+v, want {1000 2000}", p)
	}
}

func TestFundingRateOptions(t *testing.T) {
	var p fundingRateParams
	WithFundingRange(100, 200)(&p)
	WithFundingLimit(50)(&p)
	if p.startTime != 100 || p.endTime != 200 || p.limit != 50 {
		t.Errorf("fundingRateParams = %+v, want {100 200 50}", p)
	}
}

// AccountInfo must now parse the positions array the REST endpoint returns.
func TestAccountInfoParsesPositions(t *testing.T) {
	body := `{
		"balance":"3.47",
		"totalPositionNotional":"1.50",
		"positions":[{"symbol":"EUR/USDT-P","direction":"Long","quantity":"1.28792","openPrice":"1.1646"}]
	}`
	var info AccountInfo
	if err := json.Unmarshal([]byte(body), &info); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(info.Positions) != 1 || info.Positions[0].Symbol != "EUR/USDT-P" {
		t.Fatalf("positions = %+v", info.Positions)
	}
	if info.Positions[0].Direction != "Long" {
		t.Errorf("direction = %q", info.Positions[0].Direction)
	}
}
