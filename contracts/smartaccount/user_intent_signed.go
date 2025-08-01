package smartaccount

import (
	"crypto/ed25519"
	"github.com/xssnick/tonutils-go/tlb"
)

type SignedMessage struct {
	Message   *UserIntent `tlb:"^" json:"message"`
	Signature []byte      `tlb:"bits 512" json:"signature"`
	PublicKey PublicKey   `tlb:"bits 256" json:"public_key"`
}

func (msg *SignedMessage) Hash() (string, error) {
	return msg.Message.Hash()
}

func SignMessage(msg *UserIntent, sk ed25519.PrivateKey) (*SignedMessage, error) {
	c, err := tlb.ToCell(msg)
	if err != nil {
		return nil, err
	}

	sign := c.Sign(sk)
	pk := PublicKey(sk.Public().(ed25519.PublicKey))

	return &SignedMessage{Message: msg, Signature: sign, PublicKey: pk}, nil
}

type IntentRequestBody struct {
	_         tlb.Magic   `tlb:"#588b3270" json:"_"`
	Message   *UserIntent `tlb:"^" json:"message"`
	Signature []byte      `tlb:"bits 512" json:"signature"`
}
