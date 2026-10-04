package geoapify_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/matexs/fletway-backend/internal/platform/geoapify"
)

// servidor responde como Geoapify según la ruta y guarda el último pedido.
func servidor(t *testing.T, respuestas map[string]string, ultimo *http.Request) *geoapify.Cliente {
	t.Helper()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*ultimo = *r
		cuerpo, ok := respuestas[r.URL.Path]
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(cuerpo))
	}))
	t.Cleanup(s.Close)
	return geoapify.Nuevo("clave-secreta", s.URL)
}

const autocompletado = `{"results":[
 {"address_line1":"Avenida Centenario 1200","address_line2":"1642 San Isidro, Argentina","lat":-34.4659,"lon":-58.5215,
  "result_type":"building","rank":{"confidence":1}},
 {"address_line1":"Avenida Centenario","address_line2":"Beccar, Argentina","lat":-34.4647,"lon":-58.5229,
  "result_type":"street","rank":{"confidence":0.5}}]}`

const ruta = `{"type":"FeatureCollection","features":[{"properties":{"distance":25125,"time":1566},
 "geometry":{"type":"MultiLineString","coordinates":[[[-58.52,-34.46],[-58.50,-34.50]],[[-58.41,-34.60]]]}}]}`

func TestAutocompletar(t *testing.T) {
	var pedido http.Request
	c := servidor(t, map[string]string{"/v1/geocode/autocomplete": autocompletado}, &pedido)
	zona := geoapify.Circulo{Centro: geoapify.Punto{Lat: -34.47, Lng: -58.53}, RadioM: 20000, Definir: true}

	rs, err := c.Autocompletar(context.Background(), "Av Centenario 12", zona, 5)
	require.NoError(t, err)
	require.Len(t, rs, 2)
	assert.Equal(t, "Avenida Centenario 1200", rs[0].Linea1)
	assert.InDelta(t, -34.4659, rs[0].Punto.Lat, 1e-9)
	assert.InDelta(t, -58.5215, rs[0].Punto.Lng, 1e-9)
	assert.InDelta(t, 1, rs[0].Confianza, 1e-9)

	q := pedido.URL.Query()
	assert.Equal(t, "Av Centenario 12", q.Get("text"))
	assert.Equal(t, "circle:-58.530000,-34.470000,20000", q.Get("filter"), "lng primero, como pide Geoapify")
	assert.Equal(t, "proximity:-58.530000,-34.470000", q.Get("bias"))
	assert.Equal(t, "clave-secreta", q.Get("apiKey"))
	assert.Equal(t, "es", q.Get("lang"))
}

func TestBuscarSinZonaYSinResultados(t *testing.T) {
	var pedido http.Request
	c := servidor(t, map[string]string{"/v1/geocode/search": `{"results":[]}`}, &pedido)
	_, err := c.Buscar(context.Background(), "nada", geoapify.Circulo{})
	require.ErrorIs(t, err, geoapify.ErrSinResultados)
	assert.Equal(t, "countrycode:ar", pedido.URL.Query().Get("filter"))
}

func TestRutear(t *testing.T) {
	var pedido http.Request
	c := servidor(t, map[string]string{"/v1/routing": ruta}, &pedido)
	r, err := c.Rutear(context.Background(), geoapify.Punto{Lat: -34.46, Lng: -58.52}, geoapify.Punto{Lat: -34.60, Lng: -58.41})
	require.NoError(t, err)
	assert.InDelta(t, 25125, r.DistanciaM, 1e-9)
	assert.InDelta(t, 1566, r.DuracionS, 1e-9)
	require.Len(t, r.Trazado, 3, "une las partes del MultiLineString")
	assert.InDelta(t, -34.46, r.Trazado[0].Lat, 1e-9, "las coordenadas vienen como [lng, lat]")
	assert.Equal(t, "-34.460000,-58.520000|-34.600000,-58.410000", pedido.URL.Query().Get("waypoints"))
	assert.Equal(t, "drive", pedido.URL.Query().Get("mode"))
}

func TestErrorDelServidorNoExponeLaKey(t *testing.T) {
	var pedido http.Request
	c := servidor(t, map[string]string{}, &pedido)
	_, err := c.Rutear(context.Background(), geoapify.Punto{}, geoapify.Punto{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "401")
	assert.NotContains(t, err.Error(), "clave-secreta")
	assert.False(t, errors.Is(err, geoapify.ErrSinResultados))

	c = geoapify.Nuevo("clave-secreta", "http://127.0.0.1:1")
	_, err = c.Rutear(context.Background(), geoapify.Punto{}, geoapify.Punto{})
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "clave-secreta", "un error de red tampoco")
}
