// Package auth verifica el JWT emitido por Supabase Auth (GoTrue) y expone la
// identidad del usuario al resto del backend.
//
// Decisión D-04 (propuesta a confirmar): verificar contra el JWKS del proyecto
// (claves asimétricas) con fallback a HS256 usando SUPABASE_JWT_SECRET si el
// proyecto está en el esquema legacy. El claim "sub" es usuario.id == auth.uid().
//
// NOTA: skeleton stdlib-only. La verificación real (golang-jwt + keyfunc) se
// agrega con `go mod tidy` (ver go.mod).
package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/fletway/fletway-backend/internal/platform/config"
	"github.com/fletway/fletway-backend/internal/platform/database"
)

var (
	ErrNoToken      = errors.New("auth: falta header Authorization: Bearer")
	ErrInvalidToken = errors.New("auth: token inválido o expirado")
)

// Verifier valida tokens y devuelve la identidad.
type Verifier struct {
	cfg config.SupabaseConfig
	// jwks keyfunc.Keyfunc  // cache de claves públicas
}

func NewVerifier(cfg config.SupabaseConfig) *Verifier {
	return &Verifier{cfg: cfg}
}

// Verify parsea y valida el JWT, devolviendo la identidad para el RLS pass-through.
//
// Implementación futura:
//   - resolver la clave con keyfunc (JWKS) o con []byte(cfg.JWTSecret) (HS256)
//   - validar exp, aud (cfg.ExpectedAudience), iss (cfg.ExpectedIssuer)
//   - extraer sub, email, role de los claims
//   - RawClaims = json de los claims para SET LOCAL request.jwt.claims
func (v *Verifier) Verify(ctx context.Context, token string) (database.Identity, error) {
	_ = ctx
	if token == "" {
		return database.Identity{}, ErrNoToken
	}
	return database.Identity{}, ErrInvalidToken
}

// BearerToken extrae el token del header Authorization.
func BearerToken(r *http.Request) (string, error) {
	h := r.Header.Get("Authorization")
	if h == "" {
		return "", ErrNoToken
	}
	parts := strings.SplitN(h, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", ErrNoToken
	}
	return parts[1], nil
}

// --- Identidad en el contexto del request ---

type ctxKey struct{}

// WithIdentity guarda la identidad en el contexto.
func WithIdentity(ctx context.Context, id database.Identity) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

// FromContext recupera la identidad puesta por el middleware. El segundo valor es
// false si el request no pasó por el middleware de auth (no debería ocurrir en
// endpoints de negocio — RNF-01).
func FromContext(ctx context.Context) (database.Identity, bool) {
	id, ok := ctx.Value(ctxKey{}).(database.Identity)
	return id, ok
}
