package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/matexs/fletway-backend/internal/platform/config"
)

func TestLoadDerivaJWKSyEmisor(t *testing.T) {
	tests := []struct {
		name       string
		env        map[string]string
		wantJWKS   string
		wantIssuer string
	}{
		{
			name:       "derivados de SUPABASE_URL",
			env:        map[string]string{"SUPABASE_URL": "https://proyecto.supabase.co/"},
			wantJWKS:   "https://proyecto.supabase.co/auth/v1/.well-known/jwks.json",
			wantIssuer: "https://proyecto.supabase.co/auth/v1",
		},
		{
			name: "explicitos ganan",
			env: map[string]string{
				"SUPABASE_URL":        "http://127.0.0.1:54321",
				"SUPABASE_JWKS_URL":   "http://otro/jwks",
				"JWT_EXPECTED_ISSUER": "http://otro/auth/v1",
			},
			wantJWKS:   "http://otro/jwks",
			wantIssuer: "http://otro/auth/v1",
		},
		{name: "sin SUPABASE_URL", env: map[string]string{}, wantJWKS: "", wantIssuer: ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// t.Setenv no admite t.Parallel: estos casos corren en serie.
			for _, k := range []string{"SUPABASE_URL", "SUPABASE_JWKS_URL", "JWT_EXPECTED_ISSUER", "APP_ENV"} {
				t.Setenv(k, "")
			}
			for k, v := range tc.env {
				t.Setenv(k, v)
			}
			cfg, err := config.Load()
			require.NoError(t, err)
			assert.Equal(t, tc.wantJWKS, cfg.Supabase.JWKSURL)
			assert.Equal(t, tc.wantIssuer, cfg.Supabase.ExpectedIssuer)
		})
	}
}
