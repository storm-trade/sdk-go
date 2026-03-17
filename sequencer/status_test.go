package sequencer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClient_GetStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/status", r.URL.Path)
		require.Equal(t, http.MethodGet, r.Method)
		w.Write([]byte(`{"ok":true,"lag":0,"last_processed_block":46641948,"current_block":46641948,"active_shards":1,"pending_bundles":0,"finalizing_bundles":0,"committed_bundles":0}`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	status, err := c.GetStatus(context.Background())
	require.NoError(t, err)
	require.True(t, status.OK)
	require.Equal(t, uint64(0), status.Lag)
	require.Equal(t, uint32(46641948), status.LastProcessedBlock)
	require.Equal(t, uint32(46641948), status.CurrentBlock)
	require.Equal(t, 1, status.ActiveShards)
	require.Equal(t, 0, status.PendingBundles)
}

func TestClient_GetIntent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/intent/abc123", r.URL.Path)
		w.Write([]byte(`{"hash":"abc123","date":"2025-01-01T00:00:00Z","payment_mode":1}`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	intent, err := c.GetIntent(context.Background(), "abc123")
	require.NoError(t, err)
	require.Equal(t, "abc123", intent.Hash)
	require.Equal(t, "2025-01-01T00:00:00Z", intent.Date)
	require.Equal(t, uint8(1), intent.PaymentMode)
}

func TestClient_GetIntent_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"error":"Key not found"}`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	intent, err := c.GetIntent(context.Background(), "0000")
	require.NoError(t, err)
	require.Empty(t, intent.Hash)
}
