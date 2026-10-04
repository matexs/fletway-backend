package vehiculo

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

// vehiculoID valida el {id} del path; uno mal formado es un 404.
func vehiculoID(r *http.Request) (string, error) {
	v := r.PathValue("id")
	if !uuidValido.MatchString(v) {
		return "", errNoEncontrado
	}
	return v, nil
}

func (h *handler) tipos(w http.ResponseWriter, r *http.Request) error {
	id, err := identidad(r)
	if err != nil {
		return err
	}
	out, err := h.svc.Tipos(r.Context(), id)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, out)
	return nil
}

func (h *handler) crear(w http.ResponseWriter, r *http.Request) error {
	id, err := identidad(r)
	if err != nil {
		return err
	}
	var req CrearVehiculoRequest
	if err := httpx.Decode(r, &req); err != nil {
		return err
	}
	if err := req.Validate(); err != nil {
		return err
	}
	v, err := h.svc.Crear(r.Context(), id, req)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusCreated, v)
	return nil
}

func (h *handler) propios(w http.ResponseWriter, r *http.Request) error {
	id, err := identidad(r)
	if err != nil {
		return err
	}
	out, err := h.svc.Propios(r.Context(), id)
	if err != nil {
		return err
	}
	if out == nil {
		out = []VehiculoResponse{}
	}
	httpx.JSON(w, http.StatusOK, out)
	return nil
}

func (h *handler) activo(w http.ResponseWriter, r *http.Request) error {
	id, err := identidad(r)
	if err != nil {
		return err
	}
	vid, err := vehiculoID(r)
	if err != nil {
		return err
	}
	var req ActivoRequest
	if err := httpx.Decode(r, &req); err != nil {
		return err
	}
	if err := req.Validate(); err != nil {
		return err
	}
	v, err := h.svc.CambiarActivo(r.Context(), id, vid, *req.Activo)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, v)
	return nil
}
