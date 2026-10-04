// Package geografia implementa el catálogo de zonas, las zonas de trabajo del
// Transportista (RN-04, D-21) y el autocompletado de direcciones dentro de una
// zona (D-35).
//
// El matchmaking empareja una solicitud con los Transportistas que tienen entre
// sus zonas la de origen o la de destino, así que las zonas elegidas definen qué
// solicitudes ve cada uno.
package geografia

import (
	"net/http"

	"github.com/matexs/fletway-backend/internal/platform/database"
	"github.com/matexs/fletway-backend/internal/platform/geocodificacion"
	"github.com/matexs/fletway-backend/internal/platform/httpx"
)

// Register monta en el mux de negocio (detrás del middleware Auth, bajo /api):
//
//	GET /zonas
//	GET /transportista/zonas
//	PUT /transportista/zonas
//	GET /direcciones/sugerencias?zona_id=&q=
//
// geo sugiere las direcciones.
func Register(mux *http.ServeMux, db *database.DB, geo geocodificacion.Geocodificador) {
	h := &handler{svc: NewService(db, geo)}
	mux.HandleFunc("GET /zonas", httpx.Wrap(h.zonas))
	mux.HandleFunc("GET /transportista/zonas", httpx.Wrap(h.deTransportista))
	mux.HandleFunc("PUT /transportista/zonas", httpx.Wrap(h.reemplazar))
	mux.HandleFunc("GET /direcciones/sugerencias", httpx.Wrap(h.sugerencias))
}
