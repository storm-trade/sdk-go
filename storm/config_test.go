package storm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFetchConfig(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	}))
	defer server.Close()

	cfg, err := fetchConfig(context.Background(), server.URL, http.DefaultClient)
	require.NoError(t, err)
	require.Len(t, cfg.markets, 1)
	require.Equal(t, "BTC/USDT", cfg.markets[0].Name)
	require.Equal(t, "base", cfg.markets[0].Type)
	require.NotNil(t, cfg.markets[0].VammAddress)
	require.NotNil(t, cfg.markets[0].VaultAddress)
	require.Len(t, cfg.assets, 1)
	require.Equal(t, "TON", cfg.assets[0].Name)
	require.Equal(t, 9, cfg.assets[0].Decimals)
	require.NotNil(t, cfg.assets[0].VaultAddress)
}

func TestFetchConfig_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`internal error`))
	}))
	defer server.Close()

	_, err := fetchConfig(context.Background(), server.URL, http.DefaultClient)
	require.Error(t, err)
	require.Contains(t, err.Error(), "500")
}
