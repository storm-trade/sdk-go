package vamm

import (
	"context"
	"fmt"
	"math/big"

	"github.com/storm-trade/sdk-go/contracts/vamm"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
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

func (c *Client) GetSpotPrice(ctx context.Context) (*big.Int, error) {
	res, err := c.runGet(ctx, "get_spot_price")
	if err != nil {
		return nil, err
	}
	return res.Int(0)
}

func (c *Client) GetTerminalAmmPrice(ctx context.Context) (*big.Int, error) {
	res, err := c.runGet(ctx, "get_terminal_amm_price")
	if err != nil {
		return nil, err
	}
	return res.Int(0)
}

func (c *Client) GetAmmState(ctx context.Context) (*vamm.AmmState, error) {
	res, err := c.runGet(ctx, "get_amm_state")
	if err != nil {
		return nil, err
	}
	return &vamm.AmmState{
		QuoteAssetReserve:                    coinFromBigInt(res.MustInt(0)),
		BaseAssetReserve:                     coinFromBigInt(res.MustInt(1)),
		QuoteAssetWeight:                     res.MustInt(2).Uint64(),
		TotalLongPositionSize:                coinFromBigInt(res.MustInt(3)),
		TotalShortPositionSize:               coinFromBigInt(res.MustInt(4)),
		OpenInterestLong:                     coinFromBigInt(res.MustInt(5)),
		OpenInterestShort:                    coinFromBigInt(res.MustInt(6)),
		LatestLongCumulativePremiumFraction:  res.MustInt(7).Int64(),
		LatestShortCumulativePremiumFraction: res.MustInt(8).Int64(),
		NextFundingBlockTimestamp:            uint32(res.MustInt(9).Int64()),
	}, nil
}

func (c *Client) GetAmmStatus(ctx context.Context) (*vamm.AmmStatus, error) {
	res, err := c.runGet(ctx, "get_amm_status")
	if err != nil {
		return nil, err
	}
	return &vamm.AmmStatus{
		CloseOnly: res.MustInt(0).Int64() != 0,
		Paused:    res.MustInt(1).Int64() != 0,
		Stopped:   res.MustInt(2).Int64() != 0,
		PauseAt:   uint32(res.MustInt(3).Int64()),
		UnpauseAt: uint32(res.MustInt(4).Int64()),
	}, nil
}

func (c *Client) GetExchangeSettings(ctx context.Context) (*vamm.ExchangeSettings, error) {
	res, err := c.runGet(ctx, "get_exchange_settings")
	if err != nil {
		return nil, err
	}

	fundRes, err := c.runGet(ctx, "get_funding_settings")
	if err != nil {
		return nil, err
	}

	return &vamm.ExchangeSettings{
		CommonExchangeSettings: &vamm.CommonExchangeSettings{
			Fee:                           uint32(res.MustInt(0).Int64()),
			RolloverFee:                   uint32(res.MustInt(1).Int64()),
			FundingPeriod:                 uint32(res.MustInt(2).Int64()),
			InitMarginRatio:               uint32(res.MustInt(3).Int64()),
			MaintenanceMarginRatio:        uint32(res.MustInt(4).Int64()),
			LiquidationFeeRatio:           uint32(res.MustInt(5).Int64()),
			PartialLiquidationRatio:       uint32(res.MustInt(6).Int64()),
			SpreadLimit:                   uint32(res.MustInt(7).Int64()),
			MaxPriceImpact:                uint32(res.MustInt(8).Int64()),
			MaxPriceSpread:                uint32(res.MustInt(9).Int64()),
			MaxOpenNotional:               coinFromBigInt(res.MustInt(10)),
			FeeToStakersPercent:           uint32(res.MustInt(11).Int64()),
			FundingMode:                   uint8(res.MustInt(12).Int64()),
			MinPartialLiquidationNotional: coinFromBigInt(res.MustInt(13)),
			MinLeverage:                   uint32(res.MustInt(14).Int64()),
		},
		FundingSettings: &vamm.FundingSettings{
			LowFundingFnA:   fundRes.MustInt(0).Int64(),
			LowFundingFnB:   fundRes.MustInt(1).Int64(),
			HighFundingFnA:  fundRes.MustInt(2).Int64(),
			HighFundingFnB:  fundRes.MustInt(3).Int64(),
			InflectionPoint: fundRes.MustInt(4).Int64(),
		},
	}, nil
}

func (c *Client) GetOracleData(ctx context.Context) (*vamm.OracleData, error) {
	res, err := c.runGet(ctx, "get_oracle_data")
	if err != nil {
		return nil, err
	}
	return &vamm.OracleData{
		OracleLastPrice:      coinFromBigInt(res.MustInt(0)),
		OracleLastSpread:     coinFromBigInt(res.MustInt(1)),
		OracleLastTimestamp:  uint32(res.MustInt(2).Int64()),
		OracleMaxDeviation:   coinFromBigInt(res.MustInt(3)),
		OracleValidityPeriod: uint32(res.MustInt(4).Int64()),
		OracleAssetID:        uint16(res.MustInt(5).Int64()),
	}, nil
}

func (c *Client) GetFunding(ctx context.Context, price, settlementPrice *big.Int) (*vamm.FundingData, error) {
	res, err := c.runGet(ctx, "get_funding", price, settlementPrice)
	if err != nil {
		return nil, err
	}
	return &vamm.FundingData{
		LongFunding:        res.MustInt(0),
		ShortFunding:       res.MustInt(1),
		PremiumToVault:     res.MustInt(2),
		SyncExchangeAmount: res.MustInt(3),
		SpotPrice:          res.MustInt(4),
	}, nil
}

func (c *Client) GetPremium(ctx context.Context, price *big.Int) (*big.Int, error) {
	res, err := c.runGet(ctx, "get_premium", price)
	if err != nil {
		return nil, err
	}
	return res.Int(0)
}

func (c *Client) GetDayTradingData(ctx context.Context) (*vamm.DayTradingData, error) {
	res, err := c.runGet(ctx, "get_day_trading_data")
	if err != nil {
		return nil, err
	}
	return &vamm.DayTradingData{
		Active:      res.MustInt(0).Int64() != 0,
		MaxLeverage: res.MustInt(1),
	}, nil
}

func (c *Client) GetRemainMargin(ctx context.Context, oraclePrice *big.Int, positionCell *cell.Cell, settlementPrice *big.Int) (*vamm.MarginData, error) {
	res, err := c.runGet(ctx, "get_remain_margin_with_funding_payment", oraclePrice, positionCell, settlementPrice)
	if err != nil {
		return nil, err
	}
	return &vamm.MarginData{
		RemainMargin:     res.MustInt(0),
		FundingPayment:   res.MustInt(1),
		MarginRatio:      res.MustInt(2),
		UnrealizedPnl:    res.MustInt(3),
		BadDebt:          res.MustInt(4),
		PositionNotional: res.MustInt(5),
		RolloverFee:      res.MustInt(6),
		OraclePrice:      res.MustInt(7),
		SpotPrice:        res.MustInt(8),
	}, nil
}

func (c *Client) GetPositionManagerAddress(ctx context.Context, trader *address.Address) (*address.Address, error) {
	traderCell := cell.BeginCell().MustStoreAddr(trader).EndCell()
	res, err := c.runGet(ctx, "get_position_manager_address", traderCell.MustBeginParse())
	if err != nil {
		return nil, err
	}
	slice, err := res.Slice(0)
	if err != nil {
		return nil, err
	}
	return slice.LoadAddr()
}

func coinFromBigInt(v *big.Int) *tlb.Coins {
	c := tlb.FromNanoTON(v)
	return &c
}
