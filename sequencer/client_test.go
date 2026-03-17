package sequencer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestHTTPTransport_Get(t *testing.T) {
	type response struct {
		Value string `json:"value"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/test/path", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response{Value: "ok"})
	}))
	defer server.Close()

	tr := newHTTPTransport(server.URL, http.DefaultClient)

	var result response
	err := tr.Do(context.Background(), http.MethodGet, "/test/path", nil, &result)
	require.NoError(t, err)
	require.Equal(t, "ok", result.Value)
}

func TestHTTPTransport_Post(t *testing.T) {
	type request struct {
		Name string `json:"name"`
	}
	type response struct {
		ID int `json:"id"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var req request
		json.NewDecoder(r.Body).Decode(&req)
		require.Equal(t, "test", req.Name)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response{ID: 42})
	}))
	defer server.Close()

	tr := newHTTPTransport(server.URL, http.DefaultClient)

	var result response
	err := tr.Do(context.Background(), http.MethodPost, "/create", request{Name: "test"}, &result)
	require.NoError(t, err)
	require.Equal(t, 42, result.ID)
}

func TestHTTPTransport_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"bad_request","msg":"invalid"}`))
	}))
	defer server.Close()

	tr := newHTTPTransport(server.URL, http.DefaultClient)

	var result struct{}
	err := tr.Do(context.Background(), http.MethodGet, "/fail", nil, &result)
	require.Error(t, err)

	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	require.Equal(t, 400, apiErr.StatusCode)
	require.Equal(t, "invalid", apiErr.Msg)
}

func TestNewClient(t *testing.T) {
	c := NewClient(TestnetURL)
	require.NotNil(t, c)
	require.NotNil(t, c.transport)
}

func TestNewClient_WithTimeout(t *testing.T) {
	c := NewClient(TestnetURL, WithTimeout(5*time.Second))
	require.NotNil(t, c)
	require.NotNil(t, c.transport)
}

func TestNewClient_WithHTTPClient(t *testing.T) {
	custom := &http.Client{Timeout: 10 * time.Second}
	c := NewClient(TestnetURL, WithHTTPClient(custom))
	require.NotNil(t, c)

	c2 := NewClient(TestnetURL, WithHTTPClient(custom), WithTimeout(1*time.Second))
	require.NotNil(t, c2)
	require.Equal(t, 10*time.Second, custom.Timeout)
}
