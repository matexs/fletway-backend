// Package solicitud implementa la publicación de necesidades de traslado del
// Cliente (RF-06, D-20): alta con objetos del catálogo o cargados a mano, listado
// con el vencimiento calculado al leer, cancelación y republicación.
//
// Ningún endpoint de solicitud calcula ni devuelve un monto: el único precio es
// el de cada oferta (RN-01). Una solicitud no se edita: se cancela y se publica
// otra; la base lo impone (migración 0014).
package solicitud

import (
	"net/http"

	"github.com/matexs/fletway-backend/internal/platform/httpx"
)

// Register monta en el mux de negocio (detrás del middleware Auth, bajo /api):
//
//	POST /solicitudes
//	GET  /solicitudes
//	GET  /solicitudes/{id}
//	GET  /solicitudes/{id}/ruta
//	POST /solicitudes/{id}/cancelar
//	POST /solicitudes/{id}/republicar
func Register(mux *http.ServeMux, svc *Service) {
	h := &handler{svc: svc}
	mux.HandleFunc("POST /solicitudes", httpx.Wrap(h.crear))
	mux.HandleFunc("GET /solicitudes", httpx.Wrap(h.propias))
	mux.HandleFunc("GET /solicitudes/{id}", httpx.Wrap(h.detalle))
	mux.HandleFunc("GET /solicitudes/{id}/ruta", httpx.Wrap(h.ruta))
	mux.HandleFunc("POST /solicitudes/{id}/cancelar", httpx.Wrap(h.cancelar))
	mux.HandleFunc("POST /solicitudes/{id}/republicar", httpx.Wrap(h.republicar))
}
