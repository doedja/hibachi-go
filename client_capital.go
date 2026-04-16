package hibachi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// GetCapitalBalance retrieves the capital balance for the authenticated account.
func (c *Client) GetCapitalBalance(ctx context.Context) (*CapitalBalance, error) {
	if err := c.requireAuth(); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/capital/balance?accountId=%d", c.accountID)
	data, err := c.transport.SendAuthorizedRequest(ctx, c.apiURL, http.MethodGet, path, nil, c.apiKey)
	if err != nil {
		return nil, err
	}

	var resp CapitalBalance
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("unmarshaling capital balance: %w", err)
	}
	return &resp, nil
}

// GetCapitalHistory retrieves the capital transaction history for the authenticated account.
func (c *Client) GetCapitalHistory(ctx context.Context) (*CapitalHistory, error) {
	if err := c.requireAuth(); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/capital/history?accountId=%d", c.accountID)
	data, err := c.transport.SendAuthorizedRequest(ctx, c.apiURL, http.MethodGet, path, nil, c.apiKey)
	if err != nil {
		return nil, err
	}

	var resp CapitalHistory
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("unmarshaling capital history: %w", err)
	}
	return &resp, nil
}

// GetDepositInfo retrieves deposit address info for the authenticated account.
func (c *Client) GetDepositInfo(ctx context.Context, publicKey string) (*DepositInfo, error) {
	if err := c.requireAuth(); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/capital/deposit-info?accountId=%d&publicKey=%s", c.accountID, publicKey)
	data, err := c.transport.SendAuthorizedRequest(ctx, c.apiURL, http.MethodGet, path, nil, c.apiKey)
	if err != nil {
		return nil, err
	}

	var resp DepositInfo
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("unmarshaling deposit info: %w", err)
	}
	return &resp, nil
}

// Withdraw initiates a withdrawal from the authenticated account.
func (c *Client) Withdraw(ctx context.Context, coin, amount, address string) (*WithdrawResponse, error) {
	if err := c.requireAuth(); err != nil {
		return nil, err
	}

	signature, err := c.signer.Sign([]byte(amount))
	if err != nil {
		return nil, fmt.Errorf("signing withdrawal: %w", err)
	}

	body := map[string]interface{}{
		"accountId": c.accountID,
		"coin":      coin,
		"amount":    amount,
		"address":   address,
		"signature": signature,
	}

	data, err := c.transport.SendAuthorizedRequest(ctx, c.apiURL, http.MethodPost, "/capital/withdraw", body, c.apiKey)
	if err != nil {
		return nil, err
	}

	var resp WithdrawResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("unmarshaling withdraw response: %w", err)
	}
	return &resp, nil
}

// Transfer initiates a transfer between accounts.
func (c *Client) Transfer(ctx context.Context, toAccountID int, amount, coin string) (*TransferResponse, error) {
	if err := c.requireAuth(); err != nil {
		return nil, err
	}

	signature, err := c.signer.Sign([]byte(amount))
	if err != nil {
		return nil, fmt.Errorf("signing transfer: %w", err)
	}

	body := map[string]interface{}{
		"fromAccountId": c.accountID,
		"toAccountId":   toAccountID,
		"amount":        amount,
		"coin":          coin,
		"signature":     signature,
	}

	data, err := c.transport.SendAuthorizedRequest(ctx, c.apiURL, http.MethodPost, "/capital/transfer", body, c.apiKey)
	if err != nil {
		return nil, err
	}

	var resp TransferResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("unmarshaling transfer response: %w", err)
	}
	return &resp, nil
}
