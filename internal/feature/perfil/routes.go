// Package perfil implementa el perfil público del Transportista (RF-11): lo que
// el Cliente necesita para decidir entre ofertas (forma de trabajo, reputación,
// zonas, tipos de vehículo y reseñas). Nunca expone datos de contacto, patentes
// ni datos financieros.
package perfil

import (
	"net/http"

	"github.com/matexs/fletway-backend/internal/platform/database"
	"github.com/matexs/fletway-backend/internal/platform/httpx"
)

// Register monta en el mux de negocio (detrás del middleware Auth, bajo /api):
//
//	GET /transportistas/{id}
func Register(mux *http.ServeMux, db *database.DB) {
	h := &handler{svc: NewService(db)}
	mux.HandleFunc("GET /transportistas/{id}", httpx.Wrap(h.perfil))
}
