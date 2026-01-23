package tlb

import (
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/tvm/cell"
)

func init() {
	tlb.Register(MarketOrder{})
	tlb.Register(LimitOrder{})
	tlb.Register(StopOrder{})
	tlb.Register(TakeOrder{})
	tlb.Register(AddMarginOrder{})    // const int order_type::add_margin = 4;
	tlb.Register(RemoveMarginOrder{}) // const int order_type::remove_margin = 5;
}

type OrderType int

const (
	StopOrderType OrderType = iota
	TakeOrderType
	LimitOrderType
	MarketOrderType
	AddMarginOrderType
	RemoveMarginOrderType
	UnknownOrderType = -1
)

func (t OrderType) String() string {
	switch t {
	case StopOrderType:
		return "stopLoss"
	case TakeOrderType:
		return "takeProfit"
	case LimitOrderType:
		return "limit"
	case MarketOrderType:
		return "market"
	case AddMarginOrderType:
		return "addMargin"
	case RemoveMarginOrderType:
		return "removeMargin"
	default:
		panic("unknown order type")
	}
}

type LimitOrderData struct {
	Expiration       uint32     `tlb:"## 32" json:"expiration"`
	Direction        uint       `tlb:"## 1" json:"direction"`
	Amount           *tlb.Coins `tlb:"." json:"amount"`
	Leverage         uint64     `tlb:"## 64" json:"leverage"`
	LimitPrice       *tlb.Coins `tlb:"." json:"limit_price"`
	StopPrice        *tlb.Coins `tlb:"." json:"stop_price"`
	StopTriggerPrice *tlb.Coins `tlb:"." json:"stop_trigger_price"`
	TakeTriggerPrice *tlb.Coins `tlb:"." json:"take_trigger_price"`
}

type StopOrderData struct {
	Expiration   uint32     `tlb:"## 32" json:"expiration"`
	Direction    uint       `tlb:"## 1" json:"direction"`
	Amount       *tlb.Coins `tlb:"." json:"amount"`
	TriggerPrice *tlb.Coins `tlb:"." json:"limit_price"`
}

type AnyOrder interface {
	AsStopOrder() *StopOrderData
	AsLimitOrder() *LimitOrderData
	AsMarginOrder() *MarginOrderData
}

type Order struct {
	Value AnyOrder `tlb:"[StopOrder,TakeOrder,LimitOrder,MarketOrder,AddMarginOrder,RemoveMarginOrder]" json:"order"`
}

type OrderWithIndex struct {
	Order *Order `json:"order"`
	Index int    `json:"index"`
}

func (o *Order) MarshalJSON() ([]byte, error) {
	switch ord := o.Value.(type) {
	case TakeOrder, StopOrder:
		limit := ord.AsStopOrder()

		return json.Marshal(struct {
			*StopOrderData
			Type string `json:"type"`
		}{
			StopOrderData: limit,
			Type:          o.GetType().String(),
		})
	case LimitOrder, MarketOrder:
		limit := ord.AsLimitOrder()

		return json.Marshal(struct {
			*LimitOrderData
			Type string `json:"type"`
		}{
			LimitOrderData: limit,
			Type:           o.GetType().String(),
		})
	case AddMarginOrder, RemoveMarginOrder:
		margin := ord.AsMarginOrder()

		return json.Marshal(struct {
			*MarginOrderData
			Type string `json:"type"`
		}{
			MarginOrderData: margin,
			Type:            o.GetType().String(),
		})
	default:
		panic("unexpected order type")
	}

}

func (o *Order) UnmarshalJSON(value []byte) error {
	str := struct {
		Type string `json:"type,omitempty"`
	}{}

	if err := json.Unmarshal(value, &str); err != nil {
		return err
	}

	switch str.Type {
	case "addMargin":
		var payload MarginOrderData
		if err := json.Unmarshal(value, &payload); err != nil {
			return err
		}

		o.Value = AddMarginOrder{Payload: payload}
	case "removeMargin":
		var payload MarginOrderData
		if err := json.Unmarshal(value, &payload); err != nil {
			return err
		}

		o.Value = RemoveMarginOrder{Payload: payload}
	case "market":
		var payload LimitOrderData
		if err := json.Unmarshal(value, &payload); err != nil {
			return err
		}

		o.Value = MarketOrder{Payload: payload}
	case "limit":
		var payload LimitOrderData
		if err := json.Unmarshal(value, &payload); err != nil {
			return err
		}

		o.Value = LimitOrder{Payload: payload}
	case "stopLoss":
		var payload StopOrderData

		if err := json.Unmarshal(value, &payload); err != nil {

			fmt.Println("err")
			return err
		}

		o.Value = StopOrder{Payload: payload}
	case "takeProfit":
		var payload StopOrderData

		if err := json.Unmarshal(value, &payload); err != nil {

			fmt.Println("err")
			return err
		}

		o.Value = TakeOrder{Payload: payload}
	default:
		panic("unexpected order type")
	}

	return nil
}

func (o *Order) PriceLevel() int64 {
	switch o := o.Value.(type) {
	case TakeOrder, StopOrder:
		limit := o.AsStopOrder()

		return limit.TriggerPrice.Nano().Int64()
	case LimitOrder, MarketOrder:
		limit := o.AsLimitOrder()

		return limit.LimitPrice.Nano().Int64()
	}
	panic("unexpected order type")
}

func (o *Order) DirectionNum() uint {
	switch payload := o.Value.(type) {
	case StopOrder, TakeOrder:
		return payload.AsStopOrder().Direction
	case AddMarginOrder, RemoveMarginOrder:
		return payload.AsMarginOrder().Direction
	case LimitOrder, MarketOrder:
		return payload.AsLimitOrder().Direction
	}

	panic("unexpected order type")
}

func (o *Order) GetDirection() Direction {
	switch payload := o.Value.(type) {
	case StopOrder, TakeOrder:
		return DirectionFromInt(int(payload.AsStopOrder().Direction))
	case LimitOrder, MarketOrder:
		return DirectionFromInt(int(payload.AsLimitOrder().Direction))
	case AddMarginOrder, RemoveMarginOrder:
		return DirectionFromInt(int(payload.AsMarginOrder().Direction))
	}

	panic("unexpected order type")
}

func (o *Order) GetExpiration() uint32 {
	switch o.GetType() {
	case MarketOrderType, LimitOrderType:
		return o.Value.AsLimitOrder().Expiration
	case TakeOrderType, StopOrderType:
		return o.Value.AsStopOrder().Expiration
	default:
		return 0
	}
}

func (o *Order) GetLeverage() uint64 {
	switch o.GetType() {
	case MarketOrderType, LimitOrderType:
		return o.Value.AsLimitOrder().Leverage
	default:
		return 0
	}
}

func (o *Order) GetAmount() *big.Int {
	switch o.GetType() {
	case MarketOrderType, LimitOrderType, AddMarginOrderType:
		return o.Value.AsLimitOrder().Amount.Nano()
	case StopOrderType, TakeOrderType, RemoveMarginOrderType:
		return o.Value.AsStopOrder().Amount.Nano()
	default:
		return nil
	}
}

func (o *Order) IsLong() bool {
	return o.GetDirection() == DirectionLong
}

func (o *Order) IsShort() bool {
	return o.GetDirection() == DirectionShort
}

func (o *Order) IsExpired() bool {
	var expiredAt time.Time

	switch o.GetType() {
	case MarketOrderType, LimitOrderType:
		order := o.Value.AsLimitOrder()
		if order.Expiration == 0 {
			return false
		}

		expiredAt = time.Unix(int64(order.Expiration), 0)

		break
	case TakeOrderType, StopOrderType:
		order := o.Value.AsStopOrder()
		if order.Expiration == 0 {
			return false
		}

		expiredAt = time.Unix(int64(order.Expiration), 0)

		break
	default:
		return false
	}

	return expiredAt.Before(time.Now())
}

func (o *Order) GetType() OrderType {
	switch o.Value.(type) {
	case StopOrder, *StopOrder:
		return StopOrderType
	case TakeOrder, *TakeOrder:
		return TakeOrderType
	case LimitOrder, *LimitOrder:
		return LimitOrderType
	case MarketOrder, *MarketOrder:
		return MarketOrderType
	case AddMarginOrder, *AddMarginOrder:
		return AddMarginOrderType
	case RemoveMarginOrder, *RemoveMarginOrder:
		return RemoveMarginOrderType
	}

	return UnknownOrderType
}

func (o *Order) IsDeferred() bool {
	switch {
	case o.GetType() == LimitOrderType:
		return true
	case o.GetType() == StopOrderType:
		return true
	case o.GetType() == TakeOrderType:
		return !IsMarketOrder(*o)
	default:
		return false
	}
}

func MapOrders(d *cell.Dictionary) (map[int]Order, error) {
	ret := map[int]Order{}

	v, err := d.LoadAll()
	if err != nil {
		return nil, err
	}

	for _, item := range v {
		v := Order{}

		key, err := item.Key.LoadUInt(3)

		if err != nil {
			return nil, err
		}

		ref, err := item.Value.LoadRef()
		if err != nil {
			return nil, err
		}

		if err = tlb.LoadFromCell(&v, ref); err != nil {
			return nil, err
		}

		ret[int(key)] = v
	}

	return ret, nil
}
