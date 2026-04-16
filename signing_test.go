package hibachi

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"math/big"
	"strings"
	"testing"

	"github.com/shopspring/decimal"
)

func TestNewSigner_ECDSAFromHexWith0x(t *testing.T) {
	key := "0x" + strings.Repeat("ab", 32)
	s, err := NewSigner(key)
	if err != nil {
		t.Fatalf("NewSigner: %v", err)
	}
	if _, ok := s.(*ecdsaSigner); !ok {
		t.Fatalf("expected ecdsaSigner, got %T", s)
	}
}

func TestNewSigner_ECDSAFromHex64(t *testing.T) {
	key := strings.Repeat("ab", 32)
	s, err := NewSigner(key)
	if err != nil {
		t.Fatalf("NewSigner: %v", err)
	}
	if _, ok := s.(*ecdsaSigner); !ok {
		t.Fatalf("expected ecdsaSigner, got %T", s)
	}
}

func TestNewSigner_HMACFallback(t *testing.T) {
	s, err := NewSigner("not-a-hex-key")
	if err != nil {
		t.Fatalf("NewSigner: %v", err)
	}
	if _, ok := s.(*hmacSigner); !ok {
		t.Fatalf("expected hmacSigner, got %T", s)
	}
}

func TestNewSigner_InvalidHexWith0x(t *testing.T) {
	// "0x" prefix forces ECDSA branch; bad hex must error.
	_, err := NewSigner("0xZZ")
	if err == nil {
		t.Fatal("expected error for invalid hex")
	}
}

func TestECDSASigner_ProducesDeterministicLength(t *testing.T) {
	// go-ethereum's crypto.Sign returns 65-byte signatures (R|S|V).
	key := strings.Repeat("11", 32)
	s, err := NewSigner(key)
	if err != nil {
		t.Fatalf("NewSigner: %v", err)
	}
	sig, err := s.Sign([]byte("hello"))
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	raw, err := hex.DecodeString(sig)
	if err != nil {
		t.Fatalf("decode sig: %v", err)
	}
	if len(raw) != 65 {
		t.Fatalf("expected 65-byte signature, got %d", len(raw))
	}
}

func TestHMACSigner_ProducesSHA256(t *testing.T) {
	s, err := NewSigner("secret")
	if err != nil {
		t.Fatalf("NewSigner: %v", err)
	}
	sig, err := s.Sign([]byte("payload"))
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	raw, err := hex.DecodeString(sig)
	if err != nil {
		t.Fatalf("decode sig: %v", err)
	}
	if len(raw) != 32 {
		t.Fatalf("HMAC-SHA256 signature should be 32 bytes, got %d", len(raw))
	}
}

func TestCreateOrderPayload_LimitLayout(t *testing.T) {
	// Layout: nonce(8) + contractID(4) + quantity(8) + side(4) + price(8) + maxFees(8) = 40 bytes.
	nonce := int64(1234567890)
	contractID := 2
	quantity := decimal.RequireFromString("0.001")
	price := decimal.RequireFromString("50000")
	maxFees := decimal.RequireFromString("0.001")

	// BTC/USDT-P live params: underlyingDecimals=10, settlementDecimals=6.
	payload := CreateOrderPayload(nonce, contractID, quantity, SideBid, &price, maxFees, 10, 6)
	if len(payload) != 40 {
		t.Fatalf("limit payload length: got %d, want 40", len(payload))
	}

	gotNonce := binary.BigEndian.Uint64(payload[0:8])
	if int64(gotNonce) != nonce {
		t.Fatalf("nonce: got %d, want %d", gotNonce, nonce)
	}

	gotContract := binary.BigEndian.Uint32(payload[8:12])
	if int(gotContract) != contractID {
		t.Fatalf("contractID: got %d, want %d", gotContract, contractID)
	}

	// quantity = 0.001 * 10^10 = 10_000_000.
	gotQty := binary.BigEndian.Uint64(payload[12:20])
	if gotQty != 10_000_000 {
		t.Fatalf("quantity: got %d, want 10000000", gotQty)
	}

	// side BID → 1.
	gotSide := binary.BigEndian.Uint32(payload[20:24])
	if gotSide != 1 {
		t.Fatalf("side BID: got %d, want 1", gotSide)
	}

	// price bytes: check the PriceToBytes helper produces the same bytes.
	wantPrice := PriceToBytes(price, 6, 10)
	if !bytes.Equal(payload[24:32], wantPrice) {
		t.Fatalf("price bytes mismatch")
	}

	// maxFees = 0.001 * 10^8 = 100000.
	gotFees := binary.BigEndian.Uint64(payload[32:40])
	if gotFees != 100_000 {
		t.Fatalf("maxFees: got %d, want 100000", gotFees)
	}
}

func TestCreateOrderPayload_MarketHasNoPrice(t *testing.T) {
	payload := CreateOrderPayload(1, 1, decimal.NewFromInt(1), SideAsk, nil, decimal.RequireFromString("0.001"), 8, 6)
	// nonce(8) + contractID(4) + quantity(8) + side(4) + maxFees(8) = 32 bytes.
	if len(payload) != 32 {
		t.Fatalf("market payload length: got %d, want 32", len(payload))
	}
	// side ASK → 0.
	if got := binary.BigEndian.Uint32(payload[20:24]); got != 0 {
		t.Fatalf("side ASK: got %d, want 0", got)
	}
}

func TestCreateOrderPayload_BuyEqualsBid(t *testing.T) {
	a := CreateOrderPayload(1, 1, decimal.NewFromInt(1), SideBuy, nil, decimal.NewFromInt(0), 6, 6)
	b := CreateOrderPayload(1, 1, decimal.NewFromInt(1), SideBid, nil, decimal.NewFromInt(0), 6, 6)
	if !bytes.Equal(a, b) {
		t.Fatal("BUY and BID must produce identical payloads")
	}
}

func TestPriceToBytes_UsesFullFormula(t *testing.T) {
	// Formula: price * 2^32 * 10^(settlementDecimals - underlyingDecimals), big-endian 8 bytes.
	// For price=1, settlement=underlying → just 2^32.
	got := PriceToBytes(decimal.NewFromInt(1), 6, 6)
	want := make([]byte, 8)
	binary.BigEndian.PutUint64(want, uint64(1)<<32)
	if !bytes.Equal(got, want) {
		t.Fatalf("PriceToBytes(1, 6, 6): got %x, want %x", got, want)
	}

	// settlement > underlying: extra 10^(diff) scaling.
	// price=1, settlement=9, underlying=6 → 2^32 * 10^3.
	got = PriceToBytes(decimal.NewFromInt(1), 9, 6)
	exp := new(big.Int).Lsh(big.NewInt(1), 32)
	exp.Mul(exp, big.NewInt(1000))
	want = make([]byte, 8)
	expBytes := exp.Bytes()
	copy(want[8-len(expBytes):], expBytes)
	if !bytes.Equal(got, want) {
		t.Fatalf("PriceToBytes(1, 9, 6): got %x, want %x", got, want)
	}
}
