package storm

import (
	"crypto/ed25519"
	"net/http"
	"sync"
	"time"

	"github.com/storm-trade/sdk-go/config"
	"github.com/storm-trade/sdk-go/sequencer"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/ton"
)

var sequencerURLs = map[config.Network]string{
	config.Testnet: sequencer.TestnetURL,
	config.Mainnet: sequencer.MainnetURL,
}

type Client struct {
	queryMu       sync.Mutex
	queryLeases   map[string]queryLease
	vaultMu       sync.Mutex
	checkedVaults map[string]bool
	network       config.Network
	seq           *sequencer.Client
	defaults      clientOptions
	cfg           configLoader
}

func NewClient(network config.Network, opts ...Option) *Client {
	c := &Client{
		network:       network,
		queryLeases:   make(map[string]queryLease),
		checkedVaults: make(map[string]bool),
		seq:           sequencer.NewClient(sequencerURLs[network]),
		defaults:      clientOptions{clockSkew: 15 * time.Second},
		cfg: configLoader{
			network:    network,
			httpClient: &http.Client{Timeout: 30 * time.Second},
		},
	}
	for _, opt := range opts {
		opt(&c.defaults)
	}
	if c.defaults.sequencer != nil {
		c.seq = c.defaults.sequencer
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
