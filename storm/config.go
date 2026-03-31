package storm

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/storm-trade/sdk-go/config"
)

type configLoader struct {
	network    config.Network
	httpClient *http.Client
	mu         sync.Mutex
	data       *config.ConfigData
}

func (cl *configLoader) ensure(ctx context.Context) error {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	if cl.data != nil {
		return nil
	}
	cfg, err := config.FetchConfig(ctx, config.Networks[cl.network].ConfigURL, cl.httpClient)
	if err != nil {
		return err
	}
	cl.data = cfg
	return nil
}

func (cl *configLoader) load(ctx context.Context) error {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	cfg, err := config.FetchConfig(ctx, config.Networks[cl.network].ConfigURL, cl.httpClient)
	if err != nil {
		return err
	}
	cl.data = cfg
	return nil
}

func (c *Client) Markets() ([]config.Market, error) {
	if err := c.cfg.ensure(context.Background()); err != nil {
		return nil, err
	}
	return c.cfg.data.Markets, nil
}

func (c *Client) Market(name string, settlement ...string) (*config.Market, error) {
	if err := c.cfg.ensure(context.Background()); err != nil {
		return nil, err
	}
	for i := range c.cfg.data.Markets {
		m := &c.cfg.data.Markets[i]
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

func (c *Client) MustMarket(name string, settlement ...string) *config.Market {
	m, err := c.Market(name, settlement...)
	if err != nil {
		panic(err)
	}
	return m
}

func (c *Client) Assets() ([]config.Asset, error) {
	if err := c.cfg.ensure(context.Background()); err != nil {
		return nil, err
	}
	return c.cfg.data.Assets, nil
}

func (c *Client) Asset(name string) (*config.Asset, error) {
	if err := c.cfg.ensure(context.Background()); err != nil {
		return nil, err
	}
	for i := range c.cfg.data.Assets {
		if c.cfg.data.Assets[i].Name == name {
			return &c.cfg.data.Assets[i], nil
		}
	}
	return nil, fmt.Errorf("unknown asset %q", name)
}

func (c *Client) MustAsset(name string) *config.Asset {
	a, err := c.Asset(name)
	if err != nil {
		panic(err)
	}
	return a
}

func (c *Client) RefreshConfig(ctx context.Context) error {
	return c.cfg.load(ctx)
}
