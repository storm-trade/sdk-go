package smartaccount_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/require"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/tvm/cell"
	"math/big"
	"storm-go/pkg/schemas"
	"storm-go/sequencer/offchain/contracts/smartaccount"
	"testing"
	"time"
)

func GenerateRandomAddress() *address.Address {
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	addr := address.NewAddress(0, 0, cell.BeginCell().MustStoreSlice(pub, 32).EndCell().Hash())

	return addr
}

func GenerateRandomKey() smartaccount.PublicKey {
	pub, _, _ := ed25519.GenerateKey(rand.Reader)

	return smartaccount.PublicKey(pub)
}

func GenerateRandomInt(max int64) *big.Int {
	n, _ := rand.Int(rand.Reader, big.NewInt(max))
	return n
}

func GenerateRandomPosition() *schemas.PositionState {
	Notional, _ := tlb.FromNano(GenerateRandomInt(1_000_000), 9)
	Margin, _ := tlb.FromNano(GenerateRandomInt(1_000_000), 9)

	return &schemas.PositionState{
		Size:                         GenerateRandomInt(1_000_000_000),
		Direction:                    0,
		Margin:                       &Margin,
		OpenNotional:                 &Notional,
		LastUpdatedCumulativePremium: GenerateRandomInt(1_000_000).Uint64(),
		Fee:                          GenerateRandomInt(1_000_000).Uint64(),
		Discount:                     GenerateRandomInt(1_000_000).Uint64(),
		Rebate:                       GenerateRandomInt(1_000_000).Uint64(),
		LastUpdatedTimestamp:         uint64(time.Now().Unix()),
	}
}

func Test_LoadSmartAccountState(t *testing.T) {
	t.Run("should serialize account", func(t *testing.T) {
		accountData := smartaccount.AccountData{
			Type:    0,
			Factory: GenerateRandomAddress(),
			Owner:   GenerateRandomAddress(),
			Balances: &smartaccount.BalanceList{Balances: map[*address.Address]uint64{
				GenerateRandomAddress(): 228,
			}},
			Version: 0,
			Keys: &smartaccount.Keys{
				Hot:  GenerateRandomKey(),
				Cold: GenerateRandomKey(),
				UserPublicKeys: &smartaccount.UserPublicKeys{
					Values: []smartaccount.PublicKey{
						GenerateRandomKey(),
					},
				},
				KeysCount: 2,
			},
			Positions: &smartaccount.UserPositions{
				Values: map[smartaccount.UserPositionKey]*smartaccount.UserPosition{
					smartaccount.NewUserPositionKey(GenerateRandomAddress(), schemas.DirectionLong): {
						Locked: false,
						Data:   GenerateRandomPosition(),
					},
				},
			},
			Highload: &smartaccount.Highload{
				OldQueries:    cell.NewDict(13),
				Queries:       cell.NewDict(13),
				LastCleanTime: uint64(time.Now().Unix()),
				Timeout:       uint64(time.Now().Unix()) / 60 / 60 / 24,
			},
		}

		c, err := tlb.ToCell(accountData)
		require.Nil(t, err)

		loaded := new(smartaccount.AccountData)
		err = tlb.LoadFromCell(loaded, c.BeginParse())
		require.Nil(t, err)

		require.Equal(t, loaded.Factory.String(), accountData.Factory.String())
		require.Equal(t, loaded.Owner.String(), accountData.Owner.String())
		require.Equal(t, loaded.Version, accountData.Version)

		// balances
		values := map[string]uint64{}
		for k, v := range accountData.Balances.Balances {
			values[k.StringRaw()] = v
		}

		for k, v := range loaded.Balances.Balances {
			require.Equal(t, values[k.StringRaw()], v)
		}

		// keys
		require.Equal(t, accountData.Keys.KeysCount, loaded.Keys.KeysCount)
		require.Equal(t, hex.EncodeToString(accountData.Keys.Cold), hex.EncodeToString(loaded.Keys.Cold))
		require.Equal(t, hex.EncodeToString(accountData.Keys.Hot), hex.EncodeToString(loaded.Keys.Hot))

		pkValues := map[string]any{}
		for _, v := range accountData.Keys.UserPublicKeys.Values {
			pkValues[hex.EncodeToString(v)] = true
		}

		for _, v := range loaded.Keys.UserPublicKeys.Values {
			require.Equal(t, pkValues[hex.EncodeToString(v)], true)
		}

		// highload
		require.Equal(t, accountData.Highload.LastCleanTime, loaded.Highload.LastCleanTime)
		require.Equal(t, accountData.Highload.Timeout, loaded.Highload.Timeout)

		// positions
		posValues := map[string]*smartaccount.UserPosition{}
		for _, v := range accountData.Keys.UserPublicKeys.Values {
			pkValues[hex.EncodeToString(v)] = true
		}

		for k, v := range accountData.Positions.Values {
			posValues[k.String()] = v
		}

		for k, v := range loaded.Positions.Values {
			before, ok := posValues[k.String()]

			require.Equal(t, ok, true)

			require.Equal(t, v.Locked, before.Locked)

			require.Equal(t, v.Data.Direction, before.Data.Direction)
			require.Equal(t, v.Data.Fee, before.Data.Fee)
			require.Equal(t, v.Data.OpenNotional, before.Data.OpenNotional)
			require.Equal(t, v.Data.Size, before.Data.Size)
			require.Equal(t, v.Data.Margin, before.Data.Margin)
			require.Equal(t, v.Data.Discount, before.Data.Discount)
			require.Equal(t, v.Data.Rebate, before.Data.Rebate)
			require.Equal(t, v.Data.LastUpdatedTimestamp, before.Data.LastUpdatedTimestamp)
		}
	})

	t.Run("should deserialize account from tonviewer", func(t *testing.T) {
		accountData := new(smartaccount.AccountData)
		dataHex := "b5ee9c720102230100062700048a008009566c885eef33f5ec5c452238ca7a2c3a6836bb6166ff89526d336f24673bd19002a20874b9a2f5fa9ee7071e1e07cc88ac7d8f7f9a90e59514792e289f9573024601010203040051a17000af6438db768342b19bd68b959c7d0f7cc66edf9254848a076dcdf8f4bf6462497291f70764a00183f623a002bacde0eb34c0b9cd533b2d37b8b5379fbc288cd10d0fe5f28c287b17616be6147717a9226c66caf66cd6be0e50e5920956e344561cbfb7b342f4259181c005020581700208090117400000001a28d049bb5380200c02014806070042bf8fe2bd6270350ec3da187b433da5b7f36a42594e37b5d914684c4d1bb6a243df0042bf88082bdfb3d8f60d2687634f15404bbf1e2bb3d957c9d3bb80e7024a009adb920143a016a4251061afce588ba81c7db1c3b1cf85dc31bb8cb22f0323d19fa4c3b6b99d080a0143a0002e9ec97ae88e7666fcca6b12b7ce86478a2c9a49396293ceb462940928eb32880b006500000000000000000000002325d2fbd721d640ec728125e893c600000000001be1fd800927c000000000000000003451a44240006900000000000000000000008ea9dc6d8e28e1fe95cffb008d3f1d9a068000000002988ff8800927c000000000000000003452c8b5c00201480d0e0101d70f0201e7101100ffb1ff0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000010103bdd812020120131400ff00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000400000000010103b45015020158161700ff000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000102015818190101d2220101f41a0201201b1c00ff04000008000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000010201201d1e010158210101201f0101202000ff000000000000000000000000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000100ff000000000000000000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000100ff000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000000000100ff0000000000000000000000000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000001000000000000000004000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001"

		dataBoc, err := hex.DecodeString(dataHex)
		require.Nil(t, err)

		dataCell, err := cell.FromBOC(dataBoc)

		err = tlb.LoadFromCell(accountData, dataCell.BeginParse())
		require.Nil(t, err)

		cdata, err := tlb.ToCell(accountData)
		require.Nil(t, err)

		require.Equal(t, hex.EncodeToString(dataCell.Hash()), hex.EncodeToString(cdata.Hash()))

		_, err = accountData.Highload.MarshalJSON()
		require.Nil(t, err)
	})

	t.Run("should deserialize user positions", func(t *testing.T) {
		k := smartaccount.UserPositionKey{
			Market:    address.MustParseAddr("kQBOtIJPw_GfI29wr__nQI9PfuasVjkAGU_F_3GVU2275Dw5").StringRaw(),
			Direction: schemas.DirectionLong,
		}

		pos := smartaccount.UserPositions{
			Values: map[smartaccount.UserPositionKey]*smartaccount.UserPosition{},
		}

		pos.Values[k] = &smartaccount.UserPosition{
			Locked: false,
			Data:   schemas.NewEmptyPosition(0),
		}

		spew.Dump(pos.Values[smartaccount.UserPositionKey{
			Market:    address.MustParseAddr("kQBOtIJPw_GfI29wr__nQI9PfuasVjkAGU_F_3GVU2275Dw5").StringRaw(),
			Direction: schemas.DirectionLong,
		}])
	})
}
