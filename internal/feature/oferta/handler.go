package oferta

import (
	"net/http"
	"strconv"

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

// pathID valida el {id} del path; uno mal formado es un 404 con noEncontrado.
func pathID(r *http.Request, noEncontrado error) (string, error) {
	v := r.PathValue("id")
	if !uuidValido.MatchString(v) {
		return "", noEncontrado
	}
	return v, nil
}

// pedido lee la identidad, la solicitud del path y el cuerpo validado.
func pedido(r *http.Request) (database.Identity, string, OfertaRequest, error) {
	var req OfertaRequest
	id, err := identidad(r)
	if err != nil {
		return id, "", req, err
	}
	sid, err := pathID(r, errSolicitudNoDisponible)
	if err != nil {
		return id, "", req, err
	}
	if err := httpx.Decode(r, &req); err != nil {
		return id, "", req, err
	}
	return id, sid, req, req.Validate()
}

func (h *handler) cotizar(w http.ResponseWriter, r *http.Request) error {
	id, sid, req, err := pedido(r)
	if err != nil {
		return err
	}
	out, err := h.svc.Cotizar(r.Context(), id, sid, req)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, out)
	return nil
}

func (h *handler) crear(w http.ResponseWriter, r *http.Request) error {
	id, sid, req, err := pedido(r)
	if err != nil {
		return err
	}
	out, err := h.svc.Crear(r.Context(), id, sid, req)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusCreated, out)
	return nil
}

func (h *handler) retirar(w http.ResponseWriter, r *http.Request) error {
	id, err := identidad(r)
	if err != nil {
		return err
	}
	oid, err := pathID(r, errNoEncontradaAPI)
	if err != nil {
		return err
	}
	out, err := h.svc.Retirar(r.Context(), id, oid)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, out)
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

// pagina lee ?ver_mas, ?cursor y ?limit. Devuelve datos_invalidos si no son
// números válidos.
func pagina(r *http.Request) (Pagina, error) {
	q := r.URL.Query()
	p := Pagina{VerMas: q.Get("ver_mas") == "true", Cursor: topOfertas, Limite: limiteVerMas}
	e := map[string]any{}
	if v := q.Get("cursor"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			e["cursor"] = "tiene que ser un entero mayor o igual que 0"
		}
		p.Cursor = n
	}
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > maxLimiteVerMas {
			e["limit"] = "entre 1 y 50"
		}
		p.Limite = n
	}
	if len(e) > 0 {
		return Pagina{}, &httpx.APIError{Status: http.StatusBadRequest, Code: "datos_invalidos",
			Message: "hay parámetros con valores inválidos", Details: e}
	}
	return p, nil
}

func (h *handler) deSolicitud(w http.ResponseWriter, r *http.Request) error {
	id, err := identidad(r)
	if err != nil {
		return err
	}
	sid, err := pathID(r, errSolicitudClienteAPI)
	if err != nil {
		return err
	}
	p, err := pagina(r)
	if err != nil {
		return err
	}
	out, err := h.svc.OfertasDeSolicitud(r.Context(), id, sid, p)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, out)
	return nil
}

func (h *handler) aceptar(w http.ResponseWriter, r *http.Request) error {
	id, err := identidad(r)
	if err != nil {
		return err
	}
	oid, err := pathID(r, errNoEncontradaAPI)
	if err != nil {
		return err
	}
	out, err := h.svc.Aceptar(r.Context(), id, oid)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusCreated, out)
	return nil
}
