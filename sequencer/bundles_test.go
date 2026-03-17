package sequencer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClient_GetBundles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/bundles", r.URL.Path)
		w.Write([]byte(`[]`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	bundles, err := c.GetBundles(context.Background())
	require.NoError(t, err)
	require.Empty(t, bundles)
}

func TestClient_GetAccountBundles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/smartaccount/0:test/bundles", r.URL.Path)
		w.Write([]byte(`[]`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	bundles, err := c.GetAccountBundles(context.Background(), "0:test")
	require.NoError(t, err)
	require.Empty(t, bundles)
}
