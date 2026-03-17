package sequencer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClient_GetGaslessBalance(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/gasless/balance/EQTest", r.URL.Path)
		w.Write([]byte(`{"balances":{"TON":"1000000000"},"updated_at":"2026-03-16T22:50:06.109956211Z"}`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	balance, err := c.GetGaslessBalance(context.Background(), "EQTest")
	require.NoError(t, err)
	require.Equal(t, "1000000000", balance.Balances["TON"])
	require.Equal(t, "2026-03-16T22:50:06.109956211Z", balance.UpdatedAt)
}

func TestClient_GetGaslessBalance_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"balances":{},"updated_at":"2026-03-16T22:50:06Z"}`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	balance, err := c.GetGaslessBalance(context.Background(), "EQTest")
	require.NoError(t, err)
	require.Empty(t, balance.Balances)
}

func TestClient_GaslessWithdraw(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/gasless/withdrawal", r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)

		var req GaslessWithdrawRequest
		json.NewDecoder(r.Body).Decode(&req)
		require.Equal(t, "EQTest", req.SmartAccount)
		require.Equal(t, "msg_base64", req.Message)

		w.Write([]byte(`{"ok":true,"withdrawal_id":"0:user","status":"pending","tx_hash":"abc123","amount":"1000000000","asset":"TON","destination":"0:user","query_id":42}`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	resp, err := c.GaslessWithdraw(context.Background(), GaslessWithdrawRequest{
		SmartAccount: "EQTest",
		Message:      "msg_base64",
		PublicKey:    "pubkey_base64",
		Signature:    "sig_base64",
	})
	require.NoError(t, err)
	require.True(t, resp.OK)
	require.Equal(t, "pending", resp.Status)
	require.Equal(t, "abc123", resp.TxHash)
	require.Equal(t, "1000000000", resp.Amount)
	require.Equal(t, uint64(42), resp.QueryID)
}
