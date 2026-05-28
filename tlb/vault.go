package tlb

import (
	"math/big"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/tvm/cell"
)

type WhitelistedAddress = int

const (
	LpMinterAddressKey           WhitelistedAddress = 0
	ReferralCollectionAddressKey WhitelistedAddress = 1
	ExecutorCollectionAddressKey WhitelistedAddress = 2
	AdminAddressKey              WhitelistedAddress = 3
)

type BufferParams struct {
	Rate      *tlb.Coins `tlb:"." json:"rate"`
	UnderRate *tlb.Coins `tlb:"." json:"under_rate"`
	OverRate  *tlb.Coins `tlb:"." json:"over_rate"`
}

type VaultData struct {
	JettonAddress      *address.Address `tlb:"addr" json:"jetton_address"`
	Rate               uint64           `tlb:"## 64" json:"rate"`
	TotalSupply        *tlb.Coins       `tlb:"." json:"total_supply"`
	FreeBalance        *tlb.Coins       `tlb:"." json:"free_balance"`
	LockedBalance      *tlb.Coins       `tlb:"." json:"locked_balance"`
	BufferBalance      uint64           `tlb:"## 64" json:"buffer_balance"`
	StakersBalance     *tlb.Coins       `tlb:"." json:"stakers_balance"`
	ExecutorsBalance   *tlb.Coins       `tlb:"." json:"executors_balance"`
	WhitelistAddresses *cell.Dictionary `tlb:"dict 4" json:"whitelist_addresses"`
	Buffer             *BufferParams    `tlb:"^" json:"buffer"`
}

func (v *VaultData) GetWhiteListAddress(t WhitelistedAddress) (*address.Address, error) {
	res, err := v.WhitelistAddresses.LoadValueByIntKey(big.NewInt(int64(t)))
	if err != nil {
		return nil, err
	}

	addr, err := res.LoadAddr()
	if err != nil {
		return nil, err
	}

	return addr, nil
}
