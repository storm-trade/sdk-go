package tlb_test

import (
	"fmt"
	"testing"

	"github.com/cristalhq/base64"
	"github.com/stretchr/testify/require"
	"github.com/xssnick/tonutils-go/tvm/cell"
)

func Test_UnpackPriceRef(t *testing.T) {
	c, err := base64.StdEncoding.DecodeString("te6cckEBAQEAGwAAMmAoWu03bAQu6KrAZyErAQAGZyDKZGcg0Ww04me7")
	require.Nil(t, err)

	priceRef, err := cell.FromBOC(c)
	require.Nil(t, err)

	slice := priceRef.MustBeginParse()

	price, err := slice.LoadCoins()
	require.Nil(t, err)

	spread, err := slice.LoadCoins()
	require.Nil(t, err)

	timestamp, err := slice.LoadUInt(32)
	require.Nil(t, err)

	asset_index, err := slice.LoadUInt(16)
	require.Nil(t, err)

	pause_at, err := slice.LoadUInt(32)
	require.Nil(t, err)

	unpause_at, err := slice.LoadUInt(32)
	require.Nil(t, err)

	fmt.Println(price, spread, timestamp, asset_index, pause_at, unpause_at)
}
