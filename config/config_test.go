package config

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFetchConfig(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"assets": [{"name":"TON","decimals":9,"assetId":"TON"}],
			"liquiditySources": [{
				"asset": {"name":"TON","decimals":9,"assetId":"TON"},
				"vaultAddress": "0:929ca0bef8881c6b5defaed9d523e23415827102f70924d643c597d101519e58",
				"quoteAssetId": "TON"
			}],
			"openedMarkets": [{
				"id": 1,
				"name": "BTC/USDT",
				"ticker": "BTC/USDT",
				"address": "0:38dc7ce7c6e3d5d43d61324bad11e78fdb8bc8b48fe28c2d8cd5d710b345a0d0",
				"vaultAddress": "0:929ca0bef8881c6b5defaed9d523e23415827102f70924d643c597d101519e58",
				"quoteAsset": "USDT",
				"quoteAssetId": "usdt",
				"baseAsset": "BTC",
				"settlementToken": "TON",
				"tags": ["hot"],
				"type": "base",
				"isHidden": false
			}]
		}`))
	})
	mux.HandleFunc("/assets", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"name":"BTC","index":1,"type":"Crypto"},{"name":"ETH","index":2,"type":"Crypto"}]`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	cfg, err := FetchConfig(context.Background(), server.URL, http.DefaultClient)
	require.NoError(t, err)
	require.Len(t, cfg.Markets, 1)
	require.Equal(t, "BTC/USDT", cfg.Markets[0].Name)
	require.Equal(t, 1, cfg.Markets[0].AssetIndex)
	require.Equal(t, "base", cfg.Markets[0].Type)
	require.NotNil(t, cfg.Markets[0].VammAddress)
	require.NotNil(t, cfg.Markets[0].VaultAddress)
	require.Len(t, cfg.Assets, 1)
	require.Equal(t, "TON", cfg.Assets[0].Name)
	require.Equal(t, 9, cfg.Assets[0].Decimals)
	require.NotNil(t, cfg.Assets[0].VaultAddress)
	require.Len(t, cfg.BaseAssets, 2)
	require.Equal(t, "BTC", cfg.BaseAssets[0].Name)
	require.Equal(t, 1, cfg.BaseAssets[0].Index)
}

func TestFetchConfig_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`internal error`))
	}))
	defer server.Close()

	_, err := FetchConfig(context.Background(), server.URL, http.DefaultClient)
	require.Error(t, err)
	require.Contains(t, err.Error(), "500")
}
