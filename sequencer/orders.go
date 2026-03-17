package sequencer

import (
	"context"
	"net/http"
)

func (c *Client) PlaceOrder(ctx context.Context, req PlaceOrderRequest) (*PlaceOrderResponse, error) {
	var result PlaceOrderResponse
	if err := c.transport.Do(ctx, http.MethodPost, "/order/place", req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) CancelOrder(ctx context.Context, req CancelOrderRequest) error {
	return c.transport.Do(ctx, http.MethodPost, "/order/cancel", req, nil)
}
