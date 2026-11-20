package sequencer

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	tonaddress "github.com/xssnick/tonutils-go/address"
)

func buildEventsQuery(filter EventsFilter) string {
	params := url.Values{}
	if filter.Limit > 0 {
		params.Set("limit", strconv.Itoa(filter.Limit))
	}
	if filter.Offset > 0 {
		params.Set("offset", strconv.Itoa(filter.Offset))
	}
	if filter.EventType != "" {
		params.Set("event_type", filter.EventType)
	}
	if filter.Order != "" {
		params.Set("order", strings.ToUpper(filter.Order))
	}
	if len(params) == 0 {
		return ""
	}
	return "?" + params.Encode()
}

func (c *Client) GetEvents(ctx context.Context, filter EventsFilter) ([]Event, error) {
	filter.EventType = ""
	var result []Event
	if err := c.transport.Do(ctx, http.MethodGet, "/events"+buildEventsQuery(filter), nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) GetAccountEvents(ctx context.Context, address string, filter EventsFilter) ([]Event, error) {
	sa, err := tonaddress.ParseRawAddr(address)
	if err != nil {
		if sa, err = tonaddress.ParseAddr(address); err != nil {
			return nil, err
		}
	}
	var result []Event
	path := smartAccountPath(sa.StringRaw(), "events") + buildEventsQuery(filter)
	if err := c.transport.Do(ctx, http.MethodGet, path, nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}
