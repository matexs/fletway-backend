package habilitacion_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/matexs/fletway-backend/internal/feature/habilitacion"
	"github.com/matexs/fletway-backend/internal/feature/notificacion"
	"github.com/matexs/fletway-backend/internal/platform/async"
	"github.com/matexs/fletway-backend/internal/platform/auth"
	"github.com/matexs/fletway-backend/internal/platform/database"
	"github.com/matexs/fletway-backend/internal/platform/database/dbtest"
	"github.com/matexs/fletway-backend/internal/platform/httpx"
)

const bucket = "documentos-transportista"

var tipos = []string{"dni", "registro", "seguro", "vtv"}

// firmadorFalso devuelve una URL reconocible sin llamar a Storage.
type firmadorFalso struct{}

func (firmadorFalso) FirmarURL(_ context.Context, _, b, path string, _ time.Duration) (string, error) {
	return "https://firmada/" + b + "/" + path, nil
}

// jobsInmediatos corre cada job en el momento, para poder verificar su efecto.
type jobsInmediatos struct{ t *testing.T }

func (j jobsInmediatos) Enqueue(job async.Job) bool {
	require.NoError(j.t, job.Run(context.Background()))
	return true
}

type api struct {
	t   *testing.T
	db  *database.DB
	mux *http.ServeMux
}

func nuevaAPI(t *testing.T) *api {
	t.Helper()
	db := dbtest.Abrir(t)
	mux := http.NewServeMux()
	habilitacion.Register(mux, habilitacion.NewService(db, firmadorFalso{}, notificacion.NewInApp(db), jobsInmediatos{t}))
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

func codigoDeError(t *testing.T, raw json.RawMessage) string {
	t.Helper()
	var e httpx.ErrorBody
	require.NoError(t, json.Unmarshal(raw, &e))
	return e.Error.Code
}

// cargar sube el archivo y registra el documento; devuelve el id.
func (a *api) cargar(id database.Identity, tipo string) string {
	a.t.Helper()
	path := "transportista/" + id.UserID + "/" + tipo + "-" + time.Now().Format("150405.000000000") + ".pdf"
	dbtest.SubirArchivo(a.t, bucket, path)
	code, raw := a.llamar(id, "POST", "/transportista/documentos",
		map[string]string{"tipo_documento_codigo": tipo, "path": path})
	require.Equal(a.t, http.StatusCreated, code, string(raw))
	var d habilitacion.DocumentoResponse
	require.NoError(a.t, json.Unmarshal(raw, &d))
	assert.Equal(a.t, "pendiente", d.Estado)
	return d.ID
}

func (a *api) revisar(admin database.Identity, docID, accion, motivo string) string {
	a.t.Helper()
	var body any
	if accion == "rechazar" {
		body = map[string]string{"motivo": motivo}
	}
	code, raw := a.llamar(admin, "POST", "/admin/documentos/"+docID+"/"+accion, body)
	require.Equal(a.t, http.StatusOK, code, string(raw))
	var r habilitacion.RevisionResponse
	require.NoError(a.t, json.Unmarshal(raw, &r))
	return r.EstadoHabilitacion
}

func (a *api) estado(id database.Identity) string {
	a.t.Helper()
	code, raw := a.llamar(id, "GET", "/transportista/documentos", nil)
	require.Equal(a.t, http.StatusOK, code, string(raw))
	var r habilitacion.MiHabilitacionResponse
	require.NoError(a.t, json.Unmarshal(raw, &r))
	return r.EstadoHabilitacion
}

func (a *api) notificaciones(id database.Identity) []string {
	a.t.Helper()
	var out []string
	require.NoError(a.t, a.db.WithinTx(context.Background(), id, func(tx pgx.Tx) error {
		rows, err := tx.Query(context.Background(), `
			SELECT contenido FROM notificacion
			WHERE destinatario_usuario_id = (select auth.uid()) AND tipo_notificacion_codigo = 'documentacion_revisada'
			ORDER BY creado_en`)
		if err != nil {
			return err
		}
		out, err = pgx.CollectRows(rows, pgx.RowTo[string])
		return err
	}))
	return out
}

// TestFlujoDeHabilitacion recorre D-33: rechazo inmediato, nueva carga, habilitación
// con los cuatro aprobados y renovación de un habilitado.
func TestFlujoDeHabilitacion(t *testing.T) {
	a := nuevaAPI(t)
	transportista := dbtest.CrearTransportista(t)
	admin := dbtest.CrearAdministrador(t)

	docs := map[string]string{}
	for _, tipo := range tipos {
		docs[tipo] = a.cargar(transportista, tipo)
	}
	assert.Equal(t, "pendiente", a.estado(transportista))

	assert.Equal(t, "pendiente", a.revisar(admin, docs["dni"], "aprobar", ""))
	assert.Equal(t, "pendiente", a.revisar(admin, docs["registro"], "aprobar", ""))
	assert.Equal(t, "pendiente", a.revisar(admin, docs["seguro"], "aprobar", ""))
	assert.Empty(t, a.notificaciones(transportista), "aprobar documentos sueltos no notifica")

	// Rechazo inmediato, con el motivo en la notificación.
	assert.Equal(t, "rechazado", a.revisar(admin, docs["vtv"], "rechazar", "La VTV está vencida"))
	assert.Equal(t, []string{"Tu documentación fue rechazada. Motivo: La VTV está vencida"}, a.notificaciones(transportista))

	// Vuelve a cargar: pendiente otra vez. Sin límite de reintentos (RF-01).
	vtv := a.cargar(transportista, "vtv")
	assert.Equal(t, "pendiente", a.estado(transportista))

	// Los cuatro aprobados: habilitado y notificado.
	assert.Equal(t, "habilitado", a.revisar(admin, vtv, "aprobar", ""))
	assert.Len(t, a.notificaciones(transportista), 2)
	assert.Equal(t, "Tu documentación fue aprobada.", a.notificaciones(transportista)[1])

	// Renovación: sigue habilitado mientras el documento nuevo espera revisión.
	seguro := a.cargar(transportista, "seguro")
	assert.Equal(t, "habilitado", a.estado(transportista))
	assert.Equal(t, "rechazado", a.revisar(admin, seguro, "rechazar", "Póliza ilegible"))
}

func TestCargarDocumentoErrores(t *testing.T) {
	a := nuevaAPI(t)
	transportista := dbtest.CrearTransportista(t)
	otro := dbtest.CrearTransportista(t)
	cliente := dbtest.CrearUsuario(t, "cliente")

	propio := "transportista/" + transportista.UserID + "/dni.pdf"
	dbtest.SubirArchivo(t, bucket, propio)
	ajeno := "transportista/" + otro.UserID + "/dni.pdf"
	dbtest.SubirArchivo(t, bucket, ajeno)

	tests := []struct {
		name     string
		id       database.Identity
		body     any
		wantCode int
		wantErr  string
	}{
		{"sin datos", transportista, map[string]string{}, http.StatusBadRequest, "datos_incompletos"},
		{"campo desconocido", transportista, map[string]string{"estado": "aprobado"}, http.StatusBadRequest, "json_invalido"},
		{"tipo inexistente", transportista, map[string]string{"tipo_documento_codigo": "pasaporte", "path": propio}, http.StatusBadRequest, "tipo_documento_invalido"},
		{"path de otro transportista", transportista, map[string]string{"tipo_documento_codigo": "dni", "path": ajeno}, http.StatusBadRequest, "path_invalido"},
		{"path con ..", transportista, map[string]string{"tipo_documento_codigo": "dni", "path": "transportista/" + transportista.UserID + "/../x.pdf"}, http.StatusBadRequest, "path_invalido"},
		{"archivo no subido", transportista, map[string]string{"tipo_documento_codigo": "dni", "path": "transportista/" + transportista.UserID + "/no-existe.pdf"}, http.StatusBadRequest, "archivo_no_encontrado"},
		{"cliente", cliente, map[string]string{"tipo_documento_codigo": "dni", "path": "transportista/" + cliente.UserID + "/dni.pdf"}, http.StatusForbidden, "no_es_transportista"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			code, raw := a.llamar(tc.id, "POST", "/transportista/documentos", tc.body)
			assert.Equal(t, tc.wantCode, code, string(raw))
			assert.Equal(t, tc.wantErr, codigoDeError(t, raw))
		})
	}

	t.Run("cliente no tiene habilitación", func(t *testing.T) {
		code, raw := a.llamar(cliente, "GET", "/transportista/documentos", nil)
		assert.Equal(t, http.StatusForbidden, code)
		assert.Equal(t, "no_es_transportista", codigoDeError(t, raw))
	})
}

func TestAdministrador(t *testing.T) {
	a := nuevaAPI(t)
	transportista := dbtest.CrearTransportista(t)
	admin := dbtest.CrearAdministrador(t)
	docID := a.cargar(transportista, "dni")

	t.Run("lista pendientes con URL firmada", func(t *testing.T) {
		code, raw := a.llamar(admin, "GET", "/admin/transportistas", nil)
		require.Equal(t, http.StatusOK, code, string(raw))
		var lista []habilitacion.TransportistaRevisionResponse
		require.NoError(t, json.Unmarshal(raw, &lista))
		var encontrado *habilitacion.TransportistaRevisionResponse
		for i := range lista {
			if lista[i].UsuarioID == transportista.UserID {
				encontrado = &lista[i]
			}
		}
		require.NotNil(t, encontrado)
		require.Len(t, encontrado.Documentos, 4)
		dni := encontrado.Documentos[0]
		assert.Equal(t, "dni", dni.TipoDocumentoCodigo)
		require.NotNil(t, dni.Ultimo)
		assert.Contains(t, dni.Ultimo.URL, "https://firmada/documentos-transportista/transportista/"+transportista.UserID)
		assert.Nil(t, encontrado.Documentos[1].Ultimo, "registro todavía no se cargó")
	})

	t.Run("el transportista no ve URLs", func(t *testing.T) {
		_, raw := a.llamar(transportista, "GET", "/transportista/documentos", nil)
		assert.NotContains(t, string(raw), "firmada")
	})

	tests := []struct {
		name     string
		id       database.Identity
		metodo   string
		path     string
		body     any
		wantCode int
		wantErr  string
	}{
		{"transportista no lista", transportista, "GET", "/admin/transportistas", nil, http.StatusForbidden, "requiere_administrador"},
		{"transportista no se aprueba", transportista, "POST", "/admin/documentos/" + docID + "/aprobar", nil, http.StatusForbidden, "requiere_administrador"},
		{"estado inválido", admin, "GET", "/admin/transportistas?estado=aprobado", nil, http.StatusBadRequest, "estado_invalido"},
		{"rechazo sin motivo", admin, "POST", "/admin/documentos/" + docID + "/rechazar", map[string]string{"motivo": " "}, http.StatusBadRequest, "motivo_requerido"},
		{"id mal formado", admin, "POST", "/admin/documentos/abc/aprobar", nil, http.StatusNotFound, "documento_no_encontrado"},
		{"id inexistente", admin, "POST", "/admin/documentos/8f1d2c3b-0000-4000-8000-0000000000aa/aprobar", nil, http.StatusNotFound, "documento_no_encontrado"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			code, raw := a.llamar(tc.id, tc.metodo, tc.path, tc.body)
			assert.Equal(t, tc.wantCode, code, string(raw))
			assert.Equal(t, tc.wantErr, codigoDeError(t, raw))
		})
	}

	t.Run("no se revisa dos veces", func(t *testing.T) {
		a.revisar(admin, docID, "aprobar", "")
		code, raw := a.llamar(admin, "POST", "/admin/documentos/"+docID+"/rechazar", map[string]string{"motivo": "x"})
		assert.Equal(t, http.StatusConflict, code)
		assert.Equal(t, "documento_ya_revisado", codigoDeError(t, raw))
	})

	t.Run("filtro por estado", func(t *testing.T) {
		code, raw := a.llamar(admin, "GET", "/admin/transportistas?estado=habilitado", nil)
		require.Equal(t, http.StatusOK, code)
		assert.NotContains(t, string(raw), transportista.UserID)
	})
}

// TestSeguridadEnLaBase prueba que la base no deja saltear la revisión por fuera
// de la API (PostgREST con el JWT del Transportista).
func TestSeguridadEnLaBase(t *testing.T) {
	a := nuevaAPI(t)
	ctx := context.Background()
	transportista := dbtest.CrearTransportista(t)
	otro := dbtest.CrearTransportista(t)
	docID := a.cargar(transportista, "dni")

	t.Run("alta directa aprobada queda pendiente", func(t *testing.T) {
		var estado string
		err := a.db.WithinTx(ctx, transportista, func(tx pgx.Tx) error {
			return tx.QueryRow(ctx, `
				INSERT INTO documento_transportista (transportista_id, tipo_documento_codigo, url_archivo, estado)
				VALUES ((select auth.uid()), 'seguro', 'transportista/' || (select auth.uid()) || '/s.pdf', 'aprobado')
				RETURNING estado`).Scan(&estado)
		})
		require.NoError(t, err)
		assert.Equal(t, "pendiente", estado)
	})

	t.Run("no registra un path de otro", func(t *testing.T) {
		err := a.db.WithinTx(ctx, transportista, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, `
				INSERT INTO documento_transportista (transportista_id, tipo_documento_codigo, url_archivo)
				VALUES ((select auth.uid()), 'dni', $1)`, "transportista/"+otro.UserID+"/dni.pdf")
			return err
		})
		require.Error(t, err)
	})

	t.Run("no aprueba su propio documento", func(t *testing.T) {
		err := a.db.WithinTx(ctx, transportista, func(tx pgx.Tx) error {
			tag, err := tx.Exec(ctx, `UPDATE documento_transportista SET estado = 'aprobado' WHERE id = $1`, docID)
			assert.Equal(t, int64(0), tag.RowsAffected())
			return err
		})
		require.NoError(t, err)
	})

	t.Run("no se habilita solo", func(t *testing.T) {
		var estado string
		err := a.db.WithinTx(ctx, transportista, func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, `UPDATE transportista SET estado_habilitacion_codigo = 'habilitado'
				WHERE usuario_id = (select auth.uid())`); err != nil {
				return err
			}
			return tx.QueryRow(ctx, `SELECT estado_habilitacion_codigo FROM transportista
				WHERE usuario_id = (select auth.uid())`).Scan(&estado)
		})
		require.NoError(t, err)
		assert.Equal(t, "pendiente", estado)
	})

	t.Run("no ve documentos de otro", func(t *testing.T) {
		var n int
		err := a.db.WithinTx(ctx, otro, func(tx pgx.Tx) error {
			return tx.QueryRow(ctx, `SELECT count(*) FROM documento_transportista WHERE transportista_id = $1`,
				transportista.UserID).Scan(&n)
		})
		require.NoError(t, err)
		assert.Zero(t, n)
	})

	t.Run("no crea notificaciones", func(t *testing.T) {
		err := a.db.WithinTx(ctx, transportista, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, `INSERT INTO notificacion (destinatario_usuario_id, tipo_notificacion_codigo, contenido)
				VALUES ((select auth.uid()), 'documentacion_revisada', 'Tu documentación fue aprobada.')`)
			return err
		})
		require.Error(t, err)
	})
}
