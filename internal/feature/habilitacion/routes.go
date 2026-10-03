// Package habilitacion implementa la carga de la documentación del Transportista
// (RF-16) y su revisión por el Administrador (RF-01), según D-19 y D-33.
//
// Los archivos se suben desde la app directo a Storage, bajo
// transportista/{usuario_id}/ del bucket documentos-transportista; acá se
// registran y se revisan. El estado de habilitación lo deriva la base de los
// documentos (trg_recalcular_habilitacion), así que ningún camino lo saltea.
package habilitacion

import (
	"net/http"

	"github.com/matexs/fletway-backend/internal/platform/httpx"
)

// Register monta en el mux de negocio (detrás del middleware Auth, bajo /api):
//
//	POST /transportista/documentos
//	GET  /transportista/documentos
//	GET  /admin/transportistas[?estado=pendiente|habilitado|rechazado]
//	POST /admin/documentos/{id}/aprobar
//	POST /admin/documentos/{id}/rechazar
func Register(mux *http.ServeMux, svc *Service) {
	h := &handler{svc: svc}
	mux.HandleFunc("POST /transportista/documentos", httpx.Wrap(h.cargar))
	mux.HandleFunc("GET /transportista/documentos", httpx.Wrap(h.mios))
	mux.HandleFunc("GET /admin/transportistas", httpx.Wrap(h.enRevision))
	mux.HandleFunc("POST /admin/documentos/{id}/aprobar", httpx.Wrap(h.aprobar))
	mux.HandleFunc("POST /admin/documentos/{id}/rechazar", httpx.Wrap(h.rechazar))
}
