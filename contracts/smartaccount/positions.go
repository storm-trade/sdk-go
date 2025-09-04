package smartaccount

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	schemas "github.com/storm-trade/sdk-go/tlb"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/tvm/cell"
)

type UserPosition struct {
	Locked bool                   `tlb:"bool" json:"locked"`
	Data   *schemas.PositionState `tlb:"^" json:"data"`
}

type UserPositionKey struct {
	Market    string            `json:"market"`
	Direction schemas.Direction `json:"direction"`
}

func NewUserPositionKey(addr *address.Address, direction schemas.Direction) UserPositionKey {
	return UserPositionKey{
		Market:    addr.StringRaw(),
		Direction: direction,
	}
}

func (k UserPositionKey) String() string {
	return fmt.Sprintf("%s_%s", k.Market, k.Direction)
}

type UserPositions struct {
	Values map[UserPositionKey]*UserPosition
}

func (b *UserPositions) GetByAddressAndDir(addr *address.Address, direction schemas.Direction) (*UserPosition, bool) {
	v, ok := b.Values[NewUserPositionKey(addr, direction)]

	return v, ok
}

func (b *UserPositions) MarshalJSON() ([]byte, error) {
	result := make(map[string]*UserPosition)

	for k, v := range b.Values {
		result[k.String()] = v
	}

	return json.Marshal(result)
}

func (b *UserPositions) LoadFromCell(slice *cell.Slice) error {
	balanceDict, err := slice.LoadDict(268)
	if err != nil {
		return errors.Wrap(err, "load balances dict")
	}

	if balanceDict.IsEmpty() {
		return nil
	}

	kv, err := balanceDict.LoadAll()
	if err != nil {
		return errors.Wrap(err, "load all dict balances")
	}

	result := make(map[UserPositionKey]*UserPosition)

	for _, k := range kv {
		market := k.Key.MustLoadAddr()
		direction := k.Key.MustLoadUInt(1)

		pkey := UserPositionKey{
			Market:    market.StringRaw(),
			Direction: schemas.DirectionFromInt(int(direction)),
		}

		pvalue := new(UserPosition)
		if err := tlb.LoadFromCell(pvalue, k.Value); err != nil {
			return errors.Wrap(err, "load position")
		}

		result[pkey] = pvalue
	}

	b.Values = result

	return nil
}

func (b *UserPositions) ToDictionary() (*cell.Dictionary, error) {
	devicesDict := cell.NewDict(268)
	for k, v := range b.Values {
		mrkt := address.MustParseRawAddr(k.Market)

		key := cell.BeginCell().
			MustStoreAddr(mrkt).
			MustStoreUInt(uint64(k.Direction.GetInt()), 1).
			EndCell()

		pcell, err := tlb.ToCell(v)
		if err != nil {
			return nil, errors.Wrap(err, "to cell position")
		}

		if err = devicesDict.Set(key, pcell); err != nil {
			return nil, err
		}
	}

	return devicesDict, nil
}

func (b *UserPositions) ToCell() (*cell.Cell, error) {
	dict, err := b.ToDictionary()
	if err != nil {
		return nil, errors.Wrap(err, "to dictionary")
	}

	return cell.BeginCell().MustStoreDict(dict).EndCell(), nil
}
