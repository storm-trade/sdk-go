package tlb

import (
	"encoding/json"

	"github.com/xssnick/tonutils-go/tlb"
)

type OrderRequest struct {
	_        tlb.Magic `tlb:"#8"`
	Selector uint      `tlb:"## 1" json:"selector"`
	Order    AnyOrder  `tlb:"[LimitOrder,MarketOrder]" json:"order"`
}

func (s OrderRequest) AsStopOrder() *StopOrderData     { return nil }
func (s OrderRequest) AsLimitOrder() *LimitOrderData   { return nil }
func (s OrderRequest) AsMarginOrder() *MarginOrderData { return nil }

func (s OrderRequest) GetDirection() uint {
	switch internal := s.Order.(type) {
	case MarketOrder:
		return internal.AsLimitOrder().Direction
	case LimitOrder:
		return internal.AsLimitOrder().Direction
	default:
		panic("invalid order type in order request")
	}
}

func (s OrderRequest) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Selector uint   `json:"selector"`
		Order    *Order `json:"order"`
		Type     string `json:"type"`
	}{
		Selector: s.Selector,
		Order:    &Order{Value: s.Order},
		Type:     OrderRequestOrderType.String(),
	})
}

func (s *OrderRequest) UnmarshalJSON(value []byte) error {
	str := struct {
		Selector uint   `json:"selector"`
		Order    *Order `json:"order"`
	}{}

	if err := json.Unmarshal(value, &str); err != nil {
		return err
	}

	s.Selector = str.Selector
	s.Order = str.Order.Value

	return nil
}
