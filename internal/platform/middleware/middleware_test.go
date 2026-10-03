package middleware_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/matexs/fletway-backend/internal/platform/auth"
	"github.com/matexs/fletway-backend/internal/platform/auth/authtest"
	"github.com/matexs/fletway-backend/internal/platform/config"
	"github.com/matexs/fletway-backend/internal/platform/middleware"
)

func TestAuth(t *testing.T) {
	t.Parallel()
	j := authtest.Nuevo(t)
	v, err := auth.NewVerifier(context.Background(), config.SupabaseConfig{
		JWKSURL: j.URL, ExpectedAudience: authtest.Audiencia, ExpectedIssuer: j.Emisor,
	})
	require.NoError(t, err)
	const sub = "8f1d2c3b-0000-4000-8000-000000000002"

	// El handler protegido devuelve el sub de la identidad que dejó el middleware.
	protegido := middleware.Auth(v)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := auth.FromContext(r.Context())
		require.True(t, ok)
		_, _ = w.Write([]byte(id.UserID))
	}))

	tests := []struct {
		name       string
		header     string
		wantStatus int
		wantCode   string
		wantBody   string
	}{
		{"sin token", "", http.StatusUnauthorized, "no_autenticado", ""},
		{"token invalido", "Bearer " + authtest.FirmarHS256(t, j.ClaimsValidos(sub)), http.StatusUnauthorized, "token_invalido", ""},
		{"token valido", "Bearer " + j.Firmar(t, j.ClaimsValidos(sub)), http.StatusOK, "", sub},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := httptest.NewRequest(http.MethodGet, "/me", nil)
			if tc.header != "" {
				r.Header.Set("Authorization", tc.header)
			}
			w := httptest.NewRecorder()
			protegido.ServeHTTP(w, r)
			assert.Equal(t, tc.wantStatus, w.Code)
			if tc.wantCode != "" {
				var body struct {
					Error struct {
						Code string `json:"code"`
					} `json:"error"`
				}
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
				assert.Equal(t, tc.wantCode, body.Error.Code)
				return
			}
			assert.Equal(t, tc.wantBody, w.Body.String())
		})
	}
}
