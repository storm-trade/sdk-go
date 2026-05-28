package vault

import (
	"context"
	"fmt"
	"math/big"

	"github.com/storm-trade/sdk-go/contracts/vault"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/tvm/cell"
)

type Client struct {
	api  ton.APIClientWrapped
	addr *address.Address
}

func NewClient(api ton.APIClientWrapped, addr *address.Address) *Client {
	return &Client{api: api, addr: addr}
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

func (c *Client) GetVaultData(ctx context.Context) (*vault.VaultData, error) {
	res, err := c.runGet(ctx, "get_vault_data")
	if err != nil {
		return nil, err
	}
	addrSlice, err := res.Slice(0)
	if err != nil {
		return nil, err
	}
	jettonAddr, err := addrSlice.LoadAddr()
	if err != nil {
		return nil, err
	}
	return &vault.VaultData{
		JettonWalletAddress: jettonAddr,
		Rate:                res.MustInt(1),
		LpTotalSupply:       res.MustInt(2),
		FreeBalance:         res.MustInt(3),
		LockedBalance:       res.MustInt(4),
		BufferBalance:       res.MustInt(5),
		StakersBalance:      res.MustInt(6),
		ExecutorsBalance:    res.MustInt(7),
		V3Paused:            res.MustInt(8).Int64() != 0,
	}, nil
}

func (c *Client) GetBufferData(ctx context.Context) (*vault.BufferData, error) {
	res, err := c.runGet(ctx, "get_buffer_data")
	if err != nil {
		return nil, err
	}
	return &vault.BufferData{
		Balance:   res.MustInt(0),
		Rate:      res.MustInt(1),
		UnderRate: res.MustInt(2),
		OverRate:  res.MustInt(3),
	}, nil
}

func (c *Client) GetPositionAddress(ctx context.Context, trader, vammAddr *address.Address) (*address.Address, error) {
	traderSlice := cell.BeginCell().MustStoreAddr(trader).EndCell().MustBeginParse()
	vammSlice := cell.BeginCell().MustStoreAddr(vammAddr).EndCell().MustBeginParse()
	res, err := c.runGet(ctx, "get_position_address", traderSlice, vammSlice)
	if err != nil {
		return nil, err
	}
	slice, err := res.Slice(0)
	if err != nil {
		return nil, err
	}
	return slice.LoadAddr()
}

func (c *Client) GetVAMMAddress(ctx context.Context, assetIndex int) (*address.Address, error) {
	res, err := c.runGet(ctx, "get_vamm_address", big.NewInt(int64(assetIndex)))
	if err != nil {
		return nil, err
	}
	slice, err := res.Slice(0)
	if err != nil {
		return nil, err
	}
	return slice.LoadAddr()
}

func (c *Client) GetLpMinterAddress(ctx context.Context) (*address.Address, error) {
	res, err := c.runGet(ctx, "get_lp_minter_address")
	if err != nil {
		return nil, err
	}
	slice, err := res.Slice(0)
	if err != nil {
		return nil, err
	}
	return slice.LoadAddr()
}
