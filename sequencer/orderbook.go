package sequencer

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

func orderbookPath(asset, endpoint string) string {
	return fmt.Sprintf("/orderbook/%s/%s", url.PathEscape(asset), endpoint)
}

func (c *Client) GetOrderbookDepth(ctx context.Context, asset string, levels int) (*OrderbookDepth, error) {
	var result OrderbookDepth
	params := url.Values{"levels": {strconv.Itoa(levels)}}
	if err := c.transport.Do(ctx, http.MethodGet, orderbookPath(asset, "depth")+"?"+params.Encode(), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) GetBids(ctx context.Context, asset string) ([]OrderbookOrder, error) {
	var result []OrderbookOrder
	if err := c.transport.Do(ctx, http.MethodGet, orderbookPath(asset, "bids"), nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) GetAsks(ctx context.Context, asset string) ([]OrderbookOrder, error) {
	var result []OrderbookOrder
	if err := c.transport.Do(ctx, http.MethodGet, orderbookPath(asset, "asks"), nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}
