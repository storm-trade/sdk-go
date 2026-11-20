package storm

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/storm-trade/sdk-go/config"
	"github.com/storm-trade/sdk-go/contracts/smartaccount"
	"github.com/storm-trade/sdk-go/sequencer"
	stlb "github.com/storm-trade/sdk-go/tlb"
	"github.com/stretchr/testify/require"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/wallet"
	"github.com/xssnick/tonutils-go/tvm/cell"
)

func TestStdBase64BuilderSentInRawForm(t *testing.T) {
	var posted string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var req sequencer.PlaceOrderRequest
			require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
			posted = req.Builder
		}
		w.Write([]byte(`{"query_id":1,"ok":true}`))
	}))
	defer server.Close()
	addr := address.NewAddress(0, 0, make([]byte, 32))
	client := NewClient(config.Testnet, WithSigner(ed25519.NewKeyFromSeed(make([]byte, 32))), WithSmartAccount(addr),
		WithBuilder("EQB8jr1VjJDoyBdx+Ntm4SWCtKIjsuHAOhjK4+h3C4NYEh3N"), WithSequencer(sequencer.NewClient(server.URL)))
	amount := tlb.MustFromTON("1")
	_, err := client.PlaceMarketOrder(context.Background(), &config.Market{VammAddress: addr}, config.Long, &amount, 2_000_000_000)
	require.NoError(t, err)
	require.Equal(t, "0:7c8ebd558c90e8c81771f8db66e12582b4a223b2e1c03a18cae3e8770b835812", posted)
}

func TestLeaseKeptWhenPreviousIntentAppliedBetweenReads(t *testing.T) {
	var ids []uint64
	applied := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost:
			ids = append(ids, postedIntent(t, r).QueryId.Seqno())
			w.Write([]byte(`{"ok":true,"accepted":true}`))
		case strings.HasSuffix(r.URL.Path, "/status"):
			if applied {
				w.Write([]byte(`{"status":"done"}`))
			} else {
				w.Write([]byte(`{"status":"accepted"}`))
			}
		default:
			if applied {
				w.Write([]byte(`{"query_id":1023}`))
			} else {
				w.Write([]byte(`{"query_id":1022}`))
			}
			if len(ids) == 1 {
				applied = true
			}
		}
	}))
	defer server.Close()
	addr := address.NewAddress(0, 0, make([]byte, 32))
	client := NewClient(config.Testnet, WithSigner(ed25519.NewKeyFromSeed(make([]byte, 32))), WithSmartAccount(addr), WithSequencer(sequencer.NewClient(server.URL)))
	amount := tlb.MustFromTON("1")
	for i := 0; i < 2; i++ {
		_, err := client.PlaceLimitOrder(context.Background(), &config.Market{VammAddress: addr}, config.Long, &amount, 2_000_000_000, &amount)
		require.NoError(t, err)
	}
	require.Equal(t, []uint64{1022, 1023}, ids)
}

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
		if strings.HasSuffix(r.URL.Path, "/status") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.Method == http.MethodGet {
			w.Write([]byte(`{"query_id":1022}`))
			return
		}
		var req sequencer.PlaceOrderRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		require.Equal(t, "0:7c8ebd558c90e8c81771f8db66e12582b4a223b2e1c03a18cae3e8770b835812", req.Builder)
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
	client := NewClient(config.Testnet, WithSigner(ed25519.NewKeyFromSeed(make([]byte, 32))), WithSmartAccount(addr), WithBuilder("EQB8jr1VjJDoyBdx-Ntm4SWCtKIjsuHAOhjK4-h3C4NYEh3N"), WithSequencer(sequencer.NewClient(server.URL)))
	amount := tlb.MustFromTON("1")
	for i := 0; i < 2; i++ {
		result, err := client.PlaceMarketOrder(context.Background(), &config.Market{VammAddress: addr}, config.Long, &amount, 2_000_000_000, WithStopLoss(&amount), WithTakeProfit(&amount))
		require.Error(t, err)
		require.NotNil(t, result)
		require.Len(t, result.OrderHash, 32)
	}
	require.Equal(t, []uint64{1022, 1022}, ids)
}

func mustDecode(t *testing.T, s string) []byte {
	t.Helper()
	b, err := base64.StdEncoding.DecodeString(s)
	require.NoError(t, err)
	return b
}

func postedIntent(t *testing.T, r *http.Request) smartaccount.UserIntent {
	var req sequencer.PlaceOrderRequest
	require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
	c, err := cell.FromBOC(mustDecode(t, req.Message))
	require.NoError(t, err)
	var intent smartaccount.UserIntent
	require.NoError(t, tlb.LoadFromCell(&intent, c.MustBeginParse()))
	return intent
}

func TestConcurrentQueryReservations(t *testing.T) {
	var mu sync.Mutex
	seen := map[uint64]bool{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost:
			id := postedIntent(t, r).QueryId.Seqno()
			mu.Lock()
			seen[id] = true
			mu.Unlock()
			w.Write([]byte(`{"ok":true,"accepted":true}`))
		case strings.HasSuffix(r.URL.Path, "/status"):
			w.Write([]byte(`{"status":"accepted"}`))
		default:
			w.Write([]byte(`{"query_id":1022}`))
		}
	}))
	defer server.Close()
	addr := address.NewAddress(0, 0, make([]byte, 32))
	client := NewClient(config.Testnet, WithSigner(ed25519.NewKeyFromSeed(make([]byte, 32))), WithSmartAccount(addr), WithSequencer(sequencer.NewClient(server.URL)))
	amount := tlb.MustFromTON("1")
	errs := make(chan error, 20)
	for i := 0; i < 20; i++ {
		go func() {
			_, err := client.PlaceMarketOrder(context.Background(), &config.Market{VammAddress: addr}, config.Long, &amount, 2_000_000_000)
			errs <- err
		}()
	}
	for i := 0; i < 20; i++ {
		require.NoError(t, <-errs)
	}
	mu.Lock()
	defer mu.Unlock()
	require.Len(t, seen, 20)
}

func TestQueryIDsResyncAfterRejectedOrFailedOrder(t *testing.T) {
	addr := address.NewAddress(0, 0, make([]byte, 32))
	amount := tlb.MustFromTON("1")
	for _, tc := range []struct {
		name    string
		post    int
		status  string
		queryID string
		opts    []Option
		want    []uint64
	}{
		{"rejected", http.StatusServiceUnavailable, "", "1022", nil, []uint64{1022, 1022}},
		{"failed", http.StatusOK, `{"status":"failed"}`, "1022", nil, []uint64{1022, 1022}},
		{"pending", http.StatusOK, `{"status":"accepted"}`, "1022", nil, []uint64{1022, 1023}},
		{"user ceiling", http.StatusOK, "", "6137999", []Option{WithStopLoss(&amount)}, nil},
	} {
		var ids []uint64
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodPost:
				ids = append(ids, postedIntent(t, r).QueryId.Seqno())
				w.WriteHeader(tc.post)
				w.Write([]byte(`{"ok":true}`))
			case strings.HasSuffix(r.URL.Path, "/status") && tc.status == "":
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"error":"intent not found"}`))
			case strings.HasSuffix(r.URL.Path, "/status"):
				w.Write([]byte(tc.status))
			default:
				w.Write([]byte(`{"query_id":` + tc.queryID + `}`))
			}
		}))
		client := NewClient(config.Testnet, WithSigner(ed25519.NewKeyFromSeed(make([]byte, 32))), WithSmartAccount(addr), WithSequencer(sequencer.NewClient(server.URL)))
		for i := 0; i < 2; i++ {
			_, err := client.PlaceMarketOrder(context.Background(), &config.Market{VammAddress: addr}, config.Long, &amount, 2_000_000_000, tc.opts...)
			if tc.want == nil {
				require.ErrorContains(t, err, "range exhausted", tc.name)
			}
		}
		server.Close()
		require.Equal(t, tc.want, ids, tc.name)
	}
}

func TestClosePositionFullClosesShortFromState(t *testing.T) {
	addr := address.NewAddress(0, 0, make([]byte, 32))
	vamm := address.NewAddress(0, 0, bytes.Repeat([]byte{1}, 32))
	var closed *stlb.Order
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost:
			closed = postedIntent(t, r).Intent.Order
			w.Write([]byte(`{"ok":true}`))
		case strings.HasSuffix(r.URL.Path, "/state"):
			fmt.Fprintf(w, `{"positions":{"%s_short":{"locked":false,"data":{"size":-2673928924942,"direction":1}}}}`, vamm.StringRaw())
		default:
			w.Write([]byte(`{"query_id":1}`))
		}
	}))
	defer server.Close()
	client := NewClient(config.Testnet, WithSigner(ed25519.NewKeyFromSeed(make([]byte, 32))), WithSmartAccount(addr), WithSequencer(sequencer.NewClient(server.URL)))
	_, err := client.ClosePositionFull(context.Background(), &config.Market{VammAddress: vamm}, config.Short)
	require.NoError(t, err)
	require.Equal(t, stlb.TakeOrderType, closed.GetType())
	require.Equal(t, stlb.DirectionShort, closed.GetDirection())
	require.Equal(t, "2673928924942", closed.Value.AsStopOrder().Amount.Nano().String())
}

func TestWorstPriceIsTheMarketOrderLimitPrice(t *testing.T) {
	addr := address.NewAddress(0, 0, make([]byte, 32))
	amount := tlb.MustFromTON("1")
	worst := tlb.MustFromTON("65000")
	for _, tc := range []struct {
		opts []Option
		want string
	}{
		{nil, "0"},
		{[]Option{WithWorstPrice(&worst)}, "65000000000000"},
	} {
		var limit string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				limit = postedIntent(t, r).Intent.Order.Value.AsLimitOrder().LimitPrice.Nano().String()
			}
			w.Write([]byte(`{"query_id":1,"ok":true}`))
		}))
		client := NewClient(config.Testnet, WithSigner(ed25519.NewKeyFromSeed(make([]byte, 32))), WithSmartAccount(addr), WithSequencer(sequencer.NewClient(server.URL)))
		_, err := client.PlaceMarketOrder(context.Background(), &config.Market{VammAddress: addr}, config.Long, &amount, 2_000_000_000, tc.opts...)
		server.Close()
		require.NoError(t, err)
		require.Equal(t, tc.want, limit)
	}
}

type chain struct {
	ton.APIClientWrapped
	results map[string]func() []any
	reads   map[string]int
}

type lite struct{ ton.LiteClient }

func (lite) StickyContext(ctx context.Context) context.Context { return ctx }

func (c *chain) Client() ton.LiteClient { return lite{} }

func (c *chain) CurrentMasterchainInfo(context.Context) (*ton.BlockIDExt, error) {
	return &ton.BlockIDExt{}, nil
}

func (c *chain) RunGetMethod(_ context.Context, _ *ton.BlockIDExt, _ *address.Address, method string, _ ...any) (*ton.ExecutionResult, error) {
	c.reads[method]++
	if result, ok := c.results[method]; ok {
		return ton.NewExecutionResult(result()), nil
	}
	return nil, ton.ContractExecError{Code: ton.ErrCodeContractNotInitialized}
}

func TestWalletFlowsRefuseBeforeSending(t *testing.T) {
	ctx := context.Background()
	addr := address.NewAddress(0, 0, make([]byte, 32))
	addrSlice := func() *cell.Slice { return cell.BeginCell().MustStoreAddr(addr).EndCell().MustBeginParse() }
	sdkFactory := address.MustParseAddr(config.Networks[config.Testnet].FactoryAddress)
	key := ed25519.NewKeyFromSeed(make([]byte, 32))
	var zeroKey ed25519.PrivateKey
	for i := uint32(0); zeroKey == nil; i++ {
		seed := make([]byte, 32)
		binary.BigEndian.PutUint32(seed, i)
		if k := ed25519.NewKeyFromSeed(seed); k.Public().(ed25519.PublicKey)[0] == 0 {
			zeroKey = k
		}
	}
	newClient := func(factory *address.Address) (*Client, *chain) {
		whitelist := cell.NewDict(4)
		require.NoError(t, whitelist.SetIntKey(big.NewInt(6), cell.BeginCell().MustStoreAddr(factory).EndCell()))
		api := &chain{reads: map[string]int{}, results: map[string]func() []any{
			"get_vault_whitelisted_addresses": func() []any { return []any{whitelist.AsCell()} },
			"get_nft_data": func() []any {
				return []any{big.NewInt(-1), big.NewInt(0), addrSlice(), addrSlice(), cell.BeginCell().EndCell()}
			},
			"get_vault_data": func() []any {
				return []any{addrSlice(), big.NewInt(1), big.NewInt(2), big.NewInt(3), big.NewInt(4), big.NewInt(5), big.NewInt(6), big.NewInt(7)}
			},
		}}
		c := NewClient(config.Testnet, WithTONApi(api), WithWallet(&wallet.Wallet{}), WithSmartAccount(addr), WithSigner(key))
		c.cfg.data = &config.ConfigData{Assets: []config.Asset{
			{Name: "TON", Decimals: 9, VaultAddress: addr},
			{Name: "USDT", Decimals: 6, VaultAddress: addr, JettonMaster: addr},
			{Name: "jUSDT", Decimals: 6, VaultAddress: addr, JettonMaster: addr},
		}}
		return c, api
	}
	amount := tlb.MustFromTON("1")
	belowTON := tlb.MustFromDecimal("0.000999999", 9)
	belowUSDT := tlb.MustFromDecimal("0.099999", 6)
	for _, tc := range []struct {
		name    string
		factory *address.Address
		call    func(c *Client) error
		want    string
	}{
		{"init on initialized account", sdkFactory, func(c *Client) error { return c.Deposit(ctx, "TON", &amount, WithInit()) }, "already initialized"},
		{"key on initialized account", sdkFactory, func(c *Client) error {
			return c.Deposit(ctx, "TON", &amount, WithPublicKey(key.Public().(ed25519.PublicKey)))
		}, "AddPublicKey"},
		{"vault of another factory", addr, func(c *Client) error { return c.Withdraw(ctx, "TON", &amount) }, "update the SDK"},
		{"retired vault deposit", sdkFactory, func(c *Client) error { return c.Deposit(ctx, "jUSDT", &amount) }, "retired vault"},
		{"retired vault withdraw", sdkFactory, func(c *Client) error { return c.Withdraw(ctx, "jUSDT", &amount) }, "retired vault"},
		{"zero byte key deposit", sdkFactory, func(c *Client) error {
			return c.Deposit(ctx, "TON", &amount, WithPublicKey(zeroKey.Public().(ed25519.PublicKey)))
		}, "0x00"},
		{"zero byte key add", sdkFactory, func(c *Client) error { return c.AddPublicKey(ctx, WithSigner(zeroKey)) }, "0x00"},
		{"gasless TON below minimum", sdkFactory, func(c *Client) error { return c.GaslessDeposit(ctx, "TON", &belowTON) }, "minimum"},
		{"gasless USDT below minimum", sdkFactory, func(c *Client) error { return c.GaslessDeposit(ctx, "USDT", &belowUSDT) }, "minimum"},
	} {
		c, _ := newClient(tc.factory)
		require.ErrorContains(t, tc.call(c), tc.want, tc.name)
	}

	c, api := newClient(sdkFactory)
	for i := 0; i < 2; i++ {
		require.ErrorContains(t, c.Deposit(ctx, "TON", &amount, WithInit()), "already initialized")
	}
	require.Equal(t, 1, api.reads["get_vault_whitelisted_addresses"])
	data, err := c.GetVaultData(ctx, "jUSDT")
	require.NoError(t, err)
	require.False(t, data.V3Paused)
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
