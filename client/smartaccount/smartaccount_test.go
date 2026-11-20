package smartaccount

import (
	"context"
	"crypto/ed25519"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/wallet"
)

type walletAPI struct {
	ton.APIClientWrapped
	outMsgs uint16
}

func (a *walletAPI) WaitForBlock(uint32) ton.APIClientWrapped { return a }

func (a *walletAPI) CurrentMasterchainInfo(context.Context) (*ton.BlockIDExt, error) {
	return &ton.BlockIDExt{}, nil
}

func (a *walletAPI) GetAccount(context.Context, *ton.BlockIDExt, *address.Address) (*tlb.Account, error) {
	return &tlb.Account{}, nil
}

func (a *walletAPI) RunGetMethod(context.Context, *ton.BlockIDExt, *address.Address, string, ...any) (*ton.ExecutionResult, error) {
	return ton.NewExecutionResult([]any{big.NewInt(0)}), nil
}

func (a *walletAPI) SendExternalMessageWaitTransaction(context.Context, *tlb.ExternalMessage) (*tlb.Transaction, *ton.BlockIDExt, []byte, error) {
	return &tlb.Transaction{OutMsgCount: a.outMsgs, Hash: make([]byte, 32)}, &ton.BlockIDExt{}, nil, nil
}

func TestWalletSendThatSentNothingFails(t *testing.T) {
	addr := address.NewAddress(0, 0, make([]byte, 32))
	amount := tlb.MustFromTON("1")
	for _, outMsgs := range []uint16{0, 1} {
		api := &walletAPI{outMsgs: outMsgs}
		w, err := wallet.FromPrivateKey(api, ed25519.NewKeyFromSeed(make([]byte, 32)), wallet.V4R2)
		require.NoError(t, err)
		_, err = NewClient(api, addr).Withdraw(w, addr, &amount)
		require.Equal(t, outMsgs == 0, err != nil)
	}
}
