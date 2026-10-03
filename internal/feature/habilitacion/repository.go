package habilitacion

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/matexs/fletway-backend/internal/platform/database"
)

// bucketDocumentos es el bucket privado de la documentación (D-19).
const bucketDocumentos = "documentos-transportista"

var (
	errDocumentoNoEncontrado = errors.New("habilitacion: documento no encontrado")
	errDocumentoYaRevisado   = errors.New("habilitacion: documento ya revisado")
)

// Documento es una fila de documento_transportista.
type Documento struct {
	ID                  string
	TransportistaID     string
	TipoDocumentoCodigo string
	// Path es url_archivo: el path del objeto en Storage, no una URL (D-19).
	Path          string
	Estado        string
	MotivoRechazo *string
	CargadoEn     time.Time
	RevisadoEn    *time.Time
	// URL es la URL firmada; la completa el service sólo para el Administrador.
	URL string
}

// TipoConUltimo es un tipo de documento con el último documento cargado de ese
// tipo (nil si no hay ninguno).
type TipoConUltimo struct {
	Codigo      string
	Descripcion string
	Ultimo      *Documento
}

// TransportistaRevision es un Transportista con sus documentos, para el
// Administrador.
type TransportistaRevision struct {
	UsuarioID          string
	NombreCompleto     string
	Email              string
	Telefono           string
	EstadoHabilitacion string
	Tipos              []TipoConUltimo
}

// datosCarga es lo que el service necesita validar antes de registrar un documento.
type datosCarga struct {
	esTransportista bool
	tipoValido      bool
	archivoExiste   bool
}

type repository struct {
	db *database.DB
}

const columnasDocumento = `d.id::text, d.transportista_id::text, d.tipo_documento_codigo, d.url_archivo,
	d.estado, d.motivo_rechazo, d.cargado_en, d.revisado_en`

func escanearDocumento(row pgx.Row) (Documento, error) {
	var d Documento
	err := row.Scan(&d.ID, &d.TransportistaID, &d.TipoDocumentoCodigo, &d.Path,
		&d.Estado, &d.MotivoRechazo, &d.CargadoEn, &d.RevisadoEn)
	return d, err
}

// datosParaCargar lee, con el RLS del usuario, si es Transportista, si el tipo
// existe y si el objeto está en Storage (sólo ve los de su prefijo).
func (r *repository) datosParaCargar(ctx context.Context, id database.Identity, tipo, path string) (datosCarga, error) {
	var out datosCarga
	err := r.db.WithinTx(ctx, id, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM transportista WHERE usuario_id = (select auth.uid())),
			       EXISTS (SELECT 1 FROM tipo_documento WHERE codigo = $1),
			       EXISTS (SELECT 1 FROM storage.objects WHERE bucket_id = $2 AND name = $3)`,
			tipo, bucketDocumentos, path,
		).Scan(&out.esTransportista, &out.tipoValido, &out.archivoExiste)
	})
	if err != nil {
		return datosCarga{}, fmt.Errorf("leer datos para cargar documento: %w", err)
	}
	return out, nil
}

// insertarDocumento registra el documento. La base lo deja pendiente y recalcula
// la habilitación (trg_proteger_alta_documento, trg_recalcular_habilitacion).
func (r *repository) insertarDocumento(ctx context.Context, id database.Identity, tipo, path string) (Documento, error) {
	var d Documento
	err := r.db.WithinTx(ctx, id, func(tx pgx.Tx) error {
		var err error
		d, err = escanearDocumento(tx.QueryRow(ctx, `
			INSERT INTO documento_transportista AS d (transportista_id, tipo_documento_codigo, url_archivo)
			VALUES ((select auth.uid()), $1, $2)
			RETURNING `+columnasDocumento, tipo, path))
		return err
	})
	if err != nil {
		return Documento{}, fmt.Errorf("insertar documento: %w", err)
	}
	return d, nil
}

// tiposConUltimo lee cada tipo de documento con el último cargado por el
// Transportista dado, en orden fijo.
func tiposConUltimo(ctx context.Context, tx pgx.Tx, transportistaID string) ([]TipoConUltimo, error) {
	rows, err := tx.Query(ctx, `
		SELECT td.codigo, td.descripcion, `+columnasDocumento+`
		FROM tipo_documento td
		LEFT JOIN LATERAL (
			SELECT * FROM documento_transportista x
			WHERE x.transportista_id = $1 AND x.tipo_documento_codigo = td.codigo
			ORDER BY x.cargado_en DESC, x.id DESC
			LIMIT 1
		) d ON true
		ORDER BY td.codigo`, transportistaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []TipoConUltimo
	for rows.Next() {
		var t TipoConUltimo
		var (
			docID, transportista, tipo, path, estado *string
			motivo                                   *string
			cargado                                  *time.Time
			revisado                                 *time.Time
		)
		if err := rows.Scan(&t.Codigo, &t.Descripcion, &docID, &transportista, &tipo, &path,
			&estado, &motivo, &cargado, &revisado); err != nil {
			return nil, err
		}
		if docID != nil {
			t.Ultimo = &Documento{
				ID: *docID, TransportistaID: *transportista, TipoDocumentoCodigo: *tipo, Path: *path,
				Estado: *estado, MotivoRechazo: motivo, CargadoEn: *cargado, RevisadoEn: revisado,
			}
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// miHabilitacion lee el estado y los documentos del Transportista de la identidad.
// esTransportista es false si la cuenta no tiene fila transportista.
func (r *repository) miHabilitacion(ctx context.Context, id database.Identity) (estado string, tipos []TipoConUltimo, esTransportista bool, err error) {
	err = r.db.WithinTx(ctx, id, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx,
			`SELECT estado_habilitacion_codigo FROM transportista WHERE usuario_id = (select auth.uid())`,
		).Scan(&estado)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		esTransportista = true
		tipos, err = tiposConUltimo(ctx, tx, id.UserID)
		return err
	})
	if err != nil {
		return "", nil, false, fmt.Errorf("leer mi habilitación: %w", err)
	}
	return estado, tipos, esTransportista, nil
}

// esAdministrador indica si la identidad es de un Administrador.
func (r *repository) esAdministrador(ctx context.Context, id database.Identity) (bool, error) {
	var es bool
	err := r.db.WithinTx(ctx, id, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `SELECT fn_es_administrador()`).Scan(&es)
	})
	if err != nil {
		return false, fmt.Errorf("verificar administrador: %w", err)
	}
	return es, nil
}

// transportistasEnRevision lista Transportistas con sus documentos. Con estado
// vacío, los que tienen algún documento pendiente (incluye renovaciones de
// habilitados); si no, los que están en ese estado de habilitación.
func (r *repository) transportistasEnRevision(ctx context.Context, id database.Identity, estado string) ([]TransportistaRevision, error) {
	var out []TransportistaRevision
	err := r.db.WithinTx(ctx, id, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT t.usuario_id::text, u.nombre_completo, u.email::text, u.telefono, t.estado_habilitacion_codigo
			FROM transportista t
			JOIN usuario u ON u.id = t.usuario_id
			WHERE ($1 = '' AND EXISTS (
			         SELECT 1 FROM documento_transportista d
			         WHERE d.transportista_id = t.usuario_id AND d.estado = 'pendiente'))
			   OR t.estado_habilitacion_codigo = $1
			ORDER BY t.creado_en`, estado)
		if err != nil {
			return err
		}
		lista, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (TransportistaRevision, error) {
			var t TransportistaRevision
			err := row.Scan(&t.UsuarioID, &t.NombreCompleto, &t.Email, &t.Telefono, &t.EstadoHabilitacion)
			return t, err
		})
		if err != nil {
			return err
		}
		for i := range lista {
			if lista[i].Tipos, err = tiposConUltimo(ctx, tx, lista[i].UsuarioID); err != nil {
				return err
			}
		}
		out = lista
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("listar transportistas en revisión: %w", err)
	}
	return out, nil
}

// revisar aprueba o rechaza un documento pendiente y devuelve el documento y el
// estado de habilitación resultante (lo recalcula la base). Devuelve
// errDocumentoNoEncontrado o errDocumentoYaRevisado.
func (r *repository) revisar(ctx context.Context, id database.Identity, docID, estado string, motivo *string) (Documento, string, error) {
	var (
		d          Documento
		habilitado string
	)
	err := r.db.WithinTx(ctx, id, func(tx pgx.Tx) error {
		var err error
		d, err = escanearDocumento(tx.QueryRow(ctx, `
			UPDATE documento_transportista AS d
			SET estado = $2, motivo_rechazo = $3, revisado_por_admin_id = (select auth.uid()), revisado_en = now()
			WHERE d.id = $1 AND d.estado = 'pendiente'
			RETURNING `+columnasDocumento, docID, estado, motivo))
		if errors.Is(err, pgx.ErrNoRows) {
			var existe bool
			if err := tx.QueryRow(ctx,
				`SELECT EXISTS (SELECT 1 FROM documento_transportista WHERE id = $1)`, docID,
			).Scan(&existe); err != nil {
				return err
			}
			if existe {
				return errDocumentoYaRevisado
			}
			return errDocumentoNoEncontrado
		}
		if err != nil {
			return err
		}
		return tx.QueryRow(ctx,
			`SELECT estado_habilitacion_codigo FROM transportista WHERE usuario_id = $1`, d.TransportistaID,
		).Scan(&habilitado)
	})
	if err != nil {
		return Documento{}, "", fmt.Errorf("revisar documento: %w", err)
	}
	return d, habilitado, nil
}
