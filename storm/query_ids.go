package storm

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/storm-trade/sdk-go/contracts/smartaccount"
	"github.com/storm-trade/sdk-go/sequencer"
)

type queryLease struct {
	next uint64
	hash string
}

func (c *Client) reserveQueryIDs(ctx context.Context, o clientOptions) (uint64, error) {
	var next uint64
	if o.queryID != nil {
		next = *o.queryID
	} else {
		var err error
		next, err = c.seq.GetNextQueryID(ctx, o.smartAccount.String())
		if err != nil {
			return 0, err
		}
		if l := c.queryLeases[o.smartAccount.StringRaw()]; l.next > next {
			st, err := c.seq.GetIntentStatus(ctx, l.hash)
			var apiErr *sequencer.APIError
			if err != nil && !(errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound) {
				return 0, err
			}
			if err == nil && st.Status != "failed" {
				next = l.next
			}
		}
	}
	count := uint64(1)
	if o.stopLoss != nil {
		count++
	}
	if o.takeProfit != nil {
		count++
	}
	maxID := smartaccount.MaxUserQueryID.Seqno()
	if next > maxID || count-1 > maxID-next {
		return 0, fmt.Errorf("user query ID range exhausted")
	}
	return next, nil
}
