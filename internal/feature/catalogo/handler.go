package catalogo

import (
	"net/http"

	"github.com/matexs/fletway-backend/internal/platform/auth"
	"github.com/matexs/fletway-backend/internal/platform/httpx"
)

type handler struct {
	svc *Service
}

func (h *handler) objetos(w http.ResponseWriter, r *http.Request) error {
	id, ok := auth.FromContext(r.Context())
	if !ok {
		return httpx.Unauthorized("no_autenticado", "se requiere un token JWT válido")
	}
	out, err := h.svc.Objetos(r.Context(), id)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, out)
	return nil
}
