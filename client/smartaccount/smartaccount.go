package smartaccount

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/storm-trade/sdk-go/contracts/smartaccount"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/jetton"
	"github.com/xssnick/tonutils-go/ton/wallet"
	"github.com/xssnick/tonutils-go/tvm/cell"
)

var (
	GasDepositNative     = tlb.MustFromTON("0.1")
	GasDepositNativeInit = tlb.MustFromTON("0.45")
	GasDepositJetton     = tlb.MustFromTON("0.15")
	GasDepositJettonInit = tlb.MustFromTON("0.45")
	GasWithdraw          = tlb.MustFromTON("0.08")
	GasKeyOperation      = tlb.MustFromTON("0.05")
	GasJettonTransfer    = tlb.MustFromTON("0.05")
)

type Client struct {
	addr *address.Address
	api  ton.APIClientWrapped
}

func NewClient(api ton.APIClientWrapped, addr *address.Address) *Client {
	return &Client{addr: addr, api: api}
}

func (c *Client) GetAddress() *address.Address {
	return c.addr
}

func (c *Client) GetStorageData(ctx context.Context) (*smartaccount.AccountData, error) {
	ctx = c.api.Client().StickyContext(ctx)
	ext, err := c.api.CurrentMasterchainInfo(ctx)
	if err != nil {
		return nil, err
	}
	account, err := c.api.GetAccount(ctx, ext, c.addr)
	if err != nil {
		return nil, err
	}
	if !account.IsActive {
		return nil, fmt.Errorf("account not active")
	}
	data := new(smartaccount.AccountData)
	if err := tlb.LoadFromCell(data, account.State.StateInit.Data.MustBeginParse()); err != nil {
		return nil, err
	}
	return data, nil
}

func (c *Client) DepositNative(from *wallet.Wallet, owner, vault *address.Address, amount *tlb.Coins, init bool, publicKeys ...smartaccount.PublicKey) (*tlb.Transaction, error) {
	payload, err := c.BuildDepositNativePayload(owner, amount, init, publicKeys...)
	if err != nil {
		return nil, err
	}
	gas := GasDepositNative
	if init {
		gas = GasDepositNativeInit
	}
	tonAmount, err := amount.Add(gas)
	if err != nil {
		return nil, err
	}
	msg := wallet.SimpleMessage(vault, tonAmount, payload)
	tx, _, err := from.SendWaitTransaction(context.Background(), msg)
	return tx, err
}

func (c *Client) DepositJetton(from *wallet.Wallet, owner, vault, jettonMaster *address.Address, amount *tlb.Coins, init bool, publicKeys ...smartaccount.PublicKey) (*tlb.Transaction, error) {
	jettonMasterClient := jetton.NewJettonMasterClient(c.api, jettonMaster)
	jettonWallet, err := jettonMasterClient.GetJettonWallet(context.Background(), from.WalletAddress())
	if err != nil {
		return nil, err
	}
	payload, err := c.BuildDepositJettonPayload(owner, init, publicKeys...)
	if err != nil {
		return nil, err
	}
	gas := GasDepositJetton
	if init {
		gas = GasDepositJettonInit
	}
	transferPayload, err := jetton.BuildTransferPayload(vault, from.WalletAddress(), *amount, gas, payload, nil)
	if err != nil {
		return nil, err
	}
	msg := wallet.SimpleMessage(jettonWallet.Address(), gas.MustAdd(GasJettonTransfer), transferPayload)
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
	msg := wallet.SimpleMessage(c.addr, GasKeyOperation, payloadCell)
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
	msg := wallet.SimpleMessage(c.addr, GasKeyOperation, payloadCell)
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
	msg := wallet.SimpleMessage(c.addr, GasKeyOperation, payloadCell)
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
	msg := wallet.SimpleMessage(c.addr, GasWithdraw, payloadCell)
	tx, _, err := from.SendWaitTransaction(context.Background(), msg)
	return tx, err
}

func (c *Client) BuildDepositJettonPayload(owner *address.Address, init bool, publicKeys ...smartaccount.PublicKey) (*cell.Cell, error) {
	v := &smartaccount.DepositJettonPayload{
		QueryID:         uint64(time.Now().Unix()),
		ReceiverAddress: owner,
		Init:            init,
	}
	if len(publicKeys) > 0 {
		pks := smartaccount.UserPublicKeys{Values: publicKeys}
		dict, err := pks.ToDictionary()
		if err != nil {
			return nil, err
		}
		v.KeyInit = dict
	}
	return tlb.ToCell(v)
}

func (c *Client) BuildDepositNativePayload(owner *address.Address, amount *tlb.Coins, init bool, publicKeys ...smartaccount.PublicKey) (*cell.Cell, error) {
	v := &smartaccount.DepositNativePayload{
		QueryID:         uint64(time.Now().Unix()),
		Amount:          amount,
		ReceiverAddress: owner,
		Init:            init,
	}
	if len(publicKeys) > 0 {
		pks := smartaccount.UserPublicKeys{Values: publicKeys}
		dict, err := pks.ToDictionary()
		if err != nil {
			return nil, err
		}
		v.KeyInit = dict
	}
	return tlb.ToCell(v)
}

func (c *Client) runGet(ctx context.Context, method string, params ...any) (*ton.ExecutionResult, error) {
	ctx = c.api.Client().StickyContext(ctx)
	block, err := c.api.CurrentMasterchainInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s: get block: %w", method, err)
	}
	res, err := c.api.RunGetMethod(ctx, block, c.addr, method, params...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", method, err)
	}
	return res, nil
}

func (c *Client) GetKeysData(ctx context.Context) (*smartaccount.KeysData, error) {
	res, err := c.runGet(ctx, "get_keys_data")
	if err != nil {
		return nil, err
	}
	var userKeys *cell.Cell
	isNil, _ := res.IsNil(2)
	if !isNil {
		userKeys, _ = res.Cell(2)
	}
	return &smartaccount.KeysData{
		HotPublicKey:  res.MustInt(0),
		ColdPublicKey: res.MustInt(1),
		UserKeys:      userKeys,
		KeysCount:     uint32(res.MustInt(3).Int64()),
	}, nil
}

func (c *Client) GetBalance(ctx context.Context, vaultAddress *address.Address) (*big.Int, error) {
	vaultSlice := cell.BeginCell().MustStoreAddr(vaultAddress).EndCell().MustBeginParse()
	res, err := c.runGet(ctx, "get_vault_balance", vaultSlice)
	if err != nil {
		return nil, err
	}
	return res.Int(0)
}

func (c *Client) GetUserPublicKeys(ctx context.Context) (*cell.Cell, error) {
	res, err := c.runGet(ctx, "get_user_public_keys")
	if err != nil {
		return nil, err
	}
	isNil, _ := res.IsNil(0)
	if isNil {
		return nil, nil
	}
	return res.Cell(0)
}

func (c *Client) GetNftData(ctx context.Context) (*smartaccount.NftData, error) {
	res, err := c.runGet(ctx, "get_nft_data")
	if err != nil {
		return nil, err
	}
	collSlice, err := res.Slice(2)
	if err != nil {
		return nil, err
	}
	collAddr, err := collSlice.LoadAddr()
	if err != nil {
		return nil, err
	}
	ownerSlice, err := res.Slice(3)
	if err != nil {
		return nil, err
	}
	ownerAddr, err := ownerSlice.LoadAddr()
	if err != nil {
		return nil, err
	}
	return &smartaccount.NftData{
		Init:              res.MustInt(0).Int64() != 0,
		Index:             res.MustInt(1),
		CollectionAddress: collAddr,
		OwnerAddress:      ownerAddr,
		Content:           res.MustCell(4),
	}, nil
}

func (c *Client) GetHighloadData(ctx context.Context) (*smartaccount.HighloadData, error) {
	res, err := c.runGet(ctx, "get_highload_data")
	if err != nil {
		return nil, err
	}
	var oldQueries, queries *cell.Cell
	isNil, _ := res.IsNil(0)
	if !isNil {
		oldQueries, _ = res.Cell(0)
	}
	isNil, _ = res.IsNil(1)
	if !isNil {
		queries, _ = res.Cell(1)
	}
	return &smartaccount.HighloadData{
		OldQueries:    oldQueries,
		Queries:       queries,
		LastCleanTime: uint32(res.MustInt(2).Int64()),
		Timeout:       uint32(res.MustInt(3).Int64()),
	}, nil
}

func (c *Client) GetPosition(ctx context.Context, ammAddress *address.Address, direction uint8) (*smartaccount.PositionRecord, error) {
	keyCell := cell.BeginCell().MustStoreAddr(ammAddress).MustStoreUInt(uint64(direction), 1).EndCell()
	res, err := c.runGet(ctx, "get_position", keyCell.MustBeginParse())
	if err != nil {
		return nil, err
	}
	locked := res.MustInt(0)
	isNil, _ := res.IsNil(1)
	if isNil {
		return nil, nil
	}
	posCell, _ := res.Cell(1)
	return &smartaccount.PositionRecord{
		Locked: locked.Int64() != 0,
		Data:   posCell,
	}, nil
}
