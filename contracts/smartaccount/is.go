package smartaccount

import (
	"github.com/xssnick/tonutils-go/tvm/cell"
)

type Opcode uint32

const (
	OpMessage     Opcode = 0x588b3270
	OpCancelOrder Opcode = 0xdb3ee02c
)

func IsIntentOpcode(op Opcode) bool {
	return op == OpMessage || op == OpCancelOrder
}

func IsValidMessage(cell *cell.Cell) bool {
	sc := cell.MustBeginParse()

	if sc.RefsNum() != 1 {
		return false
	}

	opcode, err := sc.LoadUInt(32)
	if err != nil {
		return false
	}

	return IsIntentOpcode(Opcode(opcode))
}
