package habilitacion

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/matexs/fletway-backend/internal/feature/notificacion"
	"github.com/matexs/fletway-backend/internal/platform/async"
	"github.com/matexs/fletway-backend/internal/platform/database"
	"github.com/matexs/fletway-backend/internal/platform/httpx"
	"github.com/matexs/fletway-backend/internal/platform/storage"
)

// venceURL es la vigencia de las URLs firmadas que ve el Administrador (D-19).
const venceURL = 10 * time.Minute

// Estados de documento_transportista.estado.
const (
	estadoAprobado  = "aprobado"
	estadoRechazado = "rechazado"
)

// Estados de habilitación válidos para filtrar el listado del Administrador.
var estadosHabilitacion = map[string]bool{"pendiente": true, "habilitado": true, "rechazado": true}

var (
	errNoEsTransportista = httpx.Forbidden("no_es_transportista",
		"sólo un Transportista registrado puede cargar documentación")
	errRequiereAdministrador = httpx.Forbidden("requiere_administrador",
		"la operación es sólo para Administradores")
	errTipoInvalido = httpx.BadRequest("tipo_documento_invalido",
		"tipo_documento_codigo tiene que ser dni, seguro, registro o vtv")
	errPathInvalido = httpx.BadRequest("path_invalido",
		"el archivo tiene que estar en transportista/{tu usuario_id}/ del bucket documentos-transportista")
	errArchivoNoEncontrado = httpx.BadRequest("archivo_no_encontrado",
		"no hay un archivo subido en ese path")
	errEstadoInvalido = httpx.BadRequest("estado_invalido",
		"estado tiene que ser pendiente, habilitado o rechazado")
	errNoEncontrado = httpx.NotFound("documento_no_encontrado", "no existe el documento")
	errYaRevisado   = httpx.Conflict("documento_ya_revisado", "el documento ya fue revisado")
)

// Service implementa la carga y la revisión de la documentación de los
// Transportistas (RF-16, RF-01, D-19, D-33).
type Service struct {
	repo        *repository
	firmador    storage.Firmador
	notificador notificacion.Notificador
	jobs        async.Enqueuer
}

// NewService crea el Service. firmador genera las URLs firmadas para el
// Administrador; notificador y jobs entregan la notificación del resultado en
// segundo plano (RNF-02, D-22).
func NewService(db *database.DB, firmador storage.Firmador, notificador notificacion.Notificador, jobs async.Enqueuer) *Service {
	return &Service{repo: &repository{db: db}, firmador: firmador, notificador: notificador, jobs: jobs}
}

// CargarDocumento registra un archivo que el Transportista ya subió a Storage
// (RF-16). Valida que sea Transportista (no_es_transportista), el tipo
// (tipo_documento_invalido), que el path esté bajo su prefijo (path_invalido) y
// que el objeto exista (archivo_no_encontrado). El documento nace pendiente; si
// estaba rechazado y vuelve a cargar, la base lo pasa a pendiente (D-33).
// Escribe en la base.
func (s *Service) CargarDocumento(ctx context.Context, id database.Identity, tipo, path string) (Documento, error) {
	prefijo := "transportista/" + id.UserID + "/"
	if !strings.HasPrefix(path, prefijo) || len(path) == len(prefijo) ||
		strings.Contains(path, "..") || len(path) > 512 {
		return Documento{}, errPathInvalido
	}
	datos, err := s.repo.datosParaCargar(ctx, id, tipo, path)
	if err != nil {
		return Documento{}, fmt.Errorf("cargar documento: %w", err)
	}
	switch {
	case !datos.esTransportista:
		return Documento{}, errNoEsTransportista
	case !datos.tipoValido:
		return Documento{}, errTipoInvalido
	case !datos.archivoExiste:
		return Documento{}, errArchivoNoEncontrado
	}
	d, err := s.repo.insertarDocumento(ctx, id, tipo, path)
	if err != nil {
		return Documento{}, fmt.Errorf("cargar documento: %w", err)
	}
	return d, nil
}

// MiHabilitacion devuelve el estado de habilitación y el último documento de cada
// tipo del Transportista. Devuelve no_es_transportista para otros roles.
func (s *Service) MiHabilitacion(ctx context.Context, id database.Identity) (string, []TipoConUltimo, error) {
	estado, tipos, es, err := s.repo.miHabilitacion(ctx, id)
	if err != nil {
		return "", nil, fmt.Errorf("mi habilitación: %w", err)
	}
	if !es {
		return "", nil, errNoEsTransportista
	}
	return estado, tipos, nil
}

// TransportistasEnRevision lista los Transportistas para el Administrador (RF-01),
// con una URL firmada por documento. Sin estado, los que tienen documentos
// pendientes; con estado, los de ese estado de habilitación. Devuelve
// requiere_administrador o estado_invalido. Hace una llamada a Storage por
// documento.
func (s *Service) TransportistasEnRevision(ctx context.Context, id database.Identity, estado string) ([]TransportistaRevision, error) {
	if estado != "" && !estadosHabilitacion[estado] {
		return nil, errEstadoInvalido
	}
	if err := s.exigirAdministrador(ctx, id); err != nil {
		return nil, err
	}
	lista, err := s.repo.transportistasEnRevision(ctx, id, estado)
	if err != nil {
		return nil, fmt.Errorf("transportistas en revisión: %w", err)
	}
	for i := range lista {
		for j := range lista[i].Tipos {
			d := lista[i].Tipos[j].Ultimo
			if d == nil {
				continue
			}
			if d.URL, err = s.firmador.FirmarURL(ctx, id.AccessToken, bucketDocumentos, d.Path, venceURL); err != nil {
				return nil, fmt.Errorf("transportistas en revisión: %w", err)
			}
		}
	}
	return lista, nil
}

// Aprobar aprueba un documento pendiente (RF-01). Si con eso el Transportista
// queda habilitado, le notifica el resultado (documentacion_revisada) en segundo
// plano. Devuelve requiere_administrador, documento_no_encontrado o
// documento_ya_revisado.
func (s *Service) Aprobar(ctx context.Context, id database.Identity, docID string) (Documento, string, error) {
	d, habilitacion, err := s.revisar(ctx, id, docID, estadoAprobado, nil)
	if err != nil {
		return Documento{}, "", err
	}
	if habilitacion == "habilitado" {
		s.notificar(id, d, "Tu documentación fue aprobada.")
	}
	return d, habilitacion, nil
}

// Rechazar rechaza un documento pendiente con motivo (RF-01). El Transportista
// pasa a rechazado en el acto (D-33) y se le notifica el motivo en segundo plano.
// Devuelve los mismos errores que Aprobar.
func (s *Service) Rechazar(ctx context.Context, id database.Identity, docID, motivo string) (Documento, string, error) {
	d, habilitacion, err := s.revisar(ctx, id, docID, estadoRechazado, &motivo)
	if err != nil {
		return Documento{}, "", err
	}
	s.notificar(id, d, "Tu documentación fue rechazada. Motivo: "+motivo)
	return d, habilitacion, nil
}

func (s *Service) revisar(ctx context.Context, id database.Identity, docID, estado string, motivo *string) (Documento, string, error) {
	if err := s.exigirAdministrador(ctx, id); err != nil {
		return Documento{}, "", err
	}
	d, habilitacion, err := s.repo.revisar(ctx, id, docID, estado, motivo)
	switch {
	case errors.Is(err, errDocumentoNoEncontrado):
		return Documento{}, "", errNoEncontrado
	case errors.Is(err, errDocumentoYaRevisado):
		return Documento{}, "", errYaRevisado
	case err != nil:
		return Documento{}, "", fmt.Errorf("revisar documento: %w", err)
	}
	return d, habilitacion, nil
}

func (s *Service) exigirAdministrador(ctx context.Context, id database.Identity) error {
	es, err := s.repo.esAdministrador(ctx, id)
	if err != nil {
		return err
	}
	if !es {
		return errRequiereAdministrador
	}
	return nil
}

// notificar encola la notificación del resultado. Corre con la identidad del
// Administrador: la policy de notificacion sólo le permite insertar a él. Se
// envía sin reintentos para no duplicarla si el insert llegó a hacerse.
func (s *Service) notificar(admin database.Identity, d Documento, contenido string) {
	s.jobs.Enqueue(async.Job{
		Name: "notificar-documentacion-revisada",
		Run: func(ctx context.Context) error {
			return s.notificador.Notificar(ctx, admin, notificacion.Nueva{
				DestinatarioID: d.TransportistaID,
				Tipo:           notificacion.TipoDocumentacionRevisada,
				Contenido:      contenido,
				EntidadTipo:    "documento_transportista",
				EntidadID:      d.ID,
			})
		},
	})
}
