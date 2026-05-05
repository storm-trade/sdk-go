package sequencer

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

func (c *Client) GetGaslessBalance(ctx context.Context, address string) (*GaslessBalance, error) {
	var result GaslessBalance
	path := fmt.Sprintf("/gasless/balance/%s", url.PathEscape(address))
	if err := c.transport.Do(ctx, http.MethodGet, path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) GetGaslessWithdrawals(ctx context.Context, address string) ([]GaslessWithdrawalEntry, error) {
	var result []GaslessWithdrawalEntry
	path := fmt.Sprintf("/gasless/withdrawals/%s", url.PathEscape(address))
	if err := c.transport.Do(ctx, http.MethodGet, path, nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) GaslessWithdraw(ctx context.Context, req GaslessWithdrawRequest) (*GaslessWithdrawResponse, error) {
	var result GaslessWithdrawResponse
	if err := c.transport.Do(ctx, http.MethodPost, "/gasless/withdrawal", req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
