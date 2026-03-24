package positionmanager

import (
	"context"
	"fmt"

	"github.com/storm-trade/sdk-go/contracts/positionmanager"
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

func optionalCell(res *ton.ExecutionResult, index uint) *cell.Cell {
	isNil, _ := res.IsNil(index)
	if isNil {
		return nil
	}
	c, _ := res.Cell(index)
	if c != nil && c.BitsSize() == 0 && c.RefsNum() == 0 {
		return nil
	}
	return c
}

func (c *Client) GetPositionManagerData(ctx context.Context) (*positionmanager.PositionManagerData, error) {
	res, err := c.runGet(ctx, "get_position_manager_data")
	if err != nil {
		return nil, err
	}

	traderSlice, err := res.Slice(0)
	if err != nil {
		return nil, err
	}
	traderAddr, err := traderSlice.LoadAddr()
	if err != nil {
		return nil, err
	}

	vaultSlice, err := res.Slice(1)
	if err != nil {
		return nil, err
	}
	vaultAddr, err := vaultSlice.LoadAddr()
	if err != nil {
		return nil, err
	}

	vammSlice, err := res.Slice(2)
	if err != nil {
		return nil, err
	}
	vammAddr, err := vammSlice.LoadAddr()
	if err != nil {
		return nil, err
	}

	return &positionmanager.PositionManagerData{
		TraderAddress: traderAddr,
		VaultAddress:  vaultAddr,
		VammAddress:   vammAddr,
		LongPosition:  optionalCell(res, 3),
		ShortPosition: optionalCell(res, 4),
		Orders:        optionalCell(res, 5),
		ReferralData:  optionalCell(res, 6),
		OrdersBitset:  uint32(res.MustInt(7).Int64()),
	}, nil
}

func (c *Client) GetIsInited(ctx context.Context) (bool, error) {
	res, err := c.runGet(ctx, "get_is_inited")
	if err != nil {
		return false, nil
	}
	v, err := res.Int(0)
	if err != nil {
		return false, nil
	}
	return v.Int64() != 0, nil
}

func (c *Client) GetVersion(ctx context.Context) (uint32, error) {
	res, err := c.runGet(ctx, "get_version")
	if err != nil {
		return 0, err
	}
	v, err := res.Int(0)
	if err != nil {
		return 0, err
	}
	return uint32(v.Int64()), nil
}

func (c *Client) GetContractData(ctx context.Context) (*cell.Cell, error) {
	res, err := c.runGet(ctx, "get_position_manager_contract_data")
	if err != nil {
		return nil, err
	}
	return res.Cell(0)
}
