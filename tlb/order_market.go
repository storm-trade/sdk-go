package tlb

import (
	"github.com/xssnick/tonutils-go/tlb"
)

type MarketOrder struct {
	_       tlb.Magic      `tlb:"$0011"`
	Payload LimitOrderData `tlb:"." json:"payload"`
}

func (s MarketOrder) AsStopOrder() *StopOrderData {
	return nil
}

func (s MarketOrder) AsLimitOrder() *LimitOrderData {
	return &s.Payload
}

func (s MarketOrder) AsMarginOrder() *MarginOrderData {
	return nil
}

func IsMarketOrder(order Order) bool {
	if order.GetType() == MarketOrderType {
		return true
	}
	if order.GetType() != TakeOrderType {
		return false
	}

	return IsNil(order.Value.AsStopOrder().TriggerPrice)
}

func IsNil(coins *tlb.Coins) bool {
	return coins.Nano().Uint64() == 0
}
