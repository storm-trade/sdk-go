package storm

import (
	"context"
	"fmt"
	"github.com/storm-trade/sdk-go/contracts/hw"
)

// IDs are reserved before sending, including on an unknown HTTP outcome. Applications
// sharing an account across processes must allocate IDs themselves with WithQueryID.
func (c *Client) reserveQueryIDs(ctx context.Context, o clientOptions) (uint64, error) {
	c.queryMu.Lock()
	defer c.queryMu.Unlock()
	var next uint64
	if o.queryID != nil {
		next = *o.queryID
	} else {
		var err error
		next, err = c.seq.GetNextQueryID(ctx, o.smartAccount.String())
		if err != nil {
			return 0, err
		}
		if local := c.nextQueryIDs[o.smartAccount.StringRaw()]; local > next {
			next = local
		}
	}
	count := uint64(1)
	if o.stopLoss != nil {
		count++
	}
	if o.takeProfit != nil {
		count++
	}
	if next >= uint64(hw.MaxQueryId) || count > uint64(hw.MaxQueryId)-next {
		return 0, fmt.Errorf("user query ID range exhausted")
	}
	key := o.smartAccount.StringRaw()
	if next+count > c.nextQueryIDs[key] {
		c.nextQueryIDs[key] = next + count
	}
	return next, nil
}
