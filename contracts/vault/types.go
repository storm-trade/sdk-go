package vault

import (
	"math/big"

	"github.com/xssnick/tonutils-go/address"
)

type VaultData struct {
	JettonWalletAddress *address.Address
	Rate                *big.Int
	LpTotalSupply       *big.Int
	FreeBalance         *big.Int
	LockedBalance       *big.Int
	BufferBalance       *big.Int
	StakersBalance      *big.Int
	ExecutorsBalance    *big.Int
	V3Paused            bool
}

type BufferData struct {
	Balance   *big.Int
	Rate      *big.Int
	UnderRate *big.Int
	OverRate  *big.Int
}
