package smartaccount

import (
	"crypto/ed25519"
	"encoding/json"
	"github.com/pkg/errors"
	"github.com/storm-trade/sdk-go/contracts/hw"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tvm/cell"
	"math/big"
)

type AccountData struct {
	Type      uint8            `tlb:"## 8" json:"type"`
	Factory   *address.Address `tlb:"addr" json:"factory"`
	Owner     *address.Address `tlb:"addr" json:"owner"`
	Balances  *BalanceList     `tlb:"." json:"balances"`
	Version   uint8            `tlb:"## 8" json:"version"`
	Keys      *Keys            `tlb:"^" json:"keys"`
	Positions *UserPositions   `tlb:"." json:"positions"`
	Highload  *Highload        `tlb:"^" json:"highload"`
}

type AccountDataBlank struct {
	Type     uint8            `tlb:"## 8" json:"type"`
	Factory  *address.Address `tlb:"addr" json:"factory"`
	Owner    *address.Address `tlb:"addr" json:"owner"`
	Balances *BalanceList     `tlb:"." json:"balances"`
}

type Keys struct {
	Hot            []byte          `tlb:"bits 256" json:"hot"`
	Cold           []byte          `tlb:"bits 256" json:"cold"`
	UserPublicKeys *UserPublicKeys `tlb:"." json:"user_public_keys"`
	KeysCount      int64           `tlb:"## 8" json:"keys_count"`
}

type Highload struct {
	OldQueries    *cell.Dictionary `tlb:"dict 13" json:"old_queries"`
	Queries       *cell.Dictionary `tlb:"dict 13" json:"queries"`
	LastCleanTime uint64           `tlb:"## 64" json:"last_clean_time"`
	Timeout       uint64           `tlb:"## 24" json:"timeout"`
}

func QueriesUsed(dictionary *cell.Dictionary) ([]uint64, error) {
	queriesUsed := make([]uint64, 0)
	kv, err := dictionary.LoadAll()
	if err != nil {
		return nil, err
	}

	for _, k := range kv {
		shift, err := k.Key.LoadUInt(13)
		if err != nil {
			return nil, err
		}

		bitsRef, err := k.Value.LoadRef()
		if err != nil {
			return nil, err
		}

		bits := bitsRef.BitsLeft()
		for bit := range bits {
			v := bitsRef.MustLoadBoolBit()

			if v {
				qid := hw.QueryId{Shift: uint16(shift), BitNumber: uint16(bit)}
				queriesUsed = append(queriesUsed, qid.Seqno())
			}
		}

	}

	return queriesUsed, nil
}

func (h *Highload) MarshalJSON() ([]byte, error) {
	oldQueries, err := QueriesUsed(h.OldQueries)
	if err != nil {
		return nil, err
	}

	queries, err := QueriesUsed(h.Queries)
	if err != nil {
		return nil, err
	}

	res := struct {
		OldQueries    []uint64 `json:"old_queries"`
		Queries       []uint64 `json:"queries"`
		LastCleanTime uint64   `json:"last_clean_time"`
		Timeout       uint64   `json:"timeout"`
	}{
		OldQueries:    oldQueries,
		Queries:       queries,
		LastCleanTime: h.LastCleanTime,
		Timeout:       h.Timeout,
	}

	return json.Marshal(res)
}

type UserPublicKeys struct {
	Values []PublicKey `json:"values"`
}

type PublicKey ed25519.PublicKey

func (m *UserPublicKeys) Add(key PublicKey) {
	m.Values = append(m.Values, key)
}

func (m *UserPublicKeys) LoadFromCell(slice *cell.Slice) error {
	deviceDict, err := slice.LoadDict(256)
	if err != nil {
		return errors.Wrap(err, "load devices dict")
	}

	if deviceDict.IsEmpty() {
		m.Values = make([]PublicKey, 0)
		return nil
	}

	kv, err := deviceDict.LoadAll()
	if err != nil {
		return errors.Wrap(err, "load all dict devices")
	}

	pks := make([]PublicKey, 0)
	for _, v := range kv {
		pk, err := v.Key.LoadBigUInt(256)
		if err != nil {
			return errors.Wrap(err, "load all dict devices")
		}

		pks = append(pks, pk.Bytes())
	}

	m.Values = pks

	return nil
}

func (m *UserPublicKeys) ToCell() (*cell.Cell, error) {
	dict := cell.NewDict(256)

	for _, item := range m.Values {
		key := cell.BeginCell().MustStoreBigInt(new(big.Int).SetBytes(item), 256).EndCell()
		err := dict.Set(key, cell.BeginCell().EndCell())

		if err != nil {
			return nil, err
		}
	}

	return cell.BeginCell().MustStoreDict(dict).EndCell(), nil
}

func (m *UserPublicKeys) ToDictionary() (*cell.Dictionary, error) {
	dict := cell.NewDict(256)

	for _, item := range m.Values {
		key := cell.BeginCell().MustStoreBigInt(new(big.Int).SetBytes(item), 256).EndCell()
		err := dict.Set(key, cell.BeginCell().EndCell())

		if err != nil {
			return nil, err
		}
	}

	return dict, nil
}

type BalanceList struct {
	Balances map[*address.Address]uint64 `json:"balances"`
}

type BalanceEntry struct {
	Addr  address.Address `json:"address"`
	Count uint64          `json:"amount"`
}

func (b *BalanceList) Get(addr *address.Address) (uint64, error) {
	for k, v := range b.Balances {
		if k.StringRaw() == addr.StringRaw() {
			return v, nil
		}
	}

	return 0, nil
}

func (b *BalanceList) Set(addr *address.Address, v uint64) {
	b.Balances[addr] = v
}

func (b *BalanceList) MarshalJSON() ([]byte, error) {
	entries := make([]BalanceEntry, 0, len(b.Balances))
	for addrPtr, count := range b.Balances {
		entries = append(entries, BalanceEntry{
			Addr:  *addrPtr,
			Count: count,
		})
	}
	return json.Marshal(entries)
}

func (b *BalanceList) LoadFromCell(slice *cell.Slice) error {
	balanceDict, err := slice.LoadDict(267)
	if err != nil {
		return errors.Wrap(err, "load balances dict")
	}

	if balanceDict.IsEmpty() {
		b.Balances = make(map[*address.Address]uint64)
		return nil
	}

	kv, err := balanceDict.LoadAll()
	if err != nil {
		return errors.Wrap(err, "load all dict balances")
	}

	balances := make(map[*address.Address]uint64)

	for _, v := range kv {
		addr := v.Key.MustLoadAddr()
		balance := v.Value.MustLoadCoins()
		balances[addr] = balance
	}

	b.Balances = balances

	return nil
}

func (b *BalanceList) ToDictionary() (*cell.Dictionary, error) {
	devicesDict := cell.NewDict(267)
	for k, v := range b.Balances {
		key := cell.BeginCell().MustStoreAddr(k).EndCell()
		value := cell.BeginCell().MustStoreCoins(v).EndCell()

		err := devicesDict.Set(key, value)
		if err != nil {
			return nil, err
		}
	}

	return devicesDict, nil
}

func (b *BalanceList) ToCell() (*cell.Cell, error) {
	dict, err := b.ToDictionary()
	if err != nil {
		return nil, errors.Wrap(err, "to dictionary")
	}

	return cell.BeginCell().MustStoreDict(dict).EndCell(), nil
}
