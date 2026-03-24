package positionmanager

import (
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tvm/cell"
)

type PositionManagerData struct {
	TraderAddress *address.Address
	VaultAddress  *address.Address
	VammAddress   *address.Address
	LongPosition  *cell.Cell
	ShortPosition *cell.Cell
	Orders        *cell.Cell
	ReferralData  *cell.Cell
	OrdersBitset  uint32
}
