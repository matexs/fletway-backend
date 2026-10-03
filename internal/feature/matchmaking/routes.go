// Package matchmaking empareja solicitudes con Transportistas (RN-04, RN-05,
// D-21): el listado de solicitudes compatibles y el aviso in-app al publicar.
//
// La regla de compatibilidad vive en la base (fn_es_compatible, migración 0015)
// para que el listado y el aviso no puedan divergir.
package matchmaking

import (
	"net/http"

	"github.com/matexs/fletway-backend/internal/platform/httpx"
)

// Register monta en el mux de negocio (detrás del middleware Auth, bajo /api):
//
//	GET /transportista/solicitudes
func Register(mux *http.ServeMux, svc *Service) {
	h := &handler{svc: svc}
	mux.HandleFunc("GET /transportista/solicitudes", httpx.Wrap(h.compatibles))
}
