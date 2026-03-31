package matcher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	TestnetURL = "https://api.stage.stormtrade.dev"
	MainnetURL = "https://api5.storm.tg"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{baseURL: baseURL, httpClient: &http.Client{}}
}

type BroadcastRequest struct {
	Tx     string `json:"tx"`
	Format string `json:"format"`
}

type BroadcastResponse struct {
	Hash       string `json:"ext_msg_hash"`
	SignOK     bool   `json:"sign_ok"`
	ReceivedAt string `json:"received_at"`
}

func (c *Client) Broadcast(ctx context.Context, hexBOC string) (*BroadcastResponse, error) {
	reqBody := BroadcastRequest{Tx: hexBOC, Format: "hex"}
	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/matcher/tx/broadcast", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("broadcast: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read broadcast response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("matcher API returned %d: %s", resp.StatusCode, string(body[:min(len(body), 200)]))
	}

	var result BroadcastResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse broadcast response: %w", err)
	}

	return &result, nil
}
