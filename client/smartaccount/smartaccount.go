package smartaccount

import (
	"context"
	"github.com/pkg/errors"
	"github.com/storm-trade/sdk-go/contracts/smartaccount"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/jetton"
	"github.com/xssnick/tonutils-go/ton/wallet"
	"github.com/xssnick/tonutils-go/tvm/cell"
	"time"
)

var (
	MinGas = tlb.MustFromTON("0.35")
)

type Client struct {
	addr *address.Address
	API  ton.APIClientWrapped
}

func NewClient(api ton.APIClientWrapped, addr *address.Address) *Client {
	return &Client{addr: addr, API: api}
}

func (c *Client) GetAddress() *address.Address {
	return c.addr
}

func (c *Client) GetStorageData(ctx context.Context) (*smartaccount.AccountData, error) {
	ctx = c.API.Client().StickyContext(ctx)

	ext, err := c.API.CurrentMasterchainInfo(ctx)
	if err != nil {
		return nil, err
	}

	account, err := c.API.GetAccount(ctx, ext, c.addr)
	if err != nil {
		return nil, err
	}

	if !account.IsActive {
		return nil, errors.New("account not active")
	}

	data := new(smartaccount.AccountData)
	if err := tlb.LoadFromCell(data, account.State.StateInit.Data.BeginParse()); err != nil {
		return nil, err
	}

	return data, nil
}

func (c *Client) DepositNative(from *wallet.Wallet, owner, vault *address.Address, amount *tlb.Coins, init bool, publicKeys ...smartaccount.PublicKey) (*tlb.Transaction, error) {
	payload, err := c.BuildDepositNativePayload(owner, amount, init, publicKeys...)
	if err != nil {
		return nil, err
	}

	tonAmount, err := amount.Add(&MinGas)
	if err != nil {
		return nil, err
	}

	msg := wallet.SimpleMessage(vault, *tonAmount, payload)
	tx, _, err := from.SendWaitTransaction(context.Background(), msg)

	return tx, err
}

func (c *Client) DepositJetton(from *wallet.Wallet, owner, vault, jettonMaster *address.Address, amount *tlb.Coins, init bool, publicKeys ...smartaccount.PublicKey) (*tlb.Transaction, error) {
	jettonMasterClient := jetton.NewJettonMasterClient(c.API, jettonMaster)
	jettonWallet, err := jettonMasterClient.GetJettonWallet(context.Background(), from.WalletAddress())
	if err != nil {
		return nil, err
	}

	payload, err := c.BuildDepositJettonPayload(owner, init, publicKeys...)
	if err != nil {
		return nil, err
	}

	transferPayload, err := jetton.BuildTransferPayload(vault, from.WalletAddress(), *amount, MinGas, payload, nil)
	if err != nil {
		return nil, err
	}

	jettonTransfer := tlb.MustFromTON("0.05")
	msg := wallet.SimpleMessage(jettonWallet.Address(), *MinGas.MustAdd(&jettonTransfer), transferPayload)
	tx, _, err := from.SendWaitTransaction(context.Background(), msg)

	return tx, err
}

func (c *Client) AddPublicKey(from *wallet.Wallet, publicKey smartaccount.PublicKey) (*tlb.Transaction, error) {
	payload := smartaccount.AddPublicKey{
		QueryID:   uint64(time.Now().Unix()),
		PublicKey: publicKey,
	}

	payloadCell, err := tlb.ToCell(payload)
	if err != nil {
		return nil, err
	}

	msg := wallet.SimpleMessage(c.addr, MinGas, payloadCell)
	tx, _, err := from.SendWaitTransaction(context.Background(), msg)

	return tx, err
}

func (c *Client) RemovePublicKey(from *wallet.Wallet, publicKey smartaccount.PublicKey) (*tlb.Transaction, error) {
	payload := smartaccount.RemovePublicKey{
		QueryID:   uint64(time.Now().Unix()),
		PublicKey: publicKey,
	}

	payloadCell, err := tlb.ToCell(payload)
	if err != nil {
		return nil, err
	}

	msg := wallet.SimpleMessage(c.addr, MinGas, payloadCell)
	tx, _, err := from.SendWaitTransaction(context.Background(), msg)

	return tx, err
}

func (c *Client) RemoveAllExceptCurrentPublicKey(from *wallet.Wallet, publicKey smartaccount.PublicKey) (*tlb.Transaction, error) {
	payload := smartaccount.RemoveAllExceptCurrentPublicKey{
		QueryID:   uint64(time.Now().Unix()),
		PublicKey: publicKey,
	}

	payloadCell, err := tlb.ToCell(payload)
	if err != nil {
		return nil, err
	}

	msg := wallet.SimpleMessage(c.addr, MinGas, payloadCell)
	tx, _, err := from.SendWaitTransaction(context.Background(), msg)

	return tx, err
}

func (c *Client) Withdraw(from *wallet.Wallet, vault *address.Address, amount *tlb.Coins) (*tlb.Transaction, error) {
	payload := smartaccount.Withdraw{
		QueryID:      uint64(time.Now().Unix()),
		VaultAddress: vault,
		Amount:       amount,
	}

	payloadCell, err := tlb.ToCell(payload)
	if err != nil {
		return nil, err
	}

	msg := wallet.SimpleMessage(c.addr, tlb.MustFromTON("0.1"), payloadCell)
	tx, _, err := from.SendWaitTransaction(context.Background(), msg)

	return tx, err
}

func (c *Client) BuildDepositJettonPayload(owner *address.Address, init bool, publicKeys ...smartaccount.PublicKey) (*cell.Cell, error) {
	queryId := uint64(time.Now().Unix())

	v := &smartaccount.DepositJettonPayload{
		QueryID:         queryId,
		ReceiverAddress: owner,
		Init:            init,
		KeyInit:         nil,
	}

	if len(publicKeys) > 0 {
		pks := smartaccount.UserPublicKeys{
			Values: publicKeys,
		}

		dict, err := pks.ToDictionary()
		if err != nil {
			return nil, err
		}

		v.KeyInit = dict
	}

	return tlb.ToCell(v)
}

func (c *Client) BuildDepositNativePayload(owner *address.Address, amount *tlb.Coins, init bool, publicKeys ...smartaccount.PublicKey) (*cell.Cell, error) {
	queryId := uint64(time.Now().Unix())

	v := &smartaccount.DepositNativePayload{
		QueryID:         queryId,
		Amount:          amount,
		ReceiverAddress: owner,
		Init:            init,
		KeyInit:         nil,
	}

	if len(publicKeys) > 0 {
		pks := smartaccount.UserPublicKeys{
			Values: publicKeys,
		}

		dict, err := pks.ToDictionary()
		if err != nil {
			return nil, err
		}

		v.KeyInit = dict
	}

	return tlb.ToCell(v)
}
