// Package identidad implementa el alta de cuenta y el perfil del usuario autenticado
// (RF-05, RF-16, D-18).
//
// El signUp lo hace la app contra Supabase Auth; el trigger trg_alta_usuario crea la
// fila usuario con el rol de la metadata (sólo cliente o transportista). Después la app
// llama al endpoint de registro de su rol, que crea la fila cliente o transportista, y
// usa GET /api/me como única fuente del rol.
package identidad

import (
	"net/http"

	"github.com/matexs/fletway-backend/internal/platform/database"
	"github.com/matexs/fletway-backend/internal/platform/httpx"
)

// Register monta en el mux de negocio (detrás del middleware Auth, bajo /api):
//
//	GET  /me
//	POST /auth/registro/cliente
//	POST /auth/registro/transportista
func Register(mux *http.ServeMux, db *database.DB) {
	h := &handler{svc: NewService(db)}
	mux.HandleFunc("GET /me", httpx.Wrap(h.me))
	mux.HandleFunc("POST /auth/registro/cliente", httpx.Wrap(h.registrar(RolCliente)))
	mux.HandleFunc("POST /auth/registro/transportista", httpx.Wrap(h.registrar(RolTransportista)))
}
