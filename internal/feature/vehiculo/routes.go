// Package vehiculo implementa los vehículos del Transportista y sus costos
// operativos (RF-18, RN-01, D-32).
//
// El alta tiene dos pasos: primero el vehículo (tipo, patente, medidas útiles y
// carga útil) y después sus costos (vehiculo_costo), que sólo leen el dueño y el
// Administrador. Sin costos el vehículo no sirve para ofertar.
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
//	PUT  /transportista/vehiculos/{id}/costos
//	GET  /transportista/vehiculos/{id}/costos
func Register(mux *http.ServeMux, db *database.DB) {
	h := &handler{svc: NewService(db)}
	mux.HandleFunc("GET /tipos-vehiculo", httpx.Wrap(h.tipos))
	mux.HandleFunc("POST /transportista/vehiculos", httpx.Wrap(h.crear))
	mux.HandleFunc("GET /transportista/vehiculos", httpx.Wrap(h.propios))
	mux.HandleFunc("PUT /transportista/vehiculos/{id}/activo", httpx.Wrap(h.activo))
	mux.HandleFunc("PUT /transportista/vehiculos/{id}/costos", httpx.Wrap(h.guardarCostos))
	mux.HandleFunc("GET /transportista/vehiculos/{id}/costos", httpx.Wrap(h.costos))
}
