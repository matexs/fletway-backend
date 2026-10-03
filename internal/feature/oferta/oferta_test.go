package oferta_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/matexs/fletway-backend/internal/feature/oferta"
	"github.com/matexs/fletway-backend/internal/platform/auth"
	"github.com/matexs/fletway-backend/internal/platform/database"
	"github.com/matexs/fletway-backend/internal/platform/database/dbtest"
	"github.com/matexs/fletway-backend/internal/platform/ruteo"
)

// rutaFija hace reproducibles los precios: 10 km y media hora de ida.
var rutaFija = ruteo.Fijo{Ruta: ruteo.Ruta{DistanciaKm: 10, DuracionH: 0.5}}

type escenario struct {
	t   *testing.T
	db  *database.DB
	mux *http.ServeMux
}

func nuevo(t *testing.T, ruta ruteo.Ruteador) *escenario {
	t.Helper()
	db := dbtest.Abrir(t)
	mux := http.NewServeMux()
	oferta.Register(mux, oferta.NewService(db, ruta))
	return &escenario{t: t, db: db, mux: mux}
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

func cliente(t *testing.T) database.Identity {
	t.Helper()
	id := dbtest.CrearUsuario(t, "cliente")
	dbtest.Exec(t, `INSERT INTO cliente (usuario_id) VALUES ($1)`, id.UserID)
	return id
}

// solicitud publica (como superusuario) una solicitud San Isidro → CABA para
// pasado mañana, con una heladera (70 kg, 0,7 × 0,7 × 1,8 m, viaja parada).
func (e *escenario) solicitud(c database.Identity) string {
	e.t.Helper()
	fecha := time.Now().In(time.FixedZone("ART", -3*3600)).AddDate(0, 0, 2).Format(time.DateOnly)
	id := e.valor(`
		INSERT INTO solicitud (cliente_id, origen_zona_id, destino_zona_id, origen_direccion, destino_direccion,
		    origen_lat, origen_lng, destino_lat, destino_lng, fecha_servicio_deseada, pisos_destino)
		VALUES ($1, $2, $3, 'Av. Centenario 1200', 'Corrientes 3500', -34.47, -58.52, -34.60, -58.38, $4::date, 2)
		RETURNING id::text`, c.UserID, e.zona("San Isidro"), e.zona("Ciudad Autónoma de Buenos Aires"), fecha)
	dbtest.Exec(e.t, `INSERT INTO solicitud_objeto (solicitud_id, objeto_id, cantidad, peso_unitario_kg, largo_m, ancho_m, alto_m)
		SELECT $1, id, 1, 0, 1, 1, 1 FROM objeto WHERE nombre = 'Heladera'`, id)
	return id
}

// vehiculo da de alta un vehículo del Transportista con las medidas útiles
// pedidas y, si conCostos, sus costos.
func (e *escenario) vehiculo(tr database.Identity, largo, ancho, alto string, activo, conCostos bool) string {
	e.t.Helper()
	// Única en la tabla, que comparten todos los tests y los datos de prueba manual.
	patente := "OF" + rand.Text()[:8]
	id := e.valor(`INSERT INTO vehiculo (transportista_id, tipo_vehiculo_id, patente, largo_util_m, ancho_util_m,
		alto_util_m, peso_maximo_kg, activo)
		SELECT $1, id, $2, $3::numeric, $4::numeric, $5::numeric, 1500, $6 FROM tipo_vehiculo WHERE nombre = 'Furgón chico'
		RETURNING id::text`, tr.UserID, patente, largo, ancho, alto, activo)
	if conCostos {
		dbtest.Exec(e.t, `INSERT INTO vehiculo_costo (vehiculo_id, combustible_precio_l, rendimiento_km_l,
			cantidad_neumaticos, costo_neumatico, vida_neumatico_km, costo_mantenimiento_km, valor_compra,
			valor_residual, vida_util_km, seguro_mensual, patente_mensual)
			VALUES ($1, 1000, 10, 6, 200000, 60000, 50, 30000000, 6000000, 600000, 192000, 38400)`, id)
	}
	return id
}

// transportista crea un Transportista habilitado con zona San Isidro y un
// vehículo grande con costos, compatible con la solicitud. Devuelve su
// identidad y el vehículo.
func (e *escenario) transportista() (database.Identity, string) {
	e.t.Helper()
	tr := dbtest.CrearTransportistaHabilitado(e.t)
	dbtest.Exec(e.t, `INSERT INTO transportista_zona (transportista_id, zona_id) VALUES ($1, $2)`,
		tr.UserID, e.zona("San Isidro"))
	return tr, e.vehiculo(tr, "3", "1.7", "1.9", true, true)
}

func (e *escenario) llamar(id database.Identity, metodo, path string, body any) *httptest.ResponseRecorder {
	e.t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(e.t, json.NewEncoder(&buf).Encode(body))
	}
	r := httptest.NewRequest(metodo, path, &buf)
	r = r.WithContext(auth.WithIdentity(r.Context(), id))
	w := httptest.NewRecorder()
	e.mux.ServeHTTP(w, r)
	return w
}

func codigo(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var b struct {
		Error struct {
			Code    string         `json:"code"`
			Details map[string]any `json:"details"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &b), w.Body.String())
	return b.Error.Code
}

func pedido(vehiculoID string, ayudantes int) map[string]any {
	return map[string]any{"vehiculo_id": vehiculoID, "cantidad_ayudantes": ayudantes}
}

func TestOfertar(t *testing.T) {
	e := nuevo(t, rutaFija)
	c := cliente(t)
	sid := e.solicitud(c)
	tr, vid := e.transportista()

	w := e.llamar(tr, "POST", "/solicitudes/"+sid+"/ofertas/cotizar", pedido(vid, 1))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var cot oferta.CotizacionResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &cot))
	assert.Equal(t, 1, cot.CantidadViajes)
	assert.Positive(t, cot.PrecioCalculado.Sign())
	assert.Equal(t, "10", cot.Desglose.DistanciaKm.String())
	assert.Equal(t, "0", cot.Desglose.MargenPct.String(), "margen de la seed (D-15)")
	assert.Equal(t, "0", e.valor(`SELECT count(*)::text FROM oferta WHERE solicitud_id = $1`, sid),
		"cotizar no guarda nada")

	w = e.llamar(tr, "POST", "/solicitudes/"+sid+"/ofertas", pedido(vid, 1))
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var of oferta.OfertaResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &of))
	assert.Equal(t, "pendiente", of.Estado)
	assert.Equal(t, sid, of.SolicitudID)
	assert.Equal(t, 1, of.CantidadViajes)
	assert.Equal(t, 1, of.CantidadAyudantes)
	assert.Equal(t, 0, cot.PrecioCalculado.Cmp(of.PrecioCalculado), "mismo precio que la cotización")
	assert.Equal(t, cot.Desglose.CostoOperativo.String(), of.Desglose.CostoOperativo.String())
	assert.Equal(t, "San Isidro", of.OrigenZona)

	// Más ayudantes encarecen la hora pero acortan la operación: el precio cambia.
	w = e.llamar(tr, "POST", "/solicitudes/"+sid+"/ofertas/cotizar", pedido(vid, 0))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var sinAyudantes oferta.CotizacionResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &sinAyudantes))
	assert.NotEqual(t, 0, sinAyudantes.PrecioCalculado.Cmp(of.PrecioCalculado))

	w = e.llamar(tr, "GET", "/transportista/ofertas", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var lista []oferta.OfertaResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &lista))
	require.Len(t, lista, 1)
	assert.Equal(t, of.ID, lista[0].ID)

	// Retirar y volver a ofertar con el mismo vehículo (D-23).
	w = e.llamar(tr, "POST", "/ofertas/"+of.ID+"/retirar", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var retirada oferta.OfertaResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &retirada))
	assert.Equal(t, "retirada", retirada.Estado)

	w = e.llamar(tr, "POST", "/ofertas/"+of.ID+"/retirar", nil)
	require.Equal(t, http.StatusConflict, w.Code)
	assert.Equal(t, "oferta_no_retirable", codigo(t, w))

	w = e.llamar(tr, "POST", "/solicitudes/"+sid+"/ofertas", pedido(vid, 2))
	require.Equal(t, http.StatusCreated, w.Code, "después de retirar se puede ofertar otra vez: %s", w.Body.String())
}

func TestOfertarErrores(t *testing.T) {
	e := nuevo(t, rutaFija)
	c := cliente(t)
	sid := e.solicitud(c)
	tr, vid := e.transportista()

	ajeno, vidAjeno := e.transportista()
	_ = ajeno
	inactivo := e.vehiculo(tr, "3", "1.7", "1.9", false, true)
	sinCostos := e.vehiculo(tr, "3", "1.7", "1.9", true, false)
	bajo := e.vehiculo(tr, "2", "1.2", "1.2", true, true) // la heladera entraría acostada, pero viaja parada

	otraZona := dbtest.CrearTransportistaHabilitado(t)
	dbtest.Exec(t, `INSERT INTO transportista_zona (transportista_id, zona_id) VALUES ($1, $2)`,
		otraZona.UserID, e.zona("Pilar"))
	vidOtraZona := e.vehiculo(otraZona, "3", "1.7", "1.9", true, true)

	noDisponible, vidNoDisponible := e.transportista()
	dbtest.Exec(t, `UPDATE transportista SET disponible = false WHERE usuario_id = $1`, noDisponible.UserID)

	pendiente := dbtest.CrearTransportista(t)

	duplicado, vidDuplicado := e.transportista()
	w := e.llamar(duplicado, "POST", "/solicitudes/"+sid+"/ofertas", pedido(vidDuplicado, 0))
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	casos := []struct {
		name   string
		id     database.Identity
		path   string
		body   any
		status int
		code   string
	}{
		{"más de 3 ayudantes", tr, sid, pedido(vid, 4), 400, "datos_invalidos"},
		{"ayudantes negativos", tr, sid, pedido(vid, -1), 400, "datos_invalidos"},
		{"vehículo mal formado", tr, sid, pedido("x", 0), 400, "datos_invalidos"},
		{"vehículo de otro Transportista", tr, sid, pedido(vidAjeno, 0), 404, "vehiculo_no_encontrado"},
		{"vehículo inactivo", tr, sid, pedido(inactivo, 0), 409, "vehiculo_inactivo"},
		{"vehículo sin costos", tr, sid, pedido(sinCostos, 0), 409, "costos_no_cargados"},
		{"la carga no entra", tr, sid, pedido(bajo, 0), 400, "carga_no_factible"},
		{"solicitud fuera de sus zonas", otraZona, sid, pedido(vidOtraZona, 0), 404, "solicitud_no_disponible"},
		{"solicitud inexistente", tr, "00000000-0000-0000-0000-000000000000", pedido(vid, 0), 404, "solicitud_no_disponible"},
		{"no disponible", noDisponible, sid, pedido(vidNoDisponible, 0), 409, "transportista_no_disponible"},
		{"no habilitado", pendiente, sid, pedido(vid, 0), 403, "transportista_no_habilitado"},
		{"un Cliente", c, sid, pedido(vid, 0), 403, "no_es_transportista"},
		{"oferta vigente con el mismo vehículo", duplicado, sid, pedido(vidDuplicado, 1), 409, "oferta_duplicada"},
	}
	for _, tc := range casos {
		t.Run(tc.name, func(t *testing.T) {
			w := e.llamar(tc.id, "POST", "/solicitudes/"+tc.path+"/ofertas", tc.body)
			require.Equal(t, tc.status, w.Code, w.Body.String())
			assert.Equal(t, tc.code, codigo(t, w))
		})
	}

	t.Run("el motivo de la carga que no entra", func(t *testing.T) {
		w := e.llamar(tr, "POST", "/solicitudes/"+sid+"/ofertas/cotizar", pedido(bajo, 0))
		require.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Heladera: no entra en la posición en que tiene que viajar")
	})

	t.Run("ruta no disponible", func(t *testing.T) {
		sinRuta := nuevo(t, ruteo.Fijo{Err: ruteo.ErrRutaNoDisponible})
		w := sinRuta.llamar(tr, "POST", "/solicitudes/"+sid+"/ofertas", pedido(vid, 0))
		require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
		assert.Equal(t, "ruta_no_disponible", codigo(t, w))
	})

	t.Run("retirar la oferta de otro", func(t *testing.T) {
		oid := e.valor(`SELECT id::text FROM oferta WHERE transportista_id = $1`, duplicado.UserID)
		w := e.llamar(tr, "POST", "/ofertas/"+oid+"/retirar", nil)
		require.Equal(t, http.StatusNotFound, w.Code)
		assert.Equal(t, "oferta_no_encontrada", codigo(t, w))
	})
}

// contar cuenta filas con la identidad (RLS aplicada).
func (e *escenario) contar(id database.Identity, sql string, args ...any) int {
	e.t.Helper()
	var n int
	require.NoError(e.t, e.db.WithinTx(context.Background(), id, func(tx pgx.Tx) error {
		return tx.QueryRow(context.Background(), sql, args...).Scan(&n)
	}))
	return n
}

func (e *escenario) como(id database.Identity, sql string, args ...any) error {
	e.t.Helper()
	return e.db.WithinTx(context.Background(), id, func(tx pgx.Tx) error {
		_, err := tx.Exec(context.Background(), sql, args...)
		return err
	})
}

// TestProteccionOferta es el criterio de terminado del módulo 8: el Cliente no
// lee el desglose y nadie modifica el precio (D-23, migración 0016).
func TestProteccionOferta(t *testing.T) {
	e := nuevo(t, rutaFija)
	c := cliente(t)
	sid := e.solicitud(c)
	tr, vid := e.transportista()
	otro, _ := e.transportista()
	admin := dbtest.CrearAdministrador(t)

	w := e.llamar(tr, "POST", "/solicitudes/"+sid+"/ofertas", pedido(vid, 0))
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var of oferta.OfertaResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &of))
	precio := of.PrecioCalculado.String()

	lectura := []struct {
		name         string
		id           database.Identity
		oferta, cost int
	}{
		{"el Transportista dueño ve oferta y desglose", tr, 1, 1},
		{"el Cliente ve la oferta pero no el desglose", c, 1, 0},
		{"otro Transportista no ve nada", otro, 0, 0},
		{"el Administrador ve todo", admin, 1, 1},
	}
	for _, tc := range lectura {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.oferta, e.contar(tc.id, `SELECT count(*)::int FROM oferta WHERE id = $1`, of.ID))
			assert.Equal(t, tc.cost, e.contar(tc.id, `SELECT count(*)::int FROM oferta_costo WHERE oferta_id = $1`, of.ID))
		})
	}

	escritura := []struct {
		name string
		id   database.Identity
		sql  string
	}{
		{"el Transportista no cambia el precio", tr, `UPDATE oferta SET precio_calculado = 1 WHERE id = $1`},
		{"el Transportista no cambia los viajes", tr, `UPDATE oferta SET cantidad_viajes = 9 WHERE id = $1`},
		{"el Cliente no cambia el precio", c, `UPDATE oferta SET precio_calculado = 1 WHERE id = $1`},
		{"el Cliente no acepta por UPDATE", c, `UPDATE oferta SET estado_codigo = 'aceptada' WHERE id = $1`},
		{"el Transportista no se autoacepta", tr, `UPDATE oferta SET estado_codigo = 'aceptada' WHERE id = $1`},
	}
	for _, tc := range escritura {
		t.Run(tc.name, func(t *testing.T) {
			assert.Error(t, e.como(tc.id, tc.sql, of.ID))
		})
	}

	t.Run("nadie modifica ni borra el desglose", func(t *testing.T) {
		costo := e.valor(`SELECT costo_operativo::text FROM oferta_costo WHERE oferta_id = $1`, of.ID)
		for _, id := range []database.Identity{tr, c} {
			require.NoError(t, e.como(id, `UPDATE oferta_costo SET costo_operativo = 1 WHERE oferta_id = $1`, of.ID))
			require.NoError(t, e.como(id, `DELETE FROM oferta_costo WHERE oferta_id = $1`, of.ID))
		}
		assert.Equal(t, costo, e.valor(`SELECT costo_operativo::text FROM oferta_costo WHERE oferta_id = $1`, of.ID),
			"sin policies de UPDATE ni DELETE no se afecta ninguna fila")
	})

	t.Run("no se crea una oferta sin desglose", func(t *testing.T) {
		// Lo que haría un Transportista por PostgREST: un INSERT suelto con su precio.
		otroVehiculo := e.vehiculo(tr, "3", "1.7", "1.9", true, true)
		err := e.como(tr, `INSERT INTO oferta (solicitud_id, transportista_id, vehiculo_id, cantidad_viajes,
			cantidad_ayudantes, precio_calculado) VALUES ($1, (select auth.uid()), $2, 1, 0, 1)`, sid, otroVehiculo)
		assert.ErrorContains(t, err, "falta el desglose")
	})

	t.Run("no se oferta con el vehículo de otro", func(t *testing.T) {
		ajeno := e.vehiculo(otro, "3", "1.7", "1.9", true, true)
		err := e.como(tr, `INSERT INTO oferta (solicitud_id, transportista_id, vehiculo_id, cantidad_viajes,
			cantidad_ayudantes, precio_calculado) VALUES ($1, (select auth.uid()), $2, 1, 0, 1)`, sid, ajeno)
		assert.ErrorContains(t, err, "el vehículo no es del Transportista")
	})

	assert.Equal(t, precio, e.valor(`SELECT precio_calculado::text FROM oferta WHERE id = $1`, of.ID))

	t.Run("cancelar la solicitud deja las ofertas como no seleccionadas", func(t *testing.T) {
		require.NoError(t, e.como(c, `UPDATE solicitud SET estado_codigo = 'cancelada' WHERE id = $1`, sid))
		require.NoError(t, e.como(c, `UPDATE oferta SET estado_codigo = 'no_seleccionada'
			WHERE solicitud_id = $1 AND estado_codigo = 'pendiente'`, sid))
		assert.Equal(t, "no_seleccionada", e.valor(`SELECT estado_codigo FROM oferta WHERE id = $1`, of.ID))
	})
}
