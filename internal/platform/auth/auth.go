// Package auth verifica el JWT emitido por Supabase Auth (GoTrue) y expone la
// identidad del usuario al resto del backend.
//
// Decisión D-04: la autenticación es Supabase Auth; el backend sólo verifica el
// JWT. El proyecto firma con ES256 y publica sus claves en el JWKS, así que se
// verifica contra ese JWKS (con cache), sin fallback a HS256. El claim "sub" es
// usuario.id == auth.uid().
package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"

	"github.com/matexs/fletway-backend/internal/platform/config"
	"github.com/matexs/fletway-backend/internal/platform/database"
)

var (
	// ErrNoToken indica que el request no trae el header Authorization: Bearer.
	ErrNoToken = errors.New("auth: falta header Authorization: Bearer")
	// ErrInvalidToken indica un token mal firmado, vencido o con claims inválidos.
	ErrInvalidToken = errors.New("auth: token inválido o expirado")
	// ErrSinJWKS indica que falta la URL del JWKS en la configuración.
	ErrSinJWKS = errors.New("auth: falta SUPABASE_JWKS_URL")
)

// algoritmoFirma es el único algoritmo aceptado (D-04).
const algoritmoFirma = "ES256"

// Verifier valida tokens contra el JWKS de Supabase y devuelve la identidad.
type Verifier struct {
	keyfunc  jwt.Keyfunc
	audience string
	issuer   string
}

// NewVerifier crea un Verifier que descarga y cachea el JWKS de cfg.JWKSURL. El
// cache se refresca en segundo plano mientras ctx siga vivo. Devuelve ErrSinJWKS
// si la URL no está configurada.
func NewVerifier(ctx context.Context, cfg config.SupabaseConfig) (*Verifier, error) {
	if cfg.JWKSURL == "" {
		return nil, ErrSinJWKS
	}
	kf, err := keyfunc.NewDefaultCtx(ctx, []string{cfg.JWKSURL})
	if err != nil {
		return nil, fmt.Errorf("cargar JWKS de Supabase: %w", err)
	}
	return newVerifier(kf.Keyfunc, cfg), nil
}

// newVerifier arma el Verifier con una función de claves dada; permite testear
// sin un JWKS remoto.
func newVerifier(kf jwt.Keyfunc, cfg config.SupabaseConfig) *Verifier {
	return &Verifier{keyfunc: kf, audience: cfg.ExpectedAudience, issuer: cfg.ExpectedIssuer}
}

// Verify parsea y valida el JWT y devuelve la identidad para el RLS
// pass-through. Exige firma ES256 válida, exp vigente, aud esperado, iss
// esperado (si está configurado) y un sub presente. Ante cualquier falla
// devuelve ErrNoToken (token vacío) o un error que envuelve ErrInvalidToken.
func (v *Verifier) Verify(ctx context.Context, token string) (database.Identity, error) {
	_ = ctx // la validación es local; el JWKS se refresca en segundo plano.
	if token == "" {
		return database.Identity{}, ErrNoToken
	}
	opts := []jwt.ParserOption{
		jwt.WithValidMethods([]string{algoritmoFirma}),
		jwt.WithExpirationRequired(),
		jwt.WithAudience(v.audience),
	}
	if v.issuer != "" {
		opts = append(opts, jwt.WithIssuer(v.issuer))
	}
	claims := jwt.MapClaims{}
	if _, err := jwt.ParseWithClaims(token, claims, v.keyfunc, opts...); err != nil {
		return database.Identity{}, fmt.Errorf("%w: %w", ErrInvalidToken, err)
	}
	sub, _ := claims["sub"].(string)
	if sub == "" {
		return database.Identity{}, fmt.Errorf("%w: falta el claim sub", ErrInvalidToken)
	}
	raw, err := json.Marshal(claims)
	if err != nil {
		return database.Identity{}, fmt.Errorf("%w: serializar claims: %w", ErrInvalidToken, err)
	}
	role, _ := claims["role"].(string)
	email, _ := claims["email"].(string)
	return database.Identity{UserID: sub, Role: role, Email: email, RawClaims: string(raw), AccessToken: token}, nil
}

// BearerToken extrae el token del header Authorization. Devuelve ErrNoToken si
// el header falta o no tiene el formato "Bearer <token>".
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
// endpoints de negocio, RNF-01).
func FromContext(ctx context.Context) (database.Identity, bool) {
	id, ok := ctx.Value(ctxKey{}).(database.Identity)
	return id, ok
}
