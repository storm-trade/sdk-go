package sequencer

import (
	"context"
	"net/http"
	"net/url"
)

func (c *Client) GetStatus(ctx context.Context) (*Status, error) {
	var result Status
	if err := c.transport.Do(ctx, http.MethodGet, "/status", nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) GetIntent(ctx context.Context, hash string) (*SettledAction, error) {
	var result SettledAction
	if err := c.transport.Do(ctx, http.MethodGet, "/intent/"+url.PathEscape(hash), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
