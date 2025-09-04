package smartaccount

import (
	schemas "github.com/storm-trade/sdk-go/tlb"
	"github.com/storm-trade/sdk-go/tlb/opcode"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/tvm/cell"
)

func init() {
	tlb.Register(NotifyUpdatePosition{})
	tlb.Register(FailedBundleExecution{})

	tlb.Register(AddPublicKey{})
	tlb.Register(RemovePublicKey{})
	tlb.Register(RemoveAllExceptCurrentPublicKey{})
	tlb.Register(Withdraw{})
	tlb.Register(AddGasUnits{})

	tlb.Register(DepositNotify{})
	tlb.Register(DepositNotifyWithDeploy{})
}

var (
	OpNotifyUpdatePosition  = opcode.Opcode(0x1ca43d2f)
	OpTradeNotificationV2   = opcode.Opcode(0x28d36491)
	OpFailedBundleExecution = opcode.Opcode(0x666392ee)
	OpZero                  = opcode.Opcode(0x0)
)

//trade_notification_v2#662f2ce9 query_id:uint64 asset_id:uint16 free_amount:uint64 locked_amount:uint64 exchange_amount:uint64
//withdraw_locked_amount:uint64 fee_to_stakers:uint64 executor_amount:uint64 total_referrer_amount:uint64 ref_count:uint4
//referral_data:(HashmapE 4 ReferrerData) sa_address:MsgAddressInt notification_payload:^NotificationPayload = InternalMsgBody;
//notify_update_position#1ca43d2f query_id:uint64 jetton_minter_address:MsgAddressInt notification_payload:^NotificationPayload = InternalMsgBody;
//_ bundle_sender_address:MsgAddress balance_delta:uint64 amm_address:MsgAddressInt direction:Direction position_data:(Maybe ^PositionData) = NotificationPayload

type DepositNativePayload struct {
	_               tlb.Magic        `tlb:"#29bb3721" json:"_"`
	QueryID         uint64           `tlb:"## 64" json:"query_id"`
	Amount          *tlb.Coins       `tlb:"." json:"amount"`
	ReceiverAddress *address.Address `tlb:"addr" json:"receiver_address"`
	Init            bool             `tlb:"bool" json:"init"`
	KeyInit         *cell.Dictionary `tlb:"maybe dict 256" json:"key_init"`
}

type DepositJettonPayload struct {
	_               tlb.Magic        `tlb:"#76840119" json:"_"`
	QueryID         uint64           `tlb:"## 64" json:"query_id"`
	ReceiverAddress *address.Address `tlb:"addr" json:"receiver_address"`
	Init            bool             `tlb:"bool" json:"init"`
	KeyInit         *cell.Dictionary `tlb:"maybe dict 256" json:"key_init"`
}

type NotificationPayload struct {
	BundleSenderAddress *address.Address       `tlb:"addr" json:"bundle_sender_address"`
	BalanceDelta        *tlb.Coins             `tlb:"." json:"balance_delta"`
	AmmAddress          *address.Address       `tlb:"addr" json:"amm_address"`
	Direction           uint8                  `tlb:"## 1" json:"direction"`
	PositionState       *schemas.PositionState `tlb:"maybe ^" json:"position_state"`
}

type NotifyUpdatePosition struct {
	_                   tlb.Magic           `tlb:"#1ca43d2f"`
	QueryId             uint64              `tlb:"## 64" json:"query_id"`
	JettonMinter        *address.Address    `tlb:"addr" json:"jetton_minter"`
	NotificationPayload NotificationPayload `tlb:"^" json:"notification_payload"`
}

type FailedBundleExecution struct {
	_         tlb.Magic `tlb:"#666392ee"`
	QueryId   uint64    `tlb:"## 64" json:"query_id"`
	AssetId   uint64    `tlb:"## 16" json:"asset_id"`
	ErrorCode uint64    `tlb:"## 16" json:"error_code"`
}

type Withdraw struct {
	_            tlb.Magic        `tlb:"#6eec039d" json:"_"`
	QueryID      uint64           `tlb:"## 64" json:"query_id"`
	VaultAddress *address.Address `tlb:"addr" json:"vault_address"`
	Amount       *tlb.Coins       `tlb:"." json:"amount"`
}

type AddPublicKey struct {
	_         tlb.Magic `tlb:"#220c4c19" json:"_"`
	QueryID   uint64    `tlb:"## 64" json:"query_id"`
	PublicKey PublicKey `tlb:"bits 256" json:"public_key"`
}

type RemovePublicKey struct {
	_         tlb.Magic `tlb:"#76519f8b" json:"_"`
	QueryID   uint64    `tlb:"## 64" json:"query_id"`
	PublicKey PublicKey `tlb:"bits 256" json:"public_key"`
}

type RemoveAllExceptCurrentPublicKey struct {
	_         tlb.Magic `tlb:"#644794b8" json:"_"`
	QueryID   uint64    `tlb:"## 64" json:"query_id"`
	PublicKey PublicKey `tlb:"bits 256" json:"public_key"`
}

type AddGasUnits struct {
	_ tlb.Magic `tlb:"#5a091c43" json:"_"`
}

type DepositNotifyWithDeploy struct {
	_             tlb.Magic        `tlb:"#18a092f7" json:"_"`
	QueryID       uint64           `tlb:"## 64" json:"query_id"`
	VaultAddress  *address.Address `tlb:"addr" json:"vault_address"`
	Amount        *tlb.Coins       `tlb:"." json:"amount"`
	SenderAddress *address.Address `tlb:"addr" json:"sender_address"`
	InitData      *InitData        `tlb:"." json:"init_data"`
}

type DepositNotify struct {
	_                   tlb.Magic        `tlb:"#186b2edf" json:"_"`
	QueryID             uint64           `tlb:"## 64" json:"query_id"`
	Amount              *tlb.Coins       `tlb:"." json:"amount"`
	SenderAddress       *address.Address `tlb:"addr" json:"sender_address"`
	JettonMinterAddress *address.Address `tlb:"addr" json:"jetton_minter_address"`
}

type InitData struct {
	HighloadTimeout uint64     `tlb:"## 24" json:"direction"`
	Version         uint8      `tlb:"## 8" json:"Version"`
	Keys            *Keys      `tlb:"^" json:"keys"`
	NewCode         *cell.Cell `tlb:"^" json:"new_code"`
}
