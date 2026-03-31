package storm

import (
	"context"
	"fmt"
	"math/big"

	vammclient "github.com/storm-trade/sdk-go/client/vamm"
	"github.com/storm-trade/sdk-go/config"
	"github.com/storm-trade/sdk-go/contracts/vamm"
	"github.com/xssnick/tonutils-go/tvm/cell"
)

func (c *Client) vammClient(market *config.Market) (*vammclient.Client, error) {
	if c.defaults.tonAPI == nil {
		return nil, fmt.Errorf("TON API required: use WithTONApi()")
	}
	return vammclient.NewClient(c.defaults.tonAPI, market.VammAddress), nil
}

func (c *Client) GetSpotPrice(ctx context.Context, market *config.Market) (*big.Int, error) {
	vc, err := c.vammClient(market)
	if err != nil {
		return nil, err
	}
	return vc.GetSpotPrice(ctx)
}

func (c *Client) GetTerminalAmmPrice(ctx context.Context, market *config.Market) (*big.Int, error) {
	vc, err := c.vammClient(market)
	if err != nil {
		return nil, err
	}
	return vc.GetTerminalAmmPrice(ctx)
}

func (c *Client) GetAmmState(ctx context.Context, market *config.Market) (*vamm.AmmState, error) {
	vc, err := c.vammClient(market)
	if err != nil {
		return nil, err
	}
	return vc.GetAmmState(ctx)
}

func (c *Client) GetAmmStatus(ctx context.Context, market *config.Market) (*vamm.AmmStatus, error) {
	vc, err := c.vammClient(market)
	if err != nil {
		return nil, err
	}
	return vc.GetAmmStatus(ctx)
}

func (c *Client) GetExchangeSettings(ctx context.Context, market *config.Market) (*vamm.ExchangeSettings, error) {
	vc, err := c.vammClient(market)
	if err != nil {
		return nil, err
	}
	return vc.GetExchangeSettings(ctx)
}

func (c *Client) GetOracleData(ctx context.Context, market *config.Market) (*vamm.OracleData, error) {
	vc, err := c.vammClient(market)
	if err != nil {
		return nil, err
	}
	return vc.GetOracleData(ctx)
}

func (c *Client) GetFunding(ctx context.Context, market *config.Market, price, settlementPrice *big.Int) (*vamm.FundingData, error) {
	vc, err := c.vammClient(market)
	if err != nil {
		return nil, err
	}
	return vc.GetFunding(ctx, price, settlementPrice)
}

func (c *Client) GetPremium(ctx context.Context, market *config.Market, price *big.Int) (*big.Int, error) {
	vc, err := c.vammClient(market)
	if err != nil {
		return nil, err
	}
	return vc.GetPremium(ctx, price)
}

func (c *Client) GetDayTradingData(ctx context.Context, market *config.Market) (*vamm.DayTradingData, error) {
	vc, err := c.vammClient(market)
	if err != nil {
		return nil, err
	}
	return vc.GetDayTradingData(ctx)
}

func (c *Client) GetRemainMargin(ctx context.Context, market *config.Market, oraclePrice *big.Int, positionCell *cell.Cell, settlementPrice *big.Int) (*vamm.MarginData, error) {
	vc, err := c.vammClient(market)
	if err != nil {
		return nil, err
	}
	return vc.GetRemainMargin(ctx, oraclePrice, positionCell, settlementPrice)
}
