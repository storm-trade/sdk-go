package matcher

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBroadcast(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/matcher/tx/broadcast", r.URL.Path)
		require.Equal(t, "POST", r.Method)
		w.Write([]byte(`{"ext_msg_hash":"abc123","sign_ok":true,"received_at":"2026-03-30T00:00:00Z"}`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	resp, err := c.Broadcast(context.Background(), "deadbeef")
	require.NoError(t, err)
	require.Equal(t, "abc123", resp.Hash)
	require.True(t, resp.SignOK)
}
