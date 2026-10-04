package perfil

import (
	"net/http"

	"github.com/matexs/fletway-backend/internal/platform/auth"
	"github.com/matexs/fletway-backend/internal/platform/httpx"
)

type handler struct {
	svc *Service
}

func (h *handler) perfil(w http.ResponseWriter, r *http.Request) error {
	id, ok := auth.FromContext(r.Context())
	if !ok {
		return httpx.Unauthorized("no_autenticado", "se requiere un token JWT válido")
	}
	tid := r.PathValue("id")
	if !uuidValido.MatchString(tid) {
		return errNoEncontradoAPI
	}
	p, err := h.svc.Perfil(r.Context(), id, tid)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, p)
	return nil
}
