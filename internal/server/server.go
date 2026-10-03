// Package server arma el http.Handler del servicio: registra middlewares globales
// y monta las rutas de cada feature.
package server

import (
	"log/slog"
	"net/http"

	"github.com/matexs/fletway-backend/internal/feature/habilitacion"
	"github.com/matexs/fletway-backend/internal/feature/health"
	"github.com/matexs/fletway-backend/internal/feature/identidad"
	"github.com/matexs/fletway-backend/internal/feature/notificacion"
	"github.com/matexs/fletway-backend/internal/platform/async"
	"github.com/matexs/fletway-backend/internal/platform/auth"
	"github.com/matexs/fletway-backend/internal/platform/config"
	"github.com/matexs/fletway-backend/internal/platform/database"
	"github.com/matexs/fletway-backend/internal/platform/middleware"
	"github.com/matexs/fletway-backend/internal/platform/storage"
)

// Server contiene las dependencias compartidas por todas las features.
type Server struct {
	cfg      config.Config
	log      *slog.Logger
	db       *database.DB
	jobs     async.Enqueuer
	verifier *auth.Verifier
	archivos storage.Firmador
}

// New construye el Server con sus dependencias. El verifier de JWT lo crea quien
// llama (necesita un contexto de vida para refrescar el JWKS); archivos firma las
// URLs de Storage (D-19).
func New(cfg config.Config, log *slog.Logger, db *database.DB, jobs async.Enqueuer, verifier *auth.Verifier, archivos storage.Firmador) *Server {
	return &Server{
		cfg:      cfg,
		log:      log,
		db:       db,
		jobs:     jobs,
		verifier: verifier,
		archivos: archivos,
	}
}

// Handler devuelve el http.Handler raíz con middlewares globales aplicados.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// --- Rutas públicas (sin JWT): solo health checks ---
	health.Register(mux, s.db)

	// --- Rutas de negocio (JWT obligatorio - RNF-01) ---
	// Se montan bajo un sub-mux envuelto por el middleware Auth. Cada feature
	// agrega sus rutas acá vía su función Register(apiMux, deps...).
	apiMux := http.NewServeMux()
	s.registerAPI(apiMux)

	authed := middleware.Auth(s.verifier)(apiMux)
	mux.Handle("/api/", http.StripPrefix("/api", authed))

	// Middlewares globales (orden externo→interno): RequestID, Recover, Logging.
	return middleware.Chain(mux,
		middleware.RequestID,
		middleware.Recover(s.log),
		middleware.Logging(s.log),
	)
}

// registerAPI monta las rutas de negocio. A medida que se implementan features,
// se agregan acá sus Register(...). Ejemplo:
//
//	solicitud.Register(apiMux, s.db, s.jobs)
//	oferta.Register(apiMux, s.db)
func (s *Server) registerAPI(apiMux *http.ServeMux) {
	identidad.Register(apiMux, s.db)
	habilitacion.Register(apiMux,
		habilitacion.NewService(s.db, s.archivos, notificacion.NewInApp(s.db), s.jobs))
}
