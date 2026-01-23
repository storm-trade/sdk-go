package tlb

import "github.com/xssnick/tonutils-go/tlb"

type LimitOrder struct {
	_       tlb.Magic      `tlb:"$0010"`
	Payload LimitOrderData `tlb:"." json:"payload"`
}

func (s LimitOrder) AsStopOrder() *StopOrderData {
	return nil
}

func (s LimitOrder) AsLimitOrder() *LimitOrderData {
	return &s.Payload
}

func (s LimitOrder) AsMarginOrder() *MarginOrderData {
	return nil
}
