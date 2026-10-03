package vehiculo_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/matexs/fletway-backend/internal/feature/vehiculo"
	"github.com/matexs/fletway-backend/internal/platform/auth"
	"github.com/matexs/fletway-backend/internal/platform/database"
	"github.com/matexs/fletway-backend/internal/platform/database/dbtest"
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
	vehiculo.Register(mux, db)
	return &api{t: t, db: db, mux: mux}
}

// llamar manda body tal cual si es string (para probar JSON crudo) o lo
// serializa si no.
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

func (a *api) tipoID(nombre string) string {
	a.t.Helper()
	code, raw := a.llamar(dbtest.CrearUsuario(a.t, "cliente"), "GET", "/tipos-vehiculo", nil)
	require.Equal(a.t, http.StatusOK, code)
	var tipos []vehiculo.TipoVehiculoResponse
	require.NoError(a.t, json.Unmarshal(raw, &tipos))
	for _, t := range tipos {
		if t.Nombre == nombre {
			return t.ID
		}
	}
	a.t.Fatalf("no existe el tipo %q", nombre)
	return ""
}

// nuevaPatente devuelve una patente Mercosur al azar: la patente es única en la
// tabla y los tests corren contra la misma base.
func nuevaPatente() string {
	const letras = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, 7)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(26))
		if err != nil {
			panic(err)
		}
		if i >= 2 && i <= 4 {
			b[i] = byte('0' + n.Int64()%10)
		} else {
			b[i] = letras[n.Int64()]
		}
	}
	return string(b)
}

// cuerpoVehiculo es un alta válida de un Furgón chico.
func cuerpoVehiculo(tipoID, patente string) string {
	return `{"tipo_vehiculo_id":"` + tipoID + `","patente":"` + patente + `","marca":"Renault","modelo":"Kangoo",
		"largo_util_m":2.40,"ancho_util_m":1.45,"alto_util_m":1.15,"peso_maximo_kg":650}`
}

func (a *api) crear(id database.Identity, tipoID string) vehiculo.VehiculoResponse {
	a.t.Helper()
	code, raw := a.llamar(id, "POST", "/transportista/vehiculos", cuerpoVehiculo(tipoID, nuevaPatente()))
	require.Equal(a.t, http.StatusCreated, code, string(raw))
	var v vehiculo.VehiculoResponse
	require.NoError(a.t, json.Unmarshal(raw, &v))
	return v
}

const costosValidos = `{"combustible_precio_l":1350.50,"rendimiento_km_l":9.5,"cantidad_neumaticos":4,
	"costo_neumatico":185000,"vida_neumatico_km":50000,"costo_mantenimiento_km":45.75,
	"valor_compra":28000000,"valor_residual":9000000,"vida_util_km":400000,
	"seguro_mensual":95000,"patente_mensual":38000}`

func TestTipos(t *testing.T) {
	a := nuevaAPI(t)
	code, raw := a.llamar(dbtest.CrearUsuario(t, "cliente"), "GET", "/tipos-vehiculo", nil)
	require.Equal(t, http.StatusOK, code)
	var tipos []vehiculo.TipoVehiculoResponse
	require.NoError(t, json.Unmarshal(raw, &tipos))
	require.Len(t, tipos, 6)
	assert.Equal(t, "Utilitario", tipos[0].Nombre)
	assert.Equal(t, "1.6", tipos[0].LargoEstandarM.String())
	assert.Equal(t, "Camión grande", tipos[5].Nombre)
	assert.Contains(t, string(raw), `"peso_maximo_estandar_kg":7000`)
}

func TestCrearYListar(t *testing.T) {
	a := nuevaAPI(t)
	transportista := dbtest.CrearTransportista(t)
	tipo := a.tipoID("Furgón chico")

	// Patente al azar escrita en minúscula, con espacio y guion (ej. "ab 123-cd").
	patente := nuevaPatente()
	escrita := strings.ToLower(patente[:2]) + " " + patente[2:5] + "-" + strings.ToLower(patente[5:])
	code, raw := a.llamar(transportista, "POST", "/transportista/vehiculos", cuerpoVehiculo(tipo, escrita))
	require.Equal(t, http.StatusCreated, code, string(raw))
	var v vehiculo.VehiculoResponse
	require.NoError(t, json.Unmarshal(raw, &v))
	assert.Equal(t, patente, v.Patente, "la patente se normaliza")
	assert.Equal(t, "Furgón chico", v.TipoVehiculoNombre)
	assert.Equal(t, "2.4", v.LargoUtilM.String())
	assert.True(t, v.Activo)
	assert.False(t, v.TieneCostos)

	t.Run("patente duplicada", func(t *testing.T) {
		otro := dbtest.CrearTransportista(t)
		code, raw := a.llamar(otro, "POST", "/transportista/vehiculos", cuerpoVehiculo(tipo, patente))
		assert.Equal(t, http.StatusConflict, code)
		assert.Equal(t, "patente_duplicada", errorDe(t, raw).Code)
	})

	t.Run("listar propios", func(t *testing.T) {
		code, raw := a.llamar(transportista, "GET", "/transportista/vehiculos", nil)
		require.Equal(t, http.StatusOK, code)
		var lista []vehiculo.VehiculoResponse
		require.NoError(t, json.Unmarshal(raw, &lista))
		require.Len(t, lista, 1)
		assert.Equal(t, v.ID, lista[0].ID)

		code, raw = a.llamar(dbtest.CrearTransportista(t), "GET", "/transportista/vehiculos", nil)
		require.Equal(t, http.StatusOK, code)
		assert.JSONEq(t, `[]`, string(raw), "otro Transportista no ve vehículos ajenos en su lista")
	})
}

func TestCrearErrores(t *testing.T) {
	a := nuevaAPI(t)
	transportista := dbtest.CrearTransportista(t)
	cliente := dbtest.CrearUsuario(t, "cliente")
	tipo := a.tipoID("Utilitario")

	tests := []struct {
		name      string
		id        database.Identity
		body      string
		wantCode  int
		wantErr   string
		wantCampo string
	}{
		{"patente inválida", transportista, cuerpoVehiculo(tipo, "1234"), 400, "datos_invalidos", "patente"},
		{"tipo inexistente", transportista, cuerpoVehiculo("8f1d2c3b-0000-4000-8000-0000000000aa", nuevaPatente()), 400, "tipo_vehiculo_invalido", ""},
		{"tipo mal formado", transportista, cuerpoVehiculo("abc", nuevaPatente()), 400, "tipo_vehiculo_invalido", ""},
		{"largo cero", transportista, strings.Replace(cuerpoVehiculo(tipo, nuevaPatente()), `"largo_util_m":2.40`, `"largo_util_m":0`, 1), 400, "datos_invalidos", "largo_util_m"},
		{"ancho mayor a 3 m", transportista, strings.Replace(cuerpoVehiculo(tipo, nuevaPatente()), `"ancho_util_m":1.45`, `"ancho_util_m":3.5`, 1), 400, "datos_invalidos", "ancho_util_m"},
		{"tres decimales", transportista, strings.Replace(cuerpoVehiculo(tipo, nuevaPatente()), `"alto_util_m":1.15`, `"alto_util_m":1.155`, 1), 400, "datos_invalidos", "alto_util_m"},
		{"número como texto", transportista, strings.Replace(cuerpoVehiculo(tipo, nuevaPatente()), `"peso_maximo_kg":650`, `"peso_maximo_kg":"650"`, 1), 400, "json_invalido", ""},
		{"cliente", cliente, cuerpoVehiculo(tipo, nuevaPatente()), 403, "no_es_transportista", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			code, raw := a.llamar(tc.id, "POST", "/transportista/vehiculos", tc.body)
			assert.Equal(t, tc.wantCode, code, string(raw))
			e := errorDe(t, raw)
			assert.Equal(t, tc.wantErr, e.Code)
			if tc.wantCampo != "" {
				assert.Contains(t, e.Details, tc.wantCampo)
			}
		})
	}
}

func TestActivo(t *testing.T) {
	a := nuevaAPI(t)
	transportista := dbtest.CrearTransportista(t)
	v := a.crear(transportista, a.tipoID("Camión chico"))

	code, raw := a.llamar(transportista, "PUT", "/transportista/vehiculos/"+v.ID+"/activo", map[string]bool{"activo": false})
	require.Equal(t, http.StatusOK, code, string(raw))
	assert.Contains(t, string(raw), `"activo":false`)

	code, raw = a.llamar(dbtest.CrearTransportista(t), "PUT", "/transportista/vehiculos/"+v.ID+"/activo", map[string]bool{"activo": true})
	assert.Equal(t, http.StatusNotFound, code)
	assert.Equal(t, "vehiculo_no_encontrado", errorDe(t, raw).Code)

	code, raw = a.llamar(transportista, "PUT", "/transportista/vehiculos/"+v.ID+"/activo", `{}`)
	assert.Equal(t, http.StatusBadRequest, code)
	assert.Equal(t, "datos_incompletos", errorDe(t, raw).Code)
}

func TestCostos(t *testing.T) {
	a := nuevaAPI(t)
	transportista := dbtest.CrearTransportista(t)
	v := a.crear(transportista, a.tipoID("Furgón grande"))
	ruta := "/transportista/vehiculos/" + v.ID + "/costos"

	code, raw := a.llamar(transportista, "GET", ruta, nil)
	assert.Equal(t, http.StatusNotFound, code)
	assert.Equal(t, "costos_no_cargados", errorDe(t, raw).Code)

	code, raw = a.llamar(transportista, "PUT", ruta, costosValidos)
	require.Equal(t, http.StatusOK, code, string(raw))
	var c vehiculo.CostosResponse
	require.NoError(t, json.Unmarshal(raw, &c))
	assert.Equal(t, "1350.5", c.CombustiblePrecioL.String(), "los montos se guardan exactos")
	assert.Equal(t, "45.75", c.CostoMantenimientoKm.String())

	// Reemplazo: el segundo PUT actualiza la misma fila.
	code, raw = a.llamar(transportista, "PUT", ruta, strings.Replace(costosValidos, "1350.50", "1400", 1))
	require.Equal(t, http.StatusOK, code, string(raw))
	code, raw = a.llamar(transportista, "GET", ruta, nil)
	require.Equal(t, http.StatusOK, code)
	assert.Contains(t, string(raw), `"combustible_precio_l":1400`)

	code, raw = a.llamar(transportista, "GET", "/transportista/vehiculos", nil)
	require.Equal(t, http.StatusOK, code)
	assert.Contains(t, string(raw), `"tiene_costos":true`)

	tests := []struct {
		name      string
		body      string
		wantCampo string
	}{
		{"residual mayor a compra", strings.Replace(costosValidos, `"valor_residual":9000000`, `"valor_residual":30000000`, 1), "valor_residual"},
		{"rendimiento cero", strings.Replace(costosValidos, `"rendimiento_km_l":9.5`, `"rendimiento_km_l":0`, 1), "rendimiento_km_l"},
		{"neumáticos cero", strings.Replace(costosValidos, `"cantidad_neumaticos":4`, `"cantidad_neumaticos":0`, 1), "cantidad_neumaticos"},
		{"km con decimales", strings.Replace(costosValidos, `"vida_util_km":400000`, `"vida_util_km":400000.5`, 1), "vida_util_km"},
		{"monto negativo", strings.Replace(costosValidos, `"seguro_mensual":95000`, `"seguro_mensual":-1`, 1), "seguro_mensual"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			code, raw := a.llamar(transportista, "PUT", ruta, tc.body)
			assert.Equal(t, http.StatusBadRequest, code, string(raw))
			e := errorDe(t, raw)
			assert.Equal(t, "datos_invalidos", e.Code)
			assert.Contains(t, e.Details, tc.wantCampo)
		})
	}

	t.Run("otro transportista no los toca", func(t *testing.T) {
		otro := dbtest.CrearTransportista(t)
		code, raw := a.llamar(otro, "PUT", ruta, costosValidos)
		assert.Equal(t, http.StatusNotFound, code)
		assert.Equal(t, "vehiculo_no_encontrado", errorDe(t, raw).Code)
		code, _ = a.llamar(otro, "GET", ruta, nil)
		assert.Equal(t, http.StatusNotFound, code)
	})
}

// TestCostosSoloDuenoYAdministrador es el criterio de terminado del módulo 4:
// nadie más que el dueño y el Administrador lee vehiculo_costo, ni por la API ni
// directo contra la base (PostgREST).
func TestCostosSoloDuenoYAdministrador(t *testing.T) {
	a := nuevaAPI(t)
	duenio := dbtest.CrearTransportista(t)
	otro := dbtest.CrearTransportista(t)
	ctx := context.Background()
	v := a.crear(duenio, a.tipoID("Utilitario"))
	code, raw := a.llamar(duenio, "PUT", "/transportista/vehiculos/"+v.ID+"/costos", costosValidos)
	require.Equal(t, http.StatusOK, code, string(raw))

	contar := func(id database.Identity) int {
		var n int
		require.NoError(t, a.db.WithinTx(ctx, id, func(tx pgx.Tx) error {
			return tx.QueryRow(ctx, `SELECT count(*) FROM vehiculo_costo WHERE vehiculo_id = $1`, v.ID).Scan(&n)
		}))
		return n
	}
	assert.Equal(t, 1, contar(duenio), "el dueño")
	assert.Equal(t, 1, contar(dbtest.CrearAdministrador(t)), "el Administrador")
	assert.Equal(t, 0, contar(otro), "otro Transportista")
	assert.Equal(t, 0, contar(dbtest.CrearUsuario(t, "cliente")), "un Cliente")

	t.Run("nadie más los modifica", func(t *testing.T) {
		require.NoError(t, a.db.WithinTx(ctx, otro, func(tx pgx.Tx) error {
			tag, err := tx.Exec(ctx, `UPDATE vehiculo_costo SET combustible_precio_l = 1 WHERE vehiculo_id = $1`, v.ID)
			assert.Zero(t, tag.RowsAffected())
			return err
		}))
	})
}
