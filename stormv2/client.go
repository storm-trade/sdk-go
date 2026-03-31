package stormv2

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/storm-trade/sdk-go/config"
	"github.com/storm-trade/sdk-go/matcher"
	"github.com/storm-trade/sdk-go/oracle"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/wallet"
)

type v2NetworkConfig struct {
	matcherURL      string
	oracleURL       string
	executorAddress *address.Address
	executorFee     tlb.Coins
}

var v2Networks = map[config.Network]v2NetworkConfig{
	config.Testnet: {
		matcherURL:      matcher.TestnetURL,
		oracleURL:       oracle.TestnetURL,
		executorAddress: address.MustParseAddr("EQAkGT4Uwl8KFix3uKxfJE0SOcDqZWhTSXYFvb2wa6HDjdJv"),
		executorFee:     tlb.MustFromTON("0.05"),
	},
	config.Mainnet: {
		matcherURL:      matcher.MainnetURL,
		oracleURL:       oracle.MainnetURL,
		executorAddress: address.MustParseAddr("UQB2OpR9gqb6lmiinABLjt1ZDGjHV6XO9S-DqzEv2OKIMkUo"),
		executorFee:     tlb.MustFromTON("0.0625"),
	},
}

type Client struct {
	network    config.Network
	matcher    *matcher.Client
	oracle     *oracle.Client
	httpClient *http.Client
	defaults   clientOptions

	configMu sync.Mutex
	cfg      *config.ConfigData
}

func NewClient(network config.Network, opts ...Option) *Client {
	nc := v2Networks[network]
	c := &Client{
		network:    network,
		matcher:    matcher.NewClient(nc.matcherURL),
		oracle:     oracle.NewClient(nc.oracleURL),
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
	if c.cfg != nil {
		return nil
	}
	cfg, err := config.FetchConfig(ctx, config.Networks[c.network].ConfigURL, c.httpClient)
	if err != nil {
		return err
	}
	c.cfg = cfg
	return nil
}

func (c *Client) Markets() ([]config.Market, error) {
	if err := c.ensureConfig(context.Background()); err != nil {
		return nil, err
	}
	return c.cfg.Markets, nil
}

func (c *Client) Market(name string, settlement ...string) (*config.Market, error) {
	if err := c.ensureConfig(context.Background()); err != nil {
		return nil, err
	}
	for i := range c.cfg.Markets {
		m := &c.cfg.Markets[i]
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

func (c *Client) Assets() ([]config.Asset, error) {
	if err := c.ensureConfig(context.Background()); err != nil {
		return nil, err
	}
	return c.cfg.Assets, nil
}

func (c *Client) Asset(name string) (*config.Asset, error) {
	if err := c.ensureConfig(context.Background()); err != nil {
		return nil, err
	}
	for i := range c.cfg.Assets {
		if c.cfg.Assets[i].Name == name {
			return &c.cfg.Assets[i], nil
		}
	}
	return nil, fmt.Errorf("unknown asset %q", name)
}

func (c *Client) Network() config.Network {
	return c.network
}

func (c *Client) TonAPI() ton.APIClientWrapped {
	return c.defaults.tonAPI
}

func (c *Client) Wallet() *wallet.Wallet {
	return c.defaults.wallet
}

func (c *Client) executorFeeMsg() *wallet.Message {
	nc := v2Networks[c.network]
	return wallet.SimpleMessage(nc.executorAddress, nc.executorFee, nil)
}
