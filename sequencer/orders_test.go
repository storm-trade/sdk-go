package sequencer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClient_PlaceOrder(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/order/place", r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var req PlaceOrderRequest
		json.NewDecoder(r.Body).Decode(&req)
		require.Equal(t, "EQTest", req.SmartAccount)
		require.Equal(t, "msg_base64", req.Message)

		w.Write([]byte(`{"ok":true,"trace":{"exit_code":0,"contract_exit_code":0},"intent":{"hash":"abc123","date":"2025-01-01T00:00:00Z"}}`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	resp, err := c.PlaceOrder(context.Background(), PlaceOrderRequest{
		SmartAccount: "EQTest",
		Message:      "msg_base64",
		PublicKey:    "pubkey_base64",
		Signature:    "sig_base64",
	})
	require.NoError(t, err)
	require.True(t, resp.OK)
	require.NotNil(t, resp.Trace)
	require.Equal(t, 0, resp.Trace.ExitCode)
	require.NotNil(t, resp.Intent)
	require.Equal(t, "abc123", resp.Intent.Hash)
}

func TestClient_CancelOrder(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/order/cancel", r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := NewClient(server.URL)
	err := c.CancelOrder(context.Background(), CancelOrderRequest{
		SmartAccount: "EQTest",
		Message:      "msg_base64",
		PublicKey:    "pubkey_base64",
		Signature:    "sig_base64",
	})
	require.NoError(t, err)
}

func TestClient_PlaceOrder_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"EMULATE_ERROR","message":"contract error","contract_exit_code":123,"vm_exit_code":456,"contract_interface":"vamm"}`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	_, err := c.PlaceOrder(context.Background(), PlaceOrderRequest{SmartAccount: "EQTest"})
	require.Error(t, err)

	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	require.Equal(t, 400, apiErr.StatusCode)
	require.Equal(t, "EMULATE_ERROR", apiErr.Err)
	require.Equal(t, "contract error", apiErr.Message)
	require.Equal(t, 123, apiErr.ContractExitCode)
	require.Equal(t, 456, apiErr.VmExitCode)
	require.Equal(t, "vamm", apiErr.ContractInterface)
	require.True(t, apiErr.IsEmulationError())
}
