package stormv2

import (
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/tvm/cell"
)

const (
	opRequestCreateOrder = 0xe0db7753
	opCreateOrderPM      = 0xa39843f4
	opCancelOrder        = 0x67134629
	opAddMargin          = 0xb9e810e2
	opRemoveMargin       = 0xecded426
	opProvidePosition    = 0x13076670
	opJettonTransfer     = 0x0f8a7ea5
)

const (
	orderTypeStopLoss   = 0
	orderTypeTakeProfit = 1
	orderTypeLimit      = 2
	orderTypeMarket     = 3
)

var (
	GasCreateOrder   = tlb.MustFromTON("0.225")
	GasForwardOrder  = tlb.MustFromTON("0.18")
	GasMargin        = tlb.MustFromTON("0.35")
	GasForwardMargin = tlb.MustFromTON("0.305")
	GasCancelOrder   = tlb.MustFromTON("0.3")
)

type createOrderParams struct {
	orderType        uint64
	assetID          uint16
	direction        uint64
	leverage         uint64
	expiration       uint32
	limitPrice       tlb.Coins
	stopPrice        tlb.Coins
	stopTriggerPrice tlb.Coins
	takeTriggerPrice tlb.Coins
	gasToAddress     *address.Address
	initPM           bool
	referralID       *uint64
}

func buildCreateOrderBody(p createOrderParams) *cell.Builder {
	b := cell.BeginCell().
		MustStoreUInt(uint64(p.assetID), 16).
		MustStoreUInt(p.orderType, 4).
		MustStoreAddr(p.gasToAddress).
		MustStoreBoolBit(p.initPM)

	if p.initPM {
		if p.referralID != nil {
			b.MustStoreBoolBit(true)
			b.MustStoreUInt(*p.referralID, 64)
		} else {
			b.MustStoreBoolBit(false)
		}
	}

	b.MustStoreRef(cell.BeginCell().
		MustStoreUInt(p.leverage, 64).
		MustStoreUInt(uint64(p.expiration), 32).
		MustStoreUInt(p.direction, 1).
		MustStoreBigCoins(p.limitPrice.Nano()).
		MustStoreBigCoins(p.stopPrice.Nano()).
		MustStoreBigCoins(p.stopTriggerPrice.Nano()).
		MustStoreBigCoins(p.takeTriggerPrice.Nano()).
		EndCell())

	return b
}

func buildNativeCreateOrderCell(p createOrderParams, amount tlb.Coins) *cell.Cell {
	b := cell.BeginCell().
		MustStoreUInt(opRequestCreateOrder, 32).
		MustStoreBigCoins(amount.Nano())

	body := buildCreateOrderBody(p)
	return b.MustStoreBuilder(body).EndCell()
}

func buildJettonCreateOrderCell(p createOrderParams) *cell.Cell {
	b := cell.BeginCell().
		MustStoreUInt(opRequestCreateOrder, 32)

	body := buildCreateOrderBody(p)
	return b.MustStoreBuilder(body).EndCell()
}

func wrapJetton(orderCell *cell.Cell, jettonAmount tlb.Coins, vaultAddr, responseAddr *address.Address, forwardTON tlb.Coins, queryID uint64) *cell.Cell {
	return cell.BeginCell().
		MustStoreUInt(opJettonTransfer, 32).
		MustStoreUInt(queryID, 64).
		MustStoreBigCoins(jettonAmount.Nano()).
		MustStoreAddr(vaultAddr).
		MustStoreAddr(responseAddr).
		MustStoreMaybeRef(nil).
		MustStoreBigCoins(forwardTON.Nano()).
		MustStoreMaybeRef(orderCell).
		EndCell()
}

type sltpParams struct {
	orderType    uint64
	direction    uint64
	expiration   uint32
	amount       tlb.Coins
	triggerPrice tlb.Coins
	gasToAddress *address.Address
}

func buildSLTPCell(p sltpParams) *cell.Cell {
	return cell.BeginCell().
		MustStoreUInt(opCreateOrderPM, 32).
		MustStoreUInt(p.orderType, 4).
		MustStoreAddr(p.gasToAddress).
		MustStoreRef(cell.BeginCell().
			MustStoreUInt(p.orderType, 4).
			MustStoreUInt(uint64(p.expiration), 32).
			MustStoreUInt(p.direction, 1).
			MustStoreBigCoins(p.amount.Nano()).
			MustStoreBigCoins(p.triggerPrice.Nano()).
			EndCell()).
		EndCell()
}

func buildCancelCell(orderType, orderIndex, direction uint64, gasToAddress *address.Address) *cell.Cell {
	return cell.BeginCell().
		MustStoreUInt(opCancelOrder, 32).
		MustStoreUInt(orderType, 4).
		MustStoreUInt(orderIndex, 3).
		MustStoreUInt(direction, 1).
		MustStoreAddr(gasToAddress).
		EndCell()
}

func buildAddMarginCell(assetID uint16, direction uint64, gasToAddress *address.Address, oraclePayload *cell.Cell) *cell.Cell {
	return cell.BeginCell().
		MustStoreUInt(opAddMargin, 32).
		MustStoreUInt(uint64(assetID), 16).
		MustStoreUInt(direction, 1).
		MustStoreAddr(gasToAddress).
		MustStoreRef(oraclePayload).
		EndCell()
}

func buildAddMarginNativeCell(amount tlb.Coins, assetID uint16, direction uint64, gasToAddress *address.Address, oraclePayload *cell.Cell) *cell.Cell {
	return cell.BeginCell().
		MustStoreUInt(opAddMargin, 32).
		MustStoreBigCoins(amount.Nano()).
		MustStoreUInt(uint64(assetID), 16).
		MustStoreUInt(direction, 1).
		MustStoreAddr(gasToAddress).
		MustStoreRef(oraclePayload).
		EndCell()
}

func buildRemoveMarginCell(direction uint64, amount tlb.Coins, gasToAddress *address.Address, oraclePayload *cell.Cell) *cell.Cell {
	return cell.BeginCell().
		MustStoreUInt(opProvidePosition, 32).
		MustStoreUInt(direction, 1).
		MustStoreAddr(gasToAddress).
		MustStoreUInt(opRemoveMargin, 32).
		MustStoreBigCoins(amount.Nano()).
		MustStoreRef(oraclePayload).
		EndCell()
}
