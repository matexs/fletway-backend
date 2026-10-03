// Package authtest ayuda a testear código que verifica JWT de Supabase sin
// depender de Supabase: genera una clave ES256, sirve su JWKS por HTTP y firma
// tokens con los claims que pida cada test. Sólo se usa desde tests.
package authtest

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// KID es el identificador de clave que publica el JWKS de prueba.
const KID = "clave-de-prueba"

// Audiencia es el aud que usan los tokens de prueba (el de los usuarios de Supabase).
const Audiencia = "authenticated"

// JWKS es un emisor de tokens de prueba con su servidor de claves.
type JWKS struct {
	// URL es la dirección del JWKS servido por httptest.
	URL string
	// Emisor es el iss que usan los tokens por defecto (la URL base del servidor).
	Emisor string

	clave *ecdsa.PrivateKey
}

// Nuevo genera una clave ES256 y levanta un servidor que publica su JWKS. El
// servidor se cierra al terminar el test.
func Nuevo(t *testing.T) *JWKS {
	t.Helper()
	clave, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generar clave ES256: %v", err)
	}
	cuerpo, err := json.Marshal(map[string]any{"keys": []any{jwkPublica(t, &clave.PublicKey)}})
	if err != nil {
		t.Fatalf("serializar JWKS: %v", err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(cuerpo)
	}))
	t.Cleanup(srv.Close)
	return &JWKS{URL: srv.URL + "/auth/v1/.well-known/jwks.json", Emisor: srv.URL + "/auth/v1", clave: clave}
}

// ClaimsValidos devuelve los claims de un usuario autenticado válido para sub,
// con vencimiento en una hora. Cada test los modifica para probar casos.
func (j *JWKS) ClaimsValidos(sub string) jwt.MapClaims {
	return jwt.MapClaims{
		"sub":   sub,
		"role":  "authenticated",
		"email": "usuario@ejemplo.com",
		"aud":   Audiencia,
		"iss":   j.Emisor,
		"exp":   time.Now().Add(time.Hour).Unix(),
	}
}

// Firmar devuelve el token ES256 firmado con la clave publicada en el JWKS.
func (j *JWKS) Firmar(t *testing.T, claims jwt.MapClaims) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	tok.Header["kid"] = KID
	s, err := tok.SignedString(j.clave)
	if err != nil {
		t.Fatalf("firmar token: %v", err)
	}
	return s
}

// FirmarConOtraClave devuelve un token con los mismos claims pero firmado por una
// clave que no está en el JWKS (debe ser rechazado).
func FirmarConOtraClave(t *testing.T, claims jwt.MapClaims) string {
	t.Helper()
	otra, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generar clave: %v", err)
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	tok.Header["kid"] = KID
	s, err := tok.SignedString(otra)
	if err != nil {
		t.Fatalf("firmar token: %v", err)
	}
	return s
}

// FirmarHS256 devuelve un token firmado con HMAC (esquema legacy). Debe ser
// rechazado: el backend sólo acepta ES256 (D-04).
func FirmarHS256(t *testing.T, claims jwt.MapClaims) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := tok.SignedString([]byte("secreto-legacy-de-al-menos-32-caracteres"))
	if err != nil {
		t.Fatalf("firmar token HS256: %v", err)
	}
	return s
}

func jwkPublica(t *testing.T, pub *ecdsa.PublicKey) map[string]string {
	t.Helper()
	// Formato sin comprimir de P-256: 0x04 || X (32 bytes) || Y (32 bytes).
	raw, err := pub.Bytes()
	if err != nil || len(raw) != 65 {
		t.Fatalf("serializar clave pública: %v", err)
	}
	enc := base64.RawURLEncoding.EncodeToString
	return map[string]string{
		"kty": "EC", "crv": "P-256", "alg": "ES256", "use": "sig", "kid": KID,
		"x": enc(raw[1:33]), "y": enc(raw[33:65]),
	}
}
