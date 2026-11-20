package sequencer

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClient_GetAccountState(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/smartaccount/0:cdd396d1/state", r.URL.Path)
		w.Write([]byte(`{
			"type":0,
			"factory":"EQDrG1ZEn3BKkFLAcj1o2bDtlyKDxHCWAyhbTqQxLmk3_bBh",
			"owner":"EQB8jr1VjJDoyBdx-Ntm4SWCtKIjsuHAOhjK4-h3C4NYEh3N",
			"balances":[{"address":"EQAr2Q42","amount":942876379639}],
			"version":0,
			"keys":{"hot":"M1C6kyB7","cold":"YWvmFHcX","user_public_keys":{"values":["QcD0TIb/"]},"keys_count":1},
			"positions":{"0:d06d_long":{"locked":false,"data":{"size":12345121,"direction":0,"margin":"98900000000","open_notional":"599017520","last_updated_cumulative_premium":0,"fee":900000,"discount":500000000,"rebate":500000000,"last_updated_timestamp":1765376249}}}
		}`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	state, err := c.GetAccountState(context.Background(), "0:cdd396d1")
	require.NoError(t, err)
	require.Equal(t, "EQB8jr1VjJDoyBdx-Ntm4SWCtKIjsuHAOhjK4-h3C4NYEh3N", state.Owner)
	require.Equal(t, "EQDrG1ZEn3BKkFLAcj1o2bDtlyKDxHCWAyhbTqQxLmk3_bBh", state.Factory)
	require.Equal(t, 0, state.Version)
	require.Len(t, state.Balances, 1)
	require.Equal(t, int64(942876379639), state.Balances[0].Amount)
	require.Equal(t, "EQAr2Q42", state.Balances[0].Address)
	require.NotNil(t, state.Keys)
	require.Equal(t, 1, state.Keys.KeysCount)
	require.Len(t, state.Keys.UserPublicKeys.Values, 1)

	pos, ok := state.Positions["0:d06d_long"]
	require.True(t, ok)
	require.False(t, pos.Locked)
	require.Equal(t, "12345121", pos.Data.Size.String())
	require.Equal(t, "98900000000", pos.Data.Margin)
	require.Equal(t, uint64(900000), pos.Data.Fee)
	require.Equal(t, uint64(1765376249), pos.Data.LastUpdatedTimestamp)
}

func TestClient_GetBalance(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/smartaccount/0:test/balance", r.URL.Path)
		require.Equal(t, "0:token_addr", r.URL.Query().Get("token_address"))
		w.Write([]byte(`{"amount":995807638997364}`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	balance, err := c.GetBalance(context.Background(), "0:test", "0:token_addr")
	require.NoError(t, err)
	require.Equal(t, uint64(995807638997364), balance.Amount)
}

func TestClient_GetBalances(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/smartaccount/0:test/balances", r.URL.Path)
		w.Write([]byte(`{"0:0533db5b":942876379639,"0:12be6e08":995807638997364}`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	balances, err := c.GetBalances(context.Background(), "0:test")
	require.NoError(t, err)
	require.Len(t, balances, 2)
	require.Equal(t, int64(942876379639), balances["0:0533db5b"])
	require.Equal(t, int64(995807638997364), balances["0:12be6e08"])
}

func TestClient_GetLockedBalance(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/smartaccount/0:test/lockedBalance", r.URL.Path)
		require.Equal(t, "0:token_addr", r.URL.Query().Get("token_address"))
		w.Write([]byte(`{"amount":0}`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	balance, err := c.GetLockedBalance(context.Background(), "0:test", "0:token_addr")
	require.NoError(t, err)
	require.Equal(t, uint64(0), balance.Amount)
}

func TestClient_GetPositions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/smartaccount/0:test/positions", r.URL.Path)
		w.Write([]byte(`[{"market":"0:d06d34a3","direction":"long","state":{"status":"active","value":{"size":12345121,"direction":0,"margin":"98900000000","open_notional":"599017520","last_updated_cumulative_premium":0,"fee":900000,"discount":500000000,"rebate":500000000,"last_updated_timestamp":1765376249}}}]`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	positions, err := c.GetPositions(context.Background(), "0:test")
	require.NoError(t, err)
	require.Len(t, positions, 1)
	require.Equal(t, "0:d06d34a3", positions[0].Market)
	require.Equal(t, "long", positions[0].Direction)
	require.Equal(t, "active", positions[0].State.Status)
	require.Equal(t, "12345121", positions[0].State.Value.Size.String())
	require.Equal(t, "98900000000", positions[0].State.Value.Margin)
}

func TestClient_PositionSizeInt128(t *testing.T) {
	const live = `[{"market":"0:38dc7ce7c6e3d5d43d61324bad11e78fdb8bc8b48fe28c2d8cd5d710b345a0d0","direction":"long","state":{"status":"confirmed","value":{"size":%s,"direction":0,"margin":"209874000","open_notional":"229461943","last_updated_cumulative_premium":149994703,"fee":400000,"discount":600000000,"rebate":0,"last_updated_timestamp":1794301247}}}]`
	for _, size := range []string{"210009392", "-2673928924942", "-170141183460469231731687303715884105728", "170141183460469231731687303715884105727"} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, live, size)
		}))
		positions, err := NewClient(server.URL).GetPositions(context.Background(), "0:test")
		server.Close()
		require.NoError(t, err)
		require.Equal(t, size, positions[0].State.Value.Size.String())
	}
}

func TestClient_GetNextQueryID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/smartaccount/0:test/queryID", r.URL.Path)
		w.Write([]byte(`{"query_id":561}`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	qid, err := c.GetNextQueryID(context.Background(), "0:test")
	require.NoError(t, err)
	require.Equal(t, uint64(561), qid)
}

func TestClient_GetGasUnits(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/smartaccount/0:test/gasUnits", r.URL.Path)
		w.Write([]byte(`{"amount":8721921352}`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	gas, err := c.GetGasUnits(context.Background(), "0:test")
	require.NoError(t, err)
	require.Equal(t, uint64(8721921352), gas)
}
