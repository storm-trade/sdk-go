package storm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/xssnick/tonutils-go/address"
)

type Market struct {
	ID              int
	Name            string
	Ticker          string
	VammAddress     *address.Address
	VaultAddress    *address.Address
	QuoteAsset      string
	BaseAsset       string
	SettlementToken string
	Type            string
	Tags            []string
	IsHidden        bool
}

type Asset struct {
	Name         string
	Decimals     int
	AssetID      string
	VaultAddress *address.Address
	JettonMaster *address.Address
	QuoteAssetID string
}

type rawConfig struct {
	Assets           []rawAsset           `json:"assets"`
	LiquiditySources []rawLiquiditySource `json:"liquiditySources"`
	OpenedMarkets    []rawMarket          `json:"openedMarkets"`
}

type rawAsset struct {
	Name     string `json:"name"`
	Decimals int    `json:"decimals"`
	AssetID  string `json:"assetId"`
}

type rawLiquiditySource struct {
	Asset        rawAsset `json:"asset"`
	VaultAddress string   `json:"vaultAddress"`
	QuoteAssetID string   `json:"quoteAssetId"`
}

type rawMarket struct {
	ID              int      `json:"id"`
	Name            string   `json:"name"`
	Ticker          string   `json:"ticker"`
	Address         string   `json:"address"`
	VaultAddress    string   `json:"vaultAddress"`
	QuoteAsset      string   `json:"quoteAsset"`
	QuoteAssetID    string   `json:"quoteAssetId"`
	BaseAsset       string   `json:"baseAsset"`
	SettlementToken string   `json:"settlementToken"`
	Tags            []string `json:"tags"`
	Type            string   `json:"type"`
	IsHidden        bool     `json:"isHidden"`
}

type configData struct {
	markets []Market
	assets  []Asset
}

func fetchConfig(ctx context.Context, configURL string, httpClient *http.Client) (*configData, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, configURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create config request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch config: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read config body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("config API returned %d: %s", resp.StatusCode, string(body[:min(len(body), 200)]))
	}

	var raw rawConfig
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return parseConfig(&raw)
}

func parseConfig(raw *rawConfig) (*configData, error) {
	cfg := &configData{}

	for _, rm := range raw.OpenedMarkets {
		vammAddr, err := address.ParseRawAddr(rm.Address)
		if err != nil {
			return nil, fmt.Errorf("parse vamm address for %s: %w", rm.Name, err)
		}
		vaultAddr, err := address.ParseRawAddr(rm.VaultAddress)
		if err != nil {
			return nil, fmt.Errorf("parse vault address for %s: %w", rm.Name, err)
		}
		cfg.markets = append(cfg.markets, Market{
			ID:              rm.ID,
			Name:            rm.Name,
			Ticker:          rm.Ticker,
			VammAddress:     vammAddr,
			VaultAddress:    vaultAddr,
			QuoteAsset:      rm.QuoteAsset,
			BaseAsset:       rm.BaseAsset,
			SettlementToken: rm.SettlementToken,
			Type:            rm.Type,
			Tags:            rm.Tags,
			IsHidden:        rm.IsHidden,
		})
	}

	for _, ls := range raw.LiquiditySources {
		vaultAddr, err := address.ParseRawAddr(ls.VaultAddress)
		if err != nil {
			return nil, fmt.Errorf("parse vault address for %s: %w", ls.Asset.Name, err)
		}
		asset := Asset{
			Name:         ls.Asset.Name,
			Decimals:     ls.Asset.Decimals,
			AssetID:      ls.Asset.AssetID,
			VaultAddress: vaultAddr,
			QuoteAssetID: ls.QuoteAssetID,
		}
		if ls.Asset.AssetID != "TON" {
			jettonAddr, err := address.ParseRawAddr(ls.Asset.AssetID)
			if err != nil {
				return nil, fmt.Errorf("parse jetton master for %s: %w", ls.Asset.Name, err)
			}
			asset.JettonMaster = jettonAddr
		}
		cfg.assets = append(cfg.assets, asset)
	}

	return cfg, nil
}
