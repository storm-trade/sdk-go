package smartaccount

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/tvm/cell"
	"os"
	"testing"
)

func TestSharedJSGoVectors(t *testing.T) {
	data, err := os.ReadFile("testdata/v3-intents.json")
	require.NoError(t, err)
	var vectors []struct {
		Type                            int
		QueryID                         uint64
		ReferenceQueryID                bool
		Hash, Boc, Signature, PublicKey string
	}
	require.NoError(t, json.Unmarshal(data, &vectors))
	for _, v := range vectors {
		t.Run(v.Hash[:8], func(t *testing.T) {
			boc, err := base64.StdEncoding.DecodeString(v.Boc)
			require.NoError(t, err)
			c, err := cell.FromBOC(boc)
			require.NoError(t, err)
			var intent UserIntent
			require.NoError(t, tlb.LoadFromCell(&intent, c.MustBeginParse()))
			require.Equal(t, v.QueryID, intent.QueryId.Seqno())
			require.Equal(t, v.Type, int(intent.Intent.Order.GetType()))
			if v.ReferenceQueryID {
				require.NotNil(t, intent.ReferenceQueryId)
				require.Zero(t, intent.ReferenceQueryId.Seqno())
			}
			rebuilt, err := tlb.ToCell(intent)
			require.NoError(t, err)
			require.Equal(t, v.Hash, hex.EncodeToString(rebuilt.Hash()))
			require.Equal(t, v.Signature, base64.StdEncoding.EncodeToString(rebuilt.Sign(ed25519.NewKeyFromSeed(make([]byte, 32)))))
		})
	}
}
