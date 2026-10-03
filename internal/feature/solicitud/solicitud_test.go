package solicitud_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/matexs/fletway-backend/internal/feature/solicitud"
	"github.com/matexs/fletway-backend/internal/platform/auth"
	"github.com/matexs/fletway-backend/internal/platform/database"
	"github.com/matexs/fletway-backend/internal/platform/database/dbtest"
	"github.com/matexs/fletway-backend/internal/platform/geocodificacion"
	"github.com/matexs/fletway-backend/internal/platform/httpx"
)

// hoy es el "hoy" fijo de los tests: 10 de octubre de 2026, mediodía en Argentina.
var hoy = time.Date(2026, 10, 10, 15, 0, 0, 0, time.UTC)

type api struct {
	t   *testing.T
	db  *database.DB
	mux *http.ServeMux
}

func nuevaAPI(t *testing.T, ahora time.Time) *api {
	t.Helper()
	db := dbtest.Abrir(t)
	return nuevaAPIConDB(t, db, ahora)
}

func nuevaAPIConDB(t *testing.T, db *database.DB, ahora time.Time) *api {
	t.Helper()
	mux := http.NewServeMux()
	svc := solicitud.NewService(db, geocodificacion.Aproximado{}).ConReloj(func() time.Time { return ahora })
	solicitud.Register(mux, svc)
	return &api{t: t, db: db, mux: mux}
}

func (a *api) llamar(id database.Identity, metodo, path string, body any) (int, json.RawMessage) {
	a.t.Helper()
	var buf bytes.Buffer
	switch b := body.(type) {
	case nil:
	case string:
		buf.WriteString(b)
	default:
		require.NoError(a.t, json.NewEncoder(&buf).Encode(b))
	}
	r := httptest.NewRequest(metodo, path, &buf)
	r = r.WithContext(auth.WithIdentity(r.Context(), id))
	w := httptest.NewRecorder()
	a.mux.ServeHTTP(w, r)
	return w.Code, w.Body.Bytes()
}

func errorDe(t *testing.T, raw json.RawMessage) httpx.ErrorDetail {
	t.Helper()
	var e httpx.ErrorBody
	require.NoError(t, json.Unmarshal(raw, &e))
	return e.Error
}

// cliente crea un Cliente registrado (con fila cliente).
func cliente(t *testing.T) database.Identity {
	t.Helper()
	id := dbtest.CrearUsuario(t, "cliente")
	dbtest.Exec(t, `INSERT INTO cliente (usuario_id) VALUES ($1)`, id.UserID)
	return id
}

// ids lee los ids que necesitan los requests.
func (a *api) id(sql string, args ...any) string {
	a.t.Helper()
	var v string
	require.NoError(a.t, a.db.WithinTx(context.Background(), cliente(a.t), func(tx pgx.Tx) error {
		return tx.QueryRow(context.Background(), sql, args...).Scan(&v)
	}))
	return v
}

// cuerpo es un alta válida: una heladera del catálogo y un piano cargado a mano.
func (a *api) cuerpo(fecha string) map[string]any {
	a.t.Helper()
	sanIsidro := a.id(`SELECT id::text FROM zona WHERE nombre = 'San Isidro'`)
	caba := a.id(`SELECT id::text FROM zona WHERE provincia = 'CABA'`)
	heladera := a.id(`SELECT id::text FROM objeto WHERE nombre = 'Heladera'`)
	return map[string]any{
		"origen": map[string]any{"zona_id": sanIsidro, "direccion": "Av. Centenario 1200",
			"pisos": 2, "ascensor_utilizable": false, "distancia_vehiculo_m": 15.5},
		"destino": map[string]any{"zona_id": caba, "direccion": "Av. Corrientes 3500",
			"pisos": 5, "ascensor_utilizable": true, "distancia_vehiculo_m": 0},
		"fecha_servicio_deseada":         fecha,
		"franja_horaria_inicio":          "09:00",
		"franja_horaria_fin":             "13:00",
		"cantidad_ayudantes_solicitados": 2,
		"objetos": []any{
			// Los valores de peso y medidas que mande el cliente para un objeto del
			// catálogo se ignoran: los pone la base.
			map[string]any{"objeto_id": heladera, "cantidad": 1},
			map[string]any{"nombre_personalizado": "Piano vertical", "cantidad": 1, "peso_unitario_kg": 250,
				"largo_m": 1.5, "ancho_m": 0.6, "alto_m": 1.3, "rotacion_vertical": false, "apilable": false},
		},
	}
}

func (a *api) crear(id database.Identity, fecha string) solicitud.SolicitudResponse {
	a.t.Helper()
	code, raw := a.llamar(id, "POST", "/solicitudes", a.cuerpo(fecha))
	require.Equal(a.t, http.StatusCreated, code, string(raw))
	var s solicitud.SolicitudResponse
	require.NoError(a.t, json.Unmarshal(raw, &s))
	return s
}

// TestPublicar es el criterio de terminado del módulo 6: objetos de catálogo y
// manuales quedan con todas las medidas copiadas y no hay ningún monto.
func TestPublicar(t *testing.T) {
	a := nuevaAPI(t, hoy)
	c := cliente(t)

	code, raw := a.llamar(c, "POST", "/solicitudes", a.cuerpo("2026-10-10"))
	require.Equal(t, http.StatusCreated, code, string(raw))
	for _, prohibido := range []string{"monto", "precio", "cotizacion", "_lat", "_lng"} {
		assert.NotContains(t, string(raw), prohibido)
	}
	var s solicitud.SolicitudResponse
	require.NoError(t, json.Unmarshal(raw, &s))
	assert.Equal(t, "publicada", s.Estado)
	assert.Equal(t, "2026-10-10", s.FechaServicioDeseada)
	assert.Equal(t, "09:00", *s.FranjaHorariaInicio)
	assert.Equal(t, "San Isidro", s.Origen.ZonaNombre)
	assert.Equal(t, 2, s.Origen.Pisos)
	assert.Equal(t, "15.5", s.Origen.DistanciaVehiculoM.String())
	require.Len(t, s.Objetos, 2)

	heladera, piano := s.Objetos[0], s.Objetos[1]
	assert.Equal(t, "Heladera", heladera.Nombre)
	assert.Equal(t, "1.8", heladera.AltoM.String(), "copiado del catálogo")
	assert.False(t, heladera.RotacionVertical)
	assert.NotNil(t, heladera.ObjetoID)
	assert.Equal(t, "Piano vertical", piano.Nombre)
	assert.Equal(t, "250", piano.PesoUnitarioKg.String())
	assert.Equal(t, "1.5", piano.LargoM.String())
	assert.True(t, piano.RotacionHorizontal, "sin dato vale true")
	assert.False(t, piano.Apilable)

	var lat, lng string
	require.NoError(t, a.db.WithinTx(context.Background(), c, func(tx pgx.Tx) error {
		return tx.QueryRow(context.Background(), `SELECT origen_lat::text, destino_lng::text FROM solicitud WHERE id = $1`,
			s.ID).Scan(&lat, &lng)
	}))
	assert.True(t, strings.HasPrefix(lat, "-34.4"), "ubicada cerca de San Isidro: %s", lat)
	assert.True(t, strings.HasPrefix(lng, "-58.3"), "ubicada cerca de CABA: %s", lng)
}

func TestPublicarErrores(t *testing.T) {
	a := nuevaAPI(t, hoy)
	c := cliente(t)
	transportista := dbtest.CrearTransportista(t)

	con := func(cambiar func(m map[string]any)) map[string]any {
		m := a.cuerpo("2026-10-12")
		cambiar(m)
		return m
	}
	objetos := func(m map[string]any) []any { return m["objetos"].([]any) }

	tests := []struct {
		name      string
		id        database.Identity
		body      any
		wantCode  int
		wantErr   string
		wantCampo string
	}{
		{"sin objetos", c, con(func(m map[string]any) { m["objetos"] = []any{} }), 400, "datos_invalidos", "objetos"},
		{"fecha mal escrita", c, con(func(m map[string]any) { m["fecha_servicio_deseada"] = "12/10/2026" }), 400, "datos_invalidos", "fecha_servicio_deseada"},
		{"franja incompleta", c, con(func(m map[string]any) { delete(m, "franja_horaria_fin") }), 400, "datos_invalidos", "franja_horaria"},
		{"franja invertida", c, con(func(m map[string]any) { m["franja_horaria_fin"] = "08:00" }), 400, "datos_invalidos", "franja_horaria"},
		{"cuatro ayudantes", c, con(func(m map[string]any) { m["cantidad_ayudantes_solicitados"] = 4 }), 400, "datos_invalidos", "cantidad_ayudantes_solicitados"},
		{"pisos negativos", c, con(func(m map[string]any) { m["origen"].(map[string]any)["pisos"] = -1 }), 400, "datos_invalidos", "origen.pisos"},
		{"dirección corta", c, con(func(m map[string]any) { m["destino"].(map[string]any)["direccion"] = "X" }), 400, "datos_invalidos", "destino.direccion"},
		{"catálogo con medidas", c, con(func(m map[string]any) { objetos(m)[0].(map[string]any)["peso_unitario_kg"] = 1 }), 400, "datos_invalidos", "objetos[0]"},
		{"manual sin medidas", c, con(func(m map[string]any) { delete(objetos(m)[1].(map[string]any), "largo_m") }), 400, "datos_invalidos", "objetos[1].largo_m"},
		{"cantidad cero", c, con(func(m map[string]any) { objetos(m)[0].(map[string]any)["cantidad"] = 0 }), 400, "datos_invalidos", "objetos[0].cantidad"},
		{"con un monto", c, con(func(m map[string]any) { m["cotizacion_estimada_monto"] = 1000 }), 400, "json_invalido", ""},
		{"fecha pasada", c, a.cuerpo("2026-10-09"), 400, "fecha_pasada", ""},
		{"zona inexistente", c, con(func(m map[string]any) {
			m["origen"].(map[string]any)["zona_id"] = "8f1d2c3b-0000-4000-8000-0000000000aa"
		}), 400, "zona_invalida", ""},
		{"objeto inexistente", c, con(func(m map[string]any) {
			objetos(m)[0].(map[string]any)["objeto_id"] = "8f1d2c3b-0000-4000-8000-0000000000aa"
		}), 400, "objeto_invalido", ""},
		{"transportista", transportista, a.cuerpo("2026-10-12"), 403, "no_es_cliente", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			code, raw := a.llamar(tc.id, "POST", "/solicitudes", tc.body)
			assert.Equal(t, tc.wantCode, code, string(raw))
			e := errorDe(t, raw)
			assert.Equal(t, tc.wantErr, e.Code)
			if tc.wantCampo != "" {
				assert.Contains(t, e.Details, tc.wantCampo)
			}
		})
	}
}

func TestListarVencerCancelarYRepublicar(t *testing.T) {
	a := nuevaAPI(t, hoy)
	c := cliente(t)
	otro := cliente(t)
	s := a.crear(c, "2026-10-10")

	t.Run("otro cliente no la ve", func(t *testing.T) {
		code, raw := a.llamar(otro, "GET", "/solicitudes/"+s.ID, nil)
		assert.Equal(t, http.StatusNotFound, code)
		assert.Equal(t, "solicitud_no_encontrada", errorDe(t, raw).Code)
		code, raw = a.llamar(otro, "GET", "/solicitudes", nil)
		require.Equal(t, http.StatusOK, code)
		assert.JSONEq(t, `[]`, string(raw))
	})

	// Tres días después, la solicitud del 10 está vencida al leer.
	despues := nuevaAPIConDB(t, a.db, hoy.AddDate(0, 0, 3))
	code, raw := despues.llamar(c, "GET", "/solicitudes", nil)
	require.Equal(t, http.StatusOK, code)
	var lista []solicitud.SolicitudResumen
	require.NoError(t, json.Unmarshal(raw, &lista))
	require.Len(t, lista, 1)
	assert.Equal(t, "vencida", lista[0].Estado)
	assert.Equal(t, 2, lista[0].CantidadObjetos)

	t.Run("republicar", func(t *testing.T) {
		code, raw := a.llamar(c, "POST", "/solicitudes/"+s.ID+"/republicar", map[string]string{"fecha_servicio_deseada": "2026-10-20"})
		assert.Equal(t, http.StatusConflict, code, "el día 10 todavía no venció")
		assert.Equal(t, "solicitud_no_vencida", errorDe(t, raw).Code)

		code, raw = despues.llamar(c, "POST", "/solicitudes/"+s.ID+"/republicar", map[string]string{"fecha_servicio_deseada": "2026-10-12"})
		assert.Equal(t, http.StatusBadRequest, code)
		assert.Equal(t, "fecha_pasada", errorDe(t, raw).Code)

		code, raw = despues.llamar(c, "POST", "/solicitudes/"+s.ID+"/republicar", map[string]string{"fecha_servicio_deseada": "2026-10-20"})
		require.Equal(t, http.StatusCreated, code, string(raw))
		var nueva solicitud.SolicitudResponse
		require.NoError(t, json.Unmarshal(raw, &nueva))
		assert.NotEqual(t, s.ID, nueva.ID)
		assert.Equal(t, "publicada", nueva.Estado)
		assert.Equal(t, "2026-10-20", nueva.FechaServicioDeseada)
		assert.Nil(t, nueva.FranjaHorariaInicio)
		assert.Len(t, nueva.Objetos, 2, "copia los objetos")

		code, _ = despues.llamar(otro, "POST", "/solicitudes/"+s.ID+"/republicar", map[string]string{"fecha_servicio_deseada": "2026-10-20"})
		assert.Equal(t, http.StatusNotFound, code)
	})

	t.Run("cancelar", func(t *testing.T) {
		otra := a.crear(c, "2026-10-15")
		code, raw := a.llamar(otro, "POST", "/solicitudes/"+otra.ID+"/cancelar", nil)
		assert.Equal(t, http.StatusNotFound, code, string(raw))

		code, raw = a.llamar(c, "POST", "/solicitudes/"+otra.ID+"/cancelar", nil)
		require.Equal(t, http.StatusOK, code, string(raw))
		assert.Contains(t, string(raw), `"estado":"cancelada"`)

		code, raw = a.llamar(c, "POST", "/solicitudes/"+otra.ID+"/cancelar", nil)
		assert.Equal(t, http.StatusConflict, code)
		assert.Equal(t, "solicitud_no_cancelable", errorDe(t, raw).Code)
	})
}

// TestSinEdicionEnLaBase prueba la migración 0014 directo contra la base, como lo
// haría PostgREST con el JWT del Cliente.
func TestSinEdicionEnLaBase(t *testing.T) {
	a := nuevaAPI(t, hoy)
	c := cliente(t)
	s := a.crear(c, "2026-10-12")
	heladera := a.id(`SELECT id::text FROM objeto WHERE nombre = 'Heladera'`)
	ctx := context.Background()
	ejecutar := func(sql string, args ...any) error {
		return a.db.WithinTx(ctx, c, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, sql, args...)
			return err
		})
	}

	tests := []struct {
		name string
		sql  string
	}{
		{"cambiar la dirección", `UPDATE solicitud SET origen_direccion = 'Otra 123' WHERE id = $1`},
		{"pasarla a asignada", `UPDATE solicitud SET estado_codigo = 'asignada' WHERE id = $1`},
		{"cambiar un objeto", `UPDATE solicitud_objeto SET peso_unitario_kg = 1 WHERE solicitud_id = $1`},
		{"borrar un objeto", `DELETE FROM solicitud_objeto WHERE solicitud_id = $1`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Error(t, ejecutar(tc.sql, s.ID))
		})
	}

	t.Run("el alta directa nace publicada y el catálogo se copia", func(t *testing.T) {
		var estado, peso string
		require.NoError(t, a.db.WithinTx(ctx, c, func(tx pgx.Tx) error {
			var nuevo string
			err := tx.QueryRow(ctx, `
				INSERT INTO solicitud (cliente_id, origen_zona_id, destino_zona_id, origen_direccion, destino_direccion,
				    origen_lat, origen_lng, destino_lat, destino_lng, fecha_servicio_deseada, estado_codigo)
				SELECT (select auth.uid()), origen_zona_id, destino_zona_id, 'a', 'b', 0, 0, 0, 0, '2026-10-12', 'asignada'
				FROM solicitud WHERE id = $1
				RETURNING id::text, estado_codigo`, s.ID).Scan(&nuevo, &estado)
			if err != nil {
				return err
			}
			return tx.QueryRow(ctx, `
				INSERT INTO solicitud_objeto (solicitud_id, objeto_id, cantidad, peso_unitario_kg, largo_m, ancho_m, alto_m)
				VALUES ($1, $2, 1, 1, 0.1, 0.1, 0.1) RETURNING peso_unitario_kg::text`, nuevo, heladera).Scan(&peso)
		}))
		assert.Equal(t, "publicada", estado)
		assert.NotEqual(t, "1.00", peso, "el peso falso se reemplaza por el del catálogo")
	})

	t.Run("no agrega objetos a una cancelada", func(t *testing.T) {
		code, _ := a.llamar(c, "POST", "/solicitudes/"+s.ID+"/cancelar", nil)
		require.Equal(t, http.StatusOK, code)
		require.Error(t, ejecutar(`INSERT INTO solicitud_objeto (solicitud_id, nombre_personalizado, cantidad,
			peso_unitario_kg, largo_m, ancho_m, alto_m) VALUES ($1, 'Caja', 1, 5, 0.5, 0.5, 0.5)`, s.ID))
	})
}
