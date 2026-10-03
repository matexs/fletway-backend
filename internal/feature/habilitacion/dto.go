package habilitacion

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/matexs/fletway-backend/internal/platform/httpx"
)

// largoMaximoMotivo acota el motivo de rechazo que ve el Transportista.
const largoMaximoMotivo = 500

// CargarDocumentoRequest registra un archivo ya subido a Storage (D-19).
type CargarDocumentoRequest struct {
	// TipoDocumentoCodigo es dni, seguro, registro o vtv.
	TipoDocumentoCodigo string `json:"tipo_documento_codigo"`
	// Path es el path del objeto en el bucket documentos-transportista; tiene que
	// estar bajo transportista/{usuario_id}/.
	Path string `json:"path"`
}

// Validate controla la forma del request; devuelve un *httpx.APIError
// (datos_incompletos) si falta algún campo.
func (r *CargarDocumentoRequest) Validate() error {
	r.TipoDocumentoCodigo = strings.TrimSpace(r.TipoDocumentoCodigo)
	r.Path = strings.TrimSpace(r.Path)
	if r.TipoDocumentoCodigo == "" || r.Path == "" {
		return httpx.BadRequest("datos_incompletos", "faltan tipo_documento_codigo o path")
	}
	return nil
}

// RechazarRequest es el motivo del rechazo de un documento (RF-01).
type RechazarRequest struct {
	Motivo string `json:"motivo"`
}

// Validate exige un motivo de 1 a 500 caracteres; devuelve un *httpx.APIError
// (motivo_requerido o motivo_demasiado_largo).
func (r *RechazarRequest) Validate() error {
	r.Motivo = strings.TrimSpace(r.Motivo)
	if r.Motivo == "" {
		return httpx.BadRequest("motivo_requerido", "el rechazo necesita un motivo")
	}
	if utf8.RuneCountInString(r.Motivo) > largoMaximoMotivo {
		return httpx.BadRequest("motivo_demasiado_largo", "el motivo no puede superar los 500 caracteres")
	}
	return nil
}

// DocumentoResponse es un documento cargado.
type DocumentoResponse struct {
	ID                  string     `json:"id"`
	TipoDocumentoCodigo string     `json:"tipo_documento_codigo"`
	Estado              string     `json:"estado"`
	MotivoRechazo       *string    `json:"motivo_rechazo"`
	CargadoEn           time.Time  `json:"cargado_en"`
	RevisadoEn          *time.Time `json:"revisado_en"`
	// URL es una URL firmada de lectura que vence a los 10 minutos. Sólo en las
	// respuestas para el Administrador.
	URL string `json:"url,omitempty"`
}

// TipoDocumentoResponse es un tipo de documento requerido con su último documento
// cargado (null si todavía no cargó ninguno).
type TipoDocumentoResponse struct {
	TipoDocumentoCodigo string             `json:"tipo_documento_codigo"`
	Descripcion         string             `json:"descripcion"`
	Ultimo              *DocumentoResponse `json:"ultimo"`
}

// MiHabilitacionResponse es el estado de habilitación del Transportista y sus
// documentos (GET /api/transportista/documentos).
type MiHabilitacionResponse struct {
	EstadoHabilitacion string                  `json:"estado_habilitacion"`
	Documentos         []TipoDocumentoResponse `json:"documentos"`
}

// TransportistaRevisionResponse es un Transportista con sus documentos, para la
// revisión del Administrador.
type TransportistaRevisionResponse struct {
	UsuarioID          string                  `json:"usuario_id"`
	NombreCompleto     string                  `json:"nombre_completo"`
	Email              string                  `json:"email"`
	Telefono           string                  `json:"telefono"`
	EstadoHabilitacion string                  `json:"estado_habilitacion"`
	Documentos         []TipoDocumentoResponse `json:"documentos"`
}

// RevisionResponse es el resultado de aprobar o rechazar un documento.
type RevisionResponse struct {
	Documento DocumentoResponse `json:"documento"`
	// EstadoHabilitacion es el estado del Transportista después de la revisión.
	EstadoHabilitacion string `json:"estado_habilitacion"`
}

func toDocumentoResponse(d Documento) DocumentoResponse {
	return DocumentoResponse{
		ID:                  d.ID,
		TipoDocumentoCodigo: d.TipoDocumentoCodigo,
		Estado:              d.Estado,
		MotivoRechazo:       d.MotivoRechazo,
		CargadoEn:           d.CargadoEn,
		RevisadoEn:          d.RevisadoEn,
		URL:                 d.URL,
	}
}

func toTiposResponse(tipos []TipoConUltimo) []TipoDocumentoResponse {
	out := make([]TipoDocumentoResponse, 0, len(tipos))
	for _, t := range tipos {
		r := TipoDocumentoResponse{TipoDocumentoCodigo: t.Codigo, Descripcion: t.Descripcion}
		if t.Ultimo != nil {
			d := toDocumentoResponse(*t.Ultimo)
			r.Ultimo = &d
		}
		out = append(out, r)
	}
	return out
}
