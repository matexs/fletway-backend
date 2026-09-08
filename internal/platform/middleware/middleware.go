// Package middleware tiene los middlewares HTTP transversales.
//
// El middleware Auth es obligatorio en TODOS los endpoints de negocio (RNF-01).
package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"time"

	"github.com/fletway/fletway-backend/internal/platform/auth"
	"github.com/fletway/fletway-backend/internal/platform/httpx"
)

// Chain aplica middlewares en orden: Chain(h, a, b, c) ejecuta a(b(c(h))).
func Chain(h http.Handler, mws ...func(http.Handler) http.Handler) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

// RequestID inyecta un id por request (header X-Request-ID o generado) en el contexto.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			b := make([]byte, 8)
			_, _ = rand.Read(b)
			id = hex.EncodeToString(b)
		}
		w.Header().Set("X-Request-ID", id)
		ctx := context.WithValue(r.Context(), requestIDKey{}, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type requestIDKey struct{}

func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

// Recover captura panics y responde 500 con el envelope de error.
func Recover(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			//nolint:contextcheck // defer de recuperación de panic; no hay contexto que propagar
			defer func() {
				if rec := recover(); rec != nil {
					log.Error("panic recuperado en handler",
						"err", rec,
						"path", r.URL.Path,
						"request_id", RequestIDFromContext(r.Context()),
					)
					httpx.Error(w, httpx.Internal("ocurrió un error inesperado"))
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// Logging registra método, path, status y duración de cada request.
func Logging(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(sw, r)
			log.Info("http",
				"method", r.Method,
				"path", r.URL.Path,
				"status", sw.status,
				"dur_ms", time.Since(start).Milliseconds(),
				"request_id", RequestIDFromContext(r.Context()),
			)
		})
	}
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (s *statusWriter) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

// Auth verifica el JWT de Supabase y pone la identidad en el contexto (RNF-01).
// Todo endpoint de negocio debe estar detrás de este middleware.
func Auth(v *auth.Verifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, err := auth.BearerToken(r)
			if err != nil {
				httpx.Error(w, httpx.Unauthorized("no_autenticado", "se requiere un token JWT válido"))
				return
			}
			id, err := v.Verify(r.Context(), token)
			if err != nil {
				httpx.Error(w, httpx.Unauthorized("token_invalido", "el token JWT es inválido o expiró"))
				return
			}
			ctx := auth.WithIdentity(r.Context(), id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
