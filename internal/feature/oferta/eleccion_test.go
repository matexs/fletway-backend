package oferta_test

import (
	"encoding/json"
	"net/http"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/matexs/fletway-backend/internal/feature/oferta"
	"github.com/matexs/fletway-backend/internal/platform/database"
	"github.com/matexs/fletway-backend/internal/platform/database/dbtest"
)

// ofertar crea la oferta de un Transportista nuevo con [ayudantes] ayudantes
// (cada ayudante cambia el precio) y le pone la calificación dada (nil = sin
// reseñas). Devuelve el Transportista y la oferta.
func (e *escenario) ofertar(sid string, ayudantes int, calificacion *float64) (database.Identity, oferta.OfertaResponse) {
	e.t.Helper()
	tr, vid := e.transportista()
	dbtest.Reputacion(e.t, tr.UserID, calificacion, 100)
	w := e.llamar(tr, "POST", "/solicitudes/"+sid+"/ofertas", pedido(vid, ayudantes))
	require.Equal(e.t, http.StatusCreated, w.Code, w.Body.String())
	var of oferta.OfertaResponse
	require.NoError(e.t, json.Unmarshal(w.Body.Bytes(), &of))
	return tr, of
}

func (e *escenario) ofertasDe(c database.Identity, sid, query string) (int, oferta.OfertasDeSolicitudResponse, string) {
	e.t.Helper()
	w := e.llamar(c, "GET", "/solicitudes/"+sid+"/ofertas"+query, nil)
	var out oferta.OfertasDeSolicitudResponse
	if w.Code == http.StatusOK {
		require.NoError(e.t, json.Unmarshal(w.Body.Bytes(), &out))
	}
	return w.Code, out, w.Body.String()
}

func ids(lista []oferta.OfertaParaCliente) []string {
	out := make([]string, len(lista))
	for i, o := range lista {
		out[i] = o.ID
	}
	return out
}

func f(v float64) *float64 { return &v }

func TestOfertasDeSolicitud(t *testing.T) {
	e := nuevo(t, rutaFija)
	c := cliente(t)
	sid := e.solicitud(c)

	// Más ayudantes, más caro (la operación no baja lo suficiente): el score las
	// ordena por precio a igual reputación, salvo la de 2 ayudantes, que tiene 5
	// estrellas.
	_, of0 := e.ofertar(sid, 0, f(4))
	_, of1 := e.ofertar(sid, 1, f(4))
	_, of2 := e.ofertar(sid, 2, f(5))
	_, of3 := e.ofertar(sid, 3, f(1))
	trRetirada, ofRetirada := e.ofertar(sid, 0, nil)
	require.Equal(t, http.StatusOK, e.llamar(trRetirada, "POST", "/ofertas/"+ofRetirada.ID+"/retirar", nil).Code)
	require.Equal(t, -1, of0.PrecioCalculado.Cmp(of1.PrecioCalculado), "más ayudantes cuesta más")

	code, top, raw := e.ofertasDe(c, sid, "")
	require.Equal(t, http.StatusOK, code, raw)
	assert.Equal(t, 4, top.Total, "la retirada no cuenta")
	assert.Equal(t, 0, top.CantidadAyudantesSolicitados)
	require.Len(t, top.Ofertas, 3, "top 3")
	assert.Equal(t, of0.ID, top.Ofertas[0].ID, "la más barata primero")
	require.NotNil(t, top.SiguienteCursor)
	assert.Equal(t, 3, *top.SiguienteCursor)
	assert.NotContains(t, ids(top.Ofertas), of3.ID, "la de 1 estrella y más cara queda afuera del top")
	assert.NotContains(t, raw, "desglose")
	assert.NotContains(t, raw, "patente")
	assert.NotContains(t, raw, "costo")
	for _, o := range top.Ofertas {
		if o.ID == of2.ID {
			require.NotNil(t, o.Calificacion)
			assert.InDelta(t, 5, *o.Calificacion, 1e-9, "la reputación llega al Cliente")
			assert.InDelta(t, 100, o.TasaCumplimiento, 1e-9)
		}
	}

	code, mas, raw := e.ofertasDe(c, sid, "?ver_mas=true")
	require.Equal(t, http.StatusOK, code, raw)
	assert.Equal(t, []string{of3.ID}, ids(mas.Ofertas), "ver más sigue la misma lista")
	assert.Nil(t, mas.SiguienteCursor)

	code, todas, _ := e.ofertasDe(c, sid, "?ver_mas=true&cursor=0&limit=50")
	require.Equal(t, http.StatusOK, code)
	assert.Equal(t, append(ids(top.Ofertas), of3.ID), ids(todas.Ofertas))

	otro := cliente(t)
	errores := []struct {
		name   string
		id     database.Identity
		query  string
		status int
		code   string
	}{
		{"otro Cliente", otro, "", 404, "solicitud_no_encontrada"},
		{"un Transportista", trRetirada, "", 404, "solicitud_no_encontrada"},
		{"cursor inválido", c, "?ver_mas=true&cursor=-1", 400, "datos_invalidos"},
		{"limit fuera de rango", c, "?ver_mas=true&limit=500", 400, "datos_invalidos"},
	}
	for _, tc := range errores {
		t.Run(tc.name, func(t *testing.T) {
			w := e.llamar(tc.id, "GET", "/solicitudes/"+sid+"/ofertas"+tc.query, nil)
			require.Equal(t, tc.status, w.Code, w.Body.String())
			assert.Equal(t, tc.code, codigo(t, w))
		})
	}
}

var pin = regexp.MustCompile(`^[0-9]{4}$`)

// TestAceptar es el criterio de terminado del módulo 9: la aceptación crea el
// viaje con su desglose y sus PIN en una transacción, y nadie más que el Cliente
// de la solicitud puede hacerla.
func TestAceptar(t *testing.T) {
	e := nuevo(t, rutaFija)
	c := cliente(t)
	sid := e.solicitud(c)
	tr, elegida := e.ofertar(sid, 1, nil)
	_, otra := e.ofertar(sid, 0, nil)
	otroCliente := cliente(t)

	t.Run("sólo el Cliente de la solicitud", func(t *testing.T) {
		for _, id := range []database.Identity{otroCliente, tr} {
			w := e.llamar(id, "POST", "/ofertas/"+elegida.ID+"/aceptar", nil)
			require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
			assert.Equal(t, "oferta_no_encontrada", codigo(t, w))
		}
	})

	w := e.llamar(c, "POST", "/ofertas/"+elegida.ID+"/aceptar", nil)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var v oferta.ViajeConfirmadoResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &v))
	assert.Equal(t, "confirmado", v.Estado)
	assert.Equal(t, tr.UserID, v.TransportistaID)
	assert.NotEmpty(t, v.VehiculoPatente, "la patente se ve al aceptar")
	assert.Equal(t, 0, elegida.PrecioCalculado.Cmp(v.MontoTotal))
	assert.Equal(t, elegida.CantidadViajes, v.CantidadViajes)
	assert.Equal(t, 1, v.CantidadAyudantes)

	assert.Equal(t, "aceptada", e.valor(`SELECT estado_codigo FROM oferta WHERE id = $1`, elegida.ID))
	assert.Equal(t, "no_seleccionada", e.valor(`SELECT estado_codigo FROM oferta WHERE id = $1`, otra.ID))
	assert.Equal(t, "asignada", e.valor(`SELECT estado_codigo FROM solicitud WHERE id = $1`, sid))
	assert.Equal(t, elegida.Desglose.CostoOperativo.String(),
		e.valor(`SELECT costo_operativo::text FROM viaje_costo WHERE viaje_id = $1`, v.ID), "copia el desglose")
	ini := e.valor(`SELECT pin_inicio FROM viaje_pin WHERE viaje_id = $1`, v.ID)
	fin := e.valor(`SELECT pin_fin FROM viaje_pin WHERE viaje_id = $1`, v.ID)
	assert.Regexp(t, pin, ini)
	assert.Regexp(t, pin, fin)
	assert.NotEqual(t, ini, fin)

	t.Run("no se acepta dos veces ni otra oferta", func(t *testing.T) {
		for _, oid := range []string{elegida.ID, otra.ID} {
			w := e.llamar(c, "POST", "/ofertas/"+oid+"/aceptar", nil)
			require.Equal(t, http.StatusConflict, w.Code, w.Body.String())
			assert.Equal(t, "oferta_no_disponible", codigo(t, w))
		}
		assert.Equal(t, "1", e.valor(`SELECT count(*)::text FROM viaje WHERE solicitud_id = $1`, sid))
	})

	t.Run("PIN y desglose: cada uno ve lo suyo", func(t *testing.T) {
		assert.Equal(t, 1, e.contar(c, `SELECT count(*)::int FROM viaje_pin WHERE viaje_id = $1`, v.ID), "el Cliente ve los PIN")
		assert.Equal(t, 0, e.contar(tr, `SELECT count(*)::int FROM viaje_pin WHERE viaje_id = $1`, v.ID), "el Transportista nunca")
		assert.Equal(t, 0, e.contar(c, `SELECT count(*)::int FROM viaje_costo WHERE viaje_id = $1`, v.ID), "el Cliente no ve el desglose")
		assert.Equal(t, 1, e.contar(tr, `SELECT count(*)::int FROM viaje_costo WHERE viaje_id = $1`, v.ID))
		assert.Equal(t, 1, e.contar(c, `SELECT count(*)::int FROM viaje WHERE id = $1`, v.ID))
		require.NoError(t, e.como(tr, `UPDATE viaje_pin SET pin_inicio = '0000' WHERE viaje_id = $1`, v.ID))
		assert.Equal(t, ini, e.valor(`SELECT pin_inicio FROM viaje_pin WHERE viaje_id = $1`, v.ID), "nadie escribe los PIN")
	})

	t.Run("el viaje sólo nace de la aceptación", func(t *testing.T) {
		err := e.como(c, `INSERT INTO viaje (oferta_id, solicitud_id, cliente_id, transportista_id,
			origen_direccion_snapshot, destino_direccion_snapshot, origen_lat_snapshot, origen_lng_snapshot,
			destino_lat_snapshot, destino_lng_snapshot, distancia_km_snapshot, transportista_nombre_snapshot,
			vehiculo_patente_snapshot, monto_total_snapshot, porcentaje_comision_snapshot, cantidad_viajes_snapshot)
			VALUES ($1, $2, (select auth.uid()), $3, 'a', 'b', 0, 0, 0, 0, 1, 'x', 'AA000AA', 1, 0, 1)`,
			otra.ID, sid, tr.UserID)
		assert.Error(t, err)
	})
}

func TestAceptarErrores(t *testing.T) {
	e := nuevo(t, rutaFija)
	admin := dbtest.CrearAdministrador(t)

	t.Run("oferta retirada", func(t *testing.T) {
		c := cliente(t)
		sid := e.solicitud(c)
		tr, of := e.ofertar(sid, 0, nil)
		require.Equal(t, http.StatusOK, e.llamar(tr, "POST", "/ofertas/"+of.ID+"/retirar", nil).Code)
		w := e.llamar(c, "POST", "/ofertas/"+of.ID+"/aceptar", nil)
		require.Equal(t, http.StatusConflict, w.Code, w.Body.String())
		assert.Equal(t, "oferta_no_disponible", codigo(t, w))
	})

	t.Run("solicitud vencida", func(t *testing.T) {
		c := cliente(t)
		sid := e.solicitud(c)
		_, of := e.ofertar(sid, 0, nil)
		dbtest.Exec(t, `UPDATE solicitud SET fecha_servicio_deseada = current_date - 3 WHERE id = $1`, sid)
		w := e.llamar(c, "POST", "/ofertas/"+of.ID+"/aceptar", nil)
		require.Equal(t, http.StatusConflict, w.Code, w.Body.String())
		assert.Equal(t, "solicitud_no_asignable", codigo(t, w))
	})

	t.Run("Transportista vetado después de ofertar", func(t *testing.T) {
		c := cliente(t)
		sid := e.solicitud(c)
		tr, of := e.ofertar(sid, 0, nil)
		dbtest.Exec(t, `INSERT INTO veto (usuario_id, tipo, motivo, admin_id) VALUES ($1, 'definitivo', 'prueba', $2)`,
			tr.UserID, admin.UserID)
		w := e.llamar(c, "POST", "/ofertas/"+of.ID+"/aceptar", nil)
		require.Equal(t, http.StatusConflict, w.Code, w.Body.String())
		assert.Equal(t, "transportista_inhabilitado", codigo(t, w))
		assert.Equal(t, "pendiente", e.valor(`SELECT estado_codigo FROM oferta WHERE id = $1`, of.ID),
			"el error no deja nada a medias")
	})

	t.Run("oferta inexistente", func(t *testing.T) {
		w := e.llamar(cliente(t), "POST", "/ofertas/00000000-0000-0000-0000-000000000000/aceptar", nil)
		require.Equal(t, http.StatusNotFound, w.Code)
		assert.Equal(t, "oferta_no_encontrada", codigo(t, w))
	})
}
