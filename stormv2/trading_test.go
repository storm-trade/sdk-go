package stormv2

import (
	"context"
	"crypto/ed25519"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/storm-trade/sdk-go/config"
	"github.com/stretchr/testify/require"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/wallet"
	"github.com/xssnick/tonutils-go/tvm/cell"
)

type chain struct {
	ton.APIClientWrapped
	sent    []*tlb.ExternalMessage
	outMsgs *uint16
}

type lite struct{ ton.LiteClient }

func (lite) StickyContext(ctx context.Context) context.Context { return ctx }

func (c *chain) Client() ton.LiteClient                   { return lite{} }
func (c *chain) WaitForBlock(uint32) ton.APIClientWrapped { return c }

func (c *chain) CurrentMasterchainInfo(context.Context) (*ton.BlockIDExt, error) {
	return &ton.BlockIDExt{}, nil
}

func (c *chain) GetAccount(context.Context, *ton.BlockIDExt, *address.Address) (*tlb.Account, error) {
	return &tlb.Account{}, nil
}

func (c *chain) RunGetMethod(_ context.Context, _ *ton.BlockIDExt, addr *address.Address, method string, _ ...any) (*ton.ExecutionResult, error) {
	switch method {
	case "seqno":
		return ton.NewExecutionResult([]any{big.NewInt(0)}), nil
	case "get_vamm_address", "get_position_address":
		return ton.NewExecutionResult([]any{cell.BeginCell().MustStoreAddr(addr).EndCell().MustBeginParse()}), nil
	}
	return nil, errors.New(method)
}

func (c *chain) SendExternalMessageWaitTransaction(_ context.Context, ext *tlb.ExternalMessage) (*tlb.Transaction, *ton.BlockIDExt, []byte, error) {
	c.sent = append(c.sent, ext)
	tx := &tlb.Transaction{OutMsgCount: uint16(ext.Body.RefsNum())}
	if c.outMsgs != nil {
		tx.OutMsgCount = *c.outMsgs
	}
	return tx, &ton.BlockIDExt{}, nil, nil
}

func newTestClient(t *testing.T, network config.Network) (*Client, *chain) {
	api := &chain{}
	w, err := wallet.FromPrivateKey(api, ed25519.NewKeyFromSeed(make([]byte, 32)), wallet.V4R2)
	require.NoError(t, err)
	return NewClient(network, WithWallet(w), WithTONApi(api)), api
}

func TestExecutorFeeOnlyOnMainnetOrders(t *testing.T) {
	addr := address.NewAddress(0, 0, make([]byte, 32))
	market := &config.Market{Name: "BTC/USD", SettlementToken: "TON", VaultAddress: addr, VammAddress: addr}
	size := tlb.MustFromTON("1")
	for _, tc := range []struct {
		network config.Network
		cancel  bool
		msgs    uint
	}{
		{config.Mainnet, false, 2},
		{config.Mainnet, true, 1},
		{config.Testnet, false, 1},
	} {
		c, api := newTestClient(t, tc.network)
		var err error
		if tc.cancel {
			_, err = c.CancelOrder(context.Background(), market, config.Long, orderTypeLimit, 0)
		} else {
			_, err = c.ClosePosition(context.Background(), market, config.Long, &size)
		}
		require.NoError(t, err)
		require.Len(t, api.sent, 1)
		require.Equal(t, tc.msgs, api.sent[0].Body.RefsNum())
		if tc.msgs == 2 {
			var fee tlb.InternalMessage
			require.NoError(t, tlb.LoadFromCell(&fee, api.sent[0].Body.MustPeekRef(1).MustBeginParse()))
			require.True(t, fee.DstAddr.Equals(address.MustParseAddr("UQCefmnNDYymiaYfPrulfl3IVUhKTENdh4xYtu2FIqdfSFTR")))
			require.Equal(t, "15000000", fee.Amount.Nano().String())
		}
	}
}

func TestSendThatSentNothingFails(t *testing.T) {
	addr := address.NewAddress(0, 0, make([]byte, 32))
	market := &config.Market{Name: "BTC/USD", SettlementToken: "TON", VaultAddress: addr, VammAddress: addr}
	size := tlb.MustFromTON("1")
	for _, out := range []uint16{0, 1, 2} {
		c, api := newTestClient(t, config.Mainnet)
		api.outMsgs = &out
		_, err := c.ClosePosition(context.Background(), market, config.Long, &size)
		require.Equal(t, out == 0, err != nil)
	}
}

func TestRetiredVaultsExpiryAndOrderValue(t *testing.T) {
	ctx := context.Background()
	addr := address.NewAddress(0, 0, make([]byte, 32))
	amount := tlb.MustFromTON("1")
	legacy := &config.Market{Name: "BTC/USD", SettlementToken: "jUSDT", VaultAddress: addr, VammAddress: addr}
	c, api := newTestClient(t, config.Mainnet)
	for _, call := range []func() (*TxResult, error){
		func() (*TxResult, error) { return c.PlaceMarketOrder(ctx, legacy, config.Long, &amount, 1) },
		func() (*TxResult, error) { return c.PlaceLimitOrder(ctx, legacy, config.Long, &amount, 1, &amount) },
		func() (*TxResult, error) { return c.ClosePosition(ctx, legacy, config.Long, &amount) },
		func() (*TxResult, error) { return c.PlaceStopLoss(ctx, legacy, config.Long, &amount, &amount) },
		func() (*TxResult, error) { return c.AddMargin(ctx, legacy, config.Long, &amount) },
		func() (*TxResult, error) { return c.RemoveMargin(ctx, legacy, config.Long, &amount) },
	} {
		_, err := call()
		require.ErrorContains(t, err, "retired vault")
	}
	require.Empty(t, api.sent)
	_, err := c.CancelOrder(ctx, legacy, config.Long, orderTypeLimit, 0)
	require.NoError(t, err)
	require.Len(t, api.sent, 1)

	require.InDelta(t, time.Now().Add(120*time.Second).Unix(), int64(defaultExpiration(orderTypeMarket)), 2)

	c.cfg = &config.ConfigData{Assets: []config.Asset{{Name: "TON", Decimals: 9, VaultAddress: addr}}}
	tonMarket := &config.Market{Name: "BTC/USD", SettlementToken: "TON", VaultAddress: addr, VammAddress: addr}
	sixDecimals := tlb.MustFromDecimal("10", 6)
	_, err = c.PlaceMarketOrder(ctx, tonMarket, config.Long, &sixDecimals, 1)
	require.ErrorContains(t, err, "decimal mismatch")
	require.Len(t, api.sent, 1)
}
