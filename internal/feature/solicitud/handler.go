package solicitud

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

// solicitudID valida el {id} del path; uno mal formado es un 404.
func solicitudID(r *http.Request) (string, error) {
	v := r.PathValue("id")
	if !uuidValido.MatchString(v) {
		return "", errSolicitudNoEncontrada
	}
	return v, nil
}

func (h *handler) crear(w http.ResponseWriter, r *http.Request) error {
	id, err := identidad(r)
	if err != nil {
		return err
	}
	var req CrearSolicitudRequest
	if err := httpx.Decode(r, &req); err != nil {
		return err
	}
	if err := req.Validate(); err != nil {
		return err
	}
	out, err := h.svc.Crear(r.Context(), id, req)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusCreated, out)
	return nil
}

func (h *handler) propias(w http.ResponseWriter, r *http.Request) error {
	id, err := identidad(r)
	if err != nil {
		return err
	}
	out, err := h.svc.Propias(r.Context(), id)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, out)
	return nil
}

func (h *handler) detalle(w http.ResponseWriter, r *http.Request) error {
	id, err := identidad(r)
	if err != nil {
		return err
	}
	sid, err := solicitudID(r)
	if err != nil {
		return err
	}
	out, err := h.svc.Detalle(r.Context(), id, sid)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, out)
	return nil
}

func (h *handler) cancelar(w http.ResponseWriter, r *http.Request) error {
	id, err := identidad(r)
	if err != nil {
		return err
	}
	sid, err := solicitudID(r)
	if err != nil {
		return err
	}
	out, err := h.svc.Cancelar(r.Context(), id, sid)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, out)
	return nil
}

func (h *handler) republicar(w http.ResponseWriter, r *http.Request) error {
	id, err := identidad(r)
	if err != nil {
		return err
	}
	sid, err := solicitudID(r)
	if err != nil {
		return err
	}
	var req RepublicarRequest
	if err := httpx.Decode(r, &req); err != nil {
		return err
	}
	if err := req.Validate(); err != nil {
		return err
	}
	out, err := h.svc.Republicar(r.Context(), id, sid, req)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusCreated, out)
	return nil
}

func (h *handler) ruta(w http.ResponseWriter, r *http.Request) error {
	id, err := identidad(r)
	if err != nil {
		return err
	}
	sid, err := solicitudID(r)
	if err != nil {
		return err
	}
	out, err := h.svc.Ruta(r.Context(), id, sid)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, out)
	return nil
}
