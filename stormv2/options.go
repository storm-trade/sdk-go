package stormv2

import (
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/wallet"
)

type Option func(*clientOptions)

type clientOptions struct {
	wallet     *wallet.Wallet
	tonAPI     ton.APIClientWrapped
	referralID *uint64
	stopLoss   *tlb.Coins
	takeProfit *tlb.Coins
	expiration *uint32
	queryID    *uint64
}

func WithWallet(w *wallet.Wallet) Option {
	return func(o *clientOptions) { o.wallet = w }
}

func WithTONApi(api ton.APIClientWrapped) Option {
	return func(o *clientOptions) { o.tonAPI = api }
}

func WithReferralID(id uint64) Option {
	return func(o *clientOptions) { o.referralID = &id }
}

func WithStopLoss(price *tlb.Coins) Option {
	return func(o *clientOptions) { o.stopLoss = price }
}

func WithTakeProfit(price *tlb.Coins) Option {
	return func(o *clientOptions) { o.takeProfit = price }
}

func WithExpiration(exp uint32) Option {
	return func(o *clientOptions) { o.expiration = &exp }
}

func WithQueryID(id uint64) Option {
	return func(o *clientOptions) { o.queryID = &id }
}

func (c *Client) resolveOptions(opts []Option) clientOptions {
	resolved := c.defaults
	for _, opt := range opts {
		opt(&resolved)
	}
	return resolved
}
