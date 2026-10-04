package perfil_test

import (
	"crypto/rand"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/matexs/fletway-backend/internal/feature/perfil"
	"github.com/matexs/fletway-backend/internal/platform/auth"
	"github.com/matexs/fletway-backend/internal/platform/database"
	"github.com/matexs/fletway-backend/internal/platform/database/dbtest"
)

func llamar(t *testing.T, mux *http.ServeMux, id database.Identity, transportistaID string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest("GET", "/transportistas/"+transportistaID, nil)
	r = r.WithContext(auth.WithIdentity(r.Context(), id))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	return w
}

func vehiculo(t *testing.T, tr database.Identity, tipo string, activo bool) {
	t.Helper()
	dbtest.Exec(t, `INSERT INTO vehiculo (transportista_id, tipo_vehiculo_id, patente, largo_util_m, ancho_util_m,
		alto_util_m, peso_maximo_kg, activo)
		SELECT $1, id, $2, 2, 1.5, 1.5, 800, $4 FROM tipo_vehiculo WHERE nombre = $3`,
		tr.UserID, "PF"+rand.Text()[:8], tipo, activo)
}

func TestPerfil(t *testing.T) {
	db := dbtest.Abrir(t)
	mux := http.NewServeMux()
	perfil.Register(mux, db)

	tr := dbtest.CrearTransportistaHabilitado(t)
	dbtest.Exec(t, `UPDATE transportista SET forma_trabajo = 'Mudanzas chicas, con cuidado.' WHERE usuario_id = $1`, tr.UserID)
	calificacion := 4.5
	dbtest.Reputacion(t, tr.UserID, &calificacion, 92.5)
	dbtest.Exec(t, `INSERT INTO transportista_zona (transportista_id, zona_id)
		SELECT $1, id FROM zona WHERE nombre IN ('Tigre', 'Escobar')`, tr.UserID)
	vehiculo(t, tr, "Furgón chico", true)
	vehiculo(t, tr, "Camión chico", false)
	c := dbtest.CrearUsuario(t, "cliente")
	dbtest.Exec(t, `INSERT INTO cliente (usuario_id) VALUES ($1)`, c.UserID)

	w := llamar(t, mux, c, tr.UserID)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var p perfil.PerfilResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &p))
	assert.Equal(t, tr.UserID, p.ID)
	require.NotNil(t, p.FormaTrabajo)
	assert.Equal(t, "Mudanzas chicas, con cuidado.", *p.FormaTrabajo)
	require.NotNil(t, p.Calificacion)
	assert.InDelta(t, 4.5, *p.Calificacion, 1e-9)
	assert.InDelta(t, 92.5, p.TasaCumplimiento, 1e-9)
	assert.Equal(t, []string{"Escobar", "Tigre"}, p.Zonas)
	assert.Equal(t, []string{"Furgón chico"}, p.TiposVehiculo, "sólo los activos")
	assert.Empty(t, p.Resenas)
	for _, privado := range []string{"email", "telefono", "patente", "PF"} {
		assert.NotContains(t, w.Body.String(), privado)
	}

	pendiente := dbtest.CrearTransportista(t)
	casos := []struct {
		name string
		id   string
	}{
		{"Transportista no habilitado", pendiente.UserID},
		{"un Cliente no tiene perfil de Transportista", c.UserID},
		{"id mal formado", "x"},
	}
	for _, tc := range casos {
		t.Run(tc.name, func(t *testing.T) {
			w := llamar(t, mux, c, tc.id)
			require.Equal(t, http.StatusNotFound, w.Code)
			assert.Contains(t, w.Body.String(), "transportista_no_encontrado")
		})
	}
}
