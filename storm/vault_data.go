package storm

import (
	"context"
	"fmt"

	vaultclient "github.com/storm-trade/sdk-go/client/vault"
	"github.com/storm-trade/sdk-go/contracts/vault"
)

func (c *Client) vaultClient(assetName string) (*vaultclient.Client, error) {
	if c.defaults.tonAPI == nil {
		return nil, fmt.Errorf("TON API required: use WithTONApi()")
	}
	asset, err := c.Asset(assetName)
	if err != nil {
		return nil, err
	}
	return vaultclient.NewClient(c.defaults.tonAPI, asset.VaultAddress), nil
}

func (c *Client) GetVaultData(ctx context.Context, assetName string) (*vault.VaultData, error) {
	vc, err := c.vaultClient(assetName)
	if err != nil {
		return nil, err
	}
	return vc.GetVaultData(ctx)
}

func (c *Client) GetBufferData(ctx context.Context, assetName string) (*vault.BufferData, error) {
	vc, err := c.vaultClient(assetName)
	if err != nil {
		return nil, err
	}
	return vc.GetBufferData(ctx)
}
