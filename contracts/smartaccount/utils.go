package smartaccount

import (
	"encoding/hex"
	"github.com/pkg/errors"
	"github.com/storm-trade/sdk-go/contracts/hw"
	tutlb "github.com/xssnick/tonutils-go/tlb"
)

var (
	ErrInvalidSignature = errors.New("invalid user signature")
)

var (
	MinExecutorQueryID = hw.QueryId{Shift: 6000, BitNumber: 0}
	MaxUserQueryID     = hw.QueryId{Shift: 5999, BitNumber: 1022}
)

func ValidateSignedPayload(payload *UserIntent, pubKey []byte, signature []byte) (string, error) {
	c, err := tutlb.ToCell(payload)
	if err != nil {
		return "", errors.Wrap(err, "serialize message to cell")
	}

	if !c.Verify(pubKey, signature) {
		return "", errors.Wrapf(ErrInvalidSignature, "signature verification failed for pk=%s siganture=%s hash=%s", hex.EncodeToString(pubKey), hex.EncodeToString(signature), hex.EncodeToString(c.Hash()))
	}

	return hex.EncodeToString(c.Hash()), nil
}
