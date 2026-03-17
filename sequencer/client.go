package sequencer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	TestnetURL = "https://api.stage.stormtrade.dev/instant-trading"
	MainnetURL = "https://api5.storm.tg/instant-trading"
)

type transport interface {
	Do(ctx context.Context, method, path string, body any, result any) error
}

type httpTransport struct {
	baseURL string
	client  *http.Client
}

func newHTTPTransport(baseURL string, client *http.Client) *httpTransport {
	return &httpTransport{baseURL: baseURL, client: client}
}

func (t *httpTransport) Do(ctx context.Context, method, path string, body any, result any) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, t.baseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		apiErr := &APIError{StatusCode: resp.StatusCode}
		if err := json.Unmarshal(respBody, apiErr); err != nil {
			apiErr.Msg = string(respBody)
		}
		return apiErr
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("unmarshal response: %w", err)
		}
	}

	return nil
}

type Option func(*options)

type options struct {
	httpClient *http.Client
	timeout    time.Duration
}

func WithHTTPClient(c *http.Client) Option {
	return func(o *options) {
		o.httpClient = c
	}
}

func WithTimeout(d time.Duration) Option {
	return func(o *options) {
		o.timeout = d
	}
}

type Client struct {
	transport transport
}

func NewClient(baseURL string, opts ...Option) *Client {
	o := &options{
		timeout: 30 * time.Second,
	}

	for _, opt := range opts {
		opt(o)
	}

	httpClient := o.httpClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: o.timeout}
	}

	return &Client{
		transport: newHTTPTransport(baseURL, httpClient),
	}
}
