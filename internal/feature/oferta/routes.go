// Package oferta implementa la postulación del Transportista a una solicitud
// (RF-17): el sistema calcula la cantidad de viajes con el vehículo real (RN-02)
// y el precio (RN-01); el Transportista sólo elige vehículo y ayudantes.
//
// El desglose del precio se guarda en oferta_costo, que el Cliente no puede leer
// (D-23). Una oferta no se edita: se retira y se crea otra; la base lo impone
// (migración 0016).
package oferta

import (
	"net/http"

	"github.com/matexs/fletway-backend/internal/platform/httpx"
)

// Register monta en el mux de negocio (detrás del middleware Auth, bajo /api):
//
//	POST /solicitudes/{id}/ofertas/cotizar
//	POST /solicitudes/{id}/ofertas
//	POST /ofertas/{id}/retirar
//	GET  /transportista/ofertas
func Register(mux *http.ServeMux, svc *Service) {
	h := &handler{svc: svc}
	mux.HandleFunc("POST /solicitudes/{id}/ofertas/cotizar", httpx.Wrap(h.cotizar))
	mux.HandleFunc("POST /solicitudes/{id}/ofertas", httpx.Wrap(h.crear))
	mux.HandleFunc("POST /ofertas/{id}/retirar", httpx.Wrap(h.retirar))
	mux.HandleFunc("GET /transportista/ofertas", httpx.Wrap(h.propias))
}
