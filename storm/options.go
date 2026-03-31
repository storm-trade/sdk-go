package storm

import (
	"crypto/ed25519"
	"time"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/wallet"
)

type Option func(*clientOptions)

type clientOptions struct {
	signer       ed25519.PrivateKey
	smartAccount *address.Address
	gasless      bool
	init         bool
	clockSkew    time.Duration
	publicKey    ed25519.PublicKey
	tonAPI       ton.APIClientWrapped
	wallet       *wallet.Wallet
	stopLoss     *tlb.Coins
	takeProfit   *tlb.Coins
	expiration   *uint32
	queryID      *uint64
}

func WithSigner(key ed25519.PrivateKey) Option {
	return func(o *clientOptions) { o.signer = key }
}

func WithSmartAccount(addr *address.Address) Option {
	return func(o *clientOptions) { o.smartAccount = addr }
}

func WithGasless() Option {
	return func(o *clientOptions) { o.gasless = true }
}

func WithTONApi(api ton.APIClientWrapped) Option {
	return func(o *clientOptions) { o.tonAPI = api }
}

func WithWallet(w *wallet.Wallet) Option {
	return func(o *clientOptions) { o.wallet = w }
}

func WithClockSkew(d time.Duration) Option {
	return func(o *clientOptions) { o.clockSkew = d }
}

func WithInit() Option {
	return func(o *clientOptions) { o.init = true }
}

func WithPublicKey(key ed25519.PublicKey) Option {
	return func(o *clientOptions) { o.publicKey = key }
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
