package sequencer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClient_GetOrderbookDepth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/orderbook/1/depth", r.URL.Path)
		require.Equal(t, "5", r.URL.Query().Get("levels"))
		w.Write([]byte(`{"bids":[{"price":6800000,"count":3}],"asks":[{"price":6810000,"count":1}]}`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	depth, err := c.GetOrderbookDepth(context.Background(), "1", 5)
	require.NoError(t, err)
	require.Len(t, depth.Bids, 1)
	require.Len(t, depth.Asks, 1)
	require.Equal(t, int64(6800000), depth.Bids[0].Price)
	require.Equal(t, 3, depth.Bids[0].Count)
	require.Equal(t, int64(6810000), depth.Asks[0].Price)
	require.Equal(t, 1, depth.Asks[0].Count)
}

func TestClient_GetOrderbookDepth_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"asks":[],"bids":[]}`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	depth, err := c.GetOrderbookDepth(context.Background(), "1", 5)
	require.NoError(t, err)
	require.Empty(t, depth.Bids)
	require.Empty(t, depth.Asks)
}

func TestClient_GetBids(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/orderbook/1/bids", r.URL.Path)
		w.Write([]byte(`[{"key":"k1","price":6800000,"smart_account":"0:sa1","vamm":"0:vamm1","order":{"size":100}}]`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	bids, err := c.GetBids(context.Background(), "1")
	require.NoError(t, err)
	require.Len(t, bids, 1)
	require.Equal(t, "k1", bids[0].Key)
	require.Equal(t, int64(6800000), bids[0].Price)
	require.Equal(t, "0:sa1", bids[0].SmartAccount)
	require.Equal(t, "0:vamm1", bids[0].VAmm)
	require.NotNil(t, bids[0].Order)
}

func TestClient_GetAsks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/orderbook/1/asks", r.URL.Path)
		w.Write([]byte(`[{"key":"k2","price":6810000,"smart_account":"0:sa2","vamm":"0:vamm2","order":{"size":50}}]`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	asks, err := c.GetAsks(context.Background(), "1")
	require.NoError(t, err)
	require.Len(t, asks, 1)
	require.Equal(t, int64(6810000), asks[0].Price)
}
