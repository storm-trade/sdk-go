package sequencer

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

type BundlesFilter struct {
	Limit        int
	Offset       int
	IncludeTrace bool
}

func (c *Client) GetBundles(ctx context.Context) ([]Bundle, error) {
	var result []Bundle
	if err := c.transport.Do(ctx, http.MethodGet, "/bundles/pending", nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) GetAccountBundles(ctx context.Context, address string, filter BundlesFilter) ([]Bundle, error) {
	if filter.Limit > 50 {
		return nil, fmt.Errorf("bundles limit %d is above the sequencer maximum of 50", filter.Limit)
	}
	params := url.Values{"order": {"desc"}}
	if filter.Limit > 0 {
		params.Set("limit", strconv.Itoa(filter.Limit))
	}
	if filter.Offset > 0 {
		params.Set("offset", strconv.Itoa(filter.Offset))
	}
	if filter.IncludeTrace {
		params.Set("include_trace", "true")
	}
	var result []Bundle
	if err := c.transport.Do(ctx, http.MethodGet, smartAccountPath(address, "bundles")+"?"+params.Encode(), nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) GetAccountBundle(ctx context.Context, address string, executorQueryID uint64) (*Bundle, error) {
	var result Bundle
	if err := c.transport.Do(ctx, http.MethodGet, smartAccountPath(address, "bundles/"+strconv.FormatUint(executorQueryID, 10)), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
