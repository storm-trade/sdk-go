package sequencer

import (
	"context"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBuildersAndIntentStatus(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/builders":
			w.Write([]byte(`[{"builder":"0:abc","rebate":10000000,"fee_earned":"18446744073709551615","active":true,"created_at":1780000000000,"updated_at":1780000000000}]`))
		case "/intent/abc/status":
			w.Write([]byte(`{"status":"failed","error":"emulation","code":"EMULATE_ERROR","contract_exit_code":171,"ts":1780000000000}`))
		case "/order/place":
			w.Write([]byte(`{"ok":true,"accepted":true,"intent_hash":"abc"}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	defer s.Close()
	c := NewClient(s.URL)
	builders, err := c.GetBuilders(context.Background())
	require.NoError(t, err)
	require.Equal(t, "18446744073709551615", builders[0].FeeEarned)
	status, err := c.GetIntentStatus(context.Background(), "abc")
	require.NoError(t, err)
	require.Equal(t, "failed", status.Status)
	require.Equal(t, 171, status.ContractExitCode)
	submitted, err := c.PlaceOrder(context.Background(), PlaceOrderRequest{})
	require.NoError(t, err)
	require.True(t, submitted.Accepted)
	require.Equal(t, "abc", submitted.IntentHash)
}
