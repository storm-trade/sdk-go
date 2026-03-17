package sequencer

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

func smartAccountPath(address, endpoint string) string {
	return fmt.Sprintf("/smartaccount/%s/%s", url.PathEscape(address), endpoint)
}

func (c *Client) GetAccountState(ctx context.Context, address string) (*AccountState, error) {
	var result AccountState
	if err := c.transport.Do(ctx, http.MethodGet, smartAccountPath(address, "state"), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) GetBalance(ctx context.Context, address, tokenAddress string) (*BalanceResponse, error) {
	var result BalanceResponse
	params := url.Values{"token_address": {tokenAddress}}
	if err := c.transport.Do(ctx, http.MethodGet, smartAccountPath(address, "balance")+"?"+params.Encode(), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) GetBalances(ctx context.Context, address string) (map[string]int64, error) {
	var result map[string]int64
	if err := c.transport.Do(ctx, http.MethodGet, smartAccountPath(address, "balances"), nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) GetLockedBalance(ctx context.Context, address, tokenAddress string) (*BalanceResponse, error) {
	var result BalanceResponse
	params := url.Values{"token_address": {tokenAddress}}
	if err := c.transport.Do(ctx, http.MethodGet, smartAccountPath(address, "lockedBalance")+"?"+params.Encode(), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) GetPositions(ctx context.Context, address string) ([]PositionResponse, error) {
	var result []PositionResponse
	if err := c.transport.Do(ctx, http.MethodGet, smartAccountPath(address, "positions"), nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) GetNextQueryID(ctx context.Context, address string) (uint64, error) {
	var result struct {
		QueryID uint64 `json:"query_id"`
	}
	if err := c.transport.Do(ctx, http.MethodGet, smartAccountPath(address, "queryID"), nil, &result); err != nil {
		return 0, err
	}
	return result.QueryID, nil
}

func (c *Client) GetGasUnits(ctx context.Context, address string) (uint64, error) {
	var result BalanceResponse
	if err := c.transport.Do(ctx, http.MethodGet, smartAccountPath(address, "gasUnits"), nil, &result); err != nil {
		return 0, err
	}
	return result.Amount, nil
}
