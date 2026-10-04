package geocodificacion_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/matexs/fletway-backend/internal/platform/decimal"
	"github.com/matexs/fletway-backend/internal/platform/geoapify"
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
		proveedor, entorno, key string
		wantErr                 bool
	}{
		{"aproximado", "development", "", false},
		{"aproximado", "production", "", true},
		{"geoapify", "production", "k", false},
		{"geoapify", "development", "", true}, // sin key
		{"google", "production", "k", true},
		{"otro", "development", "", true},
	}
	for _, tc := range tests {
		t.Run(tc.proveedor+"/"+tc.entorno+"/"+tc.key, func(t *testing.T) {
			t.Parallel()
			_, err := geocodificacion.Nuevo(tc.proveedor, tc.entorno, tc.key)
			if tc.wantErr {
				require.ErrorIs(t, err, geocodificacion.ErrProveedor)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestGeoapify(t *testing.T) {
	t.Parallel()
	var texto string
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		texto = r.URL.Query().Get("text")
		switch {
		case strings.Contains(texto, "Inexistente"):
			_, _ = w.Write([]byte(`{"results":[]}`))
		case strings.Contains(texto, "Dudosa"):
			_, _ = w.Write([]byte(`{"results":[{"address_line1":"x","lat":-34.4,"lon":-58.5,"rank":{"confidence":0.2}}]}`))
		default:
			_, _ = w.Write([]byte(`{"results":[{"address_line1":"Avenida Centenario 1200","address_line2":"San Isidro",
				"lat":-34.46590112,"lon":-58.52156381,"rank":{"confidence":1}}]}`))
		}
	}))
	t.Cleanup(s.Close)
	g := geocodificacion.Geoapify{Cliente: geoapify.Nuevo("k", s.URL)}
	ctx := context.Background()

	p, err := g.Geocodificar(ctx, "Avenida Centenario 1200", "San Isidro", "Buenos Aires")
	require.NoError(t, err)
	assert.Equal(t, "-34.465901", p.Lat.String(), "6 decimales, como numeric(9,6)")
	assert.Equal(t, "-58.521564", p.Lng.String())
	assert.Equal(t, "Avenida Centenario 1200, San Isidro, Buenos Aires, Argentina", texto)

	_, err = g.Geocodificar(ctx, "Calle Inexistente 1", "San Isidro", "Buenos Aires")
	require.ErrorIs(t, err, geocodificacion.ErrNoGeocodificable)
	_, err = g.Geocodificar(ctx, "Dudosa 1", "San Isidro", "Buenos Aires")
	require.ErrorIs(t, err, geocodificacion.ErrNoGeocodificable, "poca confianza no se acepta")

	sug, err := g.Sugerir(ctx, "Av Cent", "San Isidro", "Buenos Aires")
	require.NoError(t, err)
	require.Len(t, sug, 1)
	assert.Equal(t, "Avenida Centenario 1200", sug[0].Direccion)
	assert.Equal(t, "San Isidro", sug[0].Detalle)
}
