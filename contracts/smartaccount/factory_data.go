package smartaccount

import (
	"math/big"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tvm/cell"
)

type FactoryData struct {
	AdminAddress    *address.Address
	HighloadTimeout uint32
	HotPublicKey    *big.Int
	ColdPublicKey   *big.Int
	Content         *cell.Cell
	Version         uint32
	Code            *cell.Cell
}

type MinFees struct {
	DepositMinFeeNative           *big.Int
	DepositMinFeeJetton           *big.Int
	DepositWithDeployMinFeeNative *big.Int
	DepositWithDeployMinFeeJetton *big.Int
	WithdrawMinFee                *big.Int
	StorageFee                    *big.Int
}

type KeysData struct {
	HotPublicKey  *big.Int
	ColdPublicKey *big.Int
	UserKeys      *cell.Cell
	KeysCount     uint32
}

type NftData struct {
	Init              bool
	Index             *big.Int
	CollectionAddress *address.Address
	OwnerAddress      *address.Address
	Content           *cell.Cell
}

type HighloadData struct {
	OldQueries    *cell.Cell
	Queries       *cell.Cell
	LastCleanTime uint32
	Timeout       uint32
}

type PositionRecord struct {
	Locked bool
	Data   *cell.Cell
}
