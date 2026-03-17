package sequencer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClient_GetEvents(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/events", r.URL.Path)
		require.Equal(t, "2", r.URL.Query().Get("limit"))
		require.Equal(t, "ASC", r.URL.Query().Get("order"))
		w.Write([]byte(`[{"sequence":208,"timestamp":1773342135013,"hash":"918961626b5e1d04","account":"0:cdd396d1","tx_status":{"success":true,"exit_code":0,"action_result_code":0},"sender":"0:2bd90e36","type":"NOTIFY_UPDATE_POSITION","payload":{"query_id":1773342133}}]`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	events, err := c.GetEvents(context.Background(), EventsFilter{Limit: 2, Order: "ASC"})
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, uint64(208), events[0].Sequence)
	require.Equal(t, int64(1773342135013), events[0].Timestamp)
	require.Equal(t, "918961626b5e1d04", events[0].Hash)
	require.Equal(t, "0:cdd396d1", events[0].Account)
	require.Equal(t, "NOTIFY_UPDATE_POSITION", events[0].Type)
	require.Equal(t, "0:2bd90e36", events[0].Sender)
	require.NotNil(t, events[0].TxStatus)
	require.True(t, events[0].TxStatus.Success)
	require.NotNil(t, events[0].Payload)
}

func TestClient_GetAccountEvents(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/smartaccount/0:test/events", r.URL.Path)
		require.Equal(t, "trade_notification_v2", r.URL.Query().Get("event_type"))
		w.Write([]byte(`[{"sequence":207,"timestamp":1773342135009,"hash":"e45eba0f","account":"0:test","tx_status":{"success":true,"exit_code":0,"action_result_code":0},"sender":"0:a364f3ae","type":"TRADE_NOTIFICATION_V2","payload":{"query_id":1773342133,"asset_id":47}}]`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	events, err := c.GetAccountEvents(context.Background(), "0:test", EventsFilter{
		EventType: "trade_notification_v2",
	})
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, uint64(207), events[0].Sequence)
	require.Equal(t, "TRADE_NOTIFICATION_V2", events[0].Type)
}
