package storm

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"time"

	sa "github.com/storm-trade/sdk-go/client/smartaccount"
	"github.com/storm-trade/sdk-go/contracts/hw"
	"github.com/storm-trade/sdk-go/contracts/smartaccount"
	"github.com/storm-trade/sdk-go/sequencer"
	stlb "github.com/storm-trade/sdk-go/tlb"
	"github.com/xssnick/tonutils-go/tlb"
)

type PlaceOrderResult struct {
	QueryID        uint64
	OrderHash      []byte
	StopLossID     uint64
	StopLossHash   []byte
	TakeProfitID   uint64
	TakeProfitHash []byte
	Response       *sequencer.PlaceOrderResponse
}

func (c *Client) PlaceMarketOrder(ctx context.Context, market *Market, dir Direction, amount *tlb.Coins, leverage uint64, opts ...Option) (*PlaceOrderResult, error) {
	o := c.resolveOptions(opts)
	zeroCoins := tlb.ZeroCoins

	stopTrigger := &zeroCoins
	if o.stopLoss != nil {
		stopTrigger = o.stopLoss
	}
	takeTrigger := &zeroCoins
	if o.takeProfit != nil {
		takeTrigger = o.takeProfit
	}

	order := &stlb.Order{
		Value: stlb.MarketOrder{
			Payload: stlb.LimitOrderData{
				Expiration:       uint32(time.Now().Add(5 * time.Minute).Unix()),
				Direction:        uint(dir),
				Amount:           amount,
				Leverage:         leverage,
				LimitPrice:       &zeroCoins,
				StopPrice:        &zeroCoins,
				StopTriggerPrice: stopTrigger,
				TakeTriggerPrice: takeTrigger,
			},
		},
	}

	return c.placeOrder(ctx, market, order, o)
}

func (c *Client) PlaceLimitOrder(ctx context.Context, market *Market, dir Direction, amount *tlb.Coins, leverage uint64, limitPrice *tlb.Coins, opts ...Option) (*PlaceOrderResult, error) {
	o := c.resolveOptions(opts)
	zeroCoins := tlb.ZeroCoins

	stopTrigger := &zeroCoins
	if o.stopLoss != nil {
		stopTrigger = o.stopLoss
	}
	takeTrigger := &zeroCoins
	if o.takeProfit != nil {
		takeTrigger = o.takeProfit
	}

	order := &stlb.Order{
		Value: stlb.LimitOrder{
			Payload: stlb.LimitOrderData{
				Expiration:       uint32(time.Now().Add(24 * time.Hour).Unix()),
				Direction:        uint(dir),
				Amount:           amount,
				Leverage:         leverage,
				LimitPrice:       limitPrice,
				StopPrice:        &zeroCoins,
				StopTriggerPrice: stopTrigger,
				TakeTriggerPrice: takeTrigger,
			},
		},
	}

	return c.placeOrder(ctx, market, order, o)
}

func (c *Client) PlaceStopLimitOrder(ctx context.Context, market *Market, dir Direction, amount *tlb.Coins, leverage uint64, limitPrice, stopPrice *tlb.Coins, opts ...Option) (*PlaceOrderResult, error) {
	o := c.resolveOptions(opts)
	zeroCoins := tlb.ZeroCoins

	stopTrigger := &zeroCoins
	if o.stopLoss != nil {
		stopTrigger = o.stopLoss
	}
	takeTrigger := &zeroCoins
	if o.takeProfit != nil {
		takeTrigger = o.takeProfit
	}

	order := &stlb.Order{
		Value: stlb.LimitOrder{
			Payload: stlb.LimitOrderData{
				Expiration:       uint32(time.Now().Add(24 * time.Hour).Unix()),
				Direction:        uint(dir),
				Amount:           amount,
				Leverage:         leverage,
				LimitPrice:       limitPrice,
				StopPrice:        stopPrice,
				StopTriggerPrice: stopTrigger,
				TakeTriggerPrice: takeTrigger,
			},
		},
	}

	return c.placeOrder(ctx, market, order, o)
}

func (c *Client) ClosePositionFull(ctx context.Context, market *Market, dir Direction, opts ...Option) (*PlaceOrderResult, error) {
	o := c.resolveOptions(opts)
	if o.tonAPI == nil {
		return nil, fmt.Errorf("TON API required: use WithTONApi()")
	}
	if o.smartAccount == nil {
		return nil, fmt.Errorf("smart account required: use WithSmartAccount()")
	}

	saClient := sa.NewClient(o.tonAPI, o.smartAccount)
	pos, err := saClient.GetPosition(ctx, market.VammAddress, uint8(dir))
	if err != nil {
		return nil, fmt.Errorf("get position: %w", err)
	}
	if pos == nil || pos.Data == nil {
		return nil, fmt.Errorf("no open position")
	}

	var state stlb.PositionState
	if err := tlb.LoadFromCell(&state, pos.Data.BeginParse()); err != nil {
		return nil, fmt.Errorf("parse position: %w", err)
	}

	amount := tlb.FromNanoTON(state.Size)
	return c.ClosePosition(ctx, market, dir, &amount, opts...)
}

func (c *Client) ClosePosition(ctx context.Context, market *Market, dir Direction, size *tlb.Coins, opts ...Option) (*PlaceOrderResult, error) {
	o := c.resolveOptions(opts)
	zeroCoins := tlb.ZeroCoins

	order := &stlb.Order{
		Value: stlb.TakeOrder{
			Payload: stlb.StopOrderData{
				Direction:    uint(dir),
				Amount:       size,
				TriggerPrice: &zeroCoins,
			},
		},
	}

	return c.placeOrder(ctx, market, order, o)
}

func (c *Client) PlaceStopLoss(ctx context.Context, market *Market, dir Direction, amount *tlb.Coins, triggerPrice *tlb.Coins, opts ...Option) (*PlaceOrderResult, error) {
	o := c.resolveOptions(opts)

	order := &stlb.Order{
		Value: stlb.StopOrder{
			Payload: stlb.StopOrderData{
				Direction:    uint(dir),
				Amount:       amount,
				TriggerPrice: triggerPrice,
			},
		},
	}

	return c.placeOrder(ctx, market, order, o)
}

func (c *Client) PlaceTakeProfit(ctx context.Context, market *Market, dir Direction, amount *tlb.Coins, triggerPrice *tlb.Coins, opts ...Option) (*PlaceOrderResult, error) {
	o := c.resolveOptions(opts)

	order := &stlb.Order{
		Value: stlb.TakeOrder{
			Payload: stlb.StopOrderData{
				Direction:    uint(dir),
				Amount:       amount,
				TriggerPrice: triggerPrice,
			},
		},
	}

	return c.placeOrder(ctx, market, order, o)
}

func (c *Client) AddMargin(ctx context.Context, market *Market, dir Direction, amount *tlb.Coins, opts ...Option) (*PlaceOrderResult, error) {
	o := c.resolveOptions(opts)

	order := &stlb.Order{
		Value: stlb.AddMarginOrder{
			Payload: stlb.MarginOrderData{
				Direction: uint(dir),
				Amount:    amount,
			},
		},
	}

	return c.placeOrder(ctx, market, order, o)
}

func (c *Client) RemoveMargin(ctx context.Context, market *Market, dir Direction, amount *tlb.Coins, opts ...Option) (*PlaceOrderResult, error) {
	o := c.resolveOptions(opts)

	order := &stlb.Order{
		Value: stlb.RemoveMarginOrder{
			Payload: stlb.MarginOrderData{
				Direction: uint(dir),
				Amount:    amount,
			},
		},
	}

	return c.placeOrder(ctx, market, order, o)
}

func (c *Client) CancelOrder(ctx context.Context, orderHash []byte, opts ...Option) error {
	o := c.resolveOptions(opts)
	if o.signer == nil {
		return fmt.Errorf("signer required: use WithSigner()")
	}
	if o.smartAccount == nil {
		return fmt.Errorf("smart account required: use WithSmartAccount()")
	}

	cancelMsg := &smartaccount.CancelMessage{
		SmartAccountAddress: o.smartAccount,
		OrderId:             orderHash,
	}

	msgCell, err := tlb.ToCell(cancelMsg)
	if err != nil {
		return fmt.Errorf("serialize cancel: %w", err)
	}

	signature := msgCell.Sign(o.signer)
	pubKey := []byte(o.signer.Public().(ed25519.PublicKey))

	return c.seq.CancelOrder(ctx, sequencer.CancelOrderRequest{
		SmartAccount: o.smartAccount.String(),
		Message:      base64.StdEncoding.EncodeToString(msgCell.ToBOC()),
		PublicKey:    base64.StdEncoding.EncodeToString(pubKey),
		Signature:    base64.StdEncoding.EncodeToString(signature),
	})
}

func (c *Client) placeOrder(ctx context.Context, market *Market, order *stlb.Order, o clientOptions) (*PlaceOrderResult, error) {
	if o.signer == nil {
		return nil, fmt.Errorf("signer required: use WithSigner()")
	}
	if o.smartAccount == nil {
		return nil, fmt.Errorf("smart account required: use WithSmartAccount()")
	}

	pubKey := []byte(o.signer.Public().(ed25519.PublicKey))

	nextQID, err := c.seq.GetNextQueryID(ctx, o.smartAccount.String())
	if err != nil {
		return nil, fmt.Errorf("get next query ID: %w", err)
	}

	intent := &smartaccount.UserIntent{
		QueryId:   hw.FromSeqno(nextQID),
		CreatedAt: uint64(time.Now().Add(-o.clockSkew).Unix()),
		PublicKey: pubKey,
		Intent: &smartaccount.UserIntentPayload{
			VAmm:         market.VammAddress,
			SmartAccount: o.smartAccount,
			Order:        order,
		},
	}

	signed, err := smartaccount.SignMessage(intent, o.signer)
	if err != nil {
		return nil, fmt.Errorf("sign order: %w", err)
	}

	intentCell, err := tlb.ToCell(intent)
	if err != nil {
		return nil, fmt.Errorf("serialize intent: %w", err)
	}

	req := sequencer.PlaceOrderRequest{
		SmartAccount: o.smartAccount.String(),
		Message:      base64.StdEncoding.EncodeToString(intentCell.ToBOC()),
		PublicKey:    base64.StdEncoding.EncodeToString(pubKey),
		Signature:    base64.StdEncoding.EncodeToString(signed.Signature),
	}

	if o.gasless {
		req.PaymentMode = sequencer.PaymentModeGasless
	}

	orResult, err := c.buildOrderRequests(o, intent, order, pubKey, nextQID)
	if err != nil {
		return nil, err
	}
	req.OrderRequests = orResult.requests

	resp, err := c.seq.PlaceOrder(ctx, req)
	if err != nil {
		return nil, err
	}

	return &PlaceOrderResult{
		QueryID:        nextQID,
		OrderHash:      intentCell.Hash(),
		StopLossID:     orResult.stopLossID,
		StopLossHash:   orResult.stopLossHash,
		TakeProfitID:   orResult.takeProfitID,
		TakeProfitHash: orResult.takeProfitHash,
		Response:       resp,
	}, nil
}

type orderRequestsResult struct {
	requests       []sequencer.SignedOrderRequest
	stopLossID     uint64
	stopLossHash   []byte
	takeProfitID   uint64
	takeProfitHash []byte
}

func (c *Client) buildOrderRequests(o clientOptions, parentIntent *smartaccount.UserIntent, order *stlb.Order, pubKey []byte, parentQID uint64) (orderRequestsResult, error) {
	var selectors []uint
	if o.stopLoss != nil {
		selectors = append(selectors, 0)
	}
	if o.takeProfit != nil {
		selectors = append(selectors, 1)
	}

	if len(selectors) == 0 {
		return orderRequestsResult{}, nil
	}

	var res orderRequestsResult
	for i, sel := range selectors {
		qid := parentQID + uint64(i) + 1
		if sel == 0 {
			res.stopLossID = qid
		} else {
			res.takeProfitID = qid
		}

		orIntent := &smartaccount.UserIntent{
			QueryId:          hw.FromSeqno(qid),
			CreatedAt:        parentIntent.CreatedAt,
			ReferenceQueryId: parentIntent.QueryId,
			PublicKey:        pubKey,
			Intent: &smartaccount.UserIntentPayload{
				VAmm:         parentIntent.Intent.VAmm,
				SmartAccount: parentIntent.Intent.SmartAccount,
				Order: &stlb.Order{
					Value: stlb.OrderRequest{
						Selector: sel,
						Order:    order.Value,
					},
				},
			},
		}

		signed, err := smartaccount.SignMessage(orIntent, o.signer)
		if err != nil {
			return orderRequestsResult{}, fmt.Errorf("sign order request: %w", err)
		}

		orCell, err := tlb.ToCell(orIntent)
		if err != nil {
			return orderRequestsResult{}, fmt.Errorf("serialize order request: %w", err)
		}

		if sel == 0 {
			res.stopLossHash = orCell.Hash()
		} else {
			res.takeProfitHash = orCell.Hash()
		}

		res.requests = append(res.requests, sequencer.SignedOrderRequest{
			Message:   base64.StdEncoding.EncodeToString(orCell.ToBOC()),
			PublicKey: base64.StdEncoding.EncodeToString(pubKey),
			Signature: base64.StdEncoding.EncodeToString(signed.Signature),
		})
	}

	return res, nil
}
