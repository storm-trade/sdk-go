package tlb

import "github.com/xssnick/tonutils-go/tlb"

type TakeOrder struct {
	_       tlb.Magic     `tlb:"$0001"`
	Payload StopOrderData `tlb:"." json:"payload"`
}

func (s TakeOrder) AsStopOrder() *StopOrderData {
	return &s.Payload
}

func (s TakeOrder) AsLimitOrder() *LimitOrderData {
	return nil
}

func (s TakeOrder) AsMarginOrder() *MarginOrderData {
	return nil
}
