package storage_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/matexs/fletway-backend/internal/platform/config"
	"github.com/matexs/fletway-backend/internal/platform/storage"
)

func TestFirmarURL(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/storage/v1/object/sign/documentos-transportista/transportista/u1/dni%201.pdf", r.URL.EscapedPath())
		assert.Equal(t, "Bearer jwt-del-usuario", r.Header.Get("Authorization"))
		assert.Equal(t, "clave-publica", r.Header.Get("apikey"))
		var body map[string]int
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, 600, body["expiresIn"])
		_, _ = w.Write([]byte(`{"signedURL":"/object/sign/documentos-transportista/transportista/u1/dni%201.pdf?token=abc"}`))
	}))
	t.Cleanup(srv.Close)

	c, err := storage.New(config.SupabaseConfig{URL: srv.URL + "/", AnonKey: "clave-publica"})
	require.NoError(t, err)
	got, err := c.FirmarURL(context.Background(), "jwt-del-usuario", "documentos-transportista",
		"transportista/u1/dni 1.pdf", 10*time.Minute)
	require.NoError(t, err)
	assert.Equal(t, srv.URL+"/storage/v1/object/sign/documentos-transportista/transportista/u1/dni%201.pdf?token=abc", got)
}

func TestFirmarURLRechazada(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"not_found"}`))
	}))
	t.Cleanup(srv.Close)

	c, err := storage.New(config.SupabaseConfig{URL: srv.URL, AnonKey: "k"})
	require.NoError(t, err)
	_, err = c.FirmarURL(context.Background(), "t", "b", "p", time.Minute)
	require.Error(t, err)
}

func TestNewSinConfig(t *testing.T) {
	t.Parallel()
	_, err := storage.New(config.SupabaseConfig{URL: "http://x"})
	require.ErrorIs(t, err, storage.ErrSinConfig)
}
