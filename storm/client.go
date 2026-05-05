package storm

import (
	"crypto/ed25519"
	"net/http"

	"github.com/storm-trade/sdk-go/config"
	"github.com/storm-trade/sdk-go/sequencer"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/ton"
)

var sequencerURLs = map[config.Network]string{
	config.Testnet: "http://localhost:9091",
	config.Mainnet: sequencer.MainnetURL,
}

type Client struct {
	network  config.Network
	seq      *sequencer.Client
	defaults clientOptions
	cfg      configLoader
}

func NewClient(network config.Network, opts ...Option) *Client {
	c := &Client{
		network: network,
		seq:     sequencer.NewClient(sequencerURLs[network]),
		cfg: configLoader{
			network:    network,
			httpClient: &http.Client{},
		},
	}
	for _, opt := range opts {
		opt(&c.defaults)
	}
	return c
}

func (c *Client) TonAPI() ton.APIClientWrapped {
	return c.defaults.tonAPI
}

func (c *Client) SmartAccount() *address.Address {
	return c.defaults.smartAccount
}

func (c *Client) Signer() ed25519.PrivateKey {
	return c.defaults.signer
}

func (c *Client) Network() config.Network {
	return c.network
}
