package geografia

import (
	"net/http"

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
