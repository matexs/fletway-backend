// Package vehiculo implementa los vehículos del Transportista (RF-18, D-32):
// tipo, patente, medidas útiles y carga útil, que usa el cálculo de viajes de
// cada oferta (RN-02).
//
// Los costos del vehículo que entran en el precio no los carga el Transportista:
// los define la plataforma por tipo de vehículo en config_costo_vehiculo (D-34).
package vehiculo

import (
	"net/http"

	"github.com/matexs/fletway-backend/internal/platform/database"
	"github.com/matexs/fletway-backend/internal/platform/httpx"
)

// Register monta en el mux de negocio (detrás del middleware Auth, bajo /api):
//
//	GET  /tipos-vehiculo
//	POST /transportista/vehiculos
//	GET  /transportista/vehiculos
//	PUT  /transportista/vehiculos/{id}/activo
func Register(mux *http.ServeMux, db *database.DB) {
	h := &handler{svc: NewService(db)}
	mux.HandleFunc("GET /tipos-vehiculo", httpx.Wrap(h.tipos))
	mux.HandleFunc("POST /transportista/vehiculos", httpx.Wrap(h.crear))
	mux.HandleFunc("GET /transportista/vehiculos", httpx.Wrap(h.propios))
	mux.HandleFunc("PUT /transportista/vehiculos/{id}/activo", httpx.Wrap(h.activo))
}
