package tlb

import "github.com/xssnick/tonutils-go/tlb"

type StopOrder struct {
	_       tlb.Magic     `tlb:"$0000"`
	Payload StopOrderData `tlb:"." json:"payload"`
}

func (s StopOrder) AsStopOrder() *StopOrderData {
	return &s.Payload
}

func (s StopOrder) AsLimitOrder() *LimitOrderData {
	return nil
}

func (s StopOrder) AsMarginOrder() *MarginOrderData {
	return nil
}
