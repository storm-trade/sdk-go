package sequencer

import (
	"context"
	"net/http"
	"strconv"
)

func (c *Client) GetBundles(ctx context.Context) ([]Bundle, error) {
	var result []Bundle
	if err := c.transport.Do(ctx, http.MethodGet, "/bundles", nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) GetAccountBundles(ctx context.Context, address string) ([]Bundle, error) {
	var result []Bundle
	if err := c.transport.Do(ctx, http.MethodGet, smartAccountPath(address, "bundles"), nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) GetAccountBundle(ctx context.Context, address string, queryID uint64) (*Bundle, error) {
	var result Bundle
	if err := c.transport.Do(ctx, http.MethodGet, smartAccountPath(address, "bundles/"+strconv.FormatUint(queryID, 10)), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
