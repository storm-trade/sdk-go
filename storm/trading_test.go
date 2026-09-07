package storm

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/storm-trade/sdk-go/config"
	"github.com/storm-trade/sdk-go/contracts/smartaccount"
	"github.com/storm-trade/sdk-go/sequencer"
	"github.com/stretchr/testify/require"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/tvm/cell"
)

func TestExplicitQueryIDIsUsedWithoutFetching(t *testing.T) {
	var queries int
	var received uint64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			queries++
			w.Write([]byte(`{"query_id":1}`))
			return
		}
		var req sequencer.PlaceOrderRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		boc, err := base64.StdEncoding.DecodeString(req.Message)
		require.NoError(t, err)
		c, err := cell.FromBOC(boc)
		require.NoError(t, err)
		slice, err := c.BeginParse()
		require.NoError(t, err)
		var intent smartaccount.UserIntent
		require.NoError(t, tlb.LoadFromCell(&intent, slice))
		received = intent.QueryId.Seqno()
		w.Write([]byte(`{"ok":true,"accepted":true,"intent_hash":"abc"}`))
	}))
	defer server.Close()
	addr := address.NewAddress(0, 0, make([]byte, 32))
	client := NewClient(config.Testnet, WithSigner(ed25519.NewKeyFromSeed(make([]byte, 32))), WithSmartAccount(addr))
	client.seq = sequencer.NewClient(server.URL)
	amount := tlb.MustFromTON("1")
	_, err := client.PlaceMarketOrder(context.Background(), &config.Market{VammAddress: addr}, config.Long, &amount, 2_000_000_000, WithQueryID(1023))
	require.NoError(t, err)
	require.Equal(t, 0, queries)
	require.Equal(t, uint64(1023), received)
}

func TestBuilderAttributionAndUncertainSubmission(t *testing.T) {
	var ids []uint64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			w.Write([]byte(`{"query_id":1022}`))
			return
		}
		var req sequencer.PlaceOrderRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		require.Equal(t, "registered-builder", req.Builder)
		boc, err := base64.StdEncoding.DecodeString(req.Message)
		require.NoError(t, err)
		c, err := cell.FromBOC(boc)
		require.NoError(t, err)
		var intent smartaccount.UserIntent
		require.NoError(t, tlb.LoadFromCell(&intent, c.MustBeginParse()))
		ids = append(ids, intent.QueryId.Seqno())
		require.True(t, c.Verify(ed25519.NewKeyFromSeed(make([]byte, 32)).Public().(ed25519.PublicKey), mustDecode(t, req.Signature)))
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"error":"upstream unavailable"}`))
	}))
	defer server.Close()
	addr := address.NewAddress(0, 0, make([]byte, 32))
	client := NewClient(config.Testnet, WithSigner(ed25519.NewKeyFromSeed(make([]byte, 32))), WithSmartAccount(addr), WithBuilder("registered-builder"), WithSequencer(sequencer.NewClient(server.URL)))
	amount := tlb.MustFromTON("1")
	for i := 0; i < 2; i++ {
		result, err := client.PlaceMarketOrder(context.Background(), &config.Market{VammAddress: addr}, config.Long, &amount, 2_000_000_000, WithStopLoss(&amount), WithTakeProfit(&amount))
		require.Error(t, err)
		require.NotNil(t, result)
		require.Len(t, result.OrderHash, 32)
	}
	require.Equal(t, []uint64{1022, 1025}, ids)
}

func mustDecode(t *testing.T, s string) []byte {
	t.Helper()
	b, err := base64.StdEncoding.DecodeString(s)
	require.NoError(t, err)
	return b
}

func TestConcurrentQueryReservations(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{"query_id":1022}`)) }))
	defer server.Close()
	addr := address.NewAddress(0, 0, make([]byte, 32))
	client := NewClient(config.Testnet, WithSmartAccount(addr), WithSequencer(sequencer.NewClient(server.URL)))
	ids := make(chan uint64, 20)
	errs := make(chan error, 20)
	for i := 0; i < 20; i++ {
		go func() {
			id, err := client.reserveQueryIDs(context.Background(), client.defaults)
			ids <- id
			errs <- err
		}()
	}
	seen := map[uint64]bool{}
	for i := 0; i < 20; i++ {
		id := <-ids
		require.NoError(t, <-errs)
		require.False(t, seen[id])
		seen[id] = true
	}
	require.Len(t, seen, 20)
}

func TestCancelWithResultRetainsHashOnUncertainHTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/order/cancel", r.URL.Path)
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"error":"upstream unavailable"}`))
	}))
	defer server.Close()
	addr := address.NewAddress(0, 0, make([]byte, 32))
	c := NewClient(config.Testnet, WithSmartAccount(addr), WithSigner(ed25519.NewKeyFromSeed(make([]byte, 32))), WithSequencer(sequencer.NewClient(server.URL)))
	result, err := c.CancelOrderWithResult(context.Background(), make([]byte, 32))
	require.Error(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Hash, 32)
}
