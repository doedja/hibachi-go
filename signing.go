package hibachi

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/shopspring/decimal"
)

// Signer is the interface for signing payloads.
type Signer interface {
	Sign(payload []byte) (string, error)
}

type ecdsaSigner struct {
	privateKeyBytes []byte
}

func (s *ecdsaSigner) Sign(payload []byte) (string, error) {
	hash := sha256.Sum256(payload)
	key, err := crypto.ToECDSA(s.privateKeyBytes)
	if err != nil {
		return "", fmt.Errorf("failed to parse ECDSA private key: %w", err)
	}
	sig, err := crypto.Sign(hash[:], key)
	if err != nil {
		return "", fmt.Errorf("failed to sign: %w", err)
	}
	// sig is 65 bytes: R(32) + S(32) + V(1), already in the right format
	return hex.EncodeToString(sig), nil
}

type hmacSigner struct {
	key []byte
}

func (s *hmacSigner) Sign(payload []byte) (string, error) {
	mac := hmac.New(sha256.New, s.key)
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil)), nil
}

// NewSigner creates a Signer. If the key starts with "0x" or is valid hex of 64 characters,
// an ECDSA signer is created; otherwise an HMAC signer.
func NewSigner(privateKey string) (Signer, error) {
	trimmed := privateKey
	isECDSA := false

	if strings.HasPrefix(trimmed, "0x") || strings.HasPrefix(trimmed, "0X") {
		trimmed = trimmed[2:]
		isECDSA = true
	}

	if !isECDSA && len(trimmed) == 64 {
		if _, err := hex.DecodeString(trimmed); err == nil {
			isECDSA = true
		}
	}

	if isECDSA {
		keyBytes, err := hex.DecodeString(trimmed)
		if err != nil {
			return nil, fmt.Errorf("invalid hex private key: %w", err)
		}
		if len(keyBytes) != 32 {
			return nil, fmt.Errorf("ECDSA private key must be 32 bytes, got %d", len(keyBytes))
		}
		return &ecdsaSigner{privateKeyBytes: keyBytes}, nil
	}

	return &hmacSigner{key: []byte(privateKey)}, nil
}

// PriceToBytes converts a price to its byte representation for signing.
// Formula: price * 2^32 * 10^(settlementDecimals - underlyingDecimals) -> 8 bytes big-endian
func PriceToBytes(price decimal.Decimal, settlementDecimals, underlyingDecimals int) []byte {
	// price * 2^32
	pow2_32 := decimal.NewFromBigInt(new(big.Int).Lsh(big.NewInt(1), 32), 0)
	result := price.Mul(pow2_32)

	// * 10^(settlementDecimals - underlyingDecimals)
	decimalDiff := settlementDecimals - underlyingDecimals
	if decimalDiff > 0 {
		result = result.Mul(decimal.NewFromInt(10).Pow(decimal.NewFromInt(int64(decimalDiff))))
	} else if decimalDiff < 0 {
		result = result.Div(decimal.NewFromInt(10).Pow(decimal.NewFromInt(int64(-decimalDiff))))
	}

	bigVal := result.BigInt()
	buf := make([]byte, 8)
	b := bigVal.Bytes()
	if len(b) > 8 {
		b = b[len(b)-8:]
	}
	copy(buf[8-len(b):], b)
	return buf
}

// CreateOrderPayload creates the binary payload for order signing.
// Layout: nonce(8) + contractID(4) + quantity(8) + side(4) + [price(8)] + maxFeesPercent(8)
func CreateOrderPayload(nonce int64, contractID int, quantity decimal.Decimal, side Side, price *decimal.Decimal, maxFeesPercent decimal.Decimal, underlyingDecimals, settlementDecimals int) []byte {
	var buf []byte

	// nonce: 8 bytes big-endian
	nonceBuf := make([]byte, 8)
	binary.BigEndian.PutUint64(nonceBuf, uint64(nonce))
	buf = append(buf, nonceBuf...)

	// contractID: 4 bytes big-endian
	contractBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(contractBuf, uint32(contractID))
	buf = append(buf, contractBuf...)

	// quantity: scaled by 10^underlyingDecimals, 8 bytes big-endian
	scaledQty := quantity.Mul(decimal.NewFromInt(10).Pow(decimal.NewFromInt(int64(underlyingDecimals))))
	qtyBigInt := scaledQty.BigInt()
	qtyBuf := make([]byte, 8)
	qtyBytes := qtyBigInt.Bytes()
	if len(qtyBytes) > 8 {
		qtyBytes = qtyBytes[len(qtyBytes)-8:]
	}
	copy(qtyBuf[8-len(qtyBytes):], qtyBytes)
	buf = append(buf, qtyBuf...)

	// side: 4 bytes big-endian, 0=ASK 1=BID
	sideBuf := make([]byte, 4)
	if side == SideBid || side == SideBuy {
		binary.BigEndian.PutUint32(sideBuf, 1)
	}
	buf = append(buf, sideBuf...)

	// price: 0 or 8 bytes (only if price is provided)
	if price != nil {
		buf = append(buf, PriceToBytes(*price, settlementDecimals, underlyingDecimals)...)
	}

	// maxFeesPercent: scaled by 10^8, 8 bytes big-endian
	scaledFees := maxFeesPercent.Mul(decimal.NewFromInt(10).Pow(decimal.NewFromInt(8)))
	feesBigInt := scaledFees.BigInt()
	feesBuf := make([]byte, 8)
	feesBytes := feesBigInt.Bytes()
	if len(feesBytes) > 8 {
		feesBytes = feesBytes[len(feesBytes)-8:]
	}
	copy(feesBuf[8-len(feesBytes):], feesBytes)
	buf = append(buf, feesBuf...)

	return buf
}
