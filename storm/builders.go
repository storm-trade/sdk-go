package storm

import (
	"context"
	"github.com/storm-trade/sdk-go/sequencer"
)

func (c *Client) GetBuilders(ctx context.Context) ([]sequencer.Builder, error) {
	return c.seq.GetBuilders(ctx)
}

func (c *Client) GetIntentStatus(ctx context.Context, hash string) (*sequencer.IntentStatus, error) {
	return c.seq.GetIntentStatus(ctx, hash)
}
