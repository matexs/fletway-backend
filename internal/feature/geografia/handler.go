package geografia

import (
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/matexs/fletway-backend/internal/platform/auth"
	"github.com/matexs/fletway-backend/internal/platform/database"
	"github.com/matexs/fletway-backend/internal/platform/httpx"
)

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

func (h *handler) zonas(w http.ResponseWriter, r *http.Request) error {
	id, err := identidad(r)
	if err != nil {
		return err
	}
	out, err := h.svc.Zonas(r.Context(), id)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, out)
	return nil
}

func (h *handler) deTransportista(w http.ResponseWriter, r *http.Request) error {
	id, err := identidad(r)
	if err != nil {
		return err
	}
	out, err := h.svc.DeTransportista(r.Context(), id)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, out)
	return nil
}

func (h *handler) reemplazar(w http.ResponseWriter, r *http.Request) error {
	id, err := identidad(r)
	if err != nil {
		return err
	}
	var req ZonasTransportista
	if err := httpx.Decode(r, &req); err != nil {
		return err
	}
	if err := req.Validate(); err != nil {
		return err
	}
	out, err := h.svc.Reemplazar(r.Context(), id, req)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, out)
	return nil
}

// sugerencias responde GET /direcciones/sugerencias?zona_id=&q=. q tiene que
// tener entre 3 y 100 caracteres: con menos las sugerencias no sirven.
func (h *handler) sugerencias(w http.ResponseWriter, r *http.Request) error {
	id, err := identidad(r)
	if err != nil {
		return err
	}
	zonaID := r.URL.Query().Get("zona_id")
	texto := strings.TrimSpace(r.URL.Query().Get("q"))
	e := map[string]any{}
	if !uuidValido.MatchString(zonaID) {
		e["zona_id"] = "elegí una zona del catálogo"
	}
	if n := utf8.RuneCountInString(texto); n < 3 || n > 100 {
		e["q"] = "entre 3 y 100 caracteres"
	}
	if len(e) > 0 {
		return &httpx.APIError{Status: http.StatusBadRequest, Code: "datos_invalidos",
			Message: "hay parámetros con valores inválidos", Details: e}
	}
	out, err := h.svc.Sugerencias(r.Context(), id, zonaID, texto)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, out)
	return nil
}
