package auth_test

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/matexs/fletway-backend/internal/platform/auth"
	"github.com/matexs/fletway-backend/internal/platform/auth/authtest"
	"github.com/matexs/fletway-backend/internal/platform/config"
)

func nuevoVerifier(t *testing.T, j *authtest.JWKS) *auth.Verifier {
	t.Helper()
	v, err := auth.NewVerifier(context.Background(), config.SupabaseConfig{
		JWKSURL:          j.URL,
		ExpectedAudience: authtest.Audiencia,
		ExpectedIssuer:   j.Emisor,
	})
	require.NoError(t, err)
	return v
}

func TestVerify(t *testing.T) {
	t.Parallel()
	j := authtest.Nuevo(t)
	v := nuevoVerifier(t, j)
	const sub = "8f1d2c3b-0000-4000-8000-000000000001"

	tests := []struct {
		name    string
		token   func() string
		wantErr error
	}{
		{"token valido", func() string { return j.Firmar(t, j.ClaimsValidos(sub)) }, nil},
		{"token vacio", func() string { return "" }, auth.ErrNoToken},
		{"vencido", func() string {
			c := j.ClaimsValidos(sub)
			c["exp"] = time.Now().Add(-time.Minute).Unix()
			return j.Firmar(t, c)
		}, auth.ErrInvalidToken},
		{"sin exp", func() string {
			c := j.ClaimsValidos(sub)
			delete(c, "exp")
			return j.Firmar(t, c)
		}, auth.ErrInvalidToken},
		{"audiencia equivocada", func() string {
			c := j.ClaimsValidos(sub)
			c["aud"] = "anon"
			return j.Firmar(t, c)
		}, auth.ErrInvalidToken},
		{"emisor equivocado", func() string {
			c := j.ClaimsValidos(sub)
			c["iss"] = "https://otro-proyecto.supabase.co/auth/v1"
			return j.Firmar(t, c)
		}, auth.ErrInvalidToken},
		{"sin sub", func() string {
			c := j.ClaimsValidos(sub)
			delete(c, "sub")
			return j.Firmar(t, c)
		}, auth.ErrInvalidToken},
		{"firmado con otra clave", func() string { return authtest.FirmarConOtraClave(t, j.ClaimsValidos(sub)) }, auth.ErrInvalidToken},
		{"HS256 rechazado", func() string { return authtest.FirmarHS256(t, j.ClaimsValidos(sub)) }, auth.ErrInvalidToken},
		{"basura", func() string { return "no.es.un-jwt" }, auth.ErrInvalidToken},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			id, err := v.Verify(context.Background(), tc.token())
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, sub, id.UserID)
			assert.Equal(t, "authenticated", id.Role)
			assert.Equal(t, "usuario@ejemplo.com", id.Email)
			var claims jwt.MapClaims
			require.NoError(t, json.Unmarshal([]byte(id.RawClaims), &claims))
			assert.Equal(t, sub, claims["sub"])
		})
	}
}

func TestNewVerifierSinJWKS(t *testing.T) {
	t.Parallel()
	_, err := auth.NewVerifier(context.Background(), config.SupabaseConfig{})
	require.ErrorIs(t, err, auth.ErrSinJWKS)
}

func TestBearerToken(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		header  string
		want    string
		wantErr bool
	}{
		{"correcto", "Bearer abc.def.ghi", "abc.def.ghi", false},
		{"bearer en minuscula", "bearer abc", "abc", false},
		{"sin header", "", "", true},
		{"otro esquema", "Basic abc", "", true},
		{"sin token", "Bearer ", "", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := httptest.NewRequest("GET", "/api/me", nil)
			if tc.header != "" {
				r.Header.Set("Authorization", tc.header)
			}
			got, err := auth.BearerToken(r)
			if tc.wantErr {
				require.ErrorIs(t, err, auth.ErrNoToken)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
