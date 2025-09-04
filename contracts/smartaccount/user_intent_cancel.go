package smartaccount

import (
	"encoding/hex"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
)

type CancelMessage struct {
	SmartAccountAddress *address.Address `tlb:"addr" json:"smart_account_address"`
	OrderId             []byte           `tlb:"bits 256" json:"order_id"`
}

func (msg *CancelMessage) Hash() (string, error) {
	msgCell, err := tlb.ToCell(msg)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(msgCell.Hash()), nil
}

type SignedCancelMessage struct {
	_         tlb.Magic      `tlb:"#db3ee02c" json:"_"`
	Message   *CancelMessage `tlb:"^" json:"message"`
	PublicKey []byte         `tlb:"bits 256" json:"public_key"`
	Signature []byte         `tlb:"bits 512" json:"signature"`
}

func (msg *SignedCancelMessage) Hash() (string, error) {
	return msg.Message.Hash()
}
