package geografia_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/matexs/fletway-backend/internal/feature/geografia"
	"github.com/matexs/fletway-backend/internal/platform/auth"
	"github.com/matexs/fletway-backend/internal/platform/database"
	"github.com/matexs/fletway-backend/internal/platform/database/dbtest"
	"github.com/matexs/fletway-backend/internal/platform/geocodificacion"
	"github.com/matexs/fletway-backend/internal/platform/httpx"
)

type api struct {
	t   *testing.T
	db  *database.DB
	mux *http.ServeMux
}

func nuevaAPI(t *testing.T) *api {
	t.Helper()
	db := dbtest.Abrir(t)
	mux := http.NewServeMux()
	geografia.Register(mux, db, geocodificacion.Aproximado{})
	return &api{t: t, db: db, mux: mux}
}

func (a *api) llamar(id database.Identity, metodo, path string, body any) (int, json.RawMessage) {
	a.t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(a.t, json.NewEncoder(&buf).Encode(body))
	}
	r := httptest.NewRequest(metodo, path, &buf)
	r = r.WithContext(auth.WithIdentity(r.Context(), id))
	w := httptest.NewRecorder()
	a.mux.ServeHTTP(w, r)
	return w.Code, w.Body.Bytes()
}

func codigo(t *testing.T, raw json.RawMessage) string {
	t.Helper()
	var e httpx.ErrorBody
	require.NoError(t, json.Unmarshal(raw, &e))
	return e.Error.Code
}

func (a *api) zonas() []geografia.ZonaResponse {
	a.t.Helper()
	code, raw := a.llamar(dbtest.CrearUsuario(a.t, "cliente"), "GET", "/zonas", nil)
	require.Equal(a.t, http.StatusOK, code)
	var z []geografia.ZonaResponse
	require.NoError(a.t, json.Unmarshal(raw, &z))
	return z
}

func TestCatalogo(t *testing.T) {
	a := nuevaAPI(t)
	z := a.zonas()
	require.Len(t, z, 14, "las zonas piloto de la migración 0008")
	assert.Equal(t, "Buenos Aires", z[0].Provincia, "ordenado por provincia")
	assert.Equal(t, "CABA", z[13].Provincia)
}

func TestZonasDelTransportista(t *testing.T) {
	a := nuevaAPI(t)
	transportista := dbtest.CrearTransportista(t)
	z := a.zonas()

	code, raw := a.llamar(transportista, "GET", "/transportista/zonas", nil)
	require.Equal(t, http.StatusOK, code)
	assert.JSONEq(t, `{"zona_ids":[]}`, string(raw))

	elegir := func(ids ...string) []string {
		if ids == nil {
			ids = []string{}
		}
		code, raw := a.llamar(transportista, "PUT", "/transportista/zonas", map[string][]string{"zona_ids": ids})
		require.Equal(t, http.StatusOK, code, string(raw))
		var out geografia.ZonasTransportista
		require.NoError(t, json.Unmarshal(raw, &out))
		return out.ZonaIDs
	}
	assert.ElementsMatch(t, []string{z[0].ID, z[1].ID, z[2].ID}, elegir(z[0].ID, z[1].ID, z[2].ID))
	// Reemplaza el conjunto: saca las que no vienen y agrega las nuevas.
	assert.ElementsMatch(t, []string{z[2].ID, z[5].ID}, elegir(z[2].ID, z[5].ID))
	assert.Empty(t, elegir(), "puede dejar de trabajar en todas")

	tests := []struct {
		name     string
		id       database.Identity
		body     any
		wantCode int
		wantErr  string
	}{
		{"zona inexistente", transportista, map[string][]string{"zona_ids": {"8f1d2c3b-0000-4000-8000-0000000000aa"}}, 400, "zona_invalida"},
		{"id mal formado", transportista, map[string][]string{"zona_ids": {"caba"}}, 400, "zona_invalida"},
		{"repetidas", transportista, map[string][]string{"zona_ids": {z[0].ID, z[0].ID}}, 400, "datos_invalidos"},
		{"sin lista", transportista, map[string]any{}, 400, "datos_incompletos"},
		{"cliente", dbtest.CrearUsuario(t, "cliente"), map[string][]string{"zona_ids": {z[0].ID}}, 403, "no_es_transportista"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			code, raw := a.llamar(tc.id, "PUT", "/transportista/zonas", tc.body)
			assert.Equal(t, tc.wantCode, code, string(raw))
			assert.Equal(t, tc.wantErr, codigo(t, raw))
		})
	}
}

func TestOtroNoTocaMisZonas(t *testing.T) {
	a := nuevaAPI(t)
	ctx := context.Background()
	transportista := dbtest.CrearTransportista(t)
	z := a.zonas()
	code, _ := a.llamar(transportista, "PUT", "/transportista/zonas", map[string][]string{"zona_ids": {z[0].ID}})
	require.Equal(t, http.StatusOK, code)

	otro := dbtest.CrearTransportista(t)
	require.NoError(t, a.db.WithinTx(ctx, otro, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `DELETE FROM transportista_zona WHERE transportista_id = $1`, transportista.UserID)
		assert.Zero(t, tag.RowsAffected())
		return err
	}))
	err := a.db.WithinTx(ctx, otro, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `INSERT INTO transportista_zona (transportista_id, zona_id) VALUES ($1, $2)`,
			transportista.UserID, z[1].ID)
		return err
	})
	require.Error(t, err, "no agrega zonas a otro Transportista")
}

func TestSugerencias(t *testing.T) {
	a := nuevaAPI(t)
	c := dbtest.CrearUsuario(t, "cliente")
	var zona string
	for _, z := range a.zonas() {
		if z.Nombre == "Tigre" {
			zona = z.ID
		}
	}
	require.NotEmpty(t, zona)

	code, raw := a.llamar(c, "GET", "/direcciones/sugerencias?zona_id="+zona+"&q=Cazon%201000", nil)
	require.Equal(t, http.StatusOK, code, string(raw))
	var sug []geocodificacion.Sugerencia
	require.NoError(t, json.Unmarshal(raw, &sug))
	require.Len(t, sug, 1, "sin proveedor de mapas sugiere lo escrito")
	assert.Equal(t, "Cazon 1000", sug[0].Direccion)
	assert.Equal(t, "Tigre", sug[0].Detalle)

	casos := []struct {
		name  string
		query string
		code  string
	}{
		{"texto corto", "zona_id=" + zona + "&q=ab", "datos_invalidos"},
		{"zona mal formada", "zona_id=x&q=Cazon", "datos_invalidos"},
		{"zona inexistente", "zona_id=00000000-0000-0000-0000-000000000000&q=Cazon", "zona_invalida"},
	}
	for _, tc := range casos {
		t.Run(tc.name, func(t *testing.T) {
			code, raw := a.llamar(c, "GET", "/direcciones/sugerencias?"+tc.query, nil)
			require.Equal(t, http.StatusBadRequest, code)
			assert.Equal(t, tc.code, codigo(t, raw))
		})
	}
}
