package stormv2

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	pmclient "github.com/storm-trade/sdk-go/client/positionmanager"
	vaultclient "github.com/storm-trade/sdk-go/client/vault"
	"github.com/storm-trade/sdk-go/config"
	"github.com/storm-trade/sdk-go/oracle"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton/jetton"
	"github.com/xssnick/tonutils-go/ton/wallet"
	"github.com/xssnick/tonutils-go/tvm/cell"
)

type TxResult struct {
	Hash string
}

func (c *Client) PlaceMarketOrder(ctx context.Context, market *config.Market, dir config.Direction, amount *tlb.Coins, leverage uint64, opts ...Option) (*TxResult, error) {
	return c.placeCreateOrder(ctx, market, dir, amount, leverage, orderTypeMarket, tlb.ZeroCoins, tlb.ZeroCoins, opts)
}

func (c *Client) PlaceLimitOrder(ctx context.Context, market *config.Market, dir config.Direction, amount *tlb.Coins, leverage uint64, limitPrice *tlb.Coins, opts ...Option) (*TxResult, error) {
	return c.placeCreateOrder(ctx, market, dir, amount, leverage, orderTypeLimit, *limitPrice, tlb.ZeroCoins, opts)
}

func (c *Client) PlaceStopLimitOrder(ctx context.Context, market *config.Market, dir config.Direction, amount *tlb.Coins, leverage uint64, limitPrice, stopPrice *tlb.Coins, opts ...Option) (*TxResult, error) {
	return c.placeCreateOrder(ctx, market, dir, amount, leverage, orderTypeLimit, *limitPrice, *stopPrice, opts)
}

func (c *Client) requireWalletAndAPI(o clientOptions) error {
	if o.wallet == nil {
		return fmt.Errorf("wallet required: use WithWallet()")
	}
	if o.tonAPI == nil {
		return fmt.Errorf("TON API required: use WithTONApi()")
	}
	return nil
}

func resolveQueryID(o clientOptions) uint64 {
	if o.queryID != nil {
		return *o.queryID
	}
	return uint64(time.Now().Unix())
}

func defaultExpiration(orderType uint64) uint32 {
	switch orderType {
	case orderTypeMarket:
		return uint32(time.Now().Add(15 * time.Minute).Unix())
	default:
		return uint32(time.Now().Add(60 * 24 * time.Hour).Unix())
	}
}

func (c *Client) placeCreateOrder(ctx context.Context, market *config.Market, dir config.Direction, amount *tlb.Coins, leverage uint64, orderType uint64, limitPrice, stopPrice tlb.Coins, opts []Option) (*TxResult, error) {
	o := c.resolveOptions(opts)
	if err := c.requireWalletAndAPI(o); err != nil {
		return nil, err
	}

	asset, err := c.assetForMarket(market)
	if err != nil {
		return nil, err
	}

	needInit, err := c.needsInitPM(ctx, o, market)
	if err != nil {
		return nil, err
	}

	slTrigger := tlb.ZeroCoins
	if o.stopLoss != nil {
		slTrigger = *o.stopLoss
	}
	tpTrigger := tlb.ZeroCoins
	if o.takeProfit != nil {
		tpTrigger = *o.takeProfit
	}

	expiration := defaultExpiration(orderType)
	if o.expiration != nil {
		expiration = *o.expiration
	}

	params := createOrderParams{
		orderType:        orderType,
		assetID:          uint16(market.AssetIndex),
		direction:        uint64(dir),
		leverage:         leverage,
		expiration:       expiration,
		limitPrice:       limitPrice,
		stopPrice:        stopPrice,
		stopTriggerPrice: slTrigger,
		takeTriggerPrice: tpTrigger,
		gasToAddress:     o.wallet.WalletAddress(),
		initPM:           needInit,
		referralID:       o.referralID,
	}

	var orderMsg *wallet.Message
	if asset.JettonMaster == nil {
		body := buildNativeCreateOrderCell(params, *amount)
		totalValue, _ := amount.Add(&GasCreateOrder)
		orderMsg = wallet.SimpleMessage(market.VaultAddress, *totalValue, body)
	} else {
		jwAddr, err := jetton.NewJettonMasterClient(o.tonAPI, asset.JettonMaster).GetJettonWallet(ctx, o.wallet.WalletAddress())
		if err != nil {
			return nil, fmt.Errorf("get jetton wallet: %w", err)
		}
		orderCell := buildJettonCreateOrderCell(params)
		body := wrapJetton(orderCell, *amount, market.VaultAddress, o.wallet.WalletAddress(), GasForwardOrder, resolveQueryID(o))
		orderMsg = wallet.SimpleMessage(jwAddr.Address(), GasCreateOrder, body)
	}

	return c.sendWithFee(ctx, o.wallet, orderMsg)
}

func (c *Client) ClosePosition(ctx context.Context, market *config.Market, dir config.Direction, size *tlb.Coins, opts ...Option) (*TxResult, error) {
	return c.placeSLTP(ctx, market, dir, size, tlb.ZeroCoins, orderTypeTakeProfit, opts)
}

func (c *Client) PlaceStopLoss(ctx context.Context, market *config.Market, dir config.Direction, size *tlb.Coins, triggerPrice *tlb.Coins, opts ...Option) (*TxResult, error) {
	return c.placeSLTP(ctx, market, dir, size, *triggerPrice, orderTypeStopLoss, opts)
}

func (c *Client) PlaceTakeProfit(ctx context.Context, market *config.Market, dir config.Direction, size *tlb.Coins, triggerPrice *tlb.Coins, opts ...Option) (*TxResult, error) {
	return c.placeSLTP(ctx, market, dir, size, *triggerPrice, orderTypeTakeProfit, opts)
}

func (c *Client) placeSLTP(ctx context.Context, market *config.Market, dir config.Direction, size *tlb.Coins, triggerPrice tlb.Coins, orderType uint64, opts []Option) (*TxResult, error) {
	o := c.resolveOptions(opts)
	if err := c.requireWalletAndAPI(o); err != nil {
		return nil, err
	}

	pmAddr, err := c.getPositionManagerAddress(ctx, o, market)
	if err != nil {
		return nil, err
	}

	body := buildSLTPCell(sltpParams{
		orderType:    orderType,
		direction:    uint64(dir),
		amount:       *size,
		triggerPrice: triggerPrice,
		gasToAddress: o.wallet.WalletAddress(),
	})

	orderMsg := wallet.SimpleMessage(pmAddr, GasCreateOrder, body)
	return c.sendWithFee(ctx, o.wallet, orderMsg)
}

func (c *Client) CancelOrder(ctx context.Context, market *config.Market, dir config.Direction, orderType, orderIndex uint64, opts ...Option) (*TxResult, error) {
	o := c.resolveOptions(opts)
	if err := c.requireWalletAndAPI(o); err != nil {
		return nil, err
	}

	pmAddr, err := c.getPositionManagerAddress(ctx, o, market)
	if err != nil {
		return nil, err
	}

	body := buildCancelCell(orderType, orderIndex, uint64(dir), o.wallet.WalletAddress())
	orderMsg := wallet.SimpleMessage(pmAddr, GasCancelOrder, body)
	return c.sendWithFee(ctx, o.wallet, orderMsg)
}

func (c *Client) AddMargin(ctx context.Context, market *config.Market, dir config.Direction, amount *tlb.Coins, opts ...Option) (*TxResult, error) {
	o := c.resolveOptions(opts)
	if err := c.requireWalletAndAPI(o); err != nil {
		return nil, err
	}

	asset, err := c.assetForMarket(market)
	if err != nil {
		return nil, err
	}

	oraclePayload, err := c.fetchOraclePayload(ctx, market)
	if err != nil {
		return nil, fmt.Errorf("fetch oracle: %w", err)
	}

	var orderMsg *wallet.Message
	if asset.JettonMaster == nil {
		body := buildAddMarginNativeCell(*amount, uint16(market.AssetIndex), uint64(dir), o.wallet.WalletAddress(), oraclePayload)
		totalValue, _ := amount.Add(&GasMargin)
		orderMsg = wallet.SimpleMessage(market.VaultAddress, *totalValue, body)
	} else {
		jwAddr, err := jetton.NewJettonMasterClient(o.tonAPI, asset.JettonMaster).GetJettonWallet(ctx, o.wallet.WalletAddress())
		if err != nil {
			return nil, fmt.Errorf("get jetton wallet: %w", err)
		}
		marginCell := buildAddMarginCell(uint16(market.AssetIndex), uint64(dir), o.wallet.WalletAddress(), oraclePayload)
		body := wrapJetton(marginCell, *amount, market.VaultAddress, o.wallet.WalletAddress(), GasForwardMargin, resolveQueryID(o))
		orderMsg = wallet.SimpleMessage(jwAddr.Address(), GasMargin, body)
	}

	return c.sendWithFee(ctx, o.wallet, orderMsg)
}

func (c *Client) RemoveMargin(ctx context.Context, market *config.Market, dir config.Direction, amount *tlb.Coins, opts ...Option) (*TxResult, error) {
	o := c.resolveOptions(opts)
	if err := c.requireWalletAndAPI(o); err != nil {
		return nil, err
	}

	pmAddr, err := c.getPositionManagerAddress(ctx, o, market)
	if err != nil {
		return nil, err
	}

	oraclePayload, err := c.fetchOraclePayload(ctx, market)
	if err != nil {
		return nil, fmt.Errorf("fetch oracle: %w", err)
	}

	body := buildRemoveMarginCell(uint64(dir), *amount, o.wallet.WalletAddress(), oraclePayload)
	orderMsg := wallet.SimpleMessage(pmAddr, GasMargin, body)
	return c.sendWithFee(ctx, o.wallet, orderMsg)
}

func (c *Client) sendWithFee(ctx context.Context, w *wallet.Wallet, orderMsg *wallet.Message) (*TxResult, error) {
	feeMsg := c.executorFeeMsg()
	tx, _, err := w.SendManyWaitTransaction(ctx, []*wallet.Message{orderMsg, feeMsg})
	if err != nil {
		return nil, err
	}
	return &TxResult{Hash: hex.EncodeToString(tx.Hash)}, nil
}

func (c *Client) getPositionManagerAddress(ctx context.Context, o clientOptions, market *config.Market) (*address.Address, error) {
	vc := vaultclient.NewClient(o.tonAPI, market.VaultAddress)
	vammAddr, err := vc.GetVAMMAddress(ctx, market.AssetIndex)
	if err != nil {
		return nil, fmt.Errorf("get vamm address: %w", err)
	}
	return vc.GetPositionAddress(ctx, o.wallet.WalletAddress(), vammAddr)
}

func (c *Client) assetForMarket(market *config.Market) (*config.Asset, error) {
	return c.Asset(market.SettlementToken)
}

func (c *Client) needsInitPM(ctx context.Context, o clientOptions, market *config.Market) (bool, error) {
	pmAddr, err := c.getPositionManagerAddress(ctx, o, market)
	if err != nil {
		return false, err
	}
	pmClient := pmclient.NewClient(o.tonAPI, pmAddr)
	inited, _ := pmClient.GetIsInited(ctx)
	return !inited, nil
}

func (c *Client) fetchOraclePayload(ctx context.Context, market *config.Market) (*cell.Cell, error) {
	basePrice, err := c.oracle.GetSignedPrice(ctx, market.BaseAsset)
	if err != nil {
		return nil, err
	}

	if market.Type != config.MarketTypeCoinM {
		return oracle.BuildSimplePayload(basePrice), nil
	}

	settlementPrice, err := c.oracle.GetSignedPrice(ctx, market.SettlementToken)
	if err != nil {
		return nil, err
	}
	return oracle.BuildSettlementPayload(basePrice, settlementPrice), nil
}
