package config

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/xssnick/tonutils-go/address"
)

type Network int

const (
	Testnet Network = iota
	Mainnet
)

type Direction int

const (
	Long  Direction = 0
	Short Direction = 1
)

func (d Direction) Opposite() Direction {
	if d == Long {
		return Short
	}
	return Long
}

type Market struct {
	ID              int
	AssetIndex      int
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

const (
	MarketTypeBase      = "base"
	MarketTypeCoinM     = "coinm"
	MarketTypePrelaunch = "prelaunch"
	NativeAssetID       = "TON"
)

type NetworkInfo struct {
	ConfigURL      string
	FactoryAddress string
}

var Networks = map[Network]NetworkInfo{
	Testnet: {
		ConfigURL:      "https://api.stage.stormtrade.dev/api/config",
		FactoryAddress: "kQDrG1ZEn3BKkFLAcj1o2bDtlyKDxHCWAyhbTqQxLmk3_Qvr",
	},
	Mainnet: {
		ConfigURL:      "https://api5.storm.tg/api/config",
		FactoryAddress: "EQA34l2ywiFdu_kb-HZMqLngFVDjw0DJZHo1aBokOap8xVMU",
	},
}

type BaseAssetInfo struct {
	Name  string `json:"name"`
	Index int    `json:"index"`
	Type  string `json:"type"`
}

type ConfigData struct {
	Markets    []Market
	Assets     []Asset
	BaseAssets []BaseAssetInfo
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

func FetchConfig(ctx context.Context, configURL string, httpClient *http.Client) (*ConfigData, error) {
	raw, err := fetchJSON[rawConfig](ctx, configURL, httpClient)
	if err != nil {
		return nil, err
	}

	baseAssets, err := fetchJSON[[]BaseAssetInfo](ctx, configURL+"/assets", httpClient)
	if err != nil {
		return nil, fmt.Errorf("fetch assets config: %w", err)
	}

	return parseConfig(raw, *baseAssets)
}

func fetchJSON[T any](ctx context.Context, url string, httpClient *http.Client) (*T, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", url, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned %d: %s", resp.StatusCode, string(body[:min(len(body), 200)]))
	}

	var result T
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &result, nil
}

func parseConfig(raw *rawConfig, baseAssets []BaseAssetInfo) (*ConfigData, error) {
	assetIndexMap := make(map[string]int)
	for _, ba := range baseAssets {
		assetIndexMap[ba.Name] = ba.Index
	}

	cfg := &ConfigData{BaseAssets: baseAssets}

	for _, rm := range raw.OpenedMarkets {
		vammAddr, err := address.ParseRawAddr(rm.Address)
		if err != nil {
			return nil, fmt.Errorf("parse vamm address for %s: %w", rm.Name, err)
		}
		vaultAddr, err := address.ParseRawAddr(rm.VaultAddress)
		if err != nil {
			return nil, fmt.Errorf("parse vault address for %s: %w", rm.Name, err)
		}
		cfg.Markets = append(cfg.Markets, Market{
			ID:              rm.ID,
			AssetIndex:      assetIndexMap[rm.BaseAsset],
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
		if ls.Asset.AssetID != NativeAssetID {
			jettonAddr, err := address.ParseRawAddr(ls.Asset.AssetID)
			if err != nil {
				return nil, fmt.Errorf("parse jetton master for %s: %w", ls.Asset.Name, err)
			}
			asset.JettonMaster = jettonAddr
		}
		cfg.Assets = append(cfg.Assets, asset)
	}

	return cfg, nil
}
