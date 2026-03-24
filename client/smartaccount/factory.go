package smartaccount

import (
	"context"
	"fmt"
	"math/big"

	"github.com/storm-trade/sdk-go/contracts/smartaccount"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/ton"
)

type Factory struct {
	api  ton.APIClientWrapped
	addr *address.Address
}

func NewFactory(api ton.APIClientWrapped, addr *address.Address) *Factory {
	return &Factory{api: api, addr: addr}
}

func (f *Factory) runGet(ctx context.Context, method string, params ...any) (*ton.ExecutionResult, error) {
	ctx = f.api.Client().StickyContext(ctx)
	block, err := f.api.CurrentMasterchainInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s: get block: %w", method, err)
	}
	res, err := f.api.RunGetMethod(ctx, block, f.addr, method, params...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", method, err)
	}
	return res, nil
}

func (f *Factory) GetSmartAccountAddress(ctx context.Context, owner *address.Address) (*address.Address, error) {
	hash := big.NewInt(0).SetBytes(owner.Data())
	res, err := f.runGet(ctx, "get_nft_address_by_index", hash)
	if err != nil {
		return nil, err
	}
	addrSlice, err := res.Slice(0)
	if err != nil {
		return nil, err
	}
	return addrSlice.LoadAddr()
}

func (f *Factory) GetSmartAccount(ctx context.Context, owner *address.Address) (*Client, error) {
	addr, err := f.GetSmartAccountAddress(ctx, owner)
	if err != nil {
		return nil, err
	}
	return NewClient(f.api, addr), nil
}

func (f *Factory) GetFactoryData(ctx context.Context) (*smartaccount.FactoryData, error) {
	res, err := f.runGet(ctx, "get_factory_data")
	if err != nil {
		return nil, err
	}
	addrSlice, err := res.Slice(0)
	if err != nil {
		return nil, err
	}
	adminAddr, err := addrSlice.LoadAddr()
	if err != nil {
		return nil, err
	}
	return &smartaccount.FactoryData{
		AdminAddress:    adminAddr,
		HighloadTimeout: uint32(res.MustInt(1).Int64()),
		HotPublicKey:    res.MustInt(2),
		ColdPublicKey:   res.MustInt(3),
		Content:         res.MustCell(4),
		Version:         uint32(res.MustInt(5).Int64()),
		Code:            res.MustCell(6),
	}, nil
}

func (f *Factory) GetMinFees(ctx context.Context) (*smartaccount.MinFees, error) {
	res, err := f.runGet(ctx, "get_min_fees")
	if err != nil {
		return nil, err
	}
	return &smartaccount.MinFees{
		DepositMinFeeNative:           res.MustInt(0),
		DepositMinFeeJetton:           res.MustInt(1),
		DepositWithDeployMinFeeNative: res.MustInt(2),
		DepositWithDeployMinFeeJetton: res.MustInt(3),
		WithdrawMinFee:                res.MustInt(4),
		StorageFee:                    res.MustInt(5),
	}, nil
}
