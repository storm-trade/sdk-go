package tlb

import "github.com/xssnick/tonutils-go/tlb"

type MarginOrderData struct {
	Direction uint       `tlb:"## 1" json:"direction"`
	Amount    *tlb.Coins `tlb:"." json:"amount"`
}

type AddMarginOrder struct {
	_       tlb.Magic       `tlb:"$0100"`
	Payload MarginOrderData `tlb:"." json:"payload"`
}

type RemoveMarginOrder struct {
	_       tlb.Magic       `tlb:"$0101"`
	Payload MarginOrderData `tlb:"." json:"payload"`
}

func (s AddMarginOrder) AsStopOrder() *StopOrderData {
	return nil
}

func (s AddMarginOrder) AsLimitOrder() *LimitOrderData {
	return nil
}

func (s AddMarginOrder) AsMarginOrder() *MarginOrderData {
	return &s.Payload
}

func (s RemoveMarginOrder) AsStopOrder() *StopOrderData {
	return nil
}

func (s RemoveMarginOrder) AsLimitOrder() *LimitOrderData {
	return nil
}

func (s RemoveMarginOrder) AsMarginOrder() *MarginOrderData {
	return &s.Payload
}
