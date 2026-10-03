package identidad_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/matexs/fletway-backend/internal/feature/identidad"
	"github.com/matexs/fletway-backend/internal/platform/auth"
	"github.com/matexs/fletway-backend/internal/platform/database"
	"github.com/matexs/fletway-backend/internal/platform/database/dbtest"
	"github.com/matexs/fletway-backend/internal/platform/httpx"
)

// api arma el mux de la feature con la identidad ya en el contexto, como la deja el
// middleware Auth (que tiene sus propios tests).
type api struct {
	t   *testing.T
	mux *http.ServeMux
}

func nuevaAPI(t *testing.T, db *database.DB) *api {
	t.Helper()
	mux := http.NewServeMux()
	identidad.Register(mux, db)
	return &api{t: t, mux: mux}
}

func (a *api) llamar(id database.Identity, metodo, path string) (int, map[string]any) {
	a.t.Helper()
	r := httptest.NewRequest(metodo, path, nil)
	r = r.WithContext(auth.WithIdentity(r.Context(), id))
	w := httptest.NewRecorder()
	a.mux.ServeHTTP(w, r)
	var body map[string]any
	require.NoError(a.t, json.Unmarshal(w.Body.Bytes(), &body))
	return w.Code, body
}

func codigoDeError(t *testing.T, body map[string]any) string {
	t.Helper()
	var e httpx.ErrorBody
	raw, err := json.Marshal(body)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, &e))
	return e.Error.Code
}

func TestRegistroYMe(t *testing.T) {
	db := dbtest.Abrir(t)
	a := nuevaAPI(t, db)

	tests := []struct {
		name       string
		rol        string
		path       string
		wantEstado any
	}{
		{"cliente", identidad.RolCliente, "/auth/registro/cliente", nil},
		{"transportista queda pendiente", identidad.RolTransportista, "/auth/registro/transportista", "pendiente"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			id := dbtest.CrearUsuario(t, tc.rol)

			code, body := a.llamar(id, "GET", "/me")
			require.Equal(t, http.StatusOK, code)
			assert.Equal(t, id.UserID, body["usuario_id"])
			assert.Equal(t, tc.rol, body["rol"])
			assert.Equal(t, "Usuario de Prueba", body["nombre_completo"])
			assert.Equal(t, false, body["registro_completo"])
			assert.Nil(t, body["estado_habilitacion"])

			code, body = a.llamar(id, "POST", tc.path)
			require.Equal(t, http.StatusCreated, code)
			assert.Equal(t, true, body["registro_completo"])
			assert.Equal(t, tc.wantEstado, body["estado_habilitacion"])

			// Idempotente: reintentar no falla ni duplica.
			code, body = a.llamar(id, "POST", tc.path)
			require.Equal(t, http.StatusOK, code)
			assert.Equal(t, true, body["registro_completo"])

			code, body = a.llamar(id, "GET", "/me")
			require.Equal(t, http.StatusOK, code)
			assert.Equal(t, true, body["registro_completo"])
		})
	}
}

func TestRegistroErrores(t *testing.T) {
	db := dbtest.Abrir(t)
	a := nuevaAPI(t, db)

	cliente := dbtest.CrearUsuario(t, identidad.RolCliente)
	transportista := dbtest.CrearUsuario(t, identidad.RolTransportista)
	inactivo := dbtest.CrearUsuario(t, identidad.RolCliente)
	dbtest.Exec(t, `UPDATE usuario SET activo = false WHERE id = $1`, inactivo.UserID)
	sinUsuario := dbtest.Identidad(t, "8f1d2c3b-0000-4000-8000-0000000000ff")

	tests := []struct {
		name     string
		id       database.Identity
		metodo   string
		path     string
		wantCode int
		wantErr  string
	}{
		{"cliente se registra como transportista", cliente, "POST", "/auth/registro/transportista", http.StatusConflict, "rol_no_corresponde"},
		{"transportista se registra como cliente", transportista, "POST", "/auth/registro/cliente", http.StatusConflict, "rol_no_corresponde"},
		{"cuenta inactiva", inactivo, "POST", "/auth/registro/cliente", http.StatusForbidden, "cuenta_inactiva"},
		{"me sin fila usuario", sinUsuario, "GET", "/me", http.StatusNotFound, "usuario_no_encontrado"},
		{"registro sin fila usuario", sinUsuario, "POST", "/auth/registro/cliente", http.StatusNotFound, "usuario_no_encontrado"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			code, body := a.llamar(tc.id, tc.metodo, tc.path)
			assert.Equal(t, tc.wantCode, code)
			assert.Equal(t, tc.wantErr, codigoDeError(t, body))
		})
	}
}

func TestMeAdministrador(t *testing.T) {
	db := dbtest.Abrir(t)
	a := nuevaAPI(t, db)

	// Bootstrap como el de D-18: alta común y después rol administrador por SQL, sin JWT.
	id := dbtest.CrearUsuario(t, identidad.RolCliente)
	dbtest.Exec(t, `UPDATE usuario SET rol = 'administrador' WHERE id = $1`, id.UserID)
	dbtest.Exec(t, `INSERT INTO administrador (usuario_id) VALUES ($1)`, id.UserID)

	code, body := a.llamar(id, "GET", "/me")
	require.Equal(t, http.StatusOK, code)
	assert.Equal(t, identidad.RolAdministrador, body["rol"])
	assert.Equal(t, true, body["registro_completo"])

	code, body = a.llamar(id, "POST", "/auth/registro/cliente")
	assert.Equal(t, http.StatusConflict, code)
	assert.Equal(t, "rol_no_corresponde", codigoDeError(t, body))
}

// --- Seguridad en la base (migraciones 0010 y 0011) ---

func TestAltaRechazaRolesNoPermitidos(t *testing.T) {
	dbtest.URL(t)
	tests := []struct {
		name string
		meta string
	}{
		{"administrador", `{"rol":"administrador","nombre_completo":"X","telefono":"1"}`},
		{"rol inventado", `{"rol":"superusuario","nombre_completo":"X","telefono":"1"}`},
		{"sin rol", `{"nombre_completo":"X","telefono":"1"}`},
		{"sin telefono", `{"rol":"cliente","nombre_completo":"X"}`},
		{"nombre vacio", `{"rol":"cliente","nombre_completo":"  ","telefono":"1"}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := dbtest.Intentar(t, `INSERT INTO auth.users (id, email, aud, role, raw_user_meta_data)
				VALUES (gen_random_uuid(), gen_random_uuid() || '@prueba.fletway.local', 'authenticated', 'authenticated', $1)`, tc.meta)
			require.Error(t, err, "el alta tenía que fallar")
		})
	}
}

func TestUsuarioNoPuedeEscalarPrivilegios(t *testing.T) {
	db := dbtest.Abrir(t)
	ctx := context.Background()
	transportista := dbtest.CrearUsuario(t, identidad.RolTransportista)
	otroCliente := dbtest.CrearUsuario(t, identidad.RolCliente)

	t.Run("no cambia su rol, email ni activo", func(t *testing.T) {
		var rol, email string
		var activo bool
		err := db.WithinTx(ctx, transportista, func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, `UPDATE usuario SET rol = 'administrador', email = 'otro@x.com',
				activo = false, nombre_completo = 'Nombre Nuevo' WHERE id = (select auth.uid())`); err != nil {
				return err
			}
			return tx.QueryRow(ctx, `SELECT rol, email::text, activo FROM usuario WHERE id = (select auth.uid())`).
				Scan(&rol, &email, &activo)
		})
		require.NoError(t, err)
		assert.Equal(t, identidad.RolTransportista, rol)
		assert.NotEqual(t, "otro@x.com", email)
		assert.True(t, activo)
	})

	t.Run("alta directa de transportista queda pendiente", func(t *testing.T) {
		var estado string
		err := db.WithinTx(ctx, transportista, func(tx pgx.Tx) error {
			return tx.QueryRow(ctx, `INSERT INTO transportista (usuario_id, estado_habilitacion_codigo, calificacion_promedio)
				VALUES ((select auth.uid()), 'habilitado', 5) RETURNING estado_habilitacion_codigo`).Scan(&estado)
		})
		require.NoError(t, err)
		assert.Equal(t, "pendiente", estado)
	})

	t.Run("no crea fila de otro rol", func(t *testing.T) {
		err := db.WithinTx(ctx, transportista, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, `INSERT INTO cliente (usuario_id) VALUES ((select auth.uid()))`)
			return err
		})
		require.Error(t, err)
	})

	t.Run("no crea fila de rol de otro usuario", func(t *testing.T) {
		err := db.WithinTx(ctx, transportista, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, `INSERT INTO cliente (usuario_id) VALUES ($1)`, otroCliente.UserID)
			return err
		})
		require.Error(t, err)
	})
}

func TestPoliciesSinRecursion(t *testing.T) {
	db := dbtest.Abrir(t)
	id := dbtest.CrearUsuario(t, identidad.RolCliente)
	for _, tabla := range []string{"usuario", "cliente", "solicitud", "oferta", "solicitud_objeto"} {
		t.Run(tabla, func(t *testing.T) {
			err := db.WithinTx(context.Background(), id, func(tx pgx.Tx) error {
				_, err := tx.Exec(context.Background(), "SELECT count(*) FROM "+pgx.Identifier{tabla}.Sanitize())
				return err
			})
			require.NoError(t, err)
		})
	}
}
