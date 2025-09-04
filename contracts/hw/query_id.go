package hw

var MaxQueryId = 1023 * 8192

type QueryId struct {
	Shift     uint16 `tlb:"## 13" json:"shift"`
	BitNumber uint16 `tlb:"## 10" json:"bit_number"`
}

func (i QueryId) Seqno() uint64 {
	return uint64(i.BitNumber) + uint64(i.Shift)*1023
}

func FromSeqno(i uint64) *QueryId {
	shift := i / 1023
	bitNumber := i % 1023

	return &QueryId{
		Shift:     uint16(shift),
		BitNumber: uint16(bitNumber),
	}
}

func FromShiftAndBitNumber(shift, bitnumber uint16) *QueryId {
	return &QueryId{
		Shift:     shift,
		BitNumber: bitnumber,
	}
}
