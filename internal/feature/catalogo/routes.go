// Package catalogo expone el catálogo de objetos comunes (RN-08): muebles,
// electrodomésticos y bultos con peso, medidas y restricciones de rotación y
// apilado, que el Cliente elige al publicar una solicitud (RF-06).
package catalogo

import (
	"net/http"

	"github.com/matexs/fletway-backend/internal/platform/database"
	"github.com/matexs/fletway-backend/internal/platform/httpx"
)

// Register monta en el mux de negocio (detrás del middleware Auth, bajo /api):
//
//	GET /catalogo/objetos
func Register(mux *http.ServeMux, db *database.DB) {
	h := &handler{svc: NewService(db)}
	mux.HandleFunc("GET /catalogo/objetos", httpx.Wrap(h.objetos))
}
