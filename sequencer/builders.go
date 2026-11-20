package sequencer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

type Builder struct {
	Address   string `json:"builder"`
	Rebate    uint64 `json:"rebate"`
	FeeEarned string `json:"fee_earned"`
	Active    bool   `json:"active"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

type IntentStatus struct {
	Status            string          `json:"status"`
	Trace             json.RawMessage `json:"trace,omitempty"`
	Error             string          `json:"error,omitempty"`
	OrderHash         string          `json:"order_hash,omitempty"`
	Code              string          `json:"code,omitempty"`
	ContractInterface string          `json:"contract_interface,omitempty"`
	ContractExitCode  int             `json:"contract_exit_code,omitempty"`
	VMExitCode        int             `json:"vm_exit_code,omitempty"`
	Timestamp         int64           `json:"ts"`
}

func (c *Client) GetBuilders(ctx context.Context) ([]Builder, error) {
	var result []Builder
	if err := c.transport.Do(ctx, http.MethodGet, "/builders", nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetIntentStatus reports sequencer processing, not on-chain finality.
func (c *Client) GetIntentStatus(ctx context.Context, hash string) (*IntentStatus, error) {
	var result IntentStatus
	if err := c.transport.Do(ctx, http.MethodGet, "/intent/"+url.PathEscape(hash)+"/status", nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
