// Package health expone los endpoints de liveness/readiness (sin JWT).
//
// Sirve además como ejemplo mínimo del patrón de feature del repo:
// Register(mux, deps...) monta rutas; los handlers devuelven error y se envuelven
// con httpx.Wrap.
package health

import (
	"net/http"

	"github.com/fletway/fletway-backend/internal/platform/database"
	"github.com/fletway/fletway-backend/internal/platform/httpx"
)

// Register monta GET /healthz y GET /readyz en el mux público.
func Register(mux *http.ServeMux, db *database.DB) {
	h := &handler{db: db}
	mux.HandleFunc("GET /healthz", httpx.Wrap(h.live))
	mux.HandleFunc("GET /readyz", httpx.Wrap(h.ready))
}

type handler struct {
	db *database.DB
}

func (h *handler) live(w http.ResponseWriter, r *http.Request) error {
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	return nil
}

func (h *handler) ready(w http.ResponseWriter, r *http.Request) error {
	if err := h.db.Ping(r.Context()); err != nil {
		return &httpx.APIError{
			Status:  http.StatusServiceUnavailable,
			Code:    "db_no_disponible",
			Message: "la base de datos no responde",
		}
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "ready", "db": "ok"})
	return nil
}
