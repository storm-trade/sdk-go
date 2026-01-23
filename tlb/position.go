package tlb

import (
	"github.com/pkg/errors"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/tvm/cell"
	"math/big"
)

type ReferralData struct {
	Address  *address.Address `tlb:"addr" json:"address"`
	Discount uint64           `tlb:"## 32" json:"discount"`
	Rebate   uint64           `tlb:"## 32" json:"rebate"`
}

type PositionStorage struct {
	TraderAddress *address.Address `tlb:"addr" json:"trader_address"`
	VaultAddress  *address.Address `tlb:"addr" json:"vault_address"`
	AmmAddress    *address.Address `tlb:"addr" json:"amm_address"`
	Long          *PositionRef     `tlb:"maybe ^" json:"long"`
	Short         *PositionRef     `tlb:"maybe ^" json:"short"`
	ReferralData  *ReferralData    `tlb:"maybe ^" json:"referral_data"`
	OrdersDict    *cell.Dictionary `tlb:"dict 3" json:"limit_orders"`
	OrdersBitset  uint32           `tlb:"## 8" json:"orders_bitset,omitempty"`
}

func (ps *PositionStorage) ShortOrders() map[int]Order {
	orders := make(map[int]Order, 0)

	if ps.Short != nil {
		return ps.Short.Orders()
	}

	return orders
}

func (ps *PositionStorage) LongOrders() map[int]Order {
	orders := make(map[int]Order, 0)

	if ps.Long != nil {
		return ps.Long.Orders()
	}

	return orders
}

func (ps *PositionStorage) AllOrders() []Order {
	orders := make([]Order, 0)

	if ps.Short != nil {
		for _, order := range ps.Short.Orders() {
			orders = append(orders, order)
		}
	}

	if ps.Long != nil {
		for _, order := range ps.Long.Orders() {
			orders = append(orders, order)
		}
	}

	for _, order := range ps.Orders() {
		orders = append(orders, order)
	}

	return orders
}

func (ps *PositionStorage) OrdersCount() int64 {
	return int64(len(ps.AllOrders()))
}

func (ps *PositionStorage) Orders() map[int]Order {
	orders, _ := MapOrders(ps.OrdersDict)

	return orders
}

func (ps *PositionStorage) Slice() []*PositionState {
	return MapPositions(ps)
}

type PositionRef struct {
	Locked          bool             `tlb:"bool" json:"locked"`
	RedirectAddress *address.Address `tlb:"addr" json:"redirect_address"`
	OrdersBitset    uint32           `tlb:"## 8" json:"orders_bitset"`
	OrdersDict      *cell.Dictionary `tlb:"dict 3" json:"orders"`
	Position        *PositionState   `tlb:"^" json:"state"`
}

func (ps *PositionRef) Orders() map[int]Order {
	orders, _ := MapOrders(ps.OrdersDict)

	return orders
}

type PositionState struct {
	Size                         *big.Int   `tlb:"## 128" json:"size"`
	Direction                    uint8      `tlb:"## 1" json:"direction"`
	Margin                       *tlb.Coins `tlb:"." json:"margin"`
	OpenNotional                 *tlb.Coins `tlb:"." json:"open_notional"`
	LastUpdatedCumulativePremium int64      `tlb:"## 64" json:"last_updated_cumulative_premium"`
	Fee                          uint64     `tlb:"## 32" json:"fee"`
	Discount                     uint64     `tlb:"## 32" json:"discount"`
	Rebate                       uint64     `tlb:"## 32" json:"rebate"`
	LastUpdatedTimestamp         uint64     `tlb:"## 32" json:"last_updated_timestamp"`
}

func (ps *PositionState) GetDirection() Direction {
	return DirectionFromInt(int(ps.Direction))
}

func MapPositions(storage *PositionStorage) []*PositionState {
	values := make([]*PositionState, 0)

	if storage == nil {
		return values
	}

	func() {
		if storage.Long == nil {
			return
		}
		if err := validateState(storage.Long.Position); err != nil {
			return
		}

		values = append(values, storage.Long.Position)
	}()

	func() {
		if storage.Short == nil {
			return
		}
		if err := validateState(storage.Short.Position); err != nil {
			return
		}

		values = append(values, storage.Short.Position)
	}()

	return values
}

func validateState(st *PositionState) error {
	if st.OpenNotional.Nano().Int64() < 10 {
		return errors.New("small position size")
	}

	return nil
}
