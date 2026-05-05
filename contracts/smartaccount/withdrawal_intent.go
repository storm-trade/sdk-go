package smartaccount

import (
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
)

type WithdrawalPayload struct {
	SmartAccount *address.Address `tlb:"addr"`
	AssetAddress *address.Address `tlb:"addr"`
	Amount       *tlb.Coins       `tlb:"."`
	Nonce        uint64           `tlb:"## 64"`
	CreatedAt    uint64           `tlb:"## 32"`
}
