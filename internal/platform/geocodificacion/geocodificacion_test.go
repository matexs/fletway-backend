package geocodificacion_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/matexs/fletway-backend/internal/platform/decimal"
	"github.com/matexs/fletway-backend/internal/platform/geocodificacion"
)

func TestAproximado(t *testing.T) {
	t.Parallel()
	g := geocodificacion.Aproximado{}
	ctx := context.Background()

	a, err := g.Geocodificar(ctx, "Av. Centenario 1200", "San Isidro", "Buenos Aires")
	require.NoError(t, err)
	b, err := g.Geocodificar(ctx, "Av. Centenario 1200", "San Isidro", "Buenos Aires")
	require.NoError(t, err)
	assert.Equal(t, a, b, "la misma dirección da el mismo punto")

	c, err := g.Geocodificar(ctx, "Belgrano 300", "San Isidro", "Buenos Aires")
	require.NoError(t, err)
	assert.NotEqual(t, a, c, "otra dirección de la zona da otro punto")

	// Cerca del centro de San Isidro (-34.4708, -58.5286).
	assert.Equal(t, 1, a.Lat.Cmp(decimal.MustParse("-34.4858")))
	assert.Equal(t, -1, a.Lat.Cmp(decimal.MustParse("-34.4558")))
	assert.LessOrEqual(t, a.Lat.Escala(), 6)

	_, err = g.Geocodificar(ctx, "x", "Rosario", "Santa Fe")
	require.ErrorIs(t, err, geocodificacion.ErrNoGeocodificable)
}

func TestNuevo(t *testing.T) {
	t.Parallel()
	tests := []struct {
		proveedor, entorno string
		wantErr            bool
	}{
		{"aproximado", "development", false},
		{"aproximado", "production", true},
		{"google", "production", true}, // todavía no implementado
		{"otro", "development", true},
	}
	for _, tc := range tests {
		t.Run(tc.proveedor+"/"+tc.entorno, func(t *testing.T) {
			t.Parallel()
			_, err := geocodificacion.Nuevo(tc.proveedor, tc.entorno)
			if tc.wantErr {
				require.ErrorIs(t, err, geocodificacion.ErrProveedor)
				return
			}
			require.NoError(t, err)
		})
	}
}
