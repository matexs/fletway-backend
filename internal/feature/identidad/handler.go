package identidad

import (
	"net/http"

	"github.com/matexs/fletway-backend/internal/platform/auth"
	"github.com/matexs/fletway-backend/internal/platform/httpx"
)

type handler struct {
	svc *Service
}

func (h *handler) me(w http.ResponseWriter, r *http.Request) error {
	id, ok := auth.FromContext(r.Context())
	if !ok {
		return httpx.Unauthorized("no_autenticado", "se requiere un token JWT válido")
	}
	p, err := h.svc.Me(r.Context(), id)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, toMeResponse(p))
	return nil
}

// registrar devuelve el handler de registro para rol: 201 si creó la fila del rol y
// 200 si ya existía (la app puede reintentar sin error).
func (h *handler) registrar(rol string) httpx.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		id, ok := auth.FromContext(r.Context())
		if !ok {
			return httpx.Unauthorized("no_autenticado", "se requiere un token JWT válido")
		}
		p, creada, err := h.svc.Registrar(r.Context(), id, rol)
		if err != nil {
			return err
		}
		status := http.StatusOK
		if creada {
			status = http.StatusCreated
		}
		httpx.JSON(w, status, toMeResponse(p))
		return nil
	}
}

func (h *handler) disponibilidad(w http.ResponseWriter, r *http.Request) error {
	id, ok := auth.FromContext(r.Context())
	if !ok {
		return httpx.Unauthorized("no_autenticado", "se requiere un token JWT válido")
	}
	var req DisponibilidadRequest
	if err := httpx.Decode(r, &req); err != nil {
		return err
	}
	if err := req.Validate(); err != nil {
		return err
	}
	p, err := h.svc.CambiarDisponibilidad(r.Context(), id, *req.Disponible)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, toMeResponse(p))
	return nil
}
