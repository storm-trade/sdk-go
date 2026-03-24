package storm

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/storm-trade/sdk-go/sequencer"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/ton"
)

type Network int

const (
	Testnet Network = iota
	Mainnet
)

type networkConfig struct {
	sequencerURL   string
	configURL      string
	factoryAddress string
}

var networks = map[Network]networkConfig{
	Testnet: {
		sequencerURL:   sequencer.TestnetURL,
		configURL:      "https://api.stage.stormtrade.dev/api/config",
		factoryAddress: "kQDrG1ZEn3BKkFLAcj1o2bDtlyKDxHCWAyhbTqQxLmk3_Qvr",
	},
	Mainnet: {
		sequencerURL:   sequencer.MainnetURL,
		configURL:      "https://api5.storm.tg/api/config",
		factoryAddress: "EQA34l2ywiFdu_kb-HZMqLngFVDjw0DJZHo1aBokOap8xVMU",
	},
}

func (c *Client) FactoryAddress() string {
	return networks[c.network].factoryAddress
}

func (c *Client) TonAPI() ton.APIClientWrapped {
	return c.defaults.tonAPI
}

func (c *Client) SmartAccount() *address.Address {
	return c.defaults.smartAccount
}

type Client struct {
	network    Network
	seq        *sequencer.Client
	defaults   clientOptions
	httpClient *http.Client

	configMu sync.Mutex
	config   *configData
}

func NewClient(network Network, opts ...Option) *Client {
	nc := networks[network]

	c := &Client{
		network:    network,
		seq:        sequencer.NewClient(nc.sequencerURL),
		httpClient: &http.Client{},
	}

	for _, opt := range opts {
		opt(&c.defaults)
	}

	return c
}

func (c *Client) ensureConfig(ctx context.Context) error {
	c.configMu.Lock()
	defer c.configMu.Unlock()

	if c.config != nil {
		return nil
	}

	cfg, err := fetchConfig(ctx, networks[c.network].configURL, c.httpClient)
	if err != nil {
		return err
	}

	c.config = cfg
	return nil
}

func (c *Client) LoadConfig(ctx context.Context) error {
	c.configMu.Lock()
	defer c.configMu.Unlock()

	cfg, err := fetchConfig(ctx, networks[c.network].configURL, c.httpClient)
	if err != nil {
		return err
	}

	c.config = cfg
	return nil
}

func (c *Client) RefreshConfig(ctx context.Context) error {
	return c.LoadConfig(ctx)
}

func (c *Client) Markets() ([]Market, error) {
	if err := c.ensureConfig(context.Background()); err != nil {
		return nil, err
	}
	return c.config.markets, nil
}

func (c *Client) Market(name string, settlement ...string) (*Market, error) {
	if err := c.ensureConfig(context.Background()); err != nil {
		return nil, err
	}

	for i := range c.config.markets {
		m := &c.config.markets[i]
		if m.Name != name && m.Ticker != name {
			continue
		}
		if len(settlement) > 0 && m.SettlementToken != settlement[0] {
			continue
		}
		return m, nil
	}

	return nil, fmt.Errorf("unknown market %q", name)
}

func (c *Client) MustMarket(name string, settlement ...string) *Market {
	m, err := c.Market(name, settlement...)
	if err != nil {
		panic(err)
	}
	return m
}

func (c *Client) Assets() ([]Asset, error) {
	if err := c.ensureConfig(context.Background()); err != nil {
		return nil, err
	}
	return c.config.assets, nil
}

func (c *Client) Asset(name string) (*Asset, error) {
	if err := c.ensureConfig(context.Background()); err != nil {
		return nil, err
	}

	for i := range c.config.assets {
		if c.config.assets[i].Name == name {
			return &c.config.assets[i], nil
		}
	}

	available := make([]string, len(c.config.assets))
	for i, a := range c.config.assets {
		available[i] = a.Name
	}
	return nil, fmt.Errorf("unknown asset %q, available: %v", name, available)
}

func (c *Client) MustAsset(name string) *Asset {
	a, err := c.Asset(name)
	if err != nil {
		panic(err)
	}
	return a
}
