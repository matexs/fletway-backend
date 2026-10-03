package habilitacion

import (
	"net/http"
	"regexp"

	"github.com/matexs/fletway-backend/internal/platform/auth"
	"github.com/matexs/fletway-backend/internal/platform/database"
	"github.com/matexs/fletway-backend/internal/platform/httpx"
)

var uuidValido = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

type handler struct {
	svc *Service
}

func identidad(r *http.Request) (database.Identity, error) {
	id, ok := auth.FromContext(r.Context())
	if !ok {
		return database.Identity{}, httpx.Unauthorized("no_autenticado", "se requiere un token JWT válido")
	}
	return id, nil
}

func (h *handler) cargar(w http.ResponseWriter, r *http.Request) error {
	id, err := identidad(r)
	if err != nil {
		return err
	}
	var req CargarDocumentoRequest
	if err := httpx.Decode(r, &req); err != nil {
		return err
	}
	if err := req.Validate(); err != nil {
		return err
	}
	d, err := h.svc.CargarDocumento(r.Context(), id, req.TipoDocumentoCodigo, req.Path)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusCreated, toDocumentoResponse(d))
	return nil
}

func (h *handler) mios(w http.ResponseWriter, r *http.Request) error {
	id, err := identidad(r)
	if err != nil {
		return err
	}
	estado, tipos, err := h.svc.MiHabilitacion(r.Context(), id)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, MiHabilitacionResponse{EstadoHabilitacion: estado, Documentos: toTiposResponse(tipos)})
	return nil
}

func (h *handler) enRevision(w http.ResponseWriter, r *http.Request) error {
	id, err := identidad(r)
	if err != nil {
		return err
	}
	lista, err := h.svc.TransportistasEnRevision(r.Context(), id, r.URL.Query().Get("estado"))
	if err != nil {
		return err
	}
	out := make([]TransportistaRevisionResponse, 0, len(lista))
	for _, t := range lista {
		out = append(out, TransportistaRevisionResponse{
			UsuarioID:          t.UsuarioID,
			NombreCompleto:     t.NombreCompleto,
			Email:              t.Email,
			Telefono:           t.Telefono,
			EstadoHabilitacion: t.EstadoHabilitacion,
			Documentos:         toTiposResponse(t.Tipos),
		})
	}
	httpx.JSON(w, http.StatusOK, out)
	return nil
}

func (h *handler) aprobar(w http.ResponseWriter, r *http.Request) error {
	id, err := identidad(r)
	if err != nil {
		return err
	}
	docID, err := documentoID(r)
	if err != nil {
		return err
	}
	d, habilitacion, err := h.svc.Aprobar(r.Context(), id, docID)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, RevisionResponse{Documento: toDocumentoResponse(d), EstadoHabilitacion: habilitacion})
	return nil
}

func (h *handler) rechazar(w http.ResponseWriter, r *http.Request) error {
	id, err := identidad(r)
	if err != nil {
		return err
	}
	docID, err := documentoID(r)
	if err != nil {
		return err
	}
	var req RechazarRequest
	if err := httpx.Decode(r, &req); err != nil {
		return err
	}
	if err := req.Validate(); err != nil {
		return err
	}
	d, habilitacion, err := h.svc.Rechazar(r.Context(), id, docID, req.Motivo)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, RevisionResponse{Documento: toDocumentoResponse(d), EstadoHabilitacion: habilitacion})
	return nil
}

// documentoID valida el {id} del path; un id mal formado es un 404, igual que uno
// que no existe.
func documentoID(r *http.Request) (string, error) {
	docID := r.PathValue("id")
	if !uuidValido.MatchString(docID) {
		return "", errNoEncontrado
	}
	return docID, nil
}
