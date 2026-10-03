package matchmaking_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/matexs/fletway-backend/internal/feature/matchmaking"
	"github.com/matexs/fletway-backend/internal/platform/auth"
	"github.com/matexs/fletway-backend/internal/platform/database"
	"github.com/matexs/fletway-backend/internal/platform/database/dbtest"
)

type escenario struct {
	t   *testing.T
	db  *database.DB
	svc *matchmaking.Service
	mux *http.ServeMux
}

func nuevo(t *testing.T) *escenario {
	t.Helper()
	db := dbtest.Abrir(t)
	svc := matchmaking.NewService(db)
	mux := http.NewServeMux()
	matchmaking.Register(mux, svc)
	return &escenario{t: t, db: db, svc: svc, mux: mux}
}

func (e *escenario) valor(sql string, args ...any) string {
	e.t.Helper()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dbtest.URL(e.t))
	require.NoError(e.t, err)
	defer func() { _ = conn.Close(ctx) }()
	var v string
	require.NoError(e.t, conn.QueryRow(ctx, sql, args...).Scan(&v))
	return v
}

func (e *escenario) zona(nombre string) string {
	return e.valor(`SELECT id::text FROM zona WHERE nombre = $1`, nombre)
}

// conZona y conVehiculo arman al Transportista como lo haría la app (módulo 4).
func (e *escenario) conZona(tr database.Identity, zona string) {
	dbtest.Exec(e.t, `INSERT INTO transportista_zona (transportista_id, zona_id) VALUES ($1, $2)`, tr.UserID, e.zona(zona))
}

var patentes = 0

func (e *escenario) conVehiculo(tr database.Identity, pesoKg string, activo bool) {
	patentes++
	dbtest.Exec(e.t, `INSERT INTO vehiculo (transportista_id, tipo_vehiculo_id, patente, largo_util_m, ancho_util_m,
		alto_util_m, peso_maximo_kg, activo)
		SELECT $1, id, $2, 2.4, 1.45, 1.15, $3::numeric, $4 FROM tipo_vehiculo WHERE nombre = 'Furgón chico'`,
		tr.UserID, "MM"+time.Now().Format("150405")+string(rune('A'+patentes%26)), pesoKg, activo)
}

// solicitud publica (como superusuario) una solicitud San Isidro → CABA con una
// heladera (70 kg), para dentro de [dias] días.
func (e *escenario) solicitud(cliente database.Identity, dias int) string {
	e.t.Helper()
	fecha := time.Now().In(time.FixedZone("ART", -3*3600)).AddDate(0, 0, dias).Format(time.DateOnly)
	var id string
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dbtest.URL(e.t))
	require.NoError(e.t, err)
	defer func() { _ = conn.Close(ctx) }()
	require.NoError(e.t, conn.QueryRow(ctx, `
		INSERT INTO solicitud (cliente_id, origen_zona_id, destino_zona_id, origen_direccion, destino_direccion,
		    origen_lat, origen_lng, destino_lat, destino_lng, fecha_servicio_deseada)
		VALUES ($1, $2, $3, 'Av. Centenario 1200', 'Corrientes 3500', -34.47, -58.52, -34.60, -58.38, $4::date)
		RETURNING id::text`, cliente.UserID, e.zona("San Isidro"), e.zona("Ciudad Autónoma de Buenos Aires"), fecha,
	).Scan(&id))
	_, err = conn.Exec(ctx, `INSERT INTO solicitud_objeto (solicitud_id, objeto_id, cantidad, peso_unitario_kg, largo_m, ancho_m, alto_m)
		SELECT $1, id, 1, 0, 1, 1, 1 FROM objeto WHERE nombre = 'Heladera'`, id)
	require.NoError(e.t, err)
	return id
}

func cliente(t *testing.T) database.Identity {
	t.Helper()
	id := dbtest.CrearUsuario(t, "cliente")
	dbtest.Exec(t, `INSERT INTO cliente (usuario_id) VALUES ($1)`, id.UserID)
	return id
}

func (e *escenario) ve(tr database.Identity, solicitudID string) bool {
	e.t.Helper()
	r := httptest.NewRequest("GET", "/transportista/solicitudes", nil)
	r = r.WithContext(auth.WithIdentity(r.Context(), tr))
	w := httptest.NewRecorder()
	e.mux.ServeHTTP(w, r)
	require.Equal(e.t, http.StatusOK, w.Code, w.Body.String())
	var lista []matchmaking.SolicitudCompatible
	require.NoError(e.t, json.Unmarshal(w.Body.Bytes(), &lista))
	for _, s := range lista {
		if s.ID == solicitudID {
			return true
		}
	}
	return false
}

func (e *escenario) avisos(tr database.Identity, solicitudID string) int {
	e.t.Helper()
	var n int
	require.NoError(e.t, e.db.WithinTx(context.Background(), tr, func(tx pgx.Tx) error {
		return tx.QueryRow(context.Background(), `SELECT count(*) FROM notificacion
			WHERE destinatario_usuario_id = (select auth.uid()) AND entidad_referencia_id = $1
			  AND tipo_notificacion_codigo = 'solicitud_compatible'`, solicitudID).Scan(&n)
	}))
	return n
}

// TestCompatibilidad es el criterio de terminado del módulo 7: sólo el
// Transportista que cumple todo D-21 ve la solicitud y recibe el aviso.
func TestCompatibilidad(t *testing.T) {
	e := nuevo(t)
	c := cliente(t)
	admin := dbtest.CrearAdministrador(t)
	id := e.solicitud(c, 2)

	compatible := dbtest.CrearTransportistaHabilitado(t)
	e.conZona(compatible, "San Isidro")
	e.conVehiculo(compatible, "650", true)

	porDestino := dbtest.CrearTransportistaHabilitado(t)
	e.conZona(porDestino, "Ciudad Autónoma de Buenos Aires")
	e.conVehiculo(porDestino, "650", true)

	otraZona := dbtest.CrearTransportistaHabilitado(t)
	e.conZona(otraZona, "Pilar")
	e.conVehiculo(otraZona, "650", true)

	sinCapacidad := dbtest.CrearTransportistaHabilitado(t)
	e.conZona(sinCapacidad, "San Isidro")
	e.conVehiculo(sinCapacidad, "3", true) // 70 kg en tandas de 3 kg: 24 viajes, más de 20

	vehiculoInactivo := dbtest.CrearTransportistaHabilitado(t)
	e.conZona(vehiculoInactivo, "San Isidro")
	e.conVehiculo(vehiculoInactivo, "650", false)

	noDisponible := dbtest.CrearTransportistaHabilitado(t)
	e.conZona(noDisponible, "San Isidro")
	e.conVehiculo(noDisponible, "650", true)
	dbtest.Exec(t, `UPDATE transportista SET disponible = false WHERE usuario_id = $1`, noDisponible.UserID)

	pendiente := dbtest.CrearTransportista(t)
	e.conZona(pendiente, "San Isidro")
	e.conVehiculo(pendiente, "650", true)

	vetado := dbtest.CrearTransportistaHabilitado(t)
	e.conZona(vetado, "San Isidro")
	e.conVehiculo(vetado, "650", true)
	dbtest.Exec(t, `INSERT INTO veto (usuario_id, tipo, motivo, admin_id) VALUES ($1, 'definitivo', 'prueba', $2)`,
		vetado.UserID, admin.UserID)

	vetoVencido := dbtest.CrearTransportistaHabilitado(t)
	e.conZona(vetoVencido, "San Isidro")
	e.conVehiculo(vetoVencido, "650", true)
	dbtest.Exec(t, `INSERT INTO veto (usuario_id, tipo, motivo, admin_id, fecha_inicio, fecha_fin)
		VALUES ($1, 'temporal', 'prueba', $2, now() - interval '10 days', now() - interval '1 day')`,
		vetoVencido.UserID, admin.UserID)

	require.NoError(t, e.svc.NotificarCompatibles(context.Background(), c, id))
	require.NoError(t, e.svc.NotificarCompatibles(context.Background(), c, id), "reintentar no falla")

	tests := []struct {
		name string
		tr   database.Identity
		ve   bool
	}{
		{"zona de origen", compatible, true},
		{"zona de destino", porDestino, true},
		{"veto temporal vencido", vetoVencido, true},
		{"sin zona coincidente", otraZona, false},
		{"sin vehículo con capacidad", sinCapacidad, false},
		{"con el vehículo inactivo", vehiculoInactivo, false},
		{"no disponible", noDisponible, false},
		{"no habilitado", pendiente, false},
		{"vetado", vetado, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.ve, e.ve(tc.tr, id), "listado")
			want := 0
			if tc.ve {
				want = 1
			}
			assert.Equal(t, want, e.avisos(tc.tr, id), "un aviso, sin duplicar al reintentar")
		})
	}
}

func TestListadoYAviso(t *testing.T) {
	e := nuevo(t)
	c := cliente(t)
	tr := dbtest.CrearTransportistaHabilitado(t)
	e.conZona(tr, "San Isidro")
	e.conVehiculo(tr, "650", true)

	t.Run("datos del listado", func(t *testing.T) {
		id := e.solicitud(c, 3)
		r := httptest.NewRequest("GET", "/transportista/solicitudes", nil)
		r = r.WithContext(auth.WithIdentity(r.Context(), tr))
		w := httptest.NewRecorder()
		e.mux.ServeHTTP(w, r)
		var lista []matchmaking.SolicitudCompatible
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &lista))
		for _, s := range lista {
			if s.ID == id {
				assert.Equal(t, "San Isidro", s.OrigenZonaNombre)
				assert.Equal(t, 1, s.CantidadObjetos)
				assert.Equal(t, "70", s.PesoTotalKg.String())
				assert.Equal(t, "0.88", s.VolumenTotalM3.String())
				assert.NotContains(t, w.Body.String(), "monto")
				return
			}
		}
		t.Fatal("no aparece la solicitud compatible")
	})

	t.Run("vencida o cancelada no aparece", func(t *testing.T) {
		vencida := e.solicitud(c, -1)
		cancelada := e.solicitud(c, 2)
		dbtest.Exec(t, `UPDATE solicitud SET estado_codigo = 'cancelada' WHERE id = $1`, cancelada)
		assert.False(t, e.ve(tr, vencida))
		assert.False(t, e.ve(tr, cancelada))
	})

	t.Run("otro cliente no dispara avisos de solicitudes ajenas", func(t *testing.T) {
		id := e.solicitud(c, 2)
		err := e.svc.NotificarCompatibles(context.Background(), cliente(t), id)
		require.Error(t, err)
		assert.Zero(t, e.avisos(tr, id))
	})

	t.Run("un cliente no tiene listado", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/transportista/solicitudes", nil)
		r = r.WithContext(auth.WithIdentity(r.Context(), c))
		w := httptest.NewRecorder()
		e.mux.ServeHTTP(w, r)
		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "no_es_transportista")
	})

	t.Run("las funciones internas no se llaman desde la API", func(t *testing.T) {
		err := e.db.WithinTx(context.Background(), tr, func(tx pgx.Tx) error {
			_, err := tx.Exec(context.Background(), `SELECT fn_es_compatible($1, $1)`, tr.UserID)
			return err
		})
		require.Error(t, err)
	})
}
